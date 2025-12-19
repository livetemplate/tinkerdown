# LivePage PMF: The One-File AI App Builder

## Executive Summary

LivePage is positioned to be the **only AI app builder that outputs working apps in a single markdown file with no build step**. While competitors like v0, Bolt.new, and Lovable generate complex React projects requiring npm/build pipelines, LivePage targets the 700M "code-generators" who want: **Type → Works**.

**Tagline**: LivePage: AI builds tools in one file. No React. No build step. Just run.

---

## Market Analysis

### The Competitive Landscape (2025)

| Tool | ARR/Funding | What It Does | Limitation |
|------|-------------|--------------|------------|
| **v0 by Vercel** | Vercel ($150M+) | React + Tailwind from prompts | Outputs React, needs build step |
| **Bolt.new** | $40M ARR | Full-stack Next.js scaffolding | Token-based, complex output |
| **Lovable** | Well-funded | React + Supabase MVPs | Supabase-locked, React only |
| **Replit Agent** | $70M ARR | Full cloud IDE + AI | $25/mo, browser-based |
| **Dyad** | Open source | Local v0/Bolt alternative | Still generates React |

### Key Insight: They All Generate Complex Output

Every single player generates:

- React/Next.js (multiple files)
- Requires npm/build step
- Needs hosting/deployment
- Complex to debug

**None of them** generate a simple, single-file, no-build format.

### The Gap

```
What Exists:  Prompt → AI → [React App (10+ files)] → npm install → build → deploy → works
What's Missing: Prompt → AI → [Single markdown file] → run → works
```

### LivePage's Unique Position

| Factor | v0/Bolt/Lovable | LivePage |
|--------|-----------------|----------|
| Output format | React (10+ files) | Single markdown file |
| Build step | Required (npm) | None |
| Hosting | Required | Single binary |
| Runtime | Node.js | Go (fast) |
| Open source | Mostly no | Yes |
| Self-hosted | Complex | Easy |
| LLM complexity | High | Low |

---

## The Declarative Trifecta

Why LivePage wins for LLMs:

| Layer | What It Does | LLM Benefit |
|-------|--------------|-------------|
| **Components** | Pre-built UI (datatable, dropdown, modal) | LLM doesn't generate complex HTML/CSS |
| **Sources** | Data connectors (pg, rest, csv, exec) | LLM doesn't write SQL/API boilerplate |
| **lvt-persist** | Auto CRUD | LLM doesn't write backend code |

**Result:**

```
LivePage:    Prompt → LLM → [Component + Source declaration] → Works
Competitors: Prompt → LLM → [React + API routes + SQL + CSS] → npm install → Build → Deploy → Works
```

---

## Implementation Plan

### Phase 1: Core Infrastructure (Enable LLM Generation)

#### 1.1 `lvt-data-*` Attribute Support (HIGH PRIORITY)

**Problem**: Delete buttons require wrapper forms. LLMs generate `<button lvt-data-id="1">` and expect it to work.

**Solution**: Pass data attributes with click actions.

```html
<button lvt-click="delete" lvt-data-id="{{.ID}}">Delete</button>
```

**Implementation**:

| File | Change | Status |
|------|--------|--------|
| `client/src/blocks/interactive-block.ts` | `extractLvtData()` method already exists at line 113-126 | ✅ Done |
| `client/src/blocks/interactive-block.ts` | `handleClick()` already calls `extractLvtData()` at line 96-106 | ✅ Done |
| `livetemplate/action.go:120-125` | `GetIntOk()` only handles `float64`, needs to parse strings | ⏳ TODO |
| `autopersist_e2e_test.go` | E2E test with delete button using `lvt-data-id` | ⏳ TODO |

**Effort**: 1-2 hours

#### 1.2 Component Library Integration

**Problem**: LLMs must generate 100+ lines of HTML/CSS for common UI patterns.

**Solution**: Integrate existing `livetemplate/components` library.

**Available Components** (already built in `/Users/adnaan/code/livetemplate/components`):

| Category | Components |
|----------|------------|
| Form Controls | dropdown, autocomplete, datepicker, timepicker, tagsinput, toggle, rating |
| Layout | tabs, accordion, modal, drawer |
| Feedback | toast, tooltip, popover, progress, skeleton |
| Data Display | datatable, timeline, breadcrumbs |
| Navigation | menu |

**Usage Pattern**:

```html
{{template "lvt:datatable:default:v1" .Users}}
{{template "lvt:dropdown:searchable:v1" .CountrySelect}}
{{template "lvt:modal:confirm:v1" .DeleteConfirm}}
```

**Implementation**:

| Task | Effort |
|------|--------|
| Register component templates in LivePage template engine | Small |
| Add component CSS to default styles | Small |
| Wire component actions to server state | Medium |
| Document component usage for LLMs | Small |

**Requirements**:

1. Components must look "production-ready" (Tailwind-like polish) out of the box
2. **Smart Defaults**: `<div lvt-source="users">` should auto-render a table if no inner template provided

