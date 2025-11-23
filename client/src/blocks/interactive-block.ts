/**
 * InteractiveBlock - Wraps a LiveTemplateClient instance for interactive UI
 */

import { LiveTemplateClient } from "@livetemplate/client";
import { BaseBlock } from "./base-block";
import { BlockConfig } from "../types";
import { PersistenceManager } from "../core/persistence-manager";

export class InteractiveBlock extends BaseBlock {
  private client: LiveTemplateClient | null = null;
  private containerElement: HTMLElement | null = null;
  private sendMessage: ((blockID: string, action: string, data: any) => void) | null = null;

  constructor(config: BlockConfig, persistence: PersistenceManager, debug = false) {
    super(config, persistence, debug);
  }

  initialize(): void {
    this.log("Initializing interactive block");

    // Find or create container for the interactive content
    this.containerElement = this.element.querySelector("[data-interactive-content]") || this.element;

    // Add CSS classes
    this.element.classList.add("livepage-interactive-block");
    this.element.dataset.blockId = this.id;
    if (this.metadata.stateRef) {
      this.element.dataset.stateRef = this.metadata.stateRef;
    }

    // Initialize LiveTemplateClient instance (without WebSocket - we handle that)
    // TODO: Fix @livetemplate/client export issue
    // For now, we handle DOM updates manually via updateDOM method
    // this.client = new LiveTemplateClient({
    //   autoReconnect: false,
    //   onConnect: () => this.log("Client connected"),
    //   onDisconnect: () => this.log("Client disconnected"),
    //   onError: (error) => this.error("Client error:", error),
    // });

    // Attach event handler to intercept lvt-* events
    this.attachEventHandlers();

    this.log("Interactive block initialized");
  }

  destroy(): void {
    this.log("Destroying interactive block");
    if (this.client) {
      // Clean up the client instance
      this.client = null;
    }
  }

  handleMessage(action: string, data: any): void {
    this.log("Received message:", action, data);

    switch (action) {
      case "tree":
        // Server sent a tree update - apply it to the DOM
        if (data.tree && this.containerElement) {
          this.updateDOM(data.tree);
        }
        break;

      case "update":
        // Server sent HTML update - replace container content
        if (data.html && this.containerElement) {
          this.containerElement.innerHTML = data.html;
          this.log("DOM updated with HTML");
        }
        break;

      case "error":
        this.error("Server error:", data.message);
        break;

      default:
        this.log("Unknown action:", action);
    }
  }

  /**
   * Set the message sender (called by LivepageClient)
   */
  setMessageSender(sender: (blockID: string, action: string, data: any) => void): void {
    this.sendMessage = sender;
  }

  /**
   * Update DOM with tree from server
   */
  private updateDOM(tree: any): void {
    if (!this.client || !this.containerElement) return;

    try {
      // Use the livetemplate client's tree renderer to update DOM
      // We need to access the internal renderer - for now, update innerHTML
      // TODO: Access TreeRenderer directly for optimized updates
      if (tree.html) {
        this.containerElement.innerHTML = tree.html;
      }
      this.log("DOM updated");
    } catch (error) {
      this.error("Error updating DOM:", error);
    }
  }

  /**
   * Attach event handlers for lvt-* attributes
   */
  private attachEventHandlers(): void {
    if (!this.element) return;

    // Delegate events to our custom handler
    this.element.addEventListener("click", (e) => this.handleClick(e), true);
    this.element.addEventListener("submit", (e) => this.handleSubmit(e), true);
    this.element.addEventListener("change", (e) => this.handleChange(e), true);
  }

  /**
   * Handle click events (lvt-click)
   */
  private handleClick(e: Event): void {
    const target = e.target as HTMLElement;
    const action = target.getAttribute("lvt-click");

    if (action) {
      e.preventDefault();
      this.sendAction(action, {});
      this.log("Click action:", action);
    }
  }

  /**
   * Handle submit events (lvt-submit)
   */
  private handleSubmit(e: Event): void {
    const target = e.target as HTMLFormElement;
    const action = target.getAttribute("lvt-submit");

    if (action) {
      e.preventDefault();
      const formData = new FormData(target);
      const data: Record<string, any> = {};
      formData.forEach((value, key) => {
        data[key] = value;
      });
      this.sendAction(action, data);
      this.log("Submit action:", action, data);
    }
  }

  /**
   * Handle change events (lvt-change)
   */
  private handleChange(e: Event): void {
    const target = e.target as HTMLInputElement;
    const action = target.getAttribute("lvt-change");

    if (action) {
      const data = { value: target.value };
      this.sendAction(action, data);
      this.log("Change action:", action, data);
    }
  }

  /**
   * Send an action to the server
   */
  private sendAction(action: string, data: any = {}): void {
    if (!this.sendMessage) {
      this.error("Cannot send action - no message sender configured");
      return;
    }

    this.sendMessage(this.id, action, data);
  }
}
