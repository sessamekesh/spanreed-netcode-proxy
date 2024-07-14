import React from 'react';
import { Button } from './common/Button';

interface ChatBoxProps {
  Nickname: string;
  onSubmit: (msg: string) => void;
}

export const ChatBox: React.FC<ChatBoxProps> = ({ Nickname, onSubmit }) => {
  const [chatMsg, setChatMsg] = React.useState('');

  return <div style={{
    display: 'flex',
    flexDirection: 'column',
    padding: '8px 12px',
    border: '2px solid #ccc',
  }}>
    <p style={{ margin: 0, fontSize: '14pt', fontWeight: 700, color: '#ccc' }}>
      Send chat as: {Nickname}
    </p>
    <div style={{ display: 'flex', flexDirection: 'row' }}>
      <input type="text"
        value={chatMsg} onChange={(e) => setChatMsg(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === 'Enter') {
            onSubmit(chatMsg);
            setChatMsg('');
          }
        }}
        style={{
          border: '2px solid #88cc88',
          backgroundColor: '#3E4544',
          color: '#eee',
          resize: 'none',
          minWidth: '250px',
          borderRadius: '4px',
          padding: '4px 8px',
          margin: '8px 12px 8px 0',
          flexGrow: 1,
        }}></input>
      <Button disabled={chatMsg.length === 0} onClick={() => {
        onSubmit(chatMsg);
        setChatMsg('');
      }}>Submit!</Button>
    </div>
  </div>;
};