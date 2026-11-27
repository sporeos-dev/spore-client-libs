/*
 * Copyright 2026 Matt Harrison
 * SPDX-License-Identifier: Apache-2.0
 */

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
    typedef struct spore_stream_t spore_stream_t;

    typedef struct spore_string_view_t
    {
        const char* data;
        size_t size;
    } spore_string_view_t;

    typedef enum spore_stream_result_t
    {
        SPORE_STREAM_OK,
        SPORE_STREAM_INVALID_ARGUMENT,
        SPORE_STREAM_NOT_CONNECTED,
        SPORE_STREAM_SEND_FAILED,
        SPORE_STREAM_CLOSED
    } spore_stream_result_t;

    // handlers
    typedef void (*spore_request_fn)(spore_client_t* hClient, const spore_request_t* hRequest);
    typedef void (*spore_response_fn)(spore_client_t* hClient,
                                      const spore_response_t* hResponse,
                                      const spore_response_error_t* hError);
    // The chunk and its contents are borrowed and valid only during the callback.
    typedef void (*spore_stream_fn)(spore_client_t* hClient, const spore_stream_t* hStream);
    typedef void (*spore_witness_fn)(spore_client_t* hClient, const spore_witness_t* hWitness);
    typedef void (*spore_publish_fn)(spore_client_t* hClient, const spore_publish_t* hPublish);
    typedef void (*spore_parse_error_fn)(spore_client_t* hClient,
                                         const char* pCode,
                                         const char* pWhat,
                                         const char* pRaw);

    // runs once after the handshake, on its own thread, so it may use spore_client_send_and_wait.
    // Return true when setup succeeded and the client should announce READY.
    typedef bool (*spore_ready_fn)(spore_client_t* hClient);

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
    spore_handler_t* spore_client_register_stream_handler(spore_client_t* hClient,
                                                          spore_stream_fn fn);
    spore_handler_t* spore_client_register_publish_handler(spore_client_t* hClient,
                                                           spore_publish_fn fn);
    spore_handler_t* spore_client_register_parse_error_handler(spore_client_t* hClient,
                                                               spore_parse_error_fn fn);
    void spore_client_unregister_handler(spore_client_t* hClient, spore_handler_t* hHandler);

    // Only one ready function is kept; setting another replaces it. Set it before listen.
    void spore_client_set_ready_handler(spore_client_t* hClient, spore_ready_fn fn);

    // main event loop. Starts the ready thread, which sends READY once the ready function
    // returns true (or right away when there is none), then reads until disconnect.
    void spore_client_listen(spore_client_t* hClient);

    // Sends hRequest and blocks until the response with the same handle arrives. Returns false
    // on timeout or send failure. The raw response line is then available from
    // spore_request_get_response; parse it with spore_response_create_from_raw (or the _error_
    // variant when it has the error flag). Needs spore_client_listen running on another thread,
    // so do not call it from a handler. hRequest must have a handle.
    bool spore_client_send_and_wait(spore_client_t* hClient,
                                    const spore_request_t* hRequest,
                                    int timeoutMs);

    // raw wire send
    void spore_client_send_raw(spore_client_t* hClient, const char* pData, size_t sz);

    // Stream results belong to the individual operation, not the client's shared error state.
    // Views may contain empty text; nonempty views require data. NUL is not allowed.
    spore_stream_result_t spore_stream_send_start(spore_client_t* hClient,
                                                  spore_string_view_t handle,
                                                  spore_string_view_t command,
                                                  const spore_string_view_t* fields,
                                                  size_t fieldCount);
    spore_stream_result_t spore_stream_send_chunk(spore_client_t* hClient,
                                                  spore_string_view_t handle,
                                                  spore_string_view_t field,
                                                  spore_string_view_t text);
    spore_stream_result_t spore_stream_send_close(spore_client_t* hClient,
                                                  spore_string_view_t handle,
                                                  spore_string_view_t field);

    // Open copies the request identity and starts one streamed field. Keep the client alive
    // until the stream is destroyed. Destroying a stream does not send a response.
    spore_stream_result_t spore_stream_open(spore_client_t* hClient,
                                            const spore_request_t* hRequest,
                                            spore_string_view_t field,
                                            spore_stream_t** out);
    spore_stream_result_t spore_stream_chunk(spore_stream_t* hStream, spore_string_view_t text);
    // Close sends success; fail closes the field and sends an error with nonempty code/what.
    spore_stream_result_t spore_stream_close(spore_stream_t* hStream);
    spore_stream_result_t spore_stream_fail(spore_stream_t* hStream,
                                            spore_string_view_t code,
                                            spore_string_view_t what);
    // Do not destroy a stream concurrently with an operation on it.
    void spore_stream_destroy(spore_stream_t* hStream);

    const char* spore_stream_get_handle(const spore_stream_t* hStream);
    const char* spore_stream_get_field(const spore_stream_t* hStream);
    const char* spore_stream_get_text(const spore_stream_t* hStream, size_t* size);

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
    size_t spore_request_get_arg_ids(const spore_request_t* hRequest,
                                     const char** out,
                                     size_t capacity);
    size_t spore_request_get_flags(const spore_request_t* hRequest,
                                   const char** out,
                                   size_t capacity);
    const char* spore_request_get_raw(const spore_request_t* hRequest, size_t* size);

    // send
    void spore_request_send(spore_client_t* hClient, const spore_request_t* hRequest);

    // serialize
    void spore_request_serialize(spore_request_t* hRequest);
    const char* spore_request_get_serialized(const spore_request_t* hRequest);

    // raw response line stored by spore_client_send_and_wait; null if there is none
    const char* spore_request_get_response(const spore_request_t* hRequest);

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
    size_t spore_response_get_arg_ids(const spore_response_t* hResponse,
                                      const char** out,
                                      size_t capacity);
    size_t spore_response_get_flags(const spore_response_t* hResponse,
                                    const char** out,
                                    size_t capacity);
    const char* spore_response_get_raw(const spore_response_t* hResponse, size_t* size);

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
    size_t spore_response_error_get_arg_ids(const spore_response_error_t* hError,
                                            const char** out,
                                            size_t capacity);
    size_t spore_response_error_get_flags(const spore_response_error_t* hError,
                                          const char** out,
                                          size_t capacity);
    const char* spore_response_error_get_raw(const spore_response_error_t* hError, size_t* size);

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
    size_t spore_witness_get_arg_ids(const spore_witness_t* hWitness,
                                     const char** out,
                                     size_t capacity);
    size_t spore_witness_get_flags(const spore_witness_t* hWitness,
                                   const char** out,
                                   size_t capacity);
    const char* spore_witness_get_raw(const spore_witness_t* hWitness, size_t* size);

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
    size_t spore_publish_get_arg_ids(const spore_publish_t* hPublish,
                                     const char** out,
                                     size_t capacity);
    size_t spore_publish_get_flags(const spore_publish_t* hPublish,
                                   const char** out,
                                   size_t capacity);
    const char* spore_publish_get_raw(const spore_publish_t* hPublish, size_t* size);

    // send
    void spore_publish_send(spore_client_t* hClient, const spore_publish_t* hPublish);

    // serialize
    void spore_publish_serialize(spore_publish_t* hPublish);
    const char* spore_publish_get_serialized(const spore_publish_t* hPublish);

#ifdef __cplusplus
}
#endif

#endif  // SPORE_C_H
