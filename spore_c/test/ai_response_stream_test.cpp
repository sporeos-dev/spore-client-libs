// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: Apache-2.0

#include "spore_c.h"
#include <gtest/gtest.h>
#include <string>
#include <string_view>

namespace
{
    spore_string_view_t v(std::string_view s)
    {
        return {s.data(), s.size()};
    }
}

TEST(ResponseStreamInput, ReportsInvalidAndDisconnected)
{
    auto* client = spore_client_create("test.stream");
    auto* req = spore_request_create();
    spore_request_set_handle(req, "h");
    spore_request_set_command(req, "alias.run");
    spore_response_stream_t* stream = nullptr;
    EXPECT_EQ(spore_response_stream_open(client, req, v("bad field"), &stream),
              SPORE_STREAM_INVALID_ARGUMENT);
    EXPECT_EQ(stream, nullptr);
    EXPECT_EQ(spore_response_stream_open(client, req, v("output"), &stream),
              SPORE_STREAM_NOT_CONNECTED);
    EXPECT_EQ(stream, nullptr);
    spore_request_destroy(req);
    spore_client_destroy(client);
}

#if !defined(_WIN32)
#include <sys/socket.h>
#include <unistd.h>
#define private public
#include "defs.h"
#undef private

namespace
{
    class ResponseStreamSocket : public ::testing::Test
    {
    protected:
        spore_client_t* client = nullptr;
        spore_request_t* req = nullptr;
        int peer = -1;

        void SetUp() override
        {
            int fds[2];
            ASSERT_EQ(socketpair(AF_UNIX, SOCK_STREAM, 0, fds), 0);
            client = spore_client_create("test.stream");
            client->client.socketFd = fds[0];
            client->client.connected = true;
            peer = fds[1];
            req = spore_request_create();
            spore_request_set_handle(req, "h");
            spore_request_set_command(req, "alias.run");
        }

        void TearDown() override
        {
            spore_request_destroy(req);
            spore_client_destroy(client);
            if (peer >= 0)
                close(peer);
        }

        std::string readLine()
        {
            std::string line;
            char c;
            while (read(peer, &c, 1) == 1 && c != '\n')
                line += c;
            return line;
        }
    };
}

TEST_F(ResponseStreamSocket, CompletesOnceAndCopiesRequest)
{
    spore_response_stream_t* stream = nullptr;
    ASSERT_EQ(spore_response_stream_open(client, req, v("output"), &stream), SPORE_STREAM_OK);
    spore_request_destroy(req);
    req = nullptr;
    EXPECT_EQ(spore_response_stream_chunk(stream, v("  \"<<\" >> \\n\r\n\t ")), SPORE_STREAM_OK);
    EXPECT_EQ(spore_response_stream_close(stream), SPORE_STREAM_OK);
    EXPECT_EQ(spore_response_stream_chunk(stream, v("late")), SPORE_STREAM_CLOSED);
    EXPECT_EQ(spore_response_stream_close(stream), SPORE_STREAM_CLOSED);
    EXPECT_EQ(spore_response_stream_fail(stream, v("Execution"), v("late")), SPORE_STREAM_CLOSED);
    EXPECT_EQ(readLine(), "~h:alias.run output=<<");
    EXPECT_EQ(readLine(), R"(~h:output<<   "<<" >> \\n\r\n\t )");
    EXPECT_EQ(readLine(), "~h:output>>");
    EXPECT_EQ(readLine(), "~h:alias.run ok");
    spore_response_stream_destroy(stream);
}

TEST_F(ResponseStreamSocket, FailureClosesFieldAndPreservesOutput)
{
    spore_response_stream_t* stream = nullptr;
    ASSERT_EQ(spore_response_stream_open(client, req, v("output"), &stream), SPORE_STREAM_OK);
    ASSERT_EQ(spore_response_stream_chunk(stream, v("partial\n")), SPORE_STREAM_OK);
    EXPECT_EQ(spore_response_stream_fail(stream, v("Execution"), v("failed\nretry")), SPORE_STREAM_OK);
    EXPECT_EQ(readLine(), "~h:alias.run output=<<");
    EXPECT_EQ(readLine(), R"(~h:output<< partial\n)");
    EXPECT_EQ(readLine(), "~h:output>>");
    EXPECT_EQ(readLine(), R"(~h:alias.run error code=Execution what="failed\nretry" node_error)");
    spore_response_stream_destroy(stream);
}

TEST_F(ResponseStreamSocket, RejectsInvalidInputAndRemainsOpen)
{
    spore_response_stream_t* stream = nullptr;
    ASSERT_EQ(spore_response_stream_open(client, req, v("output"), &stream), SPORE_STREAM_OK);
    const std::string nul("bad\0text", 8);
    EXPECT_EQ(spore_response_stream_chunk(stream, v(nul)), SPORE_STREAM_INVALID_ARGUMENT);
    EXPECT_EQ(spore_response_stream_fail(stream, v("bad code"), v("reason")), SPORE_STREAM_INVALID_ARGUMENT);
    EXPECT_EQ(spore_response_stream_fail(stream, v("Execution"), v(nul)), SPORE_STREAM_INVALID_ARGUMENT);
    EXPECT_EQ(spore_response_stream_fail(stream, v("Execution"), v("")), SPORE_STREAM_INVALID_ARGUMENT);
    EXPECT_EQ(spore_response_stream_close(stream), SPORE_STREAM_OK);
    EXPECT_EQ(readLine(), "~h:alias.run output=<<");
    EXPECT_EQ(readLine(), "~h:output>>");
    EXPECT_EQ(readLine(), "~h:alias.run ok");
    spore_response_stream_destroy(stream);
}

