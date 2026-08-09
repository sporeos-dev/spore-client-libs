
#include "spore_parser.h"
#include "defs.h"

#include <stdlib.h>
#include <string.h>

spore_parser_t* spore_parser_create()
{
    return new spore_parser_t();
}

void spore_parser_destroy(spore_parser_t* hParser)
{
    if (!hParser)
        return;
    delete hParser;
}

const char* spore_parser_get_raw(spore_parser_t* hParser)
{
    if (!hParser)
        return nullptr;
    if (hParser->parser.getRaw().empty())
        return nullptr;
    return hParser->parser.getRaw().data();
}

spore_parser_type_t spore_parser_get_type(spore_parser_t* hParser)
{
    if (!hParser)
        return SPORE_PARSER_TYPE_UNKNOWN;
    return hParser->parser.getType();
}

bool spore_parser_has_error(spore_parser_t* hParser)
{
    if (!hParser)
        return false;
    return hParser->parser.hasError();
}

const char* spore_parser_get_error_code(spore_parser_t* hParser)
{
    if (!hParser)
        return nullptr;
    if (!hParser->parser.hasError())
        return nullptr;
    if (hParser->parser.getErrorCode().empty())
        return nullptr;
    return hParser->parser.getErrorCode().data();
}

const char* spore_parser_get_error_what(spore_parser_t* hParser)
{
    if (!hParser)
        return nullptr;
    if (!hParser->parser.hasError())
        return nullptr;
    if (hParser->parser.getErrorWhat().empty())
        return nullptr;
    return hParser->parser.getErrorWhat().data();
}

spore_message_t* spore_message_create()
{
    return new spore_message_t();
}

void spore_message_destroy(spore_message_t* hMessage)
{
    if (!hMessage)
        return;
    delete hMessage;
}

const char* spore_message_get_capability(spore_message_t* hMessage)
{
    if (!hMessage)
        return nullptr;
    if (hMessage->message.getCapability().empty())
        return nullptr;
    return hMessage->message.getCapability().data();
}

const char* spore_message_get_handle(spore_message_t* hMessage)
{
    if (!hMessage)
        return nullptr;
    if (hMessage->message.getHandle().empty())
        return nullptr;
    return hMessage->message.getHandle().data();
}

const spore_arg_t* spore_message_get_args(spore_message_t* hMessage, size_t* pNum)
{
    if (!hMessage || !pNum)
        return nullptr;

    if (!hMessage)
    {
        if (pNum)
            *pNum = hMessage->message.getArgs().size();
        return nullptr;
    }

    auto& args = hMessage->message.getArgs();
    *pNum = args.size();
    if (args.empty())
        return nullptr;
    return args.data();
}

const char** spore_message_get_flags(spore_message_t* hMessage, size_t* pNum)
{
    if (!hMessage || !pNum)
        return nullptr;

    if (!hMessage)
    {
        if (pNum)
            *pNum = hMessage->message.getFlags().size();
        return nullptr;
    }

    auto& flags = hMessage->message.getFlags();
    *pNum = flags.size();
    if (flags.empty())
        return nullptr;
    return flags.data();
}

const char* spore_message_get_inline(spore_message_t* hMessage)
{
    if (!hMessage)
        return nullptr;
    if (hMessage->message.getInline().empty())
        return nullptr;
    return hMessage->message.getInline().data();
}

void spore_parse(spore_parser_t* hParser, const char* pMessage, size_t sz, spore_message_t* hParsed)
{
    if (!hParser || !pMessage || !hParsed)
        return;

    std::string_view message(pMessage, sz);
    hParser->parser.parse(message, hParsed);
}
