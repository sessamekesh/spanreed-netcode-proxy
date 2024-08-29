#ifdef _WIN32
#include <WS2tcpip.h>
#include <WinSock2.h>
#else
#include <arpa/inet.h>
#include <errno.h>
#include <netdb.h>
#include <netinet/in.h>
#include <sys/socket.h>
#endif

#include <iostream>

#include "netclient.h"

#ifdef _WIN32
#define SOCKTYPE SOCKET
#define ISSOCKERR(x) (x) == SOCKET_ERROR
#else
#define SOCKTYPE int
#define ISSOCKERR(x) (x) < 0
#endif

namespace {
static bool cmp_addr(const sockaddr_in& a, const sockaddr_in& b) {
  if (a.sin_port != b.sin_port) return false;
  if (a.sin_family != b.sin_family) return false;

  if (memcmp(&a.sin_addr, &b.sin_addr, sizeof(a.sin_addr)) != 0) return false;

  return true;
}

#ifdef _WIN32
static std::string get_last_socket_error() {
  std::string errmsg;
  errmsg.resize(256);
  auto code = WSAGetLastError();
  FormatMessage(FORMAT_MESSAGE_FROM_SYSTEM, NULL, code, LANG_SYSTEM_DEFAULT,
                &errmsg[0], static_cast<DWORD>(errmsg.length()), NULL);
  return errmsg;
}
#else
static std::string get_last_socket_error() { return strerror(errno); }
#endif
}  // namespace

