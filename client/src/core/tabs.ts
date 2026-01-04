/**
 * Tinkerdown Tabs
 *
 * Handles tabbed heading interactions for filtering data.
 * Tabs are created from headings with syntax: ## [All] | [Active] not done | [Done] done
 *
 * When a tab is clicked:
 * 1. Updates visual active state
 * 2. Finds the associated interactive block
 * 3. Sends a Filter action to apply the tab's filter expression
 */

import "./tabs.css";

export interface TabGroup {
  id: string;
  container: HTMLElement;
  tabs: HTMLButtonElement[];
  activeIndex: number;
  contentPanel: HTMLElement | null;
}

export class TabsController {
  private tabGroups: Map<string, TabGroup> = new Map();
  private sendMessage:
    | ((blockID: string, action: string, data: any) => void)
    | null = null;
  private debug: boolean;

  constructor(debug = false) {
    this.debug = debug;
    this.init();
  }

  private log(...args: any[]): void {
    if (this.debug) {
      console.log("[TabsController]", ...args);
    }
  }

  /**
   * Set the message sender (called by TinkerdownClient)
   */
  setMessageSender(
    sender: (blockID: string, action: string, data: any) => void
  ): void {
    this.sendMessage = sender;
  }

  private init(): void {
    // Find all tab containers
    const containers = document.querySelectorAll<HTMLElement>(
      ".tinkerdown-tabs"
    );

    containers.forEach((container) => {
      const id = container.dataset.tabsId;
      if (!id) return;

      const tabs = Array.from(
        container.querySelectorAll<HTMLButtonElement>(".tinkerdown-tab")
      );
      const contentPanel = container.querySelector<HTMLElement>(
        "[data-tabs-content]"
      );

      if (tabs.length === 0) return;

      const group: TabGroup = {
        id,
        container,
        tabs,
        activeIndex: 0,
        contentPanel,
      };

      this.tabGroups.set(id, group);

      // Attach click handlers
      tabs.forEach((tab, index) => {
        tab.addEventListener("click", () => this.handleTabClick(id, index));
        tab.addEventListener("keydown", (e) =>
          this.handleTabKeydown(e, id, index)
        );
      });

      // Move content into the panel
      this.wrapContent(container, contentPanel);

      this.log("Registered tab group:", id, "with", tabs.length, "tabs");
    });

    this.log("Initialized with", this.tabGroups.size, "tab groups");
  }

  /**
   * Wrap the content following the tab bar into the content panel.
   * The content is everything between this tab group and the next heading or tab group.
   */
  private wrapContent(
    container: HTMLElement,
    contentPanel: HTMLElement | null
  ): void {
    if (!contentPanel) return;

    // Find all siblings after the container until we hit another heading or end
    const elementsToWrap: Node[] = [];
    let sibling = container.nextSibling;

    while (sibling) {
      // Stop at another tabbed heading container
      if (
        sibling instanceof HTMLElement &&
        sibling.classList.contains("tinkerdown-tabs")
      ) {
        break;
      }

      // Stop at a regular heading (h1-h6 without tabs)
      if (
        sibling instanceof HTMLElement &&
        /^H[1-6]$/.test(sibling.tagName) &&
        !sibling.classList.contains("tinkerdown-tabs-heading")
      ) {
        break;
      }

      elementsToWrap.push(sibling);
      sibling = sibling.nextSibling;
    }

    // Move elements into the content panel
    elementsToWrap.forEach((el) => {
      contentPanel.appendChild(el);
    });
  }

  /**
   * Handle tab click
   */
  private handleTabClick(groupId: string, tabIndex: number): void {
    const group = this.tabGroups.get(groupId);
    if (!group) return;

    // Update active state
    this.setActiveTab(group, tabIndex);

    // Get the filter expression
    const tab = group.tabs[tabIndex];
    const filter = tab.dataset.filter || "";

    this.log("Tab clicked:", groupId, "index:", tabIndex, "filter:", filter);

    // Find the associated interactive block and send filter action
    this.applyFilter(group, filter);
  }

  /**
   * Handle keyboard navigation for tabs
   */
  private handleTabKeydown(
    e: KeyboardEvent,
    groupId: string,
    currentIndex: number
  ): void {
    const group = this.tabGroups.get(groupId);
    if (!group) return;

    let newIndex = currentIndex;

    switch (e.key) {
      case "ArrowLeft":
        newIndex =
          currentIndex === 0 ? group.tabs.length - 1 : currentIndex - 1;
        break;
      case "ArrowRight":
        newIndex =
          currentIndex === group.tabs.length - 1 ? 0 : currentIndex + 1;
        break;
      case "Home":
        newIndex = 0;
        break;
      case "End":
        newIndex = group.tabs.length - 1;
        break;
      default:
        return;
    }

    e.preventDefault();
    this.setActiveTab(group, newIndex);
    group.tabs[newIndex].focus();

    // Apply filter when navigating with keyboard
    const filter = group.tabs[newIndex].dataset.filter || "";
    this.applyFilter(group, filter);
  }

  /**
   * Set the active tab in a group
   */
  private setActiveTab(group: TabGroup, index: number): void {
    group.tabs.forEach((tab, i) => {
      const isActive = i === index;
      tab.classList.toggle("active", isActive);
      tab.setAttribute("aria-selected", isActive ? "true" : "false");
      tab.setAttribute("tabindex", isActive ? "0" : "-1");
    });

    group.activeIndex = index;

    // Update panel aria-labelledby
    if (group.contentPanel) {
      group.contentPanel.setAttribute(
        "aria-labelledby",
        group.tabs[index].id || ""
      );
    }
  }

  /**
   * Apply a filter to the interactive block associated with this tab group
   */
  private applyFilter(group: TabGroup, filter: string): void {
    // Find the first interactive block within the content panel
    const interactiveBlock = group.contentPanel?.querySelector<HTMLElement>(
      ".tinkerdown-interactive-block"
    );

    if (!interactiveBlock) {
      this.log("No interactive block found for filtering");
      return;
    }

    const blockId = interactiveBlock.dataset.blockId;
    if (!blockId) {
      this.log("Interactive block has no ID");
      return;
    }

    // Send filter action to server
    if (this.sendMessage) {
      this.log("Sending Filter action to block:", blockId, "filter:", filter);
      this.sendMessage(blockId, "Filter", { filter });
    } else {
      this.log("No message sender configured");
    }
  }

  /**
   * Get the current active tab index for a group
   */
  getActiveIndex(groupId: string): number {
    return this.tabGroups.get(groupId)?.activeIndex ?? 0;
  }

  /**
   * Programmatically set the active tab
   */
  setTab(groupId: string, index: number): void {
    const group = this.tabGroups.get(groupId);
    if (!group || index < 0 || index >= group.tabs.length) return;

    this.handleTabClick(groupId, index);
  }
}
