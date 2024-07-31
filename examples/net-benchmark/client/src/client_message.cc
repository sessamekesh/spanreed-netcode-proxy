#include <net-benchmark-client/client_message.h>
#include <net-benchmark-client/littleendian.h>

namespace {
const std::uint32_t kExpectedMagicNumber = 0x5350414E;
}  // namespace

namespace spanreed::benchmark {

static void serialize_ping_message(std::uint8_t* buff,
                                   const PingMessage& ping) {
  LittleEndian::WriteU64(buff, ping.client_send_ts);
  /* proxy_recv_client_ts */ LittleEndian::WriteU64(buff + 8, 0x00);
  /* proxy_forward_client_ts */ LittleEndian::WriteU64(buff + 16, 0x00);
  /* server_recv_ts */ LittleEndian::WriteU64(buff + 24, 0x00);
  LittleEndian::WriteU16(buff + 32,
                         static_cast<std::uint16_t>(ping.payload.length()));
  memcpy(buff + 34, &ping.payload[0], ping.payload.length());
}

std::optional<std::vector<std::uint8_t>> serialize_client_message(
    const ClientMessage& msg) {
  std::size_t buff_size = sizeof(ClientMessageHeader) + 1;
  switch (msg.message_type) {
    case ClientMessageType::Ping:
      buff_size += 34 + std::get<PingMessage>(msg.body).payload.length();
  }

  if (buff_size > 10240ull) {
    return std::nullopt;
  }

  std::vector<std::uint8_t> buff(buff_size, 0x00u);

  if (msg.message_type == ClientMessageType::UNKNOWN) {
    return std::nullopt;
  }

  std::uint8_t* raw_buff = &buff[0];

  LittleEndian::WriteU32(raw_buff, msg.header.magic_header);
  LittleEndian::WriteU32(raw_buff + 4, msg.header.client_id);
  LittleEndian::WriteU16(raw_buff + 8, msg.header.message_id);
  LittleEndian::WriteU16(raw_buff + 10, msg.header.last_seen_server_message_id);
  LittleEndian::WriteU32(raw_buff + 12, msg.header.ack_field);

  switch (msg.message_type) {
    case ClientMessageType::ConnectClient:
      raw_buff[16] = 1;
      break;
    case ClientMessageType::DisconnectClient:
      raw_buff[16] = 2;
      break;
    case ClientMessageType::Ping:
      raw_buff[16] = 3;
      serialize_ping_message(raw_buff + 17, std::get<PingMessage>(msg.body));
      break;
    case ClientMessageType::GetStats:
      raw_buff[16] = 4;
      break;
    default:
      return std::nullopt;
  }

  return buff;
}
}  // namespace spanreed::benchmark
