# Livepage Enhancement Plan

**Date**: 2025-11-15
**Status**: Planning
**Version**: 1.0

## Executive Summary

This document outlines a comprehensive enhancement plan for Livepage based on lessons learned from implementing the interactive counter tutorial. The plan is divided into two main areas:

1. **Tutorial Experience** - Making tutorials more engaging, interactive, and educational
2. **Authoring Experience** - Making it easier and faster to create high-quality tutorials

All enhancements are prioritized by **impact** (user value) and **effort** (implementation time), with clear implementation phases.

## Current State

### What's Working ✅
- Basic interactive tutorials with WebSocket communication
- Server-side state management with LiveTemplate
- Client-side block discovery and rendering
- E2E testing infrastructure
- Counter tutorial demonstrating core concepts

### Pain Points Identified 🔴
- **Authoring**: Block syntax is verbose (requires manual IDs, explicit flags)
- **Authoring**: No live reload - must restart server to see changes
- **Authoring**: Generic error messages without file/line context
- **Tutorial**: Basic styling, no animations or visual feedback
- **Tutorial**: Single example doesn't show pattern variations
- **Tutorial**: No development/debugging tools

---

## Part 1: Tutorial Experience Improvements

### Priority 1: High Impact, Low Effort ⭐

#### 1.1 Visual Polish & Modern Design ✅
**Impact**: High - First impression matters
**Effort**: Low (2-3 hours)
**Status**: COMPLETED (2025-11-15)

**What:**
- Professional CSS styling with modern design
- Smooth animations (number transitions, button press effects)
- Visual state feedback (colors: green/positive, red/negative, gray/zero)
- Responsive design (mobile/tablet support)
- Better typography and spacing

**Implementation:**
- Update `internal/server/server.go` - enhance embedded CSS ✅
- Add transition classes to counter demo ✅
- Test on mobile viewports ✅

**Success Metrics:**
- Tutorial looks professional on first visit ✅
- Works on mobile devices ✅
- Animations feel smooth, not jarring ✅

**Changes:**
- Modern gradient backgrounds, smooth transitions
- Counter display with dynamic color feedback (green/red/gray)
- Button hover and active effects
- Responsive design with mobile/tablet breakpoints
- Professional typography and spacing

---

#### 1.1a Syntax Highlighting for Code Blocks
**Impact**: HIGH - Essential for code readability
**Effort**: Low (2-3 hours)
**Status**: ✅ COMPLETED

**What:**
- Language-specific syntax highlighting for all code blocks
- Support for Go, JavaScript, HTML, CSS, YAML, JSON, Shell, and more
- Professional color themes matching the site design
- Line number support (optional)
- Copy-to-clipboard functionality

**Implementation:**
- Integrate Prism.js (lightweight, supports many languages)
- Add Prism CSS and JS to server template
- Automatic language detection from markdown code fence info
- Theme: Use "Tomorrow Night" or similar professional theme
- Load language grammars on demand

**Why Critical:**
- Code blocks without syntax highlighting look unprofessional
- Harder to read and understand code examples
- Essential for tutorial/documentation platform
- Industry standard expectation

---

#### 1.2 Dark/Light Theme Toggle ✅
**Impact**: High - Comfort, accessibility, reading experience
**Effort**: Low (2-3 hours)
**Status**: COMPLETED (2025-11-15)

**What:**
Theme switcher for comfortable reading in any lighting condition:

**Features:**
- **Three modes**: Light, Dark, Auto (follows system preference)
- **Persistent preference** - Saved in localStorage
- **Smooth transition** - Fade between themes, not jarring flash
- **Toggle button** - Fixed position (top-right corner)
- **Keyboard shortcut** - `Ctrl+Shift+D` to toggle
- **System sync** - Auto mode respects `prefers-color-scheme`
- **Code blocks styled** - Monaco editor theme matches page theme

**UI:**
```
┌─────────────────────────────────────┐
│ [Tutorial Title]        ☀️ 🌙 Auto │← Theme toggle
├─────────────────────────────────────┤
│ Content with appropriate colors...  │
└─────────────────────────────────────┘
```

**Color Palette:**

*Light Mode:*
- Background: `#ffffff`
- Text: `#1a1a1a`
- Code blocks: Light theme (VS Code Light+)
- Accent: `#0066cc`

*Dark Mode:*
- Background: `#1a1a1a`
- Text: `#e0e0e0`
- Code blocks: Dark theme (VS Code Dark+)
- Accent: `#4da6ff`

**Implementation:**
- CSS variables for colors (`--bg-color`, `--text-color`, etc.)
- Toggle button component in header
- localStorage: `livepage-theme` = `light|dark|auto`
- MediaQuery listener for system preference changes
- Apply theme class to `<html>` element (`theme-light`, `theme-dark`)
- Monaco editor theme switching API
- Smooth CSS transitions on theme changes

