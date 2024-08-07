package server

import (
	"fmt"

	"github.com/ipld/go-ipld-prime/node/bindnode"
	"github.com/storacha-network/go-ucanto/core/ipld"
	"github.com/storacha-network/go-ucanto/core/result"
	sdm "github.com/storacha-network/go-ucanto/server/datamodel"
	"github.com/storacha-network/go-ucanto/ucan"
)

type HandlerNotFoundError[Caveats any] interface {
	result.Failure
	Capability() ucan.Capability[Caveats]
}

type handlerNotFoundError[Caveats any] struct {
	capability ucan.Capability[Caveats]
}

func (h *handlerNotFoundError[C]) Capability() ucan.Capability[C] {
	return h.capability
}

func (h *handlerNotFoundError[C]) Error() string {
	return fmt.Sprintf("service does not implement {can: \"%s\"} handler", h.capability.Can())
}

func (h *handlerNotFoundError[C]) Name() string {
	return "HandlerNotFoundError"
}

func (h *handlerNotFoundError[C]) Build() (ipld.Node, error) {
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
	return bindnode.Wrap(&mdl, sdm.HandlerNotFoundErrorType()), nil
}

var _ HandlerNotFoundError[any] = (*handlerNotFoundError[any])(nil)
var _ ipld.Builder = (*handlerNotFoundError[any])(nil)

func NewHandlerNotFoundError[Caveats any](capability ucan.Capability[Caveats]) *handlerNotFoundError[Caveats] {
	return &handlerNotFoundError[Caveats]{capability}
}

type HandlerExecutionError[Caveats any] interface {
	result.Failure
	result.WithStackTrace
	Cause() error
	Capability() ucan.Capability[Caveats]
}

type handlerExecutionError[Caveats any] struct {
	cause      error
	capability ucan.Capability[Caveats]
}

func (h *handlerExecutionError[C]) Capability() ucan.Capability[C] {
	return h.capability
}

func (h *handlerExecutionError[C]) Cause() error {
	return h.cause
}

func (h *handlerExecutionError[C]) Error() string {
	return fmt.Sprintf("service handler {can: \"%s\"} error: %s", h.capability.Can(), h.cause.Error())
}

func (h *handlerExecutionError[C]) Name() string {
	return "HandlerExecutionError"
}

func (h *handlerExecutionError[C]) Stack() string {
	var stack string
	if serr, ok := h.cause.(result.WithStackTrace); ok {
		stack = serr.Stack()
	}
	return stack
}

func (h *handlerExecutionError[C]) Build() (ipld.Node, error) {
	name := h.Name()
	stack := h.Stack()

	var cname string
	if ncause, ok := h.cause.(result.Named); ok {
		cname = ncause.Name()
	}

	var cstack string
	if scause, ok := h.cause.(result.WithStackTrace); ok {
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
	}
	return bindnode.Wrap(&mdl, sdm.HandlerExecutionErrorType()), nil
}

var _ HandlerExecutionError[any] = (*handlerExecutionError[any])(nil)
var _ ipld.Builder = (*handlerExecutionError[any])(nil)

func NewHandlerExecutionError[Caveats any](cause error, capability ucan.Capability[Caveats]) *handlerExecutionError[Caveats] {
	return &handlerExecutionError[Caveats]{cause, capability}
}

type InvocationCapabilityError interface {
	result.Failure
	Capabilities() []ucan.Capability[any]
}

type invocationCapabilityError struct {
	capabilities []ucan.Capability[any]
}

func (i *invocationCapabilityError) Capabilities() []ucan.Capability[any] {
	return i.capabilities
}

func (i *invocationCapabilityError) Error() string {
	return "Invocation is required to have a single capability."
}

func (i *invocationCapabilityError) Name() string {
	return "InvocationCapabilityError"
}

func (i *invocationCapabilityError) Build() (ipld.Node, error) {
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
	return bindnode.Wrap(&mdl, sdm.InvocationCapabilityErrorType()), nil
}

var _ InvocationCapabilityError = (*invocationCapabilityError)(nil)
var _ ipld.Builder = (*invocationCapabilityError)(nil)

func NewInvocationCapabilityError(capabilities []ucan.Capability[any]) *invocationCapabilityError {
	return &invocationCapabilityError{capabilities}
}
