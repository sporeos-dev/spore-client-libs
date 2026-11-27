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

namespace spore
{
    Stream::Stream(std::string_view handle, std::string_view field, std::string_view text)
        : handle(handle), field(field), text(text)
    {
    }

    Stream::Stream(spore_client_t* client, std::string_view handle, std::string_view command,
                   std::string_view field)
        : client(client), handle(handle), command(command), field(field)
    {
    }

    std::string_view Stream::getHandle() const
    {
        return handle;
    }

    std::string_view Stream::getField() const
    {
        return field;
    }

    std::string_view Stream::getText() const
    {
        return text;
    }

    spore_stream_result_t Stream::start()
    {
        auto f = toView(field);
        return spore_stream_send_start(client, toView(handle), toView(command), &f, 1);
    }

    spore_stream_result_t Stream::chunk(spore_string_view_t data)
    {
        std::lock_guard<std::mutex> lk(mu);
        if (sendError != SPORE_STREAM_OK)
            return sendError;
        if (ended)
            return SPORE_STREAM_CLOSED;
        std::string_view contents;
        if (!view(data, contents))
            return SPORE_STREAM_INVALID_ARGUMENT;
        auto result = spore_stream_send_chunk(client, toView(handle), toView(field), data);
        if (result == SPORE_STREAM_OK)
            text = contents;
        else if (result == SPORE_STREAM_SEND_FAILED || result == SPORE_STREAM_NOT_CONNECTED)
            sendError = result;
        return result;
    }

    spore_stream_result_t Stream::finish(const char* response)
    {
        auto result = spore_stream_send_close(client, toView(handle), toView(field));
        if (result == SPORE_STREAM_OK)
            result = send(client, response);
        if (result == SPORE_STREAM_OK)
            ended = true;
        else
            sendError = result;
        return result;
    }

    spore_stream_result_t Stream::close()
    {
        std::lock_guard<std::mutex> lk(mu);
        if (sendError != SPORE_STREAM_OK)
            return sendError;
        if (ended)
            return SPORE_STREAM_CLOSED;
        Response response;
        response.setHandle(handle);
        response.setCommand(command);
        response.serialize();
        return finish(response.getSerialized());
    }

    spore_stream_result_t Stream::fail(spore_string_view_t code, spore_string_view_t what)
    {
        std::lock_guard<std::mutex> lk(mu);
        if (sendError != SPORE_STREAM_OK)
            return sendError;
        if (ended)
            return SPORE_STREAM_CLOSED;
        std::string_view codeText, whatText;
        if (!view(code, codeText) || !validToken(codeText) || !view(what, whatText) ||
            whatText.empty())
            return SPORE_STREAM_INVALID_ARGUMENT;
        ResponseError response;
        response.setHandle(handle);
        response.setCommand(command);
        response.setCode(codeText);
        response.setWhat(whatText);
        response.serialize();
        return finish(response.getSerialized());
    }
}

spore_stream_result_t spore_stream_open(spore_client_t* client,
                                         const spore_request_t* request,
                                         spore_string_view_t field,
                                         spore_stream_t** out)
{
    if (!out)
        return SPORE_STREAM_INVALID_ARGUMENT;
    *out = nullptr;
    std::string_view fieldText;
    if (!client || !request || !view(field, fieldText) || !validToken(fieldText))
        return SPORE_STREAM_INVALID_ARGUMENT;
    auto* stream = new spore_stream_t{spore::Stream(
        client, request->request.getHandle(), request->request.getCommand(), fieldText)};
    auto result = stream->stream.start();
    if (result != SPORE_STREAM_OK)
    {
        delete stream;
        return result;
    }
    *out = stream;
    return SPORE_STREAM_OK;
}

spore_stream_result_t spore_stream_chunk(spore_stream_t* stream, spore_string_view_t text)
{
    return stream ? stream->stream.chunk(text) : SPORE_STREAM_INVALID_ARGUMENT;
}

spore_stream_result_t spore_stream_close(spore_stream_t* stream)
{
    return stream ? stream->stream.close() : SPORE_STREAM_INVALID_ARGUMENT;
}

spore_stream_result_t spore_stream_fail(spore_stream_t* stream,
                                         spore_string_view_t code,
                                         spore_string_view_t what)
{
    return stream ? stream->stream.fail(code, what) : SPORE_STREAM_INVALID_ARGUMENT;
}

void spore_stream_destroy(spore_stream_t* stream)
{
    delete stream;
}
