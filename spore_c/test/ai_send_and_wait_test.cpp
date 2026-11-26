// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: Apache-2.0

#if !defined(_WIN32)

#include <atomic>
#include <chrono>
#include <condition_variable>
#include <gtest/gtest.h>
#include <map>
#include <mutex>
#include <set>
#include <string>
#include <string_view>
#include <thread>
#include <vector>
#include <sys/socket.h>
#include <unistd.h>

#include "spore_c.h"

// the tests hand the client one end of a socketpair, which needs its private socket fields
#define private public
#include "defs.h"
#undef private

namespace
{
    constexpr const char* kCmd = "dev.sporeos.SPORE.state.retrieve";

    std::atomic<int> handlerCalls{ 0 };
    std::string handlerHandle;
    std::atomic<bool> readySawResponse{ false };

    bool readyDoesCall(spore_client_t* c)
    {
        auto* req = spore_request_create();
        spore_request_set_command(req, kCmd);
        spore_request_set_handle(req, "setup1");
        readySawResponse = spore_client_send_and_wait(c, req, 2000);
        spore_request_destroy(req);
        return true;
    }

    bool readyFails(spore_client_t*)
    {
        return false;
    }

    void onResponse(spore_client_t*, const spore_response_t* ok, const spore_response_error_t*)
    {
        if (ok)
            handlerHandle = spore_response_get_handle(ok);
        handlerCalls++;
    }

    class SendAndWait : public ::testing::Test
    {
    protected:
        spore_client_t* client = nullptr;
        int hubFd = -1;
        std::thread reader;

        void SetUp() override
        {
            handlerCalls = 0;
            handlerHandle.clear();

            client = spore_client_create("test.node");
            int fds[2];
            ASSERT_EQ(socketpair(AF_UNIX, SOCK_STREAM, 0, fds), 0);
            client->client.socketFd = fds[0];
            client->client.connected = true;
            hubFd = fds[1];
        }

        void TearDown() override
        {
            close(hubFd);
            if (reader.joinable())
                reader.join();
            spore_client_destroy(client);
        }

        // handlers have to be registered before this; consumes the READY the client announces
        void startReader()
        {
            reader = std::thread([this] { spore_client_listen(client); });
            EXPECT_EQ(hubReadLine(), "READY");
        }

        // blank lines are skipped, as the hub does
        std::string hubReadLine()
        {
            std::string line;
            char c;
            while (read(hubFd, &c, 1) == 1)
            {
                if (c != '\n')
                    line += c;
                else if (!line.empty())
                    break;
            }
            return line;
        }

        void hubSend(const std::string& line)
        {
            std::string wire = line + "\n";
            ASSERT_EQ(write(hubFd, wire.data(), wire.size()), static_cast<ssize_t>(wire.size()));
        }

        spore_request_t* makeRequest(const char* handle)
        {
            auto* req = spore_request_create();
            spore_request_set_command(req, kCmd);
            spore_request_set_handle(req, handle);
            spore_request_add_arg(req, "key", "state");
            return req;
        }
    };
}

TEST_F(SendAndWait, ReadyIsSentImmediatelyWithoutAHandler)
{
    reader = std::thread([this] { spore_client_listen(client); });
    EXPECT_EQ(hubReadLine(), "READY");
}

TEST_F(SendAndWait, ReadyHandlerRunsBeforeReadyAndCanSendAndWait)
{
    readySawResponse = false;
    spore_client_set_ready_handler(client, readyDoesCall);
    reader = std::thread([this] { spore_client_listen(client); });

    std::string first = hubReadLine();
    EXPECT_NE(first.find(" ~setup1"), std::string::npos);
    hubSend(std::string("~setup1:") + kCmd + " ok");

    EXPECT_EQ(hubReadLine(), "READY");
    EXPECT_TRUE(readySawResponse);
}

TEST_F(SendAndWait, FailingReadyHandlerSendsWitnessInsteadOfReady)
{
    spore_client_set_ready_handler(client, readyFails);
    reader = std::thread([this] { spore_client_listen(client); });

    EXPECT_EQ(hubReadLine().rfind("witness", 0), 0u);

    timeval tv{ 0, 200000 };
    setsockopt(hubFd, SOL_SOCKET, SO_RCVTIMEO, &tv, sizeof(tv));
    EXPECT_EQ(hubReadLine(), "");
}

