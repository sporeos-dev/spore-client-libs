// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: Apache-2.0

#include "fixture.h"
#include "parser.h"
#include "spore_parser.h"

// Markers in ordinary tokens and quoted values remain literal.

class BlockMarkers
    : public Fixture
    , public spore::parser
{
protected:
    virtual ~BlockMarkers() = default;
    std::vector<token_t> tokens;
};

TEST_F(BlockMarkers, are_ordinary_characters_outside_quotes)
{
    tokenize("Some ~h body=a<<b>>c flag", tokens);
    ASSERT_EQ(tokens.size(), 4);
    EXPECT_EQ(tokens[2].type, token_t::type_t::ARG);
    EXPECT_STREQ(tokens[2].value.c_str(), "body=a<<b>>c");
    EXPECT_STREQ(tokens[3].value.c_str(), "flag");
}

TEST_F(BlockMarkers, bare_opener_does_not_swallow_the_rest_of_the_line)
{
    tokenize("~h:cmd ok value=<< other=x", tokens);
    ASSERT_EQ(tokens.size(), 4);
    EXPECT_STREQ(tokens[2].value.c_str(), "value=<<");
    EXPECT_STREQ(tokens[3].value.c_str(), "other=x");
}

TEST_F(BlockMarkers, are_literal_inside_quotes)
{
    tokenize("Some ~h body=\"<<x>>\" other='<<y>>'", tokens);
    ASSERT_EQ(tokens.size(), 4);
    EXPECT_STREQ(tokens[2].value.c_str(), "body=<<x>>");
    EXPECT_STREQ(tokens[3].value.c_str(), "other=<<y>>");
}

TEST_F(BlockMarkers, quoted_value_decodes_line_break_escapes)
{
    tokenize("Some ~h body=\"line1\\r\\nline2\\n\"", tokens);
    ASSERT_EQ(tokens.size(), 3);
    EXPECT_STREQ(tokens[2].value.c_str(), "body=line1\r\nline2\n");
}

TEST(BlockMarkersParse, value_keeps_markers_and_is_not_trimmed)
{
    spore_parser_t* parser = spore_parser_create();
    spore_message_t* msg = spore_message_create();
    const char* raw = "~h:cmd ok body=\"\\n<<x>>\\n\"";
    spore_parse(parser, raw, std::strlen(raw), msg);
    spore_validate(parser, msg);
    ASSERT_FALSE(spore_parser_has_error(parser));

    size_t n = 0;
    const spore_arg_t* args = spore_message_get_args(msg, &n);
    ASSERT_EQ(n, 1u);
    EXPECT_STREQ(args[0].pKey, "body");
    EXPECT_STREQ(args[0].pValue, "\n<<x>>\n");

    spore_message_destroy(msg);
    spore_parser_destroy(parser);
}
