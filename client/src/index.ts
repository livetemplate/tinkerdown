/**
 * Livepage Client - Interactive documentation runtime
 * @module @livetemplate/livepage-client
 */

export { LivepageClient } from "./livepage-client";
export { MessageRouter } from "./core/message-router";
export { PersistenceManager } from "./core/persistence-manager";
export { BaseBlock } from "./blocks/base-block";
export { ServerBlock } from "./blocks/server-block";
export { InteractiveBlock } from "./blocks/interactive-block";
export { WasmBlock } from "./blocks/wasm-block";
export { MonacoEditor } from "./editor/monaco-editor";
export { loadMonaco, isMonacoLoaded, preloadMonaco, hasEditableBlocks } from "./editor/monaco-loader";
export { OutputPanel } from "./ui/output-panel";
export { RunButton } from "./ui/run-button";
export { TinyGoExecutor, initializeWasm } from "./wasm/tinygo-executor";

export type {
  BlockType,
  BlockMetadata,
  MessageEnvelope,
  LivepageClientOptions,
  BlockConfig,
  WasmExecutionResult,
  EditorOptions,
  PersistenceData,
} from "./types";
