// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: Apache-2.0

#include "spore_c.h"
#include "defs.h"
#include "spore_parser.h"
#include <iostream>

// Helper: return c_str() or nullptr when the string_view is empty.
static const char* sv(std::string_view s)
{
    return s.empty() ? nullptr : s.data();
}

static void parseRaw(spore_parser_t* parser, const char* raw, size_t size, spore_message_t* message)
{
    std::string input;
    input.reserve(size);
    for (size_t i = 0; i < size; ++i)
    {
        if (raw[i] == '\r' && i + 1 < size && raw[i + 1] == '\n')
            continue;
        input += raw[i];
    }
    if (!input.empty() && input.back() == '\n')
        input.pop_back();
    spore_parse(parser, input.data(), input.size(), message);
}

// Populate a Message from a parsed spore_message_t (args + flags).
static void populateMessage(spore::Message& msg, spore_message_t* m, bool skipErrorFields = false)
{
    msg.setCommand(spore_message_get_capability(m) ? spore_message_get_capability(m) : "");
    msg.setHandle(spore_message_get_handle(m) ? spore_message_get_handle(m) : "");
    size_t nArgs = 0;
    const spore_arg_t* args = spore_message_get_args(m, &nArgs);
    for (size_t i = 0; i < nArgs; ++i)
    {
        std::string_view key(args[i].pKey);
        if (!skipErrorFields || (key != "code" && key != "what"))
            msg.addArg(key, args[i].pValue);
    }
    size_t nFlags = 0;
    const char** flags = spore_message_get_flags(m, &nFlags);
    for (size_t i = 0; i < nFlags; ++i) msg.addFlag(flags[i]);
}

static size_t copyArgIds(const spore::Message& msg, const char** out, size_t capacity)
{
    const auto& args = msg.getArgsMap();
    size_t i = 0;
    for (const auto& arg : args)
    {
        if (out && i < capacity)
            out[i] = arg.first.c_str();
        ++i;
    }
    return args.size();
}

static size_t copyFlags(const spore::Message& msg, const char** out, size_t capacity)
{
    const auto& flags = msg.getFlagsSet();
    size_t i = 0;
    for (const auto& flag : flags)
    {
        if (out && i < capacity)
            out[i] = flag.c_str();
        ++i;
    }
    return flags.size();
}

#define SPORE_MESSAGE_ACCESSORS(name, type, member)                                      \
    size_t spore_##name##_get_arg_ids(const type* h, const char** out, size_t capacity)  \
    {                                                                                    \
        return h ? copyArgIds(h->member, out, capacity) : 0;                              \
    }                                                                                    \
    size_t spore_##name##_get_flags(const type* h, const char** out, size_t capacity)    \
    {                                                                                    \
        return h ? copyFlags(h->member, out, capacity) : 0;                               \
    }                                                                                    \
    const char* spore_##name##_get_raw(const type* h, size_t* size)                      \
    {                                                                                    \
        const auto raw = h ? h->member.getRaw() : std::string_view{};                     \
        if (size) *size = raw.size();                                                     \
        return raw.data();                                                               \
    }

SPORE_MESSAGE_ACCESSORS(request, spore_request_t, request)
SPORE_MESSAGE_ACCESSORS(response, spore_response_t, response)
SPORE_MESSAGE_ACCESSORS(response_error, spore_response_error_t, error)
SPORE_MESSAGE_ACCESSORS(witness, spore_witness_t, witness)
SPORE_MESSAGE_ACCESSORS(publish, spore_publish_t, publish)

#undef SPORE_MESSAGE_ACCESSORS

// ============================================================
// client
// ============================================================

spore_client_t* spore_client_create(const char* pNodeId)
{
    if (!pNodeId)
        return nullptr;
    auto* h = new spore_client_t{ spore::Client(pNodeId) };
    h->client.self = h;
    return h;
}

void spore_client_destroy(spore_client_t* hClient)
{
    delete hClient;
}

void spore_client_connect(spore_client_t* hClient)
{
    if (!hClient)
        return;
    hClient->client.connect();
}

void spore_client_disconnect(spore_client_t* hClient)
{
    if (!hClient)
        return;
    hClient->client.disconnect();
}

bool spore_client_is_connected(spore_client_t* hClient)
{
    return hClient && hClient->client.isConnected();
}

bool spore_client_has_error(spore_client_t* hClient)
{
    return hClient && hClient->client.hasError();
}

