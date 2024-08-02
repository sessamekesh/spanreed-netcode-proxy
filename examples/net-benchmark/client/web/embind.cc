#include <emscripten/bind.h>
#include <net-benchmark-client/benchmark_app.h>
#include <net-benchmark-client/client_message.h>
#include <net-benchmark-client/server_message.h>
#include <net-benchmark-client/timer.h>

using namespace emscripten;

// TODO (sessamekesh): Write custom serialize/parse wrappers, since those
//  take pointer parameters which seem to make embind mad!

static bool on_recv_message(std::string msg, spanreed::benchmark::Timer& timer,
                            spanreed::benchmark::BenchmarkApp& app) {
  auto parsed_msg_opt = spanreed::benchmark::parse_server_message(
      reinterpret_cast<const std::uint8_t*>(&msg[0]), msg.length());

  if (!parsed_msg_opt.has_value()) {
    return false;
  }

  auto& parsed_msg = *parsed_msg_opt;

  if (parsed_msg.message_type == spanreed::benchmark::ServerMessageType::Pong) {
    auto& pongmsg = std::get<spanreed::benchmark::Pong>(parsed_msg.body);
    pongmsg.client_recv_ts = timer.get_time();
  }

  app.add_server_message(*parsed_msg_opt);
  return true;
}

static void maybe_send_msg(spanreed::benchmark::BenchmarkApp& app,
                           spanreed::benchmark::Timer& timer, val cb) {
  auto maybe_msg = app.get_client_message();
  if (!maybe_msg.has_value()) {
    return;
  }

  auto& msg = *maybe_msg;

  if (msg.message_type == spanreed::benchmark::ClientMessageType::Ping) {
    auto& ping = std::get<spanreed::benchmark::PingMessage>(msg.body);
    ping.client_send_ts = timer.get_time();
  }

  auto bufopt = spanreed::benchmark::serialize_client_message(msg);
  if (!bufopt.has_value()) {
    return;
  }

  auto& buf = *bufopt;
  auto typed_array = val(typed_memory_view(buf.size(), &buf[0]));

  cb(typed_array);
}

EMSCRIPTEN_BINDINGS(BenchmarkWebClient) {
  class_<spanreed::benchmark::BenchmarkApp>("SpanreedBenchmarkApp")
      .constructor<>()
      .function("start_experiment",
                &spanreed::benchmark::BenchmarkApp::start_experiment)
      .function("get_results", &spanreed::benchmark::BenchmarkApp::get_results)
      .function("is_running", &spanreed::benchmark::BenchmarkApp::is_running);

  class_<spanreed::benchmark::Timer>("Timer").constructor<>().function(
      "get_time", &spanreed::benchmark::Timer::get_time);

  function("on_recv_message", &on_recv_message);
  function("maybe_send_msg", &maybe_send_msg);
}
