#include "spore_client.h"
#include "defs.h"

// Helper: return c_str() or nullptr when the string_view is empty.
static const char* sv(std::string_view s)
{
    return s.empty() ? nullptr : s.data();
}

// ============================================================
// client
// ============================================================

spore_client_t* spore_client_create(const char* pNodeId)
{
    if (!pNodeId)
        return nullptr;
    return new spore_client_t{spore::client(pNodeId)};
}

void spore_client_destroy(spore_client_t* hClient)
{
    delete hClient;
}

void spore_client_connect(spore_client_t* hClient)
{
    if (!hClient)
        return;
    hClient->client.connect();
}

void spore_client_disconnect(spore_client_t* hClient)
{
    if (!hClient)
        return;
    hClient->client.disconnect();
}

bool spore_client_is_connected(spore_client_t* hClient)
{
    return hClient && hClient->client.isConnected();
}

bool spore_client_has_error(spore_client_t* hClient)
{
    return hClient && hClient->client.hasError();
}

const char* spore_client_get_error_code(spore_client_t* hClient)
{
    if (!hClient || !hClient->client.hasError()) return nullptr;
    return sv(hClient->client.getErrorCode());
}

const char* spore_client_get_error_what(spore_client_t* hClient)
{
    if (!hClient || !hClient->client.hasError()) return nullptr;
    return sv(hClient->client.getErrorWhat());
}

// --- Handlers ---

spore_handler_t* spore_client_register_request_handler(spore_client_t* hClient,
                                                        spore_request_fn fn)
{
    if (!hClient || !fn) return nullptr;
    return hClient->client.registerRequestHandler(fn);
}

spore_handler_t* spore_client_register_response_handler(spore_client_t* hClient,
                                                         spore_response_fn fn)
{
    if (!hClient || !fn) return nullptr;
    return hClient->client.registerResponseHandler(fn);
}

spore_handler_t* spore_client_register_witness_handler(spore_client_t* hClient,
                                                        spore_witness_fn fn)
{
    if (!hClient || !fn) return nullptr;
    return hClient->client.registerWitnessHandler(fn);
}

spore_handler_t* spore_client_register_publish_handler(spore_client_t* hClient,
                                                        spore_publish_fn fn)
{
    if (!hClient || !fn) return nullptr;
    return hClient->client.registerPublishHandler(fn);
}

void spore_client_unregister_handler(spore_client_t* hClient, spore_handler_t* hHandler)
{
    if (!hClient) return;
    hClient->client.unregisterHandler(hHandler);
}

void spore_client_listen(spore_client_t* hClient)
{
    if (!hClient) return;
    hClient->client.listen();
}

// ============================================================
// request
// ============================================================

spore_request_t* spore_request_create()
{
    return new spore_request_t{};
}

spore_request_t* spore_request_create_from_raw(const char* pRaw, size_t sz)
{
    if (!pRaw) return nullptr;
    // TODO: parse via parser library, populate fields
    (void)sz;
    return nullptr;
}

void spore_request_destroy(spore_request_t* hRequest)
{
    delete hRequest;
}

void spore_request_set_command(spore_request_t* hRequest, const char* pCommand)
{
    if (!hRequest || !pCommand) return;
    hRequest->request.setCommand(pCommand);
}

void spore_request_set_handle(spore_request_t* hRequest, const char* pHandle)
{
    if (!hRequest || !pHandle) return;
    hRequest->request.setHandle(pHandle);
}

void spore_request_add_arg(spore_request_t* hRequest, const char* pKey, const char* pValue)
{
    if (!hRequest || !pKey || !pValue) return;
    hRequest->request.addArg(pKey, pValue);
}

void spore_request_add_flag(spore_request_t* hRequest, const char* pFlag)
{
    if (!hRequest || !pFlag) return;
    hRequest->request.addFlag(pFlag);
}

const char* spore_request_get_command(const spore_request_t* hRequest)
{
    if (!hRequest) return nullptr;
    return sv(hRequest->request.getCommand());
}

const char* spore_request_get_handle(const spore_request_t* hRequest)
{
    if (!hRequest) return nullptr;
    return sv(hRequest->request.getHandle());
}

const char* spore_request_get_arg(const spore_request_t* hRequest, const char* pKey)
{
    if (!hRequest || !pKey) return nullptr;
    return sv(hRequest->request.getArg(pKey));
}

