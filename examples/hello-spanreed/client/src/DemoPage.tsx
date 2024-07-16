import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { DemoCanvas } from './DemoCanvas';
import { DefaultLogCb, LogLevel, WrapLogFn } from './log';
import { Console } from './Console';
import { ConnectionStateComponent } from './ConnectionStateComponent';
import { Connection } from './Connection';
import * as flatbuffers from 'flatbuffers';
import {
  Color,
  Message,
  ServerChatMessage,
  ServerGameStateMessage,
  ServerMessage,
} from './gen/hello-spanreed';
import { COLORS } from './colors';
import { UserClickMessage } from './gen/hello-spanreed/user-click-message';
import { ClientMessage } from './gen/hello-spanreed/client-message';
import { UserMessage } from './gen/hello-spanreed/user-message';
import { ChatBox } from './ChatBox';
import { UserChatMessage } from './gen/hello-spanreed/user-chat-message';
import { UserPingMessage } from './gen/hello-spanreed/user-ping-message';

function mapColor(color: Color) {
  switch (color) {
    case Color.Blue:
      return COLORS.Blue;
    case Color.FireOrange:
      return COLORS.FireOrange;
    case Color.Green:
      return COLORS.Green;
    case Color.Indigo:
      return COLORS.Indigo;
    case Color.Red:
    default:
      return COLORS.Red;
  }
}

