/**
 * WasmBlock - Editable code block with WASM execution
 */

import { BaseBlock } from "./base-block";
import { BlockConfig, ExecMeta, CacheMeta } from "../types";
import { PersistenceManager } from "../core/persistence-manager";
import { MonacoEditor } from "../editor/monaco-editor";
import { OutputPanel } from "../ui/output-panel";
import { RunButton } from "../ui/run-button";
import { TinyGoExecutor } from "../wasm/tinygo-executor";

export class WasmBlock extends BaseBlock {
  private editor: MonacoEditor | null = null;
  private outputPanel: OutputPanel | null = null;
  private runButton: RunButton | null = null;
  private executor: TinyGoExecutor | null = null;
  private containerElement: HTMLElement | null = null;
  private editorContainer: HTMLElement | null = null;
  private controlsContainer: HTMLElement | null = null;

  constructor(config: BlockConfig, persistence: PersistenceManager, debug = false) {
    super(config, persistence, debug);
  }

  initialize(): void {
    this.log("Initializing WASM block");

    // Load persisted code
    this.currentCode = this.loadPersistedCode();

    // Create block structure
    this.createBlockStructure();

    // Initialize components
    this.initializeEditor();
    this.initializeControls();
    this.initializeExecutor();

    this.log("WASM block initialized");
  }

  destroy(): void {
    this.log("Destroying WASM block");

    this.editor?.destroy();
    this.outputPanel?.destroy();
    this.runButton?.destroy();

    this.editor = null;
    this.outputPanel = null;
    this.runButton = null;
    this.executor = null;
  }

  handleMessage(action: string, data: any, _execMeta?: ExecMeta, _cacheMeta?: CacheMeta): void {
    this.log("Received message:", action, data);
    // WASM blocks don't receive messages from server (client-side execution)
  }

  /**
   * Create the block structure
   */
  private createBlockStructure(): void {
    // Wrap existing element or create new structure
    this.containerElement = document.createElement("div");
    this.containerElement.className = "livemdtools-wasm-block";
    this.containerElement.dataset.blockId = this.id;

    // Editor container
    this.editorContainer = document.createElement("div");
    this.editorContainer.className = "wasm-editor-container";
    this.containerElement.appendChild(this.editorContainer);

    // Controls container (run button, reset, etc.)
    this.controlsContainer = document.createElement("div");
    this.controlsContainer.className = "wasm-controls";
    this.containerElement.appendChild(this.controlsContainer);

    // Replace or append to original element
    if (this.element.parentNode) {
      this.element.parentNode.replaceChild(this.containerElement, this.element);
    }
  }

  /**
   * Initialize Monaco editor
   */
  private initializeEditor(): void {
    if (!this.editorContainer) return;

    this.editor = new MonacoEditor(
      this.editorContainer,
      this.currentCode,
      {
        language: this.metadata.language || "go",
        readonly: this.metadata.readonly || false,
        theme: "vs-dark",
        minimap: false,
        lineNumbers: true,
      }
    );

    // Save code changes to persistence
    this.editor.onChange((code) => {
      this.setCode(code);
    });

    this.log("Editor initialized");
  }

  /**
   * Initialize controls (run button, reset, etc.)
   */
  private initializeControls(): void {
    if (!this.controlsContainer) return;

    // Create button container
    const buttonContainer = document.createElement("div");
    buttonContainer.className = "wasm-buttons";

    // Run button
    this.runButton = new RunButton(buttonContainer, "Run");
    this.runButton.onClick(() => this.executeCode());

    // Reset button
    const resetButton = document.createElement("button");
    resetButton.className = "livemdtools-reset-button";
    resetButton.textContent = "Reset";
    resetButton.addEventListener("click", () => this.resetCode());
    buttonContainer.appendChild(resetButton);

    this.controlsContainer.appendChild(buttonContainer);

    // Output panel
    this.outputPanel = new OutputPanel(this.controlsContainer);
    this.outputPanel.hide(); // Hide until first run

    this.log("Controls initialized");
  }

  /**
   * Initialize WASM executor
   */
  private initializeExecutor(): void {
    this.executor = new TinyGoExecutor("/api/compile", this.debug);
    this.log("Executor initialized");
  }

  /**
   * Execute the current code
   */
  private async executeCode(): Promise<void> {
    if (!this.editor || !this.executor || !this.outputPanel) {
      this.error("Cannot execute - components not initialized");
      return;
    }

    const code = this.editor.getValue();

    this.log("Executing code");
    this.outputPanel.clear();
    this.outputPanel.show();
    this.outputPanel.info("Compiling and running...");

    try {
      const result = await this.executor.execute(code);

      // Clear "Compiling..." message
      this.outputPanel.clear();

      // Display output
      if (result.stdout) {
        this.outputPanel.stdout(result.stdout);
      }

      if (result.stderr) {
        this.outputPanel.stderr(result.stderr);
      }

      if (result.error) {
        this.outputPanel.error(result.error);
      }

      if (result.exitCode === 0 && !result.stdout && !result.stderr) {
        this.outputPanel.info("Program completed successfully (no output)");
      }

      this.log("Execution completed with exit code:", result.exitCode);
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : String(error);
      this.outputPanel.error(`Execution error: ${errorMsg}`);
      this.error("Execution error:", error);
    }
  }

  /**
   * Reset code to initial state
   */
  private resetCode(): void {
    if (!this.editor) return;

    this.editor.setValue(this.initialCode);
    this.setCode(this.initialCode);
    this.outputPanel?.clear();
    this.log("Code reset to initial state");
  }
}
