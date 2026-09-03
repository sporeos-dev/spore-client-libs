#include "witness.h"
#include <string>

namespace spore
{
    void Witness::serialize() const
    {
        std::string s = "witness";
        if (!body.empty())
            s += " body=" + body;
        for (const auto& [k, v] : args) serializeArg(s, k, v);
        for (const auto& f : flags) s += " " + f;
        s += "\n";
        serialized = std::move(s);
    }

    std::string_view Witness::getBody() const
    {
        return body;
    }
    void Witness::setBody(std::string_view v)
    {
        body = v;
    }

}  // namespace spore
