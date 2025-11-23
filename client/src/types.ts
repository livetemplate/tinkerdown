/**
 * Core types for Livepage Client
 */

export type BlockType = "server" | "wasm" | "interactive" | "lvt";

export interface BlockMetadata {
  id: string;
  type: BlockType;
  language: string;
  readonly?: boolean;
  editable?: boolean;
  stateRef?: string; // Reference to server state (for interactive blocks)
}

export interface MessageEnvelope {
  blockID: string;
  action: string;
  data: any;
}

export interface LivepageClientOptions {
  wsUrl: string;
  debug?: boolean;
  persistence?: boolean;
  cdnFallback?: boolean;
  onConnect?: () => void;
  onDisconnect?: () => void;
  onError?: (error: Error) => void;
}

export interface BlockConfig {
  element: HTMLElement;
  metadata: BlockMetadata;
  initialCode?: string;
}

export interface WasmExecutionResult {
  stdout: string;
  stderr: string;
  error?: string;
  exitCode: number;
}

export interface EditorOptions {
  language: string;
  readonly: boolean;
  theme?: string;
  minimap?: boolean;
  lineNumbers?: boolean;
}

export interface PersistenceData {
  code: Record<string, string>; // blockID -> code
  timestamp: number;
}
