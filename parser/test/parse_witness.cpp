
#include "fixture.h"

class Witness : public Fixture
{
protected:
    virtual ~Witness() = default;
};

TEST_F(Witness, basic)
{
    parse("witness body=\"some message\"");
    EXPECT_EQ(spore_parser_get_type(parser), SPORE_PARSER_TYPE_WITNESS);
    EXPECT_FALSE(spore_parser_has_error(parser));
}

TEST_F(Witness, args_and_flags)
{
    parse("witness body=\"some message\" arg1=value1 arg2=value2 flag1 flag2");
    EXPECT_EQ(spore_parser_get_type(parser), SPORE_PARSER_TYPE_WITNESS);
    EXPECT_STREQ(spore_message_get_capability(message), "witness");
    EXPECT_STREQ(arg("body").data(), "some message");
    EXPECT_STREQ(arg("arg1").data(), "value1");
    EXPECT_STREQ(arg("arg2").data(), "value2");
    EXPECT_TRUE(flag("flag1"));
    EXPECT_TRUE(flag("flag2"));
    EXPECT_FALSE(flag("flag"));
    EXPECT_FALSE(flag("arg1=value1"));
}

TEST_F(Witness, missing_body)
{
    parse("witness flag1");
    EXPECT_EQ(spore_parser_get_type(parser), SPORE_PARSER_TYPE_WITNESS);
    EXPECT_TRUE(spore_parser_has_error(parser));
    EXPECT_STREQ(spore_parser_get_error_code(parser), "Malformed");
}
