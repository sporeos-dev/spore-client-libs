package sporec

/*
#cgo CFLAGS: -I../../../spore_c/include
#cgo darwin LDFLAGS: -L../../../dist -lspore_c -lspore_parser -lc++
#cgo linux  LDFLAGS: -L../../../dist -lspore_c -lspore_parser -lstdc++
#include "spore_c.h"
#include <stdlib.h>
*/
import "C"
import "unsafe"

//
// types
//

type Client = C.spore_client_t
type Handler = C.spore_handler_t
type Request = C.spore_request_t
type Response = C.spore_response_t
type ResponseError = C.spore_response_error_t
type Witness = C.spore_witness_t
type Publish = C.spore_publish_t

//
// client
//

func ClientCreate(nodeID string, trace bool) *Client {
	cs := C.CString(nodeID)
	defer C.free(unsafe.Pointer(cs))
	t := C.bool(trace)
	return C.spore_client_create(cs, t)
}

func ClientDestroy(c *Client)        { C.spore_client_destroy(c) }
func ClientForceTrace(c *Client)     { C.spore_client_force_trace(c) }
func ClientConnect(c *Client)        { C.spore_client_connect(c) }
func ClientDisconnect(c *Client)     { C.spore_client_disconnect(c) }
func ClientIsConnected(c *Client) bool { return bool(C.spore_client_is_connected(c)) }

// error
func ClientHasError(c *Client) bool       { return bool(C.spore_client_has_error(c)) }
func ClientGetErrorCode(c *Client) string { return C.GoString(C.spore_client_get_error_code(c)) }
func ClientGetErrorWhat(c *Client) string { return C.GoString(C.spore_client_get_error_what(c)) }

// send/receive
func ClientListen(c *Client)                         { C.spore_client_listen(c) }
func ClientSendRaw(c *Client, data string) {
	cs := C.CString(data)
	defer C.free(unsafe.Pointer(cs))
	C.spore_client_send_raw(c, cs, C.size_t(len(data)))
}

//
//
// request
//

func RequestCreate() *Request { return C.spore_request_create() }
func RequestCreateFromRaw(raw string) *Request {
	cs := C.CString(raw)
	defer C.free(unsafe.Pointer(cs))
	return C.spore_request_create_from_raw(cs, C.size_t(len(raw)))
}
func RequestDestroy(r *Request) { C.spore_request_destroy(r) }

// setters
func RequestSetCommand(r *Request, cmd string) {
	cs := C.CString(cmd); defer C.free(unsafe.Pointer(cs))
	C.spore_request_set_command(r, cs)
}
func RequestSetHandle(r *Request, handle string) {
	cs := C.CString(handle); defer C.free(unsafe.Pointer(cs))
	C.spore_request_set_handle(r, cs)
}
func RequestAddArg(r *Request, key, value string) {
	ck := C.CString(key); defer C.free(unsafe.Pointer(ck))
	cv := C.CString(value); defer C.free(unsafe.Pointer(cv))
	C.spore_request_add_arg(r, ck, cv)
}
func RequestAddFlag(r *Request, flag string) {
	cs := C.CString(flag); defer C.free(unsafe.Pointer(cs))
	C.spore_request_add_flag(r, cs)
}

// getters
func RequestGetCommand(r *Request) string { return C.GoString(C.spore_request_get_command(r)) }
func RequestGetHandle(r *Request) string  { return C.GoString(C.spore_request_get_handle(r)) }
func RequestGetArg(r *Request, key string) string {
	ck := C.CString(key); defer C.free(unsafe.Pointer(ck))
	return C.GoString(C.spore_request_get_arg(r, ck))
}
func RequestHasFlag(r *Request, flag string) bool {
	cs := C.CString(flag); defer C.free(unsafe.Pointer(cs))
	return bool(C.spore_request_has_flag(r, cs))
}

// send/serialize
func RequestSend(c *Client, r *Request)        { C.spore_request_send(c, r) }
func RequestSerialize(r *Request)              { C.spore_request_serialize(r) }
func RequestGetSerialized(r *Request) string   { return C.GoString(C.spore_request_get_serialized(r)) }

//
//
// response
//

func ResponseCreate() *Response { return C.spore_response_create() }
func ResponseCreateFromRaw(raw string) *Response {
	cs := C.CString(raw); defer C.free(unsafe.Pointer(cs))
	return C.spore_response_create_from_raw(cs, C.size_t(len(raw)))
}
func ResponseDestroy(r *Response) { C.spore_response_destroy(r) }

