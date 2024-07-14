Trivially simple broadcast client.

Clients click on the canvas, and a little dot appears and fades out over time. That's it. That's the game.

Little chat interface too.

WebSocket frontend, UDP backend.

### Generating TS flatbuffers

```
cd client/src/gen
flatc --ts ../../../client_message.fbs ../../../server_message.fbs --ts-no-import-ext
```
