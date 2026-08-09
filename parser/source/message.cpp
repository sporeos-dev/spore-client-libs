
#include "message.h"
#include "spore_parser.h"

namespace spore
{
    std::string_view message::getCapability() const
    {
        return capability;
    }

    std::string_view message::getHandle() const
    {
        return handle;
    }

    std::vector<spore_arg_t>& message::getArgs()
    {
        argsC.clear();
        for (const auto& a : args) argsC.push_back({ a.first.c_str(), a.second.c_str() });
        return argsC;
    }

    std::vector<const char*>& message::getFlags()
    {
        flagsC.clear();
        for (const auto& f : flags) flagsC.push_back(f.c_str());
        return flagsC;
    }

    std::string_view message::getInline() const
    {
        return inlineData;
    }

    void message::setCapability(std::string_view cap)
    {
        capability = cap;
    }

    void message::setHandle(std::string_view h)
    {
        handle = h;
    }

    void message::addArg(std::string_view key, std::string_view value)
    {
        args.emplace_back(key, value);
    }

    void message::addFlag(std::string_view flag)
    {
        flags.emplace_back(flag);
    }

    void message::setInline(std::string_view data)
    {
        inlineData = data;
    }
}  // namespace spore