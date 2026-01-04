/**
 * MessageRouter - Multiplexes WebSocket messages by blockID
 */

import { MessageEnvelope, ExecMeta, CacheMeta } from "../types";

export type MessageHandler = (action: string, data: any, execMeta?: ExecMeta, cacheMeta?: CacheMeta) => void;

export class MessageRouter {
  private handlers: Map<string, MessageHandler> = new Map();
  private debug: boolean;

  constructor(debug = false) {
    this.debug = debug;
  }

  /**
   * Register a handler for a specific block ID
   */
  register(blockID: string, handler: MessageHandler): void {
    if (this.handlers.has(blockID)) {
      console.warn(`[MessageRouter] Overwriting handler for block: ${blockID}`);
    }
    this.handlers.set(blockID, handler);
    if (this.debug) {
      console.log(`[MessageRouter] Registered handler for block: ${blockID}`);
    }
  }

  /**
   * Unregister a handler for a specific block ID
   */
  unregister(blockID: string): void {
    this.handlers.delete(blockID);
    if (this.debug) {
      console.log(`[MessageRouter] Unregistered handler for block: ${blockID}`);
    }
  }

  /**
   * Route an incoming message to the appropriate handler
   */
  route(message: string | MessageEnvelope): void {
    try {
      const envelope: MessageEnvelope =
        typeof message === "string" ? JSON.parse(message) : message;

      const { blockID, action, data, execMeta, cacheMeta } = envelope;

      // Handle reload action (special case - no blockID)
      if (action === "reload") {
        this.handleReload(data?.filePath || "");
        return;
      }

      if (!blockID) {
        console.error("[MessageRouter] Message missing blockID:", envelope);
        return;
      }

      const handler = this.handlers.get(blockID);
      if (!handler) {
        console.warn(`[MessageRouter] No handler for block: ${blockID}`);
        return;
      }

      if (this.debug) {
        console.log(`[MessageRouter] Routing to ${blockID}:`, { action, data });
      }

      handler(action, data, execMeta, cacheMeta);
    } catch (error) {
      console.error("[MessageRouter] Error routing message:", error);
    }
  }

  /**
   * Handle reload message from server
   */
  private handleReload(filePath: string): void {
    console.log(`[MessageRouter] Page reloading: ${filePath} changed`);

    // Show notification
    this.showReloadNotification(filePath);

    // Reload the page after a short delay to show the notification
    setTimeout(() => {
      window.location.reload();
    }, 500);
  }

  /**
   * Show reload notification overlay
   */
  private showReloadNotification(filePath: string): void {
    const existing = document.getElementById("livemdtools-reload-notification");
    if (existing) {
      existing.remove();
    }

    const notification = document.createElement("div");
    notification.id = "livemdtools-reload-notification";
    notification.innerHTML = `
      <div style="
        position: fixed;
        top: 20px;
        right: 20px;
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        color: white;
        padding: 16px 20px;
        border-radius: 8px;
        box-shadow: 0 4px 12px rgba(0,0,0,0.15);
        z-index: 100000;
        font-family: system-ui, -apple-system, sans-serif;
        font-size: 14px;
        animation: slideIn 0.3s ease-out;
      ">
        <div style="display: flex; align-items: center; gap: 10px;">
          <div style="font-size: 20px;">🔄</div>
          <div>
            <div style="font-weight: 600;">File Updated</div>
            <div style="opacity: 0.9; font-size: 12px; margin-top: 2px;">${filePath}</div>
          </div>
        </div>
      </div>
    `;

    // Add animation
    const style = document.createElement("style");
    style.textContent = `
      @keyframes slideIn {
        from {
          transform: translateX(400px);
          opacity: 0;
        }
        to {
          transform: translateX(0);
          opacity: 1;
        }
      }
    `;
    document.head.appendChild(style);
    document.body.appendChild(notification);
  }

  /**
   * Send a message to the server (formatted as envelope)
   */
  createEnvelope(blockID: string, action: string, data: any = {}): MessageEnvelope {
    return { blockID, action, data };
  }

  /**
   * Get all registered block IDs
   */
  getRegisteredBlocks(): string[] {
    return Array.from(this.handlers.keys());
  }

  /**
   * Clear all handlers
   */
  clear(): void {
    this.handlers.clear();
    if (this.debug) {
      console.log("[MessageRouter] Cleared all handlers");
    }
  }
}
