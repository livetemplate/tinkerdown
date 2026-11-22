/**
 * InteractiveBlock - Wraps a LiveTemplateClient instance for interactive UI
 */

import { LiveTemplateClient } from "@livetemplate/client";
import { BaseBlock } from "./base-block";
import { BlockConfig, ExecMeta, CacheMeta } from "../types";
import { PersistenceManager } from "../core/persistence-manager";
import "./exec-toolbar.css";
import "./cache-indicator.css";
import "./stream-rows.css";

/** A click waiting for the server to finish its action. */
interface PendingClick {
  el: HTMLElement;
  /** Finds the element again if an update replaced it: same tag and attribute value… */
  query: string;
  /** …and the same data-* values. */
  data: Record<string, string>;
  action: string;
  timer: ReturnType<typeof setTimeout>;
}

/** How long a click stays busy if the server never answers it. */
const PENDING_TIMEOUT_MS = 60_000;

export class InteractiveBlock extends BaseBlock {
  private client: LiveTemplateClient | null = null;
  private containerElement: HTMLElement | null = null;
  private sendMessage: ((blockID: string, action: string, data: any, ref?: string) => void) | null = null;

  // Bound event handlers (stored for removal in destroy)
  private readonly _handleClick = (e: Event) => this.handleClick(e);
  private readonly _handleSubmit = (e: Event) => this.handleSubmit(e);
  private readonly _handleChange = (e: Event) => this.handleFieldEvent(e, "change");
  private readonly _handleInput = (e: Event) => this.handleFieldEvent(e, "input");

  // Clicks whose action the server hasn't answered yet, by ref. Their element
  // shows aria-busy until the update carrying the ref arrives.
  private pending = new Map<string, PendingClick>();
  private refSeq = 0;

  // lvt-mod:debounce / lvt-mod:throttle state, per field element
  private rateLimits = new WeakMap<Element, { timer?: ReturnType<typeof setTimeout>; last: number }>();
  private timers = new Set<ReturnType<typeof setTimeout>>();

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
    this.element.classList.add("livemdtools-interactive-block");
    this.element.dataset.blockId = this.id;
    if (this.metadata.stateRef) {
      this.element.dataset.stateRef = this.metadata.stateRef;
    }

