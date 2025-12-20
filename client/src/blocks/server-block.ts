/**
 * ServerBlock - Displays server-side code (read-only)
 */

import { BaseBlock } from "./base-block";
import { BlockConfig, ExecMeta } from "../types";
import { PersistenceManager } from "../core/persistence-manager";

export class ServerBlock extends BaseBlock {
  private codeElement: HTMLElement | null = null;

  constructor(config: BlockConfig, persistence: PersistenceManager, debug = false) {
    super(config, persistence, debug);
  }

  initialize(): void {
    this.log("Initializing server block");

    // Find or create code element
    this.codeElement = this.element.querySelector("code") || this.element;

    // Add CSS classes for styling
    this.element.classList.add("livepage-server-block");
    if (this.metadata.readonly) {
      this.element.classList.add("readonly");
    }

    // Add metadata as data attributes
    this.element.dataset.blockId = this.id;
    this.element.dataset.language = this.metadata.language;

    // Display initial code
    this.render();

    this.log("Server block initialized");
  }

  destroy(): void {
    this.log("Destroying server block");
    // No cleanup needed for server blocks
  }

  handleMessage(action: string, data: any, _execMeta?: ExecMeta): void {
    this.log("Received message:", action, data);
    // Server blocks don't handle messages (they're static)
    console.warn(`[ServerBlock:${this.id}] Received unexpected message:`, action);
  }

  private render(): void {
    if (!this.codeElement) return;

    // Simply display the code (syntax highlighting handled by existing pre/code elements)
    // The server already rendered this, we're just marking it as a livepage block
    this.log("Rendered server block");
  }
}
