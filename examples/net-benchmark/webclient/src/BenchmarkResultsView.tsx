import React from "react";
import { BenchmarkResults } from "./BenchmarkWasmWrapper";

interface BenchmarkResultsViewProps {
  results: BenchmarkResults;
}

function usToMsString(us: number | null | undefined) {
  if (us == null) return "N/A";
  return (us / 1000).toFixed(1);
}

export const BenchmarkResultsView: React.FC<BenchmarkResultsViewProps> =
  React.memo(({ results }) => {
    return (
      <div>
        <h1>Benchmark Results</h1>
        <h2>Parameters:</h2>
        <span>
          {`SpanURL=${results.params.spanEndpoint}, gapMs=${results.params.gapMs}, pingCt=${results.params.pingCt}, payloadSize=${results.params.pingPayloadSize}`}
        </span>
        <h2>Reliability Metrics</h2>
        <table>
          <tbody>
            <tr>
              <td>Client Messages Sent</td>
              <td>{results.results?.client_sent_messages ?? "N/A"}</td>
            </tr>
            <tr>
              <td>Client Messages Received</td>
              <td>{results.results?.server_recv_messages ?? "N/A"}</td>
            </tr>
            <tr>
              <td>Client Out-Of-Order Receive Ct</td>
              <td>{results.results?.server_out_of_order_messages ?? "N/A"}</td>
            </tr>
            <tr>
              <td>Client Messages Dropped Ct</td>
              <td>{results.results?.server_dropped_messages ?? "N/A"}</td>
            </tr>
            <tr>
              <td>Server Messages Received</td>
              <td>{results.results?.client_recv_messages ?? "N/A"}</td>
            </tr>
            <tr>
              <td>Server Out-Of-Order Receive Ct</td>
              <td>{results.results?.client_out_of_order_messages ?? "N/A"}</td>
            </tr>
            <tr>
              <td>Server Messages Dropped Ct</td>
              <td>{results.results?.client_dropped_messages ?? "N/A"}</td>
            </tr>
          </tbody>
        </table>
        <h2>Latency Metrics</h2>
        <table>
          <thead>
            <tr>
              <th>Metric</th>
              <th>Median</th>
              <th>StdDev</th>
              <th>P90</th>
              <th>P95</th>
              <th>P99</th>
              <th>Min</th>
              <th>Max</th>
            </tr>
          </thead>
          <tbody>
            <BenchmarkResultsRow
              row={results.results?.rtt_measurements?.ClientRtt}
              rowName="ClientRTT"
            />
            <BenchmarkResultsRow
              row={results.results?.rtt_measurements?.ClientProxyNetTime}
              rowName="ClientProxyNetTime"
            />
            <BenchmarkResultsRow
              row={results.results?.rtt_measurements?.ProxyProcessTime}
              rowName="ProxyProcessTime"
            />
            <BenchmarkResultsRow
              row={results.results?.rtt_measurements?.ProxyRTT}
              rowName="ProxyRTT"
            />
            <BenchmarkResultsRow
              row={results.results?.rtt_measurements?.ProxyDestNetTime}
              rowName="ProxyDestNetTime"
            />
            <BenchmarkResultsRow
              row={results.results?.rtt_measurements?.DestProcessTime}
              rowName="DestProcessTime"
            />
          </tbody>
        </table>
      </div>
    );
  });

const BenchmarkResultsRow: React.FC<{
  row:
    | NonNullable<
        NonNullable<BenchmarkResults["results"]>["rtt_measurements"]
      >["ClientProxyNetTime"]
    | null
    | undefined;
  rowName: string;
}> = React.memo(({ row, rowName }) => {
  return (
    <tr>
      <td>{rowName}</td>
      <td>{usToMsString(row?.P50)}</td>
      <td>{usToMsString(row?.StdDev)}</td>
      <td>{usToMsString(row?.P90)}</td>
      <td>{usToMsString(row?.P95)}</td>
      <td>{usToMsString(row?.P99)}</td>
      <td>{usToMsString(row?.Min)}</td>
      <td>{usToMsString(row?.Max)}</td>
    </tr>
  );
});
