/**
 * OutputPanel - Displays execution output (stdout/stderr)
 */

export interface OutputLine {
  type: "stdout" | "stderr" | "error" | "info";
  text: string;
  timestamp: number;
}

export class OutputPanel {
  private element: HTMLElement;
  private lines: OutputLine[] = [];
  private maxLines: number;
  private autoScroll: boolean;

  constructor(container: HTMLElement, maxLines = 1000, autoScroll = true) {
    this.maxLines = maxLines;
    this.autoScroll = autoScroll;

    // Create output panel element
    this.element = this.createPanel();
    container.appendChild(this.element);
  }

  private createPanel(): HTMLElement {
    const panel = document.createElement("div");
    panel.className = "livepage-output-panel";
    panel.innerHTML = `
      <div class="output-header">
        <span class="output-title">Output</span>
        <button class="output-clear" title="Clear output">Clear</button>
      </div>
      <div class="output-content"></div>
    `;

    // Attach clear button handler
    const clearBtn = panel.querySelector(".output-clear");
    if (clearBtn) {
      clearBtn.addEventListener("click", () => this.clear());
    }

    return panel;
  }

  /**
   * Append a line to the output
   */
  append(type: OutputLine["type"], text: string): void {
    const line: OutputLine = {
      type,
      text,
      timestamp: Date.now(),
    };

    this.lines.push(line);

    // Trim if exceeds max lines
    if (this.lines.length > this.maxLines) {
      this.lines = this.lines.slice(-this.maxLines);
    }

    this.render();
  }

  /**
   * Append stdout
   */
  stdout(text: string): void {
    this.append("stdout", text);
  }

  /**
   * Append stderr
   */
  stderr(text: string): void {
    this.append("stderr", text);
  }

  /**
   * Append error
   */
  error(text: string): void {
    this.append("error", text);
  }

  /**
   * Append info message
   */
  info(text: string): void {
    this.append("info", text);
  }

  /**
   * Clear all output
   */
  clear(): void {
    this.lines = [];
    this.render();
  }

  /**
   * Show the panel
   */
  show(): void {
    this.element.style.display = "block";
  }

  /**
   * Hide the panel
   */
  hide(): void {
    this.element.style.display = "none";
  }

  /**
   * Render the output lines
   */
  private render(): void {
    const content = this.element.querySelector(".output-content");
    if (!content) return;

    // Clear existing content
    content.innerHTML = "";

    // Render each line
    for (const line of this.lines) {
      const lineElement = document.createElement("div");
      lineElement.className = `output-line output-${line.type}`;
      lineElement.textContent = line.text;
      content.appendChild(lineElement);
    }

    // Auto-scroll to bottom
    if (this.autoScroll) {
      content.scrollTop = content.scrollHeight;
    }
  }

  /**
   * Get the panel element
   */
  getElement(): HTMLElement {
    return this.element;
  }

  /**
   * Destroy the panel
   */
  destroy(): void {
    this.element.remove();
  }
}
