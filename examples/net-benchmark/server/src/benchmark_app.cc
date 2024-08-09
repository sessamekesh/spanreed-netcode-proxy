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

  std::uint32_t received_messages;
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
  log->info("App loop starting");
  is_running_ = true;

  std::unordered_map</* client_id= */ std::uint32_t, ConnectedClient> clients;

  while (is_running_) {
    ProxyMessage msg{};
    if (!incoming_messages_->try_dequeue(msg)) {
      continue;
    }

    //
    // Pre-process:
    switch (msg.message_type) {
      case ProxyMessageType::ConnectClient: {
        if (clients.count(msg.header.client_id) > 0)
          break;  // Don't re-connect, re-send verdict though

        ConnectedClient new_client{};
        new_client.client_id = msg.header.client_id;
        new_client.last_seen_message_id = msg.header.message_id;
        new_client.ack_field = ~0x0;
        clients[msg.header.client_id] = new_client;
      } break;
      case ProxyMessageType::DisconnectClient: {
        clients.erase(msg.header.client_id);
      } break;
    }

    //
    // Process:
    auto it = clients.find(msg.header.client_id);
    if (it == clients.end()) break;

    auto& client = it->second;
    client.received_messages++;

    int shift = msg.header.message_id - client.last_seen_message_id;
    if (shift < 0) {
      // Out of order message!
      client.out_of_order_messages++;
      // Go back and ACK if possible
      if (shift >= -32) {
        client.ack_field |= (0b1 << -shift);
      }
    } else if (shift > 0) {
      client.last_seen_message_id = msg.header.message_id;
      for (int i = 0; i < shift; i++) {
        if ((client.ack_field & (0b1 << i)) == 0) {
          client.dropped_messages++;
        }
      }
      client.ack_field <<= shift;
      client.ack_field |= (0b1 << (shift - 1));
    }

    DestinationMessage response{};
    response.header.client_id = msg.header.client_id;
    response.header.ack_field = client.ack_field;
    response.header.last_seen_client_message_id = client.last_seen_message_id;
    response.header.magic_header = msg.header.magic_header;

    switch (msg.message_type) {
      case ProxyMessageType::ConnectClient: {
        response.header.message_id = client.next_message_id++;
        response.message_type = DestinationMessageType::ConnectionVerdict;

        ConnectClientVerdict verdict{};
        verdict.verdict = true;
        response.body = verdict;

        udp_server_->QueueMessage(std::move(response));
      } break;
      case ProxyMessageType::Ping: {
        auto& ping = std::get<PingMessage>(msg.body);
        response.header.message_id = client.next_message_id++;
        response.message_type = DestinationMessageType::Pong;

        PongMessage pong_body{};
        pong_body.client_send_ts = ping.client_send_ts;
        pong_body.payload = ping.payload;
        pong_body.proxy_forward_client_ts = ping.proxy_forward_client_ts;
        pong_body.proxy_recv_client_ts = ping.proxy_recv_client_ts;
        pong_body.server_recv_ts = ping.server_recv_ts;
        pong_body.server_send_ts = 1;  // Will be overwritten in UdpServer

        response.body = pong_body;

        udp_server_->QueueMessage(std::move(response));
      } break;
      case ProxyMessageType::GetStats: {
        response.header.message_id = client.next_message_id++;
        response.message_type = DestinationMessageType::Stats;

        DestinationStats stats{};
        stats.dropped_messages = client.dropped_messages;
        stats.last_seen_message_id = client.last_seen_message_id;
        stats.out_of_order_messages = client.out_of_order_messages;
        stats.received_messages = client.received_messages;
        response.body = stats;

        udp_server_->QueueMessage(std::move(response));
      } break;
    }
  }

  log->info("App successfully finished running");
}

void BenchmarkApp::Stop() {
  log->info("Marking app for stop");
  is_running_ = false;
}

}  // namespace spanreed::benchmark