
#include "defs.h"
#include "spore_parser.h"
#include "ptrace.h"
#include <vector>
#include <string>
#include <string_view>

namespace spore
{
    void parser::error(std::string_view code, std::string_view what)
    {
        errorCode = code;
        errorWhat = what;
    }

    std::string_view parser::getRaw() const
    {
        return raw;
    }

    spore_parser_type_t parser::getType() const
    {
        return type;
    }

    bool parser::hasError() const
    {
        return !errorCode.empty();
    }

    std::string_view parser::getErrorCode() const
    {
        return errorCode;
    }

    std::string_view parser::getErrorWhat() const
    {
        return errorWhat;
    }

    void parser::parse(std::string_view message, spore_message_t* hMessage)
    {
        Ptrace::msg("Parsing message", 0);
        Ptrace::string("Raw", raw, 1);

        raw = message;
        errorCode.clear();
        errorWhat.clear();

        if (!hMessage)
        {
            error("Missing", "message handle is null");
            return;
        }

        std::vector<token_t> tokens;
        props_t props;

        // tokenize
        Ptrace::msg("1 Tokenize", 1);
        tokenize(message, tokens);
        Ptrace::num("Count", tokens.size(), 2);
        for (const auto& el : tokens) Ptrace::string("Token", el.value, 2);

        // build
        Ptrace::msg("2 Build", 1);
        build(hMessage, tokens, props);
        Ptrace::string("Capability", hMessage->message.getCapability(), 2);
        Ptrace::string("Handle", hMessage->message.getHandle(), 2);
        Ptrace::num("Args", hMessage->message.getArgs().size(), 2);
        for (const auto& el : hMessage->message.getArgs()) Ptrace::string(el.pKey, el.pValue, 2);
        Ptrace::num("Flags", hMessage->message.getFlags().size(), 2);
        for (const auto& el : hMessage->message.getFlags()) Ptrace::list(el, 2);

        // verify
        Ptrace::msg("3 Verify", 1);
        verify(hMessage, props);
        Ptrace::type(getType(), 2);
        Ptrace::boolean("Error", hasError(), 2);
        Ptrace::string("Error Code", getErrorCode(), 2);
        Ptrace::string("Error What", getErrorWhat(), 2);

        Ptrace::msg("Parsing complete", 0);
    }

