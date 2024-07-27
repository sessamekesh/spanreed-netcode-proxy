#include <net-benchmark-server/littleendian.h>
#include <net-benchmark-server/proxy_messages.h>

namespace {
const std::uint32_t kExpectedMagicNumber = 0x5350414E;
}

namespace spanreed::benchmark {
std::size_t ProxyMessageHeader::HEADER_SIZE = sizeof(ProxyMessageHeader);

std::optional<ProxyMessage> parse_proxy_message(std::uint8_t* buffer,
                                                std::size_t buffer_len) {
  if (buffer_len < ProxyMessageHeader::HEADER_SIZE) {
    // TODO (sessamekesh): Log error
    return {};
  }
  return {};
}
}  // namespace spanreed::benchmark
