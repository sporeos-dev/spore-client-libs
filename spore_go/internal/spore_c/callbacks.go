// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: Apache-2.0

package sporec

/*
#cgo CFLAGS: -I../../../spore_c/include
#include "spore_c.h"

// defined in bridges.c
extern spore_handler_t* registerRequestHandler(spore_client_t* c);
extern spore_handler_t* registerResponseHandler(spore_client_t* c);
extern spore_handler_t* registerWitnessHandler(spore_client_t* c);
extern spore_handler_t* registerPublishHandler(spore_client_t* c);
extern spore_handler_t* registerParseErrorHandler(spore_client_t* c);
*/
import "C"
import (
	"sync"
	"unsafe"
)

type RequestHandler func(c *Client, r *Request)
type ResponseHandler func(c *Client, r *Response, e *ResponseError)
type WitnessHandler func(c *Client, w *Witness)
type PublishHandler func(c *Client, p *Publish)
type ParseErrorHandler func(c *Client, code, what, raw string)

type handlerType uint8

const (
	requestHandlerType handlerType = iota
	responseHandlerType
	witnessHandlerType
	publishHandlerType
	parseErrorHandlerType
)

type Handler struct {
	client *Client
	type_  handlerType
	id     uint64
}

var (
	handlersMu    sync.Mutex
	nextHandlerID uint64

	requestHandlers    = make(map[uintptr]map[uint64]RequestHandler)
	responseHandlers   = make(map[uintptr]map[uint64]ResponseHandler)
	witnessHandlers    = make(map[uintptr]map[uint64]WitnessHandler)
	publishHandlers    = make(map[uintptr]map[uint64]PublishHandler)
	parseErrorHandlers = make(map[uintptr]map[uint64]ParseErrorHandler)

	requestBridges    = make(map[uintptr]*C.spore_handler_t)
	responseBridges   = make(map[uintptr]*C.spore_handler_t)
	witnessBridges    = make(map[uintptr]*C.spore_handler_t)
	publishBridges    = make(map[uintptr]*C.spore_handler_t)
	parseErrorBridges = make(map[uintptr]*C.spore_handler_t)
)

func key(c *Client) uintptr { return uintptr(unsafe.Pointer(c)) }

func nextID() uint64 {
	nextHandlerID++
	return nextHandlerID
}

//export goRequestBridge
func goRequestBridge(c *C.spore_client_t, r *C.spore_request_t) {
	handlersMu.Lock()
	handlers := make([]RequestHandler, 0, len(requestHandlers[key(c)]))
	for _, fn := range requestHandlers[key(c)] {
		handlers = append(handlers, fn)
	}
	handlersMu.Unlock()
	for _, fn := range handlers {
		fn(c, r)
	}
}

//export goResponseBridge
func goResponseBridge(c *C.spore_client_t, r *C.spore_response_t, e *C.spore_response_error_t) {
	handlersMu.Lock()
	handlers := make([]ResponseHandler, 0, len(responseHandlers[key(c)]))
	for _, fn := range responseHandlers[key(c)] {
		handlers = append(handlers, fn)
	}
	handlersMu.Unlock()
	for _, fn := range handlers {
		fn(c, r, e)
	}
}

//export goWitnessBridge
func goWitnessBridge(c *C.spore_client_t, w *C.spore_witness_t) {
	handlersMu.Lock()
	handlers := make([]WitnessHandler, 0, len(witnessHandlers[key(c)]))
	for _, fn := range witnessHandlers[key(c)] {
		handlers = append(handlers, fn)
	}
	handlersMu.Unlock()
	for _, fn := range handlers {
		fn(c, w)
	}
}

//export goPublishBridge
func goPublishBridge(c *C.spore_client_t, p *C.spore_publish_t) {
	handlersMu.Lock()
	handlers := make([]PublishHandler, 0, len(publishHandlers[key(c)]))
	for _, fn := range publishHandlers[key(c)] {
		handlers = append(handlers, fn)
	}
	handlersMu.Unlock()
	for _, fn := range handlers {
		fn(c, p)
	}
}

//export goParseErrorBridge
func goParseErrorBridge(c *C.spore_client_t, code *C.char, what *C.char, raw *C.char) {
	handlersMu.Lock()
	handlers := make([]ParseErrorHandler, 0, len(parseErrorHandlers[key(c)]))
	for _, fn := range parseErrorHandlers[key(c)] {
		handlers = append(handlers, fn)
	}
	handlersMu.Unlock()
	for _, fn := range handlers {
		fn(c, C.GoString(code), C.GoString(what), C.GoString(raw))
	}
}

func ClientOnRequest(c *Client, fn RequestHandler) *Handler {
	handlersMu.Lock()
	defer handlersMu.Unlock()
	clientKey := key(c)
	if requestHandlers[clientKey] == nil {
		requestHandlers[clientKey] = make(map[uint64]RequestHandler)
		requestBridges[clientKey] = C.registerRequestHandler(c)
	}
	id := nextID()
	requestHandlers[clientKey][id] = fn
	return &Handler{client: c, type_: requestHandlerType, id: id}
}

