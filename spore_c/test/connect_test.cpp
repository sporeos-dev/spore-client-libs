#include "fixture.h"

// --- Lifecycle ---

TEST_F(Fixture, CreateDestroy)
{
    ASSERT_NE(client, nullptr);
}

TEST_F(Fixture, InitiallyNotConnected)
{
    EXPECT_FALSE(spore_client_is_connected(client));
}

TEST_F(Fixture, InitiallyNoError)
{
    EXPECT_FALSE(spore_client_has_error(client));
    EXPECT_EQ(spore_client_get_error_code(client), nullptr);
    EXPECT_EQ(spore_client_get_error_what(client), nullptr);
}

TEST_F(Fixture, CreateNullNodeIdReturnsNull)
{
    EXPECT_EQ(spore_client_create(nullptr, false), nullptr);
}

// --- Connection (no daemon running) ---

TEST_F(Fixture, ConnectSetsError)
{
    spore_client_connect(client);
    // Either ConnectFailed (no daemon) or HandshakeFailed (daemon rejected us).
    EXPECT_TRUE(spore_client_has_error(client));
    EXPECT_NE(spore_client_get_error_code(client), nullptr);
    EXPECT_NE(spore_client_get_error_what(client), nullptr);
    EXPECT_FALSE(spore_client_is_connected(client));
}

TEST_F(Fixture, DisconnectClearsError)
{
    spore_client_connect(client);
    spore_client_disconnect(client);
    EXPECT_FALSE(spore_client_has_error(client));
}

// --- Null safety ---

TEST_F(Fixture, NullClientSafety)
{
    EXPECT_FALSE(spore_client_has_error(nullptr));
    EXPECT_EQ(spore_client_get_error_code(nullptr), nullptr);
    EXPECT_EQ(spore_client_get_error_what(nullptr), nullptr);
    EXPECT_FALSE(spore_client_is_connected(nullptr));
    spore_client_connect(nullptr);
    spore_client_disconnect(nullptr);
    spore_client_listen(nullptr);
}

// --- Handler registration ---

static void onRequest(spore_client_t*, const spore_request_t*)
{
}
static void onResponse(spore_client_t*, const spore_response_t*, const spore_response_error_t*)
{
}
static void onWitness(spore_client_t*, const spore_witness_t*)
{
}
static void onPublish(spore_client_t*, const spore_publish_t*)
{
}

TEST_F(Fixture, RegisterHandlers)
{
    auto* h1 = spore_client_register_request_handler(client, onRequest);
    auto* h2 = spore_client_register_response_handler(client, onResponse);
    auto* h3 = spore_client_register_witness_handler(client, onWitness);
    auto* h4 = spore_client_register_publish_handler(client, onPublish);
    EXPECT_NE(h1, nullptr);
    EXPECT_NE(h2, nullptr);
    EXPECT_NE(h3, nullptr);
    EXPECT_NE(h4, nullptr);
    spore_client_unregister_handler(client, h1);
    spore_client_unregister_handler(client, h2);
    spore_client_unregister_handler(client, h3);
    spore_client_unregister_handler(client, h4);
}

TEST_F(Fixture, MultipleHandlersDistinct)
{
    auto* h1 = spore_client_register_response_handler(client, onResponse);
    auto* h2 = spore_client_register_response_handler(client, onResponse);
    EXPECT_NE(h1, h2);
    spore_client_unregister_handler(client, h1);
    spore_client_unregister_handler(client, h2);
}

TEST_F(Fixture, RegisterNullFnReturnsNull)
{
    EXPECT_EQ(spore_client_register_request_handler(client, nullptr), nullptr);
    EXPECT_EQ(spore_client_register_response_handler(client, nullptr), nullptr);
    EXPECT_EQ(spore_client_register_witness_handler(client, nullptr), nullptr);
    EXPECT_EQ(spore_client_register_publish_handler(client, nullptr), nullptr);
}

// --- Request builder ---

TEST_F(Fixture, RequestCreateDestroy)
{
    auto* r = spore_request_create();
    ASSERT_NE(r, nullptr);
    spore_request_destroy(r);
}

TEST_F(Fixture, RequestSettersAndGetters)
{
    auto* r = spore_request_create();
    spore_request_set_command(r, "some.capability");
    spore_request_set_handle(r, "~abc1");
    spore_request_add_arg(r, "key", "value");
    spore_request_add_flag(r, "verbose");

    EXPECT_STREQ(spore_request_get_command(r), "some.capability");
    EXPECT_STREQ(spore_request_get_handle(r), "~abc1");
    EXPECT_STREQ(spore_request_get_arg(r, "key"), "value");
    EXPECT_EQ(spore_request_get_arg(r, "missing"), nullptr);
    EXPECT_TRUE(spore_request_has_flag(r, "verbose"));
    EXPECT_FALSE(spore_request_has_flag(r, "other"));

    spore_request_destroy(r);
}

