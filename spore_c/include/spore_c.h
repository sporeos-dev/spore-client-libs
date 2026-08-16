#ifndef SPORE_C_H
#define SPORE_C_H

#include <stddef.h>
#include <stdbool.h>

#ifdef __cplusplus
extern "C"
{
#endif

    //
    //
    // types
    //

    typedef struct spore_client_t spore_client_t;
    typedef struct spore_handler_t spore_handler_t;
    typedef struct spore_request_t spore_request_t;
    typedef struct spore_response_t spore_response_t;
    typedef struct spore_response_error_t spore_response_error_t;
    typedef struct spore_witness_t spore_witness_t;
    typedef struct spore_publish_t spore_publish_t;

    // handlers
    typedef void (*spore_request_fn)(spore_client_t* hClient, const spore_request_t* hRequest);
    typedef void (*spore_response_fn)(spore_client_t* hClient,
                                      const spore_response_t* hResponse,
                                      const spore_response_error_t* hError);
    typedef void (*spore_witness_fn)(spore_client_t* hClient, const spore_witness_t* hWitness);
    typedef void (*spore_publish_fn)(spore_client_t* hClient, const spore_publish_t* hPublish);
    typedef void (*spore_parse_error_fn)(spore_client_t* hClient,
                                         const char* pCode,
                                         const char* pWhat,
                                         const char* pRaw);

    //
    //
    // client
    //

    spore_client_t* spore_client_create(const char* pNodeId);
    void spore_client_destroy(spore_client_t* hClient);
    void spore_client_connect(spore_client_t* hClient);
    void spore_client_disconnect(spore_client_t* hClient);
    bool spore_client_is_connected(spore_client_t* hClient);

    // error
    bool spore_client_has_error(spore_client_t* hClient);
    const char* spore_client_get_error_code(spore_client_t* hClient);
    const char* spore_client_get_error_what(spore_client_t* hClient);

    // handlers
    spore_handler_t* spore_client_register_request_handler(spore_client_t* hClient,
                                                           spore_request_fn fn);
    spore_handler_t* spore_client_register_witness_handler(spore_client_t* hClient,
                                                           spore_witness_fn fn);
    spore_handler_t* spore_client_register_response_handler(spore_client_t* hClient,
                                                            spore_response_fn fn);
    spore_handler_t* spore_client_register_publish_handler(spore_client_t* hClient,
                                                           spore_publish_fn fn);
    spore_handler_t* spore_client_register_parse_error_handler(spore_client_t* hClient,
                                                               spore_parse_error_fn fn);
    void spore_client_unregister_handler(spore_client_t* hClient, spore_handler_t* hHandler);

    // main event loop
    void spore_client_listen(spore_client_t* hClient);

    // raw wire send
    void spore_client_send_raw(spore_client_t* hClient, const char* pData, size_t sz);

    //
    //
    // requests
    //

    // builder
    spore_request_t* spore_request_create();
    spore_request_t* spore_request_create_from_raw(const char* pRaw, size_t sz);
    void spore_request_destroy(spore_request_t* hRequest);

    // setters
    void spore_request_set_command(spore_request_t* hRequest, const char* pCommand);
    void spore_request_set_handle(spore_request_t* hRequest, const char* pHandle);
    void spore_request_add_arg(spore_request_t* hRequest, const char* pKey, const char* pValue);
    void spore_request_add_flag(spore_request_t* hRequest, const char* pFlag);

    // getters
    const char* spore_request_get_command(const spore_request_t* hRequest);
    const char* spore_request_get_handle(const spore_request_t* hRequest);
    const char* spore_request_get_arg(const spore_request_t* hRequest, const char* pKey);
    bool spore_request_has_flag(const spore_request_t* hRequest, const char* pFlag);

    // send
    void spore_request_send(spore_client_t* hClient, const spore_request_t* hRequest);

    // serialize
    void spore_request_serialize(spore_request_t* hRequest);
    const char* spore_request_get_serialized(const spore_request_t* hRequest);

    //
    //
    // responses
    // ok
    //

    // builder
    spore_response_t* spore_response_create();
    spore_response_t* spore_response_create_from_raw(const char* pRaw, size_t sz);
    void spore_response_destroy(spore_response_t* hResponse);

    // setters
    void spore_response_set_handle(spore_response_t* hResponse, const char* pHandle);
    void spore_response_set_command(spore_response_t* hResponse, const char* pCommand);
    void spore_response_add_arg(spore_response_t* hResponse, const char* pKey, const char* pValue);
    void spore_response_add_flag(spore_response_t* hResponse, const char* pFlag);

    // getters
    const char* spore_response_get_command(const spore_response_t* hResponse);
    const char* spore_response_get_handle(const spore_response_t* hResponse);
    const char* spore_response_get_arg(const spore_response_t* hResponse, const char* pKey);
    bool spore_response_has_flag(const spore_response_t* hResponse, const char* pFlag);

    // send
    void spore_response_send(spore_client_t* hClient, const spore_response_t* hResponse);

    // serialize
    void spore_response_serialize(spore_response_t* hResponse);
    const char* spore_response_get_serialized(const spore_response_t* hResponse);

    //
    //
    // responses
    // error
    //

    // builder
    spore_response_error_t* spore_response_error_create();
    spore_response_error_t* spore_response_error_create_from_raw(const char* pRaw, size_t sz);
    void spore_response_error_destroy(spore_response_error_t* hError);

    // setters
    void spore_response_error_set_handle(spore_response_error_t* hError, const char* pHandle);
    void spore_response_error_set_command(spore_response_error_t* hError, const char* pCommand);
    void spore_response_error_set_code(spore_response_error_t* hError, const char* pCode);
    void spore_response_error_set_what(spore_response_error_t* hError, const char* pWhat);
    void spore_response_error_add_arg(spore_response_error_t* hError,
                                      const char* pKey,
                                      const char* pValue);
    void spore_response_error_add_flag(spore_response_error_t* hError, const char* pFlag);

    // getters
    const char* spore_response_error_get_command(const spore_response_error_t* hError);
    const char* spore_response_error_get_handle(const spore_response_error_t* hError);
    const char* spore_response_error_get_code(const spore_response_error_t* hError);
    const char* spore_response_error_get_what(const spore_response_error_t* hError);
    const char* spore_response_error_get_arg(const spore_response_error_t* hError,
                                             const char* pKey);
    bool spore_response_error_has_flag(const spore_response_error_t* hError, const char* pFlag);

    // send
    void spore_response_error_send(spore_client_t* hClient, const spore_response_error_t* hError);

    // serialize
    void spore_response_error_serialize(spore_response_error_t* hError);
    const char* spore_response_error_get_serialized(const spore_response_error_t* hError);

    //
    //
    // witness
    //

    // builder
    spore_witness_t* spore_witness_create();
    spore_witness_t* spore_witness_create_from_raw(const char* pRaw, size_t sz);
    void spore_witness_destroy(spore_witness_t* hWitness);

    // setters
    void spore_witness_set_body(spore_witness_t* hWitness, const char* pBody);
    void spore_witness_add_arg(spore_witness_t* hWitness, const char* pKey, const char* pValue);
    void spore_witness_add_flag(spore_witness_t* hWitness, const char* pFlag);

    // getters
    const char* spore_witness_get_body(const spore_witness_t* hWitness);
    const char* spore_witness_get_arg(const spore_witness_t* hWitness, const char* pKey);
    bool spore_witness_has_flag(const spore_witness_t* hWitness, const char* pFlag);

    // send
    void spore_witness_send(spore_client_t* hClient, const spore_witness_t* hWitness);

    // serialize
    void spore_witness_serialize(spore_witness_t* hWitness);
    const char* spore_witness_get_serialized(const spore_witness_t* hWitness);

    //
    //
    // publish
    //

    // builder
    spore_publish_t* spore_publish_create();
    spore_publish_t* spore_publish_create_from_raw(const char* pRaw, size_t sz);
    void spore_publish_destroy(spore_publish_t* hPublish);

    // setters
    void spore_publish_set_topic(spore_publish_t* hPublish, const char* pTopic);
    void spore_publish_add_arg(spore_publish_t* hPublish, const char* pKey, const char* pValue);
    void spore_publish_add_flag(spore_publish_t* hPublish, const char* pFlag);

    // getters
    const char* spore_publish_get_topic(const spore_publish_t* hPublish);
    const char* spore_publish_get_arg(const spore_publish_t* hPublish, const char* pKey);
    bool spore_publish_has_flag(const spore_publish_t* hPublish, const char* pFlag);

    // send
    void spore_publish_send(spore_client_t* hClient, const spore_publish_t* hPublish);

    // serialize
    void spore_publish_serialize(spore_publish_t* hPublish);
    const char* spore_publish_get_serialized(const spore_publish_t* hPublish);

#ifdef __cplusplus
}
#endif

#endif  // SPORE_C_H
