// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: Apache-2.0

package spore

import (
	"fmt"
	"runtime"

	sporec "github.com/sporeos-dev/spore-client-libs/spore_go/internal/spore_c"
	"github.com/sporeos-dev/spore-client-libs/spore_go/request"
)

// ResponseStream sends one response field incrementally. A send failure makes
// the stream unusable because the peer may have received a partial frame.
type ResponseStream struct {
	h      *sporec.ResponseStream
	client *Client
}

// OpenStream starts a response with one streamed field. It copies the request's
// handle and command, so the stream may outlive the request callback.
func (c *Client) OpenStream(req *request.Request, field string) (*ResponseStream, error) {
	if req == nil || req.H() == nil {
		return nil, fmt.Errorf("stream request is nil")
	}
	h, err := sporec.StreamOpen(c.h, req.H(), field)
	runtime.KeepAlive(req)
	runtime.KeepAlive(c)
	if err != nil {
		return nil, err
	}
	s := &ResponseStream{h: h, client: c}
	runtime.SetFinalizer(s, (*ResponseStream).destroy)
	return s, nil
}

func (s *ResponseStream) destroy() {
	sporec.StreamDestroy(s.h)
}

// Chunk sends text immediately without adding a newline.
func (s *ResponseStream) Chunk(text string) error {
	err := sporec.StreamChunk(s.h, text)
	runtime.KeepAlive(s)
	return err
}

// Write implements io.Writer. Streamed values are text and cannot contain NUL.
func (s *ResponseStream) Write(p []byte) (int, error) {
	if err := s.Chunk(string(p)); err != nil {
		return 0, err
	}
	return len(p), nil
}

// Close closes the field and sends a successful response.
func (s *ResponseStream) Close() error {
	err := sporec.StreamClose(s.h)
	runtime.KeepAlive(s)
	return err
}

// Fail closes the field and sends an error response, preserving its output.
// Code and what must be nonempty; what must not contain NUL.
// A nil return means the error response was sent, not that the operation succeeded.
func (s *ResponseStream) Fail(code, what string) error {
	err := sporec.StreamFail(s.h, code, what)
	runtime.KeepAlive(s)
	return err
}

// SendResponseStart declares the string fields that will be streamed for a response.
// Finish the response with SendResponse (without repeating the fields) or SendResponseError.
func (c *Client) SendResponseStart(handle, command string, fields ...string) error {
	err := sporec.StreamSendStart(c.h, handle, command, fields)
	runtime.KeepAlive(c)
	return err
}

// SendResponseChunk immediately sends text for an open field. It does not add a
// line break to the value; include one in text if needed.
func (c *Client) SendResponseChunk(handle, field, text string) error {
	err := sporec.StreamSendChunk(c.h, handle, field, text)
	runtime.KeepAlive(c)
	return err
}

// SendResponseClose closes a field. Close all streamed fields before sending a
// successful completion.
func (c *Client) SendResponseClose(handle, field string) error {
	err := sporec.StreamSendClose(c.h, handle, field)
	runtime.KeepAlive(c)
	return err
}
