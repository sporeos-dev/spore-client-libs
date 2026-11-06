// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: Apache-2.0

package response

import (
	"strings"
	"testing"
)

func TestCancelledResponseSerializesAsSuccessful(t *testing.T) {
	wire := New("file.open", "h").WithFlag("cancelled").Serialize()
	if !strings.Contains(wire, " ok") || !strings.Contains(wire, " cancelled") ||
		strings.Contains(wire, " error") {
		t.Fatalf("unexpected cancellation wire: %q", wire)
	}
}
