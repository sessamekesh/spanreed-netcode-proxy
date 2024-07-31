#ifndef SPANREED_BENCHMARK_CLIENT_ENDIAN_H
#define SPANREED_BENCHMARK_CLIENT_ENDIAN_H

#include <cstdint>

namespace spanreed::benchmark {

struct LittleEndian {
  static std::uint16_t ParseU16(const std::uint8_t* buf);
  static std::uint32_t ParseU32(const std::uint8_t* buf);
  static std::uint64_t ParseU64(const std::uint8_t* buf);

  static void WriteU16(std::uint8_t* buf, std::uint16_t value);
  static void WriteU32(std::uint8_t* buf, std::uint32_t value);
  static void WriteU64(std::uint8_t* buf, std::uint64_t value);
};

}  // namespace spanreed::benchmark

#endif
