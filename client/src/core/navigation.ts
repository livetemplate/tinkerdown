/**
 * Tutorial Step Navigation
 *
 * Provides navigation controls for multi-step tutorials including:
 * - Sidebar table of contents
 * - Bottom navigation bar (prev/next buttons)
 * - Keyboard shortcuts
 * - URL hash support for deep linking
 * - Smooth scrolling between sections
 */

export interface TutorialStep {
  id: string;
  title: string;
  element: HTMLElement;
  index: number;
}

export class TutorialNavigation {
  private steps: TutorialStep[] = [];
  private currentStepIndex: number = 0;
  private sidebar: HTMLElement | null = null;
  private bottomNav: HTMLElement | null = null;

  constructor() {
    this.init();
  }

  private init() {
    // Skip tutorial navigation if site navigation exists
    if (document.querySelector('.livepage-nav-sidebar')) {
      return; // Site mode - navigation already rendered by server
    }

    // Parse H2 headings as tutorial steps
    this.parseSteps();

    if (this.steps.length === 0) {
      return; // No multi-step tutorial
    }

    // Create navigation UI
    this.createSidebar();
    this.createBottomNav();

    // Set up event listeners
    this.setupKeyboardShortcuts();
    this.setupHashNavigation();
    this.setupScrollTracking();

    // Initialize from URL hash or show first step
    this.initializeFromHash();
  }

  private parseSteps() {
    const headings = document.querySelectorAll('h2');

    headings.forEach((heading, index) => {
      const id = heading.id || this.generateId(heading.textContent || '');

      // Ensure heading has an ID for navigation
      if (!heading.id) {
        heading.id = id;
      }

      this.steps.push({
        id,
        title: heading.textContent || '',
        element: heading as HTMLElement,
        index
      });
    });
  }

  private generateId(text: string): string {
    return text
      .toLowerCase()
      .replace(/[^\w\s-]/g, '')
      .replace(/\s+/g, '-')
      .replace(/-+/g, '-')
      .trim();
  }

  private createSidebar() {
    // Create sidebar container
    this.sidebar = document.createElement('nav');
    this.sidebar.className = 'livepage-nav-sidebar';
    this.sidebar.innerHTML = `
      <div class="nav-sidebar-header">
        <h3>Contents</h3>
      </div>
      <ol class="nav-sidebar-steps">
        ${this.steps.map((step, i) => `
          <li class="nav-step ${i === 0 ? 'active' : ''}" data-step="${i}">
            <a href="#${step.id}">
              <span class="step-number">${i + 1}</span>
              <span class="step-title">${step.title}</span>
            </a>
          </li>
        `).join('')}
      </ol>
    `;

    // Add click handlers
    this.sidebar.querySelectorAll('.nav-step').forEach((stepEl, i) => {
      stepEl.addEventListener('click', (e) => {
        e.preventDefault();
        this.navigateToStep(i);
      });
    });

    document.body.appendChild(this.sidebar);
  }

  private createBottomNav() {
    this.bottomNav = document.createElement('nav');
    this.bottomNav.className = 'livepage-nav-bottom';
    this.bottomNav.innerHTML = `
      <button class="nav-btn nav-prev" ${this.currentStepIndex === 0 ? 'disabled' : ''}>
        <span class="nav-arrow">←</span>
        <span class="nav-label">Previous</span>
      </button>
      <span class="nav-progress">
        Step <span class="current-step">1</span> of <span class="total-steps">${this.steps.length}</span>
      </span>
      <button class="nav-btn nav-next" ${this.currentStepIndex === this.steps.length - 1 ? 'disabled' : ''}>
        <span class="nav-label">Next</span>
        <span class="nav-arrow">→</span>
      </button>
    `;

    // Add click handlers
    const prevBtn = this.bottomNav.querySelector('.nav-prev');
    const nextBtn = this.bottomNav.querySelector('.nav-next');

    prevBtn?.addEventListener('click', () => this.navigatePrev());
    nextBtn?.addEventListener('click', () => this.navigateNext());

    document.body.appendChild(this.bottomNav);
  }

