/**
 * Tests for TabsController
 */

import { TabsController } from "./tabs";

describe("TabsController", () => {
  let container: HTMLElement;

  beforeEach(() => {
    // Set up DOM
    document.body.innerHTML = "";
    container = document.createElement("div");
    document.body.appendChild(container);
  });

  afterEach(() => {
    document.body.innerHTML = "";
  });

  describe("initialization", () => {
    it("should initialize without errors when no tabs exist", () => {
      const controller = new TabsController();
      expect(controller).toBeDefined();
    });

    it("should discover tab groups in the DOM", () => {
      container.innerHTML = `
        <div class="tinkerdown-tabs" data-tabs-id="test-tabs">
          <h2 class="tinkerdown-tabs-heading">
            <span class="tinkerdown-tabs-bar" role="tablist">
              <button class="tinkerdown-tab active" data-tab-index="0" data-filter="">All</button>
              <button class="tinkerdown-tab" data-tab-index="1" data-filter="done">Done</button>
            </span>
          </h2>
          <div class="tinkerdown-tabs-content" data-tabs-content></div>
        </div>
      `;

      const controller = new TabsController();
      expect(controller.getActiveIndex("test-tabs")).toBe(0);
    });
  });

  describe("tab switching", () => {
    beforeEach(() => {
      container.innerHTML = `
        <div class="tinkerdown-tabs" data-tabs-id="switch-tabs">
          <h2 class="tinkerdown-tabs-heading">
            <span class="tinkerdown-tabs-bar" role="tablist">
              <button class="tinkerdown-tab active" data-tab-index="0" data-filter="" id="tab-0">All</button>
              <button class="tinkerdown-tab" data-tab-index="1" data-filter="active" id="tab-1">Active</button>
              <button class="tinkerdown-tab" data-tab-index="2" data-filter="done" id="tab-2">Done</button>
            </span>
          </h2>
          <div class="tinkerdown-tabs-content" data-tabs-content></div>
        </div>
      `;
    });

    it("should switch tabs programmatically", () => {
      const controller = new TabsController();

      controller.setTab("switch-tabs", 1);
      expect(controller.getActiveIndex("switch-tabs")).toBe(1);

      const tab1 = document.getElementById("tab-1");
      expect(tab1?.classList.contains("active")).toBe(true);
      expect(tab1?.getAttribute("aria-selected")).toBe("true");
    });

    it("should update aria attributes on tab switch", () => {
      const controller = new TabsController();

      controller.setTab("switch-tabs", 2);

      const tab0 = document.getElementById("tab-0");
      const tab2 = document.getElementById("tab-2");

      expect(tab0?.getAttribute("aria-selected")).toBe("false");
      expect(tab0?.getAttribute("tabindex")).toBe("-1");
      expect(tab2?.getAttribute("aria-selected")).toBe("true");
      expect(tab2?.getAttribute("tabindex")).toBe("0");
    });

    it("should ignore invalid tab index", () => {
      const controller = new TabsController();

      controller.setTab("switch-tabs", 99);
      expect(controller.getActiveIndex("switch-tabs")).toBe(0);
    });

    it("should ignore invalid group ID", () => {
      const controller = new TabsController();

      // Should not throw
      controller.setTab("nonexistent", 0);
      expect(controller.getActiveIndex("nonexistent")).toBe(0);
    });
  });

  describe("message sending", () => {
    it("should call message sender with filter data on tab click", () => {
      container.innerHTML = `
        <div class="tinkerdown-tabs" data-tabs-id="msg-tabs">
          <h2 class="tinkerdown-tabs-heading">
            <span class="tinkerdown-tabs-bar" role="tablist">
              <button class="tinkerdown-tab active" data-tab-index="0" data-filter="">All</button>
              <button class="tinkerdown-tab" data-tab-index="1" data-filter="status = active">Active</button>
            </span>
          </h2>
          <div class="tinkerdown-tabs-content" data-tabs-content>
            <div class="tinkerdown-interactive-block" data-block-id="test-block"></div>
          </div>
        </div>
      `;

      const mockSender = jest.fn();
      const controller = new TabsController();
      controller.setMessageSender(mockSender);

      // Click the second tab
      const tab1 = container.querySelector('[data-tab-index="1"]') as HTMLButtonElement;
      tab1.click();

      expect(mockSender).toHaveBeenCalledWith("test-block", "Filter", { filter: "status = active" });
    });
  });
});
