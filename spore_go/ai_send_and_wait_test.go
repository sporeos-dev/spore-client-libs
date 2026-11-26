// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: Apache-2.0

package spore_test

import (
	"testing"

	spore "github.com/sporeos-dev/spore-client-libs/spore_go"
	"github.com/sporeos-dev/spore-client-libs/spore_go/request"
	"github.com/sporeos-dev/spore-client-libs/spore_go/response"
)

func TestParseOkResponse(t *testing.T) {
	ok, rerr := response.Parse("~h1:State.retrieve ok body=hello")
	if rerr != nil || ok == nil {
		t.Fatalf("expected ok response, got ok=%v err=%v", ok, rerr)
	}
	if ok.Handle() != "h1" {
		t.Errorf("handle = %q, want h1", ok.Handle())
	}
	if v, _ := ok.Arg("body"); v != "hello" {
		t.Errorf("body = %q, want hello", v)
	}
}

func TestParseErrorResponse(t *testing.T) {
	ok, rerr := response.Parse(`~h1:State.retrieve error code=Missing what="nothing stored"`)
	if ok != nil || rerr == nil {
		t.Fatalf("expected error response, got ok=%v err=%v", ok, rerr)
	}
	if rerr.Code() != "Missing" || rerr.What() != "nothing stored" {
		t.Errorf("got code=%q what=%q", rerr.Code(), rerr.What())
	}
}

func TestParseGarbageReturnsNothing(t *testing.T) {
	ok, rerr := response.Parse("not a response")
	if ok != nil || rerr != nil {
		t.Errorf("expected nil, nil; got ok=%v err=%v", ok, rerr)
	}
}

func TestSendRequestAndWaitNotConnected(t *testing.T) {
	c := spore.New("test.sendandwait")
	req := request.New("State.retrieve", "h1")
	ok, rerr, err := c.SendRequestAndWait(req)
	if ok != nil || rerr != nil || err == nil {
		t.Errorf("expected only an error, got ok=%v rerr=%v err=%v", ok, rerr, err)
	}
}
