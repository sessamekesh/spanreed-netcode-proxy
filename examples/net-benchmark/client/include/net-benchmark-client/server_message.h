#ifndef SPANREED_BENCHMARK_CLIENT_SERVER_MESSAGE_H
#define SPANREED_BENCHMARK_CLIENT_SERVER_MESSAGE_H

#include <cstdint>
#include <optional>
#include <string>
#include <variant>

namespace spanreed::benchmark {

struct ServerMessageHeader {
  std::uint32_t magic_header;
  std::uint32_t client_id;
  std::uint16_t message_id;
  std::uint16_t last_seen_client_message_id;
  std::uint32_t ack_field;
};

enum class ServerMessageType {
  UNKNOWN,
  ConnectionVerdict,
  DisconnectRequest,
  Pong,
  ServerNetStats,
};

struct ConnectionVerdict {
  bool verdict;
};

struct Pong {
  std::uint64_t client_send_ts;
  std::uint64_t client_recv_ts;
  std::uint64_t proxy_recv_client_ts;
  std::uint64_t proxy_forward_client_ts;
  std::uint64_t server_recv_ts;
  std::uint64_t server_send_ts;
  std::uint64_t proxy_recv_destination_ts;
  std::uint64_t proxy_forward_destination_ts;
  std::string payload;
};

struct ServerNetStats {
  std::uint32_t last_seen_message_id;
  std::uint32_t received_messages;
  std::uint32_t dropped_messages;
  std::uint32_t out_of_order_messages;
};

struct ServerMessage {
  ServerMessageHeader header{};
  ServerMessageType message_type = ServerMessageType::UNKNOWN;
  std::variant<std::monostate, ConnectionVerdict, Pong, ServerNetStats> body =
      std::monostate{};
};

std::optional<ServerMessage> parse_server_message(const std::uint8_t* buff,
                                                  std::size_t buff_len);

}  // namespace spanreed::benchmark

#endif