    void parser::tokenize(std::string_view message, std::vector<token_t>& tokens)
    {
        token_t curr;
        inside where = inside::NONE;

        //
        //
        // parse
        // & tokenize
        //

        int next = 0;
        for (const auto& c : message)
        {
            next++;
            switch (where)
            {
                    // inside
                    // quotes

                case inside::NONE:
                {
                    switch (c)
                    {
                        case ' ':
                        {
                            if (!curr.value.empty())
                            {
                                tokens.push_back(std::move(curr));
                                curr.value.clear();
                                curr.type = token_t::type_t::NONE;
                            }
                        }
                        break;
                        case '"':
                        {
                            where = inside::QUOTES;
                        }
                        break;
                        case '\'':
                        {
                            where = inside::S_QUOTES;
                        }
                        break;
                        case '[':
                        {
                            curr.value.push_back(c);
                            where = inside::SQUARES;
                        }
                        break;
                        case '{':
                        {
                            curr.value.push_back(c);
                            where = inside::CURLIES;
                        }
                        break;
                        case '=':
                        {
                            curr.value.push_back(c);
                            curr.type = token_t::type_t::ARG;
                        }
                        break;
                        case '~':
                        {
                            if (curr.value.empty())
                            {
                                curr.value.push_back(c);
                                curr.type = token_t::type_t::HANDLE;
                            }
                            else
                            {
                                curr.value.push_back(c);
                            }
                        }
                        break;
                        case '<':
                        {
                            if (message.size() > next && message[next] == '<')
                            {
                                curr.value.push_back(c);
                                where = inside::TRIANGLES;
                            }
                            else
                            {
                                curr.value.push_back(c);
                            }
                        }
                        break;
                        default:
                        {
                            curr.value.push_back(c);
                        }
                        break;
                    }
                }
                break;

                    // inside
                    // single quotes

                case inside::S_QUOTES:
                {
                    if (c == '\'')
                    {
                        where = inside::NONE;
                    }
                    else
                    {
                        curr.value.push_back(c);
                    }
                }
                break;

                    // inside
                    // quotes

                case inside::QUOTES:
                {
                    if (c == '"')
                    {
                        where = inside::NONE;
                    }
                    else
                    {
                        curr.value.push_back(c);
                    }
                }
                break;

                    // inside
                    // square brackets

                case inside::SQUARES:
                {
                    if (c == ']')
                    {
                        curr.value.push_back(c);
                        where = inside::NONE;
                    }
                    else
                    {
                        curr.value.push_back(c);
                    }
                }
                break;

                    // inside
                    // curly braces

                case inside::CURLIES:
                {
                    if (c == '}')
                    {
                        curr.value.push_back(c);
                        where = inside::NONE;
                    }
                    else
                    {
                        curr.value.push_back(c);
                    }
                }
                break;

                    // inside
                    // triangles

                case inside::TRIANGLES:
                {
                    if (c == '>' && message.size() > next && message[next] == '>')
                    {
                        curr.value.push_back(c);
                        where = inside::NONE;
                    }
                    else
                    {
                        curr.value.push_back(c);
                    }
                }
                break;
            }
        }

        switch (where)
        {
            case inside::S_QUOTES:
                error("Malformed", "single quotes not closed");
                break;
            case inside::QUOTES:
                error("Malformed", "quotes not closed");
                break;
            case inside::SQUARES:
                error("Malformed", "square brackets not closed");
                curr.value.push_back(']');
                break;
            case inside::CURLIES:
                error("Malformed", "curly braces not closed");
                curr.value.push_back('}');
                break;
            case inside::TRIANGLES:
                error("Malformed", "triangles not closed");
                curr.value.push_back('>');
                curr.value.push_back('>');
                break;
            case inside::NONE:
                // intentionally left blank
                break;
        }

        if (!curr.value.empty())
        {
            tokens.push_back(std::move(curr));
            curr.value.clear();
            curr.type = token_t::type_t::NONE;
        }
    }

