export enum LogLevel {
  Debug,
  Info,
  Warning,
  Error,
}

export type LogCb = (msg: string, level?: LogLevel) => void;

export function WrapLogFn(name: string, logCb: LogCb): LogCb {
  return (msg, level) => {
    logCb(`[${name}] ${msg}`, level);
  };
}

export const DefaultLogCb: LogCb = (msg, level: LogLevel = LogLevel.Info) => {
  switch (level) {
    case LogLevel.Debug:
    case LogLevel.Info:
      console.log(msg);
      break;
    case LogLevel.Warning:
      console.warn(msg);
      break;
    case LogLevel.Error:
      console.error(msg);
      break;
  }
};
