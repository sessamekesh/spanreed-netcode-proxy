#ifdef _WIN32
#include <WS2tcpip.h>
#include <WinSock2.h>
#else
#include <arpa/inet.h>
#include <errno.h>
#include <netinet/in.h>
#include <sys/socket.h>
#endif

#include <shared_mutex>
#include <string>
#include <unordered_map>

#ifdef _WIN32
#define SOCKTYPE SOCKET
#define ISSOCKERR(x) (x) == INVALID_SOCKET
#else
#define SOCKTYPE int
#define ISSOCKERR(x) (x) < 0
#endif

#include <net-benchmark-server/udp_server.h>
#include <spdlog/sinks/stdout_color_sinks.h>

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
  FormatMessage(FORMAT_MESSAGE_FROM_SYSTEM, NULL, WSAGetLastError(),
                LANG_SYSTEM_DEFAULT, &errmsg[0],
                static_cast<DWORD>(errmsg.length()), NULL);
  return errmsg;
}
#else
static std::string get_last_socket_error() { return strerror(errno); }
#endif

class ScopeLogger {
 public:
  ScopeLogger(std::shared_ptr<spdlog::logger> log, std::string onstart,
              std::string onexit)
      : log(log), onexit_(onexit) {
    log->info(onstart);
  }

  ~ScopeLogger() { log->info(onexit_); }

 private:
  std::shared_ptr<spdlog::logger> log;
  std::string onexit_;
};

}  // namespace

