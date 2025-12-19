# LivePage PMF Implementation Plan

## Summary

This plan implements the "One-File AI App Builder" vision. See `docs/pmf-one-file-ai-builder.md` for full strategy document.

## Documentation Fixes Required

~~Before implementation, fix `docs/pmf-one-file-ai-builder.md`:~~
- ✅ Fixed: Reordered modes by simplicity (HTML attrs → YAML → Template → Exec)
- ✅ Fixed: Removed SQL from HTML examples
- ✅ Fixed: Added security rule about SQL in HTML

## Progress Tracker

| Task | Status | Notes |
|------|--------|-------|
| **Phase 0: Doc Fixes** | | |
| Fix SQL-in-HTML in pmf doc | ✅ Done | Reordered modes, removed SQL from HTML |
| **Phase 1: Core Infrastructure** | | |
| 1.1 lvt-data-* client extraction | ✅ Done | Already in interactive-block.ts |
| 1.1 lvt-data-* server string parsing | ✅ Done | action.go GetIntOk() parses strings |
| 1.1 lvt-data-* E2E test | ✅ Done | autopersist_e2e_test.go delete with lvt-data-id |
| 1.2 Smart table auto-generation | ✅ Done | `<table lvt-source>` auto-generates template |
| 1.2 Row actions support | ✅ Done | `lvt-actions="edit:✏️,delete:🗑️"` |
| 1.2 Smart select auto-generation | ✅ Done | `<select lvt-source>` auto-generates options |
| 1.2 Register component templates | ⏳ TODO | For datepicker, modal, autocomplete |
| 1.2 Add component CSS | ⏳ TODO | Must look production-ready |
| 1.3 Plugin interface | ✅ Done | Source interface in internal/source |
| 1.3 exec/stdout source | ✅ Done | Polyglot - lvtsource_e2e_test.go |
| 1.3 PostgreSQL source | ✅ Done | lvtsource_pg_e2e_test.go |
| 1.3 REST API source | ✅ Done | lvtsource_rest_e2e_test.go |
| 1.3 CSV/JSON source | ✅ Done | lvtsource_file_e2e_test.go |
| 1.4 Seamless partials | ✅ Done | `{{partial "file.md"}}` in partials_e2e_test.go |
| 1.4 Eject documentation | ⏳ TODO | |lv
| **Phase 2: Validation** | | |
| LLM testing | ⏳ TODO | |
| Reference doc | ⏳ TODO | |
| Prompt library (10) | ⏳ TODO | Copy-Paste App Store |
| **Phase 3: Future Features** | | |
| `<lvt-auth>` | ⏳ TODO | Critical for enterprise |
| `livepage build` | ⏳ TODO | Compile to binary |
| Headless API | ⏳ TODO | Auto-generated API |
| `lvt-ai` components | ⏳ TODO | AI-native UI |
| LivePage Hub | ⏳ TODO | Decentralized package manager |

## Implementation Details

### Phase 1.1: `lvt-data-*` ✅ COMPLETE

**Status: Complete.** The `lvt-data-*` infrastructure now fully works end-to-end.

**What's working:**
- Client extracts `lvt-data-*` attributes → `{ id: "123" }` ✅
- Data sent with action via WebSocket ✅
- Server parses and creates `ActionMessage` with data ✅
- Handlers access via `ctx.GetString("id")`, `ctx.Bind()` ✅
- `ctx.GetInt("id")` parses string values like "123" ✅

**Completed work:**

| File | Change | Status |
|------|--------|--------|
| `livetemplate/action.go:120-135` | `GetIntOk()` now parses strings using `strconv.Atoi` | ✅ Done |
| `livetemplate/action.go:155-166` | `GetFloatOk()` now parses strings using `strconv.ParseFloat` | ✅ Done |
| `livepage/autopersist_e2e_test.go` | E2E test: delete button with `lvt-data-id` | ✅ Done |
| `livepage/examples/autopersist-test/index.md` | Test example using `.Id` (JSON key capitalization) | ✅ Done |

### Phase 1.2: Component Library - PARTIALLY COMPLETE

**Smart Table/Select Auto-Generation: ✅ COMPLETE**

The smart table and select auto-generation is implemented. Users can now write minimal HTML and get full templates generated automatically.

