//go:build ai_stream_test && (darwin || linux)

// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: Apache-2.0

#include <atomic>
#include <condition_variable>
#include <map>
#include <mutex>
#include <set>
#include <string>
#include <string_view>
#include <thread>
#include <vector>
#include <sys/socket.h>

#define private public
#include "../../../spore_c/source/defs.h"
#undef private

extern "C" int ai_stream_test_socket(spore_client_t* client)
{
    int fds[2];
    if (socketpair(AF_UNIX, SOCK_STREAM, 0, fds) != 0)
        return -1;
    client->client.socketFd = fds[0];
    client->client.connected = true;
    return fds[1];
}
