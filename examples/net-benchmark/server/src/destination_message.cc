#include <net-benchmark-server/destination_messages.h>
#include <net-benchmark-server/littleendian.h>
#include <spdlog/sinks/stdout_color_sinks.h>
#include <spdlog/spdlog.h>

#include <memory>

namespace {
const std::uint32_t kExpectedMagicNumber = 0x5350414E;

std::shared_ptr<spdlog::logger> gLog;
std::shared_ptr<spdlog::logger> getLogger() {
  if (gLog == nullptr) {
    gLog = spdlog::stdout_color_mt("parsemsg");
  }
  return gLog;
}
}  // namespace

namespace spanreed::benchmark {

static void serialize_verdict_message(std::uint8_t* buff,
                                      const ConnectClientVerdict& verdict) {
  buff[0] = verdict.verdict ? 0x1 : 0x0;
}

static void serialize_pong_message(std::uint8_t* buff,
                                   const PongMessage& pong) {
  LittleEndian::WriteU64(buff, pong.client_send_ts);
  LittleEndian::WriteU64(buff + 8, pong.proxy_recv_client_ts);
  LittleEndian::WriteU64(buff + 16, pong.proxy_forward_client_ts);
  LittleEndian::WriteU64(buff + 24, pong.server_recv_ts);
  LittleEndian::WriteU64(buff + 32, pong.server_send_ts);
  /* proxy_recv_destintation_ts= */ LittleEndian::WriteU64(buff + 40, 0x00);
  /* proxy_send_destintation_ts= */ LittleEndian::WriteU64(buff + 48, 0x00);
  LittleEndian::WriteU16(buff + 56,
                         static_cast<std::uint16_t>(pong.payload.length()));
  memcpy(buff + 58, &pong.payload[0], pong.payload.length());
}

static void serialize_net_stats(std::uint8_t* buff,
                                const DestinationStats& stats) {
  LittleEndian::WriteU32(buff, stats.last_seen_message_id);
  LittleEndian::WriteU32(buff + 4, stats.received_messages);
  LittleEndian::WriteU32(buff + 8, stats.dropped_messages);
  LittleEndian::WriteU32(buff + 12, stats.out_of_order_messages);
}

// TODO (sessamekesh): Serialize net stats!

std::vector<std::uint8_t> serialize_destination_message(
    const DestinationMessage& msg) {
  std::size_t buff_size = sizeof(DestinationMessageHeader) + 1;
  if (msg.message_type == DestinationMessageType::ConnectionVerdict) {
    buff_size += 1;
  } else if (msg.message_type == DestinationMessageType::Pong) {
    buff_size += 58 + std::get<PongMessage>(msg.body).payload.length();
  } else if (msg.message_type == DestinationMessageType::Stats) {
    buff_size += sizeof(DestinationStats);
  }

  if (buff_size > 10240ull) {
    ::getLogger()->error(
        "Payload too large ({} bytes), cannot serialize destination message",
        buff_size);
    return std::vector<std::uint8_t>{};
  }

  std::vector<std::uint8_t> buff(buff_size, 0x00u);

  if (msg.message_type == DestinationMessageType::UNKNOWN) {
    ::getLogger()->error("UNKNOWN message type, cannot serialize");
    return std::vector<std::uint8_t>{};
  }

  std::uint8_t* raw_buff = &buff[0];

  LittleEndian::WriteU32(raw_buff, msg.header.magic_header);
  LittleEndian::WriteU32(raw_buff + 4, msg.header.client_id);
  LittleEndian::WriteU16(raw_buff + 8, msg.header.message_id);
  LittleEndian::WriteU16(raw_buff + 10, msg.header.last_seen_client_message_id);
  LittleEndian::WriteU32(raw_buff + 12, msg.header.ack_field);

  switch (msg.message_type) {
    case DestinationMessageType::ConnectionVerdict:
      *(raw_buff + 16) = 0x1;
      break;
    case DestinationMessageType::DisconnectClient:
      *(raw_buff + 16) = 0x2;
      break;
    case DestinationMessageType::Pong:
      *(raw_buff + 16) = 0x3;
      break;
    case DestinationMessageType::Stats:
      *(raw_buff + 16) = 0x4;
      break;
  }

  if (msg.message_type == DestinationMessageType::ConnectionVerdict) {
    serialize_verdict_message(raw_buff + 17,
                              std::get<ConnectClientVerdict>(msg.body));
  } else if (msg.message_type == DestinationMessageType::Pong) {
    serialize_pong_message(raw_buff + 17, std::get<PongMessage>(msg.body));
  } else if (msg.message_type == DestinationMessageType::Stats) {
    serialize_net_stats(raw_buff + 17, std::get<DestinationStats>(msg.body));
  }

  return buff;
}

}  // namespace spanreed::benchmark
