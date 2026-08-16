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

type RequestHandler  func(c *Client, r *Request)
type ResponseHandler func(c *Client, r *Response, e *ResponseError)
type WitnessHandler  func(c *Client, w *Witness)
type PublishHandler  func(c *Client, p *Publish)
type ParseErrorHandler func(c *Client, code, what, raw string)

var (
	requestHandlers    sync.Map // uintptr -> RequestHandler
	responseHandlers   sync.Map // uintptr -> ResponseHandler
	witnessHandlers    sync.Map // uintptr -> WitnessHandler
	publishHandlers    sync.Map // uintptr -> PublishHandler
	parseErrorHandlers sync.Map // uintptr -> ParseErrorHandler
)

func key(c *Client) uintptr { return uintptr(unsafe.Pointer(c)) }

//export goRequestBridge
func goRequestBridge(c *C.spore_client_t, r *C.spore_request_t) {
	if fn, ok := requestHandlers.Load(key(c)); ok {
		fn.(RequestHandler)(c, r)
	}
}

//export goResponseBridge
func goResponseBridge(c *C.spore_client_t, r *C.spore_response_t, e *C.spore_response_error_t) {
	if fn, ok := responseHandlers.Load(key(c)); ok {
		fn.(ResponseHandler)(c, r, e)
	}
}

//export goWitnessBridge
func goWitnessBridge(c *C.spore_client_t, w *C.spore_witness_t) {
	if fn, ok := witnessHandlers.Load(key(c)); ok {
		fn.(WitnessHandler)(c, w)
	}
}

//export goPublishBridge
func goPublishBridge(c *C.spore_client_t, p *C.spore_publish_t) {
	if fn, ok := publishHandlers.Load(key(c)); ok {
		fn.(PublishHandler)(c, p)
	}
}

//export goParseErrorBridge
func goParseErrorBridge(c *C.spore_client_t, code *C.char, what *C.char, raw *C.char) {
	if fn, ok := parseErrorHandlers.Load(key(c)); ok {
		fn.(ParseErrorHandler)(c, C.GoString(code), C.GoString(what), C.GoString(raw))
	}
}

func ClientOnRequest(c *Client, fn RequestHandler) *Handler {
	requestHandlers.Store(key(c), fn)
	return C.registerRequestHandler(c)
}

func ClientOnResponse(c *Client, fn ResponseHandler) *Handler {
	responseHandlers.Store(key(c), fn)
	return C.registerResponseHandler(c)
}

func ClientOnWitness(c *Client, fn WitnessHandler) *Handler {
	witnessHandlers.Store(key(c), fn)
	return C.registerWitnessHandler(c)
}

func ClientOnPublish(c *Client, fn PublishHandler) *Handler {
	publishHandlers.Store(key(c), fn)
	return C.registerPublishHandler(c)
}

func ClientOnParseError(c *Client, fn ParseErrorHandler) *Handler {
	parseErrorHandlers.Store(key(c), fn)
	return C.registerParseErrorHandler(c)
}

func ClientOffHandler(c *Client, h *Handler) {
	C.spore_client_unregister_handler(c, h)
}
