/**
 * LivepageClient - Main orchestrator for Livepage interactive documentation
 */

import { LivepageClientOptions, BlockConfig, BlockMetadata, MessageEnvelope } from "./types";
import { MessageRouter } from "./core/message-router";
import { PersistenceManager } from "./core/persistence-manager";
import { BaseBlock } from "./blocks/base-block";
import { ServerBlock } from "./blocks/server-block";
import { InteractiveBlock } from "./blocks/interactive-block";
import { WasmBlock } from "./blocks/wasm-block";

export class LivepageClient {
  private options: LivepageClientOptions;
  private router: MessageRouter;
  private persistence: PersistenceManager;
  private blocks: Map<string, BaseBlock> = new Map();
  private ws: WebSocket | null = null;
  private reconnectTimer: number | null = null;
  private isConnected = false;

  constructor(options: LivepageClientOptions) {
    this.options = {
      debug: false,
      persistence: true,
      cdnFallback: false,
      ...options,
    };

    this.router = new MessageRouter(this.options.debug);
    this.persistence = new PersistenceManager(
      "livepage:persistence",
      this.options.persistence,
      this.options.debug
    );

    if (this.options.debug) {
      console.log("[LivepageClient] Initialized with options:", this.options);
    }
  }

  /**
   * Connect to the WebSocket server
   */
  connect(): void {
    if (this.ws) {
      console.warn("[LivepageClient] Already connected");
      return;
    }

    try {
      this.ws = new WebSocket(this.options.wsUrl);

      this.ws.onopen = () => {
        this.isConnected = true;
        console.log("[LivepageClient] Connected to server");
        this.options.onConnect?.();
      };

      this.ws.onclose = () => {
        this.isConnected = false;
        console.log("[LivepageClient] Disconnected from server");
        this.options.onDisconnect?.();
        this.scheduleReconnect();
      };

      this.ws.onerror = (error) => {
        console.error("[LivepageClient] WebSocket error:", error);
        this.options.onError?.(new Error("WebSocket error"));
      };

      this.ws.onmessage = (event) => {
        this.handleMessage(event.data);
      };
    } catch (error) {
      console.error("[LivepageClient] Failed to connect:", error);
      this.options.onError?.(error as Error);
    }
  }

  /**
   * Disconnect from the server
   */
  disconnect(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }

    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }

    this.isConnected = false;
  }

  /**
   * Handle incoming WebSocket messages
   */
  private handleMessage(data: string): void {
    if (this.options.debug) {
      console.log("[LivepageClient] Received message:", data);
    }

    // Route message to appropriate block
    this.router.route(data);
  }

  /**
   * Send a message to the server
   */
  send(blockID: string, action: string, data: any = {}): void {
    if (!this.isConnected || !this.ws) {
      console.warn("[LivepageClient] Cannot send - not connected");
      return;
    }

    const envelope: MessageEnvelope = {
      blockID,
      action,
      data,
    };

    const message = JSON.stringify(envelope);

    if (this.options.debug) {
      console.log("[LivepageClient] Sending message:", message);
    }

    this.ws.send(message);
  }

  /**
   * Schedule reconnection attempt
   */
  private scheduleReconnect(): void {
    if (this.reconnectTimer) {
      return;
    }

    this.reconnectTimer = window.setTimeout(() => {
      this.reconnectTimer = null;
      console.log("[LivepageClient] Attempting to reconnect...");
      this.connect();
    }, 3000);
  }

  /**
   * Discover and register all code blocks on the page
   */
  discoverBlocks(): void {
    console.log("[LivepageClient] Discovering blocks...");

    // Find all code blocks with livepage metadata
    const codeBlocks = document.querySelectorAll<HTMLElement>(
      "[data-livepage-block]"
    );

    for (const element of Array.from(codeBlocks)) {
      try {
        const metadata = this.extractMetadata(element);
        const initialCode = this.extractCode(element);

        const config: BlockConfig = {
          element,
          metadata,
          initialCode,
        };

        this.registerBlock(config);
      } catch (error) {
        console.error("[LivepageClient] Error discovering block:", error);
      }
    }

    console.log(`[LivepageClient] Discovered ${this.blocks.size} blocks`);
  }

  /**
   * Extract metadata from a code block element
   */
  private extractMetadata(element: HTMLElement): BlockMetadata {
    const id = element.dataset.blockId || this.generateBlockId();
    const type = (element.dataset.blockType as any) || "server";
    const language = element.dataset.language || "go";
    const readonly = element.dataset.readonly === "true";
    const editable = element.dataset.editable === "true";
    const stateRef = element.dataset.stateRef;

    return {
      id,
      type,
      language,
      readonly,
      editable,
      stateRef,
    };
  }

  /**
   * Extract code from a code block element
   */
  private extractCode(element: HTMLElement): string {
    // Try to find code element
    const codeElement = element.querySelector("code");
    if (codeElement) {
      return codeElement.textContent || "";
    }

    // Fallback to element text content
    return element.textContent || "";
  }

  /**
   * Generate a unique block ID
   */
  private generateBlockId(): string {
    return `block-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
  }

  /**
   * Register a code block
   */
  registerBlock(config: BlockConfig): void {
    const { metadata } = config;

    if (this.blocks.has(metadata.id)) {
      console.warn(`[LivepageClient] Block already registered: ${metadata.id}`);
      return;
    }

    // Create appropriate block type
    let block: BaseBlock;

    switch (metadata.type) {
      case "server":
        block = new ServerBlock(config, this.persistence, this.options.debug);
        break;

      case "interactive":
      case "lvt":
        block = new InteractiveBlock(config, this.persistence, this.options.debug);
        // Set message sender for interactive blocks
        (block as InteractiveBlock).setMessageSender((blockID, action, data) => {
          this.send(blockID, action, data);
        });
        break;

      case "wasm":
        block = new WasmBlock(config, this.persistence, this.options.debug);
        break;

      default:
        console.warn(`[LivepageClient] Unknown block type: ${metadata.type}`);
        return;
    }

    // Initialize the block
    block.initialize();

    // Register with router (for message handling)
    this.router.register(metadata.id, (action, data) => {
      block.handleMessage(action, data);
    });

    // Store block reference
    this.blocks.set(metadata.id, block);

    if (this.options.debug) {
      console.log(`[LivepageClient] Registered block: ${metadata.id} (${metadata.type})`);
    }
  }

  /**
   * Unregister a code block
   */
  unregisterBlock(blockID: string): void {
    const block = this.blocks.get(blockID);
    if (!block) {
      return;
    }

    // Destroy the block
    block.destroy();

    // Unregister from router
    this.router.unregister(blockID);

    // Remove from blocks map
    this.blocks.delete(blockID);

    if (this.options.debug) {
      console.log(`[LivepageClient] Unregistered block: ${blockID}`);
    }
  }

  /**
   * Get a block by ID
   */
  getBlock(blockID: string): BaseBlock | undefined {
    return this.blocks.get(blockID);
  }

  /**
   * Get all block IDs
   */
  getBlockIds(): string[] {
    return Array.from(this.blocks.keys());
  }

  /**
   * Destroy the client and all blocks
   */
  destroy(): void {
    console.log("[LivepageClient] Destroying client");

    // Destroy all blocks
    for (const block of this.blocks.values()) {
      block.destroy();
    }

    this.blocks.clear();
    this.router.clear();
    this.disconnect();
  }
}