func ClientOnResponse(c *Client, fn ResponseHandler) *Handler {
	handlersMu.Lock()
	defer handlersMu.Unlock()
	clientKey := key(c)
	if responseHandlers[clientKey] == nil {
		responseHandlers[clientKey] = make(map[uint64]ResponseHandler)
		responseBridges[clientKey] = C.registerResponseHandler(c)
	}
	id := nextID()
	responseHandlers[clientKey][id] = fn
	return &Handler{client: c, type_: responseHandlerType, id: id}
}

func ClientOnWitness(c *Client, fn WitnessHandler) *Handler {
	handlersMu.Lock()
	defer handlersMu.Unlock()
	clientKey := key(c)
	if witnessHandlers[clientKey] == nil {
		witnessHandlers[clientKey] = make(map[uint64]WitnessHandler)
		witnessBridges[clientKey] = C.registerWitnessHandler(c)
	}
	id := nextID()
	witnessHandlers[clientKey][id] = fn
	return &Handler{client: c, type_: witnessHandlerType, id: id}
}

func ClientOnPublish(c *Client, fn PublishHandler) *Handler {
	handlersMu.Lock()
	defer handlersMu.Unlock()
	clientKey := key(c)
	if publishHandlers[clientKey] == nil {
		publishHandlers[clientKey] = make(map[uint64]PublishHandler)
		publishBridges[clientKey] = C.registerPublishHandler(c)
	}
	id := nextID()
	publishHandlers[clientKey][id] = fn
	return &Handler{client: c, type_: publishHandlerType, id: id}
}

func ClientOnParseError(c *Client, fn ParseErrorHandler) *Handler {
	handlersMu.Lock()
	defer handlersMu.Unlock()
	clientKey := key(c)
	if parseErrorHandlers[clientKey] == nil {
		parseErrorHandlers[clientKey] = make(map[uint64]ParseErrorHandler)
		parseErrorBridges[clientKey] = C.registerParseErrorHandler(c)
	}
	id := nextID()
	parseErrorHandlers[clientKey][id] = fn
	return &Handler{client: c, type_: parseErrorHandlerType, id: id}
}

func ClientOffHandler(c *Client, h *Handler) {
	if h == nil || h.client != c {
		return
	}

	handlersMu.Lock()
	defer handlersMu.Unlock()
	clientKey := key(c)
	switch h.type_ {
	case requestHandlerType:
		handlers, ok := requestHandlers[clientKey]
		if !ok {
			break
		}
		delete(handlers, h.id)
		if len(handlers) == 0 {
			C.spore_client_unregister_handler(c, requestBridges[clientKey])
			delete(requestHandlers, clientKey)
			delete(requestBridges, clientKey)
		}
	case responseHandlerType:
		handlers, ok := responseHandlers[clientKey]
		if !ok {
			break
		}
		delete(handlers, h.id)
		if len(handlers) == 0 {
			C.spore_client_unregister_handler(c, responseBridges[clientKey])
			delete(responseHandlers, clientKey)
			delete(responseBridges, clientKey)
		}
	case witnessHandlerType:
		handlers, ok := witnessHandlers[clientKey]
		if !ok {
			break
		}
		delete(handlers, h.id)
		if len(handlers) == 0 {
			C.spore_client_unregister_handler(c, witnessBridges[clientKey])
			delete(witnessHandlers, clientKey)
			delete(witnessBridges, clientKey)
		}
	case publishHandlerType:
		handlers, ok := publishHandlers[clientKey]
		if !ok {
			break
		}
		delete(handlers, h.id)
		if len(handlers) == 0 {
			C.spore_client_unregister_handler(c, publishBridges[clientKey])
			delete(publishHandlers, clientKey)
			delete(publishBridges, clientKey)
		}
	case parseErrorHandlerType:
		handlers, ok := parseErrorHandlers[clientKey]
		if !ok {
			break
		}
		delete(handlers, h.id)
		if len(handlers) == 0 {
			C.spore_client_unregister_handler(c, parseErrorBridges[clientKey])
			delete(parseErrorHandlers, clientKey)
			delete(parseErrorBridges, clientKey)
		}
	}
	h.client = nil
}

func clearClientHandlers(c *Client) {
	handlersMu.Lock()
	defer handlersMu.Unlock()
	clientKey := key(c)
	delete(requestHandlers, clientKey)
	delete(responseHandlers, clientKey)
	delete(witnessHandlers, clientKey)
	delete(publishHandlers, clientKey)
	delete(parseErrorHandlers, clientKey)
	delete(requestBridges, clientKey)
	delete(responseBridges, clientKey)
	delete(witnessBridges, clientKey)
	delete(publishBridges, clientKey)
	delete(parseErrorBridges, clientKey)
}
