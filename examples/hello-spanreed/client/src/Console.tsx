import React from 'react';
import { LogLevel } from './log';

const CONSOLE_WARN: React.CSSProperties = {
  color: '#eed202',
};
const CONSOLE_ERROR: React.CSSProperties = {
  color: '#cc3300',
};
const CONSOLE_INFO: React.CSSProperties = {
  color: '#40a6ce',
};

interface ConsoleProps {
  lines: Array<{ msg: string; logLevel: LogLevel }>;
}
export const Console: React.FC<ConsoleProps> = React.memo(({ lines }) => {
  return (
    <div
      style={{
        height: '80px',
        resize: 'vertical',
        maxWidth: '80ch',
        backgroundColor: '#282A36',
        borderRadius: '4px',
        border: '1px solid #aaa',
        fontFamily: "'Courier New', Courier, monospace",
        padding: '4px',
        margin: '8px',
        overflow: 'scroll',
        textOverflow: 'ellipsis',
        whiteSpace: 'nowrap',
      }}
    >
      {lines.map((line, idx) => {
        let MsgCssExtra: React.CSSProperties = {};
        switch (line.logLevel) {
          case LogLevel.Warning:
            MsgCssExtra = CONSOLE_WARN;
            break;
          case LogLevel.Error:
            MsgCssExtra = CONSOLE_ERROR;
            break;
          case LogLevel.Info:
            MsgCssExtra = CONSOLE_INFO;
            break;
        }
        return (
          <span>
            <span
              style={{
                color: '#aaa',
                borderRight: '1px solid #eee',
                marginRight: '1ch',
                paddingRight: '0.25ch',
                marginLeft: '0.5ch',
              }}
              id={`msgidx-${idx}`}
            >
              {idx + 1}
            </span>
            <span style={{ ...MsgCssExtra }} id={`msg-${idx}`}>
              {line.msg}
            </span>
            <br />
          </span>
        );
      })}
    </div>
  );
});
