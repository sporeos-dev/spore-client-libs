#include "defs.h"
#include "spore_parser.h"
#include <algorithm>
#include <chrono>
#include <cstring>
#include <string>
#include <iostream>

#if defined(_WIN32)
    #include <winsock2.h>
    #include <afunix.h>
    #pragma comment(lib, "ws2_32.lib")
    #define SPORE_CLOSE(fd) closesocket(fd)
    #define SPORE_WRITE(fd, buf, len) send(fd, buf, (int)(len), 0)
    #define SPORE_READ(fd, buf, le``n) recv(fd, buf, (int)(len), 0)
#else
    #include <sys/socket.h>
    #include <sys/un.h>
    #include <unistd.h>
    #define SPORE_CLOSE(fd) ::close(fd)
    #define SPORE_WRITE(fd, buf, len) ::write(fd, buf, len)
    #define SPORE_READ(fd, buf, len) ::read(fd, buf, len)
#endif

namespace spore
{
    Client::Client(std::string_view id) : nodeId(id)
    {
    }

    Client::~Client()
    {
        disconnect();
        for (auto* h : requestHandlers) delete h;
        for (auto* h : responseHandlers) delete h;
        for (auto* h : witnessHandlers) delete h;
        for (auto* h : publishHandlers) delete h;
    }

    void Client::error(std::string_view code, std::string_view what)
    {
        errorCode = code;
        errorWhat = what;
    }

    bool Client::hasError() const
    {
        return !errorCode.empty();
    }
    std::string_view Client::getErrorCode() const
    {
        return errorCode;
    }
    std::string_view Client::getErrorWhat() const
    {
        return errorWhat;
    }
    bool Client::isConnected() const
    {
        return connected;
    }

    std::string Client::defaultSocketPath()
    {
#if defined(__APPLE__)
        return "/Library/Application Support/spore-os/spore.sock";
#elif defined(__linux__)
        return "/var/lib/spore-os/spore.sock";
#elif defined(_WIN32)
        const char* base = std::getenv("LOCALAPPDATA");
        if (base)
            return std::string(base) + R"(\spore-os\spore.sock)";
        return R"(C:\Users\Default\AppData\Local\spore-os\spore.sock)";
#else
        return "/tmp/spore.sock";
#endif
    }

    void Client::connect()
    {
        errorCode.clear();
        errorWhat.clear();

        std::string path = defaultSocketPath();

#if defined(_WIN32)
        WSADATA wsa;
        WSAStartup(MAKEWORD(2, 2), &wsa);
#endif

        socket_t fd = ::socket(AF_UNIX, SOCK_STREAM, 0);
        if (fd == kInvalidSocket)
        {
            error("ConnectFailed", "failed to create socket");
            return;
        }

        struct sockaddr_un addr{};
        addr.sun_family = AF_UNIX;
        std::strncpy(addr.sun_path, path.c_str(), sizeof(addr.sun_path) - 1);

        if (::connect(fd, reinterpret_cast<struct sockaddr*>(&addr), sizeof(addr)) < 0)
        {
            SPORE_CLOSE(fd);
            error("ConnectFailed", "failed to connect to daemon at " + path);
            return;
        }

        socketFd = fd;
        handshake();

        if (hasError())
        {
            SPORE_CLOSE(socketFd);
            socketFd = kInvalidSocket;
            return;
        }

        connected = true;
    }

