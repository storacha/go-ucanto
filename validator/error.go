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
	"github.com/storacha-network/go-ucanto/core/result"
	"github.com/storacha-network/go-ucanto/did"
	"github.com/storacha-network/go-ucanto/ucan"
	vdm "github.com/storacha-network/go-ucanto/validator/datamodel"
)

// go hack for union type -- unexported method cannot be implemented outside module limiting satisfying types
type DelegationSubError interface {
	error
	isDelegationSubError()
}

type InvalidProofError interface {
	error
	isInvalidProofError()
}

type EscalatedCapabilityError[Caveats any] struct {
	result.NamedWithStackTrace
	claimed   ucan.Capability[Caveats]
	delegated interface{}
	cause     error
}

func NewEscalatedCapabilityError[Caveats any](claimed ucan.Capability[Caveats], delegated interface{}, cause error) error {
	return EscalatedCapabilityError[Caveats]{result.NamedWithCurrentStackTrace("EscalatedCapability"), claimed, delegated, cause}
}

func (ece EscalatedCapabilityError[Caveats]) Unwrap() error {
	return ece.cause
}

func (ece EscalatedCapabilityError[Caveats]) Error() string {
	return fmt.Sprintf("Constraint violation: %s", ece.cause.Error())
}

func (ece EscalatedCapabilityError[Caveats]) isDelegationSubError() {
}

/**
 * @implements {API.DelegationError}
 */
type DelegationError struct {
	result.NamedWithStackTrace
	causes  []DelegationSubError
	context interface{}
}

func NewDelegationError(causes []DelegationSubError, context interface{}) error {
	return DelegationError{result.NamedWithCurrentStackTrace("InvalidClaim"), causes, context}
}

func (de DelegationError) Error() string {
	return fmt.Sprintf("Cannot derive %s from delegated capabilities: %s", de.context, errors.Join(de.Unwrap()...).Error())
}

func (de DelegationError) Unwrap() []error {
	errs := make([]error, 0, len(de.causes))
	for _, cause := range de.causes {
		errs = append(errs, cause)
	}
	return errs
}

func (de DelegationError) isDelegationSubError() {}

type SessionEscalationError struct {
	result.NamedWithStackTrace
	delegation delegation.Delegation
	cause      error
}

func NewSessionEscalationError(delegation delegation.Delegation, cause error) error {
	return SessionEscalationError{result.NamedWithCurrentStackTrace("SessionEscalation"), delegation, cause}
}

func (see SessionEscalationError) Error() string {
	issuer := see.delegation.Issuer().DID()
	return strings.Join([]string{
		fmt.Sprintf("Delegation %s issued by %s has an invalid session", see.delegation.Link(), issuer),
		li(see.cause.Error()),
	}, "\n")
}

func (see SessionEscalationError) isInvalidProofError() {}

type InvalidSignatureError struct {
	result.NamedWithStackTrace
	delegation delegation.Delegation
	verifier   ucan.Verifier
}

func NewInvalidSignatureError(delegation delegation.Delegation, verifier ucan.Verifier) error {
	return InvalidSignatureError{result.NamedWithCurrentStackTrace("InvalidSignature"), delegation, verifier}
}

func (ise InvalidSignatureError) Issuer() ucan.Principal {
	return ise.delegation.Issuer()
}
func (ise InvalidSignatureError) Audience() ucan.Principal {
	return ise.delegation.Audience()
}

func (ise InvalidSignatureError) Error() string {
	issuer := ise.Issuer().DID()
	key := ise.verifier.DID()
	if !strings.HasPrefix(issuer.String(), "did:key") {
		return fmt.Sprintf(`Proof %s does not has a valid signature from %s`, ise.delegation.Link(), key)
	}
	return strings.Join([]string{
		fmt.Sprintf("Proof %s issued by %s does not has a valid signature from %s", ise.delegation.Link(), issuer, key),
		"  ℹ️ Probably issuer signed with a different key, which got rotated, invalidating delegations that were issued with prior keys",
	}, "\n")
}

func (ise InvalidSignatureError) isInvalidProofError() {}

type UnavailableProofError struct {
	result.NamedWithStackTrace
	link  ucan.Link
	cause error
}

func NewUnavailableProofError(link ucan.Link, cause error) UnavailableProofError {
	return UnavailableProofError{result.NamedWithCurrentStackTrace("UnavailableProof"), link, cause}
}