TEST_F(ResponseStreamSocket, DisconnectionStopsFurtherOperations)
{
    spore_response_stream_t* stream = nullptr;
    ASSERT_EQ(spore_response_stream_open(client, req, v("output"), &stream), SPORE_STREAM_OK);
    EXPECT_EQ(readLine(), "~h:alias.run output=<<");
    spore_client_disconnect(client);
    EXPECT_EQ(spore_response_stream_chunk(stream, v("text")), SPORE_STREAM_NOT_CONNECTED);
    EXPECT_EQ(spore_response_stream_close(stream), SPORE_STREAM_NOT_CONNECTED);
    EXPECT_EQ(spore_response_stream_fail(stream, v("Execution"), v("reason")), SPORE_STREAM_NOT_CONNECTED);
    spore_response_stream_destroy(stream);
}

TEST_F(ResponseStreamSocket, SendFailureIsReportedByTheStream)
{
    spore_response_stream_t* stream = nullptr;
    ASSERT_EQ(spore_response_stream_open(client, req, v("output"), &stream), SPORE_STREAM_OK);
    EXPECT_EQ(readLine(), "~h:alias.run output=<<");
    close(peer);
    peer = -1;
    EXPECT_EQ(spore_response_stream_chunk(stream, v("text")), SPORE_STREAM_SEND_FAILED);
    EXPECT_EQ(spore_response_stream_close(stream), SPORE_STREAM_SEND_FAILED);
    spore_response_stream_destroy(stream);
}

TEST_F(ResponseStreamSocket, ConcurrentStreamsKeepFramesSeparate)
{
    spore_response_stream_t* first = nullptr;
    spore_response_stream_t* second = nullptr;
    ASSERT_EQ(spore_response_stream_open(client, req, v("output"), &first), SPORE_STREAM_OK);
    spore_request_set_handle(req, "other");
    ASSERT_EQ(spore_response_stream_open(client, req, v("output"), &second), SPORE_STREAM_OK);
    std::atomic<bool> failed{false};
    auto write = [&failed](spore_response_stream_t* stream, const char* text) {
        for (int i = 0; i < 20; ++i)
            if (spore_response_stream_chunk(stream, v(text)) != SPORE_STREAM_OK)
                failed = true;
        if (spore_response_stream_close(stream) != SPORE_STREAM_OK)
            failed = true;
    };
    std::thread a(write, first, "first");
    std::thread b(write, second, "second");
    a.join();
    b.join();
    ASSERT_FALSE(failed);
    EXPECT_EQ(readLine(), "~h:alias.run output=<<");
    EXPECT_EQ(readLine(), "~other:alias.run output=<<");
    int firstChunks = 0;
    int secondChunks = 0;
    int completions = 0;
    for (int i = 0; i < 44; ++i)
    {
        auto line = readLine();
        if (line == "~h:output<< first")
            ++firstChunks;
        else if (line == "~other:output<< second")
            ++secondChunks;
        else if (line == "~h:output>>" || line == "~other:output>>" ||
                 line == "~h:alias.run ok" || line == "~other:alias.run ok")
            ++completions;
        else
            ADD_FAILURE() << "unexpected frame: " << line;
    }
    EXPECT_EQ(firstChunks, 20);
    EXPECT_EQ(secondChunks, 20);
    EXPECT_EQ(completions, 4);
    spore_response_stream_destroy(first);
    spore_response_stream_destroy(second);
}

TEST_F(ResponseStreamSocket, LowLevelMultiFieldFrames)
{
    spore_string_view_t fields[] = {v("output"), v("log")};
    EXPECT_EQ(spore_stream_send_start(client, v("h"), v("alias.run"), fields, 2), SPORE_STREAM_OK);
    EXPECT_EQ(spore_stream_send_chunk(client, v("h"), v("log"), v("message\n")), SPORE_STREAM_OK);
    EXPECT_EQ(spore_stream_send_close(client, v("h"), v("log")), SPORE_STREAM_OK);
    EXPECT_EQ(readLine(), "~h:alias.run output=<< log=<<");
    EXPECT_EQ(readLine(), R"(~h:log<< message\n)");
    EXPECT_EQ(readLine(), "~h:log>>");
    fields[1] = fields[0];
    EXPECT_EQ(spore_stream_send_start(client, v("h"), v("alias.run"), fields, 2),
              SPORE_STREAM_INVALID_ARGUMENT);
    EXPECT_EQ(spore_stream_send_close(client, v("h"), v("bad\xc2\xa0" "field")),
              SPORE_STREAM_INVALID_ARGUMENT);
}
#endif