    void Client::handshake()
    {
        // Set 5 second read timeout (matches Go handshakeTimeout).
#if defined(_WIN32)
        DWORD tv = 5000;
        setsockopt(socketFd,
                   SOL_SOCKET,
                   SO_RCVTIMEO,
                   reinterpret_cast<const char*>(&tv),
                   sizeof(tv));
#else
        struct timeval tv{ 5, 0 };
        setsockopt(socketFd, SOL_SOCKET, SO_RCVTIMEO, &tv, sizeof(tv));
#endif

        std::string msg = nodeId + "\n";
        if (SPORE_WRITE(socketFd, msg.c_str(), msg.size()) < 0)
        {
            error("HandshakeFailed", "failed to send node id");
            return;
        }

        // Read until newline, up to 63 bytes.
        char buf[64] = {};
        size_t pos = 0;
        while (pos < sizeof(buf) - 1)
        {
            auto n = SPORE_READ(socketFd, buf + pos, 1);
            if (n <= 0)
            {
                error("HandshakeFailed", "no response from daemon");
                return;
            }
            if (buf[pos] == '\n')
                break;
            pos++;
        }

        // Clear timeout.
#if defined(_WIN32)
        tv = 0;
        setsockopt(socketFd,
                   SOL_SOCKET,
                   SO_RCVTIMEO,
                   reinterpret_cast<const char*>(&tv),
                   sizeof(tv));
#else
        tv = { 0, 0 };
        setsockopt(socketFd, SOL_SOCKET, SO_RCVTIMEO, &tv, sizeof(tv));
#endif

        std::string response(buf);
        while (!response.empty() && (response.back() == '\n' || response.back() == '\r'))
            response.pop_back();

        if (response != "OK")
            error("HandshakeFailed", response);
    }

    void Client::disconnect()
    {
        errorCode.clear();
        errorWhat.clear();
        if (!connected && socketFd == kInvalidSocket)
            return;
        if (socketFd != kInvalidSocket)
        {
            SPORE_CLOSE(socketFd);
            socketFd = kInvalidSocket;
        }
        connected = false;
    }

    // --- Handler registration ---

    spore_handler_t* Client::registerRequestHandler(spore_request_fn fn)
    {
        auto* h = new spore_handler_t{};
        h->type = spore_handler_t::type_t::request;
        h->id = nextId++;
        h->onRequest = fn;
        requestHandlers.push_back(h);
        return h;
    }

    spore_handler_t* Client::registerResponseHandler(spore_response_fn fn)
    {
        auto* h = new spore_handler_t{};
        h->type = spore_handler_t::type_t::response;
        h->id = nextId++;
        h->onResponse = fn;
        responseHandlers.push_back(h);
        return h;
    }

    spore_handler_t* Client::registerWitnessHandler(spore_witness_fn fn)
    {
        auto* h = new spore_handler_t{};
        h->type = spore_handler_t::type_t::witness;
        h->id = nextId++;
        h->onWitness = fn;
        witnessHandlers.push_back(h);
        return h;
    }

    spore_handler_t* Client::registerPublishHandler(spore_publish_fn fn)
    {
        auto* h = new spore_handler_t{};
        h->type = spore_handler_t::type_t::publish;
        h->id = nextId++;
        h->onPublish = fn;
        publishHandlers.push_back(h);
        return h;
    }

    void Client::unregisterHandler(spore_handler_t* hHandler)
    {
        if (!hHandler)
            return;

        auto erase = [&](std::vector<spore_handler_t*>& list)
        {
            auto it = std::find(list.begin(), list.end(), hHandler);
            if (it != list.end())
            {
                list.erase(it);
                delete hHandler;
            }
        };

        switch (hHandler->type)
        {
            case spore_handler_t::type_t::request:
                erase(requestHandlers);
                break;
            case spore_handler_t::type_t::response:
                erase(responseHandlers);
                break;
            case spore_handler_t::type_t::witness:
                erase(witnessHandlers);
                break;
            case spore_handler_t::type_t::publish:
                erase(publishHandlers);
                break;
        }
    }

    // --- Wire sends ---

    void Client::writeRaw(const char* data, size_t len)
    {
        std::lock_guard<std::mutex> lk(writeMu);
        if (SPORE_WRITE(socketFd, data, static_cast<int>(len)) < 0)
        {
            error("SendFailed", "failed to write to socket");
            return;
        }
        // hub uses ReadString('\n') so every message must end with a newline
        if (SPORE_WRITE(socketFd, "\n", 1) < 0)
            error("SendFailed", "failed to write newline terminator");
    }