TEST_F(SendAndWait, ReturnsOkResponseWithMatchingHandle)
{
    startReader();
    std::thread hub([&] {
        hubReadLine();
        hubSend(std::string("~h1:") + kCmd + " ok body=hello");
    });

    auto* req = makeRequest("h1");
    ASSERT_TRUE(spore_client_send_and_wait(client, req, 2000));
    hub.join();

    const char* raw = spore_request_get_response(req);
    ASSERT_NE(raw, nullptr);
    auto* ok = spore_response_create_from_raw(raw, strlen(raw));
    ASSERT_NE(ok, nullptr);
    EXPECT_FALSE(spore_response_has_flag(ok, "error"));
    EXPECT_STREQ(spore_response_get_handle(ok), "h1");
    EXPECT_STREQ(spore_response_get_arg(ok, "body"), "hello");
    EXPECT_EQ(handlerCalls, 0);

    spore_response_destroy(ok);
    spore_request_destroy(req);
}

TEST_F(SendAndWait, ReturnsHubErrorAsError)
{
    startReader();
    std::thread hub([&] {
        hubReadLine();
        hubSend(std::string("~h1:") + kCmd + " error code=Missing what=\"nothing stored\"");
    });

    auto* req = makeRequest("h1");
    ASSERT_TRUE(spore_client_send_and_wait(client, req, 2000));
    hub.join();

    const char* raw = spore_request_get_response(req);
    ASSERT_NE(raw, nullptr);
    auto* err = spore_response_error_create_from_raw(raw, strlen(raw));
    ASSERT_NE(err, nullptr);
    EXPECT_STREQ(spore_response_error_get_code(err), "Missing");
    EXPECT_STREQ(spore_response_error_get_what(err), "nothing stored");

    spore_response_error_destroy(err);
    spore_request_destroy(req);
}

TEST_F(SendAndWait, UnrelatedResponsesStillReachHandlers)
{
    spore_client_register_response_handler(client, onResponse);
    startReader();
    std::thread hub([&] {
        hubReadLine();
        hubSend(std::string("~other:") + kCmd + " ok");
        hubSend(std::string("~h1:") + kCmd + " ok");
    });

    auto* req = makeRequest("h1");
    ASSERT_TRUE(spore_client_send_and_wait(client, req, 2000));
    hub.join();

    EXPECT_EQ(handlerCalls, 1);
    EXPECT_EQ(handlerHandle, "other");

    spore_request_destroy(req);
}

TEST_F(SendAndWait, SequentialCallsEachGetTheirOwnResponse)
{
    startReader();
    std::thread hub([&] {
        for (const char* handle : { "h1", "h2", "h3" })
        {
            hubReadLine();
            hubSend(std::string("~") + handle + ":" + kCmd + " ok");
        }
    });

    for (const char* handle : { "h1", "h2", "h3" })
    {
        auto* req = makeRequest(handle);
        ASSERT_TRUE(spore_client_send_and_wait(client, req, 2000));
        const char* raw = spore_request_get_response(req);
        ASSERT_NE(raw, nullptr);
        EXPECT_NE(std::string(raw).find(std::string("~") + handle + ":"), std::string::npos);
        spore_request_destroy(req);
    }
    hub.join();
}

TEST_F(SendAndWait, TimesOutAndLateResponseGoesToHandlers)
{
    spore_client_register_response_handler(client, onResponse);
    startReader();

    auto* req = makeRequest("h1");
    EXPECT_FALSE(spore_client_send_and_wait(client, req, 100));
    EXPECT_EQ(spore_request_get_response(req), nullptr);
    EXPECT_TRUE(client->client.waiters.empty());

    hubReadLine();
    hubSend(std::string("~h1:") + kCmd + " ok");
    for (int i = 0; i < 100 && handlerCalls == 0; ++i)
        std::this_thread::sleep_for(std::chrono::milliseconds(10));
    EXPECT_EQ(handlerCalls, 1);
    EXPECT_EQ(handlerHandle, "h1");

    spore_request_destroy(req);
}

TEST_F(SendAndWait, NotConnectedFailsWithoutWaiting)
{
    spore_client_t* other = spore_client_create("test.other");
    auto* req = makeRequest("h1");

    auto start = std::chrono::steady_clock::now();
    EXPECT_FALSE(spore_client_send_and_wait(other, req, 5000));
    EXPECT_LT(std::chrono::steady_clock::now() - start, std::chrono::seconds(1));
    EXPECT_TRUE(spore_client_has_error(other));

    spore_request_destroy(req);
    spore_client_destroy(other);
}

#endif
