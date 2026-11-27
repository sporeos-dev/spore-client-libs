// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: Apache-2.0

#if !defined(_WIN32)

#include "spore_c.h"
#include <atomic>
#include <chrono>
#include <condition_variable>
#include <gtest/gtest.h>
#include <mutex>
#include <string>
#include <thread>
#include <vector>
#include <sys/socket.h>
#include <unistd.h>

#define private public
#include "defs.h"
#undef private

namespace
{
    struct Chunk
    {
        std::string handle;
        std::string field;
        std::string text;
    };

    std::mutex captureMu;
    std::condition_variable captureChanged;
    std::vector<Chunk> chunks;
    std::vector<std::string> responses;
    std::vector<std::string> errors;
    int removedCalls = 0;
    int secondCalls = 0;

    void onStream(spore_client_t*, const spore_stream_t* chunk)
    {
        size_t size = 0;
        const char* text = spore_stream_get_text(chunk, &size);
        std::lock_guard<std::mutex> lk(captureMu);
        chunks.push_back({spore_stream_get_handle(chunk),
                          spore_stream_get_field(chunk),
                          std::string(text, size)});
        captureChanged.notify_all();
    }

    void removedStream(spore_client_t*, const spore_stream_t*)
    {
        std::lock_guard<std::mutex> lk(captureMu);
        ++removedCalls;
    }

    void secondStream(spore_client_t*, const spore_stream_t*)
    {
        std::lock_guard<std::mutex> lk(captureMu);
        ++secondCalls;
    }

    void onResponse(spore_client_t*, const spore_response_t* response,
                    const spore_response_error_t* error)
    {
        std::lock_guard<std::mutex> lk(captureMu);
        responses.push_back(error ? std::string("error:") + spore_response_error_get_code(error)
                                  : spore_response_get_arg(response, "output"));
        captureChanged.notify_all();
    }

    void onParseError(spore_client_t*, const char* code, const char*, const char*)
    {
        std::lock_guard<std::mutex> lk(captureMu);
        errors.emplace_back(code);
        captureChanged.notify_all();
    }

    class StreamHandlers : public ::testing::Test
    {
    protected:
        spore_client_t* client = nullptr;
        int peer = -1;
        std::thread reader;

        void SetUp() override
        {
            chunks.clear();
            responses.clear();
            errors.clear();
            removedCalls = 0;
            secondCalls = 0;
            int fds[2];
            ASSERT_EQ(socketpair(AF_UNIX, SOCK_STREAM, 0, fds), 0);
            client = spore_client_create("test.stream.handlers");
            client->client.socketFd = fds[0];
            client->client.connected = true;
            peer = fds[1];
        }

        void TearDown() override
        {
            if (peer >= 0)
                close(peer);
            if (reader.joinable())
                reader.join();
            spore_client_destroy(client);
        }

