// Must go before winsock
#include <spanreed_messages/proxy_destination_message_generated.h>
// ...

#include <WinSock2.h>
#include <concurrentqueue.h>
#include <messages/client_message_generated.h>
#include <messages/server_message_generated.h>

#include <chrono>
#include <iostream>
#include <thread>
#include <unordered_map>

struct ConnectedClient {
  sockaddr_in spanreed_client_addr{};  // Will probably be the same for all
                                       // connected clients...
  std::uint32_t spanreed_client_id{};

  std::uint64_t last_send_time{};
};

struct DrawnDot {
  float x;
  float y;
  float t;
};

struct WorldState {
  std::vector<DrawnDot> dots;
};

void handle_recv_message(SOCKET sock, char* msg, int msglen);

int main() {
  std::unordered_map<std::uint32_t, ConnectedClient> connected_clients{};

  auto client_message_queue = std::make_unique<
      moodycamel::ConcurrentQueue<HelloSpanreed::ClientMessage>>();

  std::cout << "-------- Hello Spanreed UDP Server --------" << std::endl;

  auto tp_start = std::chrono::high_resolution_clock::now();
  auto get_timestamp = [tp_start]() {
    auto now = std::chrono::high_resolution_clock::now();
    return std::chrono::duration_cast<std::chrono::microseconds>(now - tp_start)
        .count();
  };

  WSADATA wsa{};
  if (WSAStartup(MAKEWORD(2, 2), &wsa) == 0) {
    std::cerr << "Failed to startup WinSock - error code " << WSAGetLastError()
              << std::endl;
    return -1;
  }

  SOCKET server_socket{};
  if ((server_socket = socket(AF_INET, SOCK_DGRAM, 0)) == INVALID_SOCKET) {
    std::cerr << "Could not create server socket: " << WSAGetLastError()
              << std::endl;
    return -1;
  }

  sockaddr_in server_addr{};
  server_addr.sin_family = AF_INET;
  server_addr.sin_addr.s_addr = INADDR_ANY;
  server_addr.sin_port = htons(30001);

  if (bind(server_socket, (sockaddr*)&server_addr, sizeof(server_addr)) ==
      SOCKET_ERROR) {
    std::cerr << "Failed to bind socket, code: " << WSAGetLastError()
              << std::endl;
    return -1;
  }

  std::cout << "Listening on port 30001!" << std::endl;

  bool bRunning = true;
  std::thread t([&bRunning, &server_socket]() {
    while (bRunning) {
      constexpr size_t BUFLEN = 1024;
      char message[BUFLEN] = {};

      int message_len;
      int slen = sizeof(sockaddr_in);
      sockaddr_in recv_addr{};
      if ((message_len = recvfrom(server_socket, message, BUFLEN, 0x0,
                                  (sockaddr*)&recv_addr, &slen)) ==
          SOCKET_ERROR) {
        std::cout << "recvfrom() failed with error code: " << WSAGetLastError()
                  << std::endl;
        return -1;
      }

      std::cout << "Received packet (" << message_len << " bytes) from "
                << inet_ntoa(recv_addr.sin_addr) << " "
                << ntohs(recv_addr.sin_port) << std::endl;

      handle_recv_message(server_socket, message, message_len);
      // TODO (sessamekesh): C++ parser for various messages, connection flow
      // for existing connections, otherwise broadcast to connected clients
    }
  });

  while (bRunning) {
    // TODO (sessamekesh): Handle incoming messages
    //  - Add new dots to sim
    //  - Append list of messages to broadcast
    // TODO (sessamekesh): Lifetime expire dots
    // TODO (sessamekesh): Broadcast messages to clients
  }

  t.join();

  return 0;
}

void handle_recv_message(SOCKET sock, char* msg, int msglen) {}
