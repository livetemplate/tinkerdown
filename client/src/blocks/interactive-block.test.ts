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
