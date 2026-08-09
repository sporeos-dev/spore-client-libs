
#include "fixture.h"

class Response : public Fixture
{
protected:
    virtual ~Response() = default;
};

TEST_F(Response, basic)
{
    parse("~handle:Something.something ok");
    EXPECT_EQ(spore_parser_get_type(parser), SPORE_PARSER_TYPE_RESPONSE);
    EXPECT_FALSE(spore_parser_has_error(parser));
    EXPECT_STREQ(spore_message_get_capability(message), "Something.something");
    EXPECT_STREQ(spore_message_get_handle(message), "handle");
    EXPECT_TRUE(flag("ok"));
}

TEST_F(Response, error_missing_command)
{
    parse("~handle ok");
    EXPECT_EQ(spore_parser_get_type(parser), SPORE_PARSER_TYPE_RESPONSE);
    EXPECT_TRUE(spore_parser_has_error(parser));
    EXPECT_STREQ(spore_parser_get_error_code(parser), "Malformed");
}

TEST_F(Response, error_missing_ok_error)
{
    parse("~handle:Something.something");
    EXPECT_EQ(spore_parser_get_type(parser), SPORE_PARSER_TYPE_RESPONSE);
    EXPECT_TRUE(spore_parser_has_error(parser));
    EXPECT_STREQ(spore_parser_get_error_code(parser), "Malformed");
}

TEST_F(Response, error_both_ok_error)
{
    parse("~handle:Something.something ok error");
    EXPECT_EQ(spore_parser_get_type(parser), SPORE_PARSER_TYPE_RESPONSE);
    EXPECT_TRUE(spore_parser_has_error(parser));
    EXPECT_STREQ(spore_parser_get_error_code(parser), "Malformed");
}

TEST_F(Response, error_missing_code)
{
    parse("~handle:Something.something error what=\"some message\"");
    EXPECT_EQ(spore_parser_get_type(parser), SPORE_PARSER_TYPE_RESPONSE);
    EXPECT_TRUE(spore_parser_has_error(parser));
    EXPECT_STREQ(spore_parser_get_error_code(parser), "Malformed");
}

TEST_F(Response, error_missing_what)
{
    parse("~handle:Something.something error code=123");
    EXPECT_EQ(spore_parser_get_type(parser), SPORE_PARSER_TYPE_RESPONSE);
    EXPECT_TRUE(spore_parser_has_error(parser));
    EXPECT_STREQ(spore_parser_get_error_code(parser), "Malformed");
}

TEST_F(Response, args_and_flags)
{
    parse("~handle:Something.something ok arg1=value1 arg2=value2 flag1 flag2");
    EXPECT_EQ(spore_parser_get_type(parser), SPORE_PARSER_TYPE_RESPONSE);
    EXPECT_STREQ(spore_message_get_capability(message), "Something.something");
    EXPECT_STREQ(spore_message_get_handle(message), "handle");
    EXPECT_STREQ(arg("arg1").data(), "value1");
    EXPECT_STREQ(arg("arg2").data(), "value2");
    EXPECT_TRUE(flag("flag1"));
    EXPECT_TRUE(flag("flag2"));
    EXPECT_FALSE(flag("flag"));
    EXPECT_FALSE(flag("arg1=value1"));
}
