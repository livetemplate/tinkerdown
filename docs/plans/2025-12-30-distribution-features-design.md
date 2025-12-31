# Distribution Features Design

**Date:** 2025-12-30
**Status:** Proposal
**Goal:** Enable non-technical users to use tinkerdown apps and easy sharing

---

## Context

Tinkerdown's vision is **LLM-generated throwaway apps for personal productivity**. To reach this vision, two features are critical:

1. **View-Only Mode** - Non-technical users can use apps without seeing markdown
2. **Build Command** - Export to standalone HTML for easy sharing

Together, these enable the viral loop:
```
LLM generates → Developer runs → Exports HTML → Shares with team → Team uses (click-ops)
```

---

## Feature 1: View-Only Mode

### Problem

Currently, `tinkerdown serve` shows the rendered markdown page. Users see:
- Markdown headings
- Code fence syntax (```lvt)
- Raw markdown formatting

Non-technical users don't want to see this. They want a clean app interface.

### Solution

Add a `--view-only` flag that:
1. Hides all markdown source formatting
2. Shows only rendered interactive components
3. Presents a clean "app-like" interface

### UX Modes

| Mode | Command | Who Uses It | What They See |
|------|---------|-------------|---------------|
| **Edit** (default) | `tinkerdown serve` | Developer | Full markdown + rendered UI |
| **View-Only** | `tinkerdown serve --view-only` | End user | Only rendered UI |
| **Hybrid** | `tinkerdown serve --toggle` | Both | Toggle button to switch |

### Implementation Approach

**Option A: CSS-based hiding**
- Add a CSS class that hides markdown elements
- Toggle via flag or runtime button
- Pros: Simple, no parsing changes
- Cons: Markdown still in DOM (view-source reveals it)

**Option B: Render-time filtering**
- Only render `lvt` code blocks and headings
- Skip all other markdown content
- Pros: Clean output, no hidden content
- Cons: Loses documentation/context

**Option C: Layout transformation**
- Convert markdown sections to UI sections
- Use headings as section labels
- Render lvt blocks as components
- Pros: Best of both worlds
- Cons: More complex

**Recommendation:** Start with **Option A** (CSS-based), evolve to **Option C**.

### View-Only UI Elements

```
┌─────────────────────────────────────────┐
│ [App Title]                    [⚙️ Edit] │
├─────────────────────────────────────────┤
│                                         │
│  ┌─────────────────────────────────┐   │
│  │ [Form: Add Item]                │   │
│  │ [Input] [Input] [Submit]        │   │
│  └─────────────────────────────────┘   │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │ [Table: Items]                  │   │
│  │ col1 | col2 | actions           │   │
│  │ ...  | ...  | [×]               │   │
│  └─────────────────────────────────┘   │
│                                         │
└─────────────────────────────────────────┘
```

### CLI Interface

```bash
# Default: edit mode (current behavior)
tinkerdown serve app.md

# View-only mode
tinkerdown serve app.md --view-only

# View-only with edit toggle button
tinkerdown serve app.md --view-only --allow-edit

# Alias for non-technical users
tinkerdown run app.md  # Implies --view-only
```

### Config Option

```yaml
# tinkerdown.yaml
server:
  default_mode: view-only  # or "edit"
  allow_mode_toggle: true
```

### Implementation Tasks

| Task | Effort | Priority |
|------|--------|----------|
| Add `--view-only` flag to CLI | Small | P0 |
| Add CSS class to hide markdown elements | Small | P0 |
| Render only lvt blocks in view-only mode | Medium | P1 |
| Add toggle button to switch modes | Small | P1 |
| Add `tinkerdown run` alias | Small | P2 |
| Transform layout for app-like feel | Medium | P2 |

---

## Feature 2: Build Command (Export to HTML)

### Problem

To share a tinkerdown app, recipients need:
1. tinkerdown installed
2. The markdown file
3. Any data files

This is too much friction for casual sharing.

### Solution

`tinkerdown build` exports a self-contained HTML file that:
1. Includes all assets (CSS, JS)
2. Embeds initial data
3. Works offline (for read-only apps)
4. Optionally includes a mini-server for read-write apps

### Export Modes

| Mode | Use Case | Output | Data Handling |
|------|----------|--------|---------------|
| **Static** | Read-only dashboards | Single HTML | Data embedded |
| **Interactive** | Forms + local storage | HTML + JS | Browser localStorage |
| **Full** | Complete app | HTML + embedded server | Requires Go runtime |

### CLI Interface

```bash
# Static export (read-only)
tinkerdown build app.md -o app.html

# Interactive export (localStorage for writes)
tinkerdown build app.md -o app.html --interactive

# Full export (embedded server binary)
tinkerdown build app.md -o app --full

# Specify mode explicitly
tinkerdown build app.md -o app.html --mode=static|interactive|full
```

### Static Export Details

**What's included:**
- Rendered HTML (view-only mode)
- All CSS (inlined)
- LiveTemplate JS runtime (inlined)
- Initial data (JSON embedded in script tag)

**What's NOT included:**
- WebSocket connection (no live updates)
- Server-side actions (no form submissions)
- Data persistence

**Use cases:**
- Share a snapshot of data
- Demo/preview of an app
- Documentation with embedded examples

### Interactive Export Details

**What's included:**
- Everything from static export
- localStorage adapter for data persistence
- Form submission handling (writes to localStorage)
- Delete/update actions

**What's NOT included:**
- Server-side logic
- External data sources (REST, pg, exec)
- Real-time sync

**Use cases:**
- Personal tools on a single device
- Offline-capable apps
- Share app template (recipient adds own data)

### Full Export Details

**What's included:**
- Complete tinkerdown server compiled in
- Markdown file embedded
- All data sources functional
- Single executable binary

**Output:**
```bash
$ tinkerdown build expenses.md -o expenses --full
$ ./expenses
Server running at http://localhost:8080
```

**Use cases:**
- Distribute complete app to non-technical users
- "Double-click to run" experience
- No Go/tinkerdown installation needed

### Data Embedding

For static/interactive exports, data is embedded:

```html
<script id="tinkerdown-data" type="application/json">
{
  "sources": {
    "items": [
      {"id": "item_001", "title": "Example", "done": false}
    ]
  }
}
</script>
```

The JS runtime reads this on page load.

### Implementation Tasks

| Task | Effort | Priority |
|------|--------|----------|
| Add `build` command skeleton | Small | P0 |
| Implement static HTML export | Medium | P0 |
| Inline CSS/JS assets | Small | P0 |
| Embed source data as JSON | Medium | P0 |
| LocalStorage adapter for writes | Medium | P1 |
| Form handling without server | Medium | P1 |
| Full binary compilation | Large | P2 |

---

## Feature 3: Share Command (Future)

### Concept

Quick sharing via hosted service or gist:

```bash
$ tinkerdown share app.md
Uploading...
→ https://run.tinkerdown.dev/abc123
→ Expires in 7 days

$ tinkerdown share app.md --gist
→ https://gist.github.com/user/abc123
→ Run: tinkerdown run https://gist.github.com/user/abc123
```

### Deferred

This requires infrastructure (hosted service). Defer until build/view-only are validated.

---

## Implementation Roadmap

### Phase 1: Minimum Viable Distribution (2 weeks)

| Week | Feature | Deliverable |
|------|---------|-------------|
| 1 | View-only mode | `--view-only` flag with CSS hiding |
| 1 | Build static | `tinkerdown build` outputs HTML |
| 2 | Interactive build | localStorage persistence |
| 2 | Polish | Error handling, documentation |

### Phase 2: Full App Distribution (2 weeks)

| Week | Feature | Deliverable |
|------|---------|-------------|
| 3 | Binary compilation | `--full` builds standalone executable |
| 3 | Toggle mode | Runtime switch between view/edit |
| 4 | Theming | Clean default theme for view-only |
| 4 | Documentation | User guide for sharing apps |

### Phase 3: Cloud Distribution (Future)

| Feature | Deliverable |
|---------|-------------|
| Hosted playground | Web-based tinkerdown runner |
| Share command | Upload to hosted service |
| Gist integration | Run apps from gist URLs |

---

## Success Metrics

| Metric | Target | How to Measure |
|--------|--------|----------------|
| View-only adoption | 30% of serves use --view-only | CLI telemetry (opt-in) |
| Build usage | 100 exports/month | CLI telemetry (opt-in) |
| Share conversion | 10% of viewers install tinkerdown | Referral tracking |

---

## Open Questions

1. **Should view-only be the default?**
   - Pro: Better for end users
   - Con: Developers expect to see markdown
   - Proposal: Default to edit, but `tinkerdown run` implies view-only

2. **How to handle data sources in build?**
   - Static data: Embed snapshot
   - REST APIs: Fetch at build time, embed result
   - exec/pg: Require --full mode or error

3. **LocalStorage limits?**
   - 5-10MB typical limit
   - Show warning if data exceeds threshold
   - Suggest --full mode for large apps

4. **Styling in view-only mode?**
   - Need a clean, app-like default theme
   - Consider Tailwind-style utility classes
   - Or: Very minimal styling, let users customize

---

## Appendix: User Journeys

### Journey 1: Developer shares with team

```
1. Developer asks Claude: "build expense tracker"
2. Claude outputs tinkerdown markdown
3. Developer: tinkerdown serve expenses.md (tests it)
4. Developer: tinkerdown build expenses.md -o expenses.html --interactive
5. Developer: emails expenses.html to team
6. Team member: opens in browser, uses app
7. Data saved in team member's localStorage
```

### Journey 2: Developer ships tool to client

```
1. Developer builds custom tool for client
2. Developer: tinkerdown build tool.md -o tool --full
3. Developer: sends `tool` executable to client
4. Client: double-clicks tool, browser opens
5. Client: uses fully functional app, no install needed
```

### Journey 3: Quick demo

```
1. Developer: tinkerdown share demo.md
2. Gets URL: https://run.tinkerdown.dev/abc123
3. Shares URL in Slack
4. Colleague clicks, sees running app
5. If they want to modify: downloads .md, runs locally
```
