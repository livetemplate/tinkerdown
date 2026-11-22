/**
 * Tests for InteractiveBlock field events (lvt-on:change, lvt-on:input)
 */

import { InteractiveBlock } from "./interactive-block";
import { PersistenceManager } from "../core/persistence-manager";

function makeBlock(html: string) {
  document.body.innerHTML = `<div id="b">${html}</div>`;
  const el = document.getElementById("b")!;
  const block = new InteractiveBlock(
    { id: "lvt-0", element: el, metadata: {} } as any,
    new PersistenceManager() as any
  );
  const sent: Array<[string, any]> = [];
  (block as any).sendMessage = (_id: string, action: string, data: any) => sent.push([action, data]);
  block.initialize();
  return { block, sent };
}

function fire(el: Element, type: string) {
  el.dispatchEvent(new Event(type, { bubbles: true }));
}

describe("InteractiveBlock field events", () => {
  afterEach(() => jest.useRealTimers());

  it("sends a change under the field name and value", () => {
    const { sent } = makeBlock(`<select name="f.level" lvt-on:change="Filter"><option>warn</option></select>`);
    fire(document.querySelector("select")!, "change");
    expect(sent).toEqual([["Filter", { value: "warn", "f.level": "warn" }]]);
  });

  it("ignores fields without an lvt-on attribute for the event", () => {
    const { sent } = makeBlock(`<input name="q" lvt-on:change="Search">`);
    fire(document.querySelector("input")!, "input");
    expect(sent).toEqual([]);
  });

  it("debounces lvt-on:input to the last value", () => {
    jest.useFakeTimers();
    const { sent } = makeBlock(`<input name="f.line" lvt-on:input="Filter" lvt-mod:debounce="100">`);
    const input = document.querySelector("input")!;
    for (const v of ["s", "sp", "spi"]) {
      input.value = v;
      fire(input, "input");
      jest.advanceTimersByTime(50);
    }
    expect(sent).toEqual([]);
    jest.advanceTimersByTime(100);
    expect(sent).toEqual([["Filter", { value: "spi", "f.line": "spi" }]]);
  });

  it("throttles: sends the first event at once and the latest at the end of the interval", () => {
    jest.useFakeTimers();
    const { sent } = makeBlock(`<input name="q" lvt-on:input="Search" lvt-mod:throttle="100">`);
    const input = document.querySelector("input")!;
    input.value = "a";
    fire(input, "input");
    input.value = "ab";
    fire(input, "input");
    input.value = "abc";
    fire(input, "input");
    expect(sent.map(([, d]) => d.q)).toEqual(["a"]);
    jest.advanceTimersByTime(100);
    expect(sent.map(([, d]) => d.q)).toEqual(["a", "abc"]);
  });

  it("destroy cancels a pending debounced send", () => {
    jest.useFakeTimers();
    const { block, sent } = makeBlock(`<input name="q" lvt-on:input="Search" lvt-mod:debounce="100">`);
    fire(document.querySelector("input")!, "input");
    block.destroy();
    jest.advanceTimersByTime(200);
    expect(sent).toEqual([]);
  });
});

describe("InteractiveBlock busy clicks", () => {
  beforeAll(() => {
    // jsdom has no CSS.escape; browsers do.
    if (!(globalThis as any).CSS?.escape) {
      (globalThis as any).CSS = { escape: (s: string) => s.replace(/([^\w-])/g, "\\$1") };
    }
  });
  afterEach(() => jest.useRealTimers());

  // A block whose updates re-render its content from html, as a server
  // update re-creating the row would, and that records refs it sends.
  function makeBusyBlock(html: string) {
    document.body.innerHTML = `<div id="b">${html}</div>`;
    const el = document.getElementById("b")!;
    const block = new InteractiveBlock({ id: "lvt-0", element: el, metadata: {} } as any, new PersistenceManager() as any);
    const sent: Array<{ action: string; data: any; ref?: string }> = [];
    (block as any).sendMessage = (_id: string, action: string, data: any, ref?: string) => sent.push({ action, data, ref });
    block.initialize();
    (block as any).client = {
      updateDOM: (container: HTMLElement) => {
        container.innerHTML = html;
      },
    };
    return { block, sent };
  }
  const button = () => document.querySelector<HTMLButtonElement>('button[name="stop"]')!;

  it("marks a clicked button busy, sends a ref, and ignores clicks until answered", () => {
    const { sent } = makeBusyBlock(`<button name="stop" data-name="web">Stop</button>`);
    button().click();
    button().click();
    expect(sent).toHaveLength(1);
    expect(sent[0]).toMatchObject({ action: "stop", data: { name: "web" } });
    expect(sent[0].ref).toBeTruthy();
    expect(button().getAttribute("aria-busy")).toBe("true");
  });

  it("keeps a re-created button busy through other updates, and clears it on its answer", () => {
    const { block, sent } = makeBusyBlock(`<button name="stop" data-name="web">Stop</button><button name="stop" data-name="db">Stop</button>`);
    button().click();
    const clicked = () => document.querySelector('button[data-name="web"]')!;
    block.handleMessage("tree", {});
    expect(clicked().getAttribute("aria-busy")).toBe("true");
    expect(document.querySelector('button[data-name="db"]')!.hasAttribute("aria-busy")).toBe(false);
    block.handleMessage("tree", {}, undefined, undefined, sent[0].ref);
    expect(clicked().hasAttribute("aria-busy")).toBe(false);
  });

  it("clears a click on an error that carries its ref", () => {
    const { block, sent } = makeBusyBlock(`<button name="stop" data-name="web">Stop</button>`);
    button().click();
    block.handleMessage("error", { message: "unknown block" }, undefined, undefined, sent[0].ref);
    expect(button().hasAttribute("aria-busy")).toBe(false);
  });

  it("fires lvt:success on a form when its own answer arrives, not on another update", () => {
    const { block, sent } = makeBusyBlock(`<form name="save"><input name="q" value="x"><button type="submit">Save</button></form>`);
    const events: string[] = [];
    document.addEventListener("lvt:success", () => events.push("success"));
    document.querySelector("form")!.requestSubmit();
    expect(sent[0]).toMatchObject({ action: "save", data: { q: "x" } });
    block.handleMessage("tree", {});
    expect(events).toEqual([]);
    expect(document.querySelector("form")!.getAttribute("aria-busy")).toBe("true");
    block.handleMessage("tree", {}, undefined, undefined, sent[0].ref);
    expect(events).toEqual(["success"]);
  });

  it("ends pending clicks when the connection closes, or after the timeout", () => {
    jest.useFakeTimers();
    const { block } = makeBusyBlock(`<button name="stop" data-name="web">Stop</button><button name="kill" data-name="web">Kill</button>`);
    button().click();
    block.clearPending();
    expect(button().hasAttribute("aria-busy")).toBe(false);
    const kill = document.querySelector<HTMLButtonElement>('button[name="kill"]')!;
    kill.click();
    jest.advanceTimersByTime(60_000);
    expect(kill.hasAttribute("aria-busy")).toBe(false);
  });
});
