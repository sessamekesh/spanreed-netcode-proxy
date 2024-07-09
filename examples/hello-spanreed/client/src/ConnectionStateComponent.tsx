import React from 'react';
import { Connection } from './Connection';

interface ConnectionStateComponentProps {
  connection?: Connection;
  onOpenConnection?: (connection: Connection) => void;
}

export const ConnectionStateComponent: React.FC<ConnectionStateComponentProps> =
  React.memo(({ connection }) => {
    if (connection === undefined) {
      return <SetupConnectionComponent />;
    }
    return <span>Connection state component</span>;
  });

const DiagBtnCommonStyle: React.CSSProperties = {
  position: 'absolute', float: 'left', width: '130px', height: '0', cursor: 'pointer',
  userSelect: 'none',
};

interface SetupConnectionComponentProps {
  onOpenConnection?: (connection: Connection) => void;
}
const SetupConnectionComponent: React.FC<SetupConnectionComponentProps> = React.memo(({ onOpenConnection }) => {
  const [loading, setLoading] = React.useState(false);
  const [leftSelected, setLeftSelected] = React.useState(true);

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
    <div>
      <span>Spanreed Proxy Address:</span>
      <input type="text"></input>
    </div>
    <div>
      <span>Demo Server Address:</span>
      <input type="text"></input>
    </div>

    {/* Submit! */}
    <div>
      <span>Connect!</span>
    </div>
  </div>
});