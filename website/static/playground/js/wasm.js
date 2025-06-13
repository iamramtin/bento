class WasmManager {
  constructor() {
    this.isReady = false;
    this.failed = false;

    this.go = new Go();
  }

  async load() {
    const wasmPaths = [
      "playground.wasm",
      "./playground.wasm",
      "/wasm/playground.wasm",
      "bento/website/static/playground/playground.wasm",
    ];

    for (const path of wasmPaths) {
      try {
        const result = await WebAssembly.instantiateStreaming(
          fetch(path),
          this.go.importObject
        );

        this.go.run(result.instance);
        this.isReady = true;

        console.log("WASM loaded successfully");
        return;
      } catch (error) {
        console.warn("Failed to load WASM from", path, ":", error.message);
      }
    }

    this.failed = true;
    this.isReady = false;
  }

  execute(input, mapping) {
    if (this.failed) {
      throw new Error(
        "WASM not loaded. Bloblang functionality is unavailable."
      );
    }

    if (!this.isReady) {
      throw new Error("WASM not ready. Please wait for initialization.");
    }

    if (window.executeBloblang) {
      return window.executeBloblang(input, mapping);
    } else {
      throw new Error("Bloblang functionality not available in WASM context.");
    }
  }
}
