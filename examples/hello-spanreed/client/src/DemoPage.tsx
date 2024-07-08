import React, { useCallback, useState } from 'react';
import { DemoCanvas } from './DemoCanvas';
import { DefaultLogCb, LogLevel } from './log';
import { Console } from './Console';
import { ConnectionStateComponent } from './ConnectionStateComponent';

export const DemoPage: React.FC = () => {
  const [lines, setLines] = useState<
    Array<{ msg: string; logLevel: LogLevel }>
  >([]);

  const LogFn = useCallback(
    (msg: string, logLevel: LogLevel = LogLevel.Info) => {
      DefaultLogCb(msg, logLevel);
      setLines((old) => [...old, { msg, logLevel }]);
    },
    [setLines]
  );

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
      <DemoCanvas logFn={LogFn} />
      <ConnectionStateComponent />
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