// setters
func ResponseSetHandle(r *Response, handle string) {
	cs := C.CString(handle); defer C.free(unsafe.Pointer(cs))
	C.spore_response_set_handle(r, cs)
}
func ResponseSetCommand(r *Response, cmd string) {
	cs := C.CString(cmd); defer C.free(unsafe.Pointer(cs))
	C.spore_response_set_command(r, cs)
}
func ResponseAddArg(r *Response, key, value string) {
	ck := C.CString(key); defer C.free(unsafe.Pointer(ck))
	cv := C.CString(value); defer C.free(unsafe.Pointer(cv))
	C.spore_response_add_arg(r, ck, cv)
}
func ResponseAddFlag(r *Response, flag string) {
	cs := C.CString(flag); defer C.free(unsafe.Pointer(cs))
	C.spore_response_add_flag(r, cs)
}

// getters
func ResponseGetCommand(r *Response) string { return C.GoString(C.spore_response_get_command(r)) }
func ResponseGetHandle(r *Response) string  { return C.GoString(C.spore_response_get_handle(r)) }
func ResponseGetArg(r *Response, key string) string {
	ck := C.CString(key); defer C.free(unsafe.Pointer(ck))
	return C.GoString(C.spore_response_get_arg(r, ck))
}
func ResponseHasFlag(r *Response, flag string) bool {
	cs := C.CString(flag); defer C.free(unsafe.Pointer(cs))
	return bool(C.spore_response_has_flag(r, cs))
}

// send/serialize
func ResponseSend(c *Client, r *Response)       { C.spore_response_send(c, r) }
func ResponseSerialize(r *Response)             { C.spore_response_serialize(r) }
func ResponseGetSerialized(r *Response) string  { return C.GoString(C.spore_response_get_serialized(r)) }

//
//
// response error
//

func ResponseErrorCreate() *ResponseError { return C.spore_response_error_create() }
func ResponseErrorCreateFromRaw(raw string) *ResponseError {
	cs := C.CString(raw); defer C.free(unsafe.Pointer(cs))
	return C.spore_response_error_create_from_raw(cs, C.size_t(len(raw)))
}
func ResponseErrorDestroy(e *ResponseError) { C.spore_response_error_destroy(e) }

// setters
func ResponseErrorSetHandle(e *ResponseError, handle string) {
	cs := C.CString(handle); defer C.free(unsafe.Pointer(cs))
	C.spore_response_error_set_handle(e, cs)
}
func ResponseErrorSetCommand(e *ResponseError, cmd string) {
	cs := C.CString(cmd); defer C.free(unsafe.Pointer(cs))
	C.spore_response_error_set_command(e, cs)
}
func ResponseErrorSetCode(e *ResponseError, code string) {
	cs := C.CString(code); defer C.free(unsafe.Pointer(cs))
	C.spore_response_error_set_code(e, cs)
}
func ResponseErrorSetWhat(e *ResponseError, what string) {
	cs := C.CString(what); defer C.free(unsafe.Pointer(cs))
	C.spore_response_error_set_what(e, cs)
}
func ResponseErrorAddArg(e *ResponseError, key, value string) {
	ck := C.CString(key); defer C.free(unsafe.Pointer(ck))
	cv := C.CString(value); defer C.free(unsafe.Pointer(cv))
	C.spore_response_error_add_arg(e, ck, cv)
}
func ResponseErrorAddFlag(e *ResponseError, flag string) {
	cs := C.CString(flag); defer C.free(unsafe.Pointer(cs))
	C.spore_response_error_add_flag(e, cs)
}

// getters
func ResponseErrorGetCommand(e *ResponseError) string {
	return C.GoString(C.spore_response_error_get_command(e))
}
func ResponseErrorGetHandle(e *ResponseError) string {
	return C.GoString(C.spore_response_error_get_handle(e))
}
func ResponseErrorGetCode(e *ResponseError) string {
	return C.GoString(C.spore_response_error_get_code(e))
}
func ResponseErrorGetWhat(e *ResponseError) string {
	return C.GoString(C.spore_response_error_get_what(e))
}
func ResponseErrorGetArg(e *ResponseError, key string) string {
	ck := C.CString(key); defer C.free(unsafe.Pointer(ck))
	return C.GoString(C.spore_response_error_get_arg(e, ck))
}
func ResponseErrorHasFlag(e *ResponseError, flag string) bool {
	cs := C.CString(flag); defer C.free(unsafe.Pointer(cs))
	return bool(C.spore_response_error_has_flag(e, cs))
}

