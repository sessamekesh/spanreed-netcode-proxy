#include <net-benchmark-client/timer.h>

namespace spanreed::benchmark {

Timer::Timer() : start_time_(std::chrono::high_resolution_clock::now()) {}

std::uint64_t Timer::get_time() const {
  auto now = std::chrono::high_resolution_clock::now();
  return std::chrono::duration_cast<std::chrono::microseconds>(now -
                                                               start_time_)
      .count();
}

}  // namespace spanreed::benchmark
