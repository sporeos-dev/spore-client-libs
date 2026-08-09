#pragma once

#include <map>
#include <set>
#include <string>
#include <string_view>

namespace spore
{
    // Base for all Spore message types. Holds the common command, handle, args,
    // and flags that every message type carries.
    class Message
    {
    public:
        Message()          = default;
        virtual ~Message() = default;

        std::string_view getCommand() const;
        std::string_view getHandle() const;
        // Returns "" if the key is absent.
        std::string_view getArg(std::string_view key) const;
        bool hasFlag(std::string_view flag) const;

        void setCommand(std::string_view command);
        void setHandle(std::string_view handle);
        void addArg(std::string_view key, std::string_view value);
        void addFlag(std::string_view flag);

    protected:
        std::string                        command;
        std::string                        handle;
        std::map<std::string, std::string> args;
        std::set<std::string>              flags;
    };

}  // namespace spore
