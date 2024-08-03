export class BenchmarkApp {
  constructor(private readonly m: any) { }
}

export class BenchmarkWasmModule {
  private constructor(private readonly m: any) { }

  private static async maybeLoadScript(): Promise<void> {
    if (window.BenchmarkWebClient != null) return;

    const script = document.createElement('script');
    script.src = '/web-client.js';
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
        reject('Failed to load WASM bootstrap JS');
        return;
      }
      window.BenchmarkWebClient().then(resolve, reject);
    });

    return new BenchmarkWasmModule(m);
  }
}

declare global {
  interface Window {
    BenchmarkWebClient?: () => {
      then: (m: any, reject: any) => Promise<void>,
    },
  }
}
