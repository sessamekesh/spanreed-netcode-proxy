#include <net-benchmark-client/benchmark_app.h>

#include <algorithm>
#include <iostream>

namespace {
std::string kPayloadString = "spanreed-benchmark! ";

double Variance(const std::vector<spanreed::benchmark::RttSample>& samples,
                std::size_t offset) {
  int size = static_cast<int>(samples.size());

  double sum = 0.;
  for (int i = 0; i < size; i++) {
    double sample = static_cast<double>(
        *(reinterpret_cast<const std::uint64_t*>(&samples[i]) + offset / 8));
    sum += sample;
  }

  double mean = sum / static_cast<double>(size);

  double sqDiff = 0.;
  for (int i = 0; i < size; i++) {
    double sample = static_cast<double>(
        *(reinterpret_cast<const std::uint64_t*>(&samples[i]) + offset / 8));
    sqDiff += (sample - mean) * (sample - mean);
  }

  return sqDiff / static_cast<double>(size);
}

double StdDev(const std::vector<spanreed::benchmark::RttSample>& samples,
              std::size_t offset) {
  return std::sqrt(::Variance(samples, offset));
}

spanreed::benchmark::Aggregate GetAggregate(
    const std::vector<spanreed::benchmark::RttSample>& samples,
    std::size_t offset) {
  spanreed::benchmark::Aggregate agg{};

  std::vector<double> dsamples;
  for (int i = 0; i < samples.size(); i++) {
    double sample = static_cast<double>(
        *(reinterpret_cast<const std::uint64_t*>(&samples[i]) + offset / 8));
    dsamples.push_back(sample);
  }

  std::sort(dsamples.begin(), dsamples.end(), std::less<double>());

  int p50_idx = static_cast<int>(dsamples.size()) * 50 / 100;
  int p90_idx = static_cast<int>(dsamples.size()) * 90 / 100;
  int p95_idx = static_cast<int>(dsamples.size()) * 95 / 100;
  int p99_idx = static_cast<int>(dsamples.size()) * 99 / 100;

  agg.Min = dsamples[0];
  agg.Max = dsamples[dsamples.size() - 1];
  agg.StdDev = ::StdDev(samples, offset);
  agg.P50 = dsamples[p50_idx];
  agg.P90 = dsamples[p90_idx];
  agg.P95 = dsamples[p95_idx];
  agg.P99 = dsamples[p99_idx];

  return agg;
}

}  // namespace

