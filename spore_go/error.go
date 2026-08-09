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
// connection post-handshake (e.g. manifest verification failure).
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

// ErrorOrigin identifies where an error response originated.
type ErrorOrigin string

const (
	ErrorOriginSpore   ErrorOrigin = "spore_error"
	ErrorOriginNode    ErrorOrigin = "node_error"
	ErrorOriginCast    ErrorOrigin = "cast_error"
	ErrorOriginCapture ErrorOrigin = "capture_error"
)
