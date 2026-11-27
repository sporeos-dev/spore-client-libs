//go:build ai_stream_test && (darwin || linux)

// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: Apache-2.0

package spore

import (
	"bufio"
	"fmt"
	"runtime"
	"testing"
	"time"

	sporec "github.com/sporeos-dev/spore-client-libs/spore_go/internal/spore_c"
	"github.com/sporeos-dev/spore-client-libs/spore_go/response"
)

func TestOnStreamReceivesChunksBeforeResponse(t *testing.T) {
	client := New("test.stream.integration")
	peer, err := sporec.TestStreamSocket(client.h)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	started := false
	t.Cleanup(func() {
		peer.Close()
		if started {
			select {
			case err := <-done:
				if err != nil {
					t.Errorf("listen: %v", err)
				}
			case <-time.After(5 * time.Second):
				t.Error("listener did not stop")
				return
			}
		}
		runtime.SetFinalizer(client, nil)
		client.destroy()
	})
	if err := peer.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}

	chunks := make(chan *ResponseStream, 2)
	responses := make(chan string, 1)
	registrationErrors := make(chan struct{}, 1)
	var firstHandler *Handler
	firstHandler = client.OnStream(func(s *ResponseStream) {
		client.Off(firstHandler)
		if client.OnStream(func(next *ResponseStream) { chunks <- next }) == nil {
			registrationErrors <- struct{}{}
		}
		chunks <- s
	})
	if firstHandler == nil {
		t.Fatal("stream handler registration failed")
	}
	client.OnResponse(func(r *response.Response, e *response.ResponseError) {
		if e != nil {
			responses <- "error:" + e.Code()
			return
		}
		value, _ := r.Arg("output")
		responses <- value
	})
	started = true
	go func() { done <- client.Listen() }()

	ready, err := bufio.NewReader(peer).ReadString('\n')
	if err != nil || ready != "READY\n" {
		t.Fatalf("ready = %q, err = %v", ready, err)
	}
	if _, err := fmt.Fprint(peer, "~h:output<< first\\n\n"); err != nil {
		t.Fatal(err)
	}
	var first *ResponseStream
	select {
	case first = <-chunks:
	case <-time.After(5 * time.Second):
		t.Fatal("first chunk was not delivered")
	}
	select {
	case <-registrationErrors:
		t.Fatal("stream handler re-registration failed")
	default:
	}
	select {
	case <-responses:
		t.Fatal("response arrived before the stream completed")
	default:
	}
	if _, err := fmt.Fprint(peer, "~h:output<<  \\\\x\\q\\t \n~h:alias.run ok output=complete\n"); err != nil {
		t.Fatal(err)
	}
	var second *ResponseStream
	select {
	case second = <-chunks:
	case <-time.After(5 * time.Second):
		t.Fatal("second chunk was not delivered")
	}
	select {
	case value := <-responses:
		if value != "complete" {
			t.Fatalf("response output = %q", value)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("final response was not delivered")
	}
	if first.Handle() != "h" || first.Field() != "output" || first.Text() != "first\n" {
		t.Fatalf("first chunk = %q %q %q", first.Handle(), first.Field(), first.Text())
	}
	if second.Handle() != "h" || second.Field() != "output" || second.Text() != " \\x\\q\t " {
		t.Fatalf("second chunk = %q %q %q", second.Handle(), second.Field(), second.Text())
	}
}