bool spore_request_has_flag(const spore_request_t* hRequest, const char* pFlag)
{
    return hRequest && pFlag && hRequest->request.hasFlag(pFlag);
}

void spore_request_send(spore_client_t* hClient, const spore_request_t* hRequest)
{
    if (!hClient) return;
    hClient->client.send(hRequest);
}

// ============================================================
// response (ok)
// ============================================================

spore_response_t* spore_response_create()
{
    return new spore_response_t{};
}

spore_response_t* spore_response_create_from_raw(const char* pRaw, size_t sz)
{
    if (!pRaw) return nullptr;
    // TODO: parse via parser library, populate fields
    (void)sz;
    return nullptr;
}

void spore_response_destroy(spore_response_t* hResponse)
{
    delete hResponse;
}

void spore_response_set_handle(spore_response_t* hResponse, const char* pHandle)
{
    if (!hResponse || !pHandle) return;
    hResponse->response.setHandle(pHandle);
}

void spore_response_set_command(spore_response_t* hResponse, const char* pCommand)
{
    if (!hResponse || !pCommand) return;
    hResponse->response.setCommand(pCommand);
}

void spore_response_add_arg(spore_response_t* hResponse, const char* pKey, const char* pValue)
{
    if (!hResponse || !pKey || !pValue) return;
    hResponse->response.addArg(pKey, pValue);
}

void spore_response_add_flag(spore_response_t* hResponse, const char* pFlag)
{
    if (!hResponse || !pFlag) return;
    hResponse->response.addFlag(pFlag);
}

const char* spore_response_get_command(const spore_response_t* hResponse)
{
    if (!hResponse) return nullptr;
    return sv(hResponse->response.getCommand());
}

const char* spore_response_get_handle(const spore_response_t* hResponse)
{
    if (!hResponse) return nullptr;
    return sv(hResponse->response.getHandle());
}

const char* spore_response_get_arg(const spore_response_t* hResponse, const char* pKey)
{
    if (!hResponse || !pKey) return nullptr;
    return sv(hResponse->response.getArg(pKey));
}

bool spore_response_has_flag(const spore_response_t* hResponse, const char* pFlag)
{
    return hResponse && pFlag && hResponse->response.hasFlag(pFlag);
}

void spore_response_send(spore_client_t* hClient, const spore_response_t* hResponse)
{
    if (!hClient) return;
    hClient->client.sendResponse(hResponse);
}

// ============================================================
// response (error)
// ============================================================

spore_response_error_t* spore_response_error_create()
{
    return new spore_response_error_t{};
}

spore_response_error_t* spore_response_error_create_from_raw(const char* pRaw, size_t sz)
{
    if (!pRaw) return nullptr;
    // TODO: parse via parser library, populate fields
    (void)sz;
    return nullptr;
}

void spore_response_error_destroy(spore_response_error_t* hError)
{
    delete hError;
}

void spore_response_error_set_handle(spore_response_error_t* hError, const char* pHandle)
{
    if (!hError || !pHandle) return;
    hError->error.setHandle(pHandle);
}

void spore_response_error_set_command(spore_response_error_t* hError, const char* pCommand)
{
    if (!hError || !pCommand) return;
    hError->error.setCommand(pCommand);
}

void spore_response_error_set_code(spore_response_error_t* hError, const char* pCode)
{
    if (!hError || !pCode) return;
    hError->error.setCode(pCode);
}

void spore_response_error_set_what(spore_response_error_t* hError, const char* pWhat)
{
    if (!hError || !pWhat) return;
    hError->error.setWhat(pWhat);
}

void spore_response_error_add_arg(spore_response_error_t* hError,
                                   const char* pKey,
                                   const char* pValue)
{
    if (!hError || !pKey || !pValue) return;
    hError->error.addArg(pKey, pValue);
}

void spore_response_error_add_flag(spore_response_error_t* hError, const char* pFlag)
{
    if (!hError || !pFlag) return;
    hError->error.addFlag(pFlag);
}

const char* spore_response_error_get_command(const spore_response_error_t* hError)
{
    if (!hError) return nullptr;
    return sv(hError->error.getCommand());
}

const char* spore_response_error_get_handle(const spore_response_error_t* hError)
{
    if (!hError) return nullptr;
    return sv(hError->error.getHandle());
}

