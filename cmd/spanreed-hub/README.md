# Spanreed Hub

Basic, default Spanreed proxy implementation.

Supports WebSocket and WebTransport client connections, supports TCP and UDP destination server connections.

## Arguments

`-websockets` _default: true_: Set to false to disable WebSocket support.
`-ws-port` _default: 3000_: Port on which the WebSocket server should run.
`-ws-endpoint` _default: "/ws"_: HTTP endpoint that listens for WebSocket connections.

`-udp` _default: true_: Set to false to disable UDP support