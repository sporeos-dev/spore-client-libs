#pragma once

#include "spore_parser.h"
#include <string>
#include <string_view>
#include <vector>

namespace spore
{
    class parser
    {
    protected:
        struct props_t
        {
            bool hasFlagOk = false;
            bool hasFlagError = false;
            bool hasArgCode = false;
            bool hasArgWhat = false;
            bool hasArgBody = false;
        };

    private:
        std::string raw;
        spore_parser_type_t type = SPORE_PARSER_TYPE_UNKNOWN;
        std::string errorCode;
        std::string errorWhat;
        props_t props;

        void error(std::string_view code, std::string_view what);

    public:
        parser() = default;
        ~parser() = default;

        std::string_view getRaw() const;
        spore_parser_type_t getType() const;
        bool hasError() const;
        std::string_view getErrorCode() const;
        std::string_view getErrorWhat() const;

        void parse(std::string_view message, spore_message_t* hMessage);
        void validate(spore_message_t* hMessage);

    protected:
        struct token_t
        {
            enum class type_t
            {
                NONE,
                ARG,
                HANDLE,
            };
            std::string value;
            type_t type = type_t::NONE;
        };

        enum class inside
        {
            NONE,
            QUOTES,
            S_QUOTES,
            BRACKETS,
            TRIANGLES,
        };

        void tokenize(std::string_view message, std::vector<token_t>& tokens);
        void build(spore_message_t* hMessage, const std::vector<token_t>& tokens);
    };
}  // namespace spore
