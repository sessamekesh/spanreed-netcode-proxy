import React from "react";
import { DefaultLogCb, LogCb, WrapLogFn } from "./log";

interface BenchmarkParamsProps {
  onRunBenchmark?: (spanAddr: string, destAddr: string) => Promise<void>;
  logCb?: LogCb;
}

const BenchmarkParams: React.FC<BenchmarkParamsProps> = React.memo(
  ({ logCb, onRunBenchmark }) => {
    const [loading, setLoading] = React.useState(false);

    const [spanUrl, setSpanUrl] = React.useState("https://localhost:3000");
    const [destUrl, setDestUrl] = React.useState("localhost:30001");

    const log = WrapLogFn("BenchmarkParams", logCb ?? DefaultLogCb);

    return (
      <div>
        SpanURL: {spanUrl}
        DestUrl: {destUrl}
        loading: {loading ? "yes" : "no"}
      </div>
    );
  }
);
