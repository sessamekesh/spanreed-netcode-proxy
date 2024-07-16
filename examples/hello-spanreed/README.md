# Hello Spanreed

Trivially simple broadcast client.

Clients click on the canvas, and a little dot appears and fades out over time. That's it. That's the game.

Little chat interface too.

WebSocket frontend, UDP backend.

# Starting

## Client

The client is a React app, nothing too much to it.

```
cd client
yarn install
yarn start
```

Open a browser to localhost:3000 to test.

## Server

The server is a C++ application, unfortunately pretty Windows specific right now but I'd love to get UNIX socket support too.

```
cd server
mkdirn bin
cd bin
cmake ..
ninja hello-spanreed-server
./hello-spanreed-server.exe
```

## Spanreed Proxy

To test using only WebSockets, just start the basic server with no options:

```
go run ../../cmd/spanreed-hub/main.go
```

To use WebTransport locally, use something like [mkcert](https://github.com/FiloSottile/mkcert) to set up a local certificate authority (CA) for testing. Alternatively, you can use something like `openssl` to generate a public/private key pair, and tell your browser that it's OK to use that particular pair.

Whatever route you go, run the proxy with additional flags to support WebTransport:

```
go run ../../cmd/spanreed-hub/main.go -webtransport -wt-port 3000 -cert /path/to/cert.pem -key /path/to/cert-key.pem
```

See the [spanreed-hub README file](../../cmd/spanreed-hub/README.md) for more information about running the proxy.

### Generating TS flatbuffers

```
cd client/src/gen
flatc --ts ../../../client_message.fbs ../../../server_message.fbs --ts-no-import-ext
```