    void parser::build(spore_message_t* hMessage,
                       const std::vector<token_t>& tokens,
                       props_t& props)
    {
        size_t i = 0;

        for (const auto& t : tokens)
        {
            if (i == 0)
            {
                switch (t.type)
                {
                    case token_t::type_t::ARG:
                        type = SPORE_PARSER_TYPE_REQUEST;
                        hMessage->message.setCapability(t.value.c_str());
                        error("Malformed", "first token cannot be argument");
                        break;
                    case token_t::type_t::HANDLE:
                    {
                        type = SPORE_PARSER_TYPE_RESPONSE;
                        std::string_view handle = t.value;
                        handle = handle.substr(1);
                        auto pos = handle.find(':');
                        if (pos == std::string::npos)
                        {
                            hMessage->message.setHandle(std::string(handle).c_str());
                            error("Malformed",
                                  "response missing command; [ ~h ] should be [ ~h:c ]");
                        }
                        else
                        {
                            std::string_view capability = handle.substr(pos + 1);
                            handle = handle.substr(0, pos);
                            hMessage->message.setHandle(std::string(handle).c_str());
                            hMessage->message.setCapability(std::string(capability).c_str());
                        }
                    }
                    break;
                    default:
                        if (t.value == "witness")
                        {
                            type = SPORE_PARSER_TYPE_WITNESS;
                            hMessage->message.setCapability(t.value.c_str());
                        }
                        else if (t.value == "publish")
                        {
                            type = SPORE_PARSER_TYPE_PUBLISH;
                        }
                        else
                        {
                            type = SPORE_PARSER_TYPE_REQUEST;
                            hMessage->message.setCapability(t.value.c_str());
                        }
                        break;
                }
            }
            else
            {
                if (type == SPORE_PARSER_TYPE_PUBLISH && i == 1)
                {
                    hMessage->message.setCapability(t.value.c_str());
                    switch (t.type)
                    {
                        case token_t::type_t::ARG:
                            error("Malformed",
                                  "publish missing topic; cannot be argument; [ k=v ] should "
                                  "be [ t ]");
                            break;
                        case token_t::type_t::HANDLE:
                            error("Malformed",
                                  "publish missing topic; cannot be handle; [ ~h ] should be [ "
                                  "t ]");
                            break;
                        case token_t::type_t::NONE:
                            // intentionally left blank
                            break;
                    }
                }
                else
                {
                    switch (t.type)
                    {
                        case token_t::type_t::ARG:
                        {
                            auto pos = t.value.find('=');
                            std::string key = t.value.substr(0, pos);
                            std::string value = t.value.substr(pos + 1);

                            // strip block delimiters and trim one boundary newline on each side
                            if (value.size() >= 4 && value[0] == '<' && value[1] == '<' &&
                                value[value.size() - 2] == '>' && value[value.size() - 1] == '>')
                            {
                                value = value.substr(2, value.size() - 4);
                                if (!value.empty() && value.front() == '\n')
                                    value.erase(0, 1);
                                if (!value.empty() && value.back() == '\n')
                                    value.pop_back();
                            }

                            if (key == "code")
                                props.hasArgCode = true;
                            else if (key == "what")
                                props.hasArgWhat = true;
                            else if (key == "body")
                                props.hasArgBody = true;
                            hMessage->message.addArg(key.c_str(), value.c_str());
                        }
                        break;
                        case token_t::type_t::HANDLE:
                        {
                            std::string_view handle = t.value;
                            handle = handle.substr(1);
                            hMessage->message.setHandle(std::string(handle).c_str());
                        }
                        break;
                        default:
                            if (t.value == "ok")
                                props.hasFlagOk = true;
                            else if (t.value == "error")
                                props.hasFlagError = true;
                            hMessage->message.addFlag(t.value.c_str());
                            break;
                    }
                }
            }

            i++;
        }
    }

    void parser::verify(spore_message_t* hMessage, const props_t& props)
    {
        switch (type)
        {
            case SPORE_PARSER_TYPE_REQUEST:
            {
                if (hMessage->message.getCapability().empty())
                    error("Malformed", "request missing command");
                else if (hMessage->message.getHandle().empty())
                    error("Malformed", "request missing handle");
            }
            break;

            case SPORE_PARSER_TYPE_RESPONSE:
            {
                if (hMessage->message.getCapability().empty())
                    error("Malformed", "response missing command");
                else if (hMessage->message.getHandle().empty())
                    error("Malformed", "response missing handle");
                else if (!props.hasFlagOk && !props.hasFlagError)
                    error("Malformed", "response missing ok or error flag");
                else if (props.hasFlagOk && props.hasFlagError)
                    error("Malformed", "response has both ok and error flags");
                else if (props.hasFlagError && !props.hasArgCode)
                    error("Malformed", "error response missing code");
                else if (props.hasFlagError && !props.hasArgWhat)
                    error("Malformed", "error response missing what");
            }
            break;

            case SPORE_PARSER_TYPE_WITNESS:
            {
                if (!props.hasArgBody)
                    error("Malformed", "witness missing body");
            }
            break;

            case SPORE_PARSER_TYPE_PUBLISH:
            {
                if (hMessage->message.getCapability().empty())
                    error("Malformed", "publish missing topic");
            }
            break;

            default:
                error("Malformed", "unknown message type");
                break;
        }
    }
}  // namespace spore
