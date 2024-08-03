import React from "react";
import { DefaultLogCb, LogCb, WrapLogFn } from "./log";
import { Button } from "./Button";

interface BenchmarkParamsProps {
  // TODO (sessamekesh): Use an observable here instead that gives progress reports (show RTT pongs, drops, OOOs)
  onRunBenchmark?: (spanAddr: string, destAddr: string, pingCt: number, gapMs: number, payloadSize: number) => Promise<void>;
  logCb?: LogCb;
}

const UrlInputStyle: React.CSSProperties | Record<string, React.CSSProperties> =
{
  minWidth: '250px',
  borderRadius: '4px',
  border: '2px solid #88cc88',
  backgroundColor: '#3E4544',
  color: '#eee',
  padding: '4px 8px',
};

export const BenchmarkParams: React.FC<BenchmarkParamsProps> = React.memo(
  ({ logCb, onRunBenchmark }) => {
    const [loading, setLoading] = React.useState(false);

    const [spanUrl, setSpanUrl] = React.useState("https://localhost:3000");
    const [destUrl, setDestUrl] = React.useState("localhost:30001");
    const [msgCtStr, setMsgCtStr] = React.useState('500');
    const [gapMsStr, setGapMsStr] = React.useState('15');
    const [payloadSizeStr, setPayloadSizeStr] = React.useState('0');

    const msgCtErr: string | null = (() => {
      const msgCt = parseInt(msgCtStr);
      if (isNaN(msgCt)) {
        return 'Must be a number';
      }

      if (msgCt <= 0) {
        return 'Must send at least one ping';
      }

      if (msgCt > 500000) {
        return 'Okay now be reasonable...';
      }

      return null;
    })();

    const gapMsErr: string | null = (() => {
      const gapMs = parseInt(gapMsStr);
      if (isNaN(gapMs)) {
        return 'Must be a number';
      }

      if (gapMs < 1) {
        return '0ms is not allowed';
      }

      if (gapMs > 300000) {
        return 'Do you really need more than 5m?';
      }

      return null;
    })();

    const payloadSizeErr: string | null = (() => {
      const payloadSize = parseInt(payloadSizeStr);
      if (isNaN(payloadSize)) {
        return 'Must be a number';
      }

      if (payloadSize >= 10240) {
        return 'Max 10 KB (10240 bytes)';
      }

      return null;
    })();

    const log = WrapLogFn("BenchmarkParams", logCb ?? DefaultLogCb);

    return (
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          padding: '20px 24px',
          margin: '4px 12px',
          borderRadius: '4px',
          border: '1px solid #eee',
        }}
      >
        {/* Header */}
        <div
          style={{
            display: 'flex',
            flexDirection: 'row',
            justifyContent: 'space-between',
            width: '35%',
            alignItems: 'center',
            borderBottom: '1px solid gray',
            paddingBottom: '1px',
          }}
        >
          <p
            style={{
              margin: 0,
              fontSize: '14pt',
              fontWeight: 700,
              color: '#ccc',
            }}
          >
            Run Benchmark
          </p>
        </div>

        {/* Server address inputs */}
        <form
          style={{
            marginBottom: '24px',
            marginTop: '24px',
            display: 'grid',
            gridTemplateColumns: '2fr 5fr',
            gap: '10px',
          }}
        >
          <label htmlFor="tb-spanreed-url">Spanreed Proxy Address:</label>
          <input
            type="url"
            id="tb-spanreed-url"
            pattern={'https://.*'}
            placeholder={
              'https://spanreed.example.com/wt'
            }
            style={{
              ...UrlInputStyle,
            }}
            value={spanUrl}
            onChange={(e) =>
              !loading && setSpanUrl(e.target.value)
            }
          ></input>

          <label htmlFor="tb-backend-url">Demo Server Address:</label>
          <input
            id="tb-backend-url"
            type="text"
            placeholder="URL or IP address"
            style={{
              ...UrlInputStyle,
            }}
            value={destUrl}
            onChange={(e) =>
              !loading && setDestUrl(e.target.value)
            }
          ></input>

          <label htmlFor="tb-msg-ct">Ping Count:</label>
          <div style={{ display: 'flex', flexDirection: 'row' }}>
            <input
              id="tb-msg-ct"
              type="text"
              placeholder="500"
              style={{
                ...UrlInputStyle,
                ...(msgCtErr !== null ? {
                  border: '2px solid #cc3300',
                  minWidth: 0,
                } : {
                  flexGrow: 1,
                })
              }}
              value={msgCtStr}
              onChange={(e) => !loading && setMsgCtStr(e.target.value.replace(/\D/g, ''))}
            />
            {msgCtErr !== null && <span style={{
              alignSelf: 'center', textAlign: 'right', marginLeft: '40px',
              color: '#cc3300', fontWeight: 700, flexGrow: 1,
            }}>{msgCtErr}</span>}
          </div>

          <label htmlFor="tb-gap-ms">Gap Between Pings (ms):</label>
          <div style={{ display: 'flex', flexDirection: 'row' }}>
            <input
              id="tb-gap-ms"
              type="text"
              placeholder="50"
              style={{
                ...UrlInputStyle,
                ...(gapMsErr !== null ? {
                  border: '2px solid #cc3300',
                  minWidth: 0,
                } : {
                  flexGrow: 1,
                })
              }}
              value={gapMsStr}
              onChange={(e) => !loading && setGapMsStr(e.target.value.replace(/\D/g, ''))}
            />
            {gapMsErr !== null && <span style={{
              alignSelf: 'center', textAlign: 'right', marginLeft: '40px',
              color: '#cc3300', fontWeight: 700, flexGrow: 1,
            }}>{gapMsErr}</span>}
          </div>

          <label htmlFor="tb-payload-size">Ping Payload Size (bytes):</label>
          <div style={{ display: 'flex', flexDirection: 'row' }}>
            <input
              id="tb-payload-size"
              type="text"
              placeholder="50"
              style={{
                ...UrlInputStyle,
                ...(payloadSizeErr !== null ? {
                  border: '2px solid #cc3300',
                  minWidth: 0,
                } : {
                  flexGrow: 1,
                })
              }}
              value={payloadSizeStr}
              onChange={(e) => !loading && setPayloadSizeStr(e.target.value.replace(/\D/g, ''))}
            />
            {payloadSizeErr !== null && <span style={{
              alignSelf: 'center', textAlign: 'right', marginLeft: '40px',
              color: '#cc3300', fontWeight: 700, flexGrow: 1,
            }}>{payloadSizeErr}</span>}
          </div>
        </form>

        {/* Submit! */}
        <Button
          disabled={loading || msgCtErr !== null || payloadSizeErr !== null || gapMsErr !== null || spanUrl === '' || destUrl === ''}
          onClick={() => {
            if (!onRunBenchmark) return;

            setLoading(true);
            onRunBenchmark(spanUrl, destUrl, parseInt(msgCtStr), parseInt(gapMsStr), parseInt(payloadSizeStr))
              .finally(() => setLoading(false));
          }}>
          <span>Benchmark!</span>
        </Button>
      </div>
    );
  }
);