namespace spanreed::benchmark {

std::optional<RttAggregates> RttAggregates::From(
    const std::vector<RttSample>& samples) {
  if (samples.size() == 0) return std::nullopt;

  RttAggregates aggs{};

  aggs.ClientRTT = ::GetAggregate(samples, offsetof(RttSample, ClientRTT));
  aggs.ProxyRTT = ::GetAggregate(samples, offsetof(RttSample, ProxyRTT));
  aggs.DestProcessTime =
      ::GetAggregate(samples, offsetof(RttSample, DestProcessTime));
  aggs.ProxyProcessTime =
      ::GetAggregate(samples, offsetof(RttSample, ProxyProcessTime));
  aggs.ClientProxyNetTime =
      ::GetAggregate(samples, offsetof(RttSample, ClientProxyNetTime));
  aggs.ProxyDestNetTime =
      ::GetAggregate(samples, offsetof(RttSample, ProxyDestNetTime));

  return aggs;
}

BenchmarkApp::BenchmarkApp()
    : is_running_(false),
      payload_size_(0ul),
      remaining_ping_count_(0ul),
      is_connected_(false),
      net_stats_(ServerNetStats{}),
      last_seen_message_id_(0ul),
      ack_field_(0ul),
      next_message_id_(0ul),
      send_ct_(0ul),
      recv_ct_(0ul),
      drop_ct_(0ul),
      ooo_ct_(0ul),
      needs_stats_(true) {}

void BenchmarkApp::start_experiment(std::uint32_t payload_size,
                                    std::uint32_t ping_count) {
  payload_size_ = payload_size;
  remaining_ping_count_ = ping_count;
  is_connected_ = false;
  rtt_samples_ = std::vector<RttSample>{};
  net_stats_ = ServerNetStats{};
  last_seen_message_id_ = 0ul;
  ack_field_ = ~0x0;
  next_message_id_ = 0ul;
  send_ct_ = 0ul;
  recv_ct_ = 0ul;
  drop_ct_ = 0ul;
  ooo_ct_ = 0ul;
  needs_stats_ = true;
  is_running_ = true;
}

void BenchmarkApp::add_server_message(ServerMessage msg) {
  switch (msg.message_type) {
    case ServerMessageType::ConnectionVerdict: {
      auto& body = std::get<ConnectionVerdict>(msg.body);
      if (!body.verdict) {
        std::cerr << "Verdict was false from server" << std::endl;
        is_running_ = false;
        return;
      }
      is_connected_ = true;
    } break;
    case ServerMessageType::DisconnectRequest: {
      is_running_ = false;
      is_connected_ = false;
      std::cerr << "Got disconnect from server" << std::endl;
      return;
    } break;
    case ServerMessageType::Pong: {
      const auto& pong = std::get<Pong>(msg.body);

      RttSample sample{};
      if (pong.client_send_ts < pong.client_recv_ts) {
        sample.ClientRTT = pong.client_recv_ts - pong.client_send_ts;
      }

      if (pong.proxy_recv_client_ts != 0ull &&
          pong.proxy_forward_destination_ts != 0ull &&
          pong.proxy_recv_client_ts < pong.proxy_forward_destination_ts) {
        sample.ProxyRTT =
            pong.proxy_forward_destination_ts - pong.proxy_recv_client_ts;
      }

      if (pong.server_recv_ts < pong.server_send_ts) {
        sample.DestProcessTime = pong.server_send_ts - pong.server_recv_ts;
      }

      if (pong.proxy_recv_client_ts != 0ull &&
          pong.proxy_recv_destination_ts != 0ull &&
          pong.proxy_forward_client_ts != 0ull &&
          pong.proxy_forward_destination_ts != 0ull &&
          pong.proxy_recv_client_ts < pong.proxy_forward_client_ts &&
          pong.proxy_recv_destination_ts < pong.proxy_forward_destination_ts) {
        sample.ProxyProcessTime =
            (pong.proxy_forward_destination_ts -
             pong.proxy_recv_destination_ts) +
            (pong.proxy_forward_client_ts - pong.proxy_recv_client_ts);
      }

      if (sample.ProxyRTT != 0ull && sample.ClientRTT > sample.ProxyRTT) {
        sample.ClientProxyNetTime = sample.ClientRTT - sample.ProxyRTT;
      } else if (sample.ProxyRTT == 0ull &&
                 sample.ClientRTT > sample.DestProcessTime) {
        sample.ClientProxyNetTime = sample.ClientRTT - sample.DestProcessTime;
      }

      if (sample.ProxyRTT != 0ull && sample.ProxyProcessTime != 0ull &&
          sample.DestProcessTime != 0ull &&
          sample.ProxyRTT >
              (sample.ProxyProcessTime + sample.DestProcessTime)) {
        sample.ProxyDestNetTime = sample.ProxyRTT - (sample.ProxyProcessTime +
                                                     sample.DestProcessTime);
      }

      rtt_samples_.push_back(sample);
    } break;
    case ServerMessageType::ServerNetStats: {
      needs_stats_ = false;
      net_stats_ = std::get<ServerNetStats>(msg.body);
    } break;
  }

  //
  // Process:
  recv_ct_++;
  int shift = msg.header.message_id - msg.header.last_seen_client_message_id;
  if (shift < 0) {
    // Out of order
    ooo_ct_++;
    if (shift >= -32) {
      // Ack previous message if possible
      ack_field_ |= (0b1 << -shift);
    }
  } else if (shift > 0) {
    last_seen_message_id_ = msg.header.message_id;
    for (int i = 0; i < shift; i++) {
      if ((ack_field_ & (0b1 << i)) == 0) {
        drop_ct_++;
      }
    }
    ack_field_ <<= shift;
    ack_field_ |= (0b1 << (shift - 1));
  }
}

std::optional<ClientMessage> BenchmarkApp::get_client_message() {
  if (!is_running_) {
    return std::nullopt;
  }

  ClientMessage msg{};
  msg.header.client_id = 127;
  msg.header.ack_field = ack_field_;
  msg.header.message_id = next_message_id_++;
  msg.header.magic_header = 0x5350414E;
  msg.header.last_seen_server_message_id = last_seen_message_id_;

  send_ct_++;

  if (!is_connected_) {
    msg.message_type = ClientMessageType::ConnectClient;
    return msg;
  }

  if (remaining_ping_count_ == 0u) {
    if (needs_stats_) {
      msg.message_type = ClientMessageType::GetStats;
      return msg;
    }

    is_running_ = false;
    msg.message_type = ClientMessageType::DisconnectClient;
    return msg;
  }

  remaining_ping_count_--;
  msg.message_type = ClientMessageType::Ping;
  PingMessage body{};
  body.payload.resize(payload_size_, '\0');
  for (std::uint32_t i = 0; i < payload_size_; i++) {
    body.payload[i] = ::kPayloadString[i % ::kPayloadString.length()];
  }
  msg.body = body;
  return msg;
}

bool BenchmarkApp::is_running() const { return is_running_; }

std::optional<ExperimentResults> BenchmarkApp::get_results() {
  if (is_running_) return std::nullopt;

  ExperimentResults rsl{};
  rsl.client_dropped_messages = drop_ct_;
  rsl.client_out_of_order_messages = ooo_ct_;
  rsl.client_recv_messages = recv_ct_;
  rsl.client_sent_messages = send_ct_;

  rsl.rtt_measurements = std::move(rtt_samples_);
  rsl.server_dropped_messages = net_stats_.dropped_messages;
  rsl.server_out_of_order_messages = net_stats_.out_of_order_messages;
  rsl.server_recv_messages = net_stats_.received_messages;

  return rsl;
}

}  // namespace spanreed::benchmark
