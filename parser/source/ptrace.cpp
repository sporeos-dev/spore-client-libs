
#include "ptrace.h"
#include <iostream>
#include <string>

namespace
{
    inline std::string indent(size_t level)
    {
        switch (level)
        {
            case 0:
                return "";
            case 1:
                return "  ";
            case 2:
                return "    ";
            case 3:
                return "      ";
            default:
                return std::string(level * 2, ' ');
        }
    }
}  // namespace

std::atomic<bool> Ptrace::on = false;

void Ptrace::enable()
{
    on = true;
}

void Ptrace::msg(std::string_view msg, size_t ind)
{
    if (!on)
        return;

    std::cout << "[PARSER] " << indent(ind) << msg << std::endl;
}

void Ptrace::string(std::string_view key, std::string_view str, size_t ind)
{
    if (!on)
        return;

    std::cout << "[PARSER] " << indent(ind) << key << ": " << str << std::endl;
}

void Ptrace::list(std::string_view el, size_t ind)
{
    if (!on)
        return;

    std::cout << "[PARSER] " << indent(ind) << "- " << el << std::endl;
}

void Ptrace::num(std::string_view key, size_t num, size_t ind)
{
    if (!on)
        return;

    std::cout << "[PARSER] " << indent(ind) << key << ": " << num << std::endl;
}

void Ptrace::arg(std::string_view key, std::string_view value, size_t ind)
{
    if (!on)
        return;

    std::cout << "[PARSER] " << indent(ind) << "- " << key << ": " << value << std::endl;
}

void Ptrace::type(spore_parser_type_t type, size_t ind)
{
    if (!on)
        return;

    std::string_view typeStr;
    switch (type)
    {
        case SPORE_PARSER_TYPE_UNKNOWN:
            typeStr = "UNKNOWN";
            break;
        case SPORE_PARSER_TYPE_REQUEST:
            typeStr = "REQUEST";
            break;
        case SPORE_PARSER_TYPE_RESPONSE:
            typeStr = "RESPONSE";
            break;
        case SPORE_PARSER_TYPE_WITNESS:
            typeStr = "WITNESS";
            break;
        case SPORE_PARSER_TYPE_PUBLISH:
            typeStr = "PUBLISH";
            break;
    }

    std::cout << "[PARSER] " << indent(ind) << "Type: " << typeStr << std::endl;
}

void Ptrace::boolean(std::string_view key, bool value, size_t ind)
{
    if (!on)
        return;

    std::cout << "[PARSER] " << indent(ind) << key << ": " << (value ? "true" : "false")
              << std::endl;
}