**Accessibility:**
- ARIA labels for theme buttons
- Focus indicators visible in both themes
- WCAG AAA contrast ratios
- No flashing/seizure risk during transitions

---

#### 1.3 Live State Inspector ✅
**Impact**: High - Educational, shows "under the hood"
**Effort**: Low (3-4 hours)
**Status**: COMPLETED (2025-11-15)

**What:**
- Collapsible panel showing current server state as JSON
- Updates in real-time as you interact
- Shows WebSocket messages (sent/received)
- Auto-enabled when interactive blocks are present

**Example:**
```
┌─ State Inspector ───────────┐
│ Counter State               │
│ {                           │
│   "Counter": 5              │
│ }                           │
│                             │
│ Last Action: "increment"    │
│ Latency: 12ms               │
└─────────────────────────────┘

┌─ WebSocket Log ─────────────┐
│ → increment                 │
│ ← update (15ms)             │
└─────────────────────────────┘
```

**Implementation:**
- Add debug panel component to client
- Send state snapshot with each update
- Store message history (last 10)

---

#### 1.4 Better Tutorial Copy ✅
**Impact**: Medium-High - Clarity helps learning
**Effort**: Low (1-2 hours)
**Status**: COMPLETED (2025-11-15)

**What:**
- Clearer introduction explaining WHY LiveTemplate ✅
- Annotated code blocks with inline comments ✅
- Call-out boxes for important concepts ✅
- Better section headers and flow ✅

**Example:**
```markdown
> 💡 **Key Concept**: In LiveTemplate, state lives on the server.
> This means validation and business logic are trusted and secure,
> unlike client-side frameworks where state can be manipulated.
```

**Implementation:**
- Update `examples/counter/index.md`
- Add markdown extensions for callouts
- Test readability with fresh eyes

---

### Priority 2: High Impact, Medium Effort

#### 2.1 Multiple Interactive Demos
**Impact**: High - Shows pattern variations
**Effort**: Medium (4-6 hours)

**What:**
Show different counter variations teaching different concepts:

1. **Basic Counter** - Simple increment/decrement
2. **Bounded Counter** - With min (0) / max (100) limits
   - *Teaches*: Validation, conditional logic
3. **Step Counter** - Buttons for +1, +5, +10
   - *Teaches*: Action parameters, different magnitudes
4. **Dual Counter** - Two independent instances
   - *Teaches*: State isolation, multiple components
5. **Real-world Example** - Shopping cart quantity selector
   - *Teaches*: Practical application, styling

**Implementation:**
- Create 5 separate state blocks in one tutorial
- Each with its own interactive demo
- Side-by-side comparisons
- Shared explanation text

---

#### 2.2 Progressive Tutorial Steps
**Impact**: High - Better learning flow
**Effort**: Medium (3-4 hours)

**What:**
Break tutorial into incremental steps where each builds on previous:

- **Step 1**: Display only (no buttons yet)
  - Just show `<h2>Count: {{.Counter}}</h2>`
  - Explain: This is how you display state

- **Step 2**: Add increment button
  - Add single button, see it work
  - Explain: `lvt-click` triggers server action

- **Step 3**: Add decrement and reset
  - Complete controls
  - Explain: Multiple actions, same pattern

