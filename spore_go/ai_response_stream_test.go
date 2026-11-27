// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: Apache-2.0

package spore

import "testing"

func TestStreamResponseOpening(t *testing.T) {
	got, err := responseStartFrame("h", "example.run", []string{"output", "log"})
	if want := "~h:example.run output=<< log=<<"; got != want || err != nil {
		t.Fatalf("opening = %q, %v; want %q", got, err, want)
	}
	for _, fields := range [][]string{nil, {"output", "output"}, {"bad\nfield"}} {
		if got, err := responseStartFrame("h", "example.run", fields); err == nil {
			t.Errorf("opening with fields %q accepted: %q", fields, got)
		}
	}
}

func TestStreamResponseChunkPreservesText(t *testing.T) {
	got, err := responseChunkFrame("h", "output", "  \"<<\" >> \\n\r\n\t ")
	want := `~h:output<<   "<<" >> \\n\r\n\t `
	if got != want || err != nil {
		t.Fatalf("chunk = %q, %v; want %q", got, err, want)
	}
}
