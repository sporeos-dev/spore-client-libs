#pragma once

#include <gtest/gtest.h>
#include "spore_parser.h"

class Fixture : public ::testing::Test
{
protected:
    virtual ~Fixture() = default;

    spore_parser_t* parser = nullptr;
    spore_message_t* message = nullptr;
    std::map<std::string, std::string> args;
    std::vector<std::string> flags;
    const std::string null = "n/a";

    void SetUp() override
    {
        parser = spore_parser_create();
        message = spore_message_create();
    }

    void TearDown() override
    {
        spore_parser_destroy(parser);
        spore_message_destroy(message);

        args.clear();
        flags.clear();
    }

    void parse(std::string raw)
    {
        spore_parse(parser, raw.c_str(), raw.size(), message);
        makeArgs();
        makeFlags();
    }

    void makeArgs()
    {
        size_t num = 0;
        auto rawArgs = spore_message_get_args(message, &num);
        args.clear();
        for (size_t i = 0; i < num; ++i)
        {
            args[rawArgs[i].pKey] = rawArgs[i].pValue;
        }
    }

    void makeFlags()
    {
        size_t num = 0;
        auto rawFlags = spore_message_get_flags(message, &num);
        flags.clear();
        for (size_t i = 0; i < num; ++i)
        {
            flags.push_back(rawFlags[i]);
        }
    }

    std::string_view arg(std::string_view key)
    {
        if (args.find(key.data()) == args.end())
            return null;
        return args.at(key.data());
    }

    bool flag(const std::string& flag)
    {
        return std::find(flags.begin(), flags.end(), flag) != flags.end();
    }
};
