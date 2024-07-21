# Spanreed Hub

Basic, default Spanreed proxy implementation.

Supports WebSocket and WebTransport client connections, supports TCP and UDP destination server connections.

## Arguments

`-cert` _default: ""_: Path to TLS cert file.
`-key` _default: ""_: Path to TLS key file.

`-websockets` _default: true_: Set to false to disable WebSocket support.
`-ws-port` _default: 3000_: Port on which the WebSocket server should run.
`-ws-endpoint` _default: "/ws"_: HTTP endpoint that listens for WebSocket connections.
`-ws-read-buffer-size` _default: 10 KB_: Size of internal data read buffer for a WebSocket connection.
`-ws-write-buffer-size` _default: 10 KB_: Size of internal data write buffer for a WebSocket connection.
`ws-handshake-timeout` _default: 1500_: Number of milliseconds to wait for WebSocket handshake before returning an error.

`-webtransport` _default: false_: Set true to enable WebTransport support. Requires cert and key to be set as well.
`-wt-port` _default: 3001_: Port on which the WebTransport server should run.
`-wt-endpoint` _default: "/wt"_: HTTP endpoint that listens for WebTransport connections.

`-udp` _default: true_: Set to false to disable UDP support
`-udp-port` _default: 30321_: Port on which UDP server operates.

`-allow-all-hosts` _default: false_: Set true to accept connections from all hosts (except forbidden hosts)
`-deny-hosts` _default: ""_: Comma separated list of forbidden hosts
`-allow-hosts` _default: ""_: Comma separated list of allowed hosts (if allow-all-hosts is false)

`-client-incoming-queue` _default: 16_ Length of per-client internal incoming message queue.
`-client-outgoing-queue` _default: 16_ Length of per-client internal outgoing message queue.