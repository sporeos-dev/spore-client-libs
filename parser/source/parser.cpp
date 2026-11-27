// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: Apache-2.0

#include "defs.h"
#include "spore_parser.h"
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
        raw = message;
        errorCode.clear();
        errorWhat.clear();
        props = props_t{};
        type = SPORE_PARSER_TYPE_UNKNOWN;

        if (!hMessage)
            return;

        if (parseStreamChunk(message, hMessage))
            return;

        std::vector<token_t> tokens;

        // tokenize
        tokenize(message, tokens);

        // build
        build(hMessage, tokens);
    }

    bool parser::parseStreamChunk(std::string_view line, spore_message_t* hMessage)
    {
        if (line.empty() || line.front() != '~')
            return false;
        const auto marker = line.find("<<");
        const auto space = line.find(' ');
        if (marker == std::string_view::npos ||
            (space != std::string_view::npos && marker > space))
            return false;

        type = SPORE_PARSER_TYPE_STREAM;
        const auto colon = line.find(':', 1);
        if (colon == std::string_view::npos || colon == 1 || colon + 1 >= marker ||
            marker + 2 >= line.size() || line[marker + 2] != ' ' ||
            line.find('\0') != std::string_view::npos)
        {
            props.malformedStreamChunk = true;
            return true;
        }
        const auto handle = line.substr(1, colon - 1);
        const auto field = line.substr(colon + 1, marker - colon - 1);
        if (handle.find_first_of(" \t\r\n\v\f:<>") != std::string_view::npos ||
            field.find_first_of(" \t\r\n\v\f:<>") != std::string_view::npos)
        {
            props.malformedStreamChunk = true;
            return true;
        }

        std::string text;
        const auto payload = line.substr(marker + 3);
        for (size_t i = 0; i < payload.size(); ++i)
        {
            if (payload[i] == '\\' && i + 1 < payload.size())
            {
                switch (payload[i + 1])
                {
                    case '\\': text += '\\'; ++i; continue;
                    case 'n': text += '\n'; ++i; continue;
                    case 'r': text += '\r'; ++i; continue;
                    case 't': text += '\t'; ++i; continue;
                }
            }
            text += payload[i];
        }
        hMessage->message.setHandle(handle);
        hMessage->message.setStreamField(field);
        hMessage->message.setStreamText(text);
        return true;
    }

    void parser::tokenize(std::string_view message, std::vector<token_t>& tokens)
    {
        token_t curr;
        inside where = inside::NONE;
        char bracket = 0;
        size_t depth = 0;
        bool escaped = false;

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
                                curr = {};
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
                        case '{':
                        {
                            curr.value.push_back(c);
                            bracket = c;
                            depth = 1;
                            where = inside::BRACKETS;
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
                    if (escaped)
                    {
                        switch (c)
                        {
                            case '\\': curr.value.push_back('\\'); break;
                            case '"': curr.value.push_back('"'); break;
                            case 'n': curr.value.push_back('\n'); break;
                            case 'r': curr.value.push_back('\r'); break;
                            case 't': curr.value.push_back('\t'); break;
                            default:
                                curr.value.push_back('\\');
                                curr.value.push_back(c);
                                break;
                        }
                        escaped = false;
                    }
                    else if (c == '\\')
                    {
                        escaped = true;
                    }
                    else if (c == '"')
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
                    // square brackets or curly braces, tracked by depth of the same type

                case inside::BRACKETS:
                {
                    const char close = bracket == '[' ? ']' : '}';
                    curr.value.push_back(c);
                    if (c == bracket)
                    {
                        depth++;
                    }
                    else if (c == close)
                    {
                        depth--;
                        if (depth == 0)
                            where = inside::NONE;
                    }
                }
                break;
            }
        }

        // close anything left open, best effort
        switch (where)
        {
            case inside::BRACKETS:
                for (size_t i = 0; i < depth; ++i) curr.value.push_back(bracket == '[' ? ']' : '}');
                break;
            case inside::NONE:
            case inside::S_QUOTES:
                // intentionally left blank
                break;
            case inside::QUOTES:
                if (escaped)
                    curr.value.push_back('\\');
                break;
        }

        if (!curr.value.empty())
        {
            tokens.push_back(std::move(curr));
            curr.value.clear();
            curr.type = token_t::type_t::NONE;
        }
    }

    void parser::build(spore_message_t* hMessage, const std::vector<token_t>& tokens)
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

    void parser::validate(spore_message_t* hMessage)
    {
        errorCode.clear();
        errorWhat.clear();

        if (!hMessage)
        {
            error("Missing", "message handle is null");
            return;
        }

        switch (type)
        {
            case SPORE_PARSER_TYPE_STREAM:
            {
                if (props.malformedStreamChunk)
                    error("Malformed", "invalid stream chunk");
            }
            break;

            case SPORE_PARSER_TYPE_REQUEST:
            {
                if (hMessage->message.getCapability().empty())
                    error("Malformed", "request missing command");
                else if (hMessage->message.getCapability().find('=') != std::string_view::npos)
                    error("Malformed", "first token cannot be argument");
                else if (hMessage->message.getHandle().empty())
                    error("Malformed", "request missing handle");
            }
            break;

            case SPORE_PARSER_TYPE_RESPONSE:
            {
                if (hMessage->message.getCapability().empty())
                    error("Malformed", "response missing command; [ ~h ] should be [ ~h:c ]");
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
                else if (hMessage->message.getCapability().find('=') != std::string_view::npos)
                    error("Malformed",
                          "publish missing topic; cannot be argument; [ k=v ] should be [ t ]");
                else if (hMessage->message.getCapability().front() == '~')
                    error("Malformed",
                          "publish missing topic; cannot be handle; [ ~h ] should be [ t ]");
            }
            break;

            default:
                error("Malformed", "unknown message type");
                break;
        }
    }
}  // namespace spore
