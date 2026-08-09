// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: Apache-2.0

package spore

// This file contains only //export functions.  CGo rules prohibit definitions
// in the C preamble of a file that uses //export, so the static trampoline
// functions live in client.go instead.

/*
#include "spore_c.h"
*/
import "C"
import (
	"fmt"
	"strings"
)

// goOnRequest is the single C request callback registered for every Client.
// It runs on the C listen-loop thread; Call must not escape the handler.
//
//export goOnRequest
func goOnRequest(hClient *C.spore_client_t, hRequest *C.spore_request_t) {
	c, ok := lookupClient(hClient)
	if !ok {
		return
	}

	subject := C.GoString(C.spore_request_get_command(hRequest))
	handle := C.GoString(C.spore_request_get_handle(hRequest))

	call := &Call{
		Subject:  subject,
		Handle:   handle,
		cClient:  hClient,
		cRequest: hRequest,
	}
	c.dispatchRequest(call)
}

// goOnResponse is the single C response callback registered for every Client.
//
//export goOnResponse
func goOnResponse(hClient *C.spore_client_t, hResponse *C.spore_response_t, hError *C.spore_response_error_t) {
	c, ok := lookupClient(hClient)
	if !ok {
		return
	}

	resp := &Response{
		Args: make(map[string]string),
	}

	if hResponse != nil {
		// ok response
		resp.OK = true
		raw := C.GoString(C.spore_response_get_command(hResponse))
		if idx := strings.IndexByte(raw, ':'); idx >= 0 {
			resp.Handle = raw[:idx]
			resp.Subject = raw[idx+1:]
		} else {
			resp.Subject = raw
		}
		resp.Handle = C.GoString(C.spore_response_get_handle(hResponse))
		// Structured args are not iterable via the current C API; callers
		// must use the raw Send/Wait path if they need arg data from responses.
	} else if hError != nil {
		// error response
		resp.OK = false
		resp.ErrCode = cStr(C.spore_response_error_get_code(hError))
		resp.ErrWhat = cStr(C.spore_response_error_get_what(hError))
		raw := C.GoString(C.spore_response_error_get_command(hError))
		if idx := strings.IndexByte(raw, ':'); idx >= 0 {
			resp.Handle = raw[:idx]
			resp.Subject = raw[idx+1:]
		} else {
			resp.Subject = raw
		}
		resp.Handle = cStr(C.spore_response_error_get_handle(hError))
	}

	c.dispatchResponse(resp)
}

// goOnWitness is the single C witness callback registered for every Client.
//
//export goOnWitness
func goOnWitness(hClient *C.spore_client_t, hWitness *C.spore_witness_t) {
	c, ok := lookupClient(hClient)
	if !ok {
		return
	}

	// The body is the full inner message text; parse it in Go.
	body := cStr(C.spore_witness_get_body(hWitness))
	msg, err := parseWitness("witness " + body)
	if err != nil {
		return
	}
	c.dispatchWitness(msg)
}

// goOnPublish is the single C publish callback registered for every Client.
//
//export goOnPublish
func goOnPublish(hClient *C.spore_client_t, hPublish *C.spore_publish_t) {
	c, ok := lookupClient(hClient)
	if !ok {
		return
	}

	topic := cStr(C.spore_publish_get_topic(hPublish))
	cast := cStr(C.spore_publish_get_arg(hPublish, C.CString("cast")))

	// Structured arg iteration is not yet in the C API; publish messages with
	// additional args will only show topic and cast until it is added.
	msg := &PublishMessage{
		Topic: topic,
		Cast:  cast,
		Args:  make(map[string]string),
		Flags: []string{},
	}

	c.dispatchPublish(msg)
}

// cStr converts a nullable C string to a Go string.
func cStr(s *C.char) string {
	if s == nil {
		return ""
	}
	return C.GoString(s)
}

// wireNeedsQuoting reports whether v needs double-quote wrapping on the wire.
func wireNeedsQuoting(v string) bool {
	for i := 0; i < len(v); i++ {
		switch v[i] {
		case ' ', '\t', '\n', '\r', '\\', '"':
			return true
		}
	}
	return false
}

// wireEscape applies backslash escaping for double-quoted wire values.
func wireEscape(v string) string {
	var b strings.Builder
	b.Grow(len(v))
	for i := 0; i < len(v); i++ {
		switch v[i] {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteByte(v[i])
		}
	}
	return b.String()
}

// formatArg formats a key=value pair for the wire.
func formatArg(k, v string) string {
	if strings.HasPrefix(v, "[") || strings.HasPrefix(v, "{") {
		return fmt.Sprintf("%s=%s", k, v)
	}
	if wireNeedsQuoting(v) {
		return fmt.Sprintf(`%s="%s"`, k, wireEscape(v))
	}
	return fmt.Sprintf("%s=%s", k, v)
}
