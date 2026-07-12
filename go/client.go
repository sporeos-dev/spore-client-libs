// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: Apache-2.0

package spore

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// DefaultSocketPath is the path to the spore hub Unix socket.
// The daemon runs as a system-level service, so the socket lives in the
// data root directory:
//   - Linux:   /var/lib/spore-os/spore.sock
//   - macOS:   /Library/Application Support/spore-os/spore.sock
//   - Windows: %LOCALAPPDATA%\spore-os\spore.sock
var DefaultSocketPath = defaultSocketPath()

func defaultSocketPath() string {
	switch runtime.GOOS {
	case "linux":
		return "/var/lib/spore-os/spore.sock"
	case "darwin":
		return "/Library/Application Support/spore-os/spore.sock"
	case "windows":
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			return localAppData + `\spore-os\spore.sock`
		}
		return `C:\Users\Default\AppData\Local\spore-os\spore.sock`
	default:
		return "/tmp/spore.sock"
	}
}

const handshakeTimeout = 5 * time.Second

type HandlerFunc func(call *Call)

// ResponseHandlerFunc is the signature for handlers registered with HandleResponse.
type ResponseHandlerFunc func(resp *Response)

// WitnessHandlerFunc is the signature for handlers registered with HandleWitness.
type WitnessHandlerFunc func(msg *WitnessMessage)

type Client struct {
	nodeID     string
	socketPath string

	conn   net.Conn
	reader *bufio.Reader

	mu                      sync.RWMutex
	handlers                map[string]HandlerFunc
	responseHandlers        map[string]ResponseHandlerFunc
	fallbackResponseHandler ResponseHandlerFunc
	witnessHandler          WitnessHandlerFunc

	waitersMu sync.Mutex
	waiters   map[string]chan *Response

	publishHandlers map[string]PublishHandlerFunc
	handleCounter   atomic.Int64

	listening atomic.Bool
}

func NewClient(nodeID string) *Client {
	return &Client{
		nodeID:           nodeID,
		socketPath:       DefaultSocketPath,
		handlers:         make(map[string]HandlerFunc),
		responseHandlers: make(map[string]ResponseHandlerFunc),
		waiters:          make(map[string]chan *Response),
		publishHandlers:  make(map[string]PublishHandlerFunc),
	}
}

func NewClientWithSocket(nodeID string, socketPath string) *Client {
	return &Client{
		nodeID:           nodeID,
		socketPath:       socketPath,
		handlers:         make(map[string]HandlerFunc),
		responseHandlers: make(map[string]ResponseHandlerFunc),
		waiters:          make(map[string]chan *Response),
		publishHandlers:  make(map[string]PublishHandlerFunc),
	}
}

func (c *Client) HandleRequest(subject string, handler HandlerFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[subject] = handler
}

// HandleResponse registers a response handler for a specific subject.
//
// When a response arrives during Listen() with no active Wait or WaitFor waiter,
// the subject of the response (the part after the colon in "~handle:subject") is
// matched against registered response handlers. Matching uses the same two-step
// logic as HandleRequest: exact match first, then short-name suffix match (the
// last dot-separated segment). For example, HandleResponse("get_time", ...) will
// match responses for "clock.get_time" or "com.example.clock.get_time".
//
// Active Wait or WaitFor waiters always take precedence — the handler is not
// called for a response consumed by a waiter.
//
// Calling HandleResponse with the same subject again replaces the previous handler.
func (c *Client) HandleResponse(subject string, handler ResponseHandlerFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.responseHandlers[subject] = handler
}

// HandleResponseFallback registers a catch-all handler that is called for any
// response that is not consumed by a waiter and does not match a HandleResponse
// subject handler. Only one fallback handler may be registered; calling this
// again replaces the previous one. Pass nil to clear it.
func (c *Client) HandleResponseFallback(handler ResponseHandlerFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fallbackResponseHandler = handler
}

// HandleWitness registers a handler that receives every witness copy delivered
// by the hub. Witness copies arrive prefixed with "witness " on the wire and
// are routed here rather than to request or response handlers.
//
// The WitnessMessage carries a Kind field (WitnessKindIncoming, Outgoing,
// Expanded, Event, or Node) and a SporeTime timestamp so the handler can
// filter or log selectively. Only one witness handler may be registered;
// calling this again replaces the previous one. Pass nil to clear it.
func (c *Client) HandleWitness(handler WitnessHandlerFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.witnessHandler = handler
}

func (c *Client) Connect() error {
	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	c.conn = conn
	c.reader = bufio.NewReader(conn)

	if err := c.handshake(); err != nil {
		c.conn.Close()
		c.conn = nil
		c.reader = nil
		return fmt.Errorf("handshake failed: %w", err)
	}

	return nil
}

