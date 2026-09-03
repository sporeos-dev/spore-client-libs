#include "response.h"
#include <string>

namespace spore
{
    void Response::serialize() const
    {
        std::string s;
        if (!handle.empty())
            s = "~" + handle + ":" + command;
        else
            s = command;
        s += " ok";
        for (const auto& f : flags)
        {
            if (f != "ok")
                s += " " + f;
        }
        for (const auto& [k, v] : args) serializeArg(s, k, v);
        s += "\n";
        serialized = std::move(s);
    }

    void ResponseError::serialize() const
    {
        std::string s;
        if (!handle.empty())
            s = "~" + handle + ":" + command;
        else
            s = command;
        s += " error";
        if (!code.empty())
            s += " code=" + code;
        if (!what.empty())
            s += " what=" + what;
        for (const auto& [k, v] : args) serializeArg(s, k, v);
        for (const auto& f : flags) s += " " + f;
        s += " node_error\n";
        serialized = std::move(s);
    }

    std::string_view ResponseError::getCode() const
    {
        return code;
    }
    std::string_view ResponseError::getWhat() const
    {
        return what;
    }
    void ResponseError::setCode(std::string_view v)
    {
        code = v;
    }
    void ResponseError::setWhat(std::string_view v)
    {
        what = v;
    }

}  // namespace spore
