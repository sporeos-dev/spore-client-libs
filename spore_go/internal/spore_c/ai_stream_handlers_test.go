// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: Apache-2.0

package sporec

import (
	"reflect"
	"testing"
)

func TestStreamHandlersDispatchAndRemove(t *testing.T) {
	client := ClientCreate("test.stream.handlers")
	defer ClientDestroy(client)

	var first, second []string
	h1 := ClientOnStream(client, func(_ *Client, handle, field, text string) {
		first = append(first, handle+":"+field+":"+text)
	})
	h2 := ClientOnStream(client, func(_ *Client, handle, field, text string) {
		second = append(second, handle+":"+field+":"+text)
	})
	if h1 == nil || h2 == nil {
		t.Fatal("stream handler registration failed")
	}

	dispatchStream(client, "h1", "output", "first\n")
	ClientOffHandler(client, h1)
	dispatchStream(client, "h2", "log", "  second\\x ")
	if !reflect.DeepEqual(first, []string{"h1:output:first\n"}) {
		t.Fatalf("removed handler received unexpected chunks: %q", first)
	}
	if !reflect.DeepEqual(second, []string{"h1:output:first\n", "h2:log:  second\\x "}) {
		t.Fatalf("remaining handler received unexpected chunks: %q", second)
	}

	ClientOffHandler(client, h2)
	if streamHandlers[key(client)] != nil || streamBridges[key(client)] != nil {
		t.Fatal("last removal did not clear stream registration")
	}
	h3 := ClientOnStream(client, func(_ *Client, _, _, _ string) { second = append(second, "again") })
	if h3 == nil {
		t.Fatal("stream handler re-registration failed")
	}
	dispatchStream(client, "h3", "output", "")
	if second[len(second)-1] != "again" {
		t.Fatalf("re-registered handler was not called: %q", second)
	}
}

func TestStreamHandlerRejectsNilRegistration(t *testing.T) {
	client := ClientCreate("test.stream.nil")
	defer ClientDestroy(client)
	if ClientOnStream(nil, func(*Client, string, string, string) {}) != nil ||
		ClientOnStream(client, nil) != nil {
		t.Fatal("nil client or callback was accepted")
	}
}
