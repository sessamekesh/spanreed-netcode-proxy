#ifndef SPANREED_BENCHMARK_CLIENT_MESSAGE_H
#define SPANREED_BENCHMARK_CLIENT_MESSAGE_H

#include <cstdint>
#include <optional>
#include <string>
#include <variant>
#include <vector>

namespace spanreed::benchmark {

struct ClientMessageHeader {
  std::uint32_t magic_header;
  std::uint32_t client_id;
  std::uint16_t message_id;
  std::uint16_t last_seen_server_message_id;
  std::uint32_t ack_field;
};

enum class ClientMessageType {
  UNKNOWN,
  ConnectClient,
  DisconnectClient,
  Ping,
  GetStats,
};

struct PingMessage {
  std::uint64_t client_send_ts;
  std::string payload;
};

struct ClientMessage {
  ClientMessageHeader header{};
  ClientMessageType message_type = ClientMessageType::UNKNOWN;
  std::variant<PingMessage, std::monostate> body = std::monostate{};
};

std::optional<std::vector<std::uint8_t>> serialize_client_message(
    const ClientMessage& msg);

}  // namespace spanreed::benchmark

#endif
