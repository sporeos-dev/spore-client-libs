//go:build ai_stream_test && (darwin || linux)

// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: Apache-2.0

package sporec

/*
#cgo CXXFLAGS: -std=c++17 -I${SRCDIR}/../../../spore_c/include
#include "spore_c.h"
int ai_stream_test_socket(spore_client_t* client);
*/
import "C"

import (
	"fmt"
	"net"
	"os"
)

func TestStreamSocket(client *Client) (net.Conn, error) {
	fd := C.ai_stream_test_socket(client)
	if fd < 0 {
		return nil, fmt.Errorf("creating stream test socket failed")
	}
	file := os.NewFile(uintptr(fd), "stream-test")
	conn, err := net.FileConn(file)
	if closeErr := file.Close(); err == nil && closeErr != nil {
		conn.Close()
		return nil, closeErr
	}
	return conn, err
}
