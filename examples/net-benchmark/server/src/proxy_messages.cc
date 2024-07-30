#include <net-benchmark-server/littleendian.h>
#include <net-benchmark-server/proxy_messages.h>
#include <spdlog/sinks/stdout_color_sinks.h>
#include <spdlog/spdlog.h>

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
std::size_t ProxyMessageHeader::HEADER_SIZE = sizeof(ProxyMessageHeader);

static std::optional<PingMessage> unpack_ping_message(std::uint8_t* buffer,
                                                      std::size_t buffer_len) {
  // Format:
  // 4 x uint64 (ping message fields)
  // u16 payload_length
  // payload_length bytes
  constexpr std::size_t PING_MESSAGE_MIN_SIZE = 8ull * 4ull + 2ull;
  if (buffer_len < PING_MESSAGE_MIN_SIZE) {
    ::getLogger()->warn("Ping message needs at least {} bytes, received {}",
                        PING_MESSAGE_MIN_SIZE, buffer_len);
    return {};
  }

  PingMessage msg{};
  msg.client_send_ts = LittleEndian::ParseU64(buffer);
  msg.proxy_recv_client_ts = LittleEndian::ParseU64(buffer + 8);
  msg.proxy_forward_client_ts = LittleEndian::ParseU64(buffer + 16);
  msg.server_recv_ts = LittleEndian::ParseU64(buffer + 24);

  std::uint16_t payload_len = LittleEndian::ParseU16(buffer + 32);

  std::size_t remaining_bytes = buffer_len - 34;
  if (remaining_bytes < payload_len) {
    ::getLogger()->warn(
        "Expected at least {} bytes to fill payload, only have {}", payload_len,
        remaining_bytes);
    return {};
  }

  msg.payload.resize(payload_len);
  memcpy(&msg.payload[0], buffer + 34, payload_len);
  return msg;
}

std::optional<ProxyMessage> parse_proxy_message(std::uint8_t* buffer,
                                                std::size_t buffer_len) {
  if (buffer_len < ProxyMessageHeader::HEADER_SIZE + 1) {
    ::getLogger()->warn("Expected at least {} bytes, received {}",
                        ProxyMessageHeader::HEADER_SIZE + 1, buffer_len);
    return {};
  }

  ProxyMessage msg{};
  msg.header.magic_header = LittleEndian::ParseU32(buffer);
  msg.header.client_id = LittleEndian::ParseU32(buffer + 4);
  msg.header.message_id = LittleEndian::ParseU16(buffer + 8);
  msg.header.last_seen_server_message_id = LittleEndian::ParseU16(buffer + 10);
  msg.header.ack_field = LittleEndian::ParseU32(buffer + 12);

  if (msg.header.magic_header != ::kExpectedMagicNumber) {
    ::getLogger()->warn("Expected magic header {}, received {}",
                        ::kExpectedMagicNumber, msg.header.magic_header);
    return {};
  }

  std::uint8_t msgType = buffer[16];

  if (msgType == 1) {
    msg.message_type = ProxyMessageType::ConnectClient;
  } else if (msgType == 2) {
    msg.message_type = ProxyMessageType::DisconnectClient;
  } else if (msgType == 3) {
    msg.message_type = ProxyMessageType::Ping;
    auto maybe_ping = unpack_ping_message(buffer + 17, buffer_len - 17);
    if (!maybe_ping.has_value()) {
      return {};
    }
    msg.body = *maybe_ping;
  } else if (msgType == 4) {
    msg.message_type = ProxyMessageType::GetStats;
  } else {
    ::getLogger()->warn("Expected msgType in range [1,4], got {}", msgType);
    return {};
  }

  return msg;
}
}  // namespace spanreed::benchmark
