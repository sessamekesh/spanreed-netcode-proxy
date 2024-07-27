#ifndef SPANREED_BENCHMARK_ENDIAN_H
#define SPANREED_BENCHMARK_ENDIAN_H

#include <cstdint>

namespace spanreed::benchmark {

struct LittleEndian {
  static std::uint16_t ParseU16(std::uint8_t* buf);
  static std::uint32_t ParseU32(std::uint8_t* buf);
  static std::uint64_t ParseU64(std::uint8_t* buf);
};

}  // namespace spanreed::benchmark

#endif
