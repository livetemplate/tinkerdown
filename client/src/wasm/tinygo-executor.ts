/**
 * TinyGoExecutor - Executes Go code via WASM
 *
 * TODO: Client-side TinyGo compilation requires further research.
 * Options:
 * 1. Server-side compilation endpoint (simpler, requires backend)
 * 2. TinyGo compiled to WASM running in browser (complex, large bundle)
 * 3. Hybrid: cache compiled WASM, fallback to server
 *
 * For MVP, we'll implement server-side compilation approach.
 */

import { WasmExecutionResult } from "../types";

export class TinyGoExecutor {
  private serverUrl: string;
  private debug: boolean;

  constructor(serverUrl = "/api/compile", debug = false) {
    this.serverUrl = serverUrl;
    this.debug = debug;
  }

  /**
   * Execute Go code (server-side compilation)
   */
  async execute(code: string): Promise<WasmExecutionResult> {
    if (this.debug) {
      console.log("[TinyGoExecutor] Executing code:", code);
    }

    try {
      // Send code to server for compilation
      const response = await fetch(this.serverUrl, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ code }),
      });

      if (!response.ok) {
        const error = await response.text();
        return {
          stdout: "",
          stderr: error,
          error: `Compilation failed: ${error}`,
          exitCode: 1,
        };
      }

      // Get compiled WASM binary
      const wasmBuffer = await response.arrayBuffer();

      // Execute WASM
      return await this.executeWasm(wasmBuffer);
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : String(error);
      return {
        stdout: "",
        stderr: "",
        error: `Execution failed: ${errorMsg}`,
        exitCode: 1,
      };
    }
  }

  /**
   * Execute compiled WASM binary
   */
  private async executeWasm(wasmBuffer: ArrayBuffer): Promise<WasmExecutionResult> {
    let stdout = "";
    let stderr = "";

    try {
      // Create Go WASM environment
      const go = new (window as any).Go();

      // Capture stdout
      const originalLog = console.log;
      const originalError = console.error;

      console.log = (...args: any[]) => {
        stdout += args.join(" ") + "\n";
      };

      console.error = (...args: any[]) => {
        stderr += args.join(" ") + "\n";
      };

      // Instantiate and run WASM
      const result = await WebAssembly.instantiate(wasmBuffer, go.importObject);
      await go.run(result.instance);

      // Restore console
      console.log = originalLog;
      console.error = originalError;

      return {
        stdout,
        stderr,
        exitCode: 0,
      };
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : String(error);
      return {
        stdout,
        stderr,
        error: `WASM execution failed: ${errorMsg}`,
        exitCode: 1,
      };
    }
  }

  /**
   * Check if TinyGo/WASM is supported
   */
  static isSupported(): boolean {
    return (
      typeof WebAssembly !== "undefined" &&
      typeof (window as any).Go !== "undefined"
    );
  }
}

/**
 * Initialize WASM environment (loads wasm_exec.js from TinyGo)
 */
export async function initializeWasm(): Promise<void> {
  if ((window as any).Go) {
    return; // Already initialized
  }

  try {
    // Load TinyGo's wasm_exec.js
    const script = document.createElement("script");
    script.src = "/assets/wasm_exec.js";

    await new Promise<void>((resolve, reject) => {
      script.onload = () => resolve();
      script.onerror = () => reject(new Error("Failed to load wasm_exec.js"));
      document.head.appendChild(script);
    });

    console.log("[TinyGoExecutor] WASM environment initialized");
  } catch (error) {
    console.error("[TinyGoExecutor] Failed to initialize WASM:", error);
    throw error;
  }
}
