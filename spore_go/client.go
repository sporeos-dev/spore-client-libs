// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: Apache-2.0

package spore

// This file is imported by C via //export in callbacks.go; keep the preamble
// declaration-only so that both files can share import "C".

/*
#cgo CFLAGS: -I../spore_c/include
#cgo darwin LDFLAGS: -L../dist -lspore_c -lc++
#cgo linux  LDFLAGS: -L../dist -lspore_c -lstdc++
#cgo windows LDFLAGS: -L../dist -lspore_c -lstdc++

#include "spore_c.h"
#include <stdlib.h>

// Forward-declarations for the Go-exported callbacks defined in callbacks.go.
extern void goOnRequest(spore_client_t*, const spore_request_t*);
extern void goOnResponse(spore_client_t*, const spore_response_t*, const spore_response_error_t*);
extern void goOnWitness(spore_client_t*, const spore_witness_t*);
extern void goOnPublish(spore_client_t*, const spore_publish_t*);

// Trampolines: pass the Go callback address to each register_*_handler call.
// Required because Go function values cannot be used directly as C function pointers.
static spore_handler_t* cRegisterRequest(spore_client_t* c) {
    return spore_client_register_request_handler(c, (spore_request_fn)goOnRequest);
}
static spore_handler_t* cRegisterResponse(spore_client_t* c) {
    return spore_client_register_response_handler(c, (spore_response_fn)goOnResponse);
}
static spore_handler_t* cRegisterWitness(spore_client_t* c) {
    return spore_client_register_witness_handler(c, (spore_witness_fn)goOnWitness);
}
static spore_handler_t* cRegisterPublish(spore_client_t* c) {
    return spore_client_register_publish_handler(c, (spore_publish_fn)goOnPublish);
}
*/
import "C"
import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// clientRegistry maps C client pointer address → *Client for callback dispatch.
var clientRegistry sync.Map

func registerClient(cPtr *C.spore_client_t, c *Client) {
	clientRegistry.Store(uintptr(unsafe.Pointer(cPtr)), c)
}

func unregisterClient(cPtr *C.spore_client_t) {
	clientRegistry.Delete(uintptr(unsafe.Pointer(cPtr)))
}

func lookupClient(cPtr *C.spore_client_t) (*Client, bool) {
	v, ok := clientRegistry.Load(uintptr(unsafe.Pointer(cPtr)))
	if !ok {
		return nil, false
	}
	return v.(*Client), true
}

// HandlerFunc handles an incoming request routed by the hub.
type HandlerFunc func(call *Call)

// ResponseHandlerFunc handles a response to an outgoing call.
type ResponseHandlerFunc func(resp *Response)

// WitnessHandlerFunc handles a witness copy delivered by the hub.
type WitnessHandlerFunc func(msg *WitnessMessage)

// PublishHandlerFunc handles a pub/sub message delivered by the hub.
type PublishHandlerFunc func(msg *PublishMessage)

// Client manages a connection to the Spore daemon via the spore_c C++ library.
type Client struct {
	nodeID string

	mu               sync.RWMutex
	handlers         map[string]HandlerFunc
	responseHandlers map[string]ResponseHandlerFunc
	fallbackResponse ResponseHandlerFunc
	witnessHandler   WitnessHandlerFunc
	publishHandlers  map[string]PublishHandlerFunc

	waitersMu sync.Mutex
	waiters   map[string]chan *Response

	handleCounter atomic.Int64
	listening     atomic.Bool

	cClient *C.spore_client_t
}

// NewClient returns a client that identifies itself to the daemon as nodeID.
func NewClient(nodeID string) *Client {
	return &Client{
		nodeID:           nodeID,
		handlers:         make(map[string]HandlerFunc),
		responseHandlers: make(map[string]ResponseHandlerFunc),
		publishHandlers:  make(map[string]PublishHandlerFunc),
		waiters:          make(map[string]chan *Response),
	}
}

// HandleRequest registers a handler for incoming requests whose subject matches
// name. Exact match is tried first; if none, the last dot-segment is tried.
func (c *Client) HandleRequest(subject string, handler HandlerFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[subject] = handler
}

// HandleResponse registers a handler for incoming responses with the given subject.
func (c *Client) HandleResponse(subject string, handler ResponseHandlerFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.responseHandlers[subject] = handler
}

// HandleResponseFallback registers a catch-all for responses not matched by
// a specific HandleResponse subject or an active Wait / WaitFor waiter.
func (c *Client) HandleResponseFallback(handler ResponseHandlerFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fallbackResponse = handler
}

// HandleWitness registers a handler for witness copies delivered by the hub.
func (c *Client) HandleWitness(handler WitnessHandlerFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.witnessHandler = handler
}

