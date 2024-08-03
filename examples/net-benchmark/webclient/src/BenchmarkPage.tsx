import React, { useCallback, useEffect, useState } from "react";
import { DefaultLogCb, LogLevel } from "./log";
import { Console } from "./Console";
import { BenchmarkParams } from "./BenchmarkParams";
import { BenchmarkWasmModule } from "./BenchmarkWasmWrapper";

export const BenchmarkPage: React.FC = () => {
  const [lines, setLines] = useState<
    Array<{ msg: string; logLevel: LogLevel }>
  >([]);
  const [benchmarkModule, setBenchmarkModule] = useState<BenchmarkWasmModule>();

  const LogFn = useCallback(
    (msg: string, logLevel: LogLevel = LogLevel.Info) => {
      DefaultLogCb(msg, logLevel);
      setLines((old) => [...old, { msg, logLevel }]);
    },
    [setLines]
  );

  useEffect(() => {
    BenchmarkWasmModule.Create()
      .then((m) => {
        setBenchmarkModule(m);
      })
      .catch((e) => {
        LogFn("Failed to load Benchmark Wasm Module", LogLevel.Error);
      });
  }, [setBenchmarkModule, LogFn]);

  const RunBenchmark = useCallback(
    async (
      spanUrl: string,
      destUrl: string,
      pingCt: number,
      gapMs: number,
      payloadSize: number
    ) => {
      if (!benchmarkModule) {
        LogFn(
          "Benchmark WASM module not loaded, try again later (or refresh)",
          LogLevel.Warning
        );
        return;
      }

      const cleanupOps: Array<() => void> = [];
      try {
        LogFn(`Attempting WebTransport connection to endpoint ${spanUrl}...`);
        const wt = new WebTransport(spanUrl, {
          requireUnreliable: true,
        });
        await Promise.race([wt.ready, wt.closed]);
        cleanupOps.push(() => wt.close());

        LogFn(`Opening read/write datagram streams...`, LogLevel.Debug);
        const wtReader = wt.datagrams.readable.getReader();
        cleanupOps.push(() => wtReader.cancel());

        const wtWriter = wt.datagrams.writable.getWriter();
        cleanupOps.push(() => wtWriter.close());

        await Promise.race([wtWriter.ready, wtWriter.closed]);

        LogFn(
          `Starting benchmark (ct=${pingCt} gapms=${gapMs} msglen=${payloadSize})`
        );

        const benchmarkApp = benchmarkModule.CreateBenchmark();
        cleanupOps.push(() => benchmarkApp.cleanup());

        benchmarkApp.start_experiment(payloadSize, pingCt);

        const pendingMessages: Uint8Array[] = [];
        const readStreamDone = (async () => {
          while (true) {
            const maybeReadRsl = await Promise.race([
              wtReader.read(),
              wtReader.closed,
            ]);
            if (maybeReadRsl == null) return;
            const { value, done } = maybeReadRsl;
            if (done) return;
            if (!(value instanceof Uint8Array)) return;

            pendingMessages.push(value);
          }
        })();

        while (benchmarkApp.is_running()) {
          const maybe_incoming_msg = pendingMessages.shift();
          if (maybe_incoming_msg != null) {
            benchmarkApp.add_server_message(maybe_incoming_msg);
          }

          const next_msg_opt = benchmarkApp.get_next_message();
          if (next_msg_opt != null) {
            await wtWriter.write(next_msg_opt);
          }

          await new Promise((resolve) => setTimeout(resolve, gapMs));
        }
        LogFn(
          "Benchmark finished! Aggregating results and closing down WebTransport connection..."
        );

        wt.close();
        await Promise.all([readStreamDone, wt.closed]);
      } catch (e) {
        const errMsg = (() => {
          if (e instanceof Error) {
            return e.message;
          } else if (typeof e === "string") {
            return e;
          }
          return "" + e;
        })();
        LogFn(`Failed to run benchmark: ${errMsg}`, LogLevel.Error);
      } finally {
        cleanupOps.reverse().forEach((op) => op());
      }
    },
    [LogFn, benchmarkModule]
  );

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
        Spanreed Netcode Benchmark
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
