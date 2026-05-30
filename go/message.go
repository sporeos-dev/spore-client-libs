// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: Apache-2.0

package spore

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// Call represents an incoming call forwarded by the hub.
//
// The hub routes a call to this node based on the manifest registry.
// Use Arg/ArgOr/HasArg to read arguments, HasFlag for flags, and
// Reply/Error to send a response back through the hub.
type Call struct {
	// Subject is the full subject string (e.g., "clock.get_time").
	Subject string

	// Handle is the caller's correlation token, if present.
	// Used to bind the response to the originating call.
	Handle string

	// args holds key-value arguments from the message.
	args map[string]string

	// flags holds bare flag tokens from the message.
	flags map[string]bool

	// conn is the connection to write the response back on.
	conn net.Conn
}

// Arg returns the value of a named argument. Panics if the argument is not present.
// Use HasArg to check first, or ArgIf for a safe default.
func (c *Call) Arg(name string) string {
	val, ok := c.args[name]
	if !ok {
		panic(fmt.Sprintf("spore: argument %q not present in call", name))
	}
	return val
}

// ArgIf returns the value of a named argument, or def if not present.
func (c *Call) ArgIf(name string, def string) string {
	if val, ok := c.args[name]; ok {
		return val
	}
	return def
}

// HasArg returns true if the named argument is present.
func (c *Call) HasArg(name string) bool {
	_, ok := c.args[name]
	return ok
}

// HasFlag returns true if the named flag is present.
func (c *Call) HasFlag(name string) bool {
	_, ok := c.flags[name]
	return ok
}

// Reply sends a success response back through the hub.
// The hub will inject "ok" and "capture=" before forwarding to the caller.
//
// Pass nil or an empty map for void commands (no return data).
//
//	call.Reply(map[string]string{"time": "2026-03-13T14:32:00Z"})
//	call.Reply(nil) // void acknowledgment
func (c *Call) Reply(args map[string]string) error {
	response := c.formatResponsePrefix()

	for k, v := range args {
		response += " " + formatArgValue(k, v)
	}

	_, err := c.conn.Write([]byte(response + "\n"))
	return err
}

// Cancel sends a cancelled response back through the hub, indicating the call
// completed gracefully but produced no output (e.g. the user dismissed a dialog).
// This is not an error — it is a clean non-result. No data fields are sent.
// The hub will inject "capture=" before forwarding to the caller.
//
//	call.Cancel()
func (c *Call) Cancel() error {
	response := c.formatResponsePrefix() + " cancelled"
	_, err := c.conn.Write([]byte(response + "\n"))
	return err
}

// Error sends a standard error response back through the hub using a predefined
// ErrorCode constant. The hub will inject "capture=" before forwarding to the caller.
// The origin is always "node_error" for node-produced responses.
//
//	call.Error(ErrorCodeArgumentMissing, "timezone argument is required")
func (c *Call) Error(code ErrorCode, what string) error {
	response := c.formatResponsePrefix()

	if strings.Contains(what, " ") {
		response += fmt.Sprintf(" error code=%s what=\"%s\" node_error", string(code), what)
	} else {
		response += fmt.Sprintf(" error code=%s what=%s node_error", string(code), what)
	}

	_, err := c.conn.Write([]byte(response + "\n"))
	return err
}

// ErrorCustom sends a node-defined error response back through the hub.
// It uses the "custom_error" status flag (instead of "error") to signal that
// the code is declared in the node's manifest errors field.
// The origin is always "node_error" for node-produced responses.
//
//	call.ErrorCustom("clock.err.invalid_timezone", "Unknown timezone: Fakezone")
func (c *Call) ErrorCustom(code string, what string) error {
	response := c.formatResponsePrefix()

	if strings.Contains(what, " ") {
		response += fmt.Sprintf(" custom_error code=%s what=\"%s\" node_error", code, what)
	} else {
		response += fmt.Sprintf(" custom_error code=%s what=%s node_error", code, what)
	}

	_, err := c.conn.Write([]byte(response + "\n"))
	return err
}

