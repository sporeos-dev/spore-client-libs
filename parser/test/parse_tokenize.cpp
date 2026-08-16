
#include "fixture.h"
#include "parser.h"

class Tokenize
    : public Fixture
    , public spore::parser
{
protected:
    virtual ~Tokenize() = default;
    std::vector<token_t> tokens;
};

TEST_F(Tokenize, request)
{
    tokenize("Something.something ~handle", tokens);
    EXPECT_EQ(tokens.size(), 2);
    EXPECT_EQ(tokens[0].type, token_t::type_t::NONE);
    EXPECT_EQ(tokens[1].type, token_t::type_t::HANDLE);
}

TEST_F(Tokenize, response)
{
    tokenize("~handle:Something.something ok", tokens);
    EXPECT_EQ(tokens.size(), 2);
    EXPECT_EQ(tokens[0].type, token_t::type_t::HANDLE);
    EXPECT_EQ(tokens[1].type, token_t::type_t::NONE);
}

TEST_F(Tokenize, witness)
{
    tokenize("witness body=\"some message\"", tokens);
    EXPECT_EQ(tokens.size(), 2);
    EXPECT_EQ(tokens[0].type, token_t::type_t::NONE);
    EXPECT_EQ(tokens[1].type, token_t::type_t::ARG);
}

TEST_F(Tokenize, publish)
{
    tokenize("publish Something.something", tokens);
    EXPECT_EQ(tokens.size(), 2);
    EXPECT_EQ(tokens[0].type, token_t::type_t::NONE);
    EXPECT_EQ(tokens[1].type, token_t::type_t::NONE);
}

TEST_F(Tokenize, quotes)
{
    tokenize("Some ~h body=\"some message\" flag", tokens);
    EXPECT_EQ(tokens.size(), 4);
    EXPECT_EQ(tokens[0].type, token_t::type_t::NONE);
    EXPECT_EQ(tokens[1].type, token_t::type_t::HANDLE);
    EXPECT_EQ(tokens[2].type, token_t::type_t::ARG);
    EXPECT_EQ(tokens[3].type, token_t::type_t::NONE);
    EXPECT_STREQ(tokens[2].value.c_str(), "body=some message");
}

TEST_F(Tokenize, single_quotes)
{
    tokenize("Some ~h body='some message' flag", tokens);
    EXPECT_EQ(tokens.size(), 4);
    EXPECT_EQ(tokens[0].type, token_t::type_t::NONE);
    EXPECT_EQ(tokens[1].type, token_t::type_t::HANDLE);
    EXPECT_EQ(tokens[2].type, token_t::type_t::ARG);
    EXPECT_EQ(tokens[3].type, token_t::type_t::NONE);
    EXPECT_STREQ(tokens[2].value.c_str(), "body=some message");
}

TEST_F(Tokenize, squares)
{
    tokenize("Some ~h body=[some message] flag", tokens);
    EXPECT_EQ(tokens.size(), 4);
    EXPECT_EQ(tokens[0].type, token_t::type_t::NONE);
    EXPECT_EQ(tokens[1].type, token_t::type_t::HANDLE);
    EXPECT_EQ(tokens[2].type, token_t::type_t::ARG);
    EXPECT_EQ(tokens[3].type, token_t::type_t::NONE);
    EXPECT_STREQ(tokens[2].value.c_str(), "body=[some message]");
}

TEST_F(Tokenize, curlies)
{
    tokenize("Some ~h body={some message} flag", tokens);
    EXPECT_EQ(tokens.size(), 4);
    EXPECT_EQ(tokens[0].type, token_t::type_t::NONE);
    EXPECT_EQ(tokens[1].type, token_t::type_t::HANDLE);
    EXPECT_EQ(tokens[2].type, token_t::type_t::ARG);
    EXPECT_EQ(tokens[3].type, token_t::type_t::NONE);
    EXPECT_STREQ(tokens[2].value.c_str(), "body={some message}");
}

TEST_F(Tokenize, parens)
{
    tokenize("Some ~h body=(some message) flag", tokens);
    EXPECT_EQ(tokens.size(), 4);
    EXPECT_EQ(tokens[0].type, token_t::type_t::NONE);
    EXPECT_EQ(tokens[1].type, token_t::type_t::HANDLE);
    EXPECT_EQ(tokens[2].type, token_t::type_t::ARG);
    EXPECT_EQ(tokens[3].type, token_t::type_t::NONE);
    EXPECT_STREQ(tokens[2].value.c_str(), "body=(some message)");
}

TEST_F(Tokenize, left_open)
{
    {
        tokenize("Some ~h body=\"some message", tokens);
        EXPECT_EQ(tokens.size(), 3);
        EXPECT_STREQ(tokens[2].value.c_str(), "body=some message");
        tokens.clear();
    }

    {
        tokenize("Some ~h body='some message", tokens);
        EXPECT_EQ(tokens.size(), 3);
        EXPECT_STREQ(tokens[2].value.c_str(), "body=some message");
        tokens.clear();
    }

    {
        tokenize("Some ~h body=[some message", tokens);
        EXPECT_EQ(tokens.size(), 3);
        EXPECT_STREQ(tokens[2].value.c_str(), "body=[some message]");
        tokens.clear();
    }

    {
        tokenize("Some ~h body={some message", tokens);
        EXPECT_EQ(tokens.size(), 3);
        EXPECT_STREQ(tokens[2].value.c_str(), "body={some message}");
        tokens.clear();
    }

    {
        tokenize("Some ~h body=(some message", tokens);
        EXPECT_EQ(tokens.size(), 3);
        EXPECT_STREQ(tokens[2].value.c_str(), "body=(some message)");
        tokens.clear();
    }
}
