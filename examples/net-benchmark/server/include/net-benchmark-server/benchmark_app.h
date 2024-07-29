#ifndef SPANREED_BENCHMARK_APP_H
#define SPANREED_BENCHMARK_APP_H

#include <concurrentqueue.h>
#include <net-benchmark-server/udp_server.h>

namespace spanreed::benchmark {

class BenchmarkApp {
 public:
  BenchmarkApp(UdpServer* udp_server,
               moodycamel::ConcurrentQueue<ProxyMessage>* incoming_messages);

  void Start();
  void Stop();

 private:
  moodycamel::ConcurrentQueue<ProxyMessage>* incoming_messages_;
  UdpServer* udp_server_;
  std::shared_ptr<spdlog::logger> log;

  std::atomic_bool is_running_;
};

}  // namespace spanreed::benchmark

#endif
