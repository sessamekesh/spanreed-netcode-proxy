import { Builder, ByteBuffer } from 'flatbuffers';
import { LogCb, LogLevel, WrapLogFn } from './log';
import { UserConnectMessage } from './gen/hello-spanreed/user-connect-message';
import { ClientMessage } from './gen/hello-spanreed/client-message';
import { UserMessage } from './gen/hello-spanreed/user-message';
import { ConnectClientMessage } from './gen/spanreed-message/connect-client-message';
import { ConnectClientVerdict } from './gen/spanreed-message/connect-client-verdict';

type WebsocketConnection = {
  type: 'websocket';
  spanreedConnection: WebSocket;
  userName: string;
};

type WebtransportConnection = {
  type: 'webtransport';
  spanreedConnection: WebTransport;
  userName: string;
};

export type Connection = WebsocketConnection | WebtransportConnection;

export async function connectWebsocket(
  spanreedAddress: string,
  backendAddress: string,
  userName: string,
  logCb: LogCb
): Promise<Connection> {
  const log = WrapLogFn('connectWebsocket', logCb);
  const ws = new WebSocket(spanreedAddress);
  ws.binaryType = 'arraybuffer';

  log(
    `Attempting to open WebSocket connection on ${spanreedAddress} for userName=${userName}...`,
    LogLevel.Debug
  );
  await new Promise((resolve, reject) => {
    ws.onerror = (connectError) => {
      if (connectError.currentTarget instanceof WebSocket) {
        if (connectError.currentTarget.readyState === WebSocket.CLOSED) {
          reject(new Error('Failed to connect to proxy'));
        }
      }
      reject(connectError);
    };
    ws.onopen = resolve;
  });

  ws.onerror = null;
  ws.onmessage = null;

  const appDataFbb = new Builder(64);
  const pUserNameString = appDataFbb.createString(userName);
  const pUserConnectMessage = UserConnectMessage.createUserConnectMessage(
    appDataFbb,
    pUserNameString
  );
  const pClientMessage = ClientMessage.createClientMessage(
    appDataFbb,
    UserMessage.UserConnectMessage,
    pUserConnectMessage
  );
  ClientMessage.finishClientMessageBuffer(appDataFbb, pClientMessage);
  const appData = appDataFbb.asUint8Array();

  const fbb = new Builder(64);
  const pBackendAddress = fbb.createString(backendAddress);
  const pAppData = fbb.createByteVector(appData);
  const pConnectClientMessage = ConnectClientMessage.createConnectClientMessage(
    fbb,
    pBackendAddress,
    pAppData
  );
  ConnectClientMessage.finishConnectClientMessageBuffer(
    fbb,
    pConnectClientMessage
  );
  const connectClientPayload = fbb.asUint8Array();

  log(
    `Connection opened! Attempting to authorize (payload size=${connectClientPayload.length})...`,
    LogLevel.Debug
  );
  ws.send(connectClientPayload);

  await new Promise<void>((resolve, reject) => {
    ws.onmessage = (msg) => {
      const fbb = new ByteBuffer(new Uint8Array(msg.data as ArrayBuffer));
      const verdict = ConnectClientVerdict.getRootAsConnectClientVerdict(fbb);

      if (verdict.accepted()) {
        resolve();
      } else {
        reject(
          `destination rejected connection: ${verdict.errorReason() ?? '<unknown reason>'}`
        );
      }
    };
    ws.onclose = (e) => {
      reject(e);
    };
  }).catch((e) => {
    ws.close();
    throw e;
  });

  ws.onclose = null;
  ws.onmessage = null;

  log(
    `Connection opened successfully! Messages may now be forwarded to the destination server through this WebSocket connection.`,
    LogLevel.Debug
  );

  return { type: 'websocket', spanreedConnection: ws, userName };
}