    void Client::send(const spore_request_t* hRequest)
    {
        errorCode.clear();
        errorWhat.clear();
        if (!connected)
        {
            error("NotConnected", "client is not connected");
            return;
        }
        if (!hRequest)
        {
            error("NullArgument", "hRequest is null");
            return;
        }
        hRequest->request.serialize();
        const char* wire = hRequest->request.getSerialized();
        if (wire)
            writeRaw(wire, std::strlen(wire));
    }

    void Client::sendResponse(const spore_response_t* hResponse)
    {
        errorCode.clear();
        errorWhat.clear();
        if (!connected)
        {
            error("NotConnected", "client is not connected");
            return;
        }
        if (!hResponse)
        {
            error("NullArgument", "hResponse is null");
            return;
        }
        hResponse->response.serialize();
        const char* wire = hResponse->response.getSerialized();
        if (wire)
            writeRaw(wire, std::strlen(wire));
    }

    void Client::sendResponseError(const spore_response_error_t* hError)
    {
        errorCode.clear();
        errorWhat.clear();
        if (!connected)
        {
            error("NotConnected", "client is not connected");
            return;
        }
        if (!hError)
        {
            error("NullArgument", "hError is null");
            return;
        }
        hError->error.serialize();
        const char* wire = hError->error.getSerialized();
        if (wire)
            writeRaw(wire, std::strlen(wire));
    }

    void Client::sendWitness(const spore_witness_t* hWitness)
    {
        errorCode.clear();
        errorWhat.clear();
        if (!connected)
        {
            error("NotConnected", "client is not connected");
            return;
        }
        if (!hWitness)
        {
            error("NullArgument", "hWitness is null");
            return;
        }
        hWitness->witness.serialize();
        const char* wire = hWitness->witness.getSerialized();
        if (wire)
            writeRaw(wire, std::strlen(wire));
    }

    void Client::sendPublish(const spore_publish_t* hPublish)
    {
        errorCode.clear();
        errorWhat.clear();
        if (!connected)
        {
            error("NotConnected", "client is not connected");
            return;
        }
        if (!hPublish)
        {
            error("NullArgument", "hPublish is null");
            return;
        }
        hPublish->publish.serialize();
        const char* wire = hPublish->publish.getSerialized();
        if (wire)
            writeRaw(wire, std::strlen(wire));
    }

