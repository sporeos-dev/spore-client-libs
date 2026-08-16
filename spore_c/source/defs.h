#pragma once

#include "client.h"
#include "request.h"
#include "response.h"
#include "witness.h"
#include "publish.h"

// Full definitions for all opaque types forward-declared in spore_c.h.

struct spore_client_t
{
    spore::Client client;
};

// Heap-allocated token returned by register_*_handler; identifies one registration.
struct spore_handler_t
{
    enum class type_t
    {
        request,
        response,
        witness,
        publish,
        parse_error
    } type;
    int id;
    union
    {
        spore_request_fn onRequest;
        spore_response_fn onResponse;
        spore_witness_fn onWitness;
        spore_publish_fn onPublish;
        spore_parse_error_fn onParseError;
    };
};

struct spore_request_t
{
    spore::Request request;
};

struct spore_response_t
{
    spore::Response response;
};

struct spore_response_error_t
{
    spore::ResponseError error;
};

struct spore_witness_t
{
    spore::Witness witness;
};

struct spore_publish_t
{
    spore::Publish publish;
};
