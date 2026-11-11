// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: Apache-2.0

package spore_test

import (
	"testing"

	sporec "github.com/sporeos-dev/spore-client-libs/spore_go/internal/spore_c"
	"github.com/sporeos-dev/spore-client-libs/spore_go/request"
	"github.com/sporeos-dev/spore-client-libs/spore_go/response"
	"github.com/sporeos-dev/spore-client-libs/spore_go/witness"
)

func TestParserQuotedArgumentEscapes(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"double quoted", `example.run path="a \"quote\" \\path\nline\rreturn\ttab" ~h`, "a \"quote\" \\path\nline\rreturn\ttab"},
		{"unknown escape", `example.run path="a\q" ~h`, `a\q`},
		{"single quoted literal", `example.run path='a\n"quoted"\path' ~h`, `a\n"quoted"\path`},
		{"literal enclosing quotes", `example.run path='"quoted"' ~h`, `"quoted"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := sporec.RequestCreateFromRaw(tt.raw)
			if h == nil {
				t.Fatal("request parsing failed")
			}
			defer sporec.RequestDestroy(h)
			req := request.FromC(h)
			got, ok := req.Arg("path")
			if !ok || got != tt.want {
				t.Errorf("path = %q, %v; want %q, true", got, ok, tt.want)
			}
			if req.Handle() != "h" {
				t.Errorf("handle = %q, want h", req.Handle())
			}
		})
	}
}

func TestSerializedTextRoundTrips(t *testing.T) {
	values := []string{
		`C:\new folder\test`,
		`it's "quoted"`,
		"first line\nsecond line",
		"tab\tand\rreturn",
	}

	for _, value := range values {
		t.Run(value, func(t *testing.T) {
			original := request.New("example.run", "h").WithArg("path", value)
			sporec.RequestSerialize(original.H())
			wire := sporec.RequestGetSerialized(original.H())
			parsed := sporec.RequestCreateFromRaw(wire)
			if parsed == nil {
				t.Fatalf("failed to parse request %q", wire)
			}
			defer sporec.RequestDestroy(parsed)
			if got, ok := request.FromC(parsed).Arg("path"); !ok || got != value {
				t.Errorf("request path = %q, %v; want %q, true", got, ok, value)
			}

			errWire := response.Error("example.run", "h", "Generic.Node", value).Serialize()
			parsedError := sporec.ResponseErrorCreateFromRaw(errWire)
			if parsedError == nil {
				t.Fatalf("failed to parse error %q", errWire)
			}
			defer sporec.ResponseErrorDestroy(parsedError)
			if got := response.ErrorFromC(parsedError).What(); got != value {
				t.Errorf("error what = %q, want %q", got, value)
			}

			originalWitness := witness.New(value)
			sporec.WitnessSerialize(originalWitness.H())
			witnessWire := sporec.WitnessGetSerialized(originalWitness.H())
			parsedWitness := sporec.WitnessCreateFromRaw(witnessWire)
			if parsedWitness == nil {
				t.Fatalf("failed to parse witness %q", witnessWire)
			}
			defer sporec.WitnessDestroy(parsedWitness)
			if got := witness.FromC(parsedWitness).Body(); got != value {
				t.Errorf("witness body = %q, want %q", got, value)
			}
		})
	}
}
