// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: Apache-2.0

package spore

import (
	"bufio"
	"net"
	"strings"
	"testing"
	"time"
)

func TestParseCall_SimpleSubject(t *testing.T) {
	call, err := parseCall("aardvark")
	if err != nil {
		t.Fatal(err)
	}
	if call.Subject != "aardvark" {
		t.Errorf("expected subject 'aardvark', got '%s'", call.Subject)
	}
}

func TestParseCall_DottedSubject(t *testing.T) {
	call, err := parseCall("node-a.aardvark")
	if err != nil {
		t.Fatal(err)
	}
	if call.Subject != "node-a.aardvark" {
		t.Errorf("expected subject 'node-a.aardvark', got '%s'", call.Subject)
	}
}

func TestParseCall_WithArgs(t *testing.T) {
	call, err := parseCall("clock.get_time timezone=UTC")
	if err != nil {
		t.Fatal(err)
	}
	if call.Subject != "clock.get_time" {
		t.Errorf("expected subject 'clock.get_time', got '%s'", call.Subject)
	}
	if call.Arg("timezone") != "UTC" {
		t.Errorf("expected timezone 'UTC', got '%s'", call.Arg("timezone"))
	}
}

func TestParseCall_WithHandle(t *testing.T) {
	call, err := parseCall("clock.get_time timezone=UTC ~h1")
	if err != nil {
		t.Fatal(err)
	}
	if call.Handle != "h1" {
		t.Errorf("expected handle 'h1', got '%s'", call.Handle)
	}
}

func TestParseCall_WithCast(t *testing.T) {
	call, err := parseCall("clock.get_time timezone=UTC ~h1 cast=dev.sporeos.cli")
	if err != nil {
		t.Fatal(err)
	}
	if call.Arg("cast") != "dev.sporeos.cli" {
		t.Errorf("expected cast 'dev.sporeos.cli', got '%s'", call.Arg("cast"))
	}
}

func TestParseCall_WithQuotedValue(t *testing.T) {
	call, err := parseCall("logger.append entry=\"hello world\" ~l1")
	if err != nil {
		t.Fatal(err)
	}
	if call.Arg("entry") != "hello world" {
		t.Errorf("expected entry 'hello world', got '%s'", call.Arg("entry"))
	}
}

func TestParseCall_WithFlags(t *testing.T) {
	call, err := parseCall("filesystem.read path=/tmp recursive verbose")
	if err != nil {
		t.Fatal(err)
	}
	if !call.HasFlag("recursive") {
		t.Error("expected 'recursive' flag")
	}
	if !call.HasFlag("verbose") {
		t.Error("expected 'verbose' flag")
	}
	if call.Arg("path") != "/tmp" {
		t.Errorf("expected path '/tmp', got '%s'", call.Arg("path"))
	}
}

func TestParseCall_ReservedKeywordsSkipped(t *testing.T) {
	call, err := parseCall("clock.get_time ok capture=dev.sporeos.clock time=now")
	if err != nil {
		t.Fatal(err)
	}
	if !call.HasFlag("ok") {
		t.Error("'ok' should be in flags")
	}
	if call.Arg("capture") != "dev.sporeos.clock" {
		t.Errorf("expected capture 'dev.sporeos.clock', got '%s'", call.Arg("capture"))
	}
	if call.Arg("time") != "now" {
		t.Errorf("expected time 'now', got '%s'", call.Arg("time"))
	}
}

func TestParseCall_Empty(t *testing.T) {
	_, err := parseCall("")
	if err == nil {
		t.Error("expected error for empty message")
	}
}

func TestParseCall_FullProtocolMessage(t *testing.T) {
	call, err := parseCall("clock.get_time timezone=America/New_York ~h1 cast=dev.sporeos.cli")
	if err != nil {
		t.Fatal(err)
	}
	if call.Subject != "clock.get_time" {
		t.Errorf("subject: got '%s'", call.Subject)
	}
	if call.Arg("timezone") != "America/New_York" {
		t.Errorf("timezone: got '%s'", call.Arg("timezone"))
	}
	if call.Handle != "h1" {
		t.Errorf("handle: got '%s'", call.Handle)
	}
	if call.Arg("cast") != "dev.sporeos.cli" {
		t.Errorf("cast: got '%s'", call.Arg("cast"))
	}
}

func TestSplitFields_Simple(t *testing.T) {
	fields := splitFields("foo bar baz")
	if len(fields) != 3 {
		t.Fatalf("expected 3 fields, got %d: %v", len(fields), fields)
	}
}

func TestSplitFields_QuotedString(t *testing.T) {
	fields := splitFields("foo bar=\"hello world\" baz")
	if len(fields) != 3 {
		t.Fatalf("expected 3 fields, got %d: %v", len(fields), fields)
	}
	if fields[1] != "bar=\"hello world\"" {
		t.Errorf("expected bar=\"hello world\", got '%s'", fields[1])
	}
}

func TestSplitFields_MultipleSpaces(t *testing.T) {
	fields := splitFields("foo   bar   baz")
	if len(fields) != 3 {
		t.Fatalf("expected 3 fields, got %d: %v", len(fields), fields)
	}
}

func TestArgIf_Found(t *testing.T) {
	call := &Call{args: map[string]string{"timezone": "UTC"}}
	if call.ArgIf("timezone", "PST") != "UTC" {
		t.Error("ArgIf should return existing value")
	}
}

func TestArgIf_NotFound(t *testing.T) {
	call := &Call{args: map[string]string{}}
	if call.ArgIf("missing", "PST") != "PST" {
		t.Error("ArgIf should return default for missing key")
	}
}

func TestNewClient(t *testing.T) {
	client := NewClient("dev.test.app")
	if client.nodeID != "dev.test.app" {
		t.Errorf("expected nodeID 'dev.test.app', got '%s'", client.nodeID)
	}
	if client.socketPath != DefaultSocketPath {
		t.Errorf("expected default socket path, got '%s'", client.socketPath)
	}
}

func TestNewClientWithSocket(t *testing.T) {
	client := NewClientWithSocket("dev.test.app", "/custom/path.sock")
	if client.socketPath != "/custom/path.sock" {
		t.Errorf("expected custom socket path, got '%s'", client.socketPath)
	}
}

func TestDispatch_ExactMatch(t *testing.T) {
	client := NewClient("dev.test.app")
	called := false
	client.HandleRequest("aardvark", func(call *Call) {
		called = true
	})
	call := &Call{Subject: "aardvark", args: map[string]string{}, flags: map[string]bool{}}
	client.dispatch(call)
	if !called {
		t.Error("handler was not called for exact match")
	}
}

func TestDispatch_ShortNameMatch(t *testing.T) {
	client := NewClient("dev.test.app")
	called := false
	client.HandleRequest("aardvark", func(call *Call) {
		called = true
	})
	call := &Call{Subject: "a.aardvark", args: map[string]string{}, flags: map[string]bool{}}
	client.dispatch(call)
	if !called {
		t.Error("handler was not called for short name match")
	}
}

func TestDispatch_FullFormMatch(t *testing.T) {
	client := NewClient("dev.test.app")
	called := false
	client.HandleRequest("get_time", func(call *Call) {
		called = true
	})
	call := &Call{Subject: "com.example.clock.get_time", args: map[string]string{}, flags: map[string]bool{}}
	client.dispatch(call)
	if !called {
		t.Error("handler was not called for full form match")
	}
}

