/*
 * Copyright 2026 Matt Harrison
 * SPDX-License-Identifier: Apache-2.0
 */

#pragma once

#include "spore_c.h"
#include <mutex>
#include <string>
#include <string_view>

namespace spore
{
    class Stream
    {
    public:
        Stream(std::string_view handle, std::string_view field, std::string_view text);
        Stream(spore_client_t* client, std::string_view handle, std::string_view command,
               std::string_view field);

        std::string_view getHandle() const;
        std::string_view getField() const;
        std::string_view getText() const;

        spore_stream_result_t start();
        spore_stream_result_t chunk(spore_string_view_t text);
        spore_stream_result_t close();
        spore_stream_result_t fail(spore_string_view_t code, spore_string_view_t what);

    private:
        spore_stream_result_t finish(const char* response);

        spore_client_t* client = nullptr;
        std::string handle;
        std::string command;
        std::string field;
        std::string text;
        std::mutex mu;
        bool ended = false;
        spore_stream_result_t sendError = SPORE_STREAM_OK;
    };
}  // namespace spore
