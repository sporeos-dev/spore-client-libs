// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: Apache-2.0

package response

import "testing"

func TestSerializeIncludesOkFlag(t *testing.T) {
	r := New("HYPHAE.node.spawn", "hyphae-2")

	got := r.Serialize()
	want := "~hyphae-2:HYPHAE.node.spawn ok\n"
	if got != want {
		t.Fatalf("Serialize() = %q, want %q", got, want)
	}
}

func TestSerializeDoesNotDuplicateOkFlag(t *testing.T) {
	r := New("HYPHAE.node.spawn", "hyphae-2").WithFlag("ok")

	got := r.Serialize()
	want := "~hyphae-2:HYPHAE.node.spawn ok\n"
	if got != want {
		t.Fatalf("Serialize() = %q, want %q", got, want)
	}
}

func TestManifestContentSerializesOnOneLine(t *testing.T) {
	content := "name: example\r\nnote: \"<<\"\npath: \\name\n"
	r := New("manifest.read", "manifest-1").WithArg("content", content)
	wire := r.Serialize()
	want := `~manifest-1:manifest.read ok content="name: example\r\nnote: \"<<\"\npath: \\name\n"` + "\n"
	if wire != want {
		t.Fatalf("Serialize() = %q, want %q", wire, want)
	}
	parsed, rerr := Parse(wire)
	if parsed == nil || rerr != nil {
		t.Fatalf("Parse(%q) = %v, %v", wire, parsed, rerr)
	}
	if got, ok := parsed.Arg("content"); !ok || got != content {
		t.Fatalf("content = %q, %v; want %q, true", got, ok, content)
	}
}

func TestQuotedBlockMarkersAreLiteral(t *testing.T) {
	for _, wire := range []string{
		`~h:example.run ok content="<<literal>>"`,
		`~h:example.run ok content='<<literal>>'`,
		New("example.run", "h").WithArg("content", "<<literal>>").Serialize(),
	} {
		parsed, rerr := Parse(wire)
		if parsed == nil || rerr != nil {
			t.Fatalf("Parse(%q) = %v, %v", wire, parsed, rerr)
		}
		if got, _ := parsed.Arg("content"); got != "<<literal>>" {
			t.Errorf("Parse(%q) content = %q; want literal markers", wire, got)
		}
	}
}
