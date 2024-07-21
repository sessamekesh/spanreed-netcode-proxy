FROM alpine:3.20.1 AS build

RUN apk update && \
  apk add --no-cache \
  cmake=3.29.3-r0 \
  build-base=0.5-r3 \
  git=2.45.2-r0 \
  openssl=3.3.1-r3

WORKDIR /spanreed/examples/hello-spanreed/server
COPY include/ ./include/
COPY main.cc .
COPY CMakeLists.txt .

WORKDIR /spanreed/examples/hello-spanreed/server/build
RUN cmake -DCMAKE_BUILD_TYPE=Release .. && \
  cmake --build . --parallel 8

FROM alpine:3.20.1

RUN apk update && \
  apk add --no-cache \
  libstdc++=13.2.1_git20240309-r0

RUN addgroup -S appserverg && adduser -S appserveru -G appserverg
USER appserveru

COPY --chown=appserveru:appserveru --from=build \
  ./spanreed/examples/hello-spanreed/server/build/hello-spanreed-server \
  ./app/

EXPOSE 30001/udp
ENTRYPOINT [ "./app/hello-spanreed-server" ]