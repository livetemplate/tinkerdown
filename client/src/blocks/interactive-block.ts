/**
 * InteractiveBlock - Wraps a LiveTemplateClient instance for interactive UI
 */

import { LiveTemplateClient, checkLvtConfirm, extractLvtData } from "@livetemplate/client";
import { BaseBlock } from "./base-block";
import { BlockConfig } from "../types";
import { PersistenceManager } from "../core/persistence-manager";

export class InteractiveBlock extends BaseBlock {
  private client: LiveTemplateClient | null = null;
  private containerElement: HTMLElement | null = null;
  private sendMessage: ((blockID: string, action: string, data: any) => void) | null = null;
  private pendingForm: HTMLFormElement | null = null;
  private pendingAction: string | null = null;

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

    // Initialize LiveTemplateClient for tree reconciliation
    // We don't use WebSocket connection - livepage handles that
    // We only use the client's tree rendering capabilities
    this.client = new LiveTemplateClient();

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
        // Server sent a tree update - apply it to the DOM using LiveTemplateClient
        if (data && this.containerElement && this.client) {
          this.client.updateDOM(this.containerElement, data);
          this.log("DOM updated with tree");

          // Dispatch lifecycle events for pending form (mimics formLifecycleManager)
          if (this.pendingForm) {
            const meta = { success: true, errors: {}, action: this.pendingAction };

            // Dispatch lvt:success event from the form
            // The reactive attribute listeners from @livetemplate/client handle
            // lvt-reset-on:success and other lvt-{action}-on:{event} attributes
            this.pendingForm.dispatchEvent(
              new CustomEvent("lvt:success", { bubbles: true, detail: meta })
            );
            this.log("Dispatched lvt:success event");

            this.pendingForm = null;
            this.pendingAction = null;
          }
        }
        break;

      case "error":
        this.error("Server error:", data.message);
        // Dispatch lvt:error event for pending form
        if (this.pendingForm) {
          const meta = { success: false, errors: data.errors || {}, action: this.pendingAction };
          this.pendingForm.dispatchEvent(
            new CustomEvent("lvt:error", { bubbles: true, detail: meta })
          );
          this.pendingForm = null;
          this.pendingAction = null;
        }
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
   * Supports lvt-data-* attributes to pass data with the action
   * Supports lvt-confirm="message" for confirmation dialogs (uses shared utility from @livetemplate/client)
   * Example: <button lvt-click="Delete" lvt-data-id="123" lvt-confirm="Are you sure?">Delete</button>
   */
  private handleClick(e: Event): void {
    const target = e.target as HTMLElement;
    const action = target.getAttribute("lvt-click");

    if (action) {
      e.preventDefault();

      // Check for confirmation dialog (uses shared utility from @livetemplate/client)
      if (!checkLvtConfirm(target)) {
        this.log("Click action cancelled by user:", action);
        return;
      }

      const data = extractLvtData(target);
      this.sendAction(action, data);
      this.log("Click action:", action, data);
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

      // Track form and action for lifecycle events
      this.pendingForm = target;
      this.pendingAction = action;

      // Dispatch lvt:pending event
      target.dispatchEvent(
        new CustomEvent("lvt:pending", { bubbles: true, detail: { action } })
      );

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
