
#include "fixture.h"

class Request : public Fixture
{
protected:
    virtual ~Request() = default;
};

TEST_F(Request, basic)
{
    parse("Something.something ~handle");
    EXPECT_EQ(spore_parser_get_type(parser), SPORE_PARSER_TYPE_REQUEST);
    EXPECT_FALSE(spore_parser_has_error(parser));
    EXPECT_STREQ(spore_message_get_capability(message), "Something.something");
    EXPECT_STREQ(spore_message_get_handle(message), "handle");
}

TEST_F(Request, error_missing_handle)
{
    parse("Something.something arg=value flag1");
    EXPECT_EQ(spore_parser_get_type(parser), SPORE_PARSER_TYPE_REQUEST);
    EXPECT_STREQ(spore_message_get_capability(message), "Something.something");
    EXPECT_TRUE(spore_parser_has_error(parser));
    EXPECT_STREQ(spore_parser_get_error_code(parser), "Malformed");
}

TEST_F(Request, error_start_arg)
{
    parse("arg=value Something.something ~h");
    EXPECT_EQ(spore_parser_get_type(parser), SPORE_PARSER_TYPE_REQUEST);
    EXPECT_STREQ(spore_message_get_capability(message), "arg=value");
    EXPECT_TRUE(spore_parser_has_error(parser));
    EXPECT_STREQ(spore_parser_get_error_code(parser), "Malformed");
}

TEST_F(Request, args_and_flags)
{
    parse("Something.something ~h arg1=value1 arg2=value2 flag1 flag2");
    EXPECT_EQ(spore_parser_get_type(parser), SPORE_PARSER_TYPE_REQUEST);
    EXPECT_STREQ(spore_message_get_capability(message), "Something.something");
    EXPECT_STREQ(spore_message_get_handle(message), "h");
    EXPECT_STREQ(arg("arg1").data(), "value1");
    EXPECT_STREQ(arg("arg2").data(), "value2");
    EXPECT_TRUE(flag("flag1"));
    EXPECT_TRUE(flag("flag2"));
    EXPECT_FALSE(flag("flag"));
    EXPECT_FALSE(flag("arg1=value1"));
}
