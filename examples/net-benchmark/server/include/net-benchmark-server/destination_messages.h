#ifndef SPANREED_BENCHMARK_DESTINATION_MESSAGES_H
#define SPANREED_BENCHMARK_DESTINATION_MESSAGES_H

#include <cstdint>
#include <string>

namespace spanreed::benchmark {

// TODO (sessamekesh): Add ack headers

struct ConnectClientVerdict {
  std::uint32_t client_id;
  bool verdict;
};

struct PongMessage {
  std::uint32_t client_id;
  std::uint64_t client_send_ts;
  std::uint64_t proxy_recv_client_ts;
  std::uint64_t proxy_forward_client_ts;
  std::uint64_t server_recv_ts;
  std::uint64_t server_send_ts;
};

struct HashPayloadResponse {
  std::uint32_t client_id;
  std::string hash_result;
};

}  // namespace spanreed::benchmark

#endif
