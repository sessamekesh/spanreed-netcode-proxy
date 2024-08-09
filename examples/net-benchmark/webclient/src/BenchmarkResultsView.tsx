import React from "react";
import { BenchmarkResults } from "./BenchmarkWasmWrapper";

interface BenchmarkResultsViewProps {
  results: BenchmarkResults;
}

function usToMsString(us: number | null | undefined) {
  if (us == null) return "N/A";
  return (us / 1000).toFixed(1);
}

const BASIC_CELL_PROPS: React.CSSProperties = {
  padding: '4px 12px',
  fontWeight: '500',
};

const HEADER_CELL_PROPS: React.CSSProperties = {
  padding: '6px 12px',
  fontSize: '14pt',
  fontWeight: 700,
};

const ODD_TABLE_ROW_PROPS: React.CSSProperties = {
  backgroundColor: '#20202e',
};

const EVEN_TABLE_ROW_PROPS: React.CSSProperties = {
  backgroundColor: '#1a1e1d',
};

export const BenchmarkResultsView: React.FC<BenchmarkResultsViewProps> =
  React.memo(({ results }) => {
    return (
      <div style={{
        padding: '0 24px',
      }}>
        <h1 style={{
          borderBottom: '1px solid #ccc',
          maxWidth: '50%',
        }}>Benchmark Results</h1>
        <div style={{
          border: '1px solid #444',
          borderRadius: '12px',
          padding: '12px',
        }}>
          <h2 style={{ color: '#aaa', padding: 0, margin: 0, }}>Parameters:</h2>
          <span style={{ marginLeft: '8px', marginTop: '8px', display: 'block', }}>
            {`SpanURL=${results.params.spanEndpoint}, gapMs=${results.params.gapMs}, pingCt=${results.params.pingCt}, payloadSize=${results.params.pingPayloadSize}`}
          </span>
        </div>
        <div style={{
          marginTop: '16px',
          border: '1px solid #444',
          borderRadius: '12px',
          padding: '12px',
          display: 'flex',
          flexDirection: 'column',
        }}>
          <h2 style={{ color: '#cdcdff', padding: 0, margin: 0, }}>Reliability Metrics</h2>
          <table style={{
            margin: '8px',
            border: '2px solid #888',
            flexGrow: 1,
          }}>
            <tbody>
              <tr style={{ ...EVEN_TABLE_ROW_PROPS }}>
                <td style={{ ...BASIC_CELL_PROPS }}>Client Messages Sent</td>
                <td style={{ ...BASIC_CELL_PROPS }}>{results.results?.client_sent_messages ?? "N/A"}</td>
              </tr>
              <tr style={{ ...ODD_TABLE_ROW_PROPS }}>
                <td style={{ ...BASIC_CELL_PROPS }}>Client Messages Received</td>
                <td style={{ ...BASIC_CELL_PROPS }}>{results.results?.server_recv_messages ?? "N/A"}</td>
              </tr>
              <tr style={{ ...EVEN_TABLE_ROW_PROPS }}>
                <td style={{ ...BASIC_CELL_PROPS }}>Client Out-Of-Order Receive Ct</td>
                <td style={{ ...BASIC_CELL_PROPS }}>{results.results?.server_out_of_order_messages ?? "N/A"}</td>
              </tr>
              <tr style={{ ...ODD_TABLE_ROW_PROPS }}>
                <td style={{ ...BASIC_CELL_PROPS }}>Client Messages Dropped Ct</td>
                <td style={{ ...BASIC_CELL_PROPS }}>{results.results?.server_dropped_messages ?? "N/A"}</td>
              </tr>
              <tr style={{ ...EVEN_TABLE_ROW_PROPS }}>
                <td style={{ ...BASIC_CELL_PROPS }}>Server Messages Received</td>
                <td style={{ ...BASIC_CELL_PROPS }}>{results.results?.client_recv_messages ?? "N/A"}</td>
              </tr>
              <tr style={{ ...ODD_TABLE_ROW_PROPS }}>
                <td style={{ ...BASIC_CELL_PROPS }}>Server Out-Of-Order Receive Ct</td>
                <td style={{ ...BASIC_CELL_PROPS }}>{results.results?.client_out_of_order_messages ?? "N/A"}</td>
              </tr>
              <tr style={{ ...EVEN_TABLE_ROW_PROPS }}>
                <td style={{ ...BASIC_CELL_PROPS }}>Server Messages Dropped Ct</td>
                <td style={{ ...BASIC_CELL_PROPS }}>{results.results?.client_dropped_messages ?? "N/A"}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div style={{
          marginTop: '16px',
          border: '1px solid #444',
          borderRadius: '12px',
          padding: '12px',
          display: 'flex',
          flexDirection: 'column',
        }}>
          <h2 style={{ color: '#cdffff', padding: 0, margin: 0, }}>Latency Metrics</h2>
          <table style={{
            margin: '8px',
            border: '2px solid #888',
            flexGrow: 1,
          }}>
            <thead style={{
              backgroundColor: '#04041c',
            }}>
              <tr>
                <th style={{ ...HEADER_CELL_PROPS }}>Metric</th>
                <th style={{ ...HEADER_CELL_PROPS }}>Median</th>
                <th style={{ ...HEADER_CELL_PROPS }}>StdDev</th>
                <th style={{ ...HEADER_CELL_PROPS }}>P90</th>
                <th style={{ ...HEADER_CELL_PROPS }}>P95</th>
                <th style={{ ...HEADER_CELL_PROPS }}>P99</th>
                <th style={{ ...HEADER_CELL_PROPS }}>Min</th>
                <th style={{ ...HEADER_CELL_PROPS }}>Max</th>
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
                even
              />
              <BenchmarkResultsRow
                row={results.results?.rtt_measurements?.ProxyProcessTime}
                rowName="ProxyProcessTime"
              />
              <BenchmarkResultsRow
                row={results.results?.rtt_measurements?.ProxyRTT}
                rowName="ProxyRTT"
                even
              />
              <BenchmarkResultsRow
                row={results.results?.rtt_measurements?.ProxyDestNetTime}
                rowName="ProxyDestNetTime"
              />
              <BenchmarkResultsRow
                row={results.results?.rtt_measurements?.DestProcessTime}
                rowName="DestProcessTime"
                even
              />
            </tbody>
          </table>
        </div>
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
  even?: boolean;
}> = React.memo(({ row, rowName, even }) => {
  return (
    <tr style={{
      ...(even ? { ...EVEN_TABLE_ROW_PROPS } : { ...ODD_TABLE_ROW_PROPS }),
    }}>
      <td style={{ ...BASIC_CELL_PROPS }}>{rowName}</td>
      <td style={{ ...BASIC_CELL_PROPS }}>{usToMsString(row?.P50)}</td>
      <td style={{ ...BASIC_CELL_PROPS }}>{usToMsString(row?.StdDev)}</td>
      <td style={{ ...BASIC_CELL_PROPS }}>{usToMsString(row?.P90)}</td>
      <td style={{ ...BASIC_CELL_PROPS }}>{usToMsString(row?.P95)}</td>
      <td style={{ ...BASIC_CELL_PROPS }}>{usToMsString(row?.P99)}</td>
      <td style={{ ...BASIC_CELL_PROPS }}>{usToMsString(row?.Min)}</td>
      <td style={{ ...BASIC_CELL_PROPS }}>{usToMsString(row?.Max)}</td>
    </tr>
  );
});
