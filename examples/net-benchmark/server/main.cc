#include <net-benchmark-server/benchmark_app.h>
#include <net-benchmark-server/timer.h>
#include <net-benchmark-server/udp_server.h>
#include <spdlog/sinks/stdout_color_sinks.h>
#include <spdlog/spdlog.h>

#include <csignal>

std::function<void()> gSigintCb;

void signalHandler(int signum) {
  if (gSigintCb) {
    gSigintCb();
  }
}

int main() {
  signal(SIGINT, signalHandler);

  spdlog::set_pattern("[%H:%M:%S %z] %^[%n (%l)]%$ %v");
  auto console = spdlog::stdout_color_mt("console");

  console->info("Hello, world!");

  spanreed::benchmark::Timer timer;
  moodycamel::ConcurrentQueue<spanreed::benchmark::ProxyMessage>
      incoming_message_queue;
  spanreed::benchmark::UdpServer server(30001, &incoming_message_queue, &timer);
  spanreed::benchmark::BenchmarkApp app(&server, &incoming_message_queue);

  std::thread serverthread([&server] { server.Start(); });
  std::thread appthread([&app] { app.Start(); });

  gSigintCb = [&server, &app, console] {
    console->warn(
        "SIGINT received! Sending termination signal to UDP server and "
        "benchmark app...");
    server.Stop();
    app.Stop();
  };

  serverthread.join();
  appthread.join();

  // This should support native UDP and also TCP and also WebSocket
  console->info("Finished! Exiting");
  return 0;
}
