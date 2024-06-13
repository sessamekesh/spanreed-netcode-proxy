#include <WinSock2.h>

#include <iostream>
#include <unordered_map>

#include "../../../external/parsers/cpp/spanreedmsg.hpp";

#include <messages/server_message_generated.h>
#include <messages/client_message_generated.h>

struct ConnectedClient {
  sockaddr_in spanreed_client_addr{};  // Will probably be the same for all
                                       // connected clients...
  std::uint32_t spanreed_client_id{};
};

struct DrawnDot {
  float x;
  float y;
  float t;
};

struct WorldState {
  std::vector<DrawnDot> dots;
};

int main() {
  std::unordered_map<std::uint32_t, ConnectedClient> connected_clients{};

  std::cout << "-------- Hello Spanreed UDP Server --------" << std::endl;

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

  while (true) {
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

    // TODO (sessamekesh): C++ parser for various messages, connection flow for
    // existing connections, otherwise broadcast to connected clients
  }

  return 0;
}
