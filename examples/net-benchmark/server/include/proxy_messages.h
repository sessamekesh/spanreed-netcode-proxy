#ifndef SPANREED_BENCHMARK_PROXY_MESSAGES_H
#define SPANREED_BENCHMARK_PROXY_MESSAGES_H

#include <cstdint>
#include <optional>
#include <variant>
#include <vector>

namespace spanreed::benchmark {

// TODO (sessamekesh): Sequence IDs

struct ConnectClientRequest {
  std::uint32_t client_id;
};

struct DisconnectClientRequest {
  std::uint32_t client_id;
};

struct PingMessage {
  std::uint32_t client_id;
  std::uint64_t client_send_ts;
  std::uint64_t proxy_recv_client_ts;
  std::uint64_t proxy_forward_client_ts;
  std::uint64_t server_recv_ts;
};

struct HashPayloadMessage {
  std::uint32_t client_id;
  std::vector<std::uint8_t> buffer_to_hash;  // XOR
  std::uint32_t return_buffer_size;
};

typedef std::variant<ConnectClientRequest, DisconnectClientRequest, PingMessage,
                     HashPayloadMessage>
    ProxyMessage;

std::optional<ProxyMessage> parse_proxy_message(std::uint8_t* buffer,
                                                std::size_t buffer_len);

}  // namespace spanreed::benchmark

#endif
