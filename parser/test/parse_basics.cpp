
#include "fixture.h"

class Basics : public Fixture
{
protected:
    virtual ~Basics() = default;
};

TEST_F(Basics, create_destroy)
{
    EXPECT_NE(parser, nullptr);
    EXPECT_NE(message, nullptr);
}

TEST_F(Basics, before)
{
    EXPECT_EQ(spore_parser_get_raw(parser), nullptr);
    EXPECT_EQ(spore_parser_get_type(parser), SPORE_PARSER_TYPE_UNKNOWN);
    EXPECT_FALSE(spore_parser_has_error(parser));
    EXPECT_EQ(spore_parser_get_error_code(parser), nullptr);
    EXPECT_EQ(spore_parser_get_error_what(parser), nullptr);

    EXPECT_EQ(spore_message_get_capability(message), nullptr);
    EXPECT_EQ(spore_message_get_handle(message), nullptr);
    size_t num = 0;
    EXPECT_EQ(spore_message_get_args(message, &num), nullptr);
    EXPECT_EQ(num, 0);
    EXPECT_EQ(spore_message_get_flags(message, &num), nullptr);
    EXPECT_EQ(num, 0);
}

TEST_F(Basics, raw)
{
    std::string raw = "Hello, world!";
    parse(raw);

    EXPECT_STREQ(spore_parser_get_raw(parser), raw.c_str());
}