export const DemoPage: React.FC = () => {
  const [lines, setLines] = useState<
    Array<{ msg: string; logLevel: LogLevel }>
  >([]);
  const [connection, setConnection] = useState<Connection>();
  const [dots, setDots] = useState<
    Array<{
      x: number;
      y: number;
      radius: number;
      color: Uint8Array;
    }>
  >();

  const LogFn = useCallback(
    (msg: string, logLevel: LogLevel = LogLevel.Info) => {
      DefaultLogCb(msg, logLevel);
      console.log(msg);
      setLines((old) => [...old, { msg, logLevel }]);
    },
    [setLines, DefaultLogCb]
  );
  const log = useMemo(() => WrapLogFn('DemoPage', LogFn), [WrapLogFn, LogFn]);

  const onClickCanvas = useCallback(
    (x: number, y: number) => {
      if (!connection) return;

      const appDataFbb = new flatbuffers.Builder(64);
      const pClickMessage = UserClickMessage.createUserClickMessage(
        appDataFbb,
        x,
        y
      );
      const pClientMessage = ClientMessage.createClientMessage(
        appDataFbb,
        UserMessage.UserClickMessage,
        pClickMessage
      );
      ClientMessage.finishClientMessageBuffer(appDataFbb, pClientMessage);
      const appData = appDataFbb.asUint8Array();

      if (connection.type === 'websocket') {
        connection.spanreedConnection.send(appData);
        return;
      } else if (connection.type === 'webtransport') {
        connection.writer.write(appData);
        return;
      }
    },
    [connection]
  );

  const onSubmitChat = useCallback(
    (chat: string) => {
      if (!connection) return;

      const appDataFbb = new flatbuffers.Builder(64);
      const pChatString = appDataFbb.createString(chat);
      const pChatMessage = UserChatMessage.createUserChatMessage(
        appDataFbb,
        pChatString
      );
      const pClientMessage = ClientMessage.createClientMessage(
        appDataFbb,
        UserMessage.UserChatMessage,
        pChatMessage
      );
      ClientMessage.finishClientMessageBuffer(appDataFbb, pClientMessage);
      const payload = appDataFbb.asUint8Array();

      if (connection.type === 'websocket') {
        connection.spanreedConnection.send(payload);
        return;
      } else if (connection.type === 'webtransport') {
        connection.writer.write(payload);
        return;
      }
    },
    [connection]
  );

  useEffect(() => {
    if (connection === undefined) return;

    if (connection.type === 'websocket') {
      connection.spanreedConnection.onclose = (closeEvent) => {
        log('Spanreed connection is closing!', LogLevel.Info);
        setConnection(undefined);
      };
      connection.spanreedConnection.onmessage = (msg) => {
        if (!(msg.data instanceof ArrayBuffer)) {
          log('Non-ArrayBuffer for websocket message!', LogLevel.Warning);
          return;
        }

        const data = msg.data as ArrayBuffer;

        const fbb = new flatbuffers.ByteBuffer(new Uint8Array(data));
        const serverMessage = ServerMessage.getRootAsServerMessage(fbb);

        switch (serverMessage.messageType()) {
          case Message.ServerChatMessage: {
            const gameMessage = serverMessage.message(
              new ServerChatMessage()
            ) as ServerChatMessage;
            LogFn(
              `[${gameMessage.user()}]: ${gameMessage.text()}`,
              LogLevel.UserChat
            );
            break;
          }
          case Message.ServerGameStateMessage: {
            const gameState = serverMessage.message(
              new ServerGameStateMessage()
            ) as ServerGameStateMessage;
            const gameDots: typeof dots = [];
            const dotsCt = gameState.dotsLength();
            for (let dotIdx = 0; dotIdx < dotsCt; dotIdx++) {
              const dot = gameState.dots(dotIdx);
              if (!dot) continue;
              gameDots.push({
                color: mapColor(dot.color()),
                radius: dot.radius(),
                x: dot.x(),
                y: dot.y(),
              });
            }
            setDots(gameDots);
            break;
          }
          default:
            log(
              'Unexpected server message type, cannot process',
              LogLevel.Warning
            );
            break;
        }
      };

      return () => {
        setDots(undefined);
        connection.spanreedConnection.close();
      };
    } else if (connection.type === 'webtransport') {
      connection.spanreedConnection.closed.then((closeEvent) => {
        log('Spanreed connection is closing naturally', LogLevel.Info);
      }).catch((e) => {
        log(`Spanreed connection closed with error: ${e}`, LogLevel.Error);
      }).finally(() => {
        setConnection(undefined);
      });

      (async () => {
        while (true) {
          const { done, value } = await connection.reader.read();
          if (done || !value) {
            break;
          }

          const fbb = new flatbuffers.ByteBuffer(new Uint8Array(value));
          const serverMessage = ServerMessage.getRootAsServerMessage(fbb);

          switch (serverMessage.messageType()) {
            case Message.ServerChatMessage: {
              const gameMessage = serverMessage.message(
                new ServerChatMessage()
              ) as ServerChatMessage;
              LogFn(
                `[${gameMessage.user()}]: ${gameMessage.text()}`,
                LogLevel.UserChat
              );
              break;
            }
            case Message.ServerGameStateMessage: {
              const gameState = serverMessage.message(
                new ServerGameStateMessage()
              ) as ServerGameStateMessage;
              const gameDots: typeof dots = [];
              const dotsCt = gameState.dotsLength();
              for (let dotIdx = 0; dotIdx < dotsCt; dotIdx++) {
                const dot = gameState.dots(dotIdx);
                if (!dot) continue;
                gameDots.push({
                  color: mapColor(dot.color()),
                  radius: dot.radius(),
                  x: dot.x(),
                  y: dot.y(),
                });
              }
              setDots(gameDots);
              break;
            }
            default:
              log(
                'Unexpected server message type, cannot process',
                LogLevel.Warning
              );
              break;
          }
        }

        connection.writer.abort().catch((e) => {
          log(`Writable stream abort error: ${e}`, LogLevel.Error);
        });
      })();

      return () => {
        setDots(undefined);
        // connection.writer.abort().then(() => {
        //   log(`Successfully closed writable stream`, LogLevel.Info);
        // }).catch((e) => {
        //   log(`Writable stream abort error: ${e}`, LogLevel.Error);
        // });
        connection.spanreedConnection.close({
          closeCode: 0,
          reason: 'Client requested shutdown',
        });
      };
    }
  }, [connection, setConnection, setDots, log, LogFn]);

  useEffect(() => {
    if (!connection) return;

    const hInterval = setInterval(() => {
      const appDataFbb = new flatbuffers.Builder(64);
      const pPingMessage = UserPingMessage.createUserPingMessage(appDataFbb);
      const pClientMessage = ClientMessage.createClientMessage(
        appDataFbb,
        UserMessage.UserPingMessage,
        pPingMessage
      );
      ClientMessage.finishClientMessageBuffer(appDataFbb, pClientMessage);
      const payload = appDataFbb.asUint8Array();

      if (connection.type === 'websocket') {
        connection.spanreedConnection.send(payload);
        return;
      } else if (connection.type === 'webtransport') {
        connection.writer.write(payload);
        return;
      }
    }, 3000);

    return () => clearInterval(hInterval);
  }, [connection]);

  return (
    <OuterContainer>
      <span
        style={{
          alignSelf: 'center',
          fontSize: '24pt',
          fontWeight: 700,
          marginTop: '4px',
        }}
      >
        Hello Spanreed Client
      </span>
      <DemoCanvas logFn={LogFn} dots={dots} onClick={onClickCanvas} />
      <ConnectionStateComponent
        logCb={LogFn}
        connection={connection}
        onOpenConnection={setConnection}
      />
      {connection !== undefined && (
        <ChatBox Nickname={connection.userName} onSubmit={onSubmitChat} />
      )}
      <Console lines={lines} />
    </OuterContainer>
  );
};

interface OuterContainerProps {
  children?: React.ReactNode;
}
const OuterContainer: React.FC<OuterContainerProps> = ({ children }) => {
  return (
    <div
      style={{
        border: '4px solid #fff',
        borderRadius: '8px',
        display: 'flex',
        flexDirection: 'column',
        maxWidth: '800px',
        marginLeft: 'auto',
        marginRight: 'auto',
      }}
    >
      {children}
    </div>
  );
};