    // Initialize LiveTemplateClient for tree reconciliation
    // We don't use WebSocket connection - livemdtools handles that
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
    this.element.removeEventListener("click", this._handleClick, true);
    this.element.removeEventListener("submit", this._handleSubmit, true);
    this.element.removeEventListener("change", this._handleChange, true);
    this.element.removeEventListener("input", this._handleInput, true);
    this.timers.forEach((t) => clearTimeout(t));
    this.timers.clear();
    this.clearPending();
    this.client = null;
  }

  /**
   * Ends every pending click without an answer, as when the connection drops:
   * the server will never send their refs back.
   */
  clearPending(): void {
    Array.from(this.pending.keys()).forEach((ref) => this.finishClick(ref, false));
  }

  handleMessage(action: string, data: any, execMeta?: ExecMeta, cacheMeta?: CacheMeta, ref?: string): void {
    this.log("Received message:", action, data);

    switch (action) {
      case "tree":
        // Server sent a tree update - apply it to the DOM using LiveTemplateClient
        if (data && this.containerElement && this.client) {
          this.client.updateDOM(this.containerElement, data);
          this.log("DOM updated with tree");

          // An update re-renders elements without the aria-busy this block
          // added: finish the click this update answers, and mark the rest
          // again.
          if (ref) this.finishClick(ref, true);
          this.pending.forEach((click) => this.findClick(click)?.setAttribute("aria-busy", "true"));

          // Update cache indicator attributes
          if (cacheMeta) {
            this.updateCacheAttributes(cacheMeta);
          }

          // Update exec toolbar if present and we have exec metadata
          if (this.execToolbar && execMeta) {
            this.updateExecToolbar(execMeta);
          }
        }
        break;

      case "error":
        this.error("Server error:", data?.message);
        if (ref) this.finishClick(ref, false, data?.errors);
        break;

      default:
        this.log("Unknown action:", action);
    }
  }

  /**
   * Set the message sender (called by LivemdtoolsClient)
   */
  setMessageSender(sender: (blockID: string, action: string, data: any, ref?: string) => void): void {
    this.sendMessage = sender;
  }

  /**
   * Attach event handlers for standard HTML button/form routing and lvt-on:* attributes
   */
  private attachEventHandlers(): void {
    if (!this.element) return;

    // Delegate events to our custom handler (bound handlers stored for removal in destroy)
    this.element.addEventListener("click", this._handleClick, true);
    this.element.addEventListener("submit", this._handleSubmit, true);
    this.element.addEventListener("change", this._handleChange, true);
    this.element.addEventListener("input", this._handleInput, true);
  }

  /**
   * Handle click events.
   * Standard HTML: orphan button[name] routes to named action.
   * Tier 2: lvt-on:click="action" for non-button elements (e.g., checkboxes, table rows).
   * Uses closest() to walk up from e.target so child elements inside buttons are handled.
   */
  private handleClick(e: Event): void {
    const target = e.target as HTMLElement;

    // Walk up to find nearest orphan button with name (buttons inside forms use handleSubmit)
    const button = target.closest("button[name]") as HTMLButtonElement | null;
    if (button && button.form === null && this.element.contains(button)) {
      e.preventDefault();
      if (this.isBusy(button)) return; // its action is still running

      if (!this.checkConfirm(button)) {
        this.log("Click action cancelled by user:", button.name);
        return;
      }

      const data = this.extractData(button);
      this.sendAction(button.name, data, this.markBusy(button, "name", button.name, data));
      this.log("Click action (button name):", button.name, data);
      return;
    }

    // Walk up to find nearest element with lvt-on:click
    const lvtEl = target.closest("[lvt-on\\:click]") as HTMLElement | null;
    if (lvtEl && this.element.contains(lvtEl)) {
      const action = lvtEl.getAttribute("lvt-on:click");
      if (action) {
        e.preventDefault();
        if (this.isBusy(lvtEl)) return;

        if (!this.checkConfirm(lvtEl)) {
          this.log("Click action cancelled by user:", action);
          return;
        }

        const data = this.extractData(lvtEl);
        this.sendAction(action, data, this.markBusy(lvtEl, "lvt-on:click", action, data));
        this.log("Click action (lvt-on:click):", action, data);
      }
    }
  }

  /**
   * Handle submit events.
   * Action resolution: submitter button name > form name > "submit".
   */
  private handleSubmit(e: Event): void {
    const target = e.target as HTMLFormElement;
    const submitter = (e as SubmitEvent).submitter;

    // Determine action: submitter button name > form name
    let action = "";
    if (submitter instanceof HTMLButtonElement && submitter.name) {
      action = submitter.name;
    } else if (target.getAttribute("name")) {
      action = target.getAttribute("name")!;
    } else {
      // No routing signal — let the browser handle it natively
      return;
    }

    e.preventDefault();
    // A submit button with a name shows the wait; otherwise the form does.
    const busyEl: HTMLElement = submitter instanceof HTMLButtonElement && submitter.name ? submitter : target;
    if (this.isBusy(busyEl)) return;
    const formData = new FormData(target);
    const data: Record<string, any> = {};
    formData.forEach((value, key) => {
      // Skip the submitter button's own name=value entry to avoid leaking
      // the action name as a spurious key in the data payload
      if (submitter instanceof HTMLButtonElement && key === submitter.name) return;
      data[key] = value;
    });

    // Dispatch lvt:pending event
    target.dispatchEvent(
      new CustomEvent("lvt:pending", { bubbles: true, detail: { action } })
    );

    this.sendAction(action, data, this.markBusy(busyEl, "name", action, this.extractData(busyEl)));
    this.log("Submit action:", action, data);
  }

  /**
   * Handle field events (lvt-on:change, lvt-on:input) on inputs, selects and
   * textareas. The payload carries the value under the field's name, so
   * several fields can feed one action, and under "value" for older handlers.
   * lvt-mod:debounce="ms" waits for a pause in events; lvt-mod:throttle="ms"
   * sends at most once per interval, always ending with the latest value.
   *
   * @livetemplate/client routes these attributes for full apps, but its
   * EventDelegator needs the client to own the socket, which blocks don't, and
   * its rate-limit helpers drop the trailing value and can't be cancelled on
   * destroy. Embedding the delegator is the deeper fix; see the stream plan.
   */
  private handleFieldEvent(e: Event, eventType: "change" | "input"): void {
    const target = e.target as HTMLInputElement;
    const action = target.getAttribute?.(`lvt-on:${eventType}`);
    if (!action) return;

    const send = () => {
      const data: Record<string, any> = { value: target.value };
      if (target.name) data[target.name] = target.value;
      this.sendAction(action, data);
      this.log(`Field action (lvt-on:${eventType}):`, action, data);
    };

    const debounce = parseInt(target.getAttribute("lvt-mod:debounce") || "", 10);
    const throttle = parseInt(target.getAttribute("lvt-mod:throttle") || "", 10);
    if (!(debounce > 0) && !(throttle > 0)) {
      send();
      return;
    }

    const state = this.rateLimits.get(target) || { last: 0 };
    this.rateLimits.set(target, state);
    const schedule = (delay: number) => {
      if (state.timer) {
        clearTimeout(state.timer);
        this.timers.delete(state.timer);
      }
      const timer = setTimeout(() => {
        this.timers.delete(timer);
        state.timer = undefined;
        state.last = Date.now();
        send();
      }, delay);
      state.timer = timer;
      this.timers.add(timer);
    };

    if (debounce > 0) {
      schedule(debounce);
      return;
    }
    if (state.timer) return; // the trailing send will carry the latest value
    const wait = state.last + throttle - Date.now();
    if (wait <= 0) {
      state.last = Date.now();
      send();
    } else {
      schedule(wait);
    }
  }

  /**
   * Check for data-confirm attribute and show browser confirmation dialog.
   * Returns true if the action should proceed, false if cancelled.
   */
  private checkConfirm(element: HTMLElement): boolean {
    const message = element.dataset.confirm;
    if (message && !window.confirm(message)) {
      return false;
    }
    return true;
  }

  /**
   * Extract data-* attributes from an element.
   * Returns a plain object of key-value pairs (without the "data-" prefix).
   */
  private extractData(element: HTMLElement): Record<string, string> {
    const data: Record<string, string> = {};
    for (const key of Object.keys(element.dataset)) {
      if (key === "confirm") continue;
      const value = element.dataset[key];
      if (value !== undefined) {
        data[key] = value;
      }
    }
    return data;
  }

  /**
   * Send an action to the server
   */
  private sendAction(action: string, data: any = {}, ref?: string): void {
    if (!this.sendMessage) {
      this.error("Cannot send action - no message sender configured");
      return;
    }

    this.sendMessage(this.id, action, data, ref);
  }

  private isBusy(el: HTMLElement): boolean {
    return el.getAttribute("aria-busy") === "true";
  }

  /**
   * Marks el aria-busy until the server answers the action it sends, and
   * returns the ref that names the click. An update that replaces el is
   * matched again by tag, attr's value and the same data-* values.
   */
  private markBusy(el: HTMLElement, attr: string, action: string, data: Record<string, string>): string {
    const ref = `${this.id}:${++this.refSeq}`;
    const query = `${el.localName}[${CSS.escape(attr)}="${CSS.escape(el.getAttribute(attr) ?? "")}"]`;
    const timer = setTimeout(() => this.finishClick(ref, false), PENDING_TIMEOUT_MS);
    this.pending.set(ref, { el, query, data, action, timer });
    el.setAttribute("aria-busy", "true");
    return ref;
  }

  /**
   * Ends a pending click: clears its aria-busy, and for a form tells
   * lvt-el:{method}:on:success|error listeners how it went.
   */
  private finishClick(ref: string, ok: boolean, errors: Record<string, string> = {}): void {
    const click = this.pending.get(ref);
    if (!click) return;
    this.pending.delete(ref);
    clearTimeout(click.timer);
    const el = this.findClick(click);
    el?.removeAttribute("aria-busy");
    const form = el instanceof HTMLFormElement ? el : el?.closest("form");
    form?.dispatchEvent(new CustomEvent(ok ? "lvt:success" : "lvt:error", {
      bubbles: true,
      detail: { success: ok, errors, action: click.action },
    }));
  }

  private findClick(click: PendingClick): HTMLElement | null {
    if (this.element.contains(click.el)) return click.el;
    const same = (c: HTMLElement) => {
      const d = this.extractData(c);
      const keys = Object.keys(click.data);
      return keys.length === Object.keys(d).length && keys.every((k) => d[k] === click.data[k]);
    };
    const found = Array.from(this.element.querySelectorAll<HTMLElement>(click.query)).find(same);
    if (found) click.el = found;
    return found || null;
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
   * Update cache indicator data attributes on the block element
   */
  private updateCacheAttributes(cacheMeta: CacheMeta): void {
    if (!this.element) return;

    // Set data-cache-* attributes for CSS styling
    if (cacheMeta.stale) {
      this.element.dataset.cacheStale = "true";
    } else {
      delete this.element.dataset.cacheStale;
    }

    if (cacheMeta.refreshing) {
      this.element.dataset.cacheRefreshing = "true";
    } else {
      delete this.element.dataset.cacheRefreshing;
    }

    if (cacheMeta.cached) {
      this.element.dataset.cacheCached = "true";
      if (cacheMeta.age) {
        this.element.dataset.cacheAge = cacheMeta.age;
      }
      if (cacheMeta.expires_in) {
        this.element.dataset.cacheExpiresIn = cacheMeta.expires_in;
      }
    } else {
      delete this.element.dataset.cacheCached;
      delete this.element.dataset.cacheAge;
      delete this.element.dataset.cacheExpiresIn;
    }

    this.log("Cache attributes updated:", cacheMeta);
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
