#pragma once

#include "message.h"

namespace spore
{
    class Publish : public Message
    {
    public:
        std::string_view getTopic() const;
        void setTopic(std::string_view topic);

    private:
        std::string topic;
    };

}  // namespace spore
