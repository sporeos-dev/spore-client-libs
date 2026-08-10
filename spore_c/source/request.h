#pragma once

#include "message.h"

namespace spore
{
    class Request : public Message
    {
    public:
        void serialize() const;
        // command = capability name, handle = ~handle assigned by caller
    };

}  // namespace spore
