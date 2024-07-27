#include <net-benchmark-server/littleendian.h>

#include <bit>

namespace spanreed::benchmark {

std::uint16_t LittleEndian::ParseU16(std::uint8_t* buf) {
  if constexpr (std::endian::native == std::endian::little) {
    return *reinterpret_cast<std::uint16_t*>(buf);
  } else {
    std::uint16_t high = static_cast<std::uint16_t>(buf[0]);
    std::uint16_t low = static_cast<std::uint16_t>(buf[1]);
    return (low << 8) | high;
  }
}

std::uint32_t LittleEndian::ParseU32(std::uint8_t* buf) {
  if constexpr (std::endian::native == std::endian::little) {
    return *reinterpret_cast<std::uint32_t*>(buf);
  } else {
    std::uint32_t w0 = static_cast<std::uint32_t>(buf[0]);
    std::uint32_t w1 = static_cast<std::uint32_t>(buf[1]);
    std::uint32_t w2 = static_cast<std::uint32_t>(buf[2]);
    std::uint32_t w3 = static_cast<std::uint32_t>(buf[3]);
    return (w3 << 24) | (w2 << 16) | (w1 << 8) | w0;
  }
}

std::uint64_t LittleEndian::ParseU64(std::uint8_t* buf) {
  if constexpr (std::endian::native == std::endian::little) {
    return *reinterpret_cast<std::uint64_t*>(buf);
  } else {
    std::uint64_t w0 = static_cast<std::uint64_t>(buf[0]);
    std::uint64_t w1 = static_cast<std::uint64_t>(buf[1]);
    std::uint64_t w2 = static_cast<std::uint64_t>(buf[2]);
    std::uint64_t w3 = static_cast<std::uint64_t>(buf[3]);
    std::uint64_t w4 = static_cast<std::uint64_t>(buf[4]);
    std::uint64_t w5 = static_cast<std::uint64_t>(buf[5]);
    std::uint64_t w6 = static_cast<std::uint64_t>(buf[6]);
    std::uint64_t w7 = static_cast<std::uint64_t>(buf[7]);
    return (w7 << 56) | (w6 << 48) | (w5 << 40) | (w4 << 32) | (w3 << 24) |
           (w2 << 16) | (w1 << 8) | w0;
  }
}

}  // namespace spanreed::benchmark