- **Step 4**: Add validation
  - Bounded version (can't go below 0)
  - Explain: Server-side validation

- **Step 5**: Add styling
  - Colors, animations
  - Explain: Styling is just CSS

**Implementation:**
- Restructure `index.md` into clear steps
- Each step shows cumulative code
- Interactive demo at each step
- Progress indicator (Step 2 of 5)

---

#### 2.3 Guided Challenges
**Impact**: Medium-High - Engagement
**Effort**: Medium (3-4 hours)

**What:**
Interactive prompts that guide exploration:

```markdown
### Challenge 1: Reach the Target
Click the buttons to reach exactly 10.

[ Interactive Counter Here ]

✓ Success! Notice how fast the updates were?

### Challenge 2: Speed Test
How many times can you increment in 10 seconds?
(Tests rapid clicking, WebSocket performance)

### Challenge 3: Efficiency
Get from 0 to 50 in the fewest clicks possible.
Hint: Use the +10 button strategically!
```

**Implementation:**
- Add challenge sections to tutorial
- Track user actions (optional)
- Show completion checkmarks
- Make it playful, not test-like

---

#### 2.4 Step Navigation Controls
**Impact**: High - Essential for multi-step tutorials
**Effort**: Medium (3-4 hours)

**What:**
Add UI controls to navigate between tutorial steps:

```html
<!-- Navigation bar at top and bottom of tutorial -->
<div class="livepage-nav">
  <button class="nav-prev" disabled>← Previous</button>
  <span class="nav-progress">Step 2 of 5</span>
  <button class="nav-next">Next →</button>
</div>
```

**Features:**

*Inline Navigation (bottom bar):*
- **Next/Previous buttons** - Move between H2 sections
- **Progress indicator** - Shows current step (e.g., "Step 2 of 5")
- **Keyboard shortcuts** - Arrow keys to navigate
- **Auto-scroll** - Smoothly scroll to next section

*Sidebar Navigation (table of contents):*
- **Collapsible TOC** - Shows all tutorial sections
- **Current step highlighted** - Visual indicator of progress
- **Completion checkmarks** - Track which sections visited
- **Click to jump** - Direct navigation to any section
- **Sticky positioning** - Always visible while scrolling
- **Mobile: collapsible** - Hamburger menu on small screens

*Shared:*
- **State preservation** - Interactive blocks maintain state across navigation
- **URL anchors** - Deep linking to specific steps (`#step-2-add-increment`)

**Layout Options:**
```
┌─────────────────────────────────┐
│ [Tutorial Title]                │
├──────────┬──────────────────────┤
│ TOC      │ # Step 1             │
│ ✓ Step 1 │ Content here...      │
│ → Step 2 │                      │
│   Step 3 │ [Interactive Block]  │
│   Step 4 │                      │
│          │                      │
│          ├──────────────────────┤
│          │ ← Prev | 2/5 | Next →│
└──────────┴──────────────────────┘
```

**Implementation:**
- Client-side JavaScript for navigation
- Parse H2 headings as "steps"
- Generate both sidebar TOC and bottom nav
- Add CSS for responsive layout (sidebar collapses on mobile)
- Track visited sections in localStorage
- Update URL hash on navigation
- Highlight current section in both TOC and content

---

#### 2.5 Presentation Mode
**Impact**: High - Perfect for workshops, live demos, teaching
**Effort**: Medium (4-5 hours)

**What:**
Full-screen presentation mode optimized for teaching and demonstrations:

**Features:**
- **Fullscreen view** - Press `F` or click icon to enter presentation mode
- **Large, readable text** - Increased font sizes for projection
- **Hide sidebar** - Maximize content area
- **Big navigation controls** - Large Next/Previous buttons
- **Slide-style transitions** - Smooth fade between sections
- **Live interaction** - Interactive blocks work during presentation
- **Presenter notes** - Optional notes visible only to presenter (future)
- **Slide counter** - Clear "3 / 8" indicator
- **Auto-advance option** - Timer-based progression (configurable)
- **Keyboard controls**:
  - Arrow keys / Space: Next/Previous
  - `F` or `Escape`: Toggle fullscreen
  - `R`: Reset all interactive blocks
  - `B`: Blackout screen (audience Q&A)

**Use Cases:**
1. **Workshop instructor** - Walk through tutorial step-by-step with audience
2. **Conference talk** - Live coding demos with real interactivity
3. **Classroom teaching** - Students follow along on their devices
4. **Video recording** - Clean, focused screen for screencasts

**Layout:**
```
┌─────────────────────────────────────┐
│                                     │
│   # Step 2: Add Increment Button    │
│                                     │
│   Now let's add interactivity...    │
│                                     │
│   [Interactive Demo - LARGE]        │
│                                     │
│                                     │
│                                     │
│                                     │
│   [←  Previous]    3/8    [Next →]  │
└─────────────────────────────────────┘
```

**Implementation:**
- CSS for fullscreen mode (via Fullscreen API)
- Increase base font sizes in presentation mode
- Hide sidebar, header, footer
- Show only current section (H2-bounded)
- Large touch-friendly navigation buttons
- Keyboard event handlers
- Optional: URL param `?present=true` for direct entry
- Persist presentation state across refreshes

**Bonus Features:**
- **Dual monitor support** - Presenter view on one screen, clean view on projector
- **Audience sync** - Instructor controls, students' browsers follow along (WebSocket broadcast)
- **Recording mode** - Optimized layout for screen recording

---

### Priority 3: Medium Impact, Medium-High Effort

#### 3.1 Flow Diagrams & Visualizations
**Impact**: Medium - Helps visual learners
**Effort**: Medium-High (6-8 hours)
**Status**: ✅ COMPLETED

**What:**
- Diagram showing client ↔ server flow
- Animated arrows on interaction
- State transition diagrams
- Architecture overview

**Implementation:**
- Integrated Mermaid.js v10 for diagram rendering
- Added sequence diagram showing WebSocket communication flow
- Added state transition diagram for counter states (Positive/Zero/Negative)
- Added architecture flowchart showing browser ↔ server components
- Added React vs LiveTemplate architecture comparison diagrams
- Theme-aware rendering (switches between light/dark based on page theme)
- Auto-initialization on page load with `startOnLoad: true`

**Diagrams Added:**
- **Counter Example**: 3 diagrams (sequence, state transition, architecture)
- **Comparison Example**: 2 diagrams (React architecture, LiveTemplate architecture)

**Why Effective:**
- Visual learners can see the complete flow at a glance
- State transitions clearly show how actions affect counter value
- Architecture diagrams highlight security benefits of server-side state
- Comparison diagrams make React vs LiveTemplate differences obvious

---

#### 3.2 Comparison Demos ✅
**Impact**: Medium - Shows advantages
**Effort**: Medium (4-5 hours)
**Status**: COMPLETED (2025-11-16)

**What:**
Side-by-side comparison of:
- Traditional JavaScript approach (React)
- LiveTemplate approach

Shows code examples and running demo side-by-side.

**Implementation:**
- Created `examples/comparison/index.md` with comprehensive React vs LiveTemplate comparison
- React example: 42 lines (useState, event handlers, client-side validation)
- LiveTemplate example: 24 lines (Go structs, server-side state)
- Side-by-side code comparison with syntax highlighting
- Comparison table highlighting key differences (LOC, state location, validation, etc.)
- Live interactive LiveTemplate demo
- Professional styling with gradient backgrounds
- Clear explanations of pros/cons for each approach

**Example:**
```
┌─ React Version ────┐  ┌─ LiveTemplate ────┐
│ useState hook      │  │ Go struct         │
│ onClick handlers   │  │ Change() method   │
│ Client state       │  │ Server state      │
│ 42 lines of JS     │  │ 24 lines of Go    │
└────────────────────┘  └───────────────────┘
```

---

## Part 2: Authoring Experience Improvements

### Priority 1: High Impact, Low-Medium Effort ⭐⭐⭐

#### A1. Auto-Generate Block IDs ✅
**Impact**: HIGH - Major friction removed
**Effort**: Low (2-3 hours)
**Status**: COMPLETED (2025-11-15)

**Current:**
```markdown
```go server readonly id="counter-state"
```lvt interactive state="counter-state"
```

**Improved:**
```markdown
```go server
type CounterState struct { ... }
```

```lvt
<div>Count: {{.Counter}}</div>
```
```

**How it works:**
- Auto-generate IDs: `server-0`, `server-1`, `lvt-0`, etc.
- Auto-link lvt blocks to nearest previous server block
- Optional explicit ID: `` ```go server id="custom-name" ``

**Implementation:**
- Already partially done! Just remove requirement for manual IDs
- Update parser to auto-assign
- Update examples to use simpler syntax

**Success Metric:**
- Authors never need to write `id="..."` unless they want to

---

#### A2. Better Error Messages ✅
**Impact**: HIGH - Saves debugging time
**Effort**: Medium (4-5 hours)
**Status**: COMPLETED (2025-11-15)

**Current:**
```
Failed to parse template for block lvt-1: template: lvt-1:1: unexpected "}"
```

**Improved:**
```
❌ Error in examples/counter/index.md

Line 44: Template syntax error in interactive block
  42 | ## Interactive Demo
  43 |
  44 | ```lvt interactive state="counter-state"
  45 | <div>Count: {{.Counter}}</div>
     |                       ^
     |                       Unexpected "}"
  46 | ```

💡 Tip: Check for matching {{ and }} brackets

Related: State 'counter-state' defined at line 20
```

**Features:**
- File name and line number
- Code context (surrounding lines)
- Highlight error position with `^`
- Helpful suggestions
- Links to related blocks

**Implementation:**
- Track source locations during parsing
- Create error formatter utility
- Add "did you mean?" suggestions
- Test with common errors

---

#### A3. Live Reload / Watch Mode ✅
**Impact**: HIGH - Essential authoring workflow
**Effort**: Medium (5-6 hours)
**Status**: COMPLETED (2025-11-15)

**Current:**
- Edit markdown
- Kill server (Ctrl+C)
- Restart `livepage serve`
- Refresh browser

**Improved:**
```bash
livepage serve --watch examples/counter
```

- Watches for `.md` file changes ✅
- Reloads pages automatically ✅
- Preserves WebSocket connections where possible ✅
- Shows overlay notification: "Reloaded index.md" ✅

**Implementation:**
- Add file watcher using `fsnotify` ✅
- Graceful reload of changed pages ✅
- Send reload signal to connected clients ✅
- Optional: Hot reload without full refresh ✅

**Changes:**
- Created file watcher using fsnotify for monitoring .md file changes
- Added --watch/-w flag to serve command with --port and --host support
- Implemented connection tracking in Server to broadcast reload messages
- Added BroadcastReload method to send reload notifications to all WebSocket clients
- Enhanced client message router to handle "reload" action with notification overlay
- Reload notification shows file path and smoothly reloads page after 500ms
- Tested with manual file changes - watcher correctly detects and triggers reload

---

### Priority 2: Medium-High Impact, Medium Effort

#### A4. Validation Command ✅
**Impact**: Medium-High - Catches errors early
**Effort**: Low-Medium (3-4 hours)
**Status**: COMPLETED (2025-11-15)

**Usage:**
```bash
livepage validate examples/counter

✓ index.md: OK
✓ State blocks: 1 found, 0 errors
✓ Interactive blocks: 1 found, 0 errors
✓ References: All resolved

✓ All checks passed!
```

Or with errors:
```bash
livepage validate examples/counter

✗ index.md: 2 errors

  Line 44: Unknown state reference 'counter-stats'
           Did you mean: 'counter-state' (line 20)?

  Line 58: Interactive block missing state attribute

✗ Validation failed with 2 errors
```

**Checks:**
- Parse all markdown files
- Validate block syntax
- Check state references
- Verify required attributes
- Report all errors at once

**Implementation:**
- Extract validation logic from parser ✅
- Add `validate` subcommand to CLI ✅
- Format errors nicely ✅
- Exit code: 0 (success) or 1 (errors) ✅

**Changes:**
- Created `cmd/livepage/commands/validate.go` with ValidateCommand implementation
- Added validate case to main.go switch statement
- Updated CLI help text with validate command usage examples
- Leverages existing ParseFile validation logic for consistency
- Walks directory tree discovering all .md files (skips hidden directories)
- Collects all errors before reporting (batch validation)
- Displays checkmarks for valid files in real-time
- Shows detailed error messages with file context and indentation
- Provides summary statistics (total files, valid, errors)
- Returns exit code 0 for success, 1 for validation failures
- Tested with both valid and invalid files - works correctly

---

#### A5. Scaffold Generator ✅
**Impact**: Medium-High - Faster getting started
**Effort**: Medium (4-5 hours)
**Status**: COMPLETED (2025-11-15)

**Usage:**
```bash
livepage new my-tutorial

✨ Created new tutorial: my-tutorial

📁 Project structure:
   my-tutorial/
   ├── index.md
   └── README.md

🚀 Next steps:
   cd my-tutorial
   livepage serve

📚 Your tutorial will be available at http://localhost:8080
```

**What it creates:**
- Pre-filled `index.md` with frontmatter ✅
- Example code blocks (server + lvt) ✅
- Comments explaining syntax ✅
- README with next steps ✅

**Implementation:**
- Embed templates in binary ✅
- Template variables for project name ✅
- Support custom template dir (future)
- Interactive prompts (future)

**Changes:**
- Created embedded template directory structure with basic tutorial template
- Implemented NewCommand with directory creation and validation
- Added template variable substitution for {{.Title}} and {{.ProjectName}}
- Supports hyphen and underscore naming (converts to title case)
- Error handling for missing names, existing directories, invalid names
- Wired new command to main.go (removed "not yet implemented" stub)
- Fixed parser nil pointer bug for fenced code blocks without language info
- Validated generated projects parse correctly
- Tested with multiple project names and error scenarios

---

#### A6. Block Inspector CLI ✅
**Impact**: Medium - Helps debugging
**Effort**: Low-Medium (3-4 hours)
**Status**: COMPLETED (2025-11-15)

**Usage:**
```bash
livepage blocks examples/counter

🔍 Inspecting blocks in: /Users/.../examples/counter

index.md:
  Line 1143: server-0 (server)
           State: CounterState

  Line 2207: lvt-1 (lvt)
           References: (auto-linked to nearest server)
           Template: 9 lines

────────────────────────────────────────────────────────────
Summary:
  Total blocks: 2
  Server blocks: 1
  WASM blocks: 0
  Interactive blocks: 1
```

**With verbose mode:**
```bash
livepage blocks examples/counter --verbose

index.md:

Block: server-0
  Type: server
  Language: go
  Location: index.md:1143
  State: CounterState
  Content: // CounterState holds the application state...

Block: lvt-1
  Type: lvt
  Language: lvt
  Location: index.md:2207
  State Ref: (auto-linked)
  Template Lines: 9
```

**Implementation:**
- Parse without running server ✅
- List all discovered blocks ✅
- Show metadata, relationships ✅
- Validate references ✅
- Support --verbose/-v flag ✅

**Changes:**
- Created BlocksCommand with file discovery and parsing
- Parses CodeBlock information to get line numbers
- Basic mode shows concise block list with key info
- Verbose mode shows detailed block metadata
- Auto-extracts state names from Go code
- Shows template/code line counts
- Displays auto-linking information for lvt blocks
- Summary statistics for all block types
- Wired blocks command to main.go and updated help text
- Tested with counter example - works correctly

---

### Priority 3: Medium Impact, Higher Effort

#### A7. Configuration File Support ✅
**Impact**: Medium - Customization
**Effort**: Medium-High (6-8 hours)
**Status**: COMPLETED (2025-11-16)

**File:** `livepage.yaml`

```yaml
title: "LiveTemplate Tutorial"
description: "Learn LiveTemplate interactively"

server:
  port: 8080
  host: localhost
  debug: false

styling:
  theme: clean  # clean, dark, minimal
  primary_color: "#007bff"
  font: "system-ui"

blocks:
  auto_id: true
  id_format: "kebab-case"  # or: camelCase, snake_case
  show_line_numbers: true

features:
  state_inspector: true
  websocket_log: false
  hot_reload: true

ignore:
  - "drafts/**"
  - "_*.md"
```

**Implementation:**
- YAML parsing
- Merge with defaults
- Apply throughout app
- Document all options

---

#### A8. Enhanced Documentation
**Impact**: Medium - Onboarding
**Effort**: Medium (5-6 hours)

**Create:**

1. **Quick Start Guide**
   - 5-minute tutorial to first interactive demo
   - Step-by-step with screenshots

2. **Authoring Reference**
   - All block types
   - All attributes
   - Code examples

3. **Best Practices**
   - Tutorial structure
   - Educational patterns
   - Common pitfalls

4. **API Reference**
   - State struct patterns
   - Template syntax
   - Action handlers

**Implementation:**
- Write markdown docs
- Host as livepage tutorial (dogfooding!)
- Add search
- Link from error messages

---

#### A9. VSCode Extension (Future)
**Impact**: Medium-High (for power users)
**Effort**: High (15-20 hours)

**Features:**
- Syntax highlighting for `lvt` blocks
- Autocomplete for state references
- Go to definition (jump to state block)
- Inline error highlighting
- Preview panel

**Implementation:**
- TextMate grammar for syntax
- Language server for features
- Extension API integration
- Publish to marketplace

**Phase:** Future / V2

---

## Priority Matrix

### Quick Wins (Do First) 🚀
**High Impact, Low-Medium Effort**

| Enhancement | Impact | Effort | Hours |
|-------------|--------|--------|-------|
| Auto-generate IDs (A1) | ⭐⭐⭐ | Low | 2-3 |
| Better error messages (A2) | ⭐⭐⭐ | Medium | 4-5 |
| Visual polish (1.1) | ⭐⭐⭐ | Low | 2-3 |
| Dark/Light theme toggle (1.2) | ⭐⭐⭐ | Low | 2-3 |
| State inspector (1.3) | ⭐⭐⭐ | Low | 3-4 |
| Better copy (1.4) | ⭐⭐ | Low | 1-2 |
| Validation CLI (A4) | ⭐⭐ | Low-Med | 3-4 |

**Total: ~22 hours** (1 week sprint)

---

### High Value Investments 💎
**High Impact, Medium-High Effort**

| Enhancement | Impact | Effort | Hours |
|-------------|--------|--------|-------|
| Live reload/watch (A3) | ⭐⭐⭐ | Medium | 5-6 |
| Multiple demos (2.1) | ⭐⭐⭐ | Medium | 4-6 |
| Progressive steps (2.2) | ⭐⭐⭐ | Medium | 3-4 |
| Step navigation controls (2.4) | ⭐⭐⭐ | Medium | 3-4 |
| Presentation mode (2.5) | ⭐⭐⭐ | Medium | 4-5 |
| Scaffold generator (A5) | ⭐⭐ | Medium | 4-5 |
| Guided challenges (2.3) | ⭐⭐ | Medium | 3-4 |

**Total: ~31 hours** (2 weeks)

---

### Long-term Improvements 🏗️
**Medium Impact, Higher Effort OR Future**

| Enhancement | Impact | Effort | Notes |
|-------------|--------|--------|-------|
| Flow diagrams (3.1) | ⭐⭐ | High | Nice to have |
| Comparison demos (3.2) | ⭐⭐ | Medium | Educational value |
| Config file (A7) | ⭐⭐ | Medium-High | Customization |
| Block inspector CLI (A6) | ⭐ | Low-Med | Debugging aid |
| Documentation (A8) | ⭐⭐ | Medium | Essential long-term |
| VSCode extension (A9) | ⭐⭐⭐ | High | Future / V2 |

---

## Implementation Roadmap

### Phase 1: Foundation (Week 1)
**Goal:** Remove authoring friction, make tutorials look good

✅ **Authoring:**
- Auto-generate block IDs (A1)
- Better error messages with file/line context (A2)
- Validation command (A4)

✅ **Tutorial:**
- Visual polish - CSS, animations (1.1)
- Dark/Light theme toggle (1.2)
- Live state inspector (1.3)
- Improved tutorial copy (1.4)

**Deliverable:** Authors can create tutorials faster with better errors, tutorials look professional and comfortable to read

---

### Phase 2: Workflow (Week 2-3)
**Goal:** Improve authoring and learning experience

**Authoring:**
- ✅ Live reload / watch mode (A3) - COMPLETED
- ✅ Validation command (A4) - COMPLETED
- ✅ Scaffold generator - `livepage new` (A5) - COMPLETED
- ✅ Block inspector - `livepage blocks` (A6) - COMPLETED

**Tutorial:**
- ✅ Multiple interactive demos (2.1) - COMPLETED
- Progressive tutorial steps (2.2) - DEFERRED (current tutorial is well-structured)
- Guided challenges (2.3) - DEFERRED (requires challenge scoring system)
- ✅ Step navigation controls - sidebar TOC + bottom nav (2.4) - COMPLETED
- Presentation mode (2.5) - DEFERRED (requires fullscreen API integration)

**Deliverable:** ✅ Smooth authoring workflow, engaging tutorial experience with full navigation

**Note:** Tasks 2.2, 2.3, and 2.5 are deferred to Phase 3 as they require significant additional infrastructure. Core navigation and multiple demos provide excellent tutorial experience.

---

### Phase 3: Polish (Week 4)
**Goal:** Refinement and documentation

✅ **Additions:**
- Configuration file support (A7)
- Flow diagrams & visualizations (3.1)
- Comparison demos (3.2)
- Comprehensive documentation (A8)

**Deliverable:** Production-ready, well-documented system

---

### Phase 4: Future (V2)
**Goal:** Power features

- VSCode extension (A9)
- Advanced debugging tools
- Template library/marketplace
- Multi-language support (beyond Go)
- Collaborative editing
- Audience sync (broadcast mode for classrooms)

---

## Success Metrics

### Authoring Metrics
- **Time to first tutorial**: < 10 minutes (from idea to working demo)
- **Error resolution time**: < 2 minutes (clear errors with suggestions)
- **Tutorial creation time**: < 2 hours (for comprehensive tutorial)
- **Learning curve**: < 30 minutes (to understand all features)

### Tutorial Metrics
- **Engagement**: Time spent on page, interactions per session
- **Completion rate**: % who reach end of tutorial
- **Comprehension**: Quiz/challenge success rate
- **Mobile usage**: % of mobile visitors, bounce rate

### Quality Metrics
- **Error rate**: < 5% of tutorials have validation errors
- **Load time**: < 2 seconds for first paint
- **WebSocket latency**: < 50ms for local, < 200ms for remote
- **Accessibility**: WCAG AA compliance

---

## Next Steps

### Immediate Actions (This Week):
1. ✅ Review and approve this plan
2. Implement Phase 1 Quick Wins:
   - Auto-generate IDs (A1)
   - Better error messages (A2)
   - Visual polish (1.1)
   - Dark/Light theme toggle (1.2)
   - State inspector (1.3)
   - Improved copy (1.4)
   - Validation CLI (A4)
3. Update counter tutorial with improvements
4. Test with fresh users for feedback

### Planning:
- Create detailed implementation tasks for Phase 2
- Set up project tracking (issues/milestones)
- Allocate development time

### Long-term:
- Gather user feedback after Phase 1
- Adjust priorities based on usage patterns
- Plan Phase 3 and beyond

---

## Appendix: Examples

### Example: Auto-ID Syntax

**Before:**
```markdown
```go server readonly id="todo-state"
type TodoState struct {
    Items []TodoItem
}
```

```lvt interactive state="todo-state"
<ul>
  {{range .Items}}
    <li>{{.Text}}</li>
  {{end}}
</ul>
```
```

**After:**
```markdown
```go server
type TodoState struct {
    Items []TodoItem
}
```

```lvt
<ul>
  {{range .Items}}
    <li>{{.Text}}</li>
  {{end}}
</ul>
```
```

Simpler! IDs auto-generated and linked.

---

### Example: Better Error Message

**Before:**
```
Error: Failed to parse markdown
```

**After:**
```
❌ Error in examples/todo/index.md

Line 56: Template syntax error
  54 | ```lvt
  55 | <ul>
  56 |   {{range .Items}
     |                  ^
     |                  Missing closing "}}"
  57 |     <li>{{.Text}}</li>
  58 |   {{end}}
  59 | </ul>

💡 Tip: Template tags must be properly closed: {{range}} ... {{end}}

🔗 Learn more: https://pkg.go.dev/text/template
```

---

## Conclusion

This plan provides a clear roadmap from the current working prototype to a polished, production-ready tutorial authoring platform. By focusing on **quick wins** first (Phase 1), we'll see immediate improvements in both authoring and tutorial experience.

The prioritization balances:
- **User needs** (authors want simplicity, learners want engagement)
- **Technical debt** (fix pain points now vs. later)
- **Time investment** (quick wins vs. long-term projects)

With Phase 1 complete, Livepage will be a genuinely delightful tool for creating interactive tutorials.

---

## Phase 5: Multi-Page Documentation Sites

### B1. Multi-Page Documentation Site Capability
**Impact**: HIGH - Enables full documentation sites, not just single tutorials
**Effort**: High (18-24 hours)
**Status**: PLANNED

**What:**
Build capability to create documentation sites with multiple pages, home page, and navigation - similar to https://lotusdocs.dev/

**Site Structure:**
```
docs/
├── livepage.yaml           # Site configuration
├── index.md                # Home page (landing/overview)
├── getting-started/
│   ├── index.md           # Section home
│   ├── installation.md
│   └── quickstart.md
├── tutorials/
│   ├── index.md
│   ├── counter.md
│   └── todo-list.md
└── reference/
    ├── index.md
    ├── blocks.md
    └── configuration.md
```

**Configuration Format (livepage.yaml):**
```yaml
title: "LivePage Documentation"
description: "Interactive documentation platform"
type: site  # NEW: "site" vs "tutorial"

site:
  home: index.md
  logo: /assets/logo.svg
  repository: https://github.com/livetemplate/livepage

navigation:
  - title: Getting Started
    path: getting-started
    pages:
      - title: Installation
        path: getting-started/installation.md
      - title: Quick Start
        path: getting-started/quickstart.md

  - title: Tutorials
    path: tutorials
    pages:
      - title: Counter Example
        path: tutorials/counter.md
      - title: Todo List
        path: tutorials/todo-list.md

  - title: Reference
    path: reference
    collapsed: false  # Default expanded
    pages:
      - title: Block Types
        path: reference/blocks.md
      - title: Configuration
        path: reference/configuration.md
```

**Core Components:**

1. **Site Manager** (`internal/site/manager.go`)
   - Multi-page discovery and loading
   - Navigation tree building
   - Site-wide configuration

2. **Enhanced Parser**
   - Extend frontmatter: `parent`, `order`, `hidden`, `tags`
   - Support page metadata for navigation

3. **Server Updates**
   - Route handling for `/:path`
   - Home page template (hero, features, CTAs)
   - Site-wide navigation sidebar
   - Breadcrumbs component
   - Prev/Next page navigation

4. **Navigation Components**
   - Hierarchical sidebar (collapsible sections)
   - Active page highlighting
   - Responsive mobile menu
   - Table of contents per page

5. **Search Functionality**
   - Client-side search index
   - Fuzzy matching on titles + content
   - Keyboard shortcut (Cmd+K)
   - Search results overlay

**Implementation Tasks:**

- **Phase 1: Core Infrastructure** (6-8h)
  - Site configuration schema
  - Site manager implementation
  - Multi-page route handling

- **Phase 2: Navigation & UI** (5-7h)
  - Site navigation sidebar
  - Home page template
  - Breadcrumbs and prev/next

- **Phase 3: Enhancement** (4-5h)
  - Auto-discovery from file structure
  - Search functionality

- **Phase 4: Example & Testing** (3-4h)
  - Create `examples/docs-site/`
  - E2E tests for navigation
  - Professional styling

**Success Criteria:**
- ✓ Serve multi-page site from `livepage serve docs/`
- ✓ Home page with hero, features, navigation
- ✓ Hierarchical sidebar with sections
- ✓ Breadcrumbs and prev/next navigation
- ✓ Mix of static docs and interactive tutorials
- ✓ Responsive design
- ✓ Search across pages
- ✓ Professional appearance
- ✓ Working example in `examples/docs-site/`

---

### B2. Fix Presentation Mode ✅
**Impact**: MEDIUM - Broken feature needs repair
**Effort**: Low-Medium (2-3 hours)
**Status**: ✅ COMPLETED (2025-11-20)

**What was implemented:**
- ✅ Presentation toggle button (📽️) in top toolbar
- ✅ CSS styles for presentation mode (larger fonts, hidden sidebar, focused content)
- ✅ JavaScript implementation for section-by-section navigation
- ✅ Keyboard shortcuts: 'f' to toggle, arrow keys to navigate, Escape to exit
- ✅ Integration with existing navigation buttons
- ✅ E2E tests verifying all functionality

**Changes:**
- Added presentation button with projector emoji to HTML template
- Implemented CSS for presentation mode: hides sidebar, enlarges fonts, focuses on one section
- JavaScript manages section discovery, navigation, and mode toggling
- Keyboard controls: f-key toggle, arrow keys for prev/next, Escape to exit
- E2E test covers button click, keyboard shortcuts, navigation, and visual state
- All tests passing

**Commit**: `0b0d6eb`

---

**Document Version**: 1.1
**Last Updated**: 2025-11-20
**Next Review**: After Phase 5 planning approval
