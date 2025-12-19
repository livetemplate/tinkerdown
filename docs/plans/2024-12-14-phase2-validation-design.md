# Phase 2: Validation Design

## Summary

This document defines the implementation plan for Phase 2 of the LivePage PMF strategy. The goal is to validate that LLMs can reliably generate working LivePage applications by creating comprehensive documentation and a "Copy-Paste App Store" of 10 example prompts.

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Primary Goal | Both user prompts + system prompt | User-facing prompts for adoption, system prompt for Claude Code integration |
| App Types | Breadth-first (10 diverse) | Demonstrates versatility, appeals to wider audience |
| Validation | Golden file testing | Automated verification that examples compile and run |
| Format | Agent Skills + LLMS.txt | Skills for Claude Code, LLMS.txt for broader compatibility |

## Deliverables

### 1. Agent Skill for Claude Code

Create a `livepage` skill following Anthropic's Agent Skills format that both Anthropic and OpenAI are adopting.

**Directory structure:**
```
skills/livepage/
├── SKILL.md           # Entry point with name, description, triggers
├── reference.md       # API reference (components, sources, lvt-* attributes)
├── examples/
│   ├── 01-todo-app.md
│   ├── 02-dashboard.md
│   ├── 03-contact-form.md
│   ├── 04-blog.md
│   ├── 05-inventory.md
│   ├── 06-survey.md
│   ├── 07-booking.md
│   ├── 08-expense-tracker.md
│   ├── 09-faq.md
│   └── 10-status-page.md
└── scripts/
    └── validate.sh     # Golden file validation script
```

### 2. LLMS.txt Reference

Create `docs/llms.txt` following the emerging LLMS.txt standard for AI-readable documentation.

### 3. Golden File Tests

Each example in `examples/` serves as a golden file:
- Examples are complete, runnable LivePage apps
- Validation script compiles each example and verifies it serves HTTP
- CI runs validation on every commit

## Implementation Plan

### Task 1: Create Skill Directory Structure

```
skills/livepage/
├── SKILL.md
├── reference.md
└── examples/
```

### Task 2: Write SKILL.md

Entry point that Claude Code loads when skill is triggered.

```markdown
---
name: livepage
description: Build single-file web apps with LivePage
triggers:
  - livepage
  - single-file app
  - markdown app
  - no-build app
---

# LivePage Skill

LivePage is an AI app builder that outputs working apps in a single markdown file with no build step.

## When to Use

Use LivePage when building:
- Internal tools
- Admin dashboards
- Data viewers
- Simple CRUD apps
- Prototypes

## Quick Start

1. Create a `.md` file with frontmatter and server block
2. Run `livepage serve myapp.md`
3. Open in browser

## Reference

See [reference.md](./reference.md) for full API documentation.

## Examples

See [examples/](./examples/) for complete working apps.
```

### Task 3: Write reference.md

Comprehensive API reference covering:

1. **File Structure** - Frontmatter, Server Block, View
2. **Server Block (Go)** - Controller, State, methods
3. **View (HTML)** - Template syntax, binding
4. **lvt-* Attributes** - click, submit, input, change, source, persist, data-*
5. **Components** - datatable, dropdown, modal, etc.
6. **Sources** - pg, rest, csv, json, exec

### Task 4: Create 10 Example Apps (Breadth-First)

| # | App | Features Demonstrated |
|---|-----|----------------------|
| 1 | Todo App | lvt-persist, lvt-submit, lvt-click, basic CRUD |
| 2 | Dashboard | lvt-source (REST), datatable component, charts |
| 3 | Contact Form | lvt-submit, validation, email notification |
| 4 | Blog | Multiple pages, partials, markdown rendering |
| 5 | Inventory | PostgreSQL source, search/filter, pagination |
| 6 | Survey | Multi-step form, progress tracking, results view |
| 7 | Booking | Calendar, time slots, conflict detection |
| 8 | Expense Tracker | File upload (CSV), calculations, reports |
| 9 | FAQ | Accordion component, search, categories |
| 10 | Status Page | Real-time updates, exec source, system checks |

### Task 5: Write LLMS.txt

Create `docs/llms.txt` for broader AI tool compatibility:

```
# LivePage - One-File AI App Builder

> Build working web apps in a single markdown file. No React. No build step.

## Quick Reference

### File Structure
...

### Key Attributes
- lvt-click: Trigger server action on click
- lvt-submit: Handle form submission
- lvt-source: Connect to data source
- lvt-persist: Auto-save to SQLite
...
```

### Task 6: Create Validation Script

`skills/livepage/scripts/validate.sh`:
- Iterate through each example
- Run `livepage serve` in background
- Verify HTTP 200 response
- Check for JavaScript errors (optional: headless browser)
- Report pass/fail

### Task 7: Add Golden File Test

`validation_test.go`:
- Go test that runs validation script
- Part of CI pipeline
- Ensures examples stay up-to-date with API changes

## Progress Tracker

| Task | Status | Notes |
|------|--------|-------|
| Create skill directory | TODO | |
| Write SKILL.md | TODO | |
| Write reference.md | TODO | |
| Example 1: Todo App | TODO | |
| Example 2: Dashboard | TODO | |
| Example 3: Contact Form | TODO | |
| Example 4: Blog | TODO | |
| Example 5: Inventory | TODO | |
| Example 6: Survey | TODO | |
| Example 7: Booking | TODO | |
| Example 8: Expense Tracker | TODO | |
| Example 9: FAQ | TODO | |
| Example 10: Status Page | TODO | |
| Write LLMS.txt | TODO | |
| Create validation script | TODO | |
| Add golden file test | TODO | |

## Success Criteria

1. Claude Code can use the livepage skill to generate valid apps
2. All 10 examples compile and serve without errors
3. Golden file tests pass in CI
4. LLMS.txt provides sufficient context for other AI tools

## Notes

- Examples should be minimal but complete (demonstrate one pattern well)
- Reference doc should be scannable (tables, code blocks, clear headings)
- Validation should catch regressions early