TEST_F(Fixture, RequestNullSafety)
{
    EXPECT_EQ(spore_request_get_command(nullptr), nullptr);
    EXPECT_EQ(spore_request_get_handle(nullptr), nullptr);
    EXPECT_EQ(spore_request_get_arg(nullptr, "k"), nullptr);
    EXPECT_FALSE(spore_request_has_flag(nullptr, "f"));
    spore_request_destroy(nullptr);
}

// --- Response ok ---

TEST_F(Fixture, ResponseSettersAndGetters)
{
    auto* r = spore_response_create();
    spore_response_set_handle(r, "~abc1");
    spore_response_set_command(r, "some.capability");
    spore_response_add_arg(r, "result", "42");
    spore_response_add_flag(r, "done");

    EXPECT_STREQ(spore_response_get_handle(r), "~abc1");
    EXPECT_STREQ(spore_response_get_command(r), "some.capability");
    EXPECT_STREQ(spore_response_get_arg(r, "result"), "42");
    EXPECT_TRUE(spore_response_has_flag(r, "done"));

    spore_response_destroy(r);
}

TEST_F(Fixture, ResponseSerializesOkFlag)
{
    auto* r = spore_response_create();
    spore_response_set_handle(r, "hyphae-2");
    spore_response_set_command(r, "HYPHAE.node.spawn");
    spore_response_serialize(r);

    EXPECT_STREQ(spore_response_get_serialized(r), "~hyphae-2:HYPHAE.node.spawn ok\n");

    spore_response_destroy(r);
}

TEST_F(Fixture, ResponseDoesNotDuplicateOkFlag)
{
    auto* r = spore_response_create();
    spore_response_set_handle(r, "hyphae-2");
    spore_response_set_command(r, "HYPHAE.node.spawn");
    spore_response_add_flag(r, "ok");
    spore_response_serialize(r);

    const char* serialized = spore_response_get_serialized(r);
    EXPECT_STREQ(serialized, "~hyphae-2:HYPHAE.node.spawn ok\n");

    spore_response_destroy(r);
}

// --- Response error ---

TEST_F(Fixture, ResponseErrorSettersAndGetters)
{
    auto* e = spore_response_error_create();
    spore_response_error_set_handle(e, "~abc1");
    spore_response_error_set_command(e, "some.capability");
    spore_response_error_set_code(e, "NotFound");
    spore_response_error_set_what(e, "capability not registered");

    EXPECT_STREQ(spore_response_error_get_handle(e), "~abc1");
    EXPECT_STREQ(spore_response_error_get_command(e), "some.capability");
    EXPECT_STREQ(spore_response_error_get_code(e), "NotFound");
    EXPECT_STREQ(spore_response_error_get_what(e), "capability not registered");

    spore_response_error_destroy(e);
}

// --- Witness ---

TEST_F(Fixture, WitnessSettersAndGetters)
{
    auto* w = spore_witness_create();
    spore_witness_set_body(w, "some message body");
    spore_witness_add_arg(w, "source", "node1");
    spore_witness_add_flag(w, "incoming");

    EXPECT_STREQ(spore_witness_get_body(w), "some message body");
    EXPECT_STREQ(spore_witness_get_arg(w, "source"), "node1");
    EXPECT_TRUE(spore_witness_has_flag(w, "incoming"));

    spore_witness_destroy(w);
}

// --- Publish ---

TEST_F(Fixture, PublishSettersAndGetters)
{
    auto* p = spore_publish_create();
    spore_publish_set_topic(p, "events.tick");
    spore_publish_add_arg(p, "count", "5");

    EXPECT_STREQ(spore_publish_get_topic(p), "events.tick");
    EXPECT_STREQ(spore_publish_get_arg(p, "count"), "5");

    spore_publish_destroy(p);
}

// --- Send on disconnected ---

TEST_F(Fixture, SendNotConnectedSetsError)
{
    auto* r = spore_request_create();
    spore_request_set_command(r, "test.cap");
    spore_request_set_handle(r, "~1");
    spore_request_send(client, r);
    EXPECT_TRUE(spore_client_has_error(client));
    EXPECT_STREQ(spore_client_get_error_code(client), "NotConnected");
    spore_request_destroy(r);
}