#### 1.3 `lvt-source` Plugin System

**Problem**: `lvt-persist` only creates local SQLite tables. Users need to connect to existing data.

**Solution**: Extensible plugin system with four configuration modes.

**⚠️ SECURITY RULE: Never allow SQL/queries in HTML attributes.**

All queries MUST be defined server-side in `livepage.yaml`. HTML only references sources by name.

##### Mode 1: HTML Attributes (Simplest - No Go Template Knowledge)

Pure HTML with `lvt-*` attributes. No Go template syntax required. Queries defined in YAML.

```yaml
# livepage.yaml
sources:
  users:
    type: pg
    query: SELECT * FROM users WHERE active = true
```

```html
<!-- Clean HTML - no SQL, no Go templates -->
<div lvt-source="users"
     lvt-data-columns="name:Name,email:Email,role:Role"
     lvt-data-page-size="10">
</div>
```

##### Mode 2: YAML Config + Simple Reference

Define everything in YAML, reference by name in HTML with minimal Go template for iteration.

```yaml
# livepage.yaml
sources:
  users:
    type: pg
    query: SELECT * FROM users

  github-prs:
    type: rest
    url: https://api.github.com/repos/myorg/myrepo/pulls
    headers:
      Authorization: Bearer ${GITHUB_TOKEN}
```

```html
<table lvt-source="users">
  {{range .}}
    <tr><td>{{.name}}</td><td>{{.email}}</td></tr>
  {{end}}
</table>
```

##### Mode 3: Template with Dict (Power Users)

For users comfortable with Go templates. Full control over component configuration.

```html
{{template "lvt:datatable:default:v1" dict
    "source" "users"
    "columns" "name:Name,email:Email,role:Role"
    "page-size" "10"
}}
```

##### Mode 4: Exec/Stdout (Polyglot - Any Language)

Use Python, Bash, Node, or any language as a data source. Write a script that outputs JSON.

```yaml
# livepage.yaml
sources:
  sales:
    type: exec
    cmd: python scripts/fetch_sales.py
    args: ["--format", "json"]
    interval: 60s
```

```html
<table lvt-source="sales">
  {{range .}}
    <tr><td>{{.product}}</td><td>${{.revenue}}</td></tr>
  {{end}}
</table>
```

**Why Mode 4 is critical**: Python/Bash/Node developers can write a 10-line script to fetch data and use LivePage just for UI. They don't need to learn Go.

**Marketing angle**: "Build a UI for your Python/Bash scripts in 30 seconds."

##### Plugin Interface

```go
type Source interface {
    Name() string
    Configure(config map[string]string) error
    Query(ctx context.Context, attrs map[string]string) ([]map[string]any, error)
    Watch(ctx context.Context, attrs map[string]string) (<-chan []map[string]any, error)
    SupportsWatch() bool
}
```

##### Priority Sources

| Tier | Source | Why | Effort |
|------|--------|-----|--------|
| 1 | **exec/stdout** | Polyglot - any language works | Medium |
| 1 | **PostgreSQL** | Most common production DB | Medium |
| 1 | **REST API** | Universal connector to any SaaS | Medium |
| 1 | **CSV/JSON** | Zero-config, spreadsheets → dashboards | Small |
| 2 | Stripe, GitHub, Slack | Common SaaS integrations | Small each |

##### Real-Time Updates

LivePage already has WebSocket infrastructure. Extend for sources:

- **PostgreSQL**: `LISTEN/NOTIFY` - push when data changes
- **REST API**: Webhooks or polling with `lvt-data-interval`
- **CSV**: File watcher (fsnotify) - push when file changes
- **Exec**: Re-run on interval or file change

#### 1.4 Seamless Partials

**Problem**: Single-file apps become unmanageable at 500+ lines.

**Solution**: Allow `{{template "sidebar.md"}}` without a build step.

```html
{{template "components/sidebar.md"}}
{{template "components/header.md"}}
```

It stays "no build", just "read file at runtime".

---

### Phase 2: Prove the Format Works

| Task | Description |
|------|-------------|
| LLM Testing | Test if Claude/GPT-4 can reliably generate LivePage markdown |
| Reference Doc | Create LLM reference doc (components + sources + lvt-* attributes) |
| Prompt Library | Create 10 example prompts → working outputs ("Copy-Paste App Store") |

---

### Phase 3: Launch

| Task | Description |
|------|-------------|
| Landing Page | "AI builds tools in one file" |
| Playground | Paste AI output, see it run |
| Positioning | "Internal Tool Killer" vs Retool ($50/user) |

---

### Phase 4: Distribution

| Task | Description |
|------|-------------|
| Show HN | "Show HN: I made an AI app builder that outputs one markdown file" |
| Open Source | Apache 2.0 license |
| VS Code Extension | Real-time preview (type on left, see app on right) |

---

## Future Features (High-Impact Ideas)

### 1. Declarative Authentication (`<lvt-auth>`)

