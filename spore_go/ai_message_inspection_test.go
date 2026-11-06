// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: Apache-2.0

package spore_test

import (
	"reflect"
	"testing"

	sporec "github.com/sporeos-dev/spore-client-libs/spore_go/internal/spore_c"
	"github.com/sporeos-dev/spore-client-libs/spore_go/publish"
	"github.com/sporeos-dev/spore-client-libs/spore_go/request"
	"github.com/sporeos-dev/spore-client-libs/spore_go/response"
	"github.com/sporeos-dev/spore-client-libs/spore_go/witness"
)

type inspectable interface {
	Args() []string
	Flags() []string
	Raw() string
	Arg(string) (string, bool)
}

func TestMessageInspection(t *testing.T) {
	tests := []struct {
		name       string
		newMessage func() inspectable
		parse      func(string) (inspectable, func())
		raw        string
		flags      []string
	}{
		{"request", func() inspectable { return request.New("Example.run", "h") },
			func(raw string) (inspectable, func()) {
				h := sporec.RequestCreateFromRaw(raw)
				if h == nil {
					return nil, func() {}
				}
				return request.FromC(h), func() { sporec.RequestDestroy(h) }
			}, "Example.run ~h z=last a=first beta alpha\r\n", []string{"alpha", "beta"}},
		{"response", func() inspectable { return response.New("Example.run", "h") },
			func(raw string) (inspectable, func()) {
				h := sporec.ResponseCreateFromRaw(raw)
				if h == nil {
					return nil, func() {}
				}
				return response.FromC(h), func() { sporec.ResponseDestroy(h) }
			}, "~h:Example.run ok z=last a=first beta alpha\r\n", []string{"alpha", "beta", "ok"}},
		{"error response", func() inspectable { return response.Error("Example.run", "h", "Failed", "bad") },
			func(raw string) (inspectable, func()) {
				h := sporec.ResponseErrorCreateFromRaw(raw)
				if h == nil {
					return nil, func() {}
				}
				return response.ErrorFromC(h), func() { sporec.ResponseErrorDestroy(h) }
			}, "~h:Example.run error code=Failed what=bad z=last a=first beta alpha\r\n", []string{"alpha", "beta", "error"}},
		{"witness", func() inspectable { return witness.New("hello") },
			func(raw string) (inspectable, func()) {
				h := sporec.WitnessCreateFromRaw(raw)
				if h == nil {
					return nil, func() {}
				}
				return witness.FromC(h), func() { sporec.WitnessDestroy(h) }
			}, "witness body=\"hello\" z=last a=first beta alpha\r\n", []string{"alpha", "beta"}},
		{"publish", func() inspectable { return publish.New("Example.topic") },
			func(raw string) (inspectable, func()) {
				h := sporec.PublishCreateFromRaw(raw)
				if h == nil {
					return nil, func() {}
				}
				return publish.FromC(h), func() { sporec.PublishDestroy(h) }
			}, "publish Example.topic z=last a=first beta alpha\r\n", []string{"alpha", "beta"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			empty := tt.newMessage()
			if len(empty.Args()) != 0 || len(empty.Flags()) != 0 || empty.Raw() != "" {
				t.Fatalf("new message: args=%v flags=%v raw=%q", empty.Args(), empty.Flags(), empty.Raw())
			}
			msg, cleanup := tt.parse(tt.raw)
			defer cleanup()
			if msg == nil {
				t.Fatal("failed to parse test message")
			}
			if got := msg.Raw(); got != tt.raw {
				t.Errorf("Raw() = %q, want %q", got, tt.raw)
			}
			if got := msg.Args(); !reflect.DeepEqual(got, []string{"a", "z"}) {
				t.Errorf("Args() = %v", got)
			}
			if got := msg.Flags(); !reflect.DeepEqual(got, tt.flags) {
				t.Errorf("Flags() = %v, want %v", got, tt.flags)
			}
			if value, ok := msg.Arg("a"); !ok || value != "first" {
				t.Errorf("Arg(a) = %q, %v", value, ok)
			}
			args, flags := msg.Args(), msg.Flags()
			args[0], flags[0] = "changed", "changed"
			if msg.Args()[0] != "a" || msg.Flags()[0] != tt.flags[0] {
				t.Error("returned slices alias message storage")
			}
		})
	}
}

func TestRawPreservesMultilineInputAfterMutation(t *testing.T) {
	raw := "Example.run ~h text=<<\r\nhello world\r\n>> flag\r\n"
	h := sporec.RequestCreateFromRaw(raw)
	if h == nil {
		t.Fatal("failed to parse multiline request")
	}

	defer sporec.RequestDestroy(h)
	msg := request.FromC(h)
	if value, ok := msg.Arg("text"); !ok || value != "hello world" {
		t.Errorf("Arg(text) = %q, %v", value, ok)
	}
	if got := msg.Flags(); !reflect.DeepEqual(got, []string{"flag"}) {
		t.Errorf("Flags() = %v, want [flag]", got)
	}
	msg.WithArg("other", "value").WithFlag("another")
	sporec.RequestSerialize(h)
	if got := msg.Raw(); got != raw {
		t.Errorf("Raw() = %q, want %q", got, raw)
	}
}

func TestBuiltMessageInspectionAfterMutation(t *testing.T) {
	tests := []struct {
		name string
		msg  inspectable
	}{
		{"request", request.New("Example.run", "h").WithArg("z", "last").WithArg("a", "first").WithFlag("beta").WithFlag("alpha")},
		{"response", response.New("Example.run", "h").WithArg("z", "last").WithArg("a", "first").WithFlag("beta").WithFlag("alpha")},
		{"error response", response.Error("Example.run", "h", "Failed", "bad").WithArg("z", "last").WithArg("a", "first").WithFlag("beta").WithFlag("alpha")},
		{"witness", witness.New("hello").WithArg("z", "last").WithArg("a", "first").WithFlag("beta").WithFlag("alpha")},
		{"publish", publish.New("Example.topic").WithArg("z", "last").WithArg("a", "first").WithFlag("beta").WithFlag("alpha")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.msg.Args(); !reflect.DeepEqual(got, []string{"a", "z"}) {
				t.Errorf("Args() = %v", got)
			}
			if got := tt.msg.Flags(); !reflect.DeepEqual(got, []string{"alpha", "beta"}) {
				t.Errorf("Flags() = %v", got)
			}
			if got := tt.msg.Raw(); got != "" {
				t.Errorf("Raw() = %q, want empty", got)
			}
		})
	}
}
