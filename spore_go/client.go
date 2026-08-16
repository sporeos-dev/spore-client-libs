package spore

import (
	"fmt"
	"runtime"
	"strings"

	sporec "github.com/sporeos-dev/spore-client-libs/spore_go/internal/spore_c"
	"github.com/sporeos-dev/spore-client-libs/spore_go/publish"
	"github.com/sporeos-dev/spore-client-libs/spore_go/request"
	"github.com/sporeos-dev/spore-client-libs/spore_go/response"
	"github.com/sporeos-dev/spore-client-libs/spore_go/witness"
)

type Client struct {
	nodeid string
	h *sporec.Client
}

func New(nodeid string) *Client {
	c := &Client{nodeid: nodeid, h: sporec.ClientCreate(nodeid)}
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

// log to stdout &&
// send witness information
func (c *Client) WithDefaultErrorHandler() *Client {
	c.OnParseError(func(code, what, raw string) {
		fmt.Printf("Parse error: [%s] %s\n", code, what)
		fmt.Printf("Raw: %s\n", raw)
		c.SendWitness(witness.New(fmt.Sprintf(`Parse error: [%s] %s`, code, what)))
		raw = strings.ReplaceAll(raw, "\n", "\\n")
		raw = strings.ReplaceAll(raw, "\"", "\\\"")
		c.SendWitness(witness.New(fmt.Sprintf(`Raw: %s`, raw)))
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