const char* spore_client_get_error_code(spore_client_t* hClient)
{
    if (!hClient || !hClient->client.hasError())
        return nullptr;
    return sv(hClient->client.getErrorCode());
}

const char* spore_client_get_error_what(spore_client_t* hClient)
{
    if (!hClient || !hClient->client.hasError())
        return nullptr;
    return sv(hClient->client.getErrorWhat());
}

// --- Handlers ---

spore_handler_t* spore_client_register_request_handler(spore_client_t* hClient, spore_request_fn fn)
{
    if (!hClient || !fn)
        return nullptr;
    return hClient->client.registerRequestHandler(fn);
}

spore_handler_t* spore_client_register_response_handler(spore_client_t* hClient,
                                                        spore_response_fn fn)
{
    if (!hClient || !fn)
        return nullptr;
    return hClient->client.registerResponseHandler(fn);
}

spore_handler_t* spore_client_register_witness_handler(spore_client_t* hClient, spore_witness_fn fn)
{
    if (!hClient || !fn)
        return nullptr;
    return hClient->client.registerWitnessHandler(fn);
}

spore_handler_t* spore_client_register_publish_handler(spore_client_t* hClient, spore_publish_fn fn)
{
    if (!hClient || !fn)
        return nullptr;
    return hClient->client.registerPublishHandler(fn);
}

spore_handler_t* spore_client_register_parse_error_handler(spore_client_t* hClient,
                                                           spore_parse_error_fn fn)
{
    if (!hClient || !fn)
        return nullptr;
    return hClient->client.registerParseErrorHandler(fn);
}

void spore_client_unregister_handler(spore_client_t* hClient, spore_handler_t* hHandler)
{
    if (!hClient)
        return;
    hClient->client.unregisterHandler(hHandler);
}

void spore_client_listen(spore_client_t* hClient)
{
    if (!hClient)
        return;
    hClient->client.listen();
}

void spore_client_send_raw(spore_client_t* hClient, const char* pData, size_t sz)
{
    if (!hClient || !pData)
        return;
    hClient->client.sendRaw(pData, sz);
}

// ============================================================
// request
// ============================================================

spore_request_t* spore_request_create()
{
    return new spore_request_t{};
}

spore_request_t* spore_request_create_from_raw(const char* pRaw, size_t sz)
{
    if (!pRaw)
        return nullptr;
    auto* p = spore_parser_create();
    auto* m = spore_message_create();
    parseRaw(p, pRaw, sz, m);
    spore_request_t* result = nullptr;
    if (spore_parser_get_type(p) == SPORE_PARSER_TYPE_REQUEST)
    {
        result = new spore_request_t{};
        populateMessage(result->request, m);
        result->request.setRaw(std::string_view(pRaw, sz));
    }
    spore_message_destroy(m);
    spore_parser_destroy(p);
    return result;
}

void spore_request_destroy(spore_request_t* hRequest)
{
    delete hRequest;
}

void spore_request_set_command(spore_request_t* hRequest, const char* pCommand)
{
    if (!hRequest || !pCommand)
        return;
    hRequest->request.setCommand(pCommand);
}

void spore_request_set_handle(spore_request_t* hRequest, const char* pHandle)
{
    if (!hRequest || !pHandle)
        return;
    hRequest->request.setHandle(pHandle);
}

void spore_request_add_arg(spore_request_t* hRequest, const char* pKey, const char* pValue)
{
    if (!hRequest || !pKey || !pValue)
        return;
    hRequest->request.addArg(pKey, pValue);
}

void spore_request_add_flag(spore_request_t* hRequest, const char* pFlag)
{
    if (!hRequest || !pFlag)
        return;
    hRequest->request.addFlag(pFlag);
}

const char* spore_request_get_command(const spore_request_t* hRequest)
{
    if (!hRequest)
        return nullptr;
    return sv(hRequest->request.getCommand());
}

const char* spore_request_get_handle(const spore_request_t* hRequest)
{
    if (!hRequest)
        return nullptr;
    return sv(hRequest->request.getHandle());
}

const char* spore_request_get_arg(const spore_request_t* hRequest, const char* pKey)
{
    if (!hRequest || !pKey)
        return nullptr;
    return sv(hRequest->request.getArg(pKey));
}

bool spore_request_has_flag(const spore_request_t* hRequest, const char* pFlag)
{
    return hRequest && pFlag && hRequest->request.hasFlag(pFlag);
}

