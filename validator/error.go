package validator

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/storacha-network/go-ucanto/core/delegation"
	"github.com/storacha-network/go-ucanto/core/ipld"
	"github.com/storacha-network/go-ucanto/core/result/failure"
	"github.com/storacha-network/go-ucanto/did"
	"github.com/storacha-network/go-ucanto/ucan"
	vdm "github.com/storacha-network/go-ucanto/validator/datamodel"
)

// go hack for union type -- unexported method cannot be implemented outside module limiting satisfying types
type DelegationSubError interface {
	failure.Failure
	isDelegationSubError()
}

type InvalidProof interface {
	failure.Failure
	isInvalidProof()
}

type EscalatedCapabilityError[Caveats any] struct {
	failure.NamedWithStackTrace
	claimed   ucan.Capability[Caveats]
	delegated ucan.Capability[Caveats]
	cause     error
}

func NewEscalatedCapabilityError[Caveats any](claimed ucan.Capability[Caveats], delegated ucan.Capability[Caveats], cause error) EscalatedCapabilityError[Caveats] {
	return EscalatedCapabilityError[Caveats]{failure.NamedWithCurrentStackTrace("EscalatedCapability"), claimed, delegated, cause}
}

func (ece EscalatedCapabilityError[Caveats]) Unwrap() error {
	return ece.cause
}

func (ece EscalatedCapabilityError[Caveats]) Error() string {
	return fmt.Sprintf("Constraint violation: %s", ece.cause.Error())
}

func (ece EscalatedCapabilityError[Caveats]) isDelegationSubError() {}

type DelegationError interface {
	failure.Failure
	Causes() []DelegationSubError
	Context() any
	isDelegationError()
}

type delegationError struct {
	failure.NamedWithStackTrace
	causes  []DelegationSubError
	context interface{}
}

func NewDelegationError(causes []DelegationSubError, context interface{}) DelegationError {
	return delegationError{failure.NamedWithCurrentStackTrace("InvalidClaim"), causes, context}
}

func (de delegationError) Error() string {
	return fmt.Sprintf("Cannot derive %s from delegated capabilities: %s", de.context, errors.Join(de.Unwrap()...).Error())
}

func (de delegationError) Causes() []DelegationSubError {
	return de.causes
}

func (de delegationError) Context() any {
	return de.context
}

func (de delegationError) Unwrap() []error {
	errs := make([]error, 0, len(de.causes))
	for _, cause := range de.causes {
		errs = append(errs, cause)
	}
	return errs
}

func (de delegationError) isDelegationError()    {}
func (de delegationError) isDelegationSubError() {}

type SessionEscalationError struct {
	failure.NamedWithStackTrace
	delegation delegation.Delegation
	cause      error
}

func NewSessionEscalationError(delegation delegation.Delegation, cause error) InvalidProof {
	return SessionEscalationError{failure.NamedWithCurrentStackTrace("SessionEscalation"), delegation, cause}
}

func (see SessionEscalationError) Error() string {
	issuer := see.delegation.Issuer().DID()
	return strings.Join([]string{
		fmt.Sprintf("Delegation %s issued by %s has an invalid session", see.delegation.Link(), issuer),
		li(see.cause.Error()),
	}, "\n")
}

func (see SessionEscalationError) isInvalidProof() {}

// BadSignature is a signature that could not be verified or has been verified
// invalid. i.e. it is an [UnverifiableSignature] or an [InvalidSignature].
type BadSignature interface {
	InvalidProof
	Issuer() ucan.Principal
	Audience() ucan.Principal
	Delegation() delegation.Delegation
	isBadSignature()
}

// UnverifiableSignature is a signature that cannot be verified. i.e. some error
// occurred when attempting to verify the signature.
type UnverifiableSignature interface {
	BadSignature
	Unwrap() error
	isUnverifiableSignature()
}

type UnverifiableSignatureError struct {
	failure.NamedWithStackTrace
	delegation delegation.Delegation
	cause      error
}

func NewUnverifiableSignatureError(delegation delegation.Delegation, cause error) UnverifiableSignature {
	return UnverifiableSignatureError{failure.NamedWithCurrentStackTrace("UnverifiableSignature"), delegation, cause}
}

func (use UnverifiableSignatureError) Issuer() ucan.Principal {
	return use.delegation.Issuer()
}

