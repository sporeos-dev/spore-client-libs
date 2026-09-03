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