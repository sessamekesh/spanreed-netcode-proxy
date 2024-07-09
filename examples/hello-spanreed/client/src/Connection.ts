type WebsocketConnection = {
  type: 'websocket',
  spanreedConnection: WebSocket,
};

type WebtransportConnection = {
  type: 'webtransport',
  spanreedConnection: WebTransport,
};

export type Connection = WebsocketConnection | WebtransportConnection;

export async function connectWebsocket(spanreedAddress: string, backendAddress: string): Promise<Connection> {
  const ws = new WebSocket(spanreedAddress);
  ws.binaryType = 'arraybuffer';

  await new Promise((resolve, reject) => {
    ws.onerror = reject;
    ws.onopen = resolve;
  });

  ws.onerror = null;
  ws.onmessage = null;

  // TODO (sessamekehs): Auth workflow here!

  return { type: 'websocket', spanreedConnection: ws };
}