func (use UnverifiableSignatureError) Audience() ucan.Principal {
	return use.delegation.Audience()
}

func (use UnverifiableSignatureError) Delegation() delegation.Delegation {
	return use.delegation
}

func (use UnverifiableSignatureError) Error() string {
	issuer := use.Issuer().DID()
	return fmt.Sprintf("Proof %s issued by %s cannot be verified:\n%s", use.delegation.Link(), issuer, li(use.cause.Error()))
}

func (use UnverifiableSignatureError) Unwrap() error {
	return use.cause
}

func (use UnverifiableSignatureError) isUnverifiableSignature() {}
func (use UnverifiableSignatureError) isBadSignature()          {}
func (use UnverifiableSignatureError) isInvalidProof()          {}

// InvalidSignature is a signature that is verified to be invalid.
type InvalidSignature interface {
	BadSignature
	isInvalidSignature()
}

type InvalidSignatureError struct {
	failure.NamedWithStackTrace
	delegation delegation.Delegation
	verifier   ucan.Verifier
}

func NewInvalidSignatureError(delegation delegation.Delegation, verifier ucan.Verifier) InvalidSignature {
	return InvalidSignatureError{failure.NamedWithCurrentStackTrace("InvalidSignature"), delegation, verifier}
}

func (ise InvalidSignatureError) Issuer() ucan.Principal {
	return ise.delegation.Issuer()
}

func (ise InvalidSignatureError) Audience() ucan.Principal {
	return ise.delegation.Audience()
}

func (ise InvalidSignatureError) Delegation() delegation.Delegation {
	return ise.delegation
}

func (ise InvalidSignatureError) Error() string {
	issuer := ise.Issuer().DID()
	key := ise.verifier.DID()
	if strings.HasPrefix(issuer.String(), "did:key") {
		return fmt.Sprintf(`Proof %s does not has a valid signature from %s`, ise.delegation.Link(), key)
	}
	return strings.Join([]string{
		fmt.Sprintf("Proof %s issued by %s does not has a valid signature from %s", ise.delegation.Link(), issuer, key),
		"  ℹ️ Probably issuer signed with a different key, which got rotated, invalidating delegations that were issued with prior keys",
	}, "\n")
}

func (ise InvalidSignatureError) isInvalidSignature() {}
func (ise InvalidSignatureError) isBadSignature()     {}
func (ise InvalidSignatureError) isInvalidProof()     {}

type UnavailableProof interface {
	InvalidProof
	Link() ucan.Link
	isUnavailableProof()
}

type UnavailableProofError struct {
	failure.NamedWithStackTrace
	link  ucan.Link
	cause error
}

func NewUnavailableProofError(link ucan.Link, cause error) UnavailableProof {
	return UnavailableProofError{failure.NamedWithCurrentStackTrace("UnavailableProof"), link, cause}
}

func (upe UnavailableProofError) Unwrap() error {
	return upe.cause
}

func (upe UnavailableProofError) Link() ucan.Link {
	return upe.link
}

func (upe UnavailableProofError) Error() string {
	messages := []string{
		fmt.Sprintf("Linked proof '%s' is not included and could not be resolved", upe.link),
	}
	if upe.cause != nil {
		messages = append(messages, li(fmt.Sprintf("Proof resolution failed with: %s", upe.cause.Error())))
	}
	return strings.Join(messages, "\n")
}

func (upe UnavailableProofError) isUnavailableProof() {}
func (upe UnavailableProofError) isInvalidProof()     {}

type UnresolvedDID interface {
	InvalidProof
	DID() did.DID
	isUnresolvedDID()
}

type DIDKeyResolutionError struct {
	failure.NamedWithStackTrace
	did   did.DID
	cause error
}

func NewDIDKeyResolutionError(did did.DID, cause error) UnresolvedDID {
	return DIDKeyResolutionError{failure.NamedWithCurrentStackTrace("DIDKeyResolutionError"), did, cause}
}

func (dkre DIDKeyResolutionError) Unwrap() error {
	return dkre.cause
}

func (dkre DIDKeyResolutionError) DID() did.DID {
	return dkre.did
}

func (dkre DIDKeyResolutionError) Error() string {
	return fmt.Sprintf("Unable to resolve '%s' key", dkre.did)
}

