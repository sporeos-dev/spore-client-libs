// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: Apache-2.0

package spore

import "testing"

func TestReceivedStreamKeepsChunkDataAndCannotSend(t *testing.T) {
	stream := &ResponseStream{handle: "h", field: "output", text: "  hello\n"}
	if stream.Handle() != "h" || stream.Field() != "output" || stream.Text() != "  hello\n" {
		t.Fatalf("unexpected stream data: %q %q %q", stream.Handle(), stream.Field(), stream.Text())
	}
	if stream.Chunk("text") == nil || stream.Close() == nil || stream.Fail("code", "reason") == nil {
		t.Fatal("received chunk accepted a sending operation")
	}
}

func TestOnStreamRegistersAndRemovesHandler(t *testing.T) {
	client := New("test.stream.handlers")
	if client.OnStream(nil) != nil {
		t.Fatal("nil handler was accepted")
	}
	handler := client.OnStream(func(*ResponseStream) {})
	if handler == nil {
		t.Fatal("stream handler registration failed")
	}
	client.Off(handler)
}
