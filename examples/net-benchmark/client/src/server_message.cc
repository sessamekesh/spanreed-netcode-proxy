#include <net-benchmark-client/littleendian.h>
#include <net-benchmark-client/server_message.h>

namespace {
const std::uint32_t kExpectedMagicNumber = 0x5350414E;
}

namespace spanreed::benchmark {

static std::optional<ConnectionVerdict> unpack_verdict(const std::uint8_t* buff,
                                                       std::size_t buff_len) {
  if (buff_len < 1) {
    return std::nullopt;
  }

  ConnectionVerdict v{};
  v.verdict = buff[0] > 0;

  return v;
}

static std::optional<Pong> unpack_pong_message(const std::uint8_t* buff,
                                               std::size_t buff_len) {
  constexpr std::size_t PONG_MESSAGE_MIN_SIZE = 7ull * 8ull + 2ull;
  if (buff_len < PONG_MESSAGE_MIN_SIZE) {
    return std::nullopt;
  }

  Pong pong{};
  pong.client_send_ts = LittleEndian::ParseU64(buff);
  pong.proxy_recv_client_ts = LittleEndian::ParseU64(buff + 8);
  pong.proxy_forward_client_ts = LittleEndian::ParseU64(buff + 16);
  pong.server_recv_ts = LittleEndian::ParseU64(buff + 24);
  pong.server_send_ts = LittleEndian::ParseU64(buff + 32);
  pong.proxy_recv_destination_ts = LittleEndian::ParseU64(buff + 40);
  pong.proxy_forward_destination_ts = LittleEndian::ParseU64(buff + 48);

  std::uint16_t payload_len = LittleEndian::ParseU16(buff + 56);

  std::size_t remaining_bytes = buff_len - 58;
  if (remaining_bytes < payload_len) {
    return std::nullopt;
  }

  pong.payload.resize(payload_len);
  memcpy(&pong.payload[0], buff + 58, payload_len);
  return pong;
}

static std::optional<ServerNetStats> unpack_net_stats(const std::uint8_t* buff,
                                                      std::size_t buff_len) {
  if (buff_len < sizeof(ServerNetStats)) {
    return std::nullopt;
  }

  ServerNetStats stats{};
  stats.last_seen_message_id = LittleEndian::ParseU32(buff);
  stats.received_messages = LittleEndian::ParseU32(buff + 4);
  stats.dropped_messages = LittleEndian::ParseU32(buff + 8);
  stats.out_of_order_messages = LittleEndian::ParseU32(buff + 12);

  return stats;
}

std::optional<ServerMessage> spanreed::benchmark::parse_server_message(
    const std::uint8_t* buff, std::size_t buff_len) {
  if (buff_len < sizeof(ServerMessageHeader) + 1) {
    return std::nullopt;
  }

  ServerMessage msg{};
  msg.header.magic_header = LittleEndian::ParseU32(buff);
  msg.header.client_id = LittleEndian::ParseU32(buff + 4);
  msg.header.message_id = LittleEndian::ParseU16(buff + 8);
  msg.header.last_seen_client_message_id = LittleEndian::ParseU16(buff + 10);
  msg.header.ack_field = LittleEndian::ParseU32(buff + 12);

  if (msg.header.magic_header != ::kExpectedMagicNumber) {
    return std::nullopt;
  }

  std::uint8_t msgType = buff[16];

  switch (msgType) {
    case 0x1: {
      msg.message_type = ServerMessageType::ConnectionVerdict;
      auto maybe_verdict = unpack_verdict(buff + 17, buff_len - 17);
      if (!maybe_verdict) {
        return std::nullopt;
      }
      msg.body = *maybe_verdict;
    } break;
    case 0x2:
      msg.message_type = ServerMessageType::DisconnectRequest;
      break;
    case 0x3: {
      msg.message_type = ServerMessageType::Pong;
      auto maybe_ping = unpack_pong_message(buff + 17, buff_len - 17);
      if (!maybe_ping) {
        return std::nullopt;
      }
      msg.body = *maybe_ping;
    } break;
    case 0x4: {
      msg.message_type = ServerMessageType::ServerNetStats;
      auto maybe_stats = unpack_net_stats(buff + 17, buff_len - 17);
      if (!maybe_stats) {
        return std::nullopt;
      }
      msg.body = *maybe_stats;
    } break;
  }

  return msg;
}

}  // namespace spanreed::benchmark