// formatArgValue formats key=value for a wire message.
//
// [array] and {object} tokens already act as opaque delimiters in the Spore
// tokenizer, so they are never wrapped in quotes regardless of whether they
// contain spaces. Plain string values that contain spaces are wrapped in
// double-quotes. All other values are sent as-is.
func formatArgValue(k, v string) string {
	if strings.HasPrefix(v, "[") || strings.HasPrefix(v, "{") {
		return fmt.Sprintf("%s=%s", k, v)
	}
	if strings.Contains(v, " ") {
		return fmt.Sprintf("%s=\"%s\"", k, v)
	}
	return fmt.Sprintf("%s=%s", k, v)
}

// formatResponsePrefix builds the ~handle:subject or subject prefix.
func (c *Call) formatResponsePrefix() string {
	if c.Handle != "" {
		return fmt.Sprintf("~%s:%s", c.Handle, c.Subject)
	}
	return c.Subject
}

// parseCall parses a raw incoming message string into a Call.
//
// parseCall parses a raw incoming message string into a Call.
//
// Format: subject [key=value ...] [flag ...] [~handle]
//
// All key=value pairs (including reserved ones like cast=, capture=) go into
// args. All bare tokens (including reserved ones like ok, error, node_error)
// go into flags. Only ~handle has its own field.
func parseCall(raw string) (*Call, error) {
	call := &Call{
		args:  make(map[string]string),
		flags: make(map[string]bool),
	}

	parts := splitFields(raw)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty message")
	}

	// First token is the subject
	call.Subject = parts[0]

	for _, part := range parts[1:] {
		switch {
		case strings.HasPrefix(part, "~"):
			call.Handle = strings.TrimPrefix(part, "~")

		case strings.Contains(part, "="):
			kv := strings.SplitN(part, "=", 2)
			val := kv[1]
			// Remove surrounding quotes if present
			if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
				val = val[1 : len(val)-1]
			}
			call.args[kv[0]] = val

		default:
			call.flags[part] = true
		}
	}

	return call, nil
}

// Response represents a reply received after sending a call to another node.
//
// Responses are identified by a leading "~handle:subject" token and are
// produced by the hub routing the reply back to the caller.
//
// Use Arg/ArgOr to read returned values. Check OK to distinguish success
// from failure; on failure, ErrCode and ErrWhat contain the error details.
// Use FromSpore() to test whether the error was produced by the hub rather
// than the target node.
type Response struct {
	// Subject mirrors the subject of the originating call.
	Subject string

	// Handle is the correlation token matching the one sent with the call.
	Handle string

	// OK is true when the remote node replied with "ok".
	OK bool

	// Cancelled is true when the node responded with "cancelled", indicating
	// the call completed gracefully but produced no output (e.g. user dismissed
	// a dialog). Exactly one of OK, Cancelled, CustomError, or a standard error
	// is present on every response.
	Cancelled bool

	// CustomError is true when the response uses the "custom_error" status flag,
	// indicating a node-defined error declared in the manifest errors field.
	// When false and OK is also false, it is a standard error.
	CustomError bool

	// ErrCode is the namespaced error identifier on failure (e.g. "clock.err.invalid_timezone").
	ErrCode string

	// ErrWhat is the human-readable error description on failure.
	ErrWhat string

	// ErrorOrigin is the origin flag on an error response indicating where the
	// error was produced (ErrorOriginNode, ErrorOriginCast, ErrorOriginCapture,
	// ErrorOriginSpore). Empty on success responses.
	ErrorOrigin ErrorOrigin

	// Capture is the node ID of whoever handled the call, injected by the hub.
	// It is "SPORE.hub" when the hub itself produced the response (e.g. node
	// unreachable, protocol error). Use FromSpore() as a convenience check.
	Capture string

	// Args holds all key=value pairs returned by the remote node, excluding
	// the reserved routing and error fields (capture, code, what).
	Args map[string]string
}

// FromSpore reports whether this response was produced by the hub rather than
// the target node. Hub-produced responses carry capture=SPORE.hub and
// ErrorOrigin == ErrorOriginSpore.
func (r *Response) FromSpore() bool {
	return r.Capture == "SPORE.hub"
}

// Arg returns the value of a named argument. Panics if the argument is not present.
// Use ArgIf for a safe default.
func (r *Response) Arg(name string) string {
	val, ok := r.Args[name]
	if !ok {
		panic(fmt.Sprintf("spore: argument %q not present in response", name))
	}
	return val
}

