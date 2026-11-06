// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: Apache-2.0

#include "spore_c.h"
#include <gtest/gtest.h>
#include <string>

TEST(MessageAccessors, CountsCapacityAndBorrowedRaw)
{
    const std::string raw = "Example.run ~h z=last a=first beta alpha\r\n";
    auto* request = spore_request_create_from_raw(raw.data(), raw.size());
    ASSERT_NE(request, nullptr);
    EXPECT_EQ(spore_request_get_arg_ids(request, nullptr, 0), 2u);

    const char* ids[1] = {};
    EXPECT_EQ(spore_request_get_arg_ids(request, ids, 1), 2u);
    ASSERT_NE(ids[0], nullptr);
    EXPECT_STREQ(ids[0], "a");

    size_t size = 0;
    const char* original = spore_request_get_raw(request, &size);
    ASSERT_NE(original, nullptr);
    EXPECT_EQ(std::string(original, size), raw);
    spore_request_add_arg(request, "new", "value");
    spore_request_serialize(request);
    original = spore_request_get_raw(request, &size);
    EXPECT_EQ(std::string(original, size), raw);
    spore_request_destroy(request);
}

TEST(MessageAccessors, EmptyAndNullMessages)
{
    auto* request = spore_request_create();
    ASSERT_NE(request, nullptr);
    EXPECT_EQ(spore_request_get_arg_ids(request, nullptr, 0), 0u);
    EXPECT_EQ(spore_request_get_flags(request, nullptr, 0), 0u);
    size_t size = 99;
    spore_request_get_raw(request, &size);
    EXPECT_EQ(size, 0u);
    spore_request_get_raw(nullptr, &size);
    EXPECT_EQ(size, 0u);
    spore_request_destroy(request);
}