// Connect creates the underlying C client, connects to the Spore daemon, and
// registers the shared C→Go callbacks for each event type.
func (c *Client) Connect() error {
	nodeID := C.CString(c.nodeID)
	defer C.free(unsafe.Pointer(nodeID))

	cClient := C.spore_client_create(nodeID)
	if cClient == nil {
		return errors.New("spore_client_create returned null (empty node ID?)")
	}
	c.cClient = cClient
	registerClient(cClient, c)

	C.spore_client_connect(cClient)
	if C.spore_client_has_error(cClient) {
		code := C.GoString(C.spore_client_get_error_code(cClient))
		what := C.GoString(C.spore_client_get_error_what(cClient))
		unregisterClient(cClient)
		C.spore_client_destroy(cClient)
		c.cClient = nil
		return fmt.Errorf("connect: %s: %s", code, what)
	}

	// Register the single shared C callback for each event type.
	C.cRegisterRequest(cClient)
	C.cRegisterResponse(cClient)
	C.cRegisterWitness(cClient)
	C.cRegisterPublish(cClient)

	return nil
}

// Close disconnects from the daemon and frees the C client.
func (c *Client) Close() error {
	if c.cClient == nil {
		return nil
	}
	unregisterClient(c.cClient)

	c.waitersMu.Lock()
	for k, ch := range c.waiters {
		close(ch)
		delete(c.waiters, k)
	}
	c.waitersMu.Unlock()

	C.spore_client_disconnect(c.cClient)
	C.spore_client_destroy(c.cClient)
	c.cClient = nil
	return nil
}

// Listen blocks and dispatches incoming events through the registered handlers
// until the connection is closed or the C listen loop returns an error.
// Typically called in a goroutine after Connect.
func (c *Client) Listen() error {
	if c.cClient == nil {
		return errors.New("not connected")
	}

	c.listening.Store(true)
	defer func() {
		c.listening.Store(false)
		c.waitersMu.Lock()
		for k, ch := range c.waiters {
			close(ch)
			delete(c.waiters, k)
		}
		c.waitersMu.Unlock()
	}()

	C.spore_client_listen(c.cClient)

	if c.cClient != nil && C.spore_client_has_error(c.cClient) {
		code := C.GoString(C.spore_client_get_error_code(c.cClient))
		what := C.GoString(C.spore_client_get_error_what(c.cClient))
		return fmt.Errorf("listen: %s: %s", code, what)
	}
	return nil
}

// Send writes a raw wire-format message to the daemon.
func (c *Client) Send(message string) error {
	if c.cClient == nil {
		return errors.New("not connected")
	}
	wire := message + "\n"
	cs := C.CString(wire)
	defer C.free(unsafe.Pointer(cs))
	C.spore_client_send_raw(c.cClient, cs, C.size_t(len(wire)))
	if C.spore_client_has_error(c.cClient) {
		code := C.GoString(C.spore_client_get_error_code(c.cClient))
		what := C.GoString(C.spore_client_get_error_what(c.cClient))
		return fmt.Errorf("send: %s: %s", code, what)
	}
	return nil
}

// SendAndWait sends a message and blocks until the response for its handle
// arrives. Requires Listen to be running in a goroutine. The message must
// contain a handle token (e.g. "~h1").
func (c *Client) SendAndWait(message string, timeoutMs int) (*Response, error) {
	if c.cClient == nil {
		return nil, errors.New("not connected")
	}
	if !c.listening.Load() {
		return nil, errors.New("SendAndWait requires Listen() running in a goroutine")
	}

	message = strings.TrimSpace(message)
	handle := extractHandle(message)
	if handle == "" {
		return nil, errors.New("SendAndWait: message has no handle token (~h1)")
	}

	ch := make(chan *Response, 1)
	c.waitersMu.Lock()
	c.waiters[handle] = ch
	c.waitersMu.Unlock()

	if err := c.Send(message); err != nil {
		c.waitersMu.Lock()
		delete(c.waiters, handle)
		c.waitersMu.Unlock()
		return nil, err
	}

	timer := time.NewTimer(time.Duration(timeoutMs) * time.Millisecond)
	defer timer.Stop()
	select {
	case resp, ok := <-ch:
		if !ok {
			return nil, errors.New("connection closed while waiting for response")
		}
		return resp, nil
	case <-timer.C:
		c.waitersMu.Lock()
		delete(c.waiters, handle)
		c.waitersMu.Unlock()
		return nil, errors.New("send timed out waiting for response")
	}
}

// Wait blocks until the next response arrives. Requires Listen to be running.
func (c *Client) Wait(timeoutMs int) (*Response, error) {
	if c.cClient == nil {
		return nil, errors.New("not connected")
	}
	if !c.listening.Load() {
		return nil, errors.New("Wait requires Listen() running in a goroutine")
	}

	ch := make(chan *Response, 1)
	c.waitersMu.Lock()
	c.waiters[""] = ch
	c.waitersMu.Unlock()

	timer := time.NewTimer(time.Duration(timeoutMs) * time.Millisecond)
	defer timer.Stop()
	select {
	case resp, ok := <-ch:
		if !ok {
			return nil, errors.New("connection closed while waiting for response")
		}
		return resp, nil
	case <-timer.C:
		c.waitersMu.Lock()
		delete(c.waiters, "")
		c.waitersMu.Unlock()
		return nil, errors.New("wait timed out")
	}
}