// ArgIf returns the value of a named argument, or def if not present.
func (r *Response) ArgIf(name string, def string) string {
	if val, ok := r.Args[name]; ok {
		return val
	}
	return def
}

// parseResponse parses a raw response string into a Response.
//
// Expected format: ~handle:subject ok [key=value ...]
//                  ~handle:subject error code=X what=Y
func parseResponse(raw string) (*Response, error) {
	if !strings.HasPrefix(raw, "~") {
		return nil, fmt.Errorf("not a response message")
	}

	resp := &Response{
		Args: make(map[string]string),
	}

	parts := splitFields(raw)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty response")
	}

	// First token: ~handle:subject
	head := strings.TrimPrefix(parts[0], "~")
	if idx := strings.IndexByte(head, ':'); idx >= 0 {
		resp.Handle = head[:idx]
		resp.Subject = head[idx+1:]
	} else {
		resp.Handle = head
	}

	for _, part := range parts[1:] {
		switch {
		case part == "ok":
			resp.OK = true
		case part == "cancelled":
			resp.Cancelled = true
		case part == "error":
			// standard error — OK stays false, CustomError stays false
		case part == "custom_error":
			resp.CustomError = true
		case part == "spore_error":
			resp.ErrorOrigin = ErrorOriginSpore
		case part == "node_error":
			resp.ErrorOrigin = ErrorOriginNode
		case part == "cast_error":
			resp.ErrorOrigin = ErrorOriginCast
		case part == "capture_error":
			resp.ErrorOrigin = ErrorOriginCapture
		case strings.Contains(part, "="):
			kv := strings.SplitN(part, "=", 2)
			val := kv[1]
			if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
				val = val[1 : len(val)-1]
			}
			resp.Args[kv[0]] = val
		}
	}

	// Promote and remove reserved routing/error fields so Args contains only
	// node-returned data.
	resp.Capture = resp.Args["capture"]
	delete(resp.Args, "capture")
	resp.ErrCode = resp.Args["code"]
	delete(resp.Args, "code")
	resp.ErrWhat = resp.Args["what"]
	delete(resp.Args, "what")

	return resp, nil
}

// splitFields splits a string by spaces, respecting double-quoted strings and
// [array] / {object} tokens. Spaces inside any of these delimiters do not
// split the token.
//
// For example:
//
//	`foo bar="hello world" baz`          → ["foo", `bar="hello world"`, "baz"]
//	`echo items=["a", "b", "c"] ~h1`     → ["echo", `items=["a", "b", "c"]`, "~h1"]
//	`cmd data={key: value with spaces}`  → ["cmd", `data={key: value with spaces}`]
// WitnessKind identifies the type of a witness copy received from the hub.
// Exactly one kind flag is present on every witness message.
type WitnessKind string

const (
	// WitnessKindIncoming marks a raw inbound message — received from a node
	// before hub injection or routing.
	WitnessKindIncoming WitnessKind = "spore_incoming"

	// WitnessKindOutgoing marks a message sent to a node — after hub injection
	// (cast=, capture=, ok, etc.).
	WitnessKindOutgoing WitnessKind = "spore_outgoing"

	// WitnessKindExpanded marks an inline call expansion — a ~~-handled inner
	// call generated by the hub during inline substitution.
	WitnessKindExpanded WitnessKind = "spore_expanded"

	// WitnessKindEvent marks a hub lifecycle or internal error event, not a
	// routed message (e.g. SPORE.hub.event).
	WitnessKindEvent WitnessKind = "spore_event"

	// WitnessKindNode marks a node-emitted observability message sent via
	// SPORE.witness.emit — not observed traffic.
	WitnessKindNode WitnessKind = "spore_node"
)

