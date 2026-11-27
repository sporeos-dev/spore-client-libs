// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: Apache-2.0

package spore

import (
	"fmt"
	"strings"
	"unicode"
)

// SendResponseStart declares the string fields that will be streamed for a response.
// Finish the response with SendResponse (without repeating the fields) or SendResponseError.
func (c *Client) SendResponseStart(handle, command string, fields ...string) error {
	frame, err := responseStartFrame(handle, command, fields)
	if err != nil {
		return err
	}
	return c.SendRaw(frame)
}

// SendResponseChunk immediately sends text for an open field. It does not add a
// line break to the value; include one in text if needed.
func (c *Client) SendResponseChunk(handle, field, text string) error {
	frame, err := responseChunkFrame(handle, field, text)
	if err != nil {
		return err
	}
	return c.SendRaw(frame)
}

// SendResponseClose closes a field. Close all streamed fields before sending a
// successful completion.
func (c *Client) SendResponseClose(handle, field string) error {
	frame, err := responseFieldFrame(handle, field)
	if err != nil {
		return err
	}
	return c.SendRaw(frame + ">>")
}

var responseChunkEscapes = strings.NewReplacer(`\`, `\\`, "\n", `\n`, "\r", `\r`, "\t", `\t`)

func responseChunkFrame(handle, field, text string) (string, error) {
	frame, err := responseFieldFrame(handle, field)
	if err != nil {
		return "", err
	}
	if strings.ContainsRune(text, 0) {
		return "", fmt.Errorf("response chunk contains a NUL byte")
	}
	return frame + "<< " + responseChunkEscapes.Replace(text), nil
}

func responseStartFrame(handle, command string, fields []string) (string, error) {
	frame, err := responseFieldFrame(handle, command)
	if err != nil {
		return "", err
	}
	if len(fields) == 0 {
		return "", fmt.Errorf("streamed response needs at least one field")
	}
	var opening strings.Builder
	opening.WriteString(frame)
	seen := make(map[string]bool, len(fields))
	for _, field := range fields {
		if !validStreamToken(field) || seen[field] {
			return "", fmt.Errorf("invalid or duplicate streamed field %q", field)
		}
		seen[field] = true
		opening.WriteString(" " + field + "=<<")
	}
	return opening.String(), nil
}

func responseFieldFrame(handle, token string) (string, error) {
	if !validStreamToken(handle) || !validStreamToken(token) {
		return "", fmt.Errorf("invalid stream handle %q or capability/field %q", handle, token)
	}
	return "~" + handle + ":" + token, nil
}

func validStreamToken(token string) bool {
	return token != "" && strings.IndexFunc(token, unicode.IsSpace) == -1 &&
		!strings.ContainsAny(token, "\x00:<>=\"'[{")
}