func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	c.reader = nil
	return err
}

func (c *Client) Listen() error {
	if c.conn == nil {
		return errors.New("not connected")
	}

	c.listening.Store(true)
	defer func() {
		c.listening.Store(false)
		// Unblock any goroutines blocked in Wait/WaitFor so they don't leak.
		c.waitersMu.Lock()
		for k, ch := range c.waiters {
			close(ch)
			delete(c.waiters, k)
		}
		c.waitersMu.Unlock()
	}()

	for {
		raw, err := c.reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("connection closed: %w", err)
		}

		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}

		if strings.HasPrefix(raw, "witness ") {
			c.deliverWitness(raw)
			continue
		}

		if strings.HasPrefix(raw, "publish ") {
			c.deliverPublish(raw)
			continue
		}

		if strings.HasPrefix(raw, "~") {
			c.deliverResponse(raw)
			continue
		}

		// A bare "error code=... what=..." line (no ~ handle prefix) is the
		// hub's post-handshake rejection format (rejectAfterHandshake). Treat
		// it as a permanent HubError so the caller can decide not to retry.
		if strings.HasPrefix(raw, "error ") {
			call, _ := parseCall(raw)
			return &HubError{
				Code: call.ArgIf("code", "ConnectionFailure"),
				What: call.ArgIf("what", ""),
			}
		}

		call, err := parseCall(raw)
		if err != nil {
			continue
		}

		call.conn = c.conn
		c.dispatch(call)
	}
}

// deliverWitness parses a raw witness line and dispatches it to the registered
// witness handler, if any. It is a no-op when no handler is registered or
// when the message cannot be parsed.
func (c *Client) deliverWitness(raw string) {
	msg, err := parseWitness(raw)
	if err != nil {
		return
	}

	c.mu.RLock()
	handler := c.witnessHandler
	c.mu.RUnlock()

	if handler != nil {
		handler(msg)
	}
}

// deliverPublish parses a raw publish line and dispatches it to the registered
// callback for that topic, if any. It is a no-op when no callback is registered
// or when the message cannot be parsed.
func (c *Client) deliverPublish(raw string) {
	msg, err := parsePublish(raw)
	if err != nil {
		return
	}

	c.mu.RLock()
	handler := c.publishHandlers[msg.Topic]
	c.mu.RUnlock()

	if handler != nil {
		handler(msg)
	}
}

// deliverResponse routes an incoming response to an active waiter or the
// registered response handler for that subject. Delivery priority:
//
//  1. A WaitFor(handle) waiter registered for this exact handle.
//  2. A Wait() wildcard waiter (registered with the empty-string key).
//  3. A HandleResponse handler whose subject matches (exact, then short-name).
//
// Whichever route delivers first wins; the others are not called.
func (c *Client) deliverResponse(raw string) {
	resp, err := parseResponse(raw)
	if err != nil {
		return
	}

	c.waitersMu.Lock()

	// Specific handle waiter takes priority.
	if ch, ok := c.waiters[resp.Handle]; ok {
		delete(c.waiters, resp.Handle)
		c.waitersMu.Unlock()
		ch <- resp
		return
	}

	// Wildcard waiter (from Wait()) is next.
	if ch, ok := c.waiters[""]; ok {
		delete(c.waiters, "")
		c.waitersMu.Unlock()
		ch <- resp
		return
	}

	c.waitersMu.Unlock()

	// No active waiter — look up a registered response handler by subject.
	// Exact match first, then short-name suffix (last dot-segment).
	c.mu.RLock()
	handler := c.responseHandlers[resp.Subject]
	if handler == nil {
		if idx := strings.LastIndex(resp.Subject, "."); idx >= 0 {
			handler = c.responseHandlers[resp.Subject[idx+1:]]
		}
	}
	c.mu.RUnlock()

	if handler != nil {
		handler(resp)
		return
	}

	// No subject handler matched — call the fallback if one is registered.
	c.mu.RLock()
	fallback := c.fallbackResponseHandler
	c.mu.RUnlock()
	if fallback != nil {
		fallback(resp)
	}
}

func (c *Client) Send(message string) error {
	if c.conn == nil {
		return errors.New("not connected")
	}
	message = strings.TrimSpace(message)
	_, err := c.conn.Write([]byte(message + "\n"))
	return err
}

