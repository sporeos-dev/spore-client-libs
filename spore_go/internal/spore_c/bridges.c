#include "_cgo_export.h"
#include "spore_c.h"

spore_handler_t* registerRequestHandler(spore_client_t* c) {
    return spore_client_register_request_handler(c, (spore_request_fn)goRequestBridge);
}
spore_handler_t* registerResponseHandler(spore_client_t* c) {
    return spore_client_register_response_handler(c, (spore_response_fn)goResponseBridge);
}
spore_handler_t* registerWitnessHandler(spore_client_t* c) {
    return spore_client_register_witness_handler(c, (spore_witness_fn)goWitnessBridge);
}
spore_handler_t* registerPublishHandler(spore_client_t* c) {
    return spore_client_register_publish_handler(c, (spore_publish_fn)goPublishBridge);
}