const char* spore_response_error_get_code(const spore_response_error_t* hError)
{
    if (!hError) return nullptr;
    return sv(hError->error.getCode());
}

const char* spore_response_error_get_what(const spore_response_error_t* hError)
{
    if (!hError) return nullptr;
    return sv(hError->error.getWhat());
}

const char* spore_response_error_get_arg(const spore_response_error_t* hError,
                                          const char* pKey)
{
    if (!hError || !pKey) return nullptr;
    return sv(hError->error.getArg(pKey));
}

bool spore_response_error_has_flag(const spore_response_error_t* hError, const char* pFlag)
{
    return hError && pFlag && hError->error.hasFlag(pFlag);
}

void spore_response_error_send(spore_client_t* hClient, const spore_response_error_t* hError)
{
    if (!hClient) return;
    hClient->client.sendResponseError(hError);
}

// ============================================================
// witness
// ============================================================

spore_witness_t* spore_witness_create()
{
    return new spore_witness_t{};
}

spore_witness_t* spore_witness_create_from_raw(const char* pRaw, size_t sz)
{
    if (!pRaw) return nullptr;
    // TODO: parse via parser library, populate fields
    (void)sz;
    return nullptr;
}

void spore_witness_destroy(spore_witness_t* hWitness)
{
    delete hWitness;
}

void spore_witness_set_body(spore_witness_t* hWitness, const char* pBody)
{
    if (!hWitness || !pBody) return;
    hWitness->witness.setBody(pBody);
}

void spore_witness_add_arg(spore_witness_t* hWitness, const char* pKey, const char* pValue)
{
    if (!hWitness || !pKey || !pValue) return;
    hWitness->witness.addArg(pKey, pValue);
}

void spore_witness_add_flag(spore_witness_t* hWitness, const char* pFlag)
{
    if (!hWitness || !pFlag) return;
    hWitness->witness.addFlag(pFlag);
}

const char* spore_witness_get_body(const spore_witness_t* hWitness)
{
    if (!hWitness) return nullptr;
    return sv(hWitness->witness.getBody());
}

const char* spore_witness_get_arg(const spore_witness_t* hWitness, const char* pKey)
{
    if (!hWitness || !pKey) return nullptr;
    return sv(hWitness->witness.getArg(pKey));
}

bool spore_witness_has_flag(const spore_witness_t* hWitness, const char* pFlag)
{
    return hWitness && pFlag && hWitness->witness.hasFlag(pFlag);
}

void spore_witness_send(spore_client_t* hClient, const spore_witness_t* hWitness)
{
    if (!hClient) return;
    hClient->client.sendWitness(hWitness);
}

// ============================================================
// publish
// ============================================================

spore_publish_t* spore_publish_create()
{
    return new spore_publish_t{};
}

spore_publish_t* spore_publish_create_from_raw(const char* pRaw, size_t sz)
{
    if (!pRaw) return nullptr;
    // TODO: parse via parser library, populate fields
    (void)sz;
    return nullptr;
}

void spore_publish_destroy(spore_publish_t* hPublish)
{
    delete hPublish;
}

void spore_publish_set_topic(spore_publish_t* hPublish, const char* pTopic)
{
    if (!hPublish || !pTopic) return;
    hPublish->publish.setTopic(pTopic);
}

void spore_publish_add_arg(spore_publish_t* hPublish, const char* pKey, const char* pValue)
{
    if (!hPublish || !pKey || !pValue) return;
    hPublish->publish.addArg(pKey, pValue);
}

void spore_publish_add_flag(spore_publish_t* hPublish, const char* pFlag)
{
    if (!hPublish || !pFlag) return;
    hPublish->publish.addFlag(pFlag);
}

const char* spore_publish_get_topic(const spore_publish_t* hPublish)
{
    if (!hPublish) return nullptr;
    return sv(hPublish->publish.getTopic());
}

const char* spore_publish_get_arg(const spore_publish_t* hPublish, const char* pKey)
{
    if (!hPublish || !pKey) return nullptr;
    return sv(hPublish->publish.getArg(pKey));
}

bool spore_publish_has_flag(const spore_publish_t* hPublish, const char* pFlag)
{
    return hPublish && pFlag && hPublish->publish.hasFlag(pFlag);
}

void spore_publish_send(spore_client_t* hClient, const spore_publish_t* hPublish)
{
    if (!hClient) return;
    hClient->client.sendPublish(hPublish);
}
