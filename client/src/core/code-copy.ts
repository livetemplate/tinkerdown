/**
 * Code Copy Button Functionality
 * Adds copy buttons to all code blocks in documentation
 */

import './code-copy.css';

export class CodeCopy {
  private static readonly COPY_BUTTON_CLASS = 'code-copy-btn';
  private static readonly CODE_WRAPPER_CLASS = 'code-block-wrapper';
  private static readonly COPIED_CLASS = 'copied';

  constructor() {
    this.init();
  }

  /**
   * Initialize code copy functionality
   */
  private init(): void {
    // Wait for DOM to be ready
    if (document.readyState === 'loading') {
      document.addEventListener('DOMContentLoaded', () => this.addCopyButtons());
    } else {
      this.addCopyButtons();
    }
  }

  /**
   * Add copy buttons to all pre > code blocks
   */
  private addCopyButtons(): void {
    const codeBlocks = document.querySelectorAll('pre > code');

    codeBlocks.forEach((codeElement) => {
      const pre = codeElement.parentElement;
      if (!pre || pre.querySelector(`.${CodeCopy.COPY_BUTTON_CLASS}`)) {
        // Already has a copy button
        return;
      }

      // Wrap pre in a container for positioning
      if (!pre.parentElement?.classList.contains(CodeCopy.CODE_WRAPPER_CLASS)) {
        const wrapper = document.createElement('div');
        wrapper.className = CodeCopy.CODE_WRAPPER_CLASS;
        pre.parentNode?.insertBefore(wrapper, pre);
        wrapper.appendChild(pre);
      }

      // Create copy button
      const button = this.createCopyButton();
      pre.appendChild(button);

      // Add click handler
      button.addEventListener('click', (e) => {
        e.preventDefault();
        this.copyCode(codeElement as HTMLElement, button);
      });
    });
  }

  /**
   * Create copy button element
   */
  private createCopyButton(): HTMLButtonElement {
    const button = document.createElement('button');
    button.className = CodeCopy.COPY_BUTTON_CLASS;
    button.setAttribute('aria-label', 'Copy code to clipboard');
    button.innerHTML = this.getCopyIcon();
    return button;
  }

  /**
   * Copy code to clipboard
   */
  private async copyCode(codeElement: HTMLElement, button: HTMLButtonElement): Promise<void> {
    const code = codeElement.textContent || '';

    try {
      await navigator.clipboard.writeText(code);
      this.showCopiedFeedback(button);
    } catch (err) {
      console.error('Failed to copy code:', err);
      // Fallback for older browsers
      this.fallbackCopy(code, button);
    }
  }

  /**
   * Show "Copied!" feedback
   */
  private showCopiedFeedback(button: HTMLButtonElement): void {
    button.classList.add(CodeCopy.COPIED_CLASS);
    button.innerHTML = this.getCheckIcon();
    button.setAttribute('aria-label', 'Code copied!');

    setTimeout(() => {
      button.classList.remove(CodeCopy.COPIED_CLASS);
      button.innerHTML = this.getCopyIcon();
      button.setAttribute('aria-label', 'Copy code to clipboard');
    }, 2000);
  }

  /**
   * Fallback copy method for older browsers
   */
  private fallbackCopy(text: string, button: HTMLButtonElement): void {
    const textarea = document.createElement('textarea');
    textarea.value = text;
    textarea.style.position = 'fixed';
    textarea.style.opacity = '0';
    document.body.appendChild(textarea);
    textarea.select();

    try {
      document.execCommand('copy');
      this.showCopiedFeedback(button);
    } catch (err) {
      console.error('Fallback copy failed:', err);
    } finally {
      document.body.removeChild(textarea);
    }
  }

  /**
   * Get copy icon SVG
   */
  private getCopyIcon(): string {
    return `
      <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
        <path d="M10.5 2H3.5C2.67 2 2 2.67 2 3.5V10.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
        <rect x="5.5" y="5.5" width="8" height="8" rx="1" stroke="currentColor" stroke-width="1.5"/>
      </svg>
    `;
  }

  /**
   * Get check icon SVG (for "copied" state)
   */
  private getCheckIcon(): string {
    return `
      <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
        <path d="M3 8L6.5 11.5L13 4" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
      </svg>
    `;
  }
}
