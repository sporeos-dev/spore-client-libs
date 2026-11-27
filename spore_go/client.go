// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: Apache-2.0

package spore

import (
	"fmt"
	"runtime"

	sporec "github.com/sporeos-dev/spore-client-libs/spore_go/internal/spore_c"
	"github.com/sporeos-dev/spore-client-libs/spore_go/publish"
	"github.com/sporeos-dev/spore-client-libs/spore_go/request"
	"github.com/sporeos-dev/spore-client-libs/spore_go/response"
	"github.com/sporeos-dev/spore-client-libs/spore_go/witness"
)

const sendAndWaitTimeoutMs = 5000

type Client struct {
	nodeid string
	h      *sporec.Client
}

func New(nodeid string) *Client {
	c := &Client{
		nodeid: nodeid,
		h:      sporec.ClientCreate(nodeid),
	}
	if sporec.ClientHasError(c.h) {
		// the thought is that this really shouldn't fail
		// but if it does, something should catch it
		println("Error creating client:", sporec.ClientGetErrorCode(c.h), sporec.ClientGetErrorWhat(c.h))
		c.destroy()
		return nil
	}
	runtime.SetFinalizer(c, (*Client).destroy)
	return c
}

// WithReady sets the function that runs after the handshake, before the hub forwards any
// request to this node. It runs on its own thread once Listen starts, so it can use
// SendRequestAndWait. Returning an error means READY is not sent; send your own witness
// first if you want the reason in the logs.
func (c *Client) WithReady(fn func(*Client) error) *Client {
	sporec.ClientSetReady(c.h, func(_ *sporec.Client) bool {
		return fn(c) == nil
	})
	return c
}

// log to stdout &&
// send witness information
func (c *Client) WithDefaultErrorHandler() *Client {
	c.OnParseError(func(code, what, raw string) {
		fmt.Printf("Parse error: [%s] %s\n", code, what)
		fmt.Printf("Raw: %s\n", raw)
	})

	return c
}

func (c *Client) destroy() {
	if sporec.ClientIsConnected(c.h) {
		sporec.ClientDisconnect(c.h)
	}
	sporec.ClientDestroy(c.h)
	c.h = nil
}

func (c *Client) Connect() error {
	if sporec.ClientIsConnected(c.h) {
		return nil
	}
	sporec.ClientConnect(c.h)
	return c.error()
}

func (c *Client) Disconnect() error {
	if !sporec.ClientIsConnected(c.h) {
		return nil
	}
	sporec.ClientDisconnect(c.h)
	return c.error()
}

// listen
func (c *Client) Listen() error {
	if !sporec.ClientIsConnected(c.h) {
		return fmt.Errorf("client is not connected")
	}
	sporec.ClientListen(c.h)
	return c.error()
}

// send
func (c *Client) SendRaw(raw string) error {
	sporec.ClientSendRaw(c.h, raw)
	return c.error()
}

func (c *Client) SendRequest(r *request.Request) error {
	sporec.RequestSend(c.h, r.H())
	return c.error()
}

// SendRequestAndWait blocks until the response arrives. Needs Listen() running in
// another goroutine, so it can't be called from a handler. The error is for a timeout
// or send failure; a hub error comes back as the *ResponseError.
func (c *Client) SendRequestAndWait(r *request.Request) (*response.Response, *response.ResponseError, error) {
	if !sporec.ClientSendAndWait(c.h, r.H(), sendAndWaitTimeoutMs) {
		if err := c.error(); err != nil {
			return nil, nil, err
		}
		return nil, nil, fmt.Errorf("no response to %s (handle %s)", r.Command(), r.Handle())
	}
	ok, rerr := response.Parse(sporec.RequestGetResponse(r.H()))
	if ok == nil && rerr == nil {
		return nil, nil, fmt.Errorf("unparsable response to %s", r.Command())
	}
	return ok, rerr, nil
}

func (c *Client) SendResponse(r *response.Response) error {
	sporec.ResponseSend(c.h, r.H())
	return c.error()
}

func (c *Client) SendResponseError(e *response.ResponseError) error {
	sporec.ResponseErrorSend(c.h, e.H())
	return c.error()
}

func (c *Client) SendWitness(w *witness.Witness) error {
	sporec.WitnessSend(c.h, w.H())
	return c.error()
}

func (c *Client) SendPublish(p *publish.Publish) error {
	sporec.PublishSend(c.h, p.H())
	return c.error()
}

// private
func (c *Client) error() error {
	if sporec.ClientHasError(c.h) {
		return fmt.Errorf(`[%s] %s`, sporec.ClientGetErrorCode(c.h), sporec.ClientGetErrorWhat(c.h))
	}
	return nil
}

//
// handlers
//

type Handler struct {
	h *sporec.Handler
}

func (c *Client) OnRequest(fn func(*request.Request)) *Handler {
	h := sporec.ClientOnRequest(c.h, func(_ *sporec.Client, r *sporec.Request) {
		fn(request.FromC(r))
	})
	return &Handler{h: h}
}

func (c *Client) OnResponse(fn func(*response.Response, *response.ResponseError)) *Handler {
	h := sporec.ClientOnResponse(c.h, func(_ *sporec.Client, r *sporec.Response, e *sporec.ResponseError) {
		var resp *response.Response
		var rerr *response.ResponseError
		if r != nil {
			resp = response.FromC(r)
		}
		if e != nil {
			rerr = response.ErrorFromC(e)
		}
		fn(resp, rerr)
	})
	return &Handler{h: h}
}

// OnStream receives individual live chunks. Their data remains valid after the callback.
// The final response goes to OnResponse, or to the waiting SendRequestAndWait call.
// Like other handlers, it runs on the listener and must not call SendRequestAndWait.
// It returns nil if fn is nil or the handler cannot be registered.
func (c *Client) OnStream(fn func(*ResponseStream)) *Handler {
	if fn == nil {
		return nil
	}
	h := sporec.ClientOnStream(c.h, func(_ *sporec.Client, handle, field, text string) {
		fn(&ResponseStream{handle: handle, field: field, text: text})
	})
	if h == nil {
		return nil
	}
	return &Handler{h: h}
}

func (c *Client) OnWitness(fn func(*witness.Witness)) *Handler {
	h := sporec.ClientOnWitness(c.h, func(_ *sporec.Client, w *sporec.Witness) {
		fn(witness.FromC(w))
	})
	return &Handler{h: h}
}

func (c *Client) OnPublish(fn func(*publish.Publish)) *Handler {
	h := sporec.ClientOnPublish(c.h, func(_ *sporec.Client, p *sporec.Publish) {
		fn(publish.FromC(p))
	})
	return &Handler{h: h}
}

func (c *Client) OnParseError(fn func(code, what, raw string)) *Handler {
	h := sporec.ClientOnParseError(c.h, func(_ *sporec.Client, code, what, raw string) {
		fn(code, what, raw)
	})
	return &Handler{h: h}
}

func (c *Client) Off(handler *Handler) {
	sporec.ClientOffHandler(c.h, handler.h)
}
