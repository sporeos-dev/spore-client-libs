/*
 * Copyright 2026 Matt Harrison
 * SPDX-License-Identifier: Apache-2.0
 */

#pragma once

#include "message.h"

namespace spore
{
    // Witness messages use body as a standard arg; other args/flags are as-is.
    class Witness : public Message
    {
    public:
        void serialize() const;
        std::string_view getBody() const;
        void setBody(std::string_view body);

    private:
        std::string body;
    };

}  // namespace spore
