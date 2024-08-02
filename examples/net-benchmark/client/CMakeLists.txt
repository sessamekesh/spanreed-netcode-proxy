cmake_minimum_required(VERSION 3.10)

project(
  NetBenchmarkClient
  VERSION 1.0
  LANGUAGES CXX)

include(FetchContent)

set(common-hdr
  "include/net-benchmark-client/benchmark_app.h"
  "include/net-benchmark-client/client_message.h"
  "include/net-benchmark-client/littleendian.h"
  "include/net-benchmark-client/server_message.h"
  "include/net-benchmark-client/timer.h")

set(common-src
  "src/benchmark_app.cc"
  "src/client_message.cc"
  "src/littleendian.cc"
  "src/server_message.cc"
  "src/timer.cc")

add_library(client-common
  ${common-hdr}
  ${common-src})
set_property(TARGET client-common PROPERTY CXX_STANDARD 20)
target_include_directories(client-common PUBLIC include)

if (EMSCRIPTEN)
  add_executable(web-client "web/embind.cc")
  set_property(TARGET web-client PROPERTY CXX_STANDARD 20)
  target_link_libraries(web-client PRIVATE client-common)

  target_link_options(web-client
    PUBLIC "SHELL:--bind -s WASM=1 -s MODULARIZE=1 -s ENVIRONMENT=web -s WASM_BIGINT=1"
    PUBLIC "SHELL:-s EXIT_RUNTIME=0 -s FILESYSTEM=0 -O3"
    PUBLIC "SHELL:-s EXPORT_NAME=BenchmarkWebClient -lembind")
  target_compile_options(web-client PUBLIC -O3)
else ()
  FetchContent_Declare(
    concurrentqueue
    GIT_REPOSITORY "https://github.com/cameron314/concurrentqueue"
    GIT_TAG "65d6970912fc3f6bb62d80edf95ca30e0df85137"
  )
  FetchContent_MakeAvailable(concurrentqueue)

  set(native-client-hdr
    "native/netclient.h")
  set(native-client-src
    "native/netclient.cc")
  add_executable(native-client
    ${native-client-hdr}
    ${native-client-src}
    "native/main.cc")
  target_link_libraries(native-client PRIVATE concurrentqueue)
  set_property(TARGET native-client PROPERTY CXX_STANDARD 20)
  target_link_libraries(native-client PRIVATE client-common)
  if (WIN32)
    target_link_libraries(native-client PRIVATE wsock32 ws2_32)
  endif ()
endif ()

