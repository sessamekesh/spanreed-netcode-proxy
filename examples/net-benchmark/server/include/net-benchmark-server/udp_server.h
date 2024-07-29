#ifndef SPANREED_BENCHMARK_UDP_SERVER_H
#define SPANREED_BENCHMARK_UDP_SERVER_H

#include <concurrentqueue.h>
#include <net-benchmark-server/destination_messages.h>
#include <net-benchmark-server/proxy_messages.h>
#include <net-benchmark-server/timer.h>
#include <spdlog/spdlog.h>

namespace spanreed::benchmark {

class UdpServer {
 public:
  UdpServer(std::uint16_t port,
            moodycamel::ConcurrentQueue<ProxyMessage>* incoming_message_queue,
            Timer* timer);

  void QueueMessage(DestinationMessage msg);
  bool Start();
  void Stop();

 private:
  std::uint16_t port_;
  std::atomic_bool is_running_;

  std::shared_ptr<spdlog::logger> log;
  Timer* timer_;

  moodycamel::ConcurrentQueue<DestinationMessage> outgoing_messages_;
  moodycamel::ConcurrentQueue<ProxyMessage>* incoming_messages_;
};

}  // namespace spanreed::benchmark

#endif
