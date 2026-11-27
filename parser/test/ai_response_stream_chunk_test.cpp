// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: Apache-2.0

#include "spore_parser.h"
#include <gtest/gtest.h>
#include <string>

TEST(ResponseStreamChunk, ParsesPayloadWithoutTokenizingIt)
{
    auto* parser = spore_parser_create();
    auto* message = spore_message_create();
    const std::string raw = "~h:output<<  \"literal\" = [] \\\\x\\q\\n\\t ";
    spore_parse(parser, raw.data(), raw.size(), message);
    spore_validate(parser, message);

    ASSERT_FALSE(spore_parser_has_error(parser));
    EXPECT_EQ(spore_parser_get_type(parser), SPORE_PARSER_TYPE_STREAM);
    EXPECT_STREQ(spore_message_get_handle(message), "h");
    EXPECT_STREQ(spore_message_get_stream_field(message), "output");
    size_t size = 0;
    const char* text = spore_message_get_stream_text(message, &size);
    ASSERT_NE(text, nullptr);
    EXPECT_EQ(std::string(text, size), " \"literal\" = [] \\x\\q\n\t ");

    const std::string empty = "~h:output<< ";
    spore_parse(parser, empty.data(), empty.size(), message);
    spore_validate(parser, message);
    ASSERT_FALSE(spore_parser_has_error(parser));
    text = spore_message_get_stream_text(message, &size);
    ASSERT_NE(text, nullptr);
    EXPECT_EQ(size, 0u);

    spore_message_destroy(message);
    spore_parser_destroy(parser);
}

TEST(ResponseStreamChunk, RejectsMalformedFramesAndPreservesOrdinaryResponses)
{
    auto* parser = spore_parser_create();
    auto* message = spore_message_create();
    const std::string malformed[] = {
        "~:output<< text", "~h:<< text", "~h:output<<missing-space",
        std::string("~h:output<< bad\0data", 20)
    };
    for (const auto& raw : malformed)
    {
        spore_parse(parser, raw.data(), raw.size(), message);
        spore_validate(parser, message);
        EXPECT_EQ(spore_parser_get_type(parser), SPORE_PARSER_TYPE_STREAM);
        EXPECT_STREQ(spore_parser_get_error_code(parser), "Malformed");
    }

    const std::string response = "~h:command ok output=\"literal<<text>>\"";
    spore_parse(parser, response.data(), response.size(), message);
    spore_validate(parser, message);
    EXPECT_FALSE(spore_parser_has_error(parser));
    EXPECT_EQ(spore_parser_get_type(parser), SPORE_PARSER_TYPE_RESPONSE);

    spore_message_destroy(message);
    spore_parser_destroy(parser);
}