func (dkre DIDKeyResolutionError) isUnresolvedDID() {}
func (dkre DIDKeyResolutionError) isInvalidProof()  {}

type PrincipalAlignmentError struct {
	failure.NamedWithStackTrace
	audience   ucan.Principal
	delegation delegation.Delegation
}

func NewPrincipalAlignmentError(audience ucan.Principal, delegation delegation.Delegation) failure.Failure {
	return PrincipalAlignmentError{failure.NamedWithCurrentStackTrace("InvalidAudience"), audience, delegation}
}

func (pae PrincipalAlignmentError) Error() string {
	return fmt.Sprintf("Delegation audience is '%s' instead of '%s'", pae.delegation.Audience().DID(), pae.audience.DID())
}

func (pae PrincipalAlignmentError) isInvalidProof() {}

// InvalidCapability is an error produced when parsing capabilities.
type InvalidCapability interface {
	failure.Failure
	isInvalidCapability()
}

type MalformedCapability interface {
	InvalidCapability
	DelegationSubError
	Capability() ucan.Capability[any]
	isMalformedCapability()
}

type MalformedCapabilityError struct {
	failure.NamedWithStackTrace
	capability ucan.Capability[any]
	cause      error
}

func NewMalformedCapabilityError(capability ucan.Capability[any], cause error) MalformedCapability {
	return MalformedCapabilityError{failure.NamedWithCurrentStackTrace("MalformedCapability"), capability, cause}
}

func (mce MalformedCapabilityError) Error() string {
	capabilityJSON, _ := json.Marshal(mce.capability)
	return strings.Join([]string{
		fmt.Sprintf("Encountered malformed '%s' capability: %s", mce.capability.Can(), string(capabilityJSON)),
		li(mce.cause.Error()),
	}, "\n")
}

func (mce MalformedCapabilityError) Capability() ucan.Capability[any] {
	return mce.capability
}

func (mce MalformedCapabilityError) isMalformedCapability() {}
func (mce MalformedCapabilityError) isInvalidCapability()   {}
func (mce MalformedCapabilityError) isDelegationSubError()  {}

type UnknownCapability interface {
	InvalidCapability
	Capability() ucan.Capability[any]
	isUnknownCapability()
}

type UnknownCapabilityError struct {
	failure.NamedWithStackTrace
	capability ucan.Capability[any]
}

func NewUnknownCapabilityError(capability ucan.Capability[any]) UnknownCapability {
	return UnknownCapabilityError{failure.NamedWithCurrentStackTrace("UnknownCapability"), capability}
}

func (uce UnknownCapabilityError) Error() string {
	capabilityJSON, _ := json.Marshal(uce.capability)
	return fmt.Sprintf("Encountered unknown capability: %s", string(capabilityJSON))
}

func (uce UnknownCapabilityError) Capability() ucan.Capability[any] {
	return uce.capability
}

func (uce UnknownCapabilityError) isUnknownCapability()  {}
func (uce UnknownCapabilityError) isInvalidCapability()  {}
func (uce UnknownCapabilityError) isDelegationSubError() {}

type ExpiredError struct {
	failure.NamedWithStackTrace
	delegation delegation.Delegation
}

func NewExpiredError(delegation delegation.Delegation) InvalidProof {
	return ExpiredError{failure.NamedWithCurrentStackTrace("Expired"), delegation}
}

func (ee ExpiredError) Error() string {
	return fmt.Sprintf("Proof %s has expired on %s", ee.delegation.Link(),
		time.Unix(int64(ee.delegation.Expiration()), 0).Format(time.RFC3339))
}

func (ee ExpiredError) Build() (datamodel.Node, error) {
	name := ee.Name()
	stack := ee.Stack()
	expiredModel := vdm.ExpiredModel{
		Name:      &name,
		Message:   ee.Error(),
		ExpiredAt: int64(ee.delegation.Expiration()),
		Stack:     &stack,
	}
	return ipld.WrapWithRecovery(expiredModel, vdm.ExpiredType())
}

func (ee ExpiredError) isInvalidProof() {}

type Revoked interface {
	InvalidProof
	Delegation() delegation.Delegation
	isRevoked()
}

type RevokedError struct {
	failure.NamedWithStackTrace
	delegation delegation.Delegation
}

func NewRevokedError(delegation delegation.Delegation) Revoked {
	return RevokedError{failure.NamedWithCurrentStackTrace("Revoked"), delegation}
}

