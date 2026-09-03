#include "message.h"

namespace spore
{
    std::string_view Message::getCommand() const
    {
        return command;
    }
    std::string_view Message::getHandle() const
    {
        return handle;
    }

    std::string_view Message::getArg(std::string_view key) const
    {
        auto it = args.find(std::string(key));
        return it != args.end() ? std::string_view(it->second) : std::string_view{};
    }

    bool Message::hasFlag(std::string_view flag) const
    {
        return flags.count(std::string(flag)) > 0;
    }

    void Message::setCommand(std::string_view v)
    {
        command = v;
    }
    void Message::setHandle(std::string_view v)
    {
        handle = v;
    }

    void Message::addArg(std::string_view key, std::string_view value)
    {
        args[std::string(key)] = std::string(value);
    }

    void Message::addFlag(std::string_view flag)
    {
        flags.insert(std::string(flag));
    }

    void Message::serializeArg(std::string& s, const std::string& k, const std::string& v) const
    {
        if (v.find('\n') != std::string::npos)
        {
            s += " " + k + "=<<\n" + v + "\n>>";
        }
        else if (v.find('"') != std::string::npos)
        {
            s += " " + k + "='" + v + "'";
        }
        else if (v.find_first_of(" \t") != std::string::npos)
        {
            s += " " + k + "=\"" + v + "\"";
        }
        else
        {
            s += " " + k + "=" + v;
        }
    }
}  // namespace spore
