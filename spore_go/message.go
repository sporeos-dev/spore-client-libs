// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: Apache-2.0

package spore

/*
#include "spore_c.h"
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"strconv"
	"strings"
	"unsafe"
)

// Call represents an incoming request routed to this node by the hub.
//
// Call wraps a C spore_request_t pointer valid only for the duration of the
// handler. Do NOT store Call or access it after the handler returns.
type Call struct {
	Subject string
	Handle  string

	// C-side pointers; valid only while inside the handler callback.
	cClient  *C.spore_client_t
	cRequest *C.spore_request_t
}

// Arg returns the value of a named argument. Panics when not present.
func (c *Call) Arg(name string) string {
	cs := C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	v := C.spore_request_get_arg(c.cRequest, cs)
	if v == nil {
		panic(fmt.Sprintf("spore: argument %q not present in call", name))
	}
	return C.GoString(v)
}

// ArgIf returns the value of a named argument, or def when not present.
func (c *Call) ArgIf(name string, def string) string {
	cs := C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	v := C.spore_request_get_arg(c.cRequest, cs)
	if v == nil {
		return def
	}
	return C.GoString(v)
}

// HasArg reports whether the named argument is present.
func (c *Call) HasArg(name string) bool {
	cs := C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	return C.spore_request_get_arg(c.cRequest, cs) != nil
}

// HasFlag reports whether the named flag is present.
func (c *Call) HasFlag(name string) bool {
	cs := C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	return bool(C.spore_request_has_flag(c.cRequest, cs))
}

// Reply sends a success response back through the hub.
func (c *Call) Reply(args map[string]string) error {
	var sb strings.Builder
	if c.Handle != "" {
		fmt.Fprintf(&sb, "~%s:%s ok", c.Handle, c.Subject)
	} else {
		fmt.Fprintf(&sb, "%s ok", c.Subject)
	}
	for k, v := range args {
		sb.WriteByte(' ')
		sb.WriteString(formatArg(k, v))
	}
	wire := sb.String() + "\n"
	cs := C.CString(wire)
	defer C.free(unsafe.Pointer(cs))
	C.spore_client_send_raw(c.cClient, cs, C.size_t(len(wire)))
	return nil
}

// Cancel sends a cancelled response back through the hub.
func (c *Call) Cancel() error {
	var wire string
	if c.Handle != "" {
		wire = fmt.Sprintf("~%s:%s cancelled\n", c.Handle, c.Subject)
	} else {
		wire = fmt.Sprintf("%s cancelled\n", c.Subject)
	}
	cs := C.CString(wire)
	defer C.free(unsafe.Pointer(cs))
	C.spore_client_send_raw(c.cClient, cs, C.size_t(len(wire)))
	return nil
}

// Error sends a standard error response back through the hub.
func (c *Call) Error(code ErrorCode, what string) error {
	wire := c.prefix() + fmt.Sprintf(
		" error code=%s %s node_error\n", string(code), formatArg("what", what))
	cs := C.CString(wire)
	defer C.free(unsafe.Pointer(cs))
	C.spore_client_send_raw(c.cClient, cs, C.size_t(len(wire)))
	return nil
}

// ErrorCustom sends a node-defined error response back through the hub.
func (c *Call) ErrorCustom(code string, what string) error {
	wire := c.prefix() + fmt.Sprintf(
		" custom_error code=%s %s node_error\n", code, formatArg("what", what))
	cs := C.CString(wire)
	defer C.free(unsafe.Pointer(cs))
	C.spore_client_send_raw(c.cClient, cs, C.size_t(len(wire)))
	return nil
}

func (c *Call) prefix() string {
	if c.Handle != "" {
		return fmt.Sprintf("~%s:%s", c.Handle, c.Subject)
	}
	return c.Subject
}

// Response represents a reply received from another node or the hub.
type Response struct {
	Subject     string
	Handle      string
	OK          bool
	Cancelled   bool
	CustomError bool
	ErrCode     string
	ErrWhat     string
	ErrorOrigin ErrorOrigin
	Capture     string
	Args        map[string]string
}

// FromSpore reports whether this response was produced by the hub.
func (r *Response) FromSpore() bool { return r.Capture == "SPORE.hub" }

// Arg returns the value of a named argument. Panics when not present.
func (r *Response) Arg(name string) string {
	val, ok := r.Args[name]
	if !ok {
		panic(fmt.Sprintf("spore: argument %q not present in response", name))
	}
	return val
}

// ArgIf returns the value of a named argument, or def when not present.
func (r *Response) ArgIf(name string, def string) string {
	if val, ok := r.Args[name]; ok {
		return val
	}
	return def
}

// parseResponse parses a raw ~handle:subject … wire line into a Response.
// Used by the raw-socket receive path (see also goOnResponse for the C path).
func parseResponse(raw string) (*Response, error) {
	if !strings.HasPrefix(raw, "~") {
		return nil, fmt.Errorf("not a response message")
	}

	resp := &Response{Args: make(map[string]string)}

	parts := splitFields(raw)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty response")
	}

	head := strings.TrimPrefix(parts[0], "~")
	if idx := strings.IndexByte(head, ':'); idx >= 0 {
		resp.Handle = head[:idx]
		resp.Subject = head[idx+1:]
	} else {
		resp.Handle = head
	}

	for _, p := range parts[1:] {
		switch {
		case p == "ok":
			resp.OK = true
		case p == "cancelled":
			resp.Cancelled = true
		case p == "error":
			// status flag only
		case p == "custom_error":
			resp.CustomError = true
		case p == string(ErrorOriginSpore):
			resp.ErrorOrigin = ErrorOriginSpore
		case p == string(ErrorOriginNode):
			resp.ErrorOrigin = ErrorOriginNode
		case p == string(ErrorOriginCast):
			resp.ErrorOrigin = ErrorOriginCast
		case p == string(ErrorOriginCapture):
			resp.ErrorOrigin = ErrorOriginCapture
		case strings.Contains(p, "="):
			kv := strings.SplitN(p, "=", 2)
			val := kv[1]
			if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
				val = wireUnescape(val[1 : len(val)-1])
			}
			switch kv[0] {
			case "capture":
				resp.Capture = val
			case "code":
				resp.ErrCode = val
			case "what":
				resp.ErrWhat = val
			default:
				resp.Args[kv[0]] = val
			}
		}
	}

	return resp, nil
}

// WitnessKind identifies the type of a witness copy received from the hub.
type WitnessKind string

const (
	WitnessKindIncoming WitnessKind = "spore_incoming"
	WitnessKindOutgoing WitnessKind = "spore_outgoing"
	WitnessKindExpanded WitnessKind = "spore_expanded"
	WitnessKindEvent    WitnessKind = "spore_event"
	WitnessKindNode     WitnessKind = "spore_node"
)

// WitnessMessage represents a witness copy delivered by the hub.
type WitnessMessage struct {
	Kind        WitnessKind
	SporeTime   int64
	IsResponse  bool
	Subject     string
	Handle      string
	Cast        string
	OK          bool
	Cancelled   bool
	CustomError bool
	ErrCode     string
	ErrWhat     string
	Capture     string
	ErrorOrigin ErrorOrigin
	Args        map[string]string
	Flags       map[string]bool
}

func parseWitness(raw string) (*WitnessMessage, error) {
	const prefix = "witness "
	if !strings.HasPrefix(raw, prefix) {
		return nil, fmt.Errorf("not a witness message")
	}
	parts := splitFields(raw[len(prefix):])
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty witness body")
	}

	msg := &WitnessMessage{Args: make(map[string]string), Flags: make(map[string]bool)}
	msg.IsResponse = strings.HasPrefix(parts[0], "~")
	if msg.IsResponse {
		head := strings.TrimPrefix(parts[0], "~")
		if idx := strings.IndexByte(head, ':'); idx >= 0 {
			msg.Handle = head[:idx]
			msg.Subject = head[idx+1:]
		} else {
			msg.Handle = head
		}
	} else {
		msg.Subject = parts[0]
	}

	for _, p := range parts[1:] {
		switch {
		case p == "spore_incoming":
			msg.Kind = WitnessKindIncoming
		case p == "spore_outgoing":
			msg.Kind = WitnessKindOutgoing
		case p == "spore_expanded":
			msg.Kind = WitnessKindExpanded
		case p == "spore_event":
			msg.Kind = WitnessKindEvent
		case p == "spore_node":
			msg.Kind = WitnessKindNode
		case strings.HasPrefix(p, "~"):
			msg.Handle = strings.TrimPrefix(p, "~")
		case p == "ok":
			msg.OK = true
		case p == "cancelled":
			msg.Cancelled = true
		case p == "error":
			// status flag only
		case p == "custom_error":
			msg.CustomError = true
		case p == "spore_error":
			msg.ErrorOrigin = ErrorOriginSpore
		case p == "node_error":
			msg.ErrorOrigin = ErrorOriginNode
		case p == "cast_error":
			msg.ErrorOrigin = ErrorOriginCast
		case p == "capture_error":
			msg.ErrorOrigin = ErrorOriginCapture
		case strings.Contains(p, "="):
			kv := strings.SplitN(p, "=", 2)
			val := kv[1]
			if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
				val = wireUnescape(val[1 : len(val)-1])
			}
			switch kv[0] {
			case "spore_time":
				if n, err := strconv.ParseInt(val, 10, 64); err == nil {
					msg.SporeTime = n
				}
			case "capture":
				msg.Capture = val
			case "code":
				msg.ErrCode = val
			case "what":
				msg.ErrWhat = val
			case "cast":
				msg.Cast = val
			default:
				msg.Args[kv[0]] = val
			}
		default:
			msg.Flags[p] = true
		}
	}

	return msg, nil
}

// PublishMessage represents an incoming pub/sub message delivered by the hub.
type PublishMessage struct {
	Topic string
	Cast  string
	Args  map[string]string
	Flags []string
}

func (p *PublishMessage) Arg(name string) string {
	val, ok := p.Args[name]
	if !ok {
		panic(fmt.Sprintf("spore: argument %q not present in publish message", name))
	}
	return val
}

func (p *PublishMessage) ArgIf(name string, def string) string {
	if val, ok := p.Args[name]; ok {
		return val
	}
	return def
}

func (p *PublishMessage) HasArg(name string) bool {
	_, ok := p.Args[name]
	return ok
}

func (p *PublishMessage) HasFlag(name string) bool {
	for _, f := range p.Flags {
		if f == name {
			return true
		}
	}
	return false
}

// splitFields splits a wire message string by spaces, honouring double-quoted
// strings and [array] / {object} tokens.
func splitFields(s string) []string {
	var fields []string
	var cur strings.Builder
	inDouble := false
	depth := 0

	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch {
		case ch == '"' && !inDouble:
			inDouble = true
			cur.WriteByte(ch)
		case ch == '\\' && inDouble && i+1 < len(s):
			cur.WriteByte(ch)
			i++
			cur.WriteByte(s[i])
		case ch == '"' && inDouble:
			inDouble = false
			cur.WriteByte(ch)
		case (ch == '[' || ch == '{') && !inDouble:
			depth++
			cur.WriteByte(ch)
		case (ch == ']' || ch == '}') && !inDouble:
			depth--
			cur.WriteByte(ch)
		case ch == ' ' && !inDouble && depth == 0:
			if cur.Len() > 0 {
				fields = append(fields, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteByte(ch)
		}
	}
	if cur.Len() > 0 {
		fields = append(fields, cur.String())
	}
	return fields
}

func wireUnescape(s string) string {
	if !strings.ContainsRune(s, '\\') {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	i := 0
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case '\\':
				b.WriteByte('\\')
			case '"':
				b.WriteByte('"')
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case 't':
				b.WriteByte('\t')
			default:
				b.WriteByte('\\')
				b.WriteByte(s[i+1])
			}
			i += 2
		} else {
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}
