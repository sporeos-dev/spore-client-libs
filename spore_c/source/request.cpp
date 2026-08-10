#include "request.h"
#include <string>

namespace spore
{
    void Request::serialize() const
    {
        std::string s = command;
        if (!handle.empty())
            s += " ~" + handle;
        for (const auto& [k, v] : args) s += " " + k + "=" + v;
        for (const auto& f : flags) s += " " + f;
        s += "\n";
        serialized = std::move(s);
    }

}  // namespace spore
