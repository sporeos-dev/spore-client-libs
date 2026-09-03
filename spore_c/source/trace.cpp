
#include "trace.h"
#include <iostream>

std::atomic<bool> Trace::on = false;

void Trace::enable()
{
    on = true;
}

void Trace::msg(std::string_view msg)
{
    if (!on)
        return;

    std::cout << "[CLIENT] " << msg << std::endl;
}

void Trace::string(std::string_view key, std::string_view str)
{
    if (!on)
        return;

    std::cout << "[CLIENT] " << key << ": " << str << std::endl;
}
