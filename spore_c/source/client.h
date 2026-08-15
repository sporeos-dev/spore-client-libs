#pragma once

#include "spore_c.h"
#include <mutex>
#include <string>
#include <string_view>
#include <vector>
#include <map>

#if defined(_WIN32)
    #include <winsock2.h>
using socket_t = SOCKET;
static constexpr socket_t kInvalidSocket = INVALID_SOCKET;
#else
using socket_t = int;
static constexpr socket_t kInvalidSocket = -1;
#endif

namespace spore
{
    class Client
    {
    public:
        explicit Client(std::string_view nodeId);
        ~Client();

        bool isConnected() const;
        bool hasError() const;
        std::string_view getErrorCode() const;
        std::string_view getErrorWhat() const;

        void connect();
        void disconnect();

        spore_handler_t* registerRequestHandler(spore_request_fn fn);
        spore_handler_t* registerResponseHandler(spore_response_fn fn);
        spore_handler_t* registerWitnessHandler(spore_witness_fn fn);
        spore_handler_t* registerPublishHandler(spore_publish_fn fn);
        void unregisterHandler(spore_handler_t* hHandler);

        void send(const spore_request_t* hRequest);
        void sendResponse(const spore_response_t* hResponse);
        void sendResponseError(const spore_response_error_t* hError);
        void sendWitness(const spore_witness_t* hWitness);
        void sendPublish(const spore_publish_t* hPublish);

        void listen();
        void sendRaw(const char* data, size_t len);

        spore_client_t* self = nullptr;  // back-pointer set by spore_client_create

    private:
        void error(std::string_view code, std::string_view what);
        void handshake();
        void writeRaw(const char* data, size_t len);  // mutex-protected socket write
        static std::string defaultSocketPath();

        std::string nodeId;
        socket_t socketFd = kInvalidSocket;
        bool connected = false;
        int nextId = 0;
        std::string errorCode;
        std::string errorWhat;

        std::mutex writeMu;

        std::vector<spore_handler_t*> requestHandlers;
        std::vector<spore_handler_t*> responseHandlers;
        std::vector<spore_handler_t*> witnessHandlers;
        std::vector<spore_handler_t*> publishHandlers;
    };

}  // namespace spore
