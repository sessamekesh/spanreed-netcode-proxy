export class BenchmarkApp {
  constructor(
    private readonly m: any,
    private readonly timer: any,
    private readonly app: any
  ) {}

  start_experiment(payloadSize: number, messageCount: number) {
    this.app["start_experiment"](payloadSize, messageCount);
  }

  is_running(): boolean {
    return this.app["is_running"]();
  }

  get_results() {
    return this.app["get_results"]();
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

declare global {
  interface Window {
    BenchmarkWebClient?: () => {
      then: (m: any, reject: any) => Promise<void>;
    };
  }
}
