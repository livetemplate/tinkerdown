/**
 * Monaco Loader - Lazy load Monaco Editor on demand
 * This reduces initial bundle size by ~2.5MB
 */

import type * as monaco from "monaco-editor";

// Singleton pattern - only load once
let monacoInstance: typeof monaco | null = null;
let loadingPromise: Promise<typeof monaco> | null = null;

/**
 * Lazy load Monaco Editor
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

  // Start loading Monaco
  console.log("[MonacoLoader] Loading Monaco Editor...");
  const startTime = performance.now();

  loadingPromise = import("monaco-editor").then((module) => {
    monacoInstance = module;
    const loadTime = Math.round(performance.now() - startTime);
    console.log(`[MonacoLoader] Monaco Editor loaded in ${loadTime}ms`);

    // Initialize Monaco environment
    initializeMonacoEnvironment(module);

    return module;
  });

  return loadingPromise;
}

/**
 * Check if Monaco is already loaded
 */
export function isMonacoLoaded(): boolean {
  return monacoInstance !== null;
}

/**
 * Initialize Monaco environment (workers, etc.)
 */
function initializeMonacoEnvironment(_monaco: typeof import("monaco-editor")): void {
  if (typeof window !== "undefined") {
    (window as any).MonacoEnvironment = {
      getWorkerUrl: function (_moduleId: string, label: string) {
        // Note: These worker paths would need to be served by the Go server
        // For now, we'll use data URLs (monaco provides inline workers)
        if (label === "json") {
          return "/assets/monaco/json.worker.js";
        }
        if (label === "css" || label === "scss" || label === "less") {
          return "/assets/monaco/css.worker.js";
        }
        if (label === "html" || label === "handlebars" || label === "razor") {
          return "/assets/monaco/html.worker.js";
        }
        if (label === "typescript" || label === "javascript") {
          return "/assets/monaco/ts.worker.js";
        }
        return "/assets/monaco/editor.worker.js";
      },
    };
  }
}

/**
 * Preload Monaco in the background (optional optimization)
 * Call this early in page lifecycle if you know Monaco will be needed
 */
export function preloadMonaco(): void {
  if (!monacoInstance && !loadingPromise) {
    // Start loading but don't wait for it
    loadMonaco().catch((error) => {
      console.error("[MonacoLoader] Failed to preload Monaco:", error);
    });
  }
}

/**
 * Check if page has any WASM blocks that need Monaco
 */
export function hasEditableBlocks(): boolean {
  return document.querySelectorAll('[data-block-type="wasm"]').length > 0;
}
