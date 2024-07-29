#include <net-benchmark-server/timer.h>
#include <net-benchmark-server/udp_server.h>
#include <spdlog/sinks/stdout_color_sinks.h>
#include <spdlog/spdlog.h>

int main() {
  spdlog::set_pattern("[%H:%M:%S %z] %^[%n (%l)]%$ %v");
  auto console = spdlog::stdout_color_mt("console");

  console->info("Hello, world!");

  spanreed::benchmark::Timer timer;
  moodycamel::ConcurrentQueue<spanreed::benchmark::ProxyMessage>
      incoming_message_queue;
  spanreed::benchmark::UdpServer server(30001, &incoming_message_queue, &timer);

  std::thread serverthread([&server] { server.Start(); });

  serverthread.join();

  // This should support native UDP and also TCP and also WebSocket
  console->info("Finished! Exiting");
  return 0;
}