namespace spanreed::benchmark {

NetClient::NetClient(std::uint16_t client_port, std::string uri, Timer* timer)
    : client_port_(client_port),
      uri_(uri),
      timer_(timer),
      is_running_(false),
      outgoing_messages_(32u),
      incoming_messages_(32u) {}

bool NetClient::Start() {
  bool is_success = true;

#ifdef _WIN32
  WSADATA wsa{};
  if (WSAStartup(MAKEWORD(2, 2), &wsa) != 0) {
    return false;
  }
#endif

  sockaddr_in server_addr{};
  server_addr.sin_family = AF_INET;

  std::size_t colon_loc = uri_.find_first_of(":");
  std::string host;
  std::uint16_t port;
  if (colon_loc == std::string::npos) {
    host = uri_;
    port = 80;
  } else {
    host = uri_.substr(0, colon_loc);
    std::string portstring = uri_.substr(colon_loc + 1);
    try {
      port = std::stoul(portstring);
    } catch (std::invalid_argument e) {
      return false;
    }
  }
  server_addr.sin_port = htons(port);

#ifdef _WIN32
  hostent* remoteHost = nullptr;
  if (isalpha(host[0])) {
    remoteHost = gethostbyname(host.c_str());
  } else {
    server_addr.sin_addr.s_addr = inet_addr(host.c_str());
    if (server_addr.sin_addr.s_addr == INADDR_NONE) {
      return false;
    } else {
      remoteHost = gethostbyaddr(reinterpret_cast<char*>(&server_addr.sin_addr),
                                 4, AF_INET);
    }
  }
  if (remoteHost == nullptr) {
    return false;
  }

  if (remoteHost->h_addrtype != AF_INET) {
    return false;
  }

  if (remoteHost->h_addr_list[0] == nullptr) {
    return false;
  }

  server_addr.sin_addr.s_addr =
      *reinterpret_cast<ULONG*>(remoteHost->h_addr_list[0]);

#else
  hostent* remoteHost = nullptr;
  if (isalpha(host[0])) {
    remoteHost = gethostbyname(host.c_str());
  } else {
    server_addr.sin_addr.s_addr = inet_addr(host.c_str());
    if (server_addr.sin_addr.s_addr == INADDR_NONE) {
      return false;
    } else {
      remoteHost = gethostbyaddr(reinterpret_cast<char*>(&server_addr.sin_addr),
                                 4, AF_INET);
    }
  }
  if (remoteHost == nullptr) {
    return false;
  }

  if (remoteHost->h_addrtype != AF_INET) {
    return false;
  }

  if (remoteHost->h_addr_list[0] == nullptr) {
    return false;
  }

  server_addr.sin_addr.s_addr =
      *reinterpret_cast<unsigned long*>(remoteHost->h_addr_list[0]);
#endif

  SOCKTYPE client_socket{};
  if (ISSOCKERR(client_socket = socket(AF_INET, SOCK_DGRAM, 0))) {
    return false;
  }

#ifdef _WIN32
  DWORD mstimeout = 5000;
  setsockopt(client_socket, SOL_SOCKET, SO_RCVTIMEO,
             reinterpret_cast<char*>(&mstimeout), sizeof(DWORD));
#else
  timeval tv{};
  tv.tv_sec = 5;
  setsockopt(client_socket, SOL_SOCKET, SO_RCVTIMEO,
             reinterpret_cast<char*>(&tv), sizeof(timeval));
#endif

  sockaddr_in client_addr{};
  client_addr.sin_family = AF_INET;
  client_addr.sin_addr.s_addr = INADDR_ANY;
  client_addr.sin_port = htons(client_port_);

  if (ISSOCKERR(
          bind(client_socket, (sockaddr*)&client_addr, sizeof(sockaddr_in)))) {
    return false;
  }

  is_running_ = true;

  bool is_disconnected = false;

  std::thread listen_thread([this, &is_success, &client_socket, &server_addr] {
    constexpr std::size_t BUFLEN = 10240;
    int allowable_remaining_errors = 100;
    char message[BUFLEN] = {};

    while (is_running_) {
      int message_length = 0;
      socklen_t slen = sizeof(sockaddr_in);
      sockaddr_in recv_addr{};

      if (ISSOCKERR(message_length =
                        recvfrom(client_socket, message, BUFLEN, 0x00,
                                 (struct sockaddr*)&recv_addr, &slen))) {

#ifdef _WIN32
        if (WSAGetLastError() == WSAETIMEDOUT) {
#else
        if ((errno == EAGAIN) || (errno == EWOULDBLOCK)) {
#endif
          continue;
        }

        std::cerr << "recvfrom error: " << get_last_socket_error() << std::endl;
        allowable_remaining_errors--;
        if (allowable_remaining_errors < 0) {
          is_running_ = false;
          is_success = false;
          return;
        }
        continue;
      }

      if (!::cmp_addr(server_addr, recv_addr)) {
        continue;
      }

      auto parsed_msg = parse_server_message(
          reinterpret_cast<const std::uint8_t*>(message), message_length);
      if (!parsed_msg.has_value()) {
        continue;
      }

      if (parsed_msg->message_type == ServerMessageType::Pong) {
        auto& pongmsg = std::get<Pong>(parsed_msg->body);
        pongmsg.client_recv_ts = timer_->get_time();
      }

      if (parsed_msg->message_type == ServerMessageType::DisconnectRequest) {
        is_running_ = false;
      }

      if (parsed_msg->message_type == ServerMessageType::ConnectionVerdict) {
        auto& verdict = std::get<ConnectionVerdict>(parsed_msg->body);
        if (!verdict.verdict) {
          is_running_ = false;
        }
      }

      incoming_messages_.enqueue(*std::move(parsed_msg));
    }
  });

  std::thread message_thread([this, &client_socket, &server_addr] {
    while (is_running_) {
      ClientMessage msg{};
      if (!outgoing_messages_.try_dequeue(msg)) {
        continue;
      }

      if (msg.message_type == ClientMessageType::Ping) {
        auto& ping = std::get<PingMessage>(msg.body);
        ping.client_send_ts = timer_->get_time();
      }

      auto bufopt = serialize_client_message(msg);
      if (!bufopt.has_value()) {
        continue;
      }
      const auto& buf = *bufopt;

      if (ISSOCKERR(sendto(client_socket,
                           reinterpret_cast<const char*>(&buf[0]),
                           static_cast<int>(buf.size()), 0x00,
                           reinterpret_cast<const sockaddr*>(&server_addr),
                           sizeof(server_addr)))) {
        std::cout << "Failed to send message: " << ::get_last_socket_error()
                  << std::endl;
      }

      if (msg.message_type == ClientMessageType::DisconnectClient) {
        is_running_ = false;
      }
    }
  });

  listen_thread.join();
  message_thread.join();

  std::cout << "Shutting down NetClient..." << std::endl;
  return is_success;
}

void NetClient::Stop() { is_running_ = false; }

void NetClient::QueueMessage(ClientMessage msg) {
  outgoing_messages_.enqueue(msg);
}

std::optional<ServerMessage> NetClient::ReadMessage() {
  ServerMessage msg{};
  if (incoming_messages_.try_dequeue(msg)) {
    return msg;
  }

  return std::nullopt;
}

}  // namespace spanreed::benchmark
