// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: Apache-2.0

#include "spore_c.h"

#include <gtest/gtest.h>

#if defined(_WIN32)
#include <windows.h>

#include <cstdlib>
#include <string>

namespace
{
    class CRunTimeEnvironmentGuard
    {
    public:
        explicit CRunTimeEnvironmentGuard(const char* name) : name(name)
        {
            char* value = nullptr;
            size_t length = 0;
            if (_dupenv_s(&value, &length, name) == 0 && value != nullptr)
            {
                value_ = value;
                std::free(value);
                existed = true;
            }
        }

        ~CRunTimeEnvironmentGuard()
        {
            _putenv_s(name, existed ? value_.c_str() : "");
        }

    private:
        const char* name;
        std::string value_;
        bool existed = false;
    };
}

TEST(WindowsSocketPath, UsesProgramDataDirectory)
{
    CRunTimeEnvironmentGuard programData("ProgramData");
    ASSERT_EQ(_putenv_s("ProgramData", R"(C:\spore-client-test-programdata)"), 0);

    const std::string expected =
        R"(failed to connect to daemon at C:\spore-client-test-programdata\spore-os\spore.sock)";

    spore_client_t* client = spore_client_create("test.node");
    ASSERT_NE(client, nullptr);

    spore_client_connect(client);

    ASSERT_TRUE(spore_client_has_error(client));
    EXPECT_STREQ(spore_client_get_error_what(client), expected.c_str());

    spore_client_destroy(client);
}
#endif
