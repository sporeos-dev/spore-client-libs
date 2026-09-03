#pragma once

#include "spore_parser.h"
#include <vector>
#include <string>
#include <string_view>

namespace spore
{
    class message
    {
    private:
        std::string capability;
        std::string handle;
        std::vector<std::pair<std::string, std::string>> args;
        std::vector<spore_arg_t> argsC;
        std::vector<std::string> flags;
        std::vector<const char*> flagsC;

    public:
        message() = default;
        ~message() = default;

        // getters
        std::string_view getCapability() const;
        std::string_view getHandle() const;
        std::vector<spore_arg_t>& getArgs();
        std::vector<const char*>& getFlags();

        // setters
        void setCapability(std::string_view cap);
        void setHandle(std::string_view handle);
        void addArg(std::string_view key, std::string_view value);
        void addFlag(std::string_view flag);
    };
}  // namespace spore
