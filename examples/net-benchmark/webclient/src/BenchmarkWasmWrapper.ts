function asNumber(v: unknown) {
  if (typeof v === "bigint") return Number(v);
  if (typeof v !== "number") return null;
  if (isNaN(v)) return null;

  return v;
}

function asAggregate(v: unknown) {
  if (typeof v !== "object") return null;
  if (v == null) return null;

  return {
    Min: "Min" in v ? asNumber(v["Min"]) : null,
    Max: "Max" in v ? asNumber(v["Max"]) : null,
    StdDev: "StdDev" in v ? asNumber(v["StdDev"]) : null,
    P50: "P50" in v ? asNumber(v["P50"]) : null,
    P90: "P90" in v ? asNumber(v["P90"]) : null,
    P95: "P95" in v ? asNumber(v["P95"]) : null,
    P99: "P99" in v ? asNumber(v["P99"]) : null,
  };
}

export class BenchmarkApp {
  constructor(
    private readonly m: any,
    private readonly timer: any,
    private readonly app: any
  ) {}

  start_experiment(destUrl: string, payloadSize: number, messageCount: number) {
    this.app["start_experiment"](destUrl, payloadSize, messageCount);
  }

  is_running(): boolean {
    return this.app["is_running"]();
  }

  get_results() {
    const raw_results = this.app["get_results"]();
    if (raw_results == null) {
      return null;
    }

    const rtt_aggregates = this.m["RttAggregatesFrom"](
      raw_results["rtt_measurements"]
    );
    raw_results["rtt_measurements"]["delete"]();

    const rtt_measurements = (() => {
      if (rtt_aggregates == null) return null;

      return {
        ClientRtt: asAggregate(rtt_aggregates["ClientRTT"]),
        ProxyRTT: asAggregate(rtt_aggregates["ProxyRTT"]),
        DestProcessTime: asAggregate(rtt_aggregates["DestProcessTime"]),
        ProxyProcessTime: asAggregate(rtt_aggregates["ProxyProcessTime"]),
        ClientProxyNetTime: asAggregate(rtt_aggregates["ClientProxyNetTime"]),
        ProxyDestNetTime: asAggregate(rtt_aggregates["ProxyDestNetTime"]),
      };
    })();

    return {
      client_dropped_messages: asNumber(raw_results["client_dropped_messages"]),
      client_out_of_order_messages: asNumber(
        raw_results["client_out_of_order_messages"]
      ),
      client_recv_messages: asNumber(raw_results["client_recv_messages"]),
      client_sent_messages: asNumber(raw_results["client_sent_messages"]),
      server_dropped_messages: asNumber(raw_results["server_dropped_messages"]),
      server_out_of_order_messages: asNumber(
        raw_results["server_out_of_order_messages"]
      ),
      server_recv_messages: asNumber(raw_results["server_recv_messages"]),
      rtt_measurements,
    };
  }

  add_server_message(msg: Uint8Array): boolean {
    return this.m["on_recv_message"](msg, this.timer, this.app);
  }

  get_next_message(): Uint8Array | null {
    let ab: ArrayBuffer | null = null;
    this.m["maybe_send_msg"](this.app, this.timer, (buff: any) => {
      ab = buff;
    });
    return ab;
  }

  cleanup() {
    this.timer["delete"]();
    this.app["delete"]();
  }
}

export class BenchmarkWasmModule {
  private constructor(private readonly m: any) {}

  private static async maybeLoadScript(): Promise<void> {
    if (window.BenchmarkWebClient != null) return;

    const script = document.createElement("script");
    script.src = "/web-client.js";
    document.body.appendChild(script);
    await new Promise((resolve, reject) => {
      script.onerror = reject;
      script.onload = resolve;
    });
  }

  static async Create(): Promise<BenchmarkWasmModule> {
    await BenchmarkWasmModule.maybeLoadScript();

    // Wrapping bootstrap in another promise because I think Emscripten might still use a
    //  weird PromiseLike implementation that isn't actually ES6 promises?
    const m = await new Promise((resolve, reject) => {
      if (window.BenchmarkWebClient == null) {
        reject("Failed to load WASM bootstrap JS");
        return;
      }
      window.BenchmarkWebClient().then(resolve, reject);
    });

    return new BenchmarkWasmModule(m);
  }

  CreateBenchmark(): BenchmarkApp {
    const Timer = this.m["Timer"];
    const SpanreedBenchmarkApp = this.m["SpanreedBenchmarkApp"];

    const t = new Timer();
    const app = new SpanreedBenchmarkApp();

    return new BenchmarkApp(this.m, t, app);
  }
}

export type BenchmarkResults = {
  params: {
    pingCt: number;
    gapMs: number;
    pingPayloadSize: number;
    spanEndpoint: string;
  };
  results: ReturnType<BenchmarkApp["get_results"]>;
};

declare global {
  interface Window {
    BenchmarkWebClient?: () => {
      then: (m: any, reject: any) => Promise<void>;
    };
  }
}