func (re RevokedError) Delegation() delegation.Delegation {
	return re.delegation
}

func (re RevokedError) Error() string {
	return fmt.Sprintf("Proof %s has been revoked", re.delegation.Link())
}

func (re RevokedError) isInvalidProof() {}
func (re RevokedError) isRevoked()      {}

type NotValidBeforeError struct {
	failure.NamedWithStackTrace
	delegation delegation.Delegation
}

func NewNotValidBeforeError(delegation delegation.Delegation) InvalidProof {
	return NotValidBeforeError{failure.NamedWithCurrentStackTrace("NotValidBefore"), delegation}
}

func (nvbe NotValidBeforeError) Error() string {
	return fmt.Sprintf("Proof %s is not valid before %s", nvbe.delegation.Link(),
		time.Unix(int64(nvbe.delegation.NotBefore()), 0).Format(time.RFC3339))
}

func (nvbe NotValidBeforeError) Build() (datamodel.Node, error) {
	name := nvbe.Name()
	stack := nvbe.Stack()
	notValidBeforeModel := vdm.NotValidBeforeModel{
		Name:    &name,
		Message: nvbe.Error(),
		ValidAt: int64(nvbe.delegation.NotBefore()),
		Stack:   &stack,
	}
	return ipld.WrapWithRecovery(notValidBeforeModel, vdm.NotValidBeforeType())
}

func (nvbe NotValidBeforeError) isInvalidProof() {}

type InvalidClaim interface {
	failure.Failure
	Issuer() ucan.Principal
	Delegation() delegation.Delegation
}

type InvalidClaimError[Caveats any] struct {
	failure.NamedWithStackTrace
	match               Match[Caveats]
	delegationErrors    []DelegationError
	unknownCapabilities []ucan.Capability[any]
	invalidProofs       []ProofError
	failedProofs        []InvalidClaim
}

func NewInvalidClaimError[Caveats any](
	match Match[Caveats],
	delegationErrors []DelegationError,
	unknownCapabilities []ucan.Capability[any],
	invalidProofs []ProofError,
	failedProofs []InvalidClaim,
) InvalidClaim {
	return InvalidClaimError[Caveats]{
		failure.NamedWithCurrentStackTrace("InvalidClaim"),
		match,
		delegationErrors,
		unknownCapabilities,
		invalidProofs,
		failedProofs,
	}
}

func (ice InvalidClaimError[Caveats]) Error() string {
	errorStrings := make([]string, 0, len(ice.failedProofs)+len(ice.delegationErrors)+len(ice.invalidProofs))

	for _, failedProof := range ice.failedProofs {
		errorStrings = append(errorStrings, li(failedProof.Error()))
	}

	for _, delegationError := range ice.delegationErrors {
		errorStrings = append(errorStrings, li(delegationError.Error()))
	}

	for _, invalidProof := range ice.invalidProofs {
		errorStrings = append(errorStrings, li(invalidProof.Error()))
	}

	unknowns := make([]string, 0, len(ice.unknownCapabilities))
	for _, unknownCapability := range ice.unknownCapabilities {
		out, _ := unknownCapability.MarshalJSON()
		unknowns = append(unknowns, li(string(out)))
	}

	var finalList []string
	finalList = append(finalList, fmt.Sprintf("Capability %s is not authorized because:", ice.match))
	finalList = append(finalList, li(fmt.Sprintf("Capability can not be (self) issued by '%s'", ice.Issuer().DID())))
	if len(errorStrings) > 0 {
		finalList = append(finalList, errorStrings...)
	} else {
		finalList = append(finalList, li("Delegated capability not found"))
	}
	if len(unknowns) > 0 {
		finalList = append(finalList, li(fmt.Sprintf("Encountered unknown capabilities\n%s", strings.Join(unknowns, "\n"))))
	}

	return strings.Join(finalList, "\n")
}

func (ice InvalidClaimError[Caveats]) Issuer() ucan.Principal {
	return ice.Delegation().Issuer()
}

func (ice InvalidClaimError[Caveats]) Delegation() delegation.Delegation {
	return ice.match.Source()[0].Delegation()
}

func (ice InvalidClaimError[Caveats]) DelegationErrors() []DelegationError {
	return ice.delegationErrors
}

