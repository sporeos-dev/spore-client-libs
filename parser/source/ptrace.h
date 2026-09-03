#pragma once

#include "defs.h"
#include <string_view>
#include <atomic>

class spore_message_t;
namespace spore
{
    class parser;
}

class Ptrace
{
    static std::atomic<bool> on;

public:
    Ptrace() = default;
    ~Ptrace() = default;

    static void enable();
    static void msg(std::string_view msg, size_t indent);
    static void string(std::string_view key, std::string_view str, size_t indent);
    static void list(std::string_view el, size_t indent);
    static void num(std::string_view key, size_t num, size_t indent);
    static void arg(std::string_view key, std::string_view value, size_t indent);
    static void type(spore_parser_type_t type, size_t indent);
    static void boolean(std::string_view key, bool value, size_t indent);
};
