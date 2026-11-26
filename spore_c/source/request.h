/*
 * Copyright 2026 Matt Harrison
 * SPDX-License-Identifier: Apache-2.0
 */

#pragma once

#include "message.h"

namespace spore
{
    class Request : public Message
    {
    public:
        void serialize() const;
        // command = capability name, handle = ~handle assigned by caller

        // raw response line, filled in by Client::sendAndWait
        void setResponse(std::string value) const { response = std::move(value); }
        const char* getResponse() const { return response.empty() ? nullptr : response.c_str(); }

    private:
        mutable std::string response;
    };

}  // namespace spore
