package server

import (
	"fmt"
	"strings"

	"github.com/storacha/go-ucanto/core/ipld"
	"github.com/storacha/go-ucanto/core/result/failure"
	sdm "github.com/storacha/go-ucanto/server/datamodel"
	"github.com/storacha/go-ucanto/ucan"
)

type HandlerNotFoundError[Caveats any] interface {
	failure.Failure
	Capability() ucan.Capability[Caveats]
}

type handlerNotFoundError[Caveats any] struct {
	capability ucan.Capability[Caveats]
}

func (h handlerNotFoundError[C]) Capability() ucan.Capability[C] {
	return h.capability
}

func (h handlerNotFoundError[C]) Error() string {
	return fmt.Sprintf("service does not implement {can: \"%s\"} handler", h.capability.Can())
}

func (h handlerNotFoundError[C]) Name() string {
	return "HandlerNotFoundError"
}

func (h handlerNotFoundError[C]) ToIPLD() (ipld.Node, error) {
	name := h.Name()

	mdl := sdm.HandlerNotFoundErrorModel{
		Error:   true,
		Name:    &name,
		Message: h.Error(),
		Capability: sdm.CapabilityModel{
			Can:  h.capability.Can(),
			With: h.capability.With(),
		},
	}
	return ipld.WrapWithRecovery(&mdl, sdm.HandlerNotFoundErrorType())
}

func NewHandlerNotFoundError[Caveats any](capability ucan.Capability[Caveats]) HandlerNotFoundError[Caveats] {
	return handlerNotFoundError[Caveats]{capability}
}

type HandlerExecutionError[Caveats any] interface {
	failure.Failure
	failure.WithStackTrace
	Cause() error
	Capability() ucan.Capability[Caveats]
}

type handlerExecutionError[Caveats any] struct {
	cause      error
	capability ucan.Capability[Caveats]
}

func (h handlerExecutionError[C]) Capability() ucan.Capability[C] {
	return h.capability
}

func (h handlerExecutionError[C]) Cause() error {
	return h.cause
}

func (h handlerExecutionError[C]) Error() string {
	return fmt.Sprintf("service handler {can: \"%s\"} error: %s", h.capability.Can(), h.cause.Error())
}

func (h handlerExecutionError[C]) Name() string {
	return "HandlerExecutionError"
}

func (h handlerExecutionError[C]) Stack() string {
	var stack string
	if serr, ok := h.cause.(failure.WithStackTrace); ok {
		stack = serr.Stack()
	}
	return stack
}

func (h handlerExecutionError[C]) ToIPLD() (ipld.Node, error) {
	name := h.Name()
	stack := h.Stack()

	var cname string
	if ncause, ok := h.cause.(failure.Named); ok {
		cname = ncause.Name()
	}

	var cstack string
	if scause, ok := h.cause.(failure.WithStackTrace); ok {
		cstack = scause.Stack()
	}

	mdl := sdm.HandlerExecutionErrorModel{
		Error:   true,
		Name:    &name,
		Message: h.Error(),
		Stack:   &stack,
		Cause: sdm.FailureModel{
			Name:    &cname,
			Message: h.cause.Error(),
			Stack:   &cstack,
		},
		Capability: sdm.CapabilityModel{
			Can:  h.capability.Can(),
			With: h.capability.With(),
		},
	}
	return ipld.WrapWithRecovery(&mdl, sdm.HandlerExecutionErrorType())
}

func NewHandlerExecutionError[Caveats any](cause error, capability ucan.Capability[Caveats]) HandlerExecutionError[Caveats] {
	return handlerExecutionError[Caveats]{cause, capability}
}

type InvocationCapabilityError interface {
	failure.Failure
	Capabilities() []ucan.Capability[any]
}

type invocationCapabilityError struct {
	capabilities []ucan.Capability[any]
}

func (i invocationCapabilityError) Capabilities() []ucan.Capability[any] {
	return i.capabilities
}

func (i invocationCapabilityError) Error() string {
	return "Invocation is required to have a single capability."
}

func (i invocationCapabilityError) Name() string {
	return "InvocationCapabilityError"
}

func (i invocationCapabilityError) ToIPLD() (ipld.Node, error) {
	name := i.Name()
	var capmdls []sdm.CapabilityModel
	for _, cap := range i.Capabilities() {
		capmdls = append(capmdls, sdm.CapabilityModel{
			Can:  cap.Can(),
			With: cap.With(),
		})
	}

	mdl := sdm.InvocationCapabilityErrorModel{
		Error:        true,
		Name:         &name,
		Message:      i.Error(),
		Capabilities: capmdls,
	}
	return ipld.WrapWithRecovery(&mdl, sdm.InvocationCapabilityErrorType())
}

func NewInvocationCapabilityError(capabilities []ucan.Capability[any]) InvocationCapabilityError {
	return invocationCapabilityError{capabilities}
}

type InvalidAudienceError struct {
	accepted []string
	actual   string
}

func (i InvalidAudienceError) Error() string {
	return fmt.Sprintf("Invalid audience: accepted %s, got %s", strings.Join(i.accepted, ", "), i.actual)
}

func (i InvalidAudienceError) Name() string {
	return "InvalidAudienceError"
}

func (i InvalidAudienceError) ToIPLD() (ipld.Node, error) {
	name := i.Name()
	mdl := sdm.InvalidAudienceErrorModel{
		Error:   true,
		Name:    &name,
		Message: i.Error(),
	}
	return ipld.WrapWithRecovery(&mdl, sdm.InvalidAudienceErrorType())
}

func NewInvalidAudienceError(actual string, accepted ...string) InvalidAudienceError {
	return InvalidAudienceError{accepted, actual}
}