func (ice InvalidClaimError[Caveats]) UnknownCapabilities() []ucan.Capability[any] {
	return ice.unknownCapabilities
}

func (ice InvalidClaimError[Caveats]) InvalidProofs() []ProofError {
	return ice.invalidProofs
}

func (ice InvalidClaimError[Caveats]) FailedProofs() []InvalidClaim {
	return ice.failedProofs
}

type Unauthorized interface {
	failure.Failure
	DelegationErrors() []DelegationError
	UnknownCapabilities() []ucan.Capability[any]
	InvalidProofs() []InvalidProof
	FailedProofs() []InvalidClaim
	isUnauthorized()
}

type UnauthorizedError[Caveats any] struct {
	failure.NamedWithStackTrace
	capability       CapabilityParser[Caveats]
	delegationErrors []DelegationError
	// this is a hack... it will allow you to make an array of capabilities of different types
	unknownCapabilities []ucan.Capability[any]
	invalidProofs       []InvalidProof
	failedProofs        []InvalidClaim
}

func NewUnauthorizedError[Caveats any](
	capability CapabilityParser[Caveats],
	delegationErrors []DelegationError,
	unknownCapabilities []ucan.Capability[any],
	invalidProofs []InvalidProof,
	failedProofs []InvalidClaim,
) Unauthorized {
	return UnauthorizedError[Caveats]{
		failure.NamedWithCurrentStackTrace("Unauthorized"),
		capability,
		delegationErrors,
		unknownCapabilities,
		invalidProofs,
		failedProofs,
	}
}

func (ue UnauthorizedError[Caveats]) Error() string {
	errorStrings := make([]string, 0, len(ue.failedProofs)+len(ue.delegationErrors)+len(ue.invalidProofs))

	for _, failedProof := range ue.failedProofs {
		errorStrings = append(errorStrings, li(failedProof.Error()))
	}

	for _, delegationError := range ue.delegationErrors {
		errorStrings = append(errorStrings, li(delegationError.Error()))
	}

	for _, invalidProof := range ue.invalidProofs {
		errorStrings = append(errorStrings, li(invalidProof.Error()))
	}

	unknowns := make([]string, 0, len(ue.unknownCapabilities))
	for _, unknownCapability := range ue.unknownCapabilities {
		out, _ := unknownCapability.MarshalJSON()
		unknowns = append(unknowns, li(string(out)))
	}

	finalList := make([]string, 0, 2+int(math.Min(1, float64(len(errorStrings)))))
	finalList = append(finalList, fmt.Sprintf("Claim %s is not authorized", ue.capability))
	if len(errorStrings) > 0 {
		finalList = append(finalList, errorStrings...)
	} else {
		finalList = append(finalList, li("No matching delegated capability found"))
	}
	if len(unknowns) > 0 {
		finalList = append(finalList, li(fmt.Sprintf("Encountered unknown capabilities\n%s", strings.Join(unknowns, "\n"))))
	}

	return strings.Join(finalList, "\n")
}

func (ue UnauthorizedError[Caveats]) DelegationErrors() []DelegationError {
	return ue.delegationErrors
}

func (ue UnauthorizedError[Caveats]) UnknownCapabilities() []ucan.Capability[any] {
	return ue.unknownCapabilities
}

func (ue UnauthorizedError[Caveats]) InvalidProofs() []InvalidProof {
	return ue.invalidProofs
}

func (ue UnauthorizedError[Caveats]) FailedProofs() []InvalidClaim {
	return ue.failedProofs
}

func (ue UnauthorizedError[Caveats]) isUnauthorized() {}

func indent(message string) string {
	indent := "  "
	return indent + strings.Join(strings.Split(message, "\n"), "\n"+indent)
}

func li(message string) string {
	return indent("- " + message)
}

type ProofError struct {
	failure.NamedWithStackTrace
	proof ucan.Link
	cause error
}

func (pe ProofError) Error() string {
	return fmt.Sprintf("Capability can not be derived from prf: %s because: %s\n", pe.proof, li(pe.cause.Error()))
}

func (pe ProofError) Proof() ucan.Link {
	return pe.proof
}

func (pe ProofError) Unwrap() error {
	return pe.cause
}

func NewProofError(proof ucan.Link, cause error) ProofError {
	return ProofError{failure.NamedWithCurrentStackTrace("ProofError"), proof, cause}
}
