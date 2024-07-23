# Spanreed Hub

Basic, default Spanreed proxy implementation.

Supports WebSocket and WebTransport client connections, supports TCP and UDP destination server connections.

## Parameters

Use a `.env` file to override any of the following. An example `.env` file with comments is in `.env.example` (which is not automatically loaded).

`SPANREED_TLS_CERT_PATH` _default: ""_: Path to TLS cert file.
`SPANREED_TLS_KEY_PATH` _default: ""_: Path to TLS key file.

`SPANREED_WEBSOCKET` _default: true_: Set to false to disable WebSocket support.
`SPANREED_WEBSOCKET_PORT` _default: 3000_: Port on which the WebSocket server should run.
`SPANREED_WEBSOCKET_ENDPOINT` _default: "/ws"_: HTTP endpoint that listens for WebSocket connections.
`SPANREED_WEBSOCKET_READ_BUFFER_SIZE` _default: 10 KB_: Size of internal data read buffer for a WebSocket connection.
`SPANREED_WEBSOCKET_WRITE_BUFFER_SIZE` _default: 10 KB_: Size of internal data write buffer for a WebSocket connection.
`SPANREED_WEBSOCKET_HANDSHAKE_TIMEOUT_MS` _default: 1500_: Number of milliseconds to wait for WebSocket handshake before returning an error.

`SPANREED_WEBTRANSPORT` _default: false_: Set true to enable WebTransport support. Requires cert and key to be set as well.
`SPANREED_WEBTRANSPORT_PORT` _default: 3001_: Port on which the WebTransport server should run.
`SPANREED_WEBTRANSPORT_ENDPOINT` _default: "/wt"_: HTTP endpoint that listens for WebTransport connections.

`SPANREED_UDP` _default: true_: Set to false to disable UDP support
`SPANREED_UDP_PORT` _default: 30321_: Port on which UDP server operates.

`SPANREED_ALLOW_ALL_HOSTS` _default: false_: Set true to accept connections from all hosts (except forbidden hosts)
`SPANREED_DENY_HOSTS` _default: ""_: Comma separated list of forbidden hosts
`SPANREED_ALLOW_HOSTS` _default: ""_: Comma separated list of allowed hosts (if allow-all-hosts is false)

`SPANREED_CLIENT_INCOMING_MESSAGE_QUEUE_LENGTH` _default: 16_ Length of per-client internal incoming message queue.
`SPANREED_CLIENT_OUTGOING_MESSAGE_QUEUE_LENGTH` _default: 16_ Length of per-client internal outgoing message queue.