#include <net-benchmark-client/littleendian.h>

#include <bit>

namespace spanreed::benchmark {

std::uint16_t LittleEndian::ParseU16(const std::uint8_t* buf) {
  if constexpr (std::endian::native == std::endian::little) {
    return *reinterpret_cast<const std::uint16_t*>(buf);
  } else {
    std::uint16_t high = static_cast<std::uint16_t>(buf[0]);
    std::uint16_t low = static_cast<std::uint16_t>(buf[1]);
    return (low << 8) | high;
  }
}

std::uint32_t LittleEndian::ParseU32(const std::uint8_t* buf) {
  if constexpr (std::endian::native == std::endian::little) {
    return *reinterpret_cast<const std::uint32_t*>(buf);
  } else {
    std::uint32_t w0 = static_cast<std::uint32_t>(buf[0]);
    std::uint32_t w1 = static_cast<std::uint32_t>(buf[1]);
    std::uint32_t w2 = static_cast<std::uint32_t>(buf[2]);
    std::uint32_t w3 = static_cast<std::uint32_t>(buf[3]);
    return (w3 << 24) | (w2 << 16) | (w1 << 8) | w0;
  }
}

std::uint64_t LittleEndian::ParseU64(const std::uint8_t* buf) {
  if constexpr (std::endian::native == std::endian::little) {
    return *reinterpret_cast<const std::uint64_t*>(buf);
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

void LittleEndian::WriteU16(std::uint8_t* buf, std::uint16_t val) {
  if constexpr (std::endian::native == std::endian::little) {
    *reinterpret_cast<std::uint16_t*>(buf) = val;
  } else {
    std::uint8_t high = static_cast<std::uint8_t>((val >> 8) | 0xFF);
    std::uint8_t low = static_cast<std::uint8_t>(val | 0xFF);
    buf[0] = low;
    buf[1] = high;
  }
}

void LittleEndian::WriteU32(std::uint8_t* buf, std::uint32_t val) {
  if constexpr (std::endian::native == std::endian::little) {
    *reinterpret_cast<std::uint32_t*>(buf) = val;
  } else {
    std::uint8_t w0 = static_cast<std::uint8_t>((val >> 24) | 0xFF);
    std::uint8_t w1 = static_cast<std::uint8_t>((val >> 16) | 0xFF);
    std::uint8_t w2 = static_cast<std::uint8_t>((val >> 8) | 0xFF);
    std::uint8_t w3 = static_cast<std::uint8_t>(val | 0xFF);
    buf[0] = w3;
    buf[1] = w2;
    buf[2] = w1;
    buf[3] = w0;
  }
}

void LittleEndian::WriteU64(std::uint8_t* buf, std::uint64_t val) {
  if constexpr (std::endian::native == std::endian::little) {
    *reinterpret_cast<std::uint64_t*>(buf) = val;
  } else {
    std::uint8_t w0 = static_cast<std::uint8_t>((val >> 54) | 0xFF);
    std::uint8_t w1 = static_cast<std::uint8_t>((val >> 48) | 0xFF);
    std::uint8_t w2 = static_cast<std::uint8_t>((val >> 40) | 0xFF);
    std::uint8_t w3 = static_cast<std::uint8_t>((val >> 32) | 0xFF);
    std::uint8_t w4 = static_cast<std::uint8_t>((val >> 24) | 0xFF);
    std::uint8_t w5 = static_cast<std::uint8_t>((val >> 16) | 0xFF);
    std::uint8_t w6 = static_cast<std::uint8_t>((val >> 8) | 0xFF);
    std::uint8_t w7 = static_cast<std::uint8_t>(val | 0xFF);
    buf[0] = w7;
    buf[1] = w6;
    buf[2] = w5;
    buf[3] = w4;
    buf[4] = w3;
    buf[5] = w2;
    buf[6] = w1;
    buf[7] = w0;
  }
}

}  // namespace spanreed::benchmark