void spore_request_send(spore_client_t* hClient, const spore_request_t* hRequest)
{
    if (!hClient)
        return;
    hClient->client.send(hRequest);
}

void spore_request_serialize(spore_request_t* hRequest)
{
    if (!hRequest)
        return;
    hRequest->request.serialize();
}

const char* spore_request_get_serialized(const spore_request_t* hRequest)
{
    if (!hRequest)
        return nullptr;
    return hRequest->request.getSerialized();
}

// ============================================================
// response (ok)
// ============================================================

spore_response_t* spore_response_create()
{
    return new spore_response_t{};
}

spore_response_t* spore_response_create_from_raw(const char* pRaw, size_t sz)
{
    if (!pRaw)
        return nullptr;
    auto* p = spore_parser_create();
    auto* m = spore_message_create();
    parseRaw(p, pRaw, sz, m);
    spore_response_t* result = nullptr;
    if (spore_parser_get_type(p) == SPORE_PARSER_TYPE_RESPONSE)
    {
        result = new spore_response_t{};
        populateMessage(result->response, m);
        result->response.setRaw(std::string_view(pRaw, sz));
    }
    spore_message_destroy(m);
    spore_parser_destroy(p);
    return result;
}

void spore_response_destroy(spore_response_t* hResponse)
{
    delete hResponse;
}

void spore_response_set_handle(spore_response_t* hResponse, const char* pHandle)
{
    if (!hResponse || !pHandle)
        return;
    hResponse->response.setHandle(pHandle);
}

void spore_response_set_command(spore_response_t* hResponse, const char* pCommand)
{
    if (!hResponse || !pCommand)
        return;
    hResponse->response.setCommand(pCommand);
}

void spore_response_add_arg(spore_response_t* hResponse, const char* pKey, const char* pValue)
{
    if (!hResponse || !pKey || !pValue)
        return;
    hResponse->response.addArg(pKey, pValue);
}

void spore_response_add_flag(spore_response_t* hResponse, const char* pFlag)
{
    if (!hResponse || !pFlag)
        return;
    hResponse->response.addFlag(pFlag);
}

const char* spore_response_get_command(const spore_response_t* hResponse)
{
    if (!hResponse)
        return nullptr;
    return sv(hResponse->response.getCommand());
}

const char* spore_response_get_handle(const spore_response_t* hResponse)
{
    if (!hResponse)
        return nullptr;
    return sv(hResponse->response.getHandle());
}

const char* spore_response_get_arg(const spore_response_t* hResponse, const char* pKey)
{
    if (!hResponse || !pKey)
        return nullptr;
    return sv(hResponse->response.getArg(pKey));
}

bool spore_response_has_flag(const spore_response_t* hResponse, const char* pFlag)
{
    return hResponse && pFlag && hResponse->response.hasFlag(pFlag);
}

void spore_response_send(spore_client_t* hClient, const spore_response_t* hResponse)
{
    if (!hClient)
        return;
    hClient->client.sendResponse(hResponse);
}

void spore_response_serialize(spore_response_t* hResponse)
{
    if (!hResponse)
        return;
    hResponse->response.serialize();
}

const char* spore_response_get_serialized(const spore_response_t* hResponse)
{
    if (!hResponse)
        return nullptr;
    return hResponse->response.getSerialized();
}

// ============================================================
// response (error)
// ============================================================

spore_response_error_t* spore_response_error_create()
{
    return new spore_response_error_t{};
}

spore_response_error_t* spore_response_error_create_from_raw(const char* pRaw, size_t sz)
{
    if (!pRaw)
        return nullptr;
    auto* p = spore_parser_create();
    auto* m = spore_message_create();
    parseRaw(p, pRaw, sz, m);
    spore_response_error_t* result = nullptr;
    if (spore_parser_get_type(p) == SPORE_PARSER_TYPE_RESPONSE)
    {
        result = new spore_response_error_t{};
        populateMessage(result->error, m, true);
        result->error.setRaw(std::string_view(pRaw, sz));
        // extract code and what from the args map into dedicated fields
        size_t n = 0;
        const spore_arg_t* args = spore_message_get_args(m, &n);
        for (size_t i = 0; i < n; ++i)
        {
            if (std::string_view(args[i].pKey) == "code")
                result->error.setCode(args[i].pValue);
            else if (std::string_view(args[i].pKey) == "what")
                result->error.setWhat(args[i].pValue);
        }
    }
    spore_message_destroy(m);
    spore_parser_destroy(p);
    return result;
}

