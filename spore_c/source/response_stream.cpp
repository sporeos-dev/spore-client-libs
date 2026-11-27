// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: Apache-2.0

#include "defs.h"
#include <cctype>
#include <mutex>
#include <set>
#include <string>
#include <string_view>

namespace
{
    bool view(spore_string_view_t input, std::string_view& output)
    {
        if (!input.data && input.size != 0)
            return false;
        output = input.data ? std::string_view(input.data, input.size) : std::string_view{};
        return output.find('\0') == std::string_view::npos;
    }

    bool validToken(std::string_view value)
    {
        if (value.empty() || value.find_first_of("\0:<>=\"'[{", 0, 9) != std::string_view::npos)
            return false;
        for (unsigned char c : value)
            if (c < 128 && std::isspace(c))
                return false;
        // Match Unicode whitespace accepted by the Go API's previous validation.
        static constexpr std::string_view spaces[] = {
            "\xc2\x85", "\xc2\xa0", "\xe1\x9a\x80",
            "\xe2\x80\x80", "\xe2\x80\x81", "\xe2\x80\x82", "\xe2\x80\x83",
            "\xe2\x80\x84", "\xe2\x80\x85", "\xe2\x80\x86", "\xe2\x80\x87",
            "\xe2\x80\x88", "\xe2\x80\x89", "\xe2\x80\x8a", "\xe2\x80\xa8",
            "\xe2\x80\xa9", "\xe2\x80\xaf", "\xe2\x81\x9f", "\xe3\x80\x80"
        };
        for (auto space : spaces)
            if (value.find(space) != std::string_view::npos)
                return false;
        return true;
    }

    spore_stream_result_t prefix(spore_string_view_t handle,
                                spore_string_view_t token,
                                std::string& frame)
    {
        std::string_view h, t;
        if (!view(handle, h) || !view(token, t) || !validToken(h) || !validToken(t))
            return SPORE_STREAM_INVALID_ARGUMENT;
        frame = "~" + std::string(h) + ":" + std::string(t);
        return SPORE_STREAM_OK;
    }

    spore_stream_result_t send(spore_client_t* client, const std::string& frame)
    {
        return client ? client->client.sendRawChecked(frame.data(), frame.size())
                      : SPORE_STREAM_INVALID_ARGUMENT;
    }

    spore_string_view_t toView(std::string_view s)
    {
        return {s.data(), s.size()};
    }
}

struct spore_response_stream_t
{
    spore_client_t* client;
    std::string handle;
    std::string command;
    std::string field;
    std::mutex mu;
    bool ended = false;
    spore_stream_result_t sendError = SPORE_STREAM_OK;
};

spore_stream_result_t spore_stream_send_start(spore_client_t* client,
                                              spore_string_view_t handle,
                                              spore_string_view_t command,
                                              const spore_string_view_t* fields,
                                              size_t count)
{
    std::string frame;
    if (!client || prefix(handle, command, frame) != SPORE_STREAM_OK || !fields || count == 0)
        return SPORE_STREAM_INVALID_ARGUMENT;
    std::set<std::string> seen;
    for (size_t i = 0; i < count; ++i)
    {
        std::string_view field;
        if (!view(fields[i], field) || !validToken(field) || !seen.insert(std::string(field)).second)
            return SPORE_STREAM_INVALID_ARGUMENT;
        frame += " " + std::string(field) + "=<<";
    }
    return send(client, frame);
}

spore_stream_result_t spore_stream_send_chunk(spore_client_t* client,
                                              spore_string_view_t handle,
                                              spore_string_view_t field,
                                              spore_string_view_t text)
{
    std::string frame;
    std::string_view contents;
    if (!client || prefix(handle, field, frame) != SPORE_STREAM_OK || !view(text, contents))
        return SPORE_STREAM_INVALID_ARGUMENT;
    frame += "<< ";
    for (char c : contents)
    {
        switch (c)
        {
            case '\\': frame += "\\\\"; break;
            case '\n': frame += "\\n"; break;
            case '\r': frame += "\\r"; break;
            case '\t': frame += "\\t"; break;
            default: frame += c; break;
        }
    }
    return send(client, frame);
}