**What's working:**
- `<table lvt-source="users">` auto-generates thead/tbody with {{range .Data}} ✅
- `<table lvt-source="users" lvt-columns="name:Name,email:Email">` for explicit columns ✅
- `<table lvt-source="users" lvt-actions="edit:✏️,delete:🗑️">` for row actions ✅
- `<select lvt-source="countries" lvt-value="code" lvt-label="name">` auto-generates options ✅
- Auto-discovery mode when no columns specified (iterates over data keys) ✅
- E2E test coverage in component_library_e2e_test.go ✅

**Completed files:**

| File | Purpose |
|------|---------|
| `page.go:autoGenerateTableTemplate()` | Transforms empty `<table lvt-source>` into full template |
| `page.go:autoGenerateSelectTemplate()` | Transforms empty `<select lvt-source>` into full template |
| `page.go:titleCase()` | Helper for field name → label conversion |
| `examples/component-library-test/` | Test example with all features |
| `component_library_e2e_test.go` | E2E test coverage |

**Usage:**

```html
<!-- Minimal: auto-discovers columns from data -->
<table lvt-source="users"></table>

<!-- Explicit columns with custom labels -->
<table lvt-source="users" lvt-columns="name:Name,email:Email"></table>

<!-- With row actions -->
<table lvt-source="users" lvt-columns="name:Name" lvt-actions="edit:✏️,delete:🗑️"></table>

<!-- Select dropdown -->
<select lvt-source="countries" lvt-value="code" lvt-label="name"></select>
```

**Still TODO:**

Future component library items (not blocking):
- Register component templates for non-native elements (datepicker, modal, autocomplete)
- Production-ready CSS styling

**`lvt-data-actions` Implementation** (datatable component):

The `lvt-data-actions` attribute defines row-level action buttons:

```html
<div lvt-source="pending-refunds"
     lvt-component="datatable"
     lvt-data-columns="customer_name:Customer,amount:Amount"
     lvt-data-actions="approve:✓,reject:✗">
</div>
```

The datatable component:
1. Parses `lvt-data-actions="approve:✓,reject:✗"` into `[{action: "approve", label: "✓"}, ...]`
2. For each row, generates action buttons with `lvt-click` and `lvt-data-id`:
   ```html
   <button lvt-click="approve" lvt-data-id="{{row.id}}">✓</button>
   <button lvt-click="reject" lvt-data-id="{{row.id}}">✗</button>
   ```
3. The existing Phase 1.1 `lvt-click` + `lvt-data-*` infrastructure handles click events

**Dependency**: Phase 1.1 (`lvt-data-*`) must be complete before `lvt-data-actions` works

### Phase 1.3: `lvt-source` Plugin System - exec/stdout ✅ COMPLETE

**Status: exec/stdout source complete.** The polyglot approach now works end-to-end.

**What's working:**
- Source interface in `internal/source/source.go` ✅
- Exec source implementation in `internal/source/exec.go` ✅
- Config parsing for sources in `internal/config/config.go` ✅
- lvt-source attribute detection in `page.go` ✅
- Code generation in `internal/compiler/lvtsource.go` ✅
- Wiring in `internal/server/websocket.go` ✅
- E2E test in `lvtsource_e2e_test.go` ✅

**Completed files:**

| File | Purpose |
|------|---------|
| `internal/config/config.go` | Added `Sources` map and `SourceConfig` struct |
| `internal/source/source.go` | Source interface and Registry |
| `internal/source/exec.go` | ExecSource implementation |
| `internal/compiler/lvtsource.go` | Code generation for lvt-source blocks |
| `internal/compiler/serverblock.go` | Added `CompileLvtSource` method |
| `internal/server/websocket.go` | Wires source config to compiler |
| `page.go` | Added `getLvtSource` for attribute detection |
| `examples/lvt-source-test/` | Test example with exec source |
| `lvtsource_e2e_test.go` | E2E test for exec source |

**All Phase 1.3 sources complete:** exec, pg, rest, json, csv

### Phase 1.3: `lvt-source` Plugin System - PostgreSQL ✅ COMPLETE

**Status: PostgreSQL source complete.** Database queries work end-to-end.

**What's working:**
- PostgreSQL source implementation in `internal/source/postgres.go` ✅
- Code generation for pg sources in `internal/compiler/lvtsource.go` ✅
- Connection pooling with configurable limits ✅
- DATABASE_URL environment variable or DSN in options ✅
- E2E test with Docker container in `lvtsource_pg_e2e_test.go` ✅

