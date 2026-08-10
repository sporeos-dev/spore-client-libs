#include "publish.h"
#include <string>

namespace spore
{
    void Publish::serialize() const
    {
        std::string s = "publish";
        if (!topic.empty())
            s += " " + topic;
        for (const auto& [k, v] : args) s += " " + k + "=" + v;
        for (const auto& f : flags) s += " " + f;
        s += "\n";
        serialized = std::move(s);
    }

    std::string_view Publish::getTopic() const
    {
        return topic;
    }
    void Publish::setTopic(std::string_view v)
    {
        topic = v;
    }

}  // namespace spore