void spore_response_error_destroy(spore_response_error_t* hError)
{
    delete hError;
}

void spore_response_error_set_handle(spore_response_error_t* hError, const char* pHandle)
{
    if (!hError || !pHandle)
        return;
    hError->error.setHandle(pHandle);
}

void spore_response_error_set_command(spore_response_error_t* hError, const char* pCommand)
{
    if (!hError || !pCommand)
        return;
    hError->error.setCommand(pCommand);
}

void spore_response_error_set_code(spore_response_error_t* hError, const char* pCode)
{
    if (!hError || !pCode)
        return;
    hError->error.setCode(pCode);
}

void spore_response_error_set_what(spore_response_error_t* hError, const char* pWhat)
{
    if (!hError || !pWhat)
        return;
    hError->error.setWhat(pWhat);
}

void spore_response_error_add_arg(spore_response_error_t* hError,
                                  const char* pKey,
                                  const char* pValue)
{
    if (!hError || !pKey || !pValue)
        return;
    hError->error.addArg(pKey, pValue);
}

void spore_response_error_add_flag(spore_response_error_t* hError, const char* pFlag)
{
    if (!hError || !pFlag)
        return;
    hError->error.addFlag(pFlag);
}

const char* spore_response_error_get_command(const spore_response_error_t* hError)
{
    if (!hError)
        return nullptr;
    return sv(hError->error.getCommand());
}

const char* spore_response_error_get_handle(const spore_response_error_t* hError)
{
    if (!hError)
        return nullptr;
    return sv(hError->error.getHandle());
}

const char* spore_response_error_get_code(const spore_response_error_t* hError)
{
    if (!hError)
        return nullptr;
    return sv(hError->error.getCode());
}

const char* spore_response_error_get_what(const spore_response_error_t* hError)
{
    if (!hError)
        return nullptr;
    return sv(hError->error.getWhat());
}

const char* spore_response_error_get_arg(const spore_response_error_t* hError, const char* pKey)
{
    if (!hError || !pKey)
        return nullptr;
    return sv(hError->error.getArg(pKey));
}

bool spore_response_error_has_flag(const spore_response_error_t* hError, const char* pFlag)
{
    return hError && pFlag && hError->error.hasFlag(pFlag);
}

void spore_response_error_send(spore_client_t* hClient, const spore_response_error_t* hError)
{
    if (!hClient)
        return;
    hClient->client.sendResponseError(hError);
}

void spore_response_error_serialize(spore_response_error_t* hError)
{
    if (!hError)
        return;
    hError->error.serialize();
}

const char* spore_response_error_get_serialized(const spore_response_error_t* hError)
{
    if (!hError)
        return nullptr;
    return hError->error.getSerialized();
}

// ============================================================
// witness
// ============================================================

spore_witness_t* spore_witness_create()
{
    return new spore_witness_t{};
}

spore_witness_t* spore_witness_create_from_raw(const char* pRaw, size_t sz)
{
    if (!pRaw)
        return nullptr;
    auto* p = spore_parser_create();
    auto* m = spore_message_create();
    parseRaw(p, pRaw, sz, m);
    spore_witness_t* result = nullptr;
    if (spore_parser_get_type(p) == SPORE_PARSER_TYPE_WITNESS)
    {
        result = new spore_witness_t{};
        result->witness.setRaw(std::string_view(pRaw, sz));
        // body is a named arg; remaining args/flags go through populateMessage
        size_t n = 0;
        const spore_arg_t* args = spore_message_get_args(m, &n);
        for (size_t i = 0; i < n; ++i)
        {
            if (std::string_view(args[i].pKey) == "body")
                result->witness.setBody(args[i].pValue);
            else
                result->witness.addArg(args[i].pKey, args[i].pValue);
        }
        size_t nf = 0;
        const char** flags = spore_message_get_flags(m, &nf);
        for (size_t i = 0; i < nf; ++i) result->witness.addFlag(flags[i]);
    }
    spore_message_destroy(m);
    spore_parser_destroy(p);
    return result;
}

void spore_witness_destroy(spore_witness_t* hWitness)
{
    delete hWitness;
}

void spore_witness_set_body(spore_witness_t* hWitness, const char* pBody)
{
    if (!hWitness || !pBody)
        return;
    hWitness->witness.setBody(pBody);
}