// WitnessMessage represents a witness copy delivered by the hub.
//
// Every witness copy is prefixed with "witness " on the wire. The body may
// be either a call-type message (IsResponse == false, subject-first) or a
// response-type message (IsResponse == true, ~handle:subject-first).
//
// Use Kind to determine what the copy represents and filter accordingly.
// SporeTime carries the hub-stamped Unix millisecond timestamp.
type WitnessMessage struct {
	// Kind identifies what this witness copy represents.
	Kind WitnessKind

	// SporeTime is the hub-stamped timestamp in Unix milliseconds.
	SporeTime int64

	// IsResponse is true when the body is a response-type message
	// (~handle:subject). False means it is a call-type message (subject first).
	IsResponse bool

	// Subject is the subject of the inner message.
	Subject string

	// Handle is the correlation token, if present.
	Handle string

	// Cast is the originating caller node ID (from cast= on call-type bodies).
	Cast string

	// --- Response-type body fields (meaningful when IsResponse == true) ---

	// OK is true when the inner response carries "ok".
	OK bool

	// Cancelled is true when the inner response carries "cancelled".
	Cancelled bool

	// CustomError is true when the inner response carries "custom_error".
	CustomError bool

	// ErrCode is the error code on error responses.
	ErrCode string

	// ErrWhat is the human-readable error description on error responses.
	ErrWhat string

	// Capture is the responding node ID (from capture= on response-type bodies).
	Capture string

	// ErrorOrigin indicates where an error originated (response-type bodies).
	ErrorOrigin ErrorOrigin

	// Args holds all data key=value pairs, excluding protocol and spore_ fields.
	Args map[string]string

	// Flags holds bare flag tokens from the body, excluding spore_ flags and
	// protocol status flags (ok, cancelled, error, custom_error, etc.).
	Flags map[string]bool
}

// parseWitness parses a raw witness line into a WitnessMessage.
//
// Format: "witness " <body>
// where <body> is either a call-type message (subject ...) or a response-type
// message (~handle:subject ...).
func parseWitness(raw string) (*WitnessMessage, error) {
	const prefix = "witness "
	if !strings.HasPrefix(raw, prefix) {
		return nil, fmt.Errorf("not a witness message")
	}
	body := raw[len(prefix):]

	msg := &WitnessMessage{
		Args:  make(map[string]string),
		Flags: make(map[string]bool),
	}

	parts := splitFields(body)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty witness body")
	}

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

	for _, part := range parts[1:] {
		switch {
		// Witness kind flags
		case part == "spore_incoming":
			msg.Kind = WitnessKindIncoming
		case part == "spore_outgoing":
			msg.Kind = WitnessKindOutgoing
		case part == "spore_expanded":
			msg.Kind = WitnessKindExpanded
		case part == "spore_event":
			msg.Kind = WitnessKindEvent
		case part == "spore_node":
			msg.Kind = WitnessKindNode
		// Handle token in call-type bodies
		case strings.HasPrefix(part, "~"):
			msg.Handle = strings.TrimPrefix(part, "~")
		// Response status flags
		case part == "ok":
			msg.OK = true
		case part == "cancelled":
			msg.Cancelled = true
		case part == "error":
			// standard error — OK/CustomError remain false
		case part == "custom_error":
			msg.CustomError = true
		// Error origin flags
		case part == "spore_error":
			msg.ErrorOrigin = ErrorOriginSpore
		case part == "node_error":
			msg.ErrorOrigin = ErrorOriginNode
		case part == "cast_error":
			msg.ErrorOrigin = ErrorOriginCast
		case part == "capture_error":
			msg.ErrorOrigin = ErrorOriginCapture
		// Key=value pairs
		case strings.Contains(part, "="):
			kv := strings.SplitN(part, "=", 2)
			val := kv[1]
			if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
				val = val[1 : len(val)-1]
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
		// Bare flag tokens
		default:
			msg.Flags[part] = true
		}
	}

	return msg, nil
}

func splitFields(s string) []string {
	var fields []string
	var current strings.Builder
	inDouble := false
	depth := 0 // tracks unmatched [ and { outside of double-quoted strings

	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch {
		case ch == '"' && !inDouble:
			inDouble = true
			current.WriteByte(ch)
		case ch == '"' && inDouble:
			inDouble = false
			current.WriteByte(ch)
		case (ch == '[' || ch == '{') && !inDouble:
			depth++
			current.WriteByte(ch)
		case (ch == ']' || ch == '}') && !inDouble:
			depth--
			current.WriteByte(ch)
		case ch == ' ' && !inDouble && depth == 0:
			if current.Len() > 0 {
				fields = append(fields, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(ch)
		}
	}
	if current.Len() > 0 {
		fields = append(fields, current.String())
	}

	return fields
}
