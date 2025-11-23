/**
 * RunButton - Button for executing WASM code
 */

export type RunButtonCallback = () => void | Promise<void>;

export class RunButton {
  private element: HTMLButtonElement;
  private callback: RunButtonCallback | null = null;
  private isRunning = false;

  constructor(container: HTMLElement, label = "Run") {
    this.element = this.createButton(label);
    container.appendChild(this.element);
  }

  private createButton(label: string): HTMLButtonElement {
    const button = document.createElement("button");
    button.className = "livepage-run-button";
    button.textContent = label;
    button.addEventListener("click", () => this.handleClick());
    return button;
  }

  /**
   * Set the click callback
   */
  onClick(callback: RunButtonCallback): void {
    this.callback = callback;
  }

  /**
   * Handle button click
   */
  private async handleClick(): Promise<void> {
    if (this.isRunning || !this.callback) return;

    this.setRunning(true);

    try {
      await this.callback();
    } catch (error) {
      console.error("[RunButton] Error executing callback:", error);
    } finally {
      this.setRunning(false);
    }
  }

  /**
   * Set running state
   */
  setRunning(running: boolean): void {
    this.isRunning = running;
    this.element.disabled = running;

    if (running) {
      this.element.classList.add("running");
      this.element.textContent = "Running...";
    } else {
      this.element.classList.remove("running");
      this.element.textContent = "Run";
    }
  }

  /**
   * Enable the button
   */
  enable(): void {
    this.element.disabled = false;
  }

  /**
   * Disable the button
   */
  disable(): void {
    this.element.disabled = true;
  }

  /**
   * Get the button element
   */
  getElement(): HTMLButtonElement {
    return this.element;
  }

  /**
   * Destroy the button
   */
  destroy(): void {
    this.element.remove();
  }
}
