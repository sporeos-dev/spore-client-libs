#ifndef SPORE_PARSER_H
#define SPORE_PARSER_H

#include <stddef.h>

#ifdef __cplusplus
extern "C"
{
#endif

    // types
    typedef enum
    {
        SPORE_PARSER_TYPE_UNKNOWN,
        SPORE_PARSER_TYPE_REQUEST,
        SPORE_PARSER_TYPE_RESPONSE,
        SPORE_PARSER_TYPE_WITNESS,
        SPORE_PARSER_TYPE_PUBLISH
    } spore_parser_type_t;

    typedef struct spore_parser_t spore_parser_t;
    typedef struct spore_message_t spore_message_t;
    typedef struct
    {
        const char* pKey;
        const char* pValue;
    } spore_arg_t;

    // parser
    spore_parser_t* spore_parser_create(bool trace);
    void spore_parser_destroy(spore_parser_t* hParser);
    const char* spore_parser_get_raw(spore_parser_t* hParser);
    spore_parser_type_t spore_parser_get_type(spore_parser_t* hParser);
    bool spore_parser_has_error(spore_parser_t* hParser);
    const char* spore_parser_get_error_code(spore_parser_t* hParser);
    const char* spore_parser_get_error_what(spore_parser_t* hParser);

    // message
    spore_message_t* spore_message_create();
    void spore_message_destroy(spore_message_t* hMessage);
    const char* spore_message_get_capability(spore_message_t* hMessage);
    const char* spore_message_get_handle(spore_message_t* hMessage);
    const spore_arg_t* spore_message_get_args(spore_message_t* hMessage, size_t* pNum);
    const char** spore_message_get_flags(spore_message_t* hMessage, size_t* pNum);
    const char* spore_message_get_inline(spore_message_t* hMessage);

    // parse
    // check hasError() after parse for error handling
    void spore_parse(spore_parser_t* hParser,
                     const char* pRaw,
                     size_t sz,
                     spore_message_t* hmessage);

#ifdef __cplusplus
}
#endif
#endif  // SPORE_PARSER_H