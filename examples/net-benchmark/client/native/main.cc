#include <net-benchmark-client/benchmark_app.h>

#include <format>
#include <iostream>
#include <string>

#include "netclient.h"

int main() {
  std::cout << "------ Spanreed Benchmark Client (native) ------" << std::endl;

  std::string destination_addr;
  std::string msg_ct_str;
  std::string gap_ms_str;
  std::string payload_size_str;
  std::uint32_t msg_ct;
  std::uint32_t gap_ms;
  std::uint32_t payload_size;

  std::cout << "\n1. Destination (localhost:30001)? ";
  std::getline(std::cin, destination_addr);
  if (destination_addr == "") {
    destination_addr = "localhost:30001";
  }

  std::cout << "2. Number of messages to send (1000)? ";
  std::getline(std::cin, msg_ct_str);
  if (msg_ct_str == "") {
    msg_ct = 1000;
  } else {
    try {
      msg_ct = std::stoul(msg_ct_str);
    } catch (std::invalid_argument e) {
      std::cerr << "Invalid number of messages" << std::endl;
      return -1;
    }
  }

  std::cout << "3. Gap between messages (ms) (25)? ";
  std::getline(std::cin, gap_ms_str);
  if (gap_ms_str == "") {
    gap_ms = 25;
  } else {
    try {
      gap_ms = std::stoul(gap_ms_str);
    } catch (std::invalid_argument e) {
      std::cerr << "Invalid gap length" << std::endl;
      return -1;
    }
  }

  std::cout << "4. Payload size (10)? ";
  std::getline(std::cin, payload_size_str);
  if (payload_size_str == "") {
    payload_size = 10;
  } else {
    try {
      payload_size = std::stoul(payload_size_str);
    } catch (std::invalid_argument e) {
      std::cerr << "Invalid payload size" << std::endl;
      return -1;
    }
  }

  spanreed::benchmark::Timer timer;
  spanreed::benchmark::NetClient client(30005u, destination_addr, &timer);
  spanreed::benchmark::BenchmarkApp app;

  std::thread udp_thread([&client] { client.Start(); });

  std::cout << "Starting experiment!" << std::endl;
  app.start_experiment(payload_size, msg_ct);
  std::chrono::high_resolution_clock::time_point last_sample_at =
      std::chrono::high_resolution_clock::now();
  while (app.is_running()) {
    auto wait_until = last_sample_at + std::chrono::milliseconds(gap_ms);

    for (auto nextmsgopt = client.ReadMessage(); nextmsgopt.has_value();
         nextmsgopt = client.ReadMessage()) {
      app.add_server_message(*nextmsgopt);
    }

    auto maybe_msg = app.get_client_message();
    if (!maybe_msg.has_value()) {
      last_sample_at = std::chrono::high_resolution_clock::now();
      std::this_thread::sleep_until(wait_until);
      continue;
    }

    client.QueueMessage(*maybe_msg);

    last_sample_at = std::chrono::high_resolution_clock::now();
    std::this_thread::sleep_until(wait_until);
  }

  client.Stop();

  auto resultsopt = app.get_results();
  if (!resultsopt.has_value()) {
    std::cerr << "Results could not be found!" << std::endl;
    return -1;
  }

  const auto& results = *resultsopt;
  std::cout << "\n------------------ RESULTS READY ------------------\n\n";

  std::cout << std::format("{:40}| {}\n", "Client Messages Sent",
                           results.client_sent_messages);
  std::cout << std::format("{:40}| {}\n", "Client Messages Received",
                           results.server_recv_messages);
  std::cout << std::format("{:40}| {}\n", "Client Out-Of-Order Receive Ct",
                           results.server_out_of_order_messages);
  std::cout << std::format("{:40}| {}\n", "Client Messages Dropped Ct",
                           results.server_dropped_messages);

  std::cout << std::format("{:40}| {}\n", "Server Messages Received",
                           results.client_recv_messages);
  std::cout << std::format("{:40}| {}\n", "Server Out-Of-Order Receive Ct",
                           results.client_out_of_order_messages);
  std::cout << std::format("{:40}| {}\n", "Server Messages Dropped Ct",
                           results.client_dropped_messages);
  std::cout << "\n\n";

  auto aggsopt =
      spanreed::benchmark::RttAggregates::From(results.rtt_measurements);

  if (aggsopt.has_value()) {
    const auto& aggs = *aggsopt;
    std::cout << std::format(
        "{:18}|{:^8}|{:^8}|{:^8}|{:^8}|{:^8}|{:^8}|{:^8}\n", "Metric", "Median",
        "StdDev", "P90", "P95", "P99", "Min", "Max");
    std::cout << "-------------------------------------------------------------"
                 "----------------------\n";
    std::cout << std::format(
        "{:18}|{:^8.4f}|{:^8.4f}|{:^8.4f}|{:^8.4f}|{:^8.4f}|{:^8.4f}|{:^8.4f}"
        "\n",
        "ClientRTT", aggs.ClientRTT.P50 / 1000., aggs.ClientRTT.StdDev / 1000.,
        aggs.ClientRTT.P90 / 1000., aggs.ClientRTT.P95 / 1000.,
        aggs.ClientRTT.P99 / 1000., aggs.ClientRTT.Min / 1000.,
        aggs.ClientRTT.Max / 1000.);
    std::cout << std::format(
        "{:18}|{:^8.4f}|{:^8.4f}|{:^8.4f}|{:^8.4f}|{:^8.4f}|{:^8.4f}|{:^8.4f}"
        "\n",
        "DestProcessTime", aggs.DestProcessTime.P50 / 1000.,
        aggs.DestProcessTime.StdDev / 1000., aggs.DestProcessTime.P90 / 1000.,
        aggs.DestProcessTime.P95 / 1000., aggs.DestProcessTime.P99 / 1000.,
        aggs.DestProcessTime.Min / 1000., aggs.DestProcessTime.Max / 1000.);
    std::cout << std::format(
        "{:18}|{:^8.4f}|{:^8.4f}|{:^8.4f}|{:^8.4f}|{:^8.4f}|{:^8.4f}|{:^8.4f}"
        "\n",
        "NetTime", aggs.ClientProxyNetTime.P50 / 1000.,
        aggs.ClientProxyNetTime.StdDev / 1000.,
        aggs.ClientProxyNetTime.P90 / 1000.,
        aggs.ClientProxyNetTime.P95 / 1000.,
        aggs.ClientProxyNetTime.P99 / 1000.,
        aggs.ClientProxyNetTime.Min / 1000.,
        aggs.ClientProxyNetTime.Max / 1000.);
  }

  std::cout << "\n\nStopping UDP litstener to aggregate result..." << std::endl;
  udp_thread.join();

  return 0;
}