  private setupKeyboardShortcuts() {
    document.addEventListener('keydown', (e) => {
      // Ignore if user is typing in an input
      if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) {
        return;
      }

      switch (e.key) {
        case 'ArrowRight':
        case 'ArrowDown':
          e.preventDefault();
          this.navigateNext();
          break;
        case 'ArrowLeft':
        case 'ArrowUp':
          e.preventDefault();
          this.navigatePrev();
          break;
      }
    });
  }

  private setupHashNavigation() {
    window.addEventListener('hashchange', () => {
      this.handleHashChange();
    });
  }

  private setupScrollTracking() {
    // Track which section is currently visible
    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            const heading = entry.target as HTMLElement;
            const stepIndex = this.steps.findIndex(s => s.element === heading);
            if (stepIndex !== -1 && stepIndex !== this.currentStepIndex) {
              this.updateCurrentStep(stepIndex, false); // Don't scroll, just update UI
            }
          }
        });
      },
      {
        threshold: 0.5,
        rootMargin: '-100px 0px -50% 0px'
      }
    );

    this.steps.forEach(step => observer.observe(step.element));
  }

  private initializeFromHash() {
    const hash = window.location.hash.slice(1);
    if (hash) {
      const stepIndex = this.steps.findIndex(s => s.id === hash);
      if (stepIndex !== -1) {
        this.navigateToStep(stepIndex);
        return;
      }
    }

    // Default to first step
    this.updateCurrentStep(0, false);
  }

  private handleHashChange() {
    const hash = window.location.hash.slice(1);
    if (hash) {
      const stepIndex = this.steps.findIndex(s => s.id === hash);
      if (stepIndex !== -1) {
        this.navigateToStep(stepIndex, false); // Don't update hash again
      }
    }
  }

  private navigateToStep(index: number, updateHash: boolean = true) {
    if (index < 0 || index >= this.steps.length) {
      return;
    }

    this.updateCurrentStep(index, true);

    if (updateHash) {
      history.pushState(null, '', `#${this.steps[index].id}`);
    }
  }

  private navigateNext() {
    if (this.currentStepIndex < this.steps.length - 1) {
      this.navigateToStep(this.currentStepIndex + 1);
    }
  }

  private navigatePrev() {
    if (this.currentStepIndex > 0) {
      this.navigateToStep(this.currentStepIndex - 1);
    }
  }

  private updateCurrentStep(index: number, scroll: boolean = true) {
    this.currentStepIndex = index;

    // Update sidebar
    if (this.sidebar) {
      this.sidebar.querySelectorAll('.nav-step').forEach((el, i) => {
        el.classList.toggle('active', i === index);
      });
    }

    // Update bottom nav
    if (this.bottomNav) {
      const prevBtn = this.bottomNav.querySelector('.nav-prev') as HTMLButtonElement;
      const nextBtn = this.bottomNav.querySelector('.nav-next') as HTMLButtonElement;
      const currentStepEl = this.bottomNav.querySelector('.current-step');

      if (prevBtn) prevBtn.disabled = index === 0;
      if (nextBtn) nextBtn.disabled = index === this.steps.length - 1;
      if (currentStepEl) currentStepEl.textContent = String(index + 1);
    }

    // Scroll to step
    if (scroll) {
      this.scrollToStep(index);
    }
  }

  private scrollToStep(index: number) {
    const step = this.steps[index];
    if (!step) return;

    step.element.scrollIntoView({
      behavior: 'smooth',
      block: 'start'
    });
  }

  // Public API
  public getCurrentStep(): number {
    return this.currentStepIndex;
  }

  public getTotalSteps(): number {
    return this.steps.length;
  }

  public goToStep(index: number) {
    this.navigateToStep(index);
  }
}
