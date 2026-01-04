/**
 * Tinkerdown Client - Interactive documentation runtime
 * @module @livetemplate/tinkerdown-client
 */

export { TinkerdownClient } from "./tinkerdown-client";
export { MessageRouter } from "./core/message-router";
export { PersistenceManager } from "./core/persistence-manager";
export { TabsController } from "./core/tabs";
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
  TinkerdownClientOptions,
  BlockConfig,
  WasmExecutionResult,
  EditorOptions,
  PersistenceData,
} from "./types";
