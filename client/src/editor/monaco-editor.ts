/**
 * MonacoEditor - Wrapper for Monaco Editor with lazy loading
 */

import type * as monaco from "monaco-editor";
import { EditorOptions } from "../types";
import { loadMonaco } from "./monaco-loader";

export class MonacoEditor {
  private editor: monaco.editor.IStandaloneCodeEditor | null = null;
  private monaco: typeof monaco | null = null;
  private container: HTMLElement;
  private options: EditorOptions;
  private onChangeCallback: ((code: string) => void) | null = null;
  private initialCode: string;
  private editorDiv: HTMLElement | null = null;
  private initPromise: Promise<void> | null = null;

  constructor(container: HTMLElement, initialCode: string, options: EditorOptions) {
    this.container = container;
    this.initialCode = initialCode;
    this.options = options;

    // Start lazy initialization immediately
    this.initPromise = this.initialize();
  }

  private async initialize(): Promise<void> {
    // Show loading indicator
    const loadingDiv = document.createElement("div");
    loadingDiv.className = "livepage-monaco-loading";
    loadingDiv.style.padding = "2rem";
    loadingDiv.style.textAlign = "center";
    loadingDiv.style.color = "#999";
    loadingDiv.textContent = "Loading editor...";
    this.container.appendChild(loadingDiv);

    try {
      // Lazy load Monaco
      this.monaco = await loadMonaco();

      // Remove loading indicator
      loadingDiv.remove();

      // Create editor container
      this.editorDiv = document.createElement("div");
      this.editorDiv.className = "livepage-monaco-editor";
      this.editorDiv.style.height = "300px"; // Default height
      this.editorDiv.style.width = "100%";
      this.container.appendChild(this.editorDiv);

      // Initialize Monaco Editor
      this.editor = this.monaco.editor.create(this.editorDiv, {
        value: this.initialCode,
        language: this.options.language,
        theme: this.options.theme || "vs-dark",
        readOnly: this.options.readonly,
        minimap: {
          enabled: this.options.minimap ?? true,
        },
        lineNumbers: this.options.lineNumbers !== false ? "on" : "off",
        scrollBeyondLastLine: false,
        automaticLayout: true,
        fontSize: 14,
        tabSize: 4,
        insertSpaces: false, // Use tabs
      });

      // Listen for content changes
      this.editor.onDidChangeModelContent(() => {
        if (this.onChangeCallback && this.editor) {
          this.onChangeCallback(this.editor.getValue());
        }
      });
    } catch (error) {
      console.error("[MonacoEditor] Failed to load Monaco:", error);
      loadingDiv.textContent = "Failed to load editor";
      loadingDiv.style.color = "#f44";
    }
  }

  /**
   * Wait for editor to be ready
   */
  private async ensureReady(): Promise<void> {
    if (this.initPromise) {
      await this.initPromise;
    }
  }

  /**
   * Get current code from editor
   */
  getValue(): string {
    return this.editor?.getValue() || "";
  }

  /**
   * Set code in editor
   */
  async setValue(code: string): Promise<void> {
    await this.ensureReady();
    this.editor?.setValue(code);
  }

  /**
   * Set onChange callback
   */
  onChange(callback: (code: string) => void): void {
    this.onChangeCallback = callback;
  }

  /**
   * Set read-only mode
   */
  async setReadOnly(readonly: boolean): Promise<void> {
    await this.ensureReady();
    this.editor?.updateOptions({ readOnly: readonly });
  }

  /**
   * Focus the editor
   */
  async focus(): Promise<void> {
    await this.ensureReady();
    this.editor?.focus();
  }

  /**
   * Layout the editor (call after resize)
   */
  async layout(): Promise<void> {
    await this.ensureReady();
    this.editor?.layout();
  }

  /**
   * Destroy the editor
   */
  destroy(): void {
    this.editor?.dispose();
    this.editor = null;
  }

  /**
   * Get the underlying Monaco editor instance
   */
  async getEditor(): Promise<monaco.editor.IStandaloneCodeEditor | null> {
    await this.ensureReady();
    return this.editor;
  }
}
