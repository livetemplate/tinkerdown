/**
 * Auto-initialization for browser builds
 */

import { LivepageClient } from "./livepage-client";
import { TutorialNavigation } from "./core/navigation";
import { SiteSearch } from "./core/search";
import { CodeCopy } from "./core/code-copy";
import { PageTOC } from "./core/page-toc";
import { hasEditableBlocks, preloadMonaco } from "./editor/monaco-loader";

/**
 * Auto-initialization function
 */
function initializeLivepage(): void {
  // Check if auto-init is disabled
  if ((window as any).LIVEPAGE_DISABLE_AUTO_INIT) {
    console.log("[Livepage] Auto-initialization disabled");
    return;
  }

  // Get WebSocket URL from meta tag or default
  const wsMeta = document.querySelector<HTMLMetaElement>('meta[name="livepage-ws-url"]');
  const wsUrl = wsMeta?.content || `ws://${window.location.host}/ws`;

  // Get debug flag from meta tag
  const debugMeta = document.querySelector<HTMLMetaElement>('meta[name="livepage-debug"]');
  const debug = debugMeta?.content === "true";

  // Preload Monaco if page has WASM blocks (lazy load in background)
  if (hasEditableBlocks()) {
    console.log("[Livepage] Preloading Monaco Editor for WASM blocks...");
    preloadMonaco();
  }

  // Create and initialize client
  const client = new LivepageClient({
    wsUrl,
    debug,
    persistence: true,
    onConnect: () => console.log("[Livepage] Connected"),
    onDisconnect: () => console.log("[Livepage] Disconnected"),
    onError: (error: Error) => console.error("[Livepage] Error:", error),
  });

  // Discover blocks
  client.discoverBlocks();

  // Connect to server (for interactive blocks)
  const hasInteractiveBlocks = client.getBlockIds().some((id: string) => {
    const block = client.getBlock(id);
    return block?.type === "interactive" || block?.type === "lvt";
  });

  if (hasInteractiveBlocks) {
    client.connect();
  }

  // Expose client globally for debugging
  (window as any).livepageClient = client;

  // Initialize tutorial navigation (if H2 headings exist)
  const nav = new TutorialNavigation();
  (window as any).livepageNavigation = nav;

  // Initialize page TOC (for site-mode pages with H2 sections)
  const pageTOC = new PageTOC();
  (window as any).livepagePageTOC = pageTOC;

  // Initialize site search (if in site mode)
  const search = new SiteSearch();
  (window as any).livepageSearch = search;

  // Initialize code copy buttons
  const codeCopy = new CodeCopy();
  (window as any).livepageCodeCopy = codeCopy;

  console.log(`[Livepage] Initialized with ${client.getBlockIds().length} blocks`);
}

/**
 * Initialize Livepage client automatically when script loads
 */
if (typeof window !== "undefined") {
  // Auto-initialize on DOMContentLoaded
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", initializeLivepage);
  } else {
    initializeLivepage();
  }
}
