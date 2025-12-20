/**
 * InteractiveBlock - Wraps a LiveTemplateClient instance for interactive UI
 */

import { LiveTemplateClient, checkLvtConfirm, extractLvtData } from "@livetemplate/client";
import { BaseBlock } from "./base-block";
import { BlockConfig, ExecMeta } from "../types";
import { PersistenceManager } from "../core/persistence-manager";
import "./exec-toolbar.css";

export class InteractiveBlock extends BaseBlock {
  private client: LiveTemplateClient | null = null;
  private containerElement: HTMLElement | null = null;
  private sendMessage: ((blockID: string, action: string, data: any) => void) | null = null;
  private pendingForm: HTMLFormElement | null = null;
  private pendingAction: string | null = null;

  // Exec toolbar state
  private execToolbar: HTMLElement | null = null;
  private outputPanel: HTMLElement | null = null;
  private outputExpanded = false;

  constructor(config: BlockConfig, persistence: PersistenceManager, debug = false) {
    super(config, persistence, debug);
  }

  initialize(): void {
    this.log("Initializing interactive block");

    // Find container for the interactive content
    // If data-interactive-content is on the element itself, create a wrapper
    // so we can insert toolbar/output outside the morphdom zone
    let existingContainer = this.element.querySelector("[data-interactive-content]");
    if (existingContainer) {
      this.containerElement = existingContainer as HTMLElement;
    } else {
      // Create a dedicated content container so toolbar stays outside morphdom zone
      const contentWrapper = document.createElement("div");
      contentWrapper.className = "exec-content-wrapper";
      contentWrapper.dataset.interactiveContent = "true";
      // Move existing children into the wrapper
      while (this.element.firstChild) {
        contentWrapper.appendChild(this.element.firstChild);
      }
      this.element.appendChild(contentWrapper);
      this.containerElement = contentWrapper;
    }

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

    // Check if this is an exec source block and inject toolbar
    if (this.element.dataset.execSource === "true") {
      this.injectExecToolbar();
    }

    this.log("Interactive block initialized");
  }

  destroy(): void {
    this.log("Destroying interactive block");
    if (this.client) {
      // Clean up the client instance
      this.client = null;
    }
  }

  handleMessage(action: string, data: any, execMeta?: ExecMeta): void {
    this.log("Received message:", action, data);

    switch (action) {
      case "tree":
        // Server sent a tree update - apply it to the DOM using LiveTemplateClient
        if (data && this.containerElement && this.client) {
          this.client.updateDOM(this.containerElement, data);
          this.log("DOM updated with tree");

          // Update exec toolbar if present and we have exec metadata
          if (this.execToolbar && execMeta) {
            this.updateExecToolbar(execMeta);
          }

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

  /**
   * Inject exec toolbar for runbook/notebook-style command execution
   */
  private injectExecToolbar(): void {
    const command = this.element.dataset.execCommand || "...";

    // Create toolbar
    this.execToolbar = document.createElement("div");
    this.execToolbar.className = "exec-toolbar";
    this.execToolbar.innerHTML = `
      <div class="exec-toolbar-command"><code>${this.escapeHtml(command)}</code></div>
      <div class="exec-toolbar-status idle"><span>Ready</span></div>
      <span class="exec-toolbar-duration"></span>
      <button class="exec-toolbar-run-btn" type="button">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
          <path d="M8 5v14l11-7z"/>
        </svg>
        Run
      </button>
    `;

    // Create output panel
    this.outputPanel = document.createElement("div");
    this.outputPanel.className = "exec-output-panel";
    this.outputPanel.innerHTML = `
      <button class="exec-output-toggle" type="button">
        <svg class="exec-output-toggle-icon" width="12" height="12" viewBox="0 0 24 24" fill="currentColor">
          <path d="M10 6L8.59 7.41 13.17 12l-4.58 4.59L10 18l6-6z"/>
        </svg>
        Output
      </button>
      <div class="exec-output-content">
        <pre class="exec-output-stdout"></pre>
        <pre class="exec-output-stderr"></pre>
      </div>
    `;

    // Insert toolbar BEFORE the content container (so it's outside morphdom zone)
    if (this.containerElement && this.containerElement.parentNode === this.element) {
      this.element.insertBefore(this.execToolbar, this.containerElement);
      this.execToolbar.after(this.outputPanel);
    } else {
      // Fallback: insert at beginning
      this.element.insertBefore(this.execToolbar, this.element.firstChild);
      this.execToolbar.after(this.outputPanel);
    }

    // Attach event handlers
    this.execToolbar.querySelector(".exec-toolbar-run-btn")
      ?.addEventListener("click", () => this.handleExecRun());
    this.outputPanel.querySelector(".exec-output-toggle")
      ?.addEventListener("click", () => this.toggleOutput());

    this.log("Exec toolbar injected");
  }

  /**
   * Handle Run button click - sends Run action to server
   */
  private handleExecRun(): void {
    this.log("Run button clicked");
    this.sendAction("Run", {});
  }

  /**
   * Toggle output panel visibility
   */
  private toggleOutput(): void {
    this.outputExpanded = !this.outputExpanded;
    this.outputPanel?.querySelector(".exec-output-toggle")
      ?.classList.toggle("expanded", this.outputExpanded);
    this.outputPanel?.querySelector(".exec-output-content")
      ?.classList.toggle("expanded", this.outputExpanded);
  }

  /**
   * Update exec toolbar with state from server
   */
  private updateExecToolbar(execMeta: ExecMeta): void {
    const status = execMeta.status || "idle";
    const statusEl = this.execToolbar?.querySelector(".exec-toolbar-status");
    const durationEl = this.execToolbar?.querySelector(".exec-toolbar-duration");
    const runBtn = this.execToolbar?.querySelector(".exec-toolbar-run-btn") as HTMLButtonElement | null;

    // Update status indicator
    if (statusEl) {
      statusEl.className = `exec-toolbar-status ${status}`;
      if (status === "running") {
        statusEl.innerHTML = '<span class="exec-spinner"></span>';
      } else if (status === "success") {
        statusEl.innerHTML = '<span>✓ Success</span>';
      } else if (status === "error") {
        statusEl.innerHTML = '<span>✗ Error</span>';
      } else {
        statusEl.innerHTML = '<span>Ready</span>';
      }
    }

    // Update duration
    if (durationEl && execMeta.duration) {
      durationEl.textContent = `${execMeta.duration}ms`;
    }

    // Update button state
    if (runBtn) {
      runBtn.disabled = status === "running";
      if (status === "running") {
        runBtn.classList.add("running");
        runBtn.innerHTML = `
          <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
            <path d="M8 5v14l11-7z"/>
          </svg>
          Running...
        `;
      } else {
        runBtn.classList.remove("running");
        runBtn.innerHTML = `
          <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
            <path d="M8 5v14l11-7z"/>
          </svg>
          Run
        `;
      }
    }

    // Update output panel content
    const stdoutEl = this.outputPanel?.querySelector(".exec-output-stdout");
    const stderrEl = this.outputPanel?.querySelector(".exec-output-stderr") as HTMLElement | null;
    if (stdoutEl) {
      stdoutEl.textContent = execMeta.output || "";
    }
    if (stderrEl) {
      stderrEl.textContent = execMeta.stderr || "";
      stderrEl.style.display = execMeta.stderr ? "block" : "none";
    }

    this.log("Exec toolbar updated:", status, execMeta.duration);
  }

  /**
   * Escape HTML to prevent XSS
   */
  private escapeHtml(text: string): string {
    const div = document.createElement("div");
    div.textContent = text;
    return div.innerHTML;
  }
}