**Completed files:**

| File | Purpose |
|------|---------|
| `internal/source/postgres.go` | PostgresSource implementation with connection pooling |
| `internal/compiler/lvtsource.go` | Added `generatePostgresSourceCode()` |
| `examples/lvt-source-pg-test/` | Test example with PostgreSQL source |
| `lvtsource_pg_e2e_test.go` | E2E test using Docker PostgreSQL |

**Usage:**
```yaml
# livepage.yaml
sources:
  users:
    type: pg
    query: SELECT id, name, email FROM users ORDER BY id
```

```html
<div lvt-source="users">
  {{range .Data}}
    <tr><td>{{.Id}}</td><td>{{.Name}}</td></tr>
  {{end}}
</div>
```

Set `DATABASE_URL` environment variable or add `dsn` in options.

### Phase 1.3: `lvt-source` Plugin System - REST API ✅ COMPLETE

**Status: REST API source complete.** External API fetching works end-to-end.

**What's working:**
- REST source implementation in `internal/source/rest.go` ✅
- Code generation for rest sources in `internal/compiler/lvtsource.go` ✅
- Environment variable expansion in URLs and headers ✅
- Support for auth headers (Authorization, X-API-Key) ✅
- Handles wrapped responses (`data: []` and `results: []`) ✅
- E2E test with mock HTTP server in `lvtsource_rest_e2e_test.go` ✅

**Completed files:**

| File | Purpose |
|------|---------|
| `internal/source/rest.go` | RestSource implementation with HTTP client |
| `internal/compiler/lvtsource.go` | Added `generateRestSourceCode()` |
| `examples/lvt-source-rest-test/` | Test example with REST source |
| `lvtsource_rest_e2e_test.go` | E2E test using mock HTTP server |

**Usage:**
```yaml
# livepage.yaml
sources:
  users:
    type: rest
    url: https://api.example.com/users
    options:
      auth_header: "Bearer $API_TOKEN"
```

```html
<div lvt-source="users">
  {{range .Data}}
    <tr><td>{{.Id}}</td><td>{{.Name}}</td></tr>
  {{end}}
</div>
```

URLs and header values support `$ENV_VAR` expansion.

### Phase 1.3: `lvt-source` Plugin System - CSV/JSON Files ✅ COMPLETE

**Status: CSV/JSON file sources complete.** File-based data sources work end-to-end.

**What's working:**
- JSON file source implementation in `internal/source/file.go` ✅
- CSV file source implementation in `internal/source/file.go` ✅
- Code generation for json/csv sources in `internal/compiler/lvtsource.go` ✅
- JSON supports arrays, single objects, and NDJSON formats ✅
- CSV supports headers (auto-detected) and no-header modes ✅
- E2E tests in `lvtsource_file_e2e_test.go` ✅

**Completed files:**

| File | Purpose |
|------|---------|
| `internal/source/file.go` | JSONFileSource and CSVFileSource implementations |
| `internal/compiler/lvtsource.go` | Added `generateJSONFileSourceCode()` and `generateCSVFileSourceCode()` |
| `examples/lvt-source-file-test/` | Test example with JSON and CSV sources |
| `lvtsource_file_e2e_test.go` | E2E tests for JSON and CSV sources |

**Usage:**
```yaml
# livepage.yaml
sources:
  users:
    type: json
    file: users.json
  products:
    type: csv
    file: products.csv
```

```html
<div lvt-source="users">
  {{range .Data}}
    <tr><td>{{.Id}}</td><td>{{.Name}}</td></tr>
  {{end}}
</div>

<div lvt-source="products">
  {{range .Data}}
    <tr><td>{{.Name}}</td><td>{{.Price}}</td></tr>
  {{end}}
</div>
```

Files are resolved relative to the site directory.

---

**⚠️ SECURITY RULE: Never allow SQL/queries in HTML attributes.**

All queries MUST be defined server-side in `livepage.yaml`. HTML only references sources by name. Non-SQL display params (columns, page-size) are allowed in HTML.

**Configuration modes** (ordered by simplicity):

