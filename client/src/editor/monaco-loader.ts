/**
 * Monaco Loader - Load Monaco Editor from CDN on demand
 * This eliminates Monaco from the bundle (~3MB reduction)
 */

import type * as monaco from "monaco-editor";

// Monaco CDN configuration
const MONACO_VERSION = "0.45.0";
const MONACO_CDN_BASE = `https://cdn.jsdelivr.net/npm/monaco-editor@${MONACO_VERSION}/min`;

// Singleton pattern - only load once
let monacoInstance: typeof monaco | null = null;
let loadingPromise: Promise<typeof monaco> | null = null;

// Declare AMD loader types (avoid conflict with monaco's Window extension)
interface AMDRequire {
  (deps: string[], callback: (...modules: any[]) => void): void;
  config: (config: any) => void;
}

declare global {
  interface Window {
    require: AMDRequire | undefined;
  }
}

/**
 * Load the Monaco AMD loader script from CDN
 */
function loadMonacoLoader(): Promise<void> {
  return new Promise((resolve, reject) => {
    // Check if loader already exists
    if (typeof window.require === "function") {
      resolve();
      return;
    }

    const script = document.createElement("script");
    script.src = `${MONACO_CDN_BASE}/vs/loader.js`;
    script.async = true;
    script.onload = () => resolve();
    script.onerror = () => reject(new Error("Failed to load Monaco loader from CDN"));
    document.head.appendChild(script);
  });
}

/**
 * Initialize Monaco environment for workers
 */
function initializeMonacoEnvironment(): void {
  window.MonacoEnvironment = {
    getWorkerUrl: function (_moduleId: string, label: string) {
      // Use CDN for workers
      const workerMain = `${MONACO_CDN_BASE}/vs/base/worker/workerMain.js`;

      // Return appropriate worker based on language
      if (label === "json") {
        return `${MONACO_CDN_BASE}/vs/language/json/json.worker.js`;
      }
      if (label === "css" || label === "scss" || label === "less") {
        return `${MONACO_CDN_BASE}/vs/language/css/css.worker.js`;
      }
      if (label === "html" || label === "handlebars" || label === "razor") {
        return `${MONACO_CDN_BASE}/vs/language/html/html.worker.js`;
      }
      if (label === "typescript" || label === "javascript") {
        return `${MONACO_CDN_BASE}/vs/language/typescript/ts.worker.js`;
      }

      // Default editor worker
      return workerMain;
    },
  };
}

/**
 * Lazy load Monaco Editor from CDN
 * @returns Promise that resolves to monaco-editor module
 */
export async function loadMonaco(): Promise<typeof monaco> {
  // Return cached instance if already loaded
  if (monacoInstance) {
    return monacoInstance;
  }

  // Return existing promise if currently loading
  if (loadingPromise) {
    return loadingPromise;
  }

  console.log("[MonacoLoader] Loading Monaco Editor from CDN...");
  const startTime = performance.now();

  loadingPromise = (async () => {
    try {
      // Step 1: Load the AMD loader
      await loadMonacoLoader();

      // Get the AMD require function
      const amdRequire = window.require;
      if (!amdRequire) {
        throw new Error("AMD loader not available after loading");
      }

      // Step 2: Configure AMD loader paths
      amdRequire.config({
        paths: {
          vs: `${MONACO_CDN_BASE}/vs`,
        },
      });

      // Step 3: Initialize environment before loading
      initializeMonacoEnvironment();

      // Step 4: Load Monaco editor module
      return await new Promise<typeof monaco>((resolve, reject) => {
        amdRequire(["vs/editor/editor.main"], (monacoModule: typeof monaco) => {
          if (!monacoModule) {
            reject(new Error("Monaco module loaded but is undefined"));
            return;
          }

          monacoInstance = monacoModule;
          const loadTime = Math.round(performance.now() - startTime);
          console.log(`[MonacoLoader] Monaco Editor loaded from CDN in ${loadTime}ms`);
          resolve(monacoModule);
        });
      });
    } catch (error) {
      loadingPromise = null; // Reset on error so we can retry
      throw error;
    }
  })();

  return loadingPromise;
}

/**
 * Check if Monaco is already loaded
 */
export function isMonacoLoaded(): boolean {
  return monacoInstance !== null;
}

/**
 * Preload Monaco in the background (optional optimization)
 * Call this early in page lifecycle if you know Monaco will be needed
 */
export function preloadMonaco(): void {
  if (!monacoInstance && !loadingPromise) {
    // Start loading but don't wait for it
    loadMonaco().catch((error) => {
      console.error("[MonacoLoader] Failed to preload Monaco from CDN:", error);
    });
  }
}

/**
 * Check if page has any WASM blocks that need Monaco
 */
export function hasEditableBlocks(): boolean {
  return document.querySelectorAll('[data-block-type="wasm"]').length > 0;
}
