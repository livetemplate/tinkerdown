# Web Components Support for LivePage

## Summary

Add zero-config Web Components support to LivePage using Shoelace as the built-in default library. Authors simply use `sl-*` elements in their markdown, and LivePage handles detection, loading, and server-state integration automatically.

## Progress Tracker

- [ ] Phase 1: Shoelace detection + CDN injection (server-side)
- [ ] Phase 2: `lvt-action` attribute handling (client-side)
- [ ] Phase 3: Smart event mapping for Shoelace components
- [ ] Phase 4: E2E tests with chromedp

## Design Decisions

| Decision | Choice |
|----------|--------|
| Primary use case | Third-party component integration |
| Default library | Shoelace (sl-* prefix) |
| Detection | Auto-detect from HTML, no config needed |
| State model | Component owns UI state, server owns business state |
| Event handling | `lvt-action` with smart component→event mapping |
| Activation | Auto-enabled when sl-* elements detected |

## Author Experience

```markdown
# My Tutorial

```go server id=counter
type State struct { Count int; Name string }

func (s *State) Increment(ctx *livetemplate.Context) error {
    s.Count++
    return nil
}

func (s *State) SetName(ctx *livetemplate.Context) error {
    s.Name = ctx.Data["value"].(string)
    return nil
}
```

<sl-button lvt-action="increment" variant="primary">
  Count: {{.Count}}
</sl-button>

<sl-input lvt-action="setName" placeholder="Your name" value="{{.Name}}"></sl-input>
```

No configuration required. LivePage:
1. Detects `sl-button` and `sl-input`
2. Injects Shoelace CDN into page head
3. Binds `lvt-action` to appropriate Shoelace events

## Architecture

### Two-Layer State Model

```
┌─────────────────────────────────────────────────────────────┐
│                    Server (Go)                               │
│  ┌─────────────────────────────────────────────────────────┐│
│  │  Business State: Count=5, Name="Alice"                  ││
│  └─────────────────────────────────────────────────────────┘│
│              │ template render              ▲                │
│              ▼                              │ action         │
└─────────────────────────────────────────────────────────────┘
               │ WebSocket                    │
               ▼                              │
┌─────────────────────────────────────────────────────────────┐
│                    Client (Browser)                          │
│  ┌─────────────────────────────────────────────────────────┐│
│  │  <sl-input value="Alice">                               ││
│  │    └─ UI State: focused, cursor position, etc.          ││
│  └─────────────────────────────────────────────────────────┘│
│         │ user types, blurs                 │                │
│         └───────────────────────────────────┘                │
│           sl-change event → lvt-action="setName"             │
└─────────────────────────────────────────────────────────────┘
```

**Key principle**: Web Components handle UI micro-interactions locally (instant). Only meaningful state changes sync to server.

### Smart Event Mapping

```typescript
const SHOELACE_EVENT_MAP = {
  'sl-input':    { event: 'sl-change', value: 'value' },
  'sl-textarea': { event: 'sl-change', value: 'value' },
  'sl-select':   { event: 'sl-change', value: 'value' },
  'sl-checkbox': { event: 'sl-change', value: 'checked' },
  'sl-switch':   { event: 'sl-change', value: 'checked' },
  'sl-button':   { event: 'click',     value: null },
  'sl-dialog':   { event: 'sl-request-close', value: null },
  'sl-tab-group':{ event: 'sl-tab-show', value: 'detail.name' },
  'sl-rating':   { event: 'sl-change', value: 'value' },
  'sl-range':    { event: 'sl-change', value: 'value' },
  'sl-color-picker': { event: 'sl-change', value: 'value' },
};
```

Override when needed: `<sl-input lvt-action="draft" lvt-event="sl-input">`

## Implementation Plan

### Phase 1: Shoelace Detection + CDN Injection

**Files to modify:**

1. **`parser.go`** - Add custom element detection during markdown parsing
   - Scan HTML content for elements matching `sl-*` pattern
   - Store detected components in `Page` struct

2. **`page.go`** - Track Shoelace usage per page
   - Add `UsesShoelace bool` field to Page struct
   - Add `ShoelaceVersion string` field (default: "2")

3. **`internal/server/server.go`** - Inject CDN links
   - If page uses Shoelace, inject into `<head>`:
     ```html
     <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@shoelace-style/shoelace@2/cdn/themes/light.css">
     <script type="module" src="https://cdn.jsdelivr.net/npm/@shoelace-style/shoelace@2/cdn/shoelace-autoloader.js"></script>
     ```

**New file:**

4. **`internal/components/shoelace.go`**
   - `DetectShoelaceElements(html string) []string` - regex/parser for sl-* tags
   - `GetCDNURLs(version string) (cssURL, jsURL string)`
   - Constants for default version, CDN base URL

### Phase 2: lvt-action Attribute Handling

**Files to modify:**

1. **`client/src/blocks/interactive-block.ts`**
   - Add handler for `lvt-action` attribute
   - Detect element tag name to determine event type
   - Extract value from appropriate property
   - Send action via existing WebSocket infrastructure

2. **`client/src/types.ts`**
   - Add `ShoelaceEventMapping` type
   - Add `LvtActionConfig` interface

**New file:**

3. **`client/src/components/shoelace-events.ts`**
   - Export `SHOELACE_EVENT_MAP` constant
   - Export `getEventConfig(tagName: string)` helper
   - Export `extractValue(element: Element, config: EventConfig)` helper

### Phase 3: Smart Event Mapping

**Enhance `interactive-block.ts`:**

```typescript
private setupLvtAction(element: HTMLElement): void {
  const action = element.getAttribute('lvt-action');
  if (!action) return;

  const tagName = element.tagName.toLowerCase();
  const config = getEventConfig(tagName);
  const eventOverride = element.getAttribute('lvt-event');
  const eventName = eventOverride || config.event;

  element.addEventListener(eventName, (e) => {
    const value = extractValue(element, config, e);
    this.sendAction(action, { value });
  });
}
```

**Handle existing lvt-* attributes:**
- Ensure backwards compatibility with `lvt-click`, `lvt-change`, `lvt-submit`
- `lvt-action` is additive, not replacing existing attributes

### Phase 4: E2E Tests

**New file: `webcomponents_e2e_test.go`**

Tests to implement:
1. Shoelace CDN injection when sl-* elements present
2. No injection when no sl-* elements
3. `lvt-action` on sl-button triggers server action
4. `lvt-action` on sl-input sends value on change
5. `lvt-action` on sl-select sends selected value
6. `lvt-action` on sl-checkbox sends checked state
7. Server state updates re-render Shoelace components
8. `lvt-event` override works correctly

**Test example structure:**
```
examples/webcomponents-test/
├── index.md          # Test page with various Shoelace components
└── expected.html     # Expected rendered output (optional)
```

## Files Summary

### Modified
- `parser.go` - Custom element detection
- `page.go` - Shoelace tracking fields
- `internal/server/server.go` - CDN injection
- `client/src/blocks/interactive-block.ts` - lvt-action handling
- `client/src/types.ts` - New types

### New
- `internal/components/shoelace.go` - Shoelace utilities
- `client/src/components/shoelace-events.ts` - Event mapping
- `webcomponents_e2e_test.go` - E2E tests
- `examples/webcomponents-test/index.md` - Test fixture

## Optional Future Enhancements (not in scope)

- `livepage.yaml` config for additional component libraries
- Version pinning for Shoelace
- Local component support with `app-*` prefix
- Dark mode theme switching
- Shoelace form integration with auto-persist
