// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: Apache-2.0

#include "spore_c.h"
#include <gtest/gtest.h>

TEST(WitnessSerialization, QuotesSpacedBody)
{
    auto* witness = spore_witness_create();
    ASSERT_NE(witness, nullptr);
    spore_witness_set_body(witness, "Finished test main.PASS_empty");
    spore_witness_serialize(witness);

    EXPECT_STREQ(spore_witness_get_serialized(witness),
                 "witness body=\"Finished test main.PASS_empty\"\n");

    spore_witness_destroy(witness);
}
