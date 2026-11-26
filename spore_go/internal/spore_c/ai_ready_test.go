// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: Apache-2.0

package sporec

import "testing"

func TestReadyBridgeReportsHandlerResult(t *testing.T) {
	client := ClientCreate("test.ready")
	defer ClientDestroy(client)

	if !goReadyBridge(client) {
		t.Fatal("without a ready handler the bridge should report success")
	}

	calls := 0
	ClientSetReady(client, func(*Client) bool { calls++; return false })
	if goReadyBridge(client) || calls != 1 {
		t.Fatalf("handler failure should be passed through, calls=%d", calls)
	}

	ClientSetReady(client, func(*Client) bool { calls++; return true })
	if !goReadyBridge(client) || calls != 2 {
		t.Fatalf("a later handler should replace the earlier one, calls=%d", calls)
	}
}
