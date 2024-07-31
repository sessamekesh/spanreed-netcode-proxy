#ifndef SPANREED_BENCHMARK_CLIENT_NETCLIENT_H
#define SPANREED_BENCHMARK_CLIENT_NETCLIENT_H

#include <concurrentqueue.h>
#include <net-benchmark-client/client_message.h>
#include <net-benchmark-client/server_message.h>
#include <net-benchmark-client/timer.h>

#include <string>

namespace spanreed::benchmark {

class NetClient {
 public:
  NetClient(std::uint16_t client_port, std::string uri, Timer* timer);

  bool Start();
  void Stop();

  void QueueMessage(ClientMessage msg);
  std::optional<ServerMessage> ReadMessage();

 private:
  std::uint16_t client_port_;
  std::string uri_;
  Timer* timer_;

  std::atomic_bool is_running_;

  moodycamel::ConcurrentQueue<ClientMessage> outgoing_messages_;
  moodycamel::ConcurrentQueue<ServerMessage> incoming_messages_;
};

}  // namespace spanreed::benchmark

#endif
