cmake_minimum_required(VERSION 3.10)

project(
  HelloSpanreedServer
  VERSION 1.0
  LANGUAGES CXX)

# Flatbuffers (used for handling messages)
include(FetchContent)

FetchContent_Declare(
  flatbuffers
  GIT_REPOSITORY "https://github.com/google/flatbuffers"
  GIT_TAG "v24.3.25"
)
FetchContent_MakeAvailable(flatbuffers)

FetchContent_Declare(
  concurrentqueue
  GIT_REPOSITORY "https://github.com/cameron314/concurrentqueue"
  GIT_TAG "65d6970912fc3f6bb62d80edf95ca30e0df85137"
)
FetchContent_MakeAvailable(concurrentqueue)

add_executable(hello-spanreed-server main.cc)
set_property(TARGET hello-spanreed-server PROPERTY CXX_STANDARD 17)
target_link_libraries(hello-spanreed-server PUBLIC flatbuffers concurrentqueue wsock32 ws2_32)
target_include_directories(hello-spanreed-server PRIVATE include)
target_include_directories(hello-spanreed-server PRIVATE ../../../flatbuffers/generated/cc/include)