#ifndef SPANREED_BENCHMARK_CLIENT_APP_H
#define SPANREED_BENCHMARK_CLIENT_APP_H

#include <net-benchmark-client/client_message.h>
#include <net-benchmark-client/server_message.h>

#include <cstdint>

namespace spanreed::benchmark {

struct RttSample {
  /* Total time from when client sent message to receive response */
  std::uint64_t ClientRTT;

  /* Total time from when proxy sent message to receive response */
  std::uint64_t ProxyRTT;

  /* Time spent processing at the destination */
  std::uint64_t DestProcessTime;

  /* Time spent processing at the proxy level */
  std::uint64_t ProxyProcessTime;

  /* Time spent on the network between client and proxy */
  std::uint64_t ClientProxyNetTime;

  /* Time spent on the network between the proxy and destination */
  std::uint64_t ProxyDestNetTime;
};

struct Aggregate {
  double Min;
  double Max;
  double StdDev;

  double P50;
  double P90;
  double P95;
  double P99;
};

struct RttAggregates {
  static std::optional<RttAggregates> From(
      const std::vector<RttSample>& samples);

  Aggregate ClientRTT;
  Aggregate ProxyRTT;
  Aggregate DestProcessTime;
  Aggregate ProxyProcessTime;
  Aggregate ClientProxyNetTime;
  Aggregate ProxyDestNetTime;
};

struct ExperimentResults {
  std::uint32_t client_sent_messages;
  // std::uint32_t server_sent_messages;

  std::uint32_t server_recv_messages;
  std::uint32_t client_recv_messages;

  std::uint32_t server_out_of_order_messages;
  std::uint32_t client_out_of_order_messages;

  std::uint32_t client_dropped_messages;
  std::uint32_t server_dropped_messages;

  std::vector<RttSample> rtt_measurements;
};

class BenchmarkApp {
 public:
  BenchmarkApp();

  void start_experiment(std::string dest_url, std::uint32_t payload_size,
                        std::uint32_t ping_count);

  void add_server_message(ServerMessage msg);
  std::optional<ClientMessage> get_client_message();
  std::optional<ExperimentResults> get_results();
  bool is_running() const;

 private:
  bool is_running_;
  std::string dest_url_;
  std::uint32_t payload_size_;
  std::uint32_t remaining_ping_count_;

  bool is_connected_;
  bool needs_stats_;

  std::vector<RttSample> rtt_samples_;
  ServerNetStats net_stats_;

  std::uint32_t last_seen_message_id_;
  std::uint32_t ack_field_;
  std::uint32_t next_message_id_;

  std::uint32_t send_ct_;
  std::uint32_t recv_ct_;
  std::uint32_t drop_ct_;
  std::uint32_t ooo_ct_;
};

}  // namespace spanreed::benchmark

#endif
