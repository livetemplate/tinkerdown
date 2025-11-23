/**
 * PersistenceManager - Handles localStorage for code edits and state
 */

import { PersistenceData } from "../types";

export class PersistenceManager {
  private storageKey: string;
  private enabled: boolean;
  private debug: boolean;

  constructor(storageKey = "livepage:persistence", enabled = true, debug = false) {
    this.storageKey = storageKey;
    this.enabled = enabled && this.isLocalStorageAvailable();
    this.debug = debug;

    if (!this.enabled && enabled) {
      console.warn("[PersistenceManager] localStorage not available, persistence disabled");
    }
  }

  /**
   * Check if localStorage is available
   */
  private isLocalStorageAvailable(): boolean {
    try {
      const test = "__localStorage_test__";
      localStorage.setItem(test, test);
      localStorage.removeItem(test);
      return true;
    } catch (e) {
      return false;
    }
  }

  /**
   * Save code for a specific block
   */
  saveCode(blockID: string, code: string): void {
    if (!this.enabled) return;

    try {
      const data = this.loadAll();
      data.code[blockID] = code;
      data.timestamp = Date.now();

      localStorage.setItem(this.storageKey, JSON.stringify(data));

      if (this.debug) {
        console.log(`[PersistenceManager] Saved code for block: ${blockID}`);
      }
    } catch (error) {
      console.error("[PersistenceManager] Error saving code:", error);
    }
  }

  /**
   * Load code for a specific block
   */
  loadCode(blockID: string): string | null {
    if (!this.enabled) return null;

    try {
      const data = this.loadAll();
      return data.code[blockID] || null;
    } catch (error) {
      console.error("[PersistenceManager] Error loading code:", error);
      return null;
    }
  }

  /**
   * Load all persisted data
   */
  loadAll(): PersistenceData {
    if (!this.enabled) {
      return { code: {}, timestamp: Date.now() };
    }

    try {
      const raw = localStorage.getItem(this.storageKey);
      if (!raw) {
        return { code: {}, timestamp: Date.now() };
      }

      const data = JSON.parse(raw) as PersistenceData;

      // Validate structure
      if (!data.code || typeof data.code !== "object") {
        console.warn("[PersistenceManager] Invalid data structure, resetting");
        return { code: {}, timestamp: Date.now() };
      }

      return data;
    } catch (error) {
      console.error("[PersistenceManager] Error loading data:", error);
      return { code: {}, timestamp: Date.now() };
    }
  }

  /**
   * Clear code for a specific block
   */
  clearCode(blockID: string): void {
    if (!this.enabled) return;

    try {
      const data = this.loadAll();
      delete data.code[blockID];
      data.timestamp = Date.now();

      localStorage.setItem(this.storageKey, JSON.stringify(data));

      if (this.debug) {
        console.log(`[PersistenceManager] Cleared code for block: ${blockID}`);
      }
    } catch (error) {
      console.error("[PersistenceManager] Error clearing code:", error);
    }
  }

  /**
   * Clear all persisted data
   */
  clearAll(): void {
    if (!this.enabled) return;

    try {
      localStorage.removeItem(this.storageKey);

      if (this.debug) {
        console.log("[PersistenceManager] Cleared all persisted data");
      }
    } catch (error) {
      console.error("[PersistenceManager] Error clearing all data:", error);
    }
  }

  /**
   * Get all persisted block IDs
   */
  getPersistedBlocks(): string[] {
    if (!this.enabled) return [];

    const data = this.loadAll();
    return Object.keys(data.code);
  }

  /**
   * Check if a block has persisted code
   */
  hasPersistedCode(blockID: string): boolean {
    if (!this.enabled) return false;

    const data = this.loadAll();
    return blockID in data.code;
  }

  /**
   * Get the timestamp of last persistence update
   */
  getLastUpdate(): number | null {
    if (!this.enabled) return null;

    const data = this.loadAll();
    return data.timestamp;
  }
}
