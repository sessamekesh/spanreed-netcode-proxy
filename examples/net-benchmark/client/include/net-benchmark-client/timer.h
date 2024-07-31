#ifndef SPANREED_BENCHMARK_CLIENT_TIMER_H
#define SPANREED_BENCHMARK_CLIENT_TIMER_H

#include <chrono>

namespace spanreed::benchmark {

class Timer {
 public:
  Timer();

  std::uint64_t get_time() const;

 private:
  std::chrono::high_resolution_clock::time_point start_time_;
};

}  // namespace spanreed::benchmark

#endif
