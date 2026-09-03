#pragma once

#include <atomic>
#include <string_view>

class Trace
{
    static std::atomic<bool> on;

public:
    Trace() = default;
    ~Trace() = default;

    static void enable();
    static void msg(std::string_view msg);
    static void string(std::string_view key, std::string_view str);
};
