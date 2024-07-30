#ifndef SPANREED_BENCHMARK_DESTINATION_MESSAGES_H
#define SPANREED_BENCHMARK_DESTINATION_MESSAGES_H

#include <cstdint>
#include <string>
#include <variant>
#include <vector>

namespace spanreed::benchmark {

struct DestinationMessageHeader {
  std::uint32_t magic_header;
  std::uint32_t client_id;
  std::uint16_t message_id;
  std::uint16_t last_seen_client_message_id;
  std::uint32_t ack_field;
};

enum class DestinationMessageType {
  UNKNOWN,
  ConnectionVerdict,
  DisconnectClient,
  Pong,
  Stats,
};

struct ConnectClientVerdict {
  bool verdict;
};

struct PongMessage {
  std::uint64_t client_send_ts;
  std::uint64_t proxy_recv_client_ts;
  std::uint64_t proxy_forward_client_ts;
  std::uint64_t server_recv_ts;
  std::uint64_t server_send_ts;
  std::string payload;
};

struct DestinationStats {
  std::uint32_t last_seen_message_id;
  std::uint32_t received_messages;
  std::uint32_t dropped_messages;
  std::uint32_t out_of_order_messages;
};

struct DestinationMessage {
  DestinationMessageHeader header{};
  DestinationMessageType message_type = DestinationMessageType::UNKNOWN;
  std::variant<std::monostate, ConnectClientVerdict, PongMessage,
               DestinationStats>
      body = std::monostate{};
};

std::vector<std::uint8_t> serialize_destination_message(
    const DestinationMessage& msg);

}  // namespace spanreed::benchmark

#endif
