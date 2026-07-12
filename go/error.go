// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: Apache-2.0

package spore

type ErrorCode string

const (
	// Internal / meta
	ErrorCodeUnknownFailure    ErrorCode = "UnknownFailure"
	ErrorCodeSporeFailure      ErrorCode = "SporeFailure"
	ErrorCodeProtocolFailure   ErrorCode = "ProtocolFailure"
	ErrorCodeConnectionFailure ErrorCode = "ConnectionFailure"

	// General
	ErrorCodeGeneric            ErrorCode = "Generic"
	ErrorCodeFatal              ErrorCode = "Fatal"
	ErrorCodeTimeout            ErrorCode = "Timeout"
	ErrorCodeBusy               ErrorCode = "Busy"
	ErrorCodeResourcesExhausted ErrorCode = "ResourcesExhausted"
	ErrorCodeDeprecated         ErrorCode = "Deprecated"
	ErrorCodeRuntime            ErrorCode = "Runtime"
	ErrorCodeLogic              ErrorCode = "Logic"
	ErrorCodeReservedKeyword    ErrorCode = "ReservedKeyword"

	// Route
	ErrorCodeRouteNotFound       ErrorCode = "RouteNotFound"
	ErrorCodeRouteNotConnected   ErrorCode = "RouteNotConnected"
	ErrorCodeRouteNotAvailable   ErrorCode = "RouteNotAvailable"
	ErrorCodeRouteNotAllowed     ErrorCode = "RouteNotAllowed"
	ErrorCodeRouteNotImplemented ErrorCode = "RouteNotImplemented"

	// Message
	ErrorCodeMessageNotValid  ErrorCode = "MessageNotValid"
	ErrorCodeMessageMalformed ErrorCode = "MessageMalformed"

	// Arguments
	ErrorCodeArgumentMissing      ErrorCode = "RequiredArgumentMissing"
	ErrorCodeArgumentInvalidType  ErrorCode = "ArgumentInvalidType"
	ErrorCodeArgumentConflict     ErrorCode = "ArgumentConflict"
	ErrorCodeArgumentOutOfRange   ErrorCode = "ArgumentOutOfRange"
	ErrorCodeArgumentUnrecognized ErrorCode = "ArgumentUnrecognized"
	ErrorCodeArgumentDuplicated   ErrorCode = "ArgumentDuplicated"

	// Flags
	ErrorCodeFlagConflict     ErrorCode = "FlagConflict"
	ErrorCodeFlagUnrecognized ErrorCode = "FlagUnrecognized"
	ErrorCodeFlagDuplicated   ErrorCode = "FlagDuplicated"

	// Handles
	ErrorCodeHandleMissing ErrorCode = "HandleMissing"
	ErrorCodeHandleInUse   ErrorCode = "HandleInUse"
	ErrorCodeHandleExpired ErrorCode = "HandleExpired"
)

// HubError is returned by Listen when the hub explicitly rejects the
// connection after the handshake — for example when manifest or binary
// verification fails. Unlike a transient disconnect, a HubError signals
// that reconnecting without taking corrective action (e.g. reinstalling
// the node) will not resolve the problem.
type HubError struct {
	Code string
	What string
}

func (e *HubError) Error() string {
	if e.What != "" {
		return "hub rejected connection: " + e.Code + ": " + e.What
	}
	return "hub rejected connection: " + e.Code
}

// ErrorOrigin identifies where an error originated.
// Exactly one origin flag is present on every error response.
type ErrorOrigin string

const (
	// ErrorOriginSpore indicates the error was produced by the hub/runtime.
	// Nodes should not use this; the hub blocks it if a node emits it.
	ErrorOriginSpore ErrorOrigin = "spore_error"

	// ErrorOriginNode indicates the error originated in the receiving node.
	ErrorOriginNode ErrorOrigin = "node_error"

	// ErrorOriginCast indicates the error is attributable to the caller
	// (e.g. bad arguments, missing handle).
	ErrorOriginCast ErrorOrigin = "cast_error"

	// ErrorOriginCapture indicates the error is attributable to the responder's reply.
	ErrorOriginCapture ErrorOrigin = "capture_error"
)