// send/serialize
func ResponseErrorSend(c *Client, e *ResponseError)      { C.spore_response_error_send(c, e) }
func ResponseErrorSerialize(e *ResponseError)            { C.spore_response_error_serialize(e) }
func ResponseErrorGetSerialized(e *ResponseError) string {
	return C.GoString(C.spore_response_error_get_serialized(e))
}

//
//
// witness
//

func WitnessCreate() *Witness { return C.spore_witness_create() }
func WitnessCreateFromRaw(raw string) *Witness {
	cs := C.CString(raw); defer C.free(unsafe.Pointer(cs))
	return C.spore_witness_create_from_raw(cs, C.size_t(len(raw)))
}
func WitnessDestroy(w *Witness) { C.spore_witness_destroy(w) }

// setters
func WitnessSetBody(w *Witness, body string) {
	cs := C.CString(body); defer C.free(unsafe.Pointer(cs))
	C.spore_witness_set_body(w, cs)
}
func WitnessAddArg(w *Witness, key, value string) {
	ck := C.CString(key); defer C.free(unsafe.Pointer(ck))
	cv := C.CString(value); defer C.free(unsafe.Pointer(cv))
	C.spore_witness_add_arg(w, ck, cv)
}
func WitnessAddFlag(w *Witness, flag string) {
	cs := C.CString(flag); defer C.free(unsafe.Pointer(cs))
	C.spore_witness_add_flag(w, cs)
}

// getters
func WitnessGetBody(w *Witness) string { return C.GoString(C.spore_witness_get_body(w)) }
func WitnessGetArg(w *Witness, key string) string {
	ck := C.CString(key); defer C.free(unsafe.Pointer(ck))
	return C.GoString(C.spore_witness_get_arg(w, ck))
}
func WitnessHasFlag(w *Witness, flag string) bool {
	cs := C.CString(flag); defer C.free(unsafe.Pointer(cs))
	return bool(C.spore_witness_has_flag(w, cs))
}

// send/serialize
func WitnessSend(c *Client, w *Witness)       { C.spore_witness_send(c, w) }
func WitnessSerialize(w *Witness)             { C.spore_witness_serialize(w) }
func WitnessGetSerialized(w *Witness) string  { return C.GoString(C.spore_witness_get_serialized(w)) }

//
//
// publish
//

func PublishCreate() *Publish { return C.spore_publish_create() }
func PublishCreateFromRaw(raw string) *Publish {
	cs := C.CString(raw); defer C.free(unsafe.Pointer(cs))
	return C.spore_publish_create_from_raw(cs, C.size_t(len(raw)))
}
func PublishDestroy(p *Publish) { C.spore_publish_destroy(p) }

// setters
func PublishSetTopic(p *Publish, topic string) {
	cs := C.CString(topic); defer C.free(unsafe.Pointer(cs))
	C.spore_publish_set_topic(p, cs)
}
func PublishAddArg(p *Publish, key, value string) {
	ck := C.CString(key); defer C.free(unsafe.Pointer(ck))
	cv := C.CString(value); defer C.free(unsafe.Pointer(cv))
	C.spore_publish_add_arg(p, ck, cv)
}
func PublishAddFlag(p *Publish, flag string) {
	cs := C.CString(flag); defer C.free(unsafe.Pointer(cs))
	C.spore_publish_add_flag(p, cs)
}

// getters
func PublishGetTopic(p *Publish) string { return C.GoString(C.spore_publish_get_topic(p)) }
func PublishGetArg(p *Publish, key string) string {
	ck := C.CString(key); defer C.free(unsafe.Pointer(ck))
	return C.GoString(C.spore_publish_get_arg(p, ck))
}
func PublishHasFlag(p *Publish, flag string) bool {
	cs := C.CString(flag); defer C.free(unsafe.Pointer(cs))
	return bool(C.spore_publish_has_flag(p, cs))
}

// send/serialize
func PublishSend(c *Client, p *Publish)       { C.spore_publish_send(c, p) }
func PublishSerialize(p *Publish)             { C.spore_publish_serialize(p) }
func PublishGetSerialized(p *Publish) string  { return C.GoString(C.spore_publish_get_serialized(p)) }
