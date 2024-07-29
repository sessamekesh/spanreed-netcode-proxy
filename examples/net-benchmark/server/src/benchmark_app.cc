#include <net-benchmark-server/benchmark_app.h>
#include <spdlog/sinks/stdout_color_sinks.h>

#include <shared_mutex>

// LUCEM
namespace {
struct ConnectedClient {
  std::uint32_t client_id;
  std::uint16_t last_seen_message_id;
  std::uint32_t ack_field;
  std::uint32_t next_message_id;

  std::uint32_t dropped_messages;
  std::uint32_t out_of_order_messages;
};
}  // namespace

namespace spanreed::benchmark {

BenchmarkApp::BenchmarkApp(
    UdpServer* udp_server,
    moodycamel::ConcurrentQueue<ProxyMessage>* incoming_messages)
    : incoming_messages_(incoming_messages),
      udp_server_(udp_server),
      log(spdlog::stdout_color_mt("BenchmarkApp")),
      is_running_(false) {}

void BenchmarkApp::Start() {
  is_running_ = true;

  std::unordered_map</* client_id= */ std::uint32_t, ConnectedClient> clients;

  while (is_running_) {
    ProxyMessage msg{};
    if (!incoming_messages_->try_dequeue(msg)) {
      continue;
    }

    switch (msg.message_type) {
      case ProxyMessageType::ConnectClient: {
        ConnectedClient new_client{};
        new_client.client_id = msg.header.client_id;
        new_client.last_seen_message_id = msg.header.message_id;
        clients[msg.header.client_id] = new_client;
      } break;
      case ProxyMessageType::DisconnectClient: {
        clients.erase(msg.header.client_id);
      } break;
      case ProxyMessageType::Ping: {
        auto it = clients.find(msg.header.client_id);
        if (it == clients.end()) break;

        auto& client = it->second;

        // TODO (sessamekesh): Ack fields+store metrics data
        // TODO (sessamekesh): Pong!

      } break;
    }
  }
}

void BenchmarkApp::Stop() { is_running_ = false; }

}  // namespace spanreed::benchmark