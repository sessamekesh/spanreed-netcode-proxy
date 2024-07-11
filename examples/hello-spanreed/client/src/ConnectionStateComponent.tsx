import React from 'react';
import { Connection, connectWebsocket } from './Connection';
import { Button } from './common/Button';
import { DefaultLogCb, LogCb, LogLevel, WrapLogFn } from './log';

interface ConnectionStateComponentProps {
  connection?: Connection;
  onOpenConnection?: (connection: Connection) => void;
  logCb?: LogCb;
}

export const ConnectionStateComponent: React.FC<ConnectionStateComponentProps> =
  React.memo(({ connection, logCb }) => {
    if (connection === undefined) {
      return <SetupConnectionComponent logCb={logCb} />;
    }
    return <span>Connection state component</span>;
  });

const DiagBtnCommonStyle: React.CSSProperties = {
  position: 'absolute', float: 'left', width: '130px', height: '0', cursor: 'pointer',
  userSelect: 'none',
};

const UrlInputStyle: React.CSSProperties | Record<string, React.CSSProperties> = {
  minWidth: '250px',
  borderRadius: '4px',
  border: '2px solid #88cc88',
  backgroundColor: '#3E4544',
  color: '#eee',
  padding: '4px 8px',
}

interface SetupConnectionComponentProps {
  onOpenConnection?: (connection: Connection) => void;
  logCb?: LogCb;
}
const SetupConnectionComponent: React.FC<SetupConnectionComponentProps> = React.memo(({ onOpenConnection, logCb }) => {
  const [loading, setLoading] = React.useState(false);
  const [leftSelected, setLeftSelected] = React.useState(true);

  const [userName, setUserName] = React.useState('Sessamekesh');
  const [spanUrl, setSpanUrl] = React.useState("ws://localhost:3000/ws");
  const [destUrl, setDestUrl] = React.useState("udp:localhost:30001");

  const log = WrapLogFn('SetupConnectionComponent', logCb ?? DefaultLogCb);

  const onClickConnect = React.useCallback(async () => {
    log('Attempting to connect...');
    setLoading(true);

    if (leftSelected) {
      try {
        const wsConnection = await connectWebsocket(spanUrl, destUrl, userName, logCb ?? DefaultLogCb);
        if (wsConnection.type !== 'websocket') {
          log(`Not a WebSocket connection!`, LogLevel.Warning);
          return;
        }

        wsConnection.spanreedConnection.onerror = (e) => {
          log(`WebSocket error`, LogLevel.Error);
        };
        wsConnection.spanreedConnection.onmessage = (msg) => {
          log(`WebSocket message (size=${(msg.data as ArrayBuffer).byteLength})`, LogLevel.Info);
        };
        wsConnection.spanreedConnection.onclose = () => {
          log(`WebSocket connection closed`, LogLevel.Warning);
        };
      } catch (e) {
        log(`Error connecting to WebSocket! ${(e instanceof Error ? e.message : '<unknown>')}`, LogLevel.Error);
      }
    }

    setLoading(false);
  }, [setLoading, spanUrl, destUrl]);

  return <div style={{
    display: 'flex',
    flexDirection: 'column',
    padding: '16px 12px',
    margin: '4px 12px',
    borderRadius: '4px',
    border: '1px solid #eee',
  }}>
    {/* Header */}
    <div style={{ display: 'flex', flexDirection: 'row', justifyContent: 'space-between', width: '100%', alignItems: 'center' }}>
      <p style={{ margin: 0, fontSize: '14pt', fontWeight: 700, color: '#ccc' }}>Open Connection</p>
      <div style={{
        marginLeft: '20px', position: 'relative', width: '280px', border: '2px solid #fff', height: '30px', textAlign: 'center',
      }}
        onClick={() => setLeftSelected(!leftSelected)}>
        <div
          style={{
            ...DiagBtnCommonStyle,
            left: 0,
            borderTop: '30px solid black',
            borderRight: '20px solid transparent',
            zIndex: 1,
            color: '#888',
            ...(leftSelected && { borderTop: '30px solid #4538e5', color: '#fff' }),
          }}>
          <span style={{ position: 'relative', float: 'left', width: '100%', height: 'auto', margin: 0, padding: 0, top: '-27px' }}>
            WebSocket
          </span>
        </div>
        <div style={{
          borderRight: '2px solid #fff',
          height: '36px',
          top: '-3px',
          position: 'absolute',
          right: '50%',
          transform: 'rotate(34deg) translateZ(0px)',
          zIndex: 1,
        }}></div>
        <div
          style={{
            ...DiagBtnCommonStyle,
            right: 0,
            zIndex: 2,
            borderLeft: '20px solid transparent',
            borderBottom: '30px solid black',
            color: '#888',
            ...(!leftSelected && { borderBottom: '30px solid #4538e5', color: '#fff' }),
          }}>
          <span style={{ position: 'relative', float: 'left', width: '100%', height: 'auto', margin: 0, padding: 0, top: '3px' }}>
            WebTransport
          </span>
        </div>
      </div>
    </div>

    {/* Server address inputs */}
    <form style={{ marginBottom: '24px', marginTop: '12px', display: 'grid', gridTemplateColumns: '2fr 5fr', gap: '10px' }}>
      <label htmlFor="tb-username">Spanreed Proxy Address:</label>
      <input type="text" id="tb-username"
        placeholder={'Sessamekesh'}
        style={{
          ...UrlInputStyle,
        }}
        value={userName}
        onChange={(e) => loading ? setUserName(userName) : setUserName(e.target.value)}></input>

      <label htmlFor="tb-spanreed-url">Spanreed Proxy Address:</label>
      <input type="url" id="tb-spanreed-url"
        pattern={leftSelected ? 'wss?://.*' : 'https://.*'}
        placeholder={leftSelected ? 'ws://spanreed.example.com/ws' : 'https://spanreed.example.com/wt'}
        style={{
          ...UrlInputStyle,
        }}
        value={spanUrl}
        onChange={(e) => loading ? setSpanUrl(spanUrl) : setSpanUrl(e.target.value)}></input>

      <label htmlFor='tb-backend-url'>Demo Server Address:</label>
      <input id='tb-backend-url' type="text" placeholder='URL or IP address'
        style={{
          ...UrlInputStyle,
        }}
        value={destUrl}
        onChange={(e) => loading ? setDestUrl(destUrl) : setDestUrl(e.target.value)}></input>
    </form>

    {/* Submit! */}
    <Button disabled={loading} onClick={onClickConnect}><span>Connect!</span></Button>
  </div >
});