spore_stream_result_t spore_stream_send_close(spore_client_t* client,
                                              spore_string_view_t handle,
                                              spore_string_view_t field)
{
    std::string frame;
    if (!client || prefix(handle, field, frame) != SPORE_STREAM_OK)
        return SPORE_STREAM_INVALID_ARGUMENT;
    return send(client, frame + ">>");
}

spore_stream_result_t spore_response_stream_open(spore_client_t* client,
                                                  const spore_request_t* request,
                                                  spore_string_view_t field,
                                                  spore_response_stream_t** out)
{
    if (!out)
        return SPORE_STREAM_INVALID_ARGUMENT;
    *out = nullptr;
    std::string_view fieldText;
    if (!client || !request || !view(field, fieldText) || !validToken(fieldText))
        return SPORE_STREAM_INVALID_ARGUMENT;
    auto* stream = new spore_response_stream_t{
        client, std::string(request->request.getHandle()), std::string(request->request.getCommand()),
        std::string(fieldText)
    };
    auto result = spore_stream_send_start(client, toView(stream->handle), toView(stream->command),
                                          &field, 1);
    if (result != SPORE_STREAM_OK)
    {
        delete stream;
        return result;
    }
    *out = stream;
    return SPORE_STREAM_OK;
}

spore_stream_result_t spore_response_stream_chunk(spore_response_stream_t* stream,
                                                   spore_string_view_t text)
{
    if (!stream)
        return SPORE_STREAM_INVALID_ARGUMENT;
    std::lock_guard<std::mutex> lk(stream->mu);
    if (stream->sendError != SPORE_STREAM_OK)
        return stream->sendError;
    if (stream->ended)
        return SPORE_STREAM_CLOSED;
    auto result = spore_stream_send_chunk(stream->client, toView(stream->handle),
                                          toView(stream->field), text);
    if (result == SPORE_STREAM_SEND_FAILED || result == SPORE_STREAM_NOT_CONNECTED)
        stream->sendError = result;
    return result;
}

namespace
{
    spore_stream_result_t finish(spore_response_stream_t* stream, const char* response)
    {
        auto result = spore_stream_send_close(stream->client, toView(stream->handle),
                                               toView(stream->field));
        if (result == SPORE_STREAM_OK)
            result = send(stream->client, response);
        if (result == SPORE_STREAM_OK)
            stream->ended = true;
        else
            stream->sendError = result;
        return result;
    }
}

spore_stream_result_t spore_response_stream_close(spore_response_stream_t* stream)
{
    if (!stream)
        return SPORE_STREAM_INVALID_ARGUMENT;
    std::lock_guard<std::mutex> lk(stream->mu);
    if (stream->sendError != SPORE_STREAM_OK)
        return stream->sendError;
    if (stream->ended)
        return SPORE_STREAM_CLOSED;
    spore::Response response;
    response.setHandle(stream->handle);
    response.setCommand(stream->command);
    response.serialize();
    return finish(stream, response.getSerialized());
}

spore_stream_result_t spore_response_stream_fail(spore_response_stream_t* stream,
                                                  spore_string_view_t code,
                                                  spore_string_view_t what)
{
    if (!stream)
        return SPORE_STREAM_INVALID_ARGUMENT;
    std::lock_guard<std::mutex> lk(stream->mu);
    if (stream->sendError != SPORE_STREAM_OK)
        return stream->sendError;
    if (stream->ended)
        return SPORE_STREAM_CLOSED;
    std::string_view codeText, whatText;
    if (!view(code, codeText) || !validToken(codeText) || !view(what, whatText) || whatText.empty())
        return SPORE_STREAM_INVALID_ARGUMENT;
    spore::ResponseError response;
    response.setHandle(stream->handle);
    response.setCommand(stream->command);
    response.setCode(codeText);
    response.setWhat(whatText);
    response.serialize();
    return finish(stream, response.getSerialized());
}

void spore_response_stream_destroy(spore_response_stream_t* stream)
{
    delete stream;
}
