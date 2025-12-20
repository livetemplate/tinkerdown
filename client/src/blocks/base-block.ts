/**
 * BaseBlock - Abstract base class for all code blocks
 */

import { BlockConfig, BlockMetadata, ExecMeta } from "../types";
import { PersistenceManager } from "../core/persistence-manager";

export abstract class BaseBlock {
  protected element: HTMLElement;
  protected metadata: BlockMetadata;
  protected persistence: PersistenceManager;
  protected initialCode: string;
  protected currentCode: string;
  protected debug: boolean;

  constructor(config: BlockConfig, persistence: PersistenceManager, debug = false) {
    this.element = config.element;
    this.metadata = config.metadata;
    this.persistence = persistence;
    this.initialCode = config.initialCode || "";
    this.currentCode = this.initialCode;
    this.debug = debug;
  }

  /**
   * Initialize the block (called by LivepageClient)
   */
  abstract initialize(): void;

  /**
   * Destroy the block and clean up resources
   */
  abstract destroy(): void;

  /**
   * Handle incoming messages from the server
   */
  abstract handleMessage(action: string, data: any, execMeta?: ExecMeta): void;

  /**
   * Get the block ID
   */
  get id(): string {
    return this.metadata.id;
  }

  /**
   * Get the block type
   */
  get type(): string {
    return this.metadata.type;
  }

  /**
   * Get current code content
   */
  getCode(): string {
    return this.currentCode;
  }

  /**
   * Set code content
   */
  setCode(code: string): void {
    this.currentCode = code;
    if (this.metadata.editable) {
      this.persistence.saveCode(this.id, code);
    }
  }

  /**
   * Reset to initial code
   */
  reset(): void {
    this.setCode(this.initialCode);
    if (this.debug) {
      console.log(`[Block:${this.id}] Reset to initial code`);
    }
  }

  /**
   * Load persisted code if available
   */
  protected loadPersistedCode(): string {
    if (!this.metadata.editable) {
      return this.initialCode;
    }

    const persisted = this.persistence.loadCode(this.id);
    if (persisted) {
      if (this.debug) {
        console.log(`[Block:${this.id}] Loaded persisted code`);
      }
      return persisted;
    }

    return this.initialCode;
  }

  /**
   * Create wrapper HTML for the block
   */
  protected createBlockWrapper(): HTMLElement {
    const wrapper = document.createElement("div");
    wrapper.className = `livepage-block livepage-block-${this.type}`;
    wrapper.dataset.blockId = this.id;
    wrapper.dataset.blockType = this.type;
    return wrapper;
  }

  /**
   * Log debug message
   */
  protected log(...args: any[]): void {
    if (this.debug) {
      console.log(`[Block:${this.id}]`, ...args);
    }
  }

  /**
   * Log error message
   */
  protected error(...args: any[]): void {
    console.error(`[Block:${this.id}]`, ...args);
  }
}
