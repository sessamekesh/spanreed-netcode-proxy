#ifndef SPANREED_BENCHMARK_PROXY_MESSAGES_H
#define SPANREED_BENCHMARK_PROXY_MESSAGES_H

#include <cstdint>
#include <optional>
#include <string>
#include <variant>
#include <vector>

namespace spanreed::benchmark {

struct ProxyMessageHeader {
  std::uint32_t magic_header;
  std::uint32_t client_id;
  std::uint16_t message_id;
  std::uint16_t ack_field;

  static std::size_t HEADER_SIZE;
};

enum class ProxyMessageType {
  ConnectClient,
  DisconnectClient,
  Ping,
};

struct PingMessage {
  std::uint64_t client_send_ts;
  std::uint64_t proxy_recv_client_ts;
  std::uint64_t proxy_forward_client_ts;
  std::uint64_t server_recv_ts;
  std::string payload;
};

typedef std::variant<std::monostate, PingMessage> ProxyMessageBody;

struct ProxyMessage {
  ProxyMessageHeader header;
  ProxyMessageType message_type;
  ProxyMessageBody body;
};

std::optional<ProxyMessage> parse_proxy_message(std::uint8_t* buffer,
                                                std::size_t buffer_len);

}  // namespace spanreed::benchmark

#endif