        void send(const std::string& data)
        {
            ASSERT_EQ(write(peer, data.data(), data.size()), static_cast<ssize_t>(data.size()));
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

TEST_F(StreamHandlers, DeliversChunksIndividuallyBeforeCompleteResponse)
{
    ASSERT_NE(spore_client_register_stream_handler(client, onStream), nullptr);
    ASSERT_NE(spore_client_register_stream_handler(client, secondStream), nullptr);
    auto* removed = spore_client_register_stream_handler(client, removedStream);
    ASSERT_NE(removed, nullptr);
    spore_client_unregister_handler(client, removed);
    ASSERT_NE(spore_client_register_response_handler(client, onResponse), nullptr);
    ASSERT_NE(spore_client_register_parse_error_handler(client, onParseError), nullptr);
    reader = std::thread([&] { spore_client_listen(client); });
    EXPECT_EQ(readLine(), "READY");

    send("~a:output<");
    send("< first\\n\n");
    {
        std::unique_lock<std::mutex> lk(captureMu);
        EXPECT_TRUE(captureChanged.wait_for(lk, std::chrono::seconds(2), [] { return !chunks.empty(); }));
        EXPECT_TRUE(responses.empty());
    }
    send("~b:log<<  \\\\x\\q\\t \n"
         "~a:output<< \n"
         "~a:output<< end\\\n"
         "~a:alias.run ok output=\"first\\n\" capture=alias\n"
         "~b:alias.run ok output=\"literal<<text>>\" capture=alias\n");
    {
        std::unique_lock<std::mutex> lk(captureMu);
        EXPECT_TRUE(captureChanged.wait_for(lk, std::chrono::seconds(2), [] { return responses.size() == 2; }));
        ASSERT_EQ(chunks.size(), 4u);
        EXPECT_EQ(chunks[0].handle, "a");
        EXPECT_EQ(chunks[0].field, "output");
        EXPECT_EQ(chunks[0].text, "first\n");
        EXPECT_EQ(chunks[1].handle, "b");
        EXPECT_EQ(chunks[1].field, "log");
        EXPECT_EQ(chunks[1].text, " \\x\\q\t ");
        EXPECT_EQ(chunks[2].text, "");
        EXPECT_EQ(chunks[3].text, "end\\");
        EXPECT_EQ(responses[0], "first\n");
        EXPECT_EQ(responses[1], "literal<<text>>");
        EXPECT_EQ(removedCalls, 0);
        EXPECT_EQ(secondCalls, 4);
        EXPECT_TRUE(errors.empty());
    }
}

TEST_F(StreamHandlers, MalformedChunksReportParseErrors)
{
    spore_client_register_stream_handler(client, onStream);
    spore_client_register_parse_error_handler(client, onParseError);
    reader = std::thread([&] { spore_client_listen(client); });
    EXPECT_EQ(readLine(), "READY");
    send("~:output<< bad\n~h:<< bad\n~h:output<<missing-space\n");
    std::string nul = "~h:output<< bad";
    nul.push_back('\0');
    send(nul + "data\n");
    {
        std::unique_lock<std::mutex> lk(captureMu);
        EXPECT_TRUE(captureChanged.wait_for(lk, std::chrono::seconds(2), [] { return errors.size() == 4; }));
        EXPECT_TRUE(chunks.empty());
        for (const auto& error : errors)
            EXPECT_EQ(error, "Malformed");
    }
}

TEST_F(StreamHandlers, FinalErrorStillUsesResponseHandler)
{
    spore_client_register_stream_handler(client, onStream);
    spore_client_register_response_handler(client, onResponse);
    reader = std::thread([&] { spore_client_listen(client); });
    EXPECT_EQ(readLine(), "READY");
    send("~h:output<< partial\n"
         "~h:alias.run error code=Execution what=\"failed\" output=partial\n");
    {
        std::unique_lock<std::mutex> lk(captureMu);
        EXPECT_TRUE(captureChanged.wait_for(lk, std::chrono::seconds(2), [] { return !responses.empty(); }));
        ASSERT_EQ(chunks.size(), 1u);
        EXPECT_EQ(chunks[0].text, "partial");
        ASSERT_EQ(responses.size(), 1u);
        EXPECT_EQ(responses[0], "error:Execution");
    }
}

TEST_F(StreamHandlers, ChunksDoNotCompleteSynchronousWaiters)
{
    spore_client_register_stream_handler(client, onStream);
    spore_client_register_response_handler(client, onResponse);
    reader = std::thread([&] { spore_client_listen(client); });
    EXPECT_EQ(readLine(), "READY");
    auto* request = spore_request_create();
    spore_request_set_handle(request, "wait");
    spore_request_set_command(request, "alias.run");
    std::atomic<bool> done{false};
    bool received = false;
    std::thread waiter([&] {
        received = spore_client_send_and_wait(client, request, 3000);
        done = true;
    });
    EXPECT_NE(readLine().find("~wait"), std::string::npos);
    send("~wait:output<< live\n");
    {
        std::unique_lock<std::mutex> lk(captureMu);
        EXPECT_TRUE(captureChanged.wait_for(lk, std::chrono::seconds(2), [] { return !chunks.empty(); }));
        EXPECT_FALSE(done.load());
    }
    send("~wait:alias.run ok output=live\n");
    waiter.join();
    EXPECT_TRUE(received);
    ASSERT_NE(spore_request_get_response(request), nullptr);
    EXPECT_NE(std::string(spore_request_get_response(request)).find("output=live"), std::string::npos);
    {
        std::lock_guard<std::mutex> lk(captureMu);
        EXPECT_TRUE(responses.empty());
    }
    spore_request_destroy(request);
}

#endif
