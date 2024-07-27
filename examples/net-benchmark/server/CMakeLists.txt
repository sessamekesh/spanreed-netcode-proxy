cmake_minimum_required(VERSION 3.10)

project(
  NetBenchmarkServer
  VERSION 1.0
  LANGUAGES CXX)

include(FetchContent)

FetchContent_Declare(
  concurrentqueue
  GIT_REPOSITORY "https://github.com/cameron314/concurrentqueue"
  GIT_TAG "65d6970912fc3f6bb62d80edf95ca30e0df85137"
)
FetchContent_MakeAvailable(concurrentqueue)

set(net-benchmark-server-include
  "include/net-benchmark-server/destination_messages.h"
  "include/net-benchmark-server/littleendian.h"
  "include/net-benchmark-server/proxy_messages.h"
  "include/net-benchmark-server/udp_socket.h")

set(net-benchmark-server-src
  "src/littleendian.cc"
  "src/proxy_messages.cc")

add_executable(net-benchmark-server
  ${net-benchmark-server-src}
  ${net-benchmark-server-include}
  main.cc)
target_include_directories(net-benchmark-server PUBLIC include)
set_property(TARGET net-benchmark-server PROPERTY CXX_STANDARD 20)
target_link_libraries(net-benchmark-server PUBLIC concurrentqueue)

if (WIN32)
  target_link_libraries(net-benchmark-server PUBLIC wsock32 ws2_32)
endif ()
