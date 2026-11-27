// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: Apache-2.0

package sporec

/*
#include "spore_c.h"
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"io"
	"unsafe"
)

type ResponseStream = C.spore_stream_t

func streamResult(result C.spore_stream_result_t) error {
	switch result {
	case C.SPORE_STREAM_OK:
		return nil
	case C.SPORE_STREAM_INVALID_ARGUMENT:
		return fmt.Errorf("invalid stream argument")
	case C.SPORE_STREAM_NOT_CONNECTED:
		return fmt.Errorf("stream client is not connected")
	case C.SPORE_STREAM_SEND_FAILED:
		return fmt.Errorf("failed to send stream frame")
	case C.SPORE_STREAM_CLOSED:
		return io.ErrClosedPipe
	default:
		return fmt.Errorf("unknown stream result: %d", result)
	}
}

func streamView(s string) (C.spore_string_view_t, func()) {
	p := C.CString(s)
	return C.spore_string_view_t{data: p, size: C.size_t(len(s))}, func() { C.free(unsafe.Pointer(p)) }
}

func StreamOpen(client *Client, req *Request, field string) (*ResponseStream, error) {
	f, free := streamView(field)
	defer free()
	var stream *ResponseStream
	err := streamResult(C.spore_stream_open(client, req, f, &stream))
	return stream, err
}

func StreamChunk(stream *ResponseStream, text string) error {
	t, free := streamView(text)
	defer free()
	return streamResult(C.spore_stream_chunk(stream, t))
}

func StreamClose(stream *ResponseStream) error {
	return streamResult(C.spore_stream_close(stream))
}

func StreamFail(stream *ResponseStream, code, what string) error {
	c, freeCode := streamView(code)
	defer freeCode()
	w, freeWhat := streamView(what)
	defer freeWhat()
	return streamResult(C.spore_stream_fail(stream, c, w))
}

func StreamDestroy(stream *ResponseStream) {
	C.spore_stream_destroy(stream)
}

func StreamSendStart(client *Client, handle, command string, fields []string) error {
	h, freeHandle := streamView(handle)
	defer freeHandle()
	c, freeCommand := streamView(command)
	defer freeCommand()
	views := make([]C.spore_string_view_t, len(fields))
	for i, field := range fields {
		var free func()
		views[i], free = streamView(field)
		defer free()
	}
	var ptr *C.spore_string_view_t
	if len(views) > 0 {
		ptr = &views[0]
	}
	return streamResult(C.spore_stream_send_start(client, h, c, ptr, C.size_t(len(views))))
}

func StreamSendChunk(client *Client, handle, field, text string) error {
	h, freeHandle := streamView(handle)
	defer freeHandle()
	f, freeField := streamView(field)
	defer freeField()
	t, freeText := streamView(text)
	defer freeText()
	return streamResult(C.spore_stream_send_chunk(client, h, f, t))
}

func StreamSendClose(client *Client, handle, field string) error {
	h, freeHandle := streamView(handle)
	defer freeHandle()
	f, freeField := streamView(field)
	defer freeField()
	return streamResult(C.spore_stream_send_close(client, h, f))
}
