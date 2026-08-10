#pragma once

#include "message.h"

namespace spore
{
    // An ok response from the hub.
    class Response : public Message
    {
    public:
        void serialize() const;
        // command = capability name, handle = ~handle from the original request
    };

    // An error response from the hub.
    class ResponseError : public Message
    {
    public:
        void serialize() const;
        std::string_view getCode() const;
        std::string_view getWhat() const;
        void setCode(std::string_view code);
        void setWhat(std::string_view what);

    private:
        std::string code;
        std::string what;
    };

}  // namespace spore