**Mode 1: HTML Attributes (Simplest - No Go Template Knowledge)**
```yaml
# livepage.yaml
sources:
  users:
    type: pg
    query: SELECT * FROM users
```
```html
<!-- Pure HTML - no Go templates needed -->
<div lvt-source="users"
     lvt-data-columns="name:Name,email:Email"
     lvt-data-page-size="10">
</div>
```

**Mode 2: YAML + Simple Reference**
```html
<table lvt-source="users">
  {{range .}}
    <tr><td>{{.name}}</td></tr>
  {{end}}
</table>
```

**Mode 3: Template with Dict (Power Users)**
```html
{{template "lvt:datatable:default:v1" dict "source" "users" "columns" "name:Name"}}
```

**Mode 4: Exec/Stdout (Polyglot)**
```yaml
sources:
  sales:
    type: exec
    cmd: python scripts/fetch_sales.py
```

**Priority sources**: exec/stdout, PostgreSQL, REST API, CSV/JSON

### Phase 1.4: Seamless Partials ✅ COMPLETE

**Status: Complete.** The `{{partial "file.md"}}` directive allows breaking up large files.

**What's working:**
- `{{partial "file.md"}}` directive in markdown files ✅
- Recursive partial support (partials can include other partials) ✅
- Circular dependency detection ✅
- Frontmatter stripping from partials ✅
- Path resolution relative to the including file's directory ✅
- E2E test in `partials_e2e_test.go` ✅

**Completed files:**

| File | Purpose |
|------|---------|
| `parser.go` | Added `ProcessPartials()` and `ParseMarkdownWithPartials()` |
| `page.go` | Updated `ParseFile()` to use `ParseMarkdownWithPartials()` |
| `examples/partials-test/` | Test example with header, sidebar, and footer partials |
| `partials_e2e_test.go` | E2E test for partial functionality |

**Usage:**
```markdown
# Main Page

{{partial "_header.md"}}

Main content here...

{{partial "_sidebar.md"}}

{{partial "_footer.md"}}
```

Paths are resolved relative to the including file's directory. Partials can have frontmatter (it will be stripped).

---

### Phase 2: Validation

#### 2.1 LLM Testing

Test if Claude/GPT-4 can reliably generate LivePage markdown:
- Create test prompts for common use cases (admin panel, dashboard, form)
- Verify generated output runs without errors
- Measure success rate and identify failure patterns

#### 2.2 Reference Documentation

Create LLM-friendly reference doc covering:
- All `lvt-*` attributes with examples
- Component library usage
- Source configuration (YAML patterns)
- Common patterns and anti-patterns

#### 2.3 Prompt Library ("Copy-Paste App Store")

Create 10 example prompts with working outputs:

| Prompt | Output |
|--------|--------|
| "Postgres admin panel" | `postgres-admin.md` |
| "CSV data viewer" | `csv-viewer.md` |
| "REST API dashboard" | `api-dashboard.md` |
| "User management CRUD" | `user-crud.md` |
| "Sales metrics dashboard" | `sales-dashboard.md` |
| "Log viewer" | `log-viewer.md` |
| "Feature flags manager" | `feature-flags.md` |
| "Webhook debugger" | `webhook-debugger.md` |
| "Environment config editor" | `env-editor.md` |
| "Database query runner" | `query-runner.md` |

---

### Phase 3: Future Features

#### 3.1 Declarative Authentication (`<lvt-auth>`)

```html
<lvt-auth provider="google" allowed-domains="company.com">
  <!-- Protected content -->
</lvt-auth>
```

**Impact**: Critical for enterprise adoption.

#### 3.2 Compile to Binary (`livepage build`)

```bash
livepage build my-app.md -o my-app
```

Embed markdown + assets into single executable. Double-click to run.

**Impact**: High - enables distribution to non-developers.

#### 3.3 Headless API Mode

Auto-generate REST API from state methods:
```
func (s *State) Refund(ctx Context) → POST /api/action/Refund
```

**Impact**: Medium - adds backend versatility.

#### 3.4 AI-Native Components (`lvt-ai`)

```html
<div lvt-ai="summarize" lvt-data-input="{{.Ticket.Description}}">
  <!-- AI-generated summary -->
</div>
```

**Impact**: High - unique differentiator vs Retool.

#### 3.5 LivePage Hub

```bash
livepage run github.com/user/repo/postgres-admin.md
```

Decentralized package manager for LivePage apps.

**Impact**: High - drives viral growth.