// SendAndWait sends a message and blocks until the response matching its handle
// arrives, or returns an error if the timeout elapses.
//
// The message must contain a handle token (e.g. "~h1"). If no handle is present,
// SendAndWait returns an error immediately rather than waiting for an arbitrary
// response.
//
// In listener mode (Listen() running in a goroutine), the waiter is registered
// before the message is sent, so there is no race between sending and receiving.
// In standalone mode, the read deadline is set on the connection directly.
//
// timeoutMs is the maximum time to wait in milliseconds.
func (c *Client) SendAndWait(message string, timeoutMs int) (*Response, error) {
	if c.conn == nil {
		return nil, errors.New("not connected")
	}

	message = strings.TrimSpace(message)
	handle := extractHandle(message)
	if handle == "" {
		return nil, errors.New("SendAndWait requires a handle token (e.g. ~h1) in the message")
	}
	timeout := time.Duration(timeoutMs) * time.Millisecond

	if c.listening.Load() {
		// Listener mode: register the waiter BEFORE sending to avoid a race
		// where the response arrives before we start waiting.
		ch := make(chan *Response, 1)
		c.waitersMu.Lock()
		c.waiters[handle] = ch
		c.waitersMu.Unlock()

		if _, err := c.conn.Write([]byte(message + "\n")); err != nil {
			c.waitersMu.Lock()
			delete(c.waiters, handle)
			c.waitersMu.Unlock()
			return nil, err
		}

		timer := time.NewTimer(timeout)
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

	// Standalone mode: set a read deadline, send, then read directly.
	c.conn.SetReadDeadline(time.Now().Add(timeout))
	defer c.conn.SetReadDeadline(time.Time{})

	if _, err := c.conn.Write([]byte(message + "\n")); err != nil {
		return nil, err
	}

	for {
		raw, err := c.reader.ReadString('\n')
		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				return nil, errors.New("send timed out waiting for response")
			}
			return nil, fmt.Errorf("connection closed: %w", err)
		}
		raw = strings.TrimSpace(raw)
		if raw == "" || !strings.HasPrefix(raw, "~") {
			continue
		}
		resp, err := parseResponse(raw)
		if err != nil {
			continue
		}
		if resp.Handle == handle {
			return resp, nil
		}
	}
}

// extractHandle scans a raw outgoing message string for the embedded handle
// token (~token). Returns the handle name (without "~"), or empty string if
// none is present.
func extractHandle(message string) string {
	fields := splitFields(message)
	for _, f := range fields[1:] { // first token is the subject
		if strings.HasPrefix(f, "~") {
			return strings.TrimPrefix(f, "~")
		}
	}
	return ""
}

// Wait blocks until the next response arrives and returns it, or until
// timeoutMs milliseconds have elapsed, in which case an error is returned.
//
// If Listen() is running in a goroutine (listener mode), Wait registers a
// channel waiter and blocks until Listen delivers a response. It takes
// precedence over any HandleResponse handler — the handler is not called for
// a response consumed here.
//
// If Listen() is not running (standalone mode), Wait reads directly from the
// socket, skipping inbound call messages until a response arrives.
//
// Note: do not call Wait from inside a HandleRequest() handler while Listen() is
// running. Listen() dispatches handlers synchronously, so calling Wait from
// within a handler will deadlock. Use a goroutine inside the handler instead.
func (c *Client) Wait(timeoutMs int) (*Response, error) {
	if c.conn == nil {
		return nil, errors.New("not connected")
	}

	if c.listening.Load() {
		// Listener mode: register a wildcard waiter; Listen() delivers to it.
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

	// Standalone mode: set a read deadline on the connection, then read directly.
	timeout := time.Duration(timeoutMs) * time.Millisecond
	c.conn.SetReadDeadline(time.Now().Add(timeout))
	defer c.conn.SetReadDeadline(time.Time{})

	for {
		raw, err := c.reader.ReadString('\n')
		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				return nil, errors.New("wait timed out")
			}
			return nil, fmt.Errorf("connection closed: %w", err)
		}
		raw = strings.TrimSpace(raw)
		if raw == "" || !strings.HasPrefix(raw, "~") {
			continue
		}
		return parseResponse(raw)
	}
}