func (upe UnavailableProofError) Unwrap() error {
	return upe.cause
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

func (upe UnavailableProofError) isInvalidProofError() {}

type DIDKeyResolutionError struct {
	result.NamedWithStackTrace
	did   did.DID
	cause error
}

func NewDIDKeyResolutionError(did did.DID, cause error) DIDKeyResolutionError {
	return DIDKeyResolutionError{result.NamedWithCurrentStackTrace("DIDKeyResolutionError"), did, cause}
}

func (dkre DIDKeyResolutionError) Unwrap() error {
	return dkre.cause
}

func (dkre DIDKeyResolutionError) Error() string {
	return fmt.Sprintf("Unable to resolve '%s' key", dkre.did)
}

func (dkre DIDKeyResolutionError) isInvalidProofError() {}

type PrincipalAlignmentError struct {
	result.NamedWithStackTrace
	audience   ucan.Principal
	delegation delegation.Delegation
}

func NewPrincipalAlignmentError(audience ucan.Principal, delegation delegation.Delegation) PrincipalAlignmentError {
	return PrincipalAlignmentError{result.NamedWithCurrentStackTrace("InvalidAudience"), audience, delegation}
}

func (pae PrincipalAlignmentError) Error() string {
	return fmt.Sprintf("Delegation audience is '%s' instead of '%s'", pae.delegation.Audience().DID(), pae.audience.DID())
}

func (pae PrincipalAlignmentError) Build() (datamodel.Node, error) {
	name := pae.Name()
	stack := pae.Stack()
	invalidAudienceModel := vdm.InvalidAudienceModel{
		Name:       &name,
		Audience:   pae.audience.DID().String(),
		Delegation: vdm.Delegation{Audience: pae.delegation.Audience().DID().String()},
		Message:    pae.Error(),
		Stack:      &stack,
	}
	return ipld.WrapWithRecovery(&invalidAudienceModel, vdm.InvalidAudienceType())
}

func (pae PrincipalAlignmentError) isInvalidProofError() {}

type MalformedCapabilityError[Caveats any] struct {
	result.NamedWithStackTrace
	capability ucan.Capability[Caveats]
	cause      error
}

func NewMalformedCapabilityError[Caveats any](capability ucan.Capability[Caveats], cause error) MalformedCapabilityError[Caveats] {
	return MalformedCapabilityError[Caveats]{result.NamedWithCurrentStackTrace("MalformedCapability"), capability, cause}
}

func (mce MalformedCapabilityError[Caveats]) Error() string {
	capabilityJSON, _ := json.Marshal(mce.capability)
	return strings.Join([]string{
		fmt.Sprintf("Encountered malformed '%s' capability: %s", mce.capability.Can(), string(capabilityJSON)),
		li(mce.cause.Error()),
	}, "\n")
}

func (mce MalformedCapabilityError[Caveats]) isDelegationSubError() {}

type UnknownCapabilityError[Caveats any] struct {
	result.NamedWithStackTrace
	capability ucan.Capability[Caveats]
}

func NewUnknownCapabilityError[Caveats any](capability ucan.Capability[Caveats]) error {
	return UnknownCapabilityError[Caveats]{result.NamedWithCurrentStackTrace("UnknownCapability"), capability}
}

func (uce UnknownCapabilityError[Caveats]) Error() string {
	capabilityJSON, _ := json.Marshal(uce.capability)
	return fmt.Sprintf("Encountered unknown capability: %s", string(capabilityJSON))
}

func (uce UnknownCapabilityError[Caveats]) isDelegationSubError() {}

type ExpiredError struct {
	result.NamedWithStackTrace
	delegation delegation.Delegation
}

func NewExpiredError(delegation delegation.Delegation) error {
	return ExpiredError{result.NamedWithCurrentStackTrace("Expired"), delegation}
}

func (ee ExpiredError) Error() string {
	return fmt.Sprintf("Proof %s has expired on %s", ee.delegation.Link(),
		time.UnixMilli(int64(ee.delegation.Expiration())).Format(time.RFC3339))
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

func (ee ExpiredError) isInvalidProofError() {}

type RevokedError struct {
	result.NamedWithStackTrace
	delegation delegation.Delegation
}

func NewRevokedError(delegation delegation.Delegation) error {
	return RevokedError{result.NamedWithCurrentStackTrace("Revoked"), delegation}
}

func (re RevokedError) Error() string {
	return fmt.Sprintf("Proof %s has been revoked", re.delegation.Link())
}

func (re RevokedError) isInvalidProofError() {}

type NotValidBeforeError struct {
	result.NamedWithStackTrace
	delegation delegation.Delegation
}

func NewNotValidBeforeERror(delegation delegation.Delegation) error {
	return NotValidBeforeError{result.NamedWithCurrentStackTrace("NotValidBefore"), delegation}
}

func (nvbe NotValidBeforeError) Error() string {
	return fmt.Sprintf("Proof %s is not valid before %s", nvbe.delegation.Link(),
		time.UnixMilli(int64(nvbe.delegation.NotBefore())).Format(time.RFC3339))
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

func (nvbe NotValidBeforeError) isInvalidProofError() {}

// TODO: this may just be the concrete type from the implementation once
// the rest of the validator is done
type InvalidClaim interface {
	result.NamedWithStackTrace
	error
	Issuer() ucan.Principal
	Delegation() delegation.Delegation
}

type UnauthorizedError[Caveats any] struct {
	result.NamedWithStackTrace
	capability       ucan.Capability[Caveats]
	delegationErrors []DelegationError
	// this is a hack... it will allow you to make an array of capabilities of different types
	unknownCapabilities []ucan.UnknownCapability
	invalidProofs       []InvalidProofError
	failedProofs        []InvalidClaim
}

func NewUnauthorizedError[Caveats any](
	capability ucan.Capability[Caveats],
	delegationErrors []DelegationError,
	unknownCapabilities []ucan.UnknownCapability,
	invalidProofs []InvalidProofError,
	failedProofs []InvalidClaim,
) error {
	return UnauthorizedError[Caveats]{
		result.NamedWithCurrentStackTrace("Unauthorized"),
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
	finalList = append(finalList, fmt.Sprintf("Claim %+v is not authorized", ue.capability))
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

func indent(message string) string {
	indent := "  "
	return indent + strings.Join(strings.Split(message, "\n"), "\n$"+indent)
}

func li(message string) string {
	return indent("- " + message)
}