void spore_witness_add_arg(spore_witness_t* hWitness, const char* pKey, const char* pValue)
{
    if (!hWitness || !pKey || !pValue)
        return;
    hWitness->witness.addArg(pKey, pValue);
}

void spore_witness_add_flag(spore_witness_t* hWitness, const char* pFlag)
{
    if (!hWitness || !pFlag)
        return;
    hWitness->witness.addFlag(pFlag);
}

const char* spore_witness_get_body(const spore_witness_t* hWitness)
{
    if (!hWitness)
        return nullptr;
    return sv(hWitness->witness.getBody());
}

const char* spore_witness_get_arg(const spore_witness_t* hWitness, const char* pKey)
{
    if (!hWitness || !pKey)
        return nullptr;
    return sv(hWitness->witness.getArg(pKey));
}

bool spore_witness_has_flag(const spore_witness_t* hWitness, const char* pFlag)
{
    return hWitness && pFlag && hWitness->witness.hasFlag(pFlag);
}

void spore_witness_send(spore_client_t* hClient, const spore_witness_t* hWitness)
{
    if (!hClient)
        return;
    hClient->client.sendWitness(hWitness);
}

void spore_witness_serialize(spore_witness_t* hWitness)
{
    if (!hWitness)
        return;
    hWitness->witness.serialize();
}

const char* spore_witness_get_serialized(const spore_witness_t* hWitness)
{
    if (!hWitness)
        return nullptr;
    return hWitness->witness.getSerialized();
}

// ============================================================
// publish
// ============================================================

spore_publish_t* spore_publish_create()
{
    return new spore_publish_t{};
}

spore_publish_t* spore_publish_create_from_raw(const char* pRaw, size_t sz)
{
    if (!pRaw)
        return nullptr;
    auto* p = spore_parser_create();
    auto* m = spore_message_create();
    parseRaw(p, pRaw, sz, m);
    spore_publish_t* result = nullptr;
    if (spore_parser_get_type(p) == SPORE_PARSER_TYPE_PUBLISH)
    {
        result = new spore_publish_t{};
        result->publish.setRaw(std::string_view(pRaw, sz));
        // capability is the topic for publish messages
        if (const char* cap = spore_message_get_capability(m))
            result->publish.setTopic(cap);
        size_t n = 0;
        const spore_arg_t* args = spore_message_get_args(m, &n);
        for (size_t i = 0; i < n; ++i) result->publish.addArg(args[i].pKey, args[i].pValue);
        size_t nf = 0;
        const char** flags = spore_message_get_flags(m, &nf);
        for (size_t i = 0; i < nf; ++i) result->publish.addFlag(flags[i]);
    }
    spore_message_destroy(m);
    spore_parser_destroy(p);
    return result;
}

void spore_publish_destroy(spore_publish_t* hPublish)
{
    delete hPublish;
}

void spore_publish_set_topic(spore_publish_t* hPublish, const char* pTopic)
{
    if (!hPublish || !pTopic)
        return;
    hPublish->publish.setTopic(pTopic);
}

void spore_publish_add_arg(spore_publish_t* hPublish, const char* pKey, const char* pValue)
{
    if (!hPublish || !pKey || !pValue)
        return;
    hPublish->publish.addArg(pKey, pValue);
}

void spore_publish_add_flag(spore_publish_t* hPublish, const char* pFlag)
{
    if (!hPublish || !pFlag)
        return;
    hPublish->publish.addFlag(pFlag);
}

const char* spore_publish_get_topic(const spore_publish_t* hPublish)
{
    if (!hPublish)
        return nullptr;
    return sv(hPublish->publish.getTopic());
}

const char* spore_publish_get_arg(const spore_publish_t* hPublish, const char* pKey)
{
    if (!hPublish || !pKey)
        return nullptr;
    return sv(hPublish->publish.getArg(pKey));
}

bool spore_publish_has_flag(const spore_publish_t* hPublish, const char* pFlag)
{
    return hPublish && pFlag && hPublish->publish.hasFlag(pFlag);
}

void spore_publish_send(spore_client_t* hClient, const spore_publish_t* hPublish)
{
    if (!hClient)
        return;
    hClient->client.sendPublish(hPublish);
}

void spore_publish_serialize(spore_publish_t* hPublish)
{
    if (!hPublish)
        return;
    hPublish->publish.serialize();
}

const char* spore_publish_get_serialized(const spore_publish_t* hPublish)
{
    if (!hPublish)
        return nullptr;
    return hPublish->publish.getSerialized();
}