// WaitFor blocks until a response with the given handle arrives. Requires Listen.
func (c *Client) WaitFor(handle string, timeoutMs int) (*Response, error) {
	if c.cClient == nil {
		return nil, errors.New("not connected")
	}
	if !c.listening.Load() {
		return nil, errors.New("WaitFor requires Listen() running in a goroutine")
	}

	ch := make(chan *Response, 1)
	c.waitersMu.Lock()
	c.waiters[handle] = ch
	c.waitersMu.Unlock()

	timer := time.NewTimer(time.Duration(timeoutMs) * time.Millisecond)
	defer timer.Stop()
	select {
	case resp, ok := <-ch:
		if !ok {
			return nil, errors.New("connection closed while waiting for response")
		}
		return resp, nil
	case <-timer.C:
		c.waitersMu.Lock()
		delete(c.waiters, handle)
		c.waitersMu.Unlock()
		return nil, errors.New("wait timed out")
	}
}

// Subscribe registers a callback for publish messages on topic and sends a
// subscribe request to the hub on the first registration.
func (c *Client) Subscribe(topic string, callback PublishHandlerFunc, timeoutMs int) error {
	if c.cClient == nil {
		return errors.New("not connected")
	}

	c.mu.Lock()
	_, already := c.publishHandlers[topic]
	c.publishHandlers[topic] = callback
	c.mu.Unlock()

	if already {
		return nil
	}

	handle := fmt.Sprintf("sub%d", c.handleCounter.Add(1))
	resp, err := c.SendAndWait(
		fmt.Sprintf("SPORE.topic.subscribe topic=%s ~%s", topic, handle),
		timeoutMs,
	)
	if err != nil {
		c.mu.Lock()
		delete(c.publishHandlers, topic)
		c.mu.Unlock()
		return fmt.Errorf("subscribe: %w", err)
	}
	if !resp.OK {
		c.mu.Lock()
		delete(c.publishHandlers, topic)
		c.mu.Unlock()
		return fmt.Errorf("subscribe rejected: %s: %s", resp.ErrCode, resp.ErrWhat)
	}
	return nil
}

// Unsubscribe sends an unsubscribe request and removes the callback.
func (c *Client) Unsubscribe(topic string, timeoutMs int) error {
	if c.cClient == nil {
		return errors.New("not connected")
	}

	handle := fmt.Sprintf("sub%d", c.handleCounter.Add(1))
	resp, err := c.SendAndWait(
		fmt.Sprintf("SPORE.topic.unsubscribe topic=%s ~%s", topic, handle),
		timeoutMs,
	)
	if err != nil {
		return fmt.Errorf("unsubscribe: %w", err)
	}
	if !resp.OK {
		return fmt.Errorf("unsubscribe rejected: %s: %s", resp.ErrCode, resp.ErrWhat)
	}

	c.mu.Lock()
	delete(c.publishHandlers, topic)
	c.mu.Unlock()
	return nil
}

// NextHandle returns a unique handle token (e.g. "h1", "h2") for outgoing calls.
func (c *Client) NextHandle() string {
	return fmt.Sprintf("h%d", c.handleCounter.Add(1))
}

// dispatchRequest routes an incoming request to the registered Go handler.
func (c *Client) dispatchRequest(call *Call) {
	c.mu.RLock()
	h := c.handlers[call.Subject]
	if h == nil {
		if idx := strings.LastIndex(call.Subject, "."); idx >= 0 {
			h = c.handlers[call.Subject[idx+1:]]
		}
	}
	c.mu.RUnlock()
	if h != nil {
		h(call)
	}
}

// dispatchResponse routes a response to a waiter or the registered handler.
func (c *Client) dispatchResponse(resp *Response) {
	c.waitersMu.Lock()
	if ch, ok := c.waiters[resp.Handle]; ok {
		delete(c.waiters, resp.Handle)
		c.waitersMu.Unlock()
		ch <- resp
		return
	}
	if ch, ok := c.waiters[""]; ok {
		delete(c.waiters, "")
		c.waitersMu.Unlock()
		ch <- resp
		return
	}
	c.waitersMu.Unlock()

	c.mu.RLock()
	h := c.responseHandlers[resp.Subject]
	if h == nil {
		if idx := strings.LastIndex(resp.Subject, "."); idx >= 0 {
			h = c.responseHandlers[resp.Subject[idx+1:]]
		}
	}
	fallback := c.fallbackResponse
	c.mu.RUnlock()

	if h != nil {
		h(resp)
		return
	}
	if fallback != nil {
		fallback(resp)
	}
}

func (c *Client) dispatchWitness(msg *WitnessMessage) {
	c.mu.RLock()
	h := c.witnessHandler
	c.mu.RUnlock()
	if h != nil {
		h(msg)
	}
}

func (c *Client) dispatchPublish(msg *PublishMessage) {
	c.mu.RLock()
	h := c.publishHandlers[msg.Topic]
	c.mu.RUnlock()
	if h != nil {
		h(msg)
	}
}

func extractHandle(message string) string {
	for _, f := range strings.Fields(message) {
		if strings.HasPrefix(f, "~") {
			return strings.TrimPrefix(f, "~")
		}
	}
	return ""
}