// WaitFor blocks until a response with the given handle arrives, or until
// timeoutMs milliseconds have elapsed, in which case an error is returned.
//
// If Listen() is running in a goroutine (listener mode), WaitFor registers a
// channel waiter for the specific handle and blocks until Listen delivers it.
// Responses for other handles are not affected — they go to their own waiters
// or the HandleResponse handler as normal. WaitFor takes precedence over the
// HandleResponse handler for its handle.
//
// If Listen() is not running (standalone mode), WaitFor reads directly from
// the socket, discarding responses for other handles.
//
// Note: do not call WaitFor from inside a HandleRequest() handler while Listen() is
// running. Listen() dispatches handlers synchronously, so calling WaitFor from
// within a handler will deadlock. Use a goroutine inside the handler instead.
func (c *Client) WaitFor(handle string, timeoutMs int) (*Response, error) {
	if c.conn == nil {
		return nil, errors.New("not connected")
	}

	if c.listening.Load() {
		// Listener mode: register a specific handle waiter; Listen() delivers to it.
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

	// Standalone mode: set a read deadline on the connection, then read directly.
	timeout := time.Duration(timeoutMs) * time.Millisecond
	c.conn.SetReadDeadline(time.Now().Add(timeout))
	defer c.conn.SetReadDeadline(time.Time{})

	for {
		raw, err := c.reader.ReadString('\n')
		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				return nil, errors.New("wait timed out")
			}
			return nil, fmt.Errorf("connection closed: %w", err)
		}
		raw = strings.TrimSpace(raw)
		if raw == "" || !strings.HasPrefix(raw, "~") {
			continue
		}
		resp, err := parseResponse(raw)
		if err != nil {
			continue
		}
		if resp.Handle == handle {
			return resp, nil
		}
	}
}

// Subscribe registers a callback for incoming publish messages on the given topic
// and, on the first call for that topic, sends a subscribe request to the hub.
//
// If a callback is already registered for this topic, the callback is replaced
// without sending another subscribe request — the subscription on the hub side
// remains active.
//
// The callback fires on the Listen() goroutine. If the callback is slow or
// blocks, it delays all other incoming messages. Spin up a goroutine inside
// the callback if you need non-blocking processing.
//
// timeoutMs controls how long to wait for the hub's acknowledgement on the
// initial subscription. Subsequent calls (callback replacement) do not use it.
func (c *Client) Subscribe(topic string, callback PublishHandlerFunc, timeoutMs int) error {
	if c.conn == nil {
		return errors.New("not connected")
	}

	c.mu.Lock()
	_, alreadySubscribed := c.publishHandlers[topic]
	c.publishHandlers[topic] = callback
	c.mu.Unlock()

	if alreadySubscribed {
		// Just a callback swap — no wire message needed.
		return nil
	}

	handle := fmt.Sprintf("sub%d", c.handleCounter.Add(1))
	msg := fmt.Sprintf("SPORE.topic.subscribe topic=%s ~%s", topic, handle)

	resp, err := c.SendAndWait(msg, timeoutMs)
	if err != nil {
		// Roll back the registration so the caller can retry.
		c.mu.Lock()
		delete(c.publishHandlers, topic)
		c.mu.Unlock()
		return fmt.Errorf("subscribe failed: %w", err)
	}
	if !resp.OK {
		c.mu.Lock()
		delete(c.publishHandlers, topic)
		c.mu.Unlock()
		return fmt.Errorf("subscribe rejected: %s: %s", resp.ErrCode, resp.ErrWhat)
	}

	return nil
}

// Unsubscribe sends an unsubscribe request to the hub for the given topic and
// removes the registered publish callback. Returns an error if the hub rejects
// the request or the connection fails.
//
// If no callback was registered for the topic, the unsubscribe message is still
// sent (in case a previous session left a stale subscription on the hub).
func (c *Client) Unsubscribe(topic string, timeoutMs int) error {
	if c.conn == nil {
		return errors.New("not connected")
	}

	handle := fmt.Sprintf("sub%d", c.handleCounter.Add(1))
	msg := fmt.Sprintf("SPORE.topic.unsubscribe topic=%s ~%s", topic, handle)

	resp, err := c.SendAndWait(msg, timeoutMs)
	if err != nil {
		return fmt.Errorf("unsubscribe failed: %w", err)
	}
	if !resp.OK {
		return fmt.Errorf("unsubscribe rejected: %s: %s", resp.ErrCode, resp.ErrWhat)
	}

	c.mu.Lock()
	delete(c.publishHandlers, topic)
	c.mu.Unlock()

	return nil
}

func (c *Client) handshake() error {
	_, err := c.conn.Write([]byte(c.nodeID + "\n"))
	if err != nil {
		return err
	}

	c.conn.SetReadDeadline(time.Now().Add(handshakeTimeout))
	defer c.conn.SetReadDeadline(time.Time{})

	response, err := c.reader.ReadString('\n')
	if err != nil {
		return err
	}

	if strings.TrimSpace(response) != "OK" {
		return errors.New(strings.TrimSpace(response))
	}

	return nil
}

func (c *Client) dispatch(call *Call) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if handler, ok := c.handlers[call.Subject]; ok {
		handler(call)
		return
	}

	if idx := strings.LastIndex(call.Subject, "."); idx >= 0 {
		shortName := call.Subject[idx+1:]
		if handler, ok := c.handlers[shortName]; ok {
			handler(call)
			return
		}
	}
}
