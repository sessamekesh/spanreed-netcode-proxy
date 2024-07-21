FROM golang:1.22-alpine3.20 AS build

WORKDIR /spanreed
COPY go.mod .
COPY go.sum .
COPY internal internal/
COPY pkg pkg/
COPY cmd cmd/
RUN go mod download

WORKDIR /spanreed/cmd/spanreed-hub
RUN CGO_ENABLED=0 go build -o ./spanreed-hub

FROM alpine:3.20.1

RUN addgroup -S appserverg && adduser -S appserveru -G appserverg
USER appserveru

COPY --chown=appserveru:appserveru --from=build \
  /spanreed/cmd/spanreed-hub/spanreed-hub \
  ./spanreed-hub

EXPOSE 3000/tcp
EXPOSE 3000/udp

ENTRYPOINT [ "./spanreed-hub", \
  "-allow-all-hosts", \
  "-cert", "/certs/cert.pem", \
  "-key", "/certs/key.pem", \
  "-ws-port", "3000", \
  "-webtransport", \
  "-wt-port", "3000" \
  ]