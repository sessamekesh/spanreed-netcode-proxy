#ifndef SPANREED_BENCHMARK_UDP_SOCKET_H
#define SPANREED_BENCHMARK_UDP_SOCKET_H

#include <concurrentqueue.h>

#include <cstdint>
#include <string>

namespace spanreed::benchmark {

struct UdpServerStartRsl {
  bool is_success;
  std::string err_msg;
};

struct UdpSocketConnection {};

class UdpServer {
 public:
  UdpServer(std::uint16_t port);
  UdpServerStartRsl Start();

 private:
};

}  // namespace spanreed::benchmark

#endif