func TestDispatch_NoMatch(t *testing.T) {
	client := NewClient("dev.test.app")
	called := false
	client.HandleRequest("aardvark", func(call *Call) {
		called = true
	})
	call := &Call{Subject: "unknown_command", args: map[string]string{}, flags: map[string]bool{}}
	client.dispatch(call)
	if called {
		t.Error("handler should not be called for non-matching subject")
	}
}

type mockConn struct {
	net.Conn
	written []byte
}

func (m *mockConn) Write(b []byte) (int, error) {
	m.written = append(m.written, b...)
	return len(b), nil
}

func (m *mockConn) SetReadDeadline(t time.Time) error { return nil }

func TestReply_WithHandle(t *testing.T) {
	mock := &mockConn{}
	call := &Call{
		Subject: "clock.get_time",
		Handle:  "h1",
		args:    map[string]string{},
		flags:   map[string]bool{},
		conn:    mock,
	}
	err := call.Reply(map[string]string{"time": "2026-03-13T14:32:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	response := string(mock.written)
	if response[:len("~h1:clock.get_time")] != "~h1:clock.get_time" {
		t.Errorf("expected response to start with '~h1:clock.get_time', got '%s'", response)
	}
	if !stringContains(response, "time=2026-03-13T14:32:00Z") {
		t.Errorf("expected response to contain time arg, got '%s'", response)
	}
	if response[len(response)-1] != '\n' {
		t.Error("response should end with newline")
	}
}

func TestReply_WithoutHandle(t *testing.T) {
	mock := &mockConn{}
	call := &Call{
		Subject: "clock.get_time",
		Handle:  "",
		args:    map[string]string{},
		flags:   map[string]bool{},
		conn:    mock,
	}
	err := call.Reply(map[string]string{"time": "now"})
	if err != nil {
		t.Fatal(err)
	}
	response := string(mock.written)
	if response[:len("clock.get_time")] != "clock.get_time" {
		t.Errorf("expected response to start with subject, got '%s'", response)
	}
	if response[0] == '~' {
		t.Error("response without handle should not start with ~")
	}
}

func TestReply_Void(t *testing.T) {
	mock := &mockConn{}
	call := &Call{
		Subject: "logger.append",
		Handle:  "l1",
		args:    map[string]string{},
		flags:   map[string]bool{},
		conn:    mock,
	}
	err := call.Reply(nil)
	if err != nil {
		t.Fatal(err)
	}
	response := string(mock.written)
	if response != "~l1:logger.append\n" {
		t.Errorf("expected '~l1:logger.append\\n', got '%s'", response)
	}
}

func TestReply_QuotedValues(t *testing.T) {
	mock := &mockConn{}
	call := &Call{
		Subject: "test.cmd",
		Handle:  "h1",
		args:    map[string]string{},
		flags:   map[string]bool{},
		conn:    mock,
	}
	err := call.Reply(map[string]string{"message": "hello world"})
	if err != nil {
		t.Fatal(err)
	}
	response := string(mock.written)
	if !stringContains(response, "message=\"hello world\"") {
		t.Errorf("values with spaces should be quoted, got '%s'", response)
	}
}

func TestCancel_WithHandle(t *testing.T) {
	mock := &mockConn{}
	call := &Call{
		Subject: "dialog.file_picker",
		Handle:  "h1",
		args:    map[string]string{},
		flags:   map[string]bool{},
		conn:    mock,
	}
	err := call.Cancel()
	if err != nil {
		t.Fatal(err)
	}
	response := strings.TrimSpace(string(mock.written))
	if response != "~h1:dialog.file_picker cancelled" {
		t.Errorf("expected cancelled response, got '%s'", response)
	}
}

func TestCancel_WithoutHandle(t *testing.T) {
	mock := &mockConn{}
	call := &Call{
		Subject: "dialog.file_picker",
		Handle:  "",
		args:    map[string]string{},
		flags:   map[string]bool{},
		conn:    mock,
	}
	err := call.Cancel()
	if err != nil {
		t.Fatal(err)
	}
	response := strings.TrimSpace(string(mock.written))
	if response != "dialog.file_picker cancelled" {
		t.Errorf("expected cancelled response, got '%s'", response)
	}
}

func TestError_WithHandle(t *testing.T) {
	mock := &mockConn{}
	call := &Call{
		Subject: "clock.get_time",
		Handle:  "h1",
		args:    map[string]string{},
		flags:   map[string]bool{},
		conn:    mock,
	}
	err := call.ErrorCustom("clock.err.invalid_timezone", "Unknown timezone: Fakezone")
	if err != nil {
		t.Fatal(err)
	}
	response := string(mock.written)
	if response[:len("~h1:clock.get_time")] != "~h1:clock.get_time" {
		t.Errorf("error should start with handle:subject, got '%s'", response)
	}
	if !stringContains(response, "custom_error") {
		t.Errorf("custom error response should contain 'custom_error' flag, got '%s'", response)
	}
	if stringContains(response, " error ") || strings.HasSuffix(strings.TrimSpace(response), " error") {
		t.Errorf("custom error response must not contain plain 'error' flag, got '%s'", response)
	}
	if !stringContains(response, "code=clock.err.invalid_timezone") {
		t.Errorf("error should contain code, got '%s'", response)
	}
	if !stringContains(response, "what=\"Unknown timezone: Fakezone\"") {
		t.Errorf("error should contain quoted what, got '%s'", response)
	}
	if !stringContains(response, "node_error") {
		t.Errorf("error should contain origin flag 'node_error', got '%s'", response)
	}
}

func TestError_WithoutHandle(t *testing.T) {
	mock := &mockConn{}
	call := &Call{
		Subject: "clock.get_time",
		Handle:  "",
		args:    map[string]string{},
		flags:   map[string]bool{},
		conn:    mock,
	}
	err := call.ErrorCustom("clock.err.fail", "oops")
	if err != nil {
		t.Fatal(err)
	}
	response := string(mock.written)
	if response[:len("clock.get_time")] != "clock.get_time" {
		t.Errorf("expected response to start with subject, got '%s'", response)
	}
	if !stringContains(response, "custom_error code=clock.err.fail what=oops") {
		t.Errorf("expected custom_error fields, got '%s'", response)
	}
	if !stringContains(response, "node_error") {
		t.Errorf("error should contain origin flag 'node_error', got '%s'", response)
	}
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// --- parseResponse tests ---

func TestParseResponse_Success(t *testing.T) {
	resp, err := parseResponse("~h1:clock.get_time time=2026-03-13T14:32:00Z ok capture=com.example.clock")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Handle != "h1" {
		t.Errorf("expected handle 'h1', got '%s'", resp.Handle)
	}
	if resp.Subject != "clock.get_time" {
		t.Errorf("expected subject 'clock.get_time', got '%s'", resp.Subject)
	}
	if !resp.OK {
		t.Error("expected OK to be true")
	}
	if resp.Arg("time") != "2026-03-13T14:32:00Z" {
		t.Errorf("expected time arg, got '%s'", resp.Arg("time"))
	}
}

func TestParseResponse_Error(t *testing.T) {
	resp, err := parseResponse(`~h1:clock.get_time error code=clock.err.invalid_timezone what="Unknown timezone: Fakezone" node_error capture=com.example.clock`)
	if err != nil {
		t.Fatal(err)
	}
	if resp.OK {
		t.Error("expected OK to be false on error response")
	}
	if resp.CustomError {
		t.Error("expected CustomError to be false on standard error response")
	}
	if resp.ErrCode != "clock.err.invalid_timezone" {
		t.Errorf("expected ErrCode, got '%s'", resp.ErrCode)
	}
	if resp.ErrWhat != "Unknown timezone: Fakezone" {
		t.Errorf("expected ErrWhat, got '%s'", resp.ErrWhat)
	}
	if resp.ErrorOrigin != ErrorOriginNode {
		t.Errorf("expected ErrorOrigin 'node_error', got '%s'", resp.ErrorOrigin)
	}
}

func TestParseResponse_Cancelled(t *testing.T) {
	resp, err := parseResponse("~h1:dialog.file_picker cancelled capture=com.example.dialog")
	if err != nil {
		t.Fatal(err)
	}
	if resp.OK {
		t.Error("expected OK to be false on cancelled response")
	}
	if !resp.Cancelled {
		t.Error("expected Cancelled to be true")
	}
	if resp.CustomError {
		t.Error("expected CustomError to be false on cancelled response")
	}
	if resp.Capture != "com.example.dialog" {
		t.Errorf("expected Capture 'com.example.dialog', got '%s'", resp.Capture)
	}
}

func TestParseResponse_CustomError(t *testing.T) {
	resp, err := parseResponse(`~h1:clock.get_time custom_error code=clock.err.invalid_timezone what="Unknown timezone: Fakezone" node_error capture=com.example.clock`)
	if err != nil {
		t.Fatal(err)
	}
	if resp.OK {
		t.Error("expected OK to be false on custom_error response")
	}
	if !resp.CustomError {
		t.Error("expected CustomError to be true on custom_error response")
	}
	if resp.ErrCode != "clock.err.invalid_timezone" {
		t.Errorf("expected ErrCode 'clock.err.invalid_timezone', got '%s'", resp.ErrCode)
	}
	if resp.ErrorOrigin != ErrorOriginNode {
		t.Errorf("expected ErrorOrigin 'node_error', got '%s'", resp.ErrorOrigin)
	}
}

func TestParseResponse_CastErrorOrigin(t *testing.T) {
	resp, err := parseResponse(`~h1:clock.get_time error code=RequiredArgumentMissing what="timezone is required" cast_error capture=com.example.clock`)
	if err != nil {
		t.Fatal(err)
	}
	if resp.OK || resp.CustomError {
		t.Error("expected standard error response")
	}
	if resp.ErrorOrigin != ErrorOriginCast {
		t.Errorf("expected ErrorOrigin 'cast_error', got '%s'", resp.ErrorOrigin)
	}
}

func TestParseResponse_NotAResponse(t *testing.T) {
	_, err := parseResponse("clock.get_time timezone=UTC")
	if err == nil {
		t.Error("expected error for non-response message")
	}
}

func TestParseResponse_QuotedValue(t *testing.T) {
	resp, err := parseResponse(`~l1:logger.append message="hello world" ok capture=com.example.logger`)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Arg("message") != "hello world" {
		t.Errorf("expected 'hello world', got '%s'", resp.Arg("message"))
	}
}

func TestParseResponse_ArgIf_Found(t *testing.T) {
	resp, err := parseResponse("~h1:clock.get_time time=now ok capture=com.example.clock")
	if err != nil {
		t.Fatal(err)
	}
	if resp.ArgIf("time", "unknown") != "now" {
		t.Error("ArgIf should return existing value")
	}
}

func TestParseResponse_ArgIf_Missing(t *testing.T) {
	resp, err := parseResponse("~h1:clock.get_time ok capture=com.example.clock")
	if err != nil {
		t.Fatal(err)
	}
	if resp.ArgIf("time", "fallback") != "fallback" {
		t.Error("ArgIf should return default for missing key")
	}
}

func TestParseResponse_NoHandle(t *testing.T) {
	// A response without a colon in the head — handle only, no subject
	resp, err := parseResponse("~h1 ok")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Handle != "h1" {
		t.Errorf("expected handle 'h1', got '%s'", resp.Handle)
	}
	if resp.Subject != "" {
		t.Errorf("expected empty subject, got '%s'", resp.Subject)
	}
}

func TestParseResponse_CapturePromoted(t *testing.T) {
	resp, err := parseResponse("~h1:clock.get_time time=now ok capture=com.example.clock")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Capture != "com.example.clock" {
		t.Errorf("expected Capture 'com.example.clock', got '%s'", resp.Capture)
	}
	// capture must not bleed into Args
	if _, ok := resp.Args["capture"]; ok {
		t.Error("capture should not appear in Args")
	}
}

func TestParseResponse_ErrorFieldsNotInArgs(t *testing.T) {
	resp, err := parseResponse(`~h1:clock.get_time error code=clock.err.invalid_timezone what="Unknown timezone: Fakezone" capture=com.example.clock`)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := resp.Args["code"]; ok {
		t.Error("code should not appear in Args")
	}
	if _, ok := resp.Args["what"]; ok {
		t.Error("what should not appear in Args")
	}
	if _, ok := resp.Args["capture"]; ok {
		t.Error("capture should not appear in Args")
	}
}

func TestParseResponse_FromSpore_HubError(t *testing.T) {
	resp, err := parseResponse(`~h1:clock.get_time error code=SPORE.node_unavailable what="No node connected" capture=SPORE.hub`)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.FromSpore() {
		t.Error("expected FromSpore() to be true for capture=SPORE.hub")
	}
	if resp.ErrCode != "SPORE.node_unavailable" {
		t.Errorf("expected ErrCode 'SPORE.node_unavailable', got '%s'", resp.ErrCode)
	}
}

func TestParseResponse_FromSpore_NodeError(t *testing.T) {
	resp, err := parseResponse(`~h1:clock.get_time error code=clock.err.invalid_timezone what="Unknown timezone" capture=com.example.clock`)
	if err != nil {
		t.Fatal(err)
	}
	if resp.FromSpore() {
		t.Error("expected FromSpore() to be false for node-produced error")
	}
}

// --- Wait / WaitFor tests ---

// makeTestClient builds a client whose reader is primed with the given lines,
// sufficient to test Wait and WaitFor without a real socket.
func makeTestClient(lines ...string) *Client {
	buf := ""
	for _, l := range lines {
		buf += l + "\n"
	}
	r := bufio.NewReader(strings.NewReader(buf))
	return &Client{
		nodeID:   "dev.test.app",
		reader:   r,
		conn:     &mockConn{},
		handlers: make(map[string]HandlerFunc),
		waiters:  make(map[string]chan *Response),
	}
}

// listenerClient builds a client suitable for testing listener-mode behaviour.
// It does not have a real reader (not needed; deliverResponse is called directly in tests).
func listenerClient() *Client {
	c := &Client{
		handlers:         make(map[string]HandlerFunc),
		responseHandlers: make(map[string]ResponseHandlerFunc),
		waiters:          make(map[string]chan *Response),
		conn:             &mockConn{},
	}
	c.listening.Store(true)
	return c
}

func TestWait_ReturnsFirstResponse(t *testing.T) {
	client := makeTestClient("~h1:clock.get_time time=now ok capture=com.example.clock")
	resp, err := client.Wait(5000)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Handle != "h1" {
		t.Errorf("expected handle 'h1', got '%s'", resp.Handle)
	}
	if resp.Arg("time") != "now" {
		t.Errorf("expected time 'now', got '%s'", resp.Arg("time"))
	}
}

func TestWait_SkipsInboundCalls(t *testing.T) {
	// A call message comes first, then a response. Wait should skip the call.
	client := makeTestClient(
		"clock.get_time timezone=UTC ~h2 cast=dev.sporeos.cli",
		"~h1:clock.get_time time=now ok capture=com.example.clock",
	)
	resp, err := client.Wait(5000)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Handle != "h1" {
		t.Errorf("expected handle 'h1', got '%s'", resp.Handle)
	}
}

func TestWaitFor_MatchesHandle(t *testing.T) {
	// Two responses in the buffer; WaitFor should return only the matching one.
	client := makeTestClient(
		"~other:some.subject result=x ok capture=com.example.a",
		"~req1:clock.get_time time=now ok capture=com.example.clock",
	)
	resp, err := client.WaitFor("req1", 5000)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Handle != "req1" {
		t.Errorf("expected handle 'req1', got '%s'", resp.Handle)
	}
	if resp.Arg("time") != "now" {
		t.Errorf("expected time 'now', got '%s'", resp.Arg("time"))
	}
}

func TestWaitFor_SkipsNonMatchingResponses(t *testing.T) {
	client := makeTestClient(
		"~wrong1:foo.bar x=1 ok capture=com.example.a",
		"~wrong2:foo.baz x=2 ok capture=com.example.b",
		"~target:clock.get_time time=now ok capture=com.example.clock",
	)
	resp, err := client.WaitFor("target", 5000)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Handle != "target" {
		t.Errorf("expected handle 'target', got '%s'", resp.Handle)
	}
}

// --- HandleResponse / listener-mode tests ---

// waitForWaiter polls until the client has registered a waiter under the given key,
// or times out. Used in listener-mode tests to avoid a race between goroutine startup
// and deliverResponse being called.
func waitForWaiter(t *testing.T, c *Client, key string) {
	t.Helper()
	for i := 0; i < 200; i++ {
		c.waitersMu.Lock()
		_, ok := c.waiters[key]
		c.waitersMu.Unlock()
		if ok {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("timed out waiting for waiter to be registered")
}

func TestHandleResponse_ExactSubjectMatch(t *testing.T) {
	client := listenerClient()

	received := make(chan *Response, 1)
	client.HandleResponse("clock.get_time", func(resp *Response) {
		received <- resp
	})

	client.deliverResponse("~h1:clock.get_time time=now ok capture=com.example.clock")

	select {
	case resp := <-received:
		if resp.Handle != "h1" {
			t.Errorf("expected handle 'h1', got '%s'", resp.Handle)
		}
		if resp.Arg("time") != "now" {
			t.Errorf("expected time 'now', got '%s'", resp.Arg("time"))
		}
	case <-time.After(time.Second):
		t.Error("response handler was not called for exact subject match")
	}
}

func TestHandleResponse_ShortNameMatch(t *testing.T) {
	// HandleResponse("get_time") should match responses for "clock.get_time".
	client := listenerClient()

	received := make(chan *Response, 1)
	client.HandleResponse("get_time", func(resp *Response) {
		received <- resp
	})

	client.deliverResponse("~h1:clock.get_time time=now ok capture=com.example.clock")

	select {
	case resp := <-received:
		if resp.Handle != "h1" {
			t.Errorf("expected handle 'h1', got '%s'", resp.Handle)
		}
	case <-time.After(time.Second):
		t.Error("response handler was not called for short-name subject match")
	}
}

func TestHandleResponse_NotCalledForDifferentSubject(t *testing.T) {
	client := listenerClient()

	handlerCalled := false
	client.HandleResponse("filesystem.read", func(resp *Response) {
		handlerCalled = true
	})

	client.deliverResponse("~h1:clock.get_time time=now ok capture=com.example.clock")

	if handlerCalled {
		t.Error("handler should not be called for a different subject")
	}
}

func TestHandleResponse_CalledWhenNoWaiter(t *testing.T) {
	// Alias for exact-match test — kept for clarity that no waiter is needed.
	client := listenerClient()

	received := make(chan *Response, 1)
	client.HandleResponse("clock.get_time", func(resp *Response) {
		received <- resp
	})

	client.deliverResponse("~h1:clock.get_time time=now ok capture=com.example.clock")

	select {
	case resp := <-received:
		if resp.Handle != "h1" {
			t.Errorf("expected handle 'h1', got '%s'", resp.Handle)
		}
	case <-time.After(time.Second):
		t.Error("response handler was not called")
	}
}

func TestHandleResponse_NotCalledWhenHandlerWaiterActive(t *testing.T) {
	client := listenerClient()

	handlerCalled := false
	client.HandleResponse("clock.get_time", func(resp *Response) {
		handlerCalled = true
	})

	// Manually register a specific waiter — simulates WaitFor("h1") being active.
	ch := make(chan *Response, 1)
	client.waitersMu.Lock()
	client.waiters["h1"] = ch
	client.waitersMu.Unlock()

	client.deliverResponse("~h1:clock.get_time time=now ok capture=com.example.clock")

	resp := <-ch
	if resp.Handle != "h1" {
		t.Errorf("expected handle 'h1', got '%s'", resp.Handle)
	}
	if handlerCalled {
		t.Error("HandleResponse handler should not be called when a waiter is active")
	}
}

func TestWaitFor_ListenerMode_ReceivesViaChannel(t *testing.T) {
	client := listenerClient()

	result := make(chan *Response, 1)
	go func() {
		resp, err := client.WaitFor("req1", 5000)
		if err != nil {
			result <- nil
			return
		}
		result <- resp
	}()

	waitForWaiter(t, client, "req1")
	client.deliverResponse("~req1:clock.get_time time=now ok capture=com.example.clock")

	select {
	case resp := <-result:
		if resp == nil {
			t.Fatal("WaitFor returned nil in listener mode")
		}
		if resp.Handle != "req1" {
			t.Errorf("expected handle 'req1', got '%s'", resp.Handle)
		}
		if resp.Arg("time") != "now" {
			t.Errorf("expected time 'now', got '%s'", resp.Arg("time"))
		}
	case <-time.After(time.Second):
		t.Error("WaitFor did not receive response in listener mode")
	}
}

func TestWait_ListenerMode_WildcardWaiter(t *testing.T) {
	client := listenerClient()

	result := make(chan *Response, 1)
	go func() {
		resp, err := client.Wait(5000)
		if err != nil {
			result <- nil
			return
		}
		result <- resp
	}()

	waitForWaiter(t, client, "")
	client.deliverResponse("~any1:some.subject result=hello ok capture=com.example.a")

	select {
	case resp := <-result:
		if resp == nil {
			t.Fatal("Wait returned nil in listener mode")
		}
		if resp.Handle != "any1" {
			t.Errorf("expected handle 'any1', got '%s'", resp.Handle)
		}
	case <-time.After(time.Second):
		t.Error("Wait did not receive response in listener mode")
	}
}

func TestDeliverResponse_SpecificWaiterBeatsWildcard(t *testing.T) {
	// If both a WaitFor("h1") and a Wait() are active, and ~h1 arrives,
	// the specific waiter wins; the wildcard is not consumed.
	client := listenerClient()

	specificCh := make(chan *Response, 1)
	wildcardCh := make(chan *Response, 1)
	client.waitersMu.Lock()
	client.waiters["h1"] = specificCh
	client.waiters[""] = wildcardCh
	client.waitersMu.Unlock()

	client.deliverResponse("~h1:clock.get_time time=now ok capture=com.example.clock")

	select {
	case resp := <-specificCh:
		if resp.Handle != "h1" {
			t.Errorf("expected h1, got %s", resp.Handle)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("specific waiter did not receive the response")
	}

	select {
	case <-wildcardCh:
		t.Error("wildcard waiter should not have received the response consumed by the specific waiter")
	default:
		// Correct — wildcard was not triggered.
	}
}

func TestDeliverResponse_WildcardUsedWhenNoSpecific(t *testing.T) {
	// When only a wildcard waiter is registered and a response for any handle arrives,
	// the wildcard consumes it and the handler is not called.
	client := listenerClient()

	handlerCalled := false
	client.HandleResponse("foo.bar", func(resp *Response) {
		handlerCalled = true
	})

	wildcardCh := make(chan *Response, 1)
	client.waitersMu.Lock()
	client.waiters[""] = wildcardCh
	client.waitersMu.Unlock()

	client.deliverResponse("~other:foo.bar x=1 ok capture=com.example.a")

	select {
	case resp := <-wildcardCh:
		if resp.Handle != "other" {
			t.Errorf("expected handle 'other', got '%s'", resp.Handle)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("wildcard waiter did not receive response")
	}

	if handlerCalled {
		t.Error("HandleResponse should not be called when wildcard waiter consumes the response")
	}
}

func TestWait_ListenerMode_Timeout(t *testing.T) {
	client := listenerClient()

	result := make(chan error, 1)
	go func() {
		_, err := client.Wait(50) // 50 ms timeout — no response delivered
		result <- err
	}()

	select {
	case err := <-result:
		if err == nil {
			t.Fatal("expected timeout error, got nil")
		}
		if err.Error() != "wait timed out" {
			t.Errorf("expected 'wait timed out', got '%s'", err.Error())
		}
	case <-time.After(time.Second):
		t.Error("Wait did not time out within expected window")
	}
}

func TestWaitFor_ListenerMode_Timeout(t *testing.T) {
	client := listenerClient()

	result := make(chan error, 1)
	go func() {
		_, err := client.WaitFor("req1", 50) // 50 ms timeout — no response delivered
		result <- err
	}()

	select {
	case err := <-result:
		if err == nil {
			t.Fatal("expected timeout error, got nil")
		}
		if err.Error() != "wait timed out" {
			t.Errorf("expected 'wait timed out', got '%s'", err.Error())
		}
	case <-time.After(time.Second):
		t.Error("WaitFor did not time out within expected window")
	}
}

// --- SendAndWait tests ---

func TestSendAndWait_StandaloneMode_WithHandle(t *testing.T) {
	// Primed reader returns the matching response; mockConn captures the write.
	client := makeTestClient("~req1:clock.get_time time=now ok capture=com.example.clock")
	resp, err := client.SendAndWait("clock.get_time timezone=UTC ~req1", 5000)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Handle != "req1" {
		t.Errorf("expected handle 'req1', got '%s'", resp.Handle)
	}
	if resp.Arg("time") != "now" {
		t.Errorf("expected time 'now', got '%s'", resp.Arg("time"))
	}
	// Verify the message was actually written.
	written := string(client.conn.(*mockConn).written)
	if !stringContains(written, "clock.get_time") {
		t.Errorf("expected message to be written, got '%s'", written)
	}
}

func TestSendAndWait_NoHandle_ReturnsError(t *testing.T) {
	client := makeTestClient()
	_, err := client.SendAndWait("clock.get_time timezone=UTC", 5000)
	if err == nil {
		t.Fatal("expected error when no handle is present")
	}
}

func TestSendAndWait_StandaloneMode_SkipsNonMatchingHandle(t *testing.T) {
	// Two responses; only the second matches the handle.
	client := makeTestClient(
		"~other:foo.bar x=1 ok capture=com.example.a",
		"~req1:clock.get_time time=now ok capture=com.example.clock",
	)
	resp, err := client.SendAndWait("clock.get_time ~req1", 5000)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Handle != "req1" {
		t.Errorf("expected handle 'req1', got '%s'", resp.Handle)
	}
}

func TestSendAndWait_ListenerMode_WithHandle(t *testing.T) {
	client := listenerClient()

	result := make(chan *Response, 1)
	go func() {
		resp, err := client.SendAndWait("clock.get_time timezone=UTC ~req1", 5000)
		if err != nil {
			result <- nil
			return
		}
		result <- resp
	}()

	waitForWaiter(t, client, "req1")
	client.deliverResponse("~req1:clock.get_time time=now ok capture=com.example.clock")

	select {
	case resp := <-result:
		if resp == nil {
			t.Fatal("SendAndWait returned nil in listener mode")
		}
		if resp.Handle != "req1" {
			t.Errorf("expected handle 'req1', got '%s'", resp.Handle)
		}
		if resp.Arg("time") != "now" {
			t.Errorf("expected time 'now', got '%s'", resp.Arg("time"))
		}
	case <-time.After(time.Second):
		t.Error("SendAndWait did not receive response in listener mode")
	}
}

// --- parseWitness tests ---

func TestParseWitness_IncomingCall(t *testing.T) {
	msg, err := parseWitness("witness clock.get_time cast=com.example.shell ~h1 spore_incoming spore_time=1744732800000")
	if err != nil {
		t.Fatal(err)
	}
	if msg.Kind != WitnessKindIncoming {
		t.Errorf("expected WitnessKindIncoming, got '%s'", msg.Kind)
	}
	if msg.IsResponse {
		t.Error("expected IsResponse false for call-type body")
	}
	if msg.Subject != "clock.get_time" {
		t.Errorf("expected Subject 'clock.get_time', got '%s'", msg.Subject)
	}
	if msg.Handle != "h1" {
		t.Errorf("expected Handle 'h1', got '%s'", msg.Handle)
	}
	if msg.Cast != "com.example.shell" {
		t.Errorf("expected Cast 'com.example.shell', got '%s'", msg.Cast)
	}
	if msg.SporeTime != 1744732800000 {
		t.Errorf("expected SporeTime 1744732800000, got %d", msg.SporeTime)
	}
}

func TestParseWitness_OutgoingResponse(t *testing.T) {
	msg, err := parseWitness(`witness ~h1:clock.get_time time="2026-03-13T14:32:00Z" ok capture=com.example.clock spore_outgoing spore_time=1744732800001`)
	if err != nil {
		t.Fatal(err)
	}
	if msg.Kind != WitnessKindOutgoing {
		t.Errorf("expected WitnessKindOutgoing, got '%s'", msg.Kind)
	}
	if !msg.IsResponse {
		t.Error("expected IsResponse true for response-type body")
	}
	if msg.Subject != "clock.get_time" {
		t.Errorf("expected Subject 'clock.get_time', got '%s'", msg.Subject)
	}
	if msg.Handle != "h1" {
		t.Errorf("expected Handle 'h1', got '%s'", msg.Handle)
	}
	if !msg.OK {
		t.Error("expected OK true")
	}
	if msg.Capture != "com.example.clock" {
		t.Errorf("expected Capture 'com.example.clock', got '%s'", msg.Capture)
	}
	if msg.Args["time"] != "2026-03-13T14:32:00Z" {
		t.Errorf("expected time arg '2026-03-13T14:32:00Z', got '%s'", msg.Args["time"])
	}
	if msg.SporeTime != 1744732800001 {
		t.Errorf("expected SporeTime 1744732800001, got %d", msg.SporeTime)
	}
}

func TestParseWitness_HubEvent(t *testing.T) {
	msg, err := parseWitness(`witness SPORE.hub.event level=warn what="Unable to complete handshake" spore_event spore_time=1744732800000`)
	if err != nil {
		t.Fatal(err)
	}
	if msg.Kind != WitnessKindEvent {
		t.Errorf("expected WitnessKindEvent, got '%s'", msg.Kind)
	}
	if msg.IsResponse {
		t.Error("expected IsResponse false for hub event")
	}
	if msg.Subject != "SPORE.hub.event" {
		t.Errorf("expected Subject 'SPORE.hub.event', got '%s'", msg.Subject)
	}
	if msg.Args["level"] != "warn" {
		t.Errorf("expected level 'warn', got '%s'", msg.Args["level"])
	}
	if msg.ErrWhat != "Unable to complete handshake" {
		t.Errorf("expected ErrWhat 'Unable to complete handshake', got '%s'", msg.ErrWhat)
	}
}

func TestParseWitness_NodeEmitted(t *testing.T) {
	msg, err := parseWitness(`witness SPORE.witness.emit message="loaded config" cast=com.example.mynode spore_node spore_time=1744732800000`)
	if err != nil {
		t.Fatal(err)
	}
	if msg.Kind != WitnessKindNode {
		t.Errorf("expected WitnessKindNode, got '%s'", msg.Kind)
	}
	if msg.Cast != "com.example.mynode" {
		t.Errorf("expected Cast 'com.example.mynode', got '%s'", msg.Cast)
	}
	if msg.Args["message"] != "loaded config" {
		t.Errorf("expected message 'loaded config', got '%s'", msg.Args["message"])
	}
}

func TestParseWitness_ErrorResponse(t *testing.T) {
	msg, err := parseWitness(`witness ~h1:clock.get_time error code=RouteNotFound what="No node connected" spore_error capture=SPORE.hub spore_outgoing spore_time=1744732800000`)
	if err != nil {
		t.Fatal(err)
	}
	if msg.Kind != WitnessKindOutgoing {
		t.Errorf("expected WitnessKindOutgoing, got '%s'", msg.Kind)
	}
	if msg.OK {
		t.Error("expected OK false on error response")
	}
	if msg.CustomError {
		t.Error("expected CustomError false on standard error")
	}
	if msg.ErrCode != "RouteNotFound" {
		t.Errorf("expected ErrCode 'RouteNotFound', got '%s'", msg.ErrCode)
	}
	if msg.ErrorOrigin != ErrorOriginSpore {
		t.Errorf("expected ErrorOriginSpore, got '%s'", msg.ErrorOrigin)
	}
	if msg.Capture != "SPORE.hub" {
		t.Errorf("expected Capture 'SPORE.hub', got '%s'", msg.Capture)
	}
}

func TestParseWitness_ExpandedInlineCall(t *testing.T) {
	msg, err := parseWitness("witness ~~a3f9:dialog.file_picker path=/tmp/foo.yaml ok capture=com.example.dialog spore_expanded spore_time=1744732800000")
	if err != nil {
		t.Fatal(err)
	}
	if msg.Kind != WitnessKindExpanded {
		t.Errorf("expected WitnessKindExpanded, got '%s'", msg.Kind)
	}
	if msg.Handle != "~a3f9" {
		t.Errorf("expected Handle '~a3f9', got '%s'", msg.Handle)
	}
}

func TestParseWitness_NotWitnessMessage(t *testing.T) {
	_, err := parseWitness("clock.get_time ~h1")
	if err == nil {
		t.Error("expected error for non-witness message")
	}
}

func TestParseWitness_ProtocolFieldsNotInArgs(t *testing.T) {
	msg, err := parseWitness(`witness ~h1:clock.get_time time=now ok capture=com.example.clock spore_outgoing spore_time=1744732800000`)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := msg.Args["capture"]; ok {
		t.Error("capture should not appear in Args")
	}
	if _, ok := msg.Args["spore_time"]; ok {
		t.Error("spore_time should not appear in Args")
	}
}

// --- HandleWitness routing tests ---

func TestHandleWitness_CalledForWitnessMessage(t *testing.T) {
	client := listenerClient()

	received := make(chan *WitnessMessage, 1)
	client.HandleWitness(func(msg *WitnessMessage) {
		received <- msg
	})

	client.deliverWitness("witness clock.get_time cast=com.example.shell ~h1 spore_incoming spore_time=1744732800000")

	select {
	case msg := <-received:
		if msg.Kind != WitnessKindIncoming {
			t.Errorf("expected WitnessKindIncoming, got '%s'", msg.Kind)
		}
		if msg.Subject != "clock.get_time" {
			t.Errorf("expected subject 'clock.get_time', got '%s'", msg.Subject)
		}
	case <-time.After(time.Second):
		t.Error("witness handler was not called")
	}
}

func TestHandleWitness_NotCalledForNormalCall(t *testing.T) {
	client := listenerClient()

	witnessCalled := false
	client.HandleWitness(func(msg *WitnessMessage) {
		witnessCalled = true
	})

	// A normal call should not reach the witness handler.
	call := &Call{Subject: "clock.get_time", args: map[string]string{}, flags: map[string]bool{}}
	client.dispatch(call)

	if witnessCalled {
		t.Error("witness handler should not be called for normal dispatch")
	}
}

func TestHandleWitness_NotCalledForNormalResponse(t *testing.T) {
	client := listenerClient()

	witnessCalled := false
	client.HandleWitness(func(msg *WitnessMessage) {
		witnessCalled = true
	})

	received := make(chan *Response, 1)
	client.HandleResponse("clock.get_time", func(resp *Response) {
		received <- resp
	})

	client.deliverResponse("~h1:clock.get_time time=now ok capture=com.example.clock")

	select {
	case <-received:
		// Good — response handler was called.
	case <-time.After(time.Second):
		t.Error("response handler was not called")
	}

	if witnessCalled {
		t.Error("witness handler should not be called for normal responses")
	}
}

func TestHandleWitness_NilHandlerIsNoOp(t *testing.T) {
	client := listenerClient()
	// No handler registered; deliverWitness must not panic.
	client.deliverWitness("witness clock.get_time cast=com.example.shell spore_incoming spore_time=1744732800000")
}

func TestHandleWitness_ReplaceHandler(t *testing.T) {
	client := listenerClient()

	first := make(chan *WitnessMessage, 1)
	second := make(chan *WitnessMessage, 1)

	client.HandleWitness(func(msg *WitnessMessage) { first <- msg })
	client.HandleWitness(func(msg *WitnessMessage) { second <- msg }) // replaces first

	client.deliverWitness("witness clock.get_time cast=com.example.shell spore_incoming spore_time=1744732800000")

	select {
	case <-second:
		// Correct — second handler received it.
	case <-time.After(time.Second):
		t.Error("replaced witness handler was not called")
	}

	select {
	case <-first:
		t.Error("first (replaced) witness handler should not have been called")
	default:
		// Correct.
	}
}

func TestSendAndWait_ListenerMode_Timeout(t *testing.T) {
	client := listenerClient()

	result := make(chan error, 1)
	go func() {
		_, err := client.SendAndWait("clock.get_time ~req1", 50)
		result <- err
	}()

	select {
	case err := <-result:
		if err == nil {
			t.Fatal("expected timeout error, got nil")
		}
		if err.Error() != "send timed out waiting for response" {
			t.Errorf("unexpected error: %s", err.Error())
		}
	case <-time.After(time.Second):
		t.Error("SendAndWait did not time out within expected window")
	}
}

func TestExtractHandle_Present(t *testing.T) {
	h := extractHandle("clock.get_time timezone=UTC ~req1")
	if h != "req1" {
		t.Errorf("expected 'req1', got '%s'", h)
	}
}

func TestExtractHandle_Absent(t *testing.T) {
	h := extractHandle("clock.get_time timezone=UTC")
	if h != "" {
		t.Errorf("expected empty string, got '%s'", h)
	}
}

func TestExtractHandle_TildeInSubjectNotTreatedAsHandle(t *testing.T) {
	// The subject is the first token; only subsequent tokens are checked.
	h := extractHandle("clock.get_time ~req1 key=val")
	if h != "req1" {
		t.Errorf("expected 'req1', got '%s'", h)
	}
}

// =============================================================================
// splitFields — array and object delimiter handling
// Regression tests for: spaces inside [...] and {...} must not split tokens.
// =============================================================================

func TestSplitFields_ArrayValueWithSpaces(t *testing.T) {
	// e.g. expression=["a", "b", "c"] must stay as a single token
	fields := splitFields(`expression=["a", "b", "c"]`)
	if len(fields) != 1 {
		t.Fatalf("expected 1 field, got %d: %v", len(fields), fields)
	}
	if fields[0] != `expression=["a", "b", "c"]` {
		t.Errorf("unexpected field value: %q", fields[0])
	}
}

func TestSplitFields_BareArrayWithSpaces(t *testing.T) {
	fields := splitFields(`["a", "b", "c"]`)
	if len(fields) != 1 {
		t.Fatalf("expected 1 field, got %d: %v", len(fields), fields)
	}
	if fields[0] != `["a", "b", "c"]` {
		t.Errorf("unexpected field value: %q", fields[0])
	}
}

func TestSplitFields_ObjectValueWithSpaces(t *testing.T) {
	fields := splitFields("data={key: value with spaces}")
	if len(fields) != 1 {
		t.Fatalf("expected 1 field, got %d: %v", len(fields), fields)
	}
	if fields[0] != "data={key: value with spaces}" {
		t.Errorf("unexpected field value: %q", fields[0])
	}
}

func TestSplitFields_ArrayInFullMessage(t *testing.T) {
	// Simulates what the hub sends when routing a cast that has an array arg.
	fields := splitFields(`echo ~h1 expression=["a", "b", "c"] cast=dev.sporeos.cli`)
	want := []string{"echo", "~h1", `expression=["a", "b", "c"]`, "cast=dev.sporeos.cli"}
	if len(fields) != len(want) {
		t.Fatalf("expected %d fields, got %d: %v", len(want), len(fields), fields)
	}
	for i, w := range want {
		if fields[i] != w {
			t.Errorf("[%d] expected %q, got %q", i, w, fields[i])
		}
	}
}

func TestSplitFields_ObjectInFullMessage(t *testing.T) {
	fields := splitFields("cmd ~h2 data={key: value with spaces} cast=dev.sporeos.cli")
	want := []string{"cmd", "~h2", "data={key: value with spaces}", "cast=dev.sporeos.cli"}
	if len(fields) != len(want) {
		t.Fatalf("expected %d fields, got %d: %v", len(want), len(fields), fields)
	}
	for i, w := range want {
		if fields[i] != w {
			t.Errorf("[%d] expected %q, got %q", i, w, fields[i])
		}
	}
}

// =============================================================================
// parseCall — array argument regression
// =============================================================================

func TestParseCall_ArrayArgument(t *testing.T) {
	// Regression: expression=["a", "b", "c"] must be preserved as a single arg.
	call, err := parseCall(`echo ~h1 expression=["a", "b", "c"] cast=dev.sporeos.cli`)
	if err != nil {
		t.Fatal(err)
	}
	if call.Subject != "echo" {
		t.Errorf("subject: got %q", call.Subject)
	}
	if call.Handle != "h1" {
		t.Errorf("handle: got %q", call.Handle)
	}
	want := `["a", "b", "c"]`
	if call.ArgIf("expression", "") != want {
		t.Errorf("expression: got %q, want %q", call.ArgIf("expression", ""), want)
	}
}

func TestParseCall_ObjectArgument(t *testing.T) {
	call, err := parseCall("cmd ~h1 data={key: value with spaces} cast=dev.sporeos.cli")
	if err != nil {
		t.Fatal(err)
	}
	want := "{key: value with spaces}"
	if call.ArgIf("data", "") != want {
		t.Errorf("data: got %q, want %q", call.ArgIf("data", ""), want)
	}
}

// =============================================================================
// formatArgValue — quoting rules
// =============================================================================

func TestFormatArgValue_PlainNoSpace(t *testing.T) {
	if got := formatArgValue("key", "value"); got != "key=value" {
		t.Errorf("got %q", got)
	}
}

func TestFormatArgValue_PlainWithSpace(t *testing.T) {
	if got := formatArgValue("msg", "hello world"); got != `msg="hello world"` {
		t.Errorf("got %q", got)
	}
}

func TestFormatArgValue_ArrayNotQuoted(t *testing.T) {
	// Arrays must not be wrapped in double-quotes; they carry their own delimiters.
	v := `["a", "b", "c"]`
	if got := formatArgValue("items", v); got != "items="+v {
		t.Errorf("got %q", got)
	}
}

func TestFormatArgValue_ObjectNotQuoted(t *testing.T) {
	v := "{key: value with spaces}"
	if got := formatArgValue("data", v); got != "data="+v {
		t.Errorf("got %q", got)
	}
}

// ---------------------------------------------------------------------------
// PublishMessage / parsePublish tests
// ---------------------------------------------------------------------------

func TestParsePublish_Simple(t *testing.T) {
	msg, err := parsePublish("publish my.topic")
	if err != nil {
		t.Fatal(err)
	}
	if msg.Topic != "my.topic" {
		t.Errorf("expected topic 'my.topic', got '%s'", msg.Topic)
	}
}

func TestParsePublish_WithArgs(t *testing.T) {
	msg, err := parsePublish("publish sensors.temperature value=42.5 unit=celsius cast=dev.sporeos.sensor")
	if err != nil {
		t.Fatal(err)
	}
	if msg.Topic != "sensors.temperature" {
		t.Errorf("expected topic 'sensors.temperature', got '%s'", msg.Topic)
	}
	if msg.ArgIf("value", "") != "42.5" {
		t.Errorf("expected value '42.5', got '%s'", msg.ArgIf("value", ""))
	}
	if msg.ArgIf("unit", "") != "celsius" {
		t.Errorf("expected unit 'celsius', got '%s'", msg.ArgIf("unit", ""))
	}
	if msg.Cast != "dev.sporeos.sensor" {
		t.Errorf("expected cast 'dev.sporeos.sensor', got '%s'", msg.Cast)
	}
	if msg.HasArg("cast") {
		t.Error("cast should not appear in Args")
	}
}

func TestParsePublish_WithFlags(t *testing.T) {
	msg, err := parsePublish("publish alerts.door opened cast=dev.sporeos.security")
	if err != nil {
		t.Fatal(err)
	}
	if !msg.HasFlag("opened") {
		t.Error("expected 'opened' flag")
	}
	if msg.Cast != "dev.sporeos.security" {
		t.Errorf("expected cast 'dev.sporeos.security', got '%s'", msg.Cast)
	}
}

func TestParsePublish_QuotedArg(t *testing.T) {
	msg, err := parsePublish(`publish chat.messages text="hello world" cast=dev.sporeos.chat`)
	if err != nil {
		t.Fatal(err)
	}
	if msg.ArgIf("text", "") != "hello world" {
		t.Errorf("expected 'hello world', got '%s'", msg.ArgIf("text", ""))
	}
}

func TestParsePublish_MissingTopic(t *testing.T) {
	_, err := parsePublish("publish ")
	if err == nil {
		t.Error("expected error for missing topic")
	}
}

func TestParsePublish_NotPublish(t *testing.T) {
	_, err := parsePublish("clock.get_time")
	if err == nil {
		t.Error("expected error for non-publish message")
	}
}

func TestPublishMessage_Arg_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for missing arg")
		}
	}()
	msg := &PublishMessage{Args: map[string]string{}, Flags: []string{}}
	_ = msg.Arg("missing")
}

func TestPublishMessage_HasFlag(t *testing.T) {
	msg := &PublishMessage{Args: map[string]string{}, Flags: []string{"urgent", "loud"}}
	if !msg.HasFlag("urgent") {
		t.Error("expected 'urgent' flag")
	}
	if msg.HasFlag("quiet") {
		t.Error("unexpected 'quiet' flag")
	}
}

// ---------------------------------------------------------------------------
// Subscribe / Unsubscribe tests
// ---------------------------------------------------------------------------

// makeSubscribeClient creates a client primed for a single subscribe-style
// interaction. The reader is seeded with one response line so that SendAndWait
// (in standalone mode) can receive the hub ACK without a real socket.
func makeSubscribeClient(ackLine string) *Client {
	c := NewClientWithSocket("dev.test.node", "/unused")
	mock := &mockConn{}
	c.conn = mock
	c.reader = bufio.NewReader(strings.NewReader(ackLine + "\n"))
	return c
}

func TestSubscribe_FirstTime_SendsWireMessage(t *testing.T) {
	// The handle counter starts at 0, so the first auto-handle is "sub1".
	c := makeSubscribeClient("~sub1:SPORE.topic.subscribe ok capture=SPORE.hub")

	received := make(chan *PublishMessage, 1)
	if err := c.Subscribe("sensors.temperature", func(msg *PublishMessage) {
		received <- msg
	}, 500); err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	// Wire message must have been written to the connection.
	written := string(c.conn.(*mockConn).written)
	if !strings.Contains(written, "SPORE.topic.subscribe") {
		t.Errorf("expected SPORE.topic.subscribe in wire message, got: %s", written)
	}
	if !strings.Contains(written, "topic=sensors.temperature") {
		t.Errorf("expected topic=sensors.temperature in wire message, got: %s", written)
	}

	// Callback must be registered — fire a publish directly and check it fires.
	c.deliverPublish("publish sensors.temperature value=23.1 cast=dev.sporeos.sensor")
	select {
	case msg := <-received:
		if msg.ArgIf("value", "") != "23.1" {
			t.Errorf("expected value '23.1', got '%s'", msg.ArgIf("value", ""))
		}
		if msg.Cast != "dev.sporeos.sensor" {
			t.Errorf("expected cast 'dev.sporeos.sensor', got '%s'", msg.Cast)
		}
	default:
		t.Error("callback was not called after deliverPublish")
	}
}

func TestSubscribe_SecondCall_OnlyUpdatesCallback(t *testing.T) {
	// Pre-seed only ONE ack response — if Subscribe sends a second wire message
	// the reader would EOF and SendAndWait would return an error.
	c := makeSubscribeClient("~sub1:SPORE.topic.subscribe ok capture=SPORE.hub")

	firstReceived := make(chan *PublishMessage, 1)
	if err := c.Subscribe("alerts.door", func(msg *PublishMessage) {
		firstReceived <- msg
	}, 500); err != nil {
		t.Fatalf("first Subscribe: %v", err)
	}
	writtenAfterFirst := len(c.conn.(*mockConn).written)

	// Second Subscribe for same topic — must not send any wire message.
	secondReceived := make(chan *PublishMessage, 1)
	if err := c.Subscribe("alerts.door", func(msg *PublishMessage) {
		secondReceived <- msg
	}, 500); err != nil {
		t.Fatalf("second Subscribe: %v", err)
	}
	if len(c.conn.(*mockConn).written) != writtenAfterFirst {
		t.Error("second Subscribe must not write to the connection")
	}

	// Deliver a publish — only the new (second) callback should fire.
	c.deliverPublish("publish alerts.door opened cast=dev.sporeos.security")
	select {
	case msg := <-secondReceived:
		if !msg.HasFlag("opened") {
			t.Error("expected 'opened' flag in publish")
		}
	default:
		t.Error("new callback was not called")
	}
	select {
	case <-firstReceived:
		t.Error("old callback should not have been called")
	default:
	}
}

func TestUnsubscribe_SendsWireMessageAndRemovesCallback(t *testing.T) {
	c := makeSubscribeClient("~sub1:SPORE.topic.unsubscribe ok capture=SPORE.hub")

	// Pre-register a callback as if a prior Subscribe had run.
	c.mu.Lock()
	c.publishHandlers["my.topic"] = func(msg *PublishMessage) {}
	c.mu.Unlock()

	if err := c.Unsubscribe("my.topic", 500); err != nil {
		t.Fatalf("Unsubscribe: %v", err)
	}

	// Wire message must have been sent.
	written := string(c.conn.(*mockConn).written)
	if !strings.Contains(written, "SPORE.topic.unsubscribe") {
		t.Errorf("expected SPORE.topic.unsubscribe in written, got: %s", written)
	}
	if !strings.Contains(written, "topic=my.topic") {
		t.Errorf("expected topic=my.topic in written, got: %s", written)
	}

	// Callback must be gone.
	c.mu.RLock()
	_, stillThere := c.publishHandlers["my.topic"]
	c.mu.RUnlock()
	if stillThere {
		t.Error("callback was not removed after Unsubscribe")
	}
}
