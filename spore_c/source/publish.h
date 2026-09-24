/*
 * Copyright 2026 Matt Harrison
 * SPDX-License-Identifier: Apache-2.0
 */

#pragma once

#include "message.h"

namespace spore
{
    class Publish : public Message
    {
    public:
        void serialize() const;
        std::string_view getTopic() const;
        void setTopic(std::string_view topic);

    private:
        std::string topic;
    };

}  // namespace spore
