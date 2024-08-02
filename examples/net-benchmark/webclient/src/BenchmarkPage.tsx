import React, { useCallback, useState } from "react";
import { DefaultLogCb, LogLevel } from "./log";
import { Console } from "./Console";
import { BenchmarkParams } from "./BenchmarkParams";

export const BenchmarkPage: React.FC = () => {
  const [lines, setLines] = useState<
    Array<{ msg: string; logLevel: LogLevel }>
  >([]);

  const LogFn = useCallback(
    (msg: string, logLevel: LogLevel = LogLevel.Info) => {
      DefaultLogCb(msg, logLevel);
      setLines((old) => [...old, { msg, logLevel }]);
    }, [setLines]);

  const RunBenchmark = useCallback(async (spanUrl: string, destUrl: string, pingCt: number, gapMs: number, payloadSize: number) => {
    LogFn(`Starting benchmark (ct=${pingCt} gapms=${gapMs} msglen=${payloadSize})`);

    // TODO (sessamekesh): Start up WebTransport connection + benchmark stuff
    await new Promise((resolve) => setTimeout(resolve, 5000));

    LogFn('Benchmark finished!');
  }, [LogFn]);

  return (
    <OuterContainer>
      <span
        style={{
          alignSelf: "center",
          fontSize: "24pt",
          fontWeight: 700,
          marginTop: "4px",
        }}
      >
        Hello Spanreed Client
      </span>
      <BenchmarkParams logCb={LogFn} onRunBenchmark={RunBenchmark} />
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
        border: "4px solid #fff",
        borderRadius: "8px",
        display: "flex",
        flexDirection: "column",
        maxWidth: "800px",
        marginLeft: "auto",
        marginRight: "auto",
      }}
    >
      {children}
    </div>
  );
};
