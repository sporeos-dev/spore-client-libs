
#include "fixture.h"

class Publish : public Fixture
{
protected:
    virtual ~Publish() = default;
};

TEST_F(Publish, basic)
{
    parse("publish Something.something");
    EXPECT_EQ(spore_parser_get_type(parser), SPORE_PARSER_TYPE_PUBLISH);
    EXPECT_FALSE(spore_parser_has_error(parser));
}

TEST_F(Publish, error_missing_topic)
{
    parse("publish");
    EXPECT_EQ(spore_parser_get_type(parser), SPORE_PARSER_TYPE_PUBLISH);
    EXPECT_TRUE(spore_parser_has_error(parser));
    EXPECT_STREQ(spore_parser_get_error_code(parser), "Malformed");
}

TEST_F(Publish, error_topic_is_arg)
{
    parse("publish arg=value topic");
    EXPECT_EQ(spore_parser_get_type(parser), SPORE_PARSER_TYPE_PUBLISH);
    EXPECT_TRUE(spore_parser_has_error(parser));
    EXPECT_STREQ(spore_parser_get_error_code(parser), "Malformed");
}

TEST_F(Publish, args_and_flags)
{
    parse("publish Something.something arg1=value1 arg2=value2 flag1 flag2");
    EXPECT_EQ(spore_parser_get_type(parser), SPORE_PARSER_TYPE_PUBLISH);
    EXPECT_STREQ(spore_message_get_capability(message), "Something.something");
    EXPECT_STREQ(arg("arg1").data(), "value1");
    EXPECT_STREQ(arg("arg2").data(), "value2");
    EXPECT_TRUE(flag("flag1"));
    EXPECT_TRUE(flag("flag2"));
    EXPECT_FALSE(flag("flag"));
    EXPECT_FALSE(flag("arg1=value1"));
}