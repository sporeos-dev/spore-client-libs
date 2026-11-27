// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: Apache-2.0

package spore

import (
	"strings"
	"testing"

	"github.com/sporeos-dev/spore-client-libs/spore_go/request"
)

func TestOpenStreamReportsInvalidRequestAndSendFailure(t *testing.T) {
	client := New("test.stream")
	if _, err := client.OpenStream(nil, "output"); err == nil {
		t.Fatal("nil request accepted")
	}
	req := request.New("alias.run", "h")
	if _, err := client.OpenStream(req, "bad\x00field"); err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("opening with NUL = %v", err)
	}
	if _, err := client.OpenStream(req, "output"); err == nil || !strings.Contains(err.Error(), "not connected") {
		t.Fatalf("opening on disconnected client = %v", err)
	}
}

func TestLowLevelStreamInputReachesSharedValidation(t *testing.T) {
	client := New("test.stream")
	for _, fields := range [][]string{nil, {"output", "output"}, {"bad\nfield"}, {"bad\x00field"}} {
		if err := client.SendResponseStart("h", "example.run", fields...); err == nil || !strings.Contains(err.Error(), "invalid") {
			t.Errorf("opening with fields %q = %v", fields, err)
		}
	}
	if err := client.SendResponseChunk("h", "output", "a\x00b"); err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Errorf("chunk with NUL = %v", err)
	}
	if err := client.SendResponseClose("h", "output"); err == nil || !strings.Contains(err.Error(), "not connected") {
		t.Errorf("closing on disconnected client = %v", err)
	}
}