    void Client::listen()
    {
        errorCode.clear();
        errorWhat.clear();
        if (!connected)
        {
            error("NotConnected", "client is not connected");
            return;
        }

        std::string buf;
        buf.reserve(4096);
        char tmp[1024];

        spore_parser_t* parser = nullptr;
        spore_message_t* msg = nullptr;

        while (true)
        {
            auto n = SPORE_READ(socketFd, tmp, sizeof(tmp));
            if (n <= 0)
                break;

            buf.append(tmp, static_cast<size_t>(n));

            std::string::size_type pos;
            while ((pos = buf.find('\n')) != std::string::npos)
            {
                std::string line = buf.substr(0, pos);
                std::cout << "Received: " << line << std::endl;
                buf.erase(0, pos + 1);
                if (line.empty())
                    continue;

                if (parser)
                {
                    spore_parser_destroy(parser);
                    parser = nullptr;
                }
                if (msg)
                {
                    spore_message_destroy(msg);
                    msg = nullptr;
                }
                parser = spore_parser_create();
                msg = spore_message_create();

                spore_parse(parser, line.c_str(), line.size(), msg);
                if (spore_parser_has_error(parser))
                    continue;

                switch (spore_parser_get_type(parser))
                {
                    case SPORE_PARSER_TYPE_REQUEST:
                    {
                        spore_request_t req{};
                        req.request.setCommand(spore_message_get_capability(msg)
                                                   ? spore_message_get_capability(msg)
                                                   : "");
                        req.request.setHandle(
                            spore_message_get_handle(msg) ? spore_message_get_handle(msg) : "");
                        size_t na = 0;
                        const spore_arg_t* args = spore_message_get_args(msg, &na);
                        for (size_t i = 0; i < na; ++i)
                            req.request.addArg(args[i].pKey, args[i].pValue);
                        size_t nf = 0;
                        const char** flags = spore_message_get_flags(msg, &nf);
                        for (size_t i = 0; i < nf; ++i) req.request.addFlag(flags[i]);
                        for (auto* h : requestHandlers) h->onRequest(self, &req);
                        break;
                    }
                    case SPORE_PARSER_TYPE_RESPONSE:
                    {
                        // Determine ok vs error by checking flags
                        size_t nf = 0;
                        const char** flags = spore_message_get_flags(msg, &nf);
                        bool isError = false;
                        for (size_t i = 0; i < nf; ++i)
                            if (std::string_view(flags[i]) == "error" ||
                                std::string_view(flags[i]) == "custom_error")
                            {
                                isError = true;
                                break;
                            }

                        if (isError)
                        {
                            spore_response_error_t err{};
                            err.error.setCommand(spore_message_get_capability(msg));
                            err.error.setHandle(spore_message_get_handle(msg));
                            size_t na = 0;
                            const spore_arg_t* args = spore_message_get_args(msg, &na);
                            for (size_t i = 0; i < na; ++i)
                            {
                                std::string_view k(args[i].pKey);
                                if (k == "code")
                                    err.error.setCode(args[i].pValue);
                                else if (k == "what")
                                    err.error.setWhat(args[i].pValue);
                                else
                                    err.error.addArg(args[i].pKey, args[i].pValue);
                            }
                            for (size_t i = 0; i < nf; ++i) err.error.addFlag(flags[i]);
                            for (auto* h : responseHandlers) h->onResponse(self, nullptr, &err);
                        }
                        else
                        {
                            spore_response_t resp{};
                            resp.response.setCommand(spore_message_get_capability(msg)
                                                         ? spore_message_get_capability(msg)
                                                         : "");
                            resp.response.setHandle(
                                spore_message_get_handle(msg) ? spore_message_get_handle(msg) : "");
                            size_t na = 0;
                            const spore_arg_t* args = spore_message_get_args(msg, &na);
                            for (size_t i = 0; i < na; ++i)
                                resp.response.addArg(args[i].pKey, args[i].pValue);
                            for (size_t i = 0; i < nf; ++i) resp.response.addFlag(flags[i]);
                            for (auto* h : responseHandlers) h->onResponse(self, &resp, nullptr);
                        }
                        break;
                    }
                    case SPORE_PARSER_TYPE_WITNESS:
                    {
                        spore_witness_t wit{};
                        size_t na = 0;
                        const spore_arg_t* args = spore_message_get_args(msg, &na);
                        for (size_t i = 0; i < na; ++i)
                        {
                            if (std::string_view(args[i].pKey) == "body")
                                wit.witness.setBody(args[i].pValue);
                            else
                                wit.witness.addArg(args[i].pKey, args[i].pValue);
                        }
                        size_t nf = 0;
                        const char** flags = spore_message_get_flags(msg, &nf);
                        for (size_t i = 0; i < nf; ++i) wit.witness.addFlag(flags[i]);
                        for (auto* h : witnessHandlers) h->onWitness(self, &wit);
                        break;
                    }
                    case SPORE_PARSER_TYPE_PUBLISH:
                    {
                        spore_publish_t pub{};
                        if (const char* cap = spore_message_get_capability(msg))
                            pub.publish.setTopic(cap);
                        size_t na = 0;
                        const spore_arg_t* args = spore_message_get_args(msg, &na);
                        for (size_t i = 0; i < na; ++i)
                            pub.publish.addArg(args[i].pKey, args[i].pValue);
                        size_t nf = 0;
                        const char** flags = spore_message_get_flags(msg, &nf);
                        for (size_t i = 0; i < nf; ++i) pub.publish.addFlag(flags[i]);
                        for (auto* h : publishHandlers) h->onPublish(self, &pub);
                        break;
                    }
                    default:
                        break;
                }
            }
        }

        spore_message_destroy(msg);
        spore_parser_destroy(parser);
    }

    void Client::sendRaw(const char* data, size_t len)
    {
        errorCode.clear();
        errorWhat.clear();
        if (!connected)
        {
            error("NotConnected", "client is not connected");
            return;
        }
        writeRaw(data, len);
    }

}  // namespace spore
