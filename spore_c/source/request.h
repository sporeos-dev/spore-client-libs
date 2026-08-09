#pragma once

#include "message.h"

namespace spore
{
    class Request : public Message
    {
        // command = capability name, handle = ~handle assigned by caller
    };

}  // namespace spore
