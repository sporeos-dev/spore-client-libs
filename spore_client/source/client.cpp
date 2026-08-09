#include "defs.h"
#include <algorithm>
#include <cstring>

#if defined(_WIN32)
#include <winsock2.h>
#include <afunix.h>
#pragma comment(lib, "ws2_32.lib")
#define SPORE_CLOSE(fd) closesocket(fd)
#define SPORE_WRITE(fd, buf, len) send(fd, buf, (int)(len), 0)
#define SPORE_READ(fd, buf, len)  recv(fd, buf, (int)(len), 0)
#else
#include <sys/socket.h>
#include <sys/un.h>
#include <unistd.h>
#define SPORE_CLOSE(fd) ::close(fd)
#define SPORE_WRITE(fd, buf, len) ::write(fd, buf, len)
#define SPORE_READ(fd, buf, len)  ::read(fd, buf, len)
#endif

namespace spore
{
    client::client(std::string_view id) : nodeId(id) {}

    client::~client()
    {
        disconnect();
        for (auto* h : requestHandlers)  delete h;
        for (auto* h : responseHandlers) delete h;
        for (auto* h : witnessHandlers)  delete h;
        for (auto* h : publishHandlers)  delete h;
    }

    void client::error(std::string_view code, std::string_view what)
    {
        errorCode = code;
        errorWhat = what;
    }

    bool client::hasError() const              { return !errorCode.empty(); }
    std::string_view client::getErrorCode() const { return errorCode; }
    std::string_view client::getErrorWhat() const { return errorWhat; }
    bool client::isConnected() const           { return connected; }

    std::string client::defaultSocketPath()
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

    void client::connect()
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

    void client::handshake()
    {
        // Set 5 second read timeout (matches Go handshakeTimeout).
#if defined(_WIN32)
        DWORD tv = 5000;
        setsockopt(socketFd, SOL_SOCKET, SO_RCVTIMEO,
                   reinterpret_cast<const char*>(&tv), sizeof(tv));
#else
        struct timeval tv { 5, 0 };
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
        size_t pos   = 0;
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
        setsockopt(socketFd, SOL_SOCKET, SO_RCVTIMEO,
                   reinterpret_cast<const char*>(&tv), sizeof(tv));
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

    void client::disconnect()
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

    spore_handler_t* client::registerRequestHandler(spore_request_fn fn)
    {
        auto* h     = new spore_handler_t{};
        h->type     = spore_handler_t::type_t::request;
        h->id       = nextId++;
        h->onRequest = fn;
        requestHandlers.push_back(h);
        return h;
    }

    spore_handler_t* client::registerResponseHandler(spore_response_fn fn)
    {
        auto* h      = new spore_handler_t{};
        h->type      = spore_handler_t::type_t::response;
        h->id        = nextId++;
        h->onResponse = fn;
        responseHandlers.push_back(h);
        return h;
    }

    spore_handler_t* client::registerWitnessHandler(spore_witness_fn fn)
    {
        auto* h      = new spore_handler_t{};
        h->type      = spore_handler_t::type_t::witness;
        h->id        = nextId++;
        h->onWitness = fn;
        witnessHandlers.push_back(h);
        return h;
    }

    spore_handler_t* client::registerPublishHandler(spore_publish_fn fn)
    {
        auto* h      = new spore_handler_t{};
        h->type      = spore_handler_t::type_t::publish;
        h->id        = nextId++;
        h->onPublish = fn;
        publishHandlers.push_back(h);
        return h;
    }

    void client::unregisterHandler(spore_handler_t* hHandler)
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
            case spore_handler_t::type_t::request:  erase(requestHandlers);  break;
            case spore_handler_t::type_t::response: erase(responseHandlers); break;
            case spore_handler_t::type_t::witness:  erase(witnessHandlers);  break;
            case spore_handler_t::type_t::publish:  erase(publishHandlers);  break;
        }
    }

    // --- Send stubs ---

    void client::send(const spore_request_t* /*hRequest*/)
    {
        errorCode.clear();
        errorWhat.clear();
        if (!connected) { error("NotConnected", "client is not connected"); return; }
        // TODO: serialise request to Spore protocol and write to socket
        error("NotImplemented", "send not yet implemented");
    }

    void client::sendResponse(const spore_response_t* /*hResponse*/)
    {
        errorCode.clear();
        errorWhat.clear();
        if (!connected) { error("NotConnected", "client is not connected"); return; }
        // TODO: serialise ok response and write to socket
        error("NotImplemented", "sendResponse not yet implemented");
    }

    void client::sendResponseError(const spore_response_error_t* /*hError*/)
    {
        errorCode.clear();
        errorWhat.clear();
        if (!connected) { error("NotConnected", "client is not connected"); return; }
        // TODO: serialise error response and write to socket
        error("NotImplemented", "sendResponseError not yet implemented");
    }

    void client::sendWitness(const spore_witness_t* /*hWitness*/)
    {
        errorCode.clear();
        errorWhat.clear();
        if (!connected) { error("NotConnected", "client is not connected"); return; }
        // TODO: serialise witness and write to socket
        error("NotImplemented", "sendWitness not yet implemented");
    }

    void client::sendPublish(const spore_publish_t* /*hPublish*/)
    {
        errorCode.clear();
        errorWhat.clear();
        if (!connected) { error("NotConnected", "client is not connected"); return; }
        // TODO: serialise publish and write to socket
        error("NotImplemented", "sendPublish not yet implemented");
    }

    void client::listen()
    {
        errorCode.clear();
        errorWhat.clear();
        if (!connected) { error("NotConnected", "client is not connected"); return; }
        // TODO: read loop — parse lines with the parser library, dispatch to handlers
        error("NotImplemented", "listen not yet implemented");
    }

}  // namespace spore