```html
<lvt-auth provider="google" allowed-domains="company.com">
  <!-- Protected content here -->
</lvt-auth>
```

**Impact**: Critical for enterprise adoption. Removes biggest barrier to deploying internal tools safely.

### 2. Compile to Binary (`livepage build`)

```bash
livepage build my-app.md -o my-app
```

Send `my-app.exe` to a Product Manager. They double-click, it opens in browser. No installation.

**Impact**: High. Enables distribution to non-developers.

### 3. Headless Mode (Auto-Generated API)

```
func (s *State) Refund(ctx Context) → POST /api/action/Refund
```

Build UI, get API as side effect. Trigger from Slack bot or cron job.

**Impact**: Medium. Adds backend versatility.

### 4. AI-Native Components (`lvt-ai`)

```html
<div lvt-ai="summarize" lvt-data-input="{{.Ticket.Description}}">
  <!-- AI-generated summary appears here -->
</div>
```

**Impact**: High. Unique differentiator vs Retool.

### 5. LivePage Hub

```bash
livepage run github.com/user/repo/postgres-admin.md
```

Decentralized package manager. Developers maintain "dotfiles" repo of favorite tools.

**Impact**: High. Drives viral growth through network effect.

### Impact Summary

| Idea | Target Audience | Impact |
|------|-----------------|--------|
| Declarative Auth | Corp/Enterprise | Critical for team adoption |
| Compile to Binary | Tool Builders | High - distribution to non-devs |
| Headless API | Backend Devs | Medium - adds versatility |
| AI Components | AI Engineers | High - unique differentiator |
| Hub/Registry | Open Source | High - drives viral growth |

---

## Risks & Mitigations

### Risk 1: Complexity Ceiling

Single-file apps become unmanageable at 500+ lines.

**Mitigations**:

1. **Seamless Partials**: `{{template "sidebar.md"}}` without build step
2. **Eject Strategy**: Document how to convert LivePage to standard Go web server

### Risk 2: Security Perception

SQL in Markdown scares security-conscious developers.

**Mitigations**:

1. **Strict Separation**: Server Block (Go/SQL) vs View (HTML). SQL never goes to client.
2. **Env Var First**: Enforce `${DATABASE_URL}` pattern. No hardcoded credentials.

---

## Go-to-Market Strategy

### 1. The "Prompt Library" IS the Product

Don't just launch the tool; launch a **"Copy-Paste App Store"**:

- "Need a Postgres Admin?" → Copy prompt → Paste into Claude → Paste result into `admin.md` → Run
- Proves value instantly

### 2. VS Code Extension (Essential)

Real-time preview like Markdown preview. Type on left, see app update on right. **Magical DX.**

### 3. "Internal Tool Killer" Positioning

- **Competitor**: Retool ($50/user, proprietary)
- **LivePage**: Free, local, text-based (git-friendly)
- **Wedge**: "Don't pay $50/user for Retool. Just write a LivePage."

---

## Progress Tracker

| Task | Status | Notes |
|------|--------|-------|
| **Phase 1.1: lvt-data-*** | | |
| Client-side extraction | ✅ Done | Already in interactive-block.ts |
| Server-side string parsing | ⏳ TODO | action.go GetIntOk() |
| E2E test | ⏳ TODO | autopersist delete with lvt-data-id |
| **Phase 1.2: Components** | | |
| Register templates | ⏳ TODO | |
| Add component CSS | ⏳ TODO | Must look production-ready |
| Smart defaults | ⏳ TODO | Auto-render table |
| **Phase 1.3: lvt-source** | | |
| Plugin interface | ⏳ TODO | |
| exec/stdout source | ⏳ TODO | Polyglot - priority |
| PostgreSQL source | ⏳ TODO | |
| REST API source | ⏳ TODO | |
| CSV/JSON source | ⏳ TODO | |
| **Phase 1.4: Partials** | | |
| Seamless partials | ⏳ TODO | `{{template "file.md"}}` |
| Eject documentation | ⏳ TODO | |
| **Phase 2: Validation** | | |
| LLM testing | ⏳ TODO | |
| Reference doc | ⏳ TODO | |
| Prompt library (10) | ⏳ TODO | Copy-Paste App Store |

---

## Sources

- [v0/Bolt/Lovable/Replit Comparison](https://flatlogic.com/blog/lovable-vs-bolt-vs-replit-which-ai-app-coding-tool-is-best/)
- [Dyad - Open Source Alternative](https://www.dyad.sh/)
- [v0 Platform API](https://vercel.com/blog/build-your-own-ai-app-builder-with-the-v0-platform-api)
- [AI App Builders 2025](https://reflex.dev/blog/2025-05-16-top-5-ai-app-builders/)
- [Bolt.new $40M ARR](https://sacra.com/c/vercel/)
- [Retool Integrations](https://retool.com/integrations)
- [Appsmith Alternatives](https://www.appsmith.com/blog/retool-alternatives)