namespace spanreed::benchmark {

struct ConnectedClient {
  sockaddr_in client_addr{};
};

UdpServer::UdpServer(
    std::uint16_t port,
    moodycamel::ConcurrentQueue<ProxyMessage>* incoming_message_queue,
    Timer* timer)
    : port_(port),
      is_running_(false),
      log(spdlog::stdout_color_mt("UdpServer")),
      timer_(timer),
      outgoing_messages_(64),
      incoming_messages_(incoming_message_queue),
      on_exit_(nullptr) {}

void UdpServer::QueueMessage(DestinationMessage msg) {
  outgoing_messages_.enqueue(std::move(msg));
}

bool UdpServer::Start() {
#ifdef _WIN32
  WSADATA wsa{};
  if (WSAStartup(MAKEWORD(2, 2), &wsa) != 0) {
    log->error("Failed to startup WinSock: {}", get_last_socket_error());
    return false;
  }
#endif

  SOCKTYPE server_socket{};
  if (ISSOCKERR(server_socket = socket(AF_INET, SOCK_DGRAM, 0))) {
    log->error("Could not create server socket: {}", get_last_socket_error());
    return false;
  }

  timeval tv{};
  tv.tv_sec = 5000;
  setsockopt(server_socket, SOL_SOCKET, SO_RCVTIMEO,
             reinterpret_cast<char*>(&tv), sizeof(timeval));

  sockaddr_in server_addr{};
  server_addr.sin_family = AF_INET;
  server_addr.sin_addr.s_addr = INADDR_ANY;
  server_addr.sin_port = htons(port_);

  if (ISSOCKERR(
          bind(server_socket, (sockaddr*)&server_addr, sizeof(sockaddr_in)))) {
    log->error("Failed to bind socket to port {}: {}", port_,
               get_last_socket_error());
    return false;
  }

  log->info("Listening on port {}", port_);

  is_running_ = true;
  bool is_success = true;

  std::shared_mutex mut_clients;
  std::unordered_map<
      /* client_id= */ std::uint32_t, ConnectedClient>
      clients;

  std::thread listen_thread([this, &is_success, &server_socket, &mut_clients,
                             &clients]() {
    ::ScopeLogger sl(log, "Starting listen thread", "Stopped listen thread");

    constexpr std::size_t BUFLEN = 10240;
    int allowable_remaining_errors = 100;
    char message[BUFLEN] = {};
    while (is_running_) {
      int message_length = 0;
      socklen_t slen = sizeof(sockaddr_in);
      sockaddr_in recv_addr{};
      if (ISSOCKERR(message_length =
                        recvfrom(server_socket, message, BUFLEN, 0x00,
                                 (struct sockaddr*)&recv_addr, &slen))) {
#ifdef _WIN32
        if (WSAGetLastError() == WSAETIMEDOUT) {
#else
        if ((errno != EAGAIN) && (errno != EWOULDBLOCK)) {
#endif
          log->debug("No message received for 5 seconds");
          continue;
        }

        log->error("recvfrom() failed with error: {}", get_last_socket_error());
        allowable_remaining_errors--;
        if (allowable_remaining_errors < 0) {
          log->error("Too many receive errors, exiting");
          is_running_ = false;
          is_success = false;
          return;
        }
      }

      // handle recv message with buffer contents
      auto parsed_msg = parse_proxy_message(
          reinterpret_cast<std::uint8_t*>(message), message_length);
      if (!parsed_msg.has_value()) {
        continue;
      }
      if (parsed_msg->message_type == ProxyMessageType::Ping) {
        auto& pingmsg = std::get<PingMessage>(parsed_msg->body);
        pingmsg.server_recv_ts = timer_->get_time();
      }

      if (parsed_msg->message_type == ProxyMessageType::ConnectClient) {
        log->info("Connecting client {}", parsed_msg->header.client_id);
        std::lock_guard lg(mut_clients);
        ConnectedClient cc{};
        memcpy(&cc.client_addr, &recv_addr, sizeof(recv_addr));
        clients[parsed_msg->header.client_id] = std::move(cc);
      } else if (parsed_msg->message_type ==
                 ProxyMessageType::DisconnectClient) {
        log->info("Disconnecting client {} because of client request",
                  parsed_msg->header.client_id);
        std::lock_guard lg(mut_clients);
        clients.erase(parsed_msg->header.client_id);
      }

      incoming_messages_->enqueue(*std::move(parsed_msg));
    }
  });

  std::thread message_thread([this, &is_success, &server_socket, &mut_clients,
                              &clients]() {
    ::ScopeLogger sl(log, "Starting message thread", "Stopped message thread");

    while (is_running_) {
      DestinationMessage msg{};
      if (!outgoing_messages_.try_dequeue(msg)) {
        continue;
      }

      if (msg.message_type == DestinationMessageType::Pong) {
        auto& pong = std::get<PongMessage>(msg.body);
        pong.server_send_ts = timer_->get_time();
      }

      auto buf = serialize_destination_message(msg);
      if (buf.size() == 0) {
        continue;
      }

      sockaddr_in addr{};
      {
        std::shared_lock sl(mut_clients);

        auto client_it = clients.find(msg.header.client_id);
        if (client_it == clients.end()) {
          continue;
        }

        memcpy(&addr, &client_it->second.client_addr, sizeof(sockaddr_in));
      }

      sendto(server_socket, reinterpret_cast<const char*>(&buf[0]),
             static_cast<int>(buf.size()), 0x00,
             reinterpret_cast<const sockaddr*>(&addr), sizeof(addr));

      if (msg.message_type == DestinationMessageType::DisconnectClient) {
        log->info("Disconnecting client {}", msg.header.client_id);
        std::lock_guard lg(mut_clients);
        clients.erase(msg.header.client_id);
      } else if (msg.message_type ==
                     DestinationMessageType::ConnectionVerdict &&
                 !std::get<ConnectClientVerdict>(msg.body).verdict) {
        log->info("Rejecting client {}", msg.header.client_id);
        std::lock_guard lg(mut_clients);
        clients.erase(msg.header.client_id);
      }
    }
  });

  listen_thread.join();
  message_thread.join();

  {
    std::lock_guard l(mut_clients);
    log->info(
        "Graceful shutdown initiated - sending disconnect message to remaining "
        "{} clients",
        clients.size());
    for (auto& [client_id, client] : clients) {
      DestinationMessage msg{};
      msg.header.client_id = client_id;
      msg.message_type = DestinationMessageType::DisconnectClient;
      msg.header.magic_header = 0x5350414E;
      auto buf = serialize_destination_message(msg);
      if (buf.size() > 0) {
        sendto(server_socket, reinterpret_cast<const char*>(&buf[0]),
               static_cast<int>(buf.size()), 0x00,
               reinterpret_cast<const sockaddr*>(&client.client_addr),
               sizeof(client.client_addr));
      }
    }
  }

  log->info("Shutting down socket listener");

#ifdef _WIN32
  if (closesocket(server_socket) != 0) {
    log->error("Failed to close socket: {}", get_last_socket_error());
    is_success = false;
  }
  if (WSACleanup() != 0) {
    log->error("Failed to cleanup WSA: {}", get_last_socket_error());
    is_success = false;
  }
#else
  if (close(server_socket) != 0) {
    log->error("Failed to close socket: {}", get_last_socket_error());
    is_success = false;
  }
#endif

  log->info("UDP server shutdown complete (success: {})", is_success);
  return is_success;
}

void UdpServer::Stop() {
  log->info("Marking UDP server for stop");
  is_running_ = false;
  if (on_exit_) {
    on_exit_();
  }
}

// Stuff
}  // namespace spanreed::benchmark
