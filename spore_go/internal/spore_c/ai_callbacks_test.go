// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: Apache-2.0

package sporec

import "testing"

func TestMultipleHandlersReceiveCallbacksAndOffRemovesOne(t *testing.T) {
	client := ClientCreate("go-handler-test")
	defer ClientDestroy(client)

	var first, second [5]int
	requestHandler := ClientOnRequest(client, func(*Client, *Request) { first[0]++ })
	ClientOnRequest(client, func(*Client, *Request) { second[0]++ })
	responseHandler := ClientOnResponse(client, func(*Client, *Response, *ResponseError) { first[1]++ })
	ClientOnResponse(client, func(*Client, *Response, *ResponseError) { second[1]++ })
	witnessHandler := ClientOnWitness(client, func(*Client, *Witness) { first[2]++ })
	ClientOnWitness(client, func(*Client, *Witness) { second[2]++ })
	publishHandler := ClientOnPublish(client, func(*Client, *Publish) { first[3]++ })
	ClientOnPublish(client, func(*Client, *Publish) { second[3]++ })
	parseErrorHandler := ClientOnParseError(client, func(*Client, string, string, string) { first[4]++ })
	ClientOnParseError(client, func(*Client, string, string, string) { second[4]++ })

	goRequestBridge(client, nil)
	goResponseBridge(client, nil, nil)
	goWitnessBridge(client, nil)
	goPublishBridge(client, nil)
	goParseErrorBridge(client, nil, nil, nil)

	if first != [5]int{1, 1, 1, 1, 1} || second != [5]int{1, 1, 1, 1, 1} {
		t.Fatalf("all handlers should receive their callback once: first=%v second=%v", first, second)
	}

	ClientOffHandler(client, requestHandler)
	ClientOffHandler(client, responseHandler)
	ClientOffHandler(client, witnessHandler)
	ClientOffHandler(client, publishHandler)
	ClientOffHandler(client, parseErrorHandler)

	goRequestBridge(client, nil)
	goResponseBridge(client, nil, nil)
	goWitnessBridge(client, nil)
	goPublishBridge(client, nil)
	goParseErrorBridge(client, nil, nil, nil)

	if first != [5]int{1, 1, 1, 1, 1} {
		t.Fatalf("removed handlers received another callback: %v", first)
	}
	if second != [5]int{2, 2, 2, 2, 2} {
		t.Fatalf("remaining handlers should receive their second callback: %v", second)
	}
}
