# LivePage for Internal Tools: Product Vision

> **Core Insight**: Build internal tools in markdown + HTML. No Go code required for 99% of use cases.

---

## The Vision

```sh
Markdown + HTML + Declarative Integrations = Internal Tools
```

No drag-and-drop. No JavaScript. No build step. Just files in your repo.

---

## The Unified Model: Snippets + UIs + Reusable Sources

LivePage combines three capabilities that no single tool offers today:

### 1. Runnable Code Snippets (Like Runme)

Execute code directly from markdown. One step. No ceremony.

```markdown
## Check Database Status

```bash run
pg_isready -h localhost -p 5432
```

```bash run
psql -c "SELECT count(*) FROM pg_stat_activity;"
```
```

**Value**: Fastest path from documentation to execution. Zero overhead.

### 2. Rich Interactive UIs (LivePage's Differentiator)

When you need more than a code block—forms, tables, buttons, real-time updates.

```markdown
## Database Connections

<div lvt-source="pg_connections">
| PID | User | State | Duration | Action |
|-----|------|-------|----------|--------|
{{range .Results}}
| {{.pid}} | {{.usename}} | {{.state}} | {{.duration}}s | <button lvt-action="kill_connection" data-pid="{{.pid}}">Kill</button> |
{{end}}
</div>
```

**Value**: Build admin UIs in markdown. No React, no frontend framework.

### 3. Reusable Data Sources (The Ecosystem Play)

Pre-built, pre-approved integrations—community and company-specific.

```yaml
# livepage.yaml
integrations:
  # Community sources (shared, audited)
  - postgres
  - slack
  - kubernetes

  # Company sources (pre-approved workflows)
  - git: git@github.com:acme/livepage-internal
    ref: main
```

**Value**: Operators define secure, pre-approved data sources. Developers consume them.

---

## When to Use What

| Scenario | Approach | Example |
|----------|----------|---------|
| Quick check | **Snippet** | `psql -c "SELECT count(*)..."` |
| Multi-step workflow | **Snippet + UI** | Run command → display formatted results |
| Data display with actions | **UI** | Table of connections with "Kill" buttons |
| Complex dashboard | **UI** | Service health with charts, actions, refresh |
| Ad-hoc debugging | **Snippet** | `kubectl logs pod-name` |
| Compliance workflow | **UI** | Access review with approvals, audit trail |

---

## The Three Modes in Action

### Mode 1: Pure Snippets (Runme-Style)

For quick, ad-hoc operations. The code IS the interface.

```markdown
## Incident Response: API Down

```bash run
kubectl get pods -n production -l app=api
```

```bash run
kubectl logs -n production -l app=api --tail=100 --since=5m
```

```bash run
kubectl rollout restart deployment/api -n production
```
```

**No sources, no YAML, no config.** Just markdown with runnable code.

### Mode 2: UI-Enhanced Snippets

Snippets that render results into a richer display.

```markdown
## Pod Status

```bash run format="table"
kubectl get pods -n production -o json
```

<div lvt-render="last_output">
| Pod | Status | Restarts | Age |
|-----|--------|----------|-----|
{{range .items}}
| {{.metadata.name}} | {{.status.phase}} | {{index .status.containerStatuses 0 "restartCount"}} | {{.metadata.creationTimestamp | timeago}} |
{{end}}
</div>
```

**The snippet runs, the UI formats.** Best of both worlds.

### Mode 3: Full Interactive UI

For complex workflows that need persistent state, multiple data sources, and user interactions.

```markdown
## Service Health Dashboard

<div lvt-source="service_health" lvt-interval="30s">
{{range .Results}}
<div class="service-card">
  <h3>{{.name}}</h3>
  <span class="{{.status}}">{{.status}}</span>
  <div class="metrics">
    CPU: {{.cpu}}% | Memory: {{.memory}}%
  </div>
  <button lvt-action="restart_service" data-name="{{.name}}">Restart</button>
  <button lvt-action="scale_up" data-name="{{.name}}">Scale +1</button>
</div>
{{end}}
</div>
```

**Full declarative UI** with auto-refresh, actions, and styling.

---

## The Source Hierarchy

Sources can be defined at three levels, with increasing security and reusability:

### Level 1: Inline Snippets (Ad-hoc)

```markdown
```bash run
kubectl get pods -n production
```

```yaml

- **Security**: User must have permission to run
- **Reusability**: None (one-off)
- **Approval**: Per-execution


```yaml
# livepage.yaml (in your app)
sources:
  production_pods:
    integration: kubernetes
    namespace: production
    resource: pods
```

```html
<div lvt-source="production_pods">...</div>
```

- **Security**: Defined in code, reviewed in PR
- **Reusability**: Within this app
- **Approval**: Code review

### Level 3: Organization Sources (Pre-approved)

```yaml
# Company integration package
# git@github.com:acme/livepage-internal

sources:
  # Pre-approved, audited queries
  active_users:
    integration: postgres
    query: SELECT * FROM users WHERE status = 'active'
    allowed_roles: [admin, support]

  # Safe, parameterized operations
  user_lookup:
    integration: postgres
    query: SELECT id, name, email FROM users WHERE id = $1
    params:
      - name: user_id
        type: integer
    allowed_roles: [admin, support, engineering]
```

```html
<!-- Developers use pre-approved sources -->
<div lvt-source="acme.active_users">...</div>
```

- **Security**: Defined by platform team, audited
- **Reusability**: Across all apps in org
- **Approval**: Central governance

### Level 4: Community Sources (Shared)

```yaml
# Official integration: github.com/livepage-integrations/slack
integrations:
  - slack  # Uses community-maintained integration
```

```html
<button lvt-action="slack.post" data-channel="#alerts">Notify</button>
```

- **Security**: Open source, community reviewed
- **Reusability**: Global
- **Approval**: Community + your config

---

## Security Model: Snippets vs Sources

| Aspect | Runnable Snippets | Declarative Sources |
|--------|------------------|---------------------|
| **Execution** | Direct shell/command | Through integration layer |
| **Permissions** | User's shell access | Source-level RBAC |
| **Audit** | Shell history | Structured audit log |
| **Parameters** | Raw input | Validated, typed params |
| **Use case** | Ad-hoc, debugging | Production workflows |

**The insight**: Snippets are for operators with shell access. Sources are for governed, repeatable workflows.

### Permission Model

Permissions are defined at three levels:

```yaml
# livepage.yaml

# Level 1: Global defaults
permissions:
  snippets:
    # Who can run inline code snippets (default for all snippets)
    allowed_roles: [admin, oncall]
    # Or disable entirely
    enabled: false

  sources:
    # Default for all sources
    default_read: [viewer, editor, admin]
    default_write: [editor, admin]

# Level 2: Per-source/action override (in sources/actions definitions)
sources:
  pg_connections:
    integration: postgres
    query: ...
    allowed_roles: [admin, support]  # Overrides default

actions:
  kill_connection:
    integration: postgres
    query: ...
    require_role: admin  # Same as allowed_roles: [admin]

# Level 3: In HTML (visibility only, enforced server-side)
# <div lvt-require="admin">...</div>
```

__Note__: `require_role` and `allowed_roles` are equivalent—use whichever reads better. Server-side enforcement ensures roles can't be bypassed.

---

## Server-Side Workflows

For sequential actions (Action A → Action B), define workflows in YAML. The server executes them atomically.

```yaml
# livepage.yaml
workflows:
  # Simple linear workflow
  kill_and_notify:
    steps:
      - action: kill_idle_connections
      - action: slack.post
        with:
          channel: "#db-incidents"
          message: "Killed idle connections - {{.user}}"

  # Workflow with error handling
  create_incident:
    steps:
      - action: postgres.insert
        into: incidents
      - action: pagerduty.create_incident
        on_error: continue  # Don't fail if PagerDuty is down
      - action: slack.post
        with:
          channel: "#incidents"
    on_error: rollback  # Default: rollback all on failure
```

**Usage in HTML** - single action triggers entire workflow:

```html
<button lvt-click="kill_and_notify" lvt-data-ticket_id="{{.ID}}">
  Kill & Notify
</button>
```

**Why server-side workflows?**

- Fully declarative (YAML + HTML attributes only, no JavaScript)
- Atomic execution with rollback support
- Audit trail in one place
- Reusable across pages
- Aligns with LivePage's server-centric philosophy

**Conditional UI (reactive attributes)** - for showing success/error states:

```html
<button lvt-click="restart_service" lvt-data-service="api">Restart</button>
<div lvt-show-on:restart_service:success>✅ Service restarted!</div>
<div lvt-show-on:restart_service:error>❌ Failed to restart</div>
```

---

## Putting It Together: A Complete Runbook

This example shows all three modes in a single document:

```markdown
# Runbook: Database Connection Pool Exhausted

**Severity**: P1
**On-call team**: Platform

---

## Quick Check (Snippets)

```bash run
pg_isready -h $DATABASE_HOST -p 5432
```

```bash run
psql -c "SELECT count(*) FROM pg_stat_activity WHERE datname = current_database();"
```

---

## Connection Details (UI)

<div lvt-source="pg_connections">
<table>
  <tr>
    <th>PID</th>
    <th>User</th>
    <th>App</th>
    <th>State</th>
    <th>Duration</th>
    <th lvt-require="admin">Action</th>
  </tr>
{{range .Results}}
  <tr class="{{if gt .duration_seconds 300}}bg-red-100{{end}}">
    <td>{{.pid}}</td>
    <td>{{.usename}}</td>
    <td>{{.application_name}}</td>
    <td>{{.state}}</td>
    <td>{{.duration_seconds | printf "%.0f"}}s</td>
    <td lvt-require="admin">
      <button lvt-action="kill_connection" data-pid="{{.pid}}">Kill</button>
    </td>
  </tr>
{{end}}
</table>
</div>

---

## Kill Stale Connections (Governed Action)

<div lvt-require="admin" class="warning-box">
  ⚠️ This will terminate all idle connections older than 5 minutes.

  <button lvt-click="kill_and_notify">Kill All Idle (5+ min)</button>
</div>

<!-- Workflow defined in livepage.yaml:
workflows:
  kill_and_notify:
    steps:
      - action: kill_idle_connections
      - action: slack.post
        with:
          channel: "#db-incidents"
          message: "Killed idle connections - {{.user}}"
-->

---

## Manual Override (Snippet, Emergency Only)

If the UI isn't working, run directly:

```bash run require="admin"
psql -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE state = 'idle' AND state_change < now() - interval '5 minutes';"
```

---

## Verify Recovery

<div lvt-source="pg_connection_count">
{{if lt .total 50}}
✅ Pool recovered: {{.total}} connections
{{else}}
❌ Still elevated: {{.total}} connections
{{end}}
</div>

Or check manually:
```bash run
psql -c "SELECT count(*) FROM pg_stat_activity;"
```
```

**This runbook demonstrates**:

1. **Snippets** for quick checks (no setup)
2. **UIs** for rich data display with actions
3. **Governed actions** with RBAC and audit
4. **Fallback snippets** for emergencies
5. **Both modes** coexisting naturally

---

## Architecture

```ini
┌─────────────────────────────────────────────────────────────────┐
│  Markdown Document                                              │
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐   │
│  │ ```bash run     │ │ lvt-source      │ │ lvt-action      │   │
│  │ (snippets)      │ │ (data display)  │ │ (mutations)     │   │
│  └────────┬────────┘ └────────┬────────┘ └────────┬────────┘   │
└───────────┼───────────────────┼───────────────────┼────────────┘
            │                   │                   │
            ▼                   ▼                   ▼
┌───────────────────┐ ┌─────────────────────────────────────────┐
│  Shell Executor   │ │  Integration Layer                      │
│  ┌─────────────┐  │ │  ┌──────────┐ ┌──────────┐ ┌─────────┐ │
│  │ bash/zsh    │  │ │  │ postgres │ │ slack    │ │ k8s     │ │
│  │ kubectl     │  │ │  └──────────┘ └──────────┘ └─────────┘ │
│  │ psql        │  │ │                                         │
│  └─────────────┘  │ │  Source Hierarchy:                      │
│                   │ │  ├── Level 4: Community (GitHub)        │
│  Ad-hoc execution │ │  ├── Level 3: Organization (private Git)│
│  User's shell env │ │  ├── Level 2: Local (livepage.yaml)     │
│  (Level 1)        │ │  └── (Snippets are Level 1 - ad-hoc)    │
└───────────────────┘ └─────────────────────────────────────────┘
            │                   │
            ▼                   ▼
┌─────────────────────────────────────────────────────────────────┐
│  Security & Governance                                          │
│  ┌───────────────┐  ┌───────────────┐  ┌─────────────────────┐ │
│  │ Snippet RBAC  │  │ Source RBAC   │  │ Audit Log           │ │
│  │ (who can run) │  │ (per-source)  │  │ (who did what)      │ │
│  └───────────────┘  └───────────────┘  └─────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
            │                   │
            ▼                   ▼
┌─────────────────────────────────────────────────────────────────┐
│  Secrets Layer                                                  │
│  Environment vars → SOPS encrypted → Vault/AWS Secrets Manager  │
└─────────────────────────────────────────────────────────────────┘
```

**Two execution paths**:

1. **Snippets (Level 1)** → Shell executor → Direct command execution
2. **Sources/Actions (Levels 2-4)** → Integration layer → Governed, typed, audited

---

## Market Context

| Metric | Value |
|--------|-------|
| Market size (2025) | $26-38B |
| CAGR | 20-30% |
| Top players | Retool ($478M raised), Appsmith (35K GitHub stars) |
| Key pain point | Developer friction, vendor lock-in, JSON blob diffs |

### Go-To-Market Strategy: "Apps vs Ops"
*   **Don't fight Retool head-on initially**. Retool is for "Apps" (Customer Support Dashboard). LivePage is for "Ops" (Incident Response, Database Maintenance, Deployment scripts).
*   **Target the "On-Call" Engineer**. The person waking up at 3 AM doesn't want a complex UI; they want a guided, safe runbook. LivePage is the "Executable Runbook" platform.

---

## Why Markdown-Based Internal Tools Can Win

### Strategic Strengths
*   **The "Docs as Code" Moat**: By keeping everything in Markdown/Git, you solve the biggest pain point of Retool/Appsmith: version control, diffs, and PR reviews. This is your strongest wedge into engineering teams.
*   **The "Gradual Adoption" Path**: The three modes (Snippets → UI-Enhanced → Full UI) allow a team to start with simple runbooks (just text + code blocks) and incrementally upgrade them into full apps without rewriting. This lowers the barrier to entry significantly compared to "starting a new app" in a visual builder.
*   **Unified Context**: Most internal tools are 80% context (why am I running this?) and 20% action. Visual builders often lose the context. Your approach keeps the "Runbook" narrative front and center.

### The Precedent: Successful "X as Code" Movements

| Movement | Before | After | Why It Won |
|----------|--------|-------|------------|
| Infrastructure as Code | ClickOps | Terraform/Pulumi | Version control, code review |
| Docs as Code | Word/Confluence | Markdown in Git | Developer workflow |
| GitOps | Manual deployments | YAML in Git | Single source of truth |

**Pattern**: Move it INTO the developer workflow (Git, plain text, PRs) → adoption follows.

### The Retool/Appsmith Git Problem

| Issue | Retool/Appsmith | Markdown-Based |
|-------|-----------------|----------------|
| File format | YAML/JSON blobs | Human-readable markdown |
| Diffing | Hard to review | Clean, meaningful diffs |
| Merge conflicts | "Invalid changes can prevent deploys" | Standard text merge |
| AI-friendliness | Complex nested structures | 15% fewer tokens than JSON |

**Insight**: They bolted Git onto a visual builder. We build FROM markdown up.

## Critical Challenges & Risks

### The "Template Spaghetti" Risk
*   **Issue**: While "No JS" is a great tagline, complex logic inside Go templates (`{{if .status}} ... {{end}}`) can quickly become unmaintainable "spaghetti code" that is harder to debug than actual JavaScript.
*   **Mitigation**: Ensure your standard library of components (tables, status badges, buttons) is very rich so users rarely have to write raw HTML/CSS or complex template logic.

### Security Perception
*   **Issue**: "Runnable Snippets" (Mode 1) effectively gives shell access. Security teams at larger enterprises (your target for SaaS revenue) will be terrified of this.
*   **Mitigation**: The "Source Hierarchy" is the right answer, but you need to emphasize that **Mode 1 can be disabled globally** for production environments, forcing all actions to go through the governed "Level 2/3" integrations.

### The "Blank Page" Problem
*   **Issue**: Visual builders give you a canvas. Markdown gives you a blinking cursor.
*   **Mitigation**: You need a `livepage new` command or a VS Code extension that scaffolds these patterns instantly.

## Feature Suggestions for SaaS Success

### AI as a First-Class Citizen
Markdown is the native language of LLMs. You have a massive advantage over JSON-based builders here.
*   **Feature**: "Generate Runbook from Alert". Paste a PagerDuty alert payload, and LivePage generates a draft runbook with the correct `kubectl` or `postgres` queries.
*   **Feature**: "Natural Language Actions". Allow users to write `<!-- ai: create a table showing all pods with restart count > 5 -->` and have the CLI expand it into the correct HTML/Template syntax.

### The "Localhost" Experience
*   **Feature**: A robust CLI (`livepage serve`) that renders the UI locally is critical. If I have to push to a server to see if my table renders correctly, the feedback loop is too slow.
*   **Feature**: VS Code Extension. Since you are targeting developers, a preview pane in VS Code that allows interaction (clicking buttons) would be a killer feature.

### "Approval Workflows" as a Native Primitive
*   **Feature**: In `livepage.yaml`, define an approval gate.
    ```yaml
    actions:
      drop_table:
        integration: postgres
        require_approval:
          from: @tech-leads
          via: slack
    ```
    When a user clicks the button, it doesn't run. It sends a Slack message. A lead clicks "Approve", and *then* the original user sees the action complete. This is a high-value enterprise feature.

### Five Reasons to Win

1. **Developer workflow alignment** — Already use VS Code, Git, Markdown
2. **Code review works** — `<button>Approve</button>` vs JSON blob
3. **AI/LLM native** — Cursor/Copilot can edit markdown tools directly
4. **No vendor lock-in** — Universal format, edit anywhere
5. **Simplicity scales** — No JS frameworks, no build tools

---

## Specific Use Cases

| Use Case | Why LivePage Wins | What LivePage Offers | Competitor Gap |
|----------|-------------------|---------------------|----------------|
| **Runbooks** | Already markdown; add snippets + UIs | Snippets for quick checks, UIs for dashboards | Runme = snippets only, Rundeck = $125/user, no Git |
| **Pentest Reports** | PR-reviewable, version controlled | Snippets to re-run exploits, UIs for evidence tracking | Static PDF, no live testing |
| **Onboarding** | Step-by-step + state tracking | Snippets for setup commands, UIs for progress | TechDocs = static, no interactivity |
| **Ops Dashboards** | 30 lines vs React app | UIs for metrics, snippets for ad-hoc debugging | Grafana = overkill, Retool = no snippets |
| **Compliance** | Audit trail + evidence collection | UIs for reviews, governed actions for approvals | Spreadsheets, no automation |

**The thread**: Documentation that wants to be interactive—with BOTH quick snippets AND rich UIs.

### The "Zero Onboarding" Sweet Spot: The "Airplane.dev" Replacement

With the shutdown of **Airplane.dev** and **Interval**, there is a vacuum for "Code-first internal tools".
**Evidence.dev** proved that "Markdown + SQL = BI" is a winning formula.
**LivePage** is "Markdown + SQL + Actions = Admin Panels".

**The Use Case: "The Lightweight Admin Panel"**
*   **Scenario**: Product Manager needs to "Enable Beta Feature" for a specific Customer ID.
*   **The Old Way**: PM asks Dev → Dev runs SQL `UPDATE features SET enabled=true WHERE user_id=123`.
*   **The Retool Way**: Dev spends 2 days building a "Customer Admin" dashboard.
*   **The LivePage Way**: Dev adds `admin/customers.md`:
    ```markdown
    # Customer Admin
    <form lvt-action="enable_beta">
      <input name="user_id" placeholder="User ID">
      <button>Enable Beta</button>
    </form>
    ```
*   **Why it wins**: It replaces a Jira ticket or a Slack message with a safe, self-service file. No new "app" to deploy. Just a file in the repo.

### Real-World Examples: The "Form for a Script" Pattern

These are tasks that usually live in a `scripts/` folder and require an engineer to run them manually because they take arguments.

#### 1. The "Reset 2FA" Ticket (Support)
**The Pain**: Support agent gets a ticket "User lost phone". Agent pings Engineer. Engineer runs `python manage.py reset_2fa --user 123`.
**The LivePage Fix**: `ops/user-support.md`

```html
<h3>Reset 2FA</h3>
<form lvt-action="reset_2fa_script">
  <label>User Email</label>
  <input type="email" name="email" required placeholder="user@example.com">
  
  <label>Reason (for audit log)</label>
  <select name="reason">
    <option>Lost Device</option>
    <option>Suspicious Activity</option>
  </select>

  <button class="bg-red-600 text-white">Reset 2FA</button>
</form>
```

#### 2. The "Extend Trial" Request (Sales)
**The Pain**: Sales rep wants to close a deal, needs 7 more days. Pings CTO. CTO runs SQL update.
**The LivePage Fix**: `ops/sales-admin.md`

```html
<h3>Extend Trial</h3>
<form lvt-action="extend_trial">
  <label>Organization Domain</label>
  <input type="text" name="domain" placeholder="acme.com">
  
  <label>Extension</label>
  <div class="flex gap-2">
    <button name="days" value="7">1 Week</button>
    <button name="days" value="14">2 Weeks</button>
    <button name="days" value="30" lvt-require="admin">1 Month (Admin only)</button>
  </div>
</form>
```

#### 3. The "Manual Job Retry" (DevOps)
**The Pain**: A background job fails. The fix is "just run it again with force=true".
**The LivePage Fix**: `ops/jobs.md`

```html
<h3>Retry Dead Letter Job</h3>
<form lvt-action="retry_job">
  <label>Job ID</label>
  <input type="text" name="job_id" class="font-mono">
  
  <label>
    <input type="checkbox" name="force"> Force Retry (Ignore idempotency check)
  </label>

  <button>Retry Job</button>
</form>
<div lvt-show-on:retry_job:success>Job queued! ID: {{.new_job_id}}</div>
```

## The "Willingness to Pay" Use Case: "SOC2-in-a-Box for Operations"

While "Executable Runbooks" are great for adoption, the **Enterprise Value** lies in **Compliance, Auditability, and Correctness**.

**The Problem**:
1.  **The "ClickOps" Trap**: Manual actions in AWS/SaaS consoles are un-auditable and dangerous.
2.  **The "Script" Trap**: Scripts on laptops are un-governed and hard to share.
3.  **The "Retool" Trap**: Building full internal apps for every small operational task is too expensive (time-wise) and creates maintenance debt.

**The Solution**: LivePage as the "Standard Library for Operational Actions".
*   **Correctness**: Actions are defined in code (Go/SQL), reviewed in PRs.
*   **Safety**: RBAC is declarative.
*   **Audit**: Every click is a commit or a log entry.
*   **LLM Synergy**: LLMs write the *Markdown* and the *SQL*. LivePage provides the *Safe Runtime*.

### The Killer App: "The Living Compliance Document"

Imagine a "Quarterly Access Review" or "Production Change Log". Instead of a static PDF or a Jira ticket, it's a LivePage.

#### 1. Just-in-Time (JIT) Access & Break-Glass Workflows
**The Pain**: Dev needs prod access to debug. They request it. Manager approves. Access granted for 1 hour. Audit log generated.
**The LivePage Fix**: `ops/request-access.md`
*   User fills form.
*   Slack notification.
*   Manager approves.
*   LivePage executes the grant.

#### 2. "Customer Ops" for B2B SaaS
**The Pain**: Onboarding complex enterprise customers often requires manual DB tweaks, config setting, provisioning.
**The LivePage Fix**: `ops/customer-onboarding.md`
*   Core devs don't want to build a UI for them (waste of time).
*   Core devs don't want to give them SQL access (dangerous).
*   LivePage unblocks Sales/Support without distracting Engineering.

### Why this beats LLMs alone
*   LLM generates the *code*, but it cannot provide the *trust*.
*   You don't trust an LLM to "run this python script to refund the user" without a sandbox.
*   LivePage is the sandbox.

### Why this beats Retool
*   **Auditability**: The *entire UI* is code. You can `git blame` the "Refund" button to see who added it and who approved the PR. You can't do that easily in Retool.
*   **Speed**: "I need a refund form" -> LLM generates Markdown -> PR -> Merged -> Live. 10 minutes. Retool is hours/days.

---

## Real-World Examples

### 1. Runbook: Database Connection Pool Exhausted

```yaml
# livepage.yaml
sources:
  pg_connections:
    integration: postgres
    query: |
      SELECT pid, usename, application_name, state,
             query_start, state_change,
             EXTRACT(EPOCH FROM (now() - query_start)) as duration_seconds
      FROM pg_stat_activity
      WHERE datname = current_database()
      ORDER BY query_start DESC

  pg_connection_count:
    integration: postgres
    query: |
      SELECT count(*) as total,
             count(*) FILTER (WHERE state = 'active') as active,
             count(*) FILTER (WHERE state = 'idle') as idle,
             count(*) FILTER (WHERE state = 'idle in transaction') as idle_in_transaction
      FROM pg_stat_activity
      WHERE datname = current_database()

actions:
  kill_connection:
    integration: postgres
    query: SELECT pg_terminate_backend($1)
    params:
      - name: pid
        type: integer
        required: true
    require_role: admin

  kill_idle_connections:
    integration: postgres
    query: |
      SELECT pg_terminate_backend(pid)
      FROM pg_stat_activity
      WHERE datname = current_database()
        AND state = 'idle'
        AND state_change < now() - interval '5 minutes'
    require_role: admin

  notify_oncall:
    integration: slack
    action: post
    channel: "#db-incidents"
```

```markdown
<!-- runbooks/db-connection-pool.md -->
# Runbook: Database Connection Pool Exhausted

**Severity**: P1
**On-call team**: Platform
**Last updated**: 2024-01-15

## Symptoms
- Application logs show "connection pool exhausted" errors
- Increased latency on all database operations
- New connections timing out

---

## Quick Check (Snippets)

```bash run
pg_isready -h $DATABASE_HOST -p 5432
```

```bash run
psql -c "SELECT count(*) FROM pg_stat_activity WHERE datname = current_database();"
```

---

## Step 1: Assess Current State (UI)

<div lvt-source="pg_connection_count">
| Metric | Count |
|--------|-------|
| Total Connections | {{.total}} |
| Active | {{.active}} |
| Idle | {{.idle}} |
| Idle in Transaction | {{.idle_in_transaction}} |
</div>

<button lvt-action="notify_oncall"
        data-message="Investigating DB connection pool exhaustion. Current: {{.total}} connections">
  📢 Notify On-Call
</button>

---

## Step 2: Identify Long-Running Queries

<div lvt-source="pg_connections">
<table>
  <thead>
    <tr>
      <th>PID</th>
      <th>User</th>
      <th>App</th>
      <th>State</th>
      <th>Duration</th>
      <th>Action</th>
    </tr>
  </thead>
  <tbody>
  {{range .Results}}
    <tr class="{{if gt .duration_seconds 300.0}}bg-red-100{{end}}">
      <td>{{.pid}}</td>
      <td>{{.usename}}</td>
      <td>{{.application_name}}</td>
      <td>{{.state}}</td>
      <td>{{.duration_seconds | printf "%.0f"}}s</td>
      <td>
        <button lvt-action="kill_connection"
                data-pid="{{.pid}}"
                class="text-red-600">
          Kill
        </button>
      </td>
    </tr>
  {{end}}
  </tbody>
</table>
</div>

---

## Step 3: Kill Stale Connections (if needed)

<div lvt-require="admin" class="bg-yellow-50 p-4 rounded">
  ⚠️ **Caution**: This will terminate all idle connections older than 5 minutes.

  <button lvt-click="kill_and_notify_oncall" class="bg-red-600 text-white px-4 py-2 mt-2">
    Kill All Idle Connections (5+ min)
  </button>
</div>

---

## Step 4: Verify Recovery

After killing connections, wait 30 seconds and check the count again:

<div lvt-source="pg_connection_count">
| Metric | Count |
|--------|-------|
| Total | {{.total}} |
| Active | {{.active}} |

{{if lt .total 50}}
✅ Connection pool recovered. Total connections now under threshold.
{{else}}
❌ Still elevated. Consider scaling the database or investigating application issues.
{{end}}
</div>

---

## Post-Incident

- [ ] Update incident timeline
- [ ] Identify root cause (query, deployment, traffic spike)
- [ ] Create follow-up ticket if needed
```

---

### 2. Pentest Report: SQL Injection Finding

```yaml
# livepage.yaml
sources:
  finding_history:
    integration: postgres
    query: |
      SELECT * FROM pentest_findings
      WHERE finding_id = $1
      ORDER BY tested_at DESC
    params:
      - name: finding_id
        type: string
        required: true

actions:
  test_payload:
    type: code
    file: sources/pentest.go
    require_role: security

  update_status:
    integration: postgres
    query: UPDATE pentest_findings SET status = $1, verified_by = $2, verified_at = NOW() WHERE id = $3
    params:
      - name: status
        type: string
      - name: verified_by
        source: user
      - name: id
        type: integer

  create_jira_ticket:
    integration: jira
    action: create_issue
    project: SEC
    issue_type: Bug
```

```go
// sources/pentest.go
package sources

import (
    "context"
    "net/http"
    "github.com/livepage/sdk"
)

func init() {
    sdk.RegisterAction("test_payload", TestPayload)
}

func TestPayload(ctx context.Context, params sdk.Params) (any, error) {
    targetURL := params.String("url")
    payload := params.String("payload")
    method := params.String("method", "GET")

    client := &http.Client{Timeout: 10 * time.Second}

    var resp *http.Response
    var err error

    if method == "POST" {
        resp, err = client.Post(targetURL, "application/x-www-form-urlencoded", strings.NewReader(payload))
    } else {
        resp, err = client.Get(targetURL + "?" + payload)
    }
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)

    return map[string]any{
        "status_code": resp.StatusCode,
        "body":        string(body),
        "headers":     resp.Header,
        "vulnerable":  detectSQLiSignatures(string(body)),
    }, nil
}
```

```markdown
<!-- reports/2024-q1-pentest/sqli-user-api.md -->
# Finding: SQL Injection in User Search API

| Field | Value |
|-------|-------|
| **ID** | VULN-2024-0042 |
| **Severity** | 🔴 Critical |
| **CVSS** | 9.8 |
| **Status** | Open |
| **Found** | 2024-01-10 |
| **Endpoint** | `GET /api/v1/users/search` |

---

## Description

The `query` parameter in the user search endpoint is vulnerable to SQL injection.
User input is concatenated directly into SQL query without parameterization.

---

## Proof of Concept


<div class="bg-gray-100 p-4 rounded">
  <label>Target URL</label>
  <input type="text" name="url" value="https://staging.example.com/api/v1/users/search"
         class="w-full p-2 border rounded" lvt-model="TargetURL">

  <label class="mt-2">Payload</label>
  <select name="payload" lvt-model="Payload" class="w-full p-2 border rounded">
    <option value="query=' OR '1'='1">Basic OR injection</option>
    <option value="query=' UNION SELECT username,password FROM users--">UNION-based extraction</option>
    <option value="query='; DROP TABLE users;--">Destructive (DO NOT RUN IN PROD)</option>
    <option value="query=' AND SLEEP(5)--">Time-based blind</option>
  </select>

  <button lvt-action="test_payload"
          data-url="{{.TargetURL}}"
          data-payload="{{.Payload}}"
          class="mt-4 bg-blue-600 text-white px-4 py-2 rounded">
    🧪 Test Payload
  </button>
</div>


<div lvt-source="test_result">
{{if .Results}}
| Metric | Value |
|--------|-------|
| Status Code | {{.Results.status_code}} |
| Vulnerable | {{if .Results.vulnerable}}🔴 YES{{else}}🟢 NO{{end}} |

<details>
<summary>Response Body</summary>

```
{{.Results.body}}
```
</details>
{{end}}
</div>

---

## Evidence

```http
GET /api/v1/users/search?query=' OR '1'='1 HTTP/1.1
Host: staging.example.com
Authorization: Bearer eyJ...
```

```json
{
  "users": [
    {"id": 1, "email": "admin@example.com", "role": "admin"},
    {"id": 2, "email": "user@example.com", "role": "user"},
    ... (2,847 more records)
  ]
}
```

---

## Remediation

```diff
- query := "SELECT * FROM users WHERE name LIKE '%" + searchQuery + "%'"
+ query := "SELECT * FROM users WHERE name LIKE $1"
+ rows, err := db.Query(query, "%" + searchQuery + "%")
```

---

## Verification

<div class="flex gap-4">
  <button lvt-click="update_status"
          lvt-data-status="remediated"
          lvt-data-id="42"
          class="bg-green-600 text-white px-4 py-2 rounded">
    ✅ Mark as Remediated
  </button>

  <button lvt-click="verify_and_close_jira"
          lvt-data-id="42"
          lvt-data-summary="Close VULN-2024-0042: SQLi in user search"
          class="bg-blue-600 text-white px-4 py-2 rounded">
    🔒 Verify Fix & Close
  </button>
</div>


<div lvt-source="finding_history" data-finding_id="VULN-2024-0042">
| Date | Tester | Result |
|------|--------|--------|
{{range .Results}}
| {{.tested_at | date "2006-01-02"}} | {{.tested_by}} | {{if .vulnerable}}🔴 Vulnerable{{else}}🟢 Fixed{{end}} |
{{end}}
</div>
```

---

### 3. Onboarding: New Engineer Day 1

```yaml
# livepage.yaml
sources:
  onboarding_progress:
    integration: postgres
    query: |
      SELECT step_id, completed, completed_at
      FROM onboarding_progress
      WHERE user_id = $1
    params:
      - name: user_id
        source: user

  github_membership:
    type: code
    file: sources/onboarding.go

  slack_membership:
    type: code
    file: sources/onboarding.go

  aws_access:
    type: code
    file: sources/onboarding.go

actions:
  complete_step:
    integration: postgres
    query: |
      INSERT INTO onboarding_progress (user_id, step_id, completed, completed_at)
      VALUES ($1, $2, true, NOW())
      ON CONFLICT (user_id, step_id) DO UPDATE SET completed = true, completed_at = NOW()
    params:
      - name: user_id
        source: user
      - name: step_id
        type: string

  request_access:
    integration: slack
    action: post
    channel: "#it-requests"

  send_welcome:
    integration: slack
    action: post
    channel: "#engineering"
```

```go
// sources/onboarding.go
package sources

import (
    "context"
    "github.com/livepage/sdk"
    "github.com/google/go-github/v50/github"
)

func init() {
    sdk.RegisterSource("github_membership", CheckGitHubMembership)
    sdk.RegisterSource("slack_membership", CheckSlackMembership)
    sdk.RegisterSource("aws_access", CheckAWSAccess)
}

func CheckGitHubMembership(ctx context.Context, params sdk.Params) (any, error) {
    user := sdk.UserFromContext(ctx)
    client := github.NewClient(nil).WithAuthToken(os.Getenv("GITHUB_TOKEN"))

    // Check org membership
    membership, _, err := client.Organizations.GetOrgMembership(ctx, user.Email, "acme-corp")
    if err != nil {
        return map[string]any{"member": false, "error": err.Error()}, nil
    }

    // Check team memberships
    teams, _, _ := client.Teams.ListTeams(ctx, "acme-corp", nil)
    userTeams := []string{}
    for _, team := range teams {
        isMember, _, _ := client.Teams.GetTeamMembershipBySlug(ctx, "acme-corp", *team.Slug, user.GitHubUsername)
        if isMember != nil {
            userTeams = append(userTeams, *team.Name)
        }
    }

    return map[string]any{
        "member": true,
        "role":   *membership.Role,
        "teams":  userTeams,
    }, nil
}

func CheckSlackMembership(ctx context.Context, params sdk.Params) (any, error) {
    user := sdk.UserFromContext(ctx)
    // ... similar Slack API check
}

func CheckAWSAccess(ctx context.Context, params sdk.Params) (any, error) {
    user := sdk.UserFromContext(ctx)
    // ... AWS IAM check
}
```

```markdown
<!-- onboarding/day-1.md -->
# 🎉 Welcome to Acme Corp!

**Hi {{.User.FirstName}}!** We're excited to have you on the team.

This interactive checklist will help you get set up on Day 1.

---

## Your Progress

<div lvt-source="onboarding_progress">
{{$completed := 0}}
{{range .Results}}{{if .completed}}{{$completed = add $completed 1}}{{end}}{{end}}

**{{$completed}} of 6 steps completed**

<div class="w-full bg-gray-200 rounded-full h-4">
  <div class="bg-green-600 h-4 rounded-full" style="width: {{mul (div $completed 6.0) 100}}%"></div>
</div>
</div>

---

## Step 1: Verify Your Access


<div lvt-source="github_membership">
{{if .member}}
✅ **GitHub**: You're a member of `acme-corp`
- Role: {{.role}}
- Teams: {{range .teams}}`{{.}}` {{end}}
<button lvt-action="complete_step" data-step_id="github" class="text-sm text-green-600">
  Mark Complete
</button>
{{else}}
❌ **GitHub**: Not a member yet

<button lvt-action="request_access"
        data-message="🆕 Access request: {{.User.Name}} ({{.User.Email}}) needs GitHub org access"
        class="bg-blue-600 text-white px-3 py-1 rounded text-sm">
  Request GitHub Access
</button>
{{end}}
</div>


<div lvt-source="slack_membership">
{{if .member}}
✅ **Slack**: Connected
- Workspace: {{.workspace}}
- Channels: {{len .channels}} joined
<button lvt-action="complete_step" data-step_id="slack" class="text-sm text-green-600">
  Mark Complete
</button>
{{else}}
❌ **Slack**: Not connected

Check your email for the Slack invite, or:
<button lvt-action="request_access"
        data-message="🆕 Slack invite needed for {{.User.Email}}"
        class="bg-blue-600 text-white px-3 py-1 rounded text-sm">
  Request Slack Invite
</button>
{{end}}
</div>


<div lvt-source="aws_access">
{{if .has_access}}
✅ **AWS**: IAM user configured
- Account: {{.account_id}}
- Role: {{.role}}
<button lvt-action="complete_step" data-step_id="aws" class="text-sm text-green-600">
  Mark Complete
</button>
{{else}}
❌ **AWS**: No access configured

<button lvt-action="request_access"
        data-message="🆕 AWS IAM access needed for {{.User.Email}} - Team: Engineering"
        class="bg-blue-600 text-white px-3 py-1 rounded text-sm">
  Request AWS Access
</button>
{{end}}
</div>

---

## Step 2: Clone the Repos

```bash
# Main application
git clone git@github.com:acme-corp/platform.git

# Infrastructure
git clone git@github.com:acme-corp/infrastructure.git

# Documentation
git clone git@github.com:acme-corp/docs.git
```

<button lvt-action="complete_step" data-step_id="repos" class="bg-green-600 text-white px-4 py-2 rounded">
  ✅ I've cloned the repos
</button>

---

## Step 3: Set Up Local Environment

```bash
cd platform

# Install dependencies
make setup

# Copy environment template
cp .env.example .env

# Start local services
docker-compose up -d

# Run the app
make run
```

You should see the app at [http://localhost:3000](http://localhost:3000)

<button lvt-action="complete_step" data-step_id="local_env" class="bg-green-600 text-white px-4 py-2 rounded">
  ✅ Local environment working
</button>

---

## Step 4: Make Your First Commit

1. Create a branch: `git checkout -b {{.User.Username}}/hello-world`
2. Add yourself to `CONTRIBUTORS.md`
3. Push and create a PR

<button lvt-click="complete_and_welcome"
        lvt-data-step_id="first_commit"
        class="bg-green-600 text-white px-4 py-2 rounded">
  ✅ First PR created!
</button>

---

## 🎊 All Done?

<div lvt-source="onboarding_progress">
{{$completed := 0}}
{{range .Results}}{{if .completed}}{{$completed = add $completed 1}}{{end}}{{end}}

{{if eq $completed 6}}

Your manager has been notified. Next steps:
- Check out the [Engineering Wiki](/wiki)
- Join the #engineering-questions Slack channel
- Schedule 1:1s with your team members
{{else}}
You have **{{sub 6 $completed}} steps remaining**. Take your time!
{{end}}
</div>
```

---

### 4. Ops Dashboard: Service Health

```yaml
# livepage.yaml
sources:
  service_health:
    type: code
    file: sources/health.go

  recent_deployments:
    integration: postgres
    query: |
      SELECT service, version, deployed_at, deployed_by, status
      FROM deployments
      WHERE deployed_at > NOW() - interval '24 hours'
      ORDER BY deployed_at DESC
      LIMIT 10

  error_rates:
    type: code
    file: sources/health.go

actions:
  restart_service:
    integration: kubernetes
    action: rollout_restart
    require_role: admin

  scale_service:
    integration: kubernetes
    action: scale
    require_role: admin

  silence_alert:
    integration: pagerduty
    action: create_maintenance
    require_role: oncall
```

```go
// sources/health.go
package sources

import (
    "context"
    "github.com/livepage/sdk"
)

type ServiceHealth struct {
    Name        string  `json:"name"`
    Status      string  `json:"status"`
    Replicas    int     `json:"replicas"`
    ReadyPods   int     `json:"ready_pods"`
    CPU         float64 `json:"cpu_percent"`
    Memory      float64 `json:"memory_percent"`
    ErrorRate   float64 `json:"error_rate_5m"`
    Latency     float64 `json:"p99_latency_ms"`
    LastRestart string  `json:"last_restart"`
}

func init() {
    sdk.RegisterSource("service_health", GetServiceHealth)
    sdk.RegisterSource("error_rates", GetErrorRates)
}

func GetServiceHealth(ctx context.Context, params sdk.Params) (any, error) {
    namespace := params.String("namespace", "production")

    // Get deployments from Kubernetes
    deployments, _ := k8s.ListDeployments(namespace)

    // Enrich with Prometheus metrics
    services := make([]ServiceHealth, 0)
    for _, d := range deployments {
        errorRate, _ := prometheus.Query(fmt.Sprintf(
            `rate(http_requests_total{service="%s",status=~"5.."}[5m]) / rate(http_requests_total{service="%s"}[5m]) * 100`,
            d.Name, d.Name,
        ))
        latency, _ := prometheus.Query(fmt.Sprintf(
            `histogram_quantile(0.99, rate(http_request_duration_seconds_bucket{service="%s"}[5m]))`,
            d.Name,
        ))

        services = append(services, ServiceHealth{
            Name:        d.Name,
            Status:      d.Status,
            Replicas:    d.Spec.Replicas,
            ReadyPods:   d.Status.ReadyReplicas,
            CPU:         getResourceUsage(d.Name, "cpu"),
            Memory:      getResourceUsage(d.Name, "memory"),
            ErrorRate:   errorRate,
            Latency:     latency * 1000, // Convert to ms
            LastRestart: d.Status.LastRestartTime,
        })
    }

    return services, nil
}
```

```markdown
<!-- dashboards/service-health.md -->
# 🖥️ Service Health Dashboard

**Last updated**: <span lvt-interval="30s">{{now | date "15:04:05"}}</span>

---

## Overview

<div lvt-source="service_health" data-namespace="production">
<table class="w-full">
  <thead>
    <tr class="bg-gray-100">
      <th>Service</th>
      <th>Status</th>
      <th>Pods</th>
      <th>CPU</th>
      <th>Memory</th>
      <th>Error Rate</th>
      <th>P99 Latency</th>
      <th>Actions</th>
    </tr>
  </thead>
  <tbody>
  {{range .Results}}
    <tr class="{{if eq .status "Degraded"}}bg-yellow-50{{else if eq .status "Down"}}bg-red-50{{end}}">
      <td class="font-medium">{{.name}}</td>
      <td>
        {{if eq .status "Healthy"}}🟢{{else if eq .status "Degraded"}}🟡{{else}}🔴{{end}}
        {{.status}}
      </td>
      <td>{{.ready_pods}}/{{.replicas}}</td>
      <td class="{{if gt .cpu_percent 80.0}}text-red-600 font-bold{{end}}">
        {{.cpu_percent | printf "%.1f"}}%
      </td>
      <td class="{{if gt .memory_percent 85.0}}text-red-600 font-bold{{end}}">
        {{.memory_percent | printf "%.1f"}}%
      </td>
      <td class="{{if gt .error_rate_5m 1.0}}text-red-600 font-bold{{end}}">
        {{.error_rate_5m | printf "%.2f"}}%
      </td>
      <td class="{{if gt .p99_latency_ms 500.0}}text-yellow-600{{end}}{{if gt .p99_latency_ms 1000.0}}text-red-600 font-bold{{end}}">
        {{.p99_latency_ms | printf "%.0f"}}ms
      </td>
      <td>
        <div class="flex gap-2">
          <button lvt-action="restart_service"
                  data-service="{{.name}}"
                  data-namespace="production"
                  class="text-blue-600 hover:underline text-sm">
            Restart
          </button>
          <button lvt-action="scale_service"
                  data-service="{{.name}}"
                  data-replicas="{{add .replicas 2}}"
                  class="text-green-600 hover:underline text-sm">
            +2 Pods
          </button>
        </div>
      </td>
    </tr>
  {{end}}
  </tbody>
</table>
</div>

---

## Recent Deployments (24h)

<div lvt-source="recent_deployments">
<table class="w-full">
  <thead>
    <tr class="bg-gray-100">
      <th>Time</th>
      <th>Service</th>
      <th>Version</th>
      <th>Deployed By</th>
      <th>Status</th>
    </tr>
  </thead>
  <tbody>
  {{range .Results}}
    <tr>
      <td>{{.deployed_at | timeago}}</td>
      <td>{{.service}}</td>
      <td><code>{{.version | truncate 8}}</code></td>
      <td>{{.deployed_by}}</td>
      <td>
        {{if eq .status "success"}}✅{{else if eq .status "rolling"}}🔄{{else}}❌{{end}}
        {{.status}}
      </td>
    </tr>
  {{end}}
  </tbody>
</table>
</div>

---

## Quick Actions

<div lvt-require="oncall" class="flex gap-4 p-4 bg-gray-50 rounded">
  <button lvt-action="silence_alert"
          data-duration="30m"
          data-reason="Investigating"
          class="bg-yellow-500 text-white px-4 py-2 rounded">
    🔕 Silence Alerts (30m)
  </button>

  <button lvt-action="scale_service"
          data-service="api"
          data-replicas="10"
          class="bg-blue-600 text-white px-4 py-2 rounded">
    📈 Scale API to 10 pods
  </button>
</div>
```

---

### 5. Compliance: SOC2 Access Review

```yaml
# livepage.yaml
sources:
  admin_users:
    type: code
    file: sources/compliance.go

  access_review_status:
    integration: postgres
    query: |
      SELECT
        ar.id, ar.review_period, ar.status, ar.due_date,
        COUNT(ari.id) FILTER (WHERE ari.decision IS NULL) as pending,
        COUNT(ari.id) FILTER (WHERE ari.decision = 'approved') as approved,
        COUNT(ari.id) FILTER (WHERE ari.decision = 'revoked') as revoked
      FROM access_reviews ar
      LEFT JOIN access_review_items ari ON ar.id = ari.review_id
      WHERE ar.control_id = 'CC6.1'
      GROUP BY ar.id
      ORDER BY ar.created_at DESC
      LIMIT 5

  audit_log:
    integration: postgres
    query: |
      SELECT timestamp, actor, action, target, details
      FROM audit_log
      WHERE category = 'access_review'
      ORDER BY timestamp DESC
      LIMIT 50

actions:
  approve_access:
    integration: postgres
    query: |
      UPDATE access_review_items
      SET decision = 'approved', decided_by = $1, decided_at = NOW(), justification = $2
      WHERE id = $3
    params:
      - name: decided_by
        source: user
      - name: justification
        type: string
        required: true
      - name: item_id
        type: integer

  revoke_access:
    type: code
    file: sources/compliance.go
    require_role: admin

  export_evidence:
    type: code
    file: sources/compliance.go
    require_role: compliance
```

```go
// sources/compliance.go
package sources

import (
    "context"
    "github.com/livepage/sdk"
)

type AdminUser struct {
    ID            string   `json:"id"`
    Email         string   `json:"email"`
    Name          string   `json:"name"`
    Role          string   `json:"role"`
    LastLogin     string   `json:"last_login"`
    MFAEnabled    bool     `json:"mfa_enabled"`
    AccessSources []string `json:"access_sources"` // AWS, GitHub, etc.
}

func init() {
    sdk.RegisterSource("admin_users", GetAdminUsers)
    sdk.RegisterAction("revoke_access", RevokeAccess)
    sdk.RegisterAction("export_evidence", ExportEvidence)
}

func GetAdminUsers(ctx context.Context, params sdk.Params) (any, error) {
    var users []AdminUser

    // Aggregate from multiple sources
    awsAdmins, _ := aws.ListIAMUsersWithPolicy("AdministratorAccess")
    githubAdmins, _ := github.ListOrgOwners("acme-corp")
    oktaAdmins, _ := okta.ListUsersInGroup("Administrators")

    // Deduplicate and merge
    userMap := make(map[string]*AdminUser)
    for _, u := range awsAdmins {
        if existing, ok := userMap[u.Email]; ok {
            existing.AccessSources = append(existing.AccessSources, "AWS")
        } else {
            userMap[u.Email] = &AdminUser{
                Email:         u.Email,
                Name:          u.Name,
                AccessSources: []string{"AWS"},
            }
        }
    }
    // ... similar for GitHub, Okta

    // Enrich with last login from Okta
    for email, user := range userMap {
        oktaUser, _ := okta.GetUser(email)
        user.LastLogin = oktaUser.LastLogin
        user.MFAEnabled = oktaUser.MFAEnabled
        users = append(users, *user)
    }

    return users, nil
}

func RevokeAccess(ctx context.Context, params sdk.Params) error {
    userID := params.String("user_id")
    source := params.String("source")
    reviewer := sdk.UserFromContext(ctx)

    switch source {
    case "AWS":
        aws.RemoveUserFromGroup(userID, "Administrators")
    case "GitHub":
        github.RemoveOrgOwner("acme-corp", userID)
    case "Okta":
        okta.RemoveUserFromGroup(userID, "Administrators")
    }

    // Audit log
    audit.Log(ctx, "access.revoked", map[string]any{
        "user_id":  userID,
        "source":   source,
        "reviewer": reviewer.Email,
    })

    return nil
}

func ExportEvidence(ctx context.Context, params sdk.Params) (any, error) {
    reviewID := params.Int("review_id")

    // Generate PDF evidence package
    review, _ := db.GetAccessReview(reviewID)
    items, _ := db.GetAccessReviewItems(reviewID)
    auditLogs, _ := db.GetAuditLogs("access_review", reviewID)

    pdf := generateEvidencePDF(review, items, auditLogs)

    return map[string]any{
        "download_url": uploadToS3(pdf),
        "filename":     fmt.Sprintf("SOC2-CC6.1-AccessReview-%s.pdf", review.Period),
    }, nil
}
```

```markdown
<!-- compliance/soc2-cc6.1-access-review.md -->
# SOC2 CC6.1: Quarterly Access Review

**Control**: CC6.1 - The entity implements logical access security software, infrastructure, and architectures over protected information assets.

**Review Period**: Q1 2024
**Due Date**: 2024-04-15
**Reviewer**: {{.User.Name}}

---

## Review Status

<div lvt-source="access_review_status">
{{range .Results}}
{{if eq .review_period "Q1 2024"}}
| Metric | Count |
|--------|-------|
| Pending Review | {{.pending}} |
| Approved | {{.approved}} |
| Revoked | {{.revoked}} |
| Status | {{if eq .status "complete"}}✅ Complete{{else}}🔄 In Progress{{end}} |
| Due | {{.due_date | date "2006-01-02"}} |
{{end}}
{{end}}
</div>

---

## Users with Administrative Access

<div lvt-source="admin_users">
<table class="w-full">
  <thead>
    <tr class="bg-gray-100">
      <th>Name</th>
      <th>Email</th>
      <th>Access Sources</th>
      <th>Last Login</th>
      <th>MFA</th>
      <th>Decision</th>
    </tr>
  </thead>
  <tbody>
  {{range .Results}}
    <tr class="{{if not .mfa_enabled}}bg-red-50{{end}}">
      <td>{{.name}}</td>
      <td>{{.email}}</td>
      <td>
        {{range .access_sources}}
        <span class="inline-block bg-blue-100 text-blue-800 text-xs px-2 py-1 rounded mr-1">
          {{.}}
        </span>
        {{end}}
      </td>
      <td class="{{if gt (daysSince .last_login) 30}}text-yellow-600{{end}}{{if gt (daysSince .last_login) 90}}text-red-600 font-bold{{end}}">
        {{.last_login | timeago}}
      </td>
      <td>{{if .mfa_enabled}}✅{{else}}❌{{end}}</td>
      <td>
        <div class="flex gap-2">
          <button lvt-action="approve_access"
                  data-item_id="{{.id}}"
                  data-justification="Access required for job function"
                  class="bg-green-600 text-white px-2 py-1 rounded text-xs">
            ✓ Approve
          </button>
          {{range .access_sources}}
          <button lvt-action="revoke_access"
                  data-user_id="{{$.id}}"
                  data-source="{{.}}"
                  class="bg-red-600 text-white px-2 py-1 rounded text-xs">
            ✗ Revoke {{.}}
          </button>
          {{end}}
        </div>
      </td>
    </tr>
  {{end}}
  </tbody>
</table>
</div>

---

## Findings

<div class="bg-yellow-50 border-l-4 border-yellow-500 p-4 my-4">
  ⚠️ **Finding**: 2 users without MFA enabled

  **Recommendation**: Enforce MFA for all administrative accounts per security policy.
</div>

<div class="bg-yellow-50 border-l-4 border-yellow-500 p-4 my-4">
  ⚠️ **Finding**: 1 user with no login in 90+ days

  **Recommendation**: Review if access is still required. Consider revoking dormant accounts.
</div>

---

## Audit Trail

<div lvt-source="audit_log">
<table class="w-full text-sm">
  <thead>
    <tr class="bg-gray-100">
      <th>Timestamp</th>
      <th>Actor</th>
      <th>Action</th>
      <th>Target</th>
      <th>Details</th>
    </tr>
  </thead>
  <tbody>
  {{range .Results}}
    <tr>
      <td>{{.timestamp | date "2006-01-02 15:04"}}</td>
      <td>{{.actor}}</td>
      <td>
        {{if eq .action "access.approved"}}✅ Approved{{end}}
        {{if eq .action "access.revoked"}}❌ Revoked{{end}}
        {{if eq .action "review.started"}}🔄 Started{{end}}
      </td>
      <td>{{.target}}</td>
      <td class="text-gray-500">{{.details}}</td>
    </tr>
  {{end}}
  </tbody>
</table>
</div>

---

## Export Evidence

<div lvt-require="compliance" class="p-4 bg-gray-50 rounded">
  Generate evidence package for auditors:

  <button lvt-action="export_evidence"
          data-review_id="42"
          class="bg-blue-600 text-white px-4 py-2 rounded mt-2">
    📥 Export PDF Evidence Package
  </button>

  <p class="text-sm text-gray-500 mt-2">
    Includes: User list, decisions, justifications, audit trail, timestamps
  </p>
</div>

---

## Sign-Off

<div lvt-require="admin" class="p-4 border rounded">
  <p>I have reviewed all administrative access and confirm that:</p>
  <ul class="list-disc ml-6 my-2">
    <li>All approved users require admin access for their job function</li>
    <li>All findings have been addressed or documented</li>
    <li>Revoked access has been removed from all systems</li>
  </ul>

  <button lvt-action="complete_review"
          data-review_id="42"
          class="bg-green-600 text-white px-4 py-2 rounded mt-4">
    ✅ Complete Review & Sign Off
  </button>
</div>
```

---

## Data Sources & Actions

### Design Principle: No Raw Queries in HTML

**Problem**: Putting SQL/queries directly in HTML is a security risk:

- Exposes database schema to anyone viewing source
- Enables SQL injection if parameters aren't properly escaped
- Mixes concerns (presentation vs data access)
- Hard to audit what queries are running

**Solution**: Define data sources and actions in YAML, reference by name in HTML.

### Defining Data Sources (in `livepage.yaml` or `queries.yaml`)

```yaml
# livepage.yaml
sources:
  # Simple query
  open_tickets:
    integration: postgres
    query: SELECT id, customer, issue, created_at FROM tickets WHERE status = 'open'

  # Query with parameters (safe, parameterized)
  tickets_by_status:
    integration: postgres
    query: SELECT * FROM tickets WHERE status = $1
    params:
      - name: status
        type: string
        required: true

  # Query with pagination built-in
  recent_incidents:
    integration: postgres
    query: SELECT * FROM incidents ORDER BY created_at DESC
    pagination:
      default_limit: 20
      max_limit: 100

  # REST API source
  github_issues:
    integration: github
    endpoint: repos/{repo}/issues
    params:
      - name: repo
        type: string
        required: true

  # Kubernetes source
  production_pods:
    integration: kubernetes
    resource: pods
    namespace: production

actions:
  # Slack notification
  notify_support:
    integration: slack
    action: post
    channel: "#support"
    message: "Ticket {{.ticket_id}} claimed by {{.User.Name}}"

  # Database mutation
  close_ticket:
    integration: postgres
    query: UPDATE tickets SET status = 'closed', closed_by = $1, closed_at = NOW() WHERE id = $2
    params:
      - name: user_id
        source: user  # Auto-filled from authenticated user
      - name: ticket_id
        type: integer
        required: true

  # Kubernetes action
  restart_api:
    integration: kubernetes
    action: restart
    deployment: api
    namespace: production
    require_role: admin  # Built-in authorization
```

### Using Sources & Actions in HTML

```html
<!-- Reference source by name - no query exposed -->
<div lvt-source="open_tickets">
  {{range .Results}}
  <div>{{.customer}}: {{.issue}}</div>
  {{end}}
</div>

<!-- Source with parameters -->
<div lvt-source="tickets_by_status" data-status="{{.SelectedStatus}}">
  ...
</div>

<!-- Source with pagination -->
<div lvt-source="recent_incidents" data-limit="10" data-offset="{{.Page * 10}}">
  ...
</div>

<!-- Single action -->
<button lvt-click="notify_support" lvt-data-ticket_id="{{.ID}}">
  Claim Ticket
</button>

<!-- Sequential actions via workflow -->
<button lvt-click="close_and_notify" lvt-data-ticket_id="{{.ID}}">
  Close & Notify
</button>
```

### Custom/Company-Specific Sources

For company-specific data access patterns, define in a local integration:

```yaml
# integrations/company-api/integration.yaml
name: company-api
version: 1.0.0

auth:
  - name: COMPANY_API_KEY
    type: env

sources:
  active_users:
    description: Get active users from internal API
    http:
      method: GET
      url: https://api.internal/users?status=active
      headers:
        Authorization: "Bearer {{auth.COMPANY_API_KEY}}"

  user_permissions:
    description: Get permissions for a user
    params:
      - name: user_id
        type: string
        required: true
    http:
      method: GET
      url: "https://api.internal/users/{{params.user_id}}/permissions"
      headers:
        Authorization: "Bearer {{auth.COMPANY_API_KEY}}"

actions:
  revoke_access:
    description: Revoke user access
    params:
      - name: user_id
        type: string
        required: true
    http:
      method: DELETE
      url: "https://api.internal/users/{{params.user_id}}/access"
      headers:
        Authorization: "Bearer {{auth.COMPANY_API_KEY}}"
```

Then use in HTML:

```html
<div lvt-source="company-api.active_users">
  {{range .Results}}
  <div>
    {{.name}}
    <button lvt-action="company-api.revoke_access" data-user_id="{{.id}}">
      Revoke
    </button>
  </div>
  {{end}}
</div>
```

### Role-Based Access

```html
<!-- Only admins see this -->
<div lvt-require="admin">
  <button lvt-action="restart_api">Restart API</button>
</div>

<!-- Editors can submit, viewers can only view -->
<form lvt-require="editor">
  <input type="text" name="title">
  <button lvt-action="create_ticket">Create</button>
</form>
```

### Code-Based Sources (for Complex Cases)

YAML is great for simple cases, but sometimes you need:

- Complex data transformations
- Multi-step API calls
- Custom protocols (gRPC, GraphQL, proprietary)
- Business logic / validation
- Aggregations across multiple sources

**Solution**: Define sources in code (Go, TypeScript, or Python), reference the same way in HTML.

#### Go Example

```go
// sources/tickets.go
package sources

import (
    "context"
    "github.com/livepage/sdk"
)

func init() {
    sdk.RegisterSource("tickets_with_sla", TicketsWithSLA)
    sdk.RegisterAction("escalate_ticket", EscalateTicket)
}

// Source: complex query with business logic
func TicketsWithSLA(ctx context.Context, params sdk.Params) (any, error) {
    status := params.String("status", "open")

    tickets, err := db.Query(`
        SELECT t.*, c.sla_hours
        FROM tickets t
        JOIN customers c ON t.customer_id = c.id
        WHERE t.status = $1
    `, status)
    if err != nil {
        return nil, err
    }

    // Add computed SLA status
    for i := range tickets {
        tickets[i].SLABreached = time.Since(tickets[i].CreatedAt).Hours() > tickets[i].SLAHours
        tickets[i].TimeRemaining = tickets[i].SLAHours - time.Since(tickets[i].CreatedAt).Hours()
    }

    return tickets, nil
}

// Action: multi-step with side effects
func EscalateTicket(ctx context.Context, params sdk.Params) error {
    ticketID := params.Int("ticket_id")
    user := sdk.UserFromContext(ctx)

    // 1. Update ticket
    if err := db.Exec("UPDATE tickets SET priority = 'high' WHERE id = $1", ticketID); err != nil {
        return err
    }

    // 2. Notify on-call
    oncall, _ := pagerduty.GetOnCall("platform")
    slack.Post("#incidents", fmt.Sprintf("Ticket %d escalated by %s, paging %s", ticketID, user.Name, oncall.Name))

    // 3. Create audit log
    audit.Log(ctx, "ticket.escalated", ticketID)

    return nil
}
```

#### TypeScript Example

```typescript
// sources/analytics.ts
import { registerSource, registerAction, Params, Context } from '@livepage/sdk';

registerSource('dashboard_metrics', async (ctx: Context, params: Params) => {
  const timeRange = params.string('range', '7d');

  // Fetch from multiple sources
  const [tickets, revenue, users] = await Promise.all([
    db.query('SELECT COUNT(*) as count FROM tickets WHERE created_at > $1', [timeRange]),
    stripe.getRevenue(timeRange),
    analytics.getActiveUsers(timeRange),
  ]);

  // Transform and aggregate
  return {
    tickets: tickets.count,
    revenue: revenue.total,
    users: users.count,
    revenuePerUser: revenue.total / users.count,
  };
});

registerAction('sync_to_warehouse', async (ctx: Context, params: Params) => {
  const data = await db.query('SELECT * FROM events WHERE synced = false');
  await bigquery.insert('events', data);
  await db.exec('UPDATE events SET synced = true WHERE synced = false');
});
```

#### Python Example

```python
# sources/ml_predictions.py
from livepage import register_source, register_action
import pandas as pd

@register_source("churn_predictions")
def churn_predictions(ctx, params):
    customers = db.query("SELECT * FROM customers WHERE status = 'active'")
    df = pd.DataFrame(customers)

    # Run ML model
    df['churn_probability'] = model.predict_proba(df[features])[:, 1]
    df['risk_level'] = pd.cut(df['churn_probability'], bins=[0, 0.3, 0.7, 1], labels=['low', 'medium', 'high'])

    return df.to_dict('records')

@register_action("trigger_retention_campaign")
def trigger_retention(ctx, params):
    customer_id = params['customer_id']

    # Complex workflow
    customer = db.get("customers", customer_id)
    segment = ml.get_segment(customer)
    template = campaigns.get_template(segment)

    email.send(customer.email, template)
    db.exec("INSERT INTO campaign_sends (customer_id, campaign) VALUES ($1, $2)",
            customer_id, template.name)
```

#### Registration in `livepage.yaml`

```yaml
# livepage.yaml
sources:
  # YAML-defined (simple)
  open_tickets:
    integration: postgres
    query: SELECT * FROM tickets WHERE status = 'open'

  # Code-defined (complex) - just reference, implementation in code
  tickets_with_sla:
    type: code
    file: sources/tickets.go  # or .ts, .py

  dashboard_metrics:
    type: code
    file: sources/analytics.ts

  churn_predictions:
    type: code
    file: sources/ml_predictions.py

actions:
  # YAML-defined
  notify_support:
    integration: slack
    action: post
    channel: "#support"

  # Code-defined
  escalate_ticket:
    type: code
    file: sources/tickets.go
    require_role: admin

  sync_to_warehouse:
    type: code
    file: sources/analytics.ts
    require_role: admin
```

#### Usage in HTML (Same as YAML-defined)

```html
<!-- Code-defined source - same syntax -->
<div lvt-source="tickets_with_sla" data-status="open">
  {{range .Results}}
  <tr class="{{if .SLABreached}}bg-red-100{{end}}">
    <td>{{.ID}}</td>
    <td>{{.Customer}}</td>
    <td>{{.TimeRemaining | printf "%.1f"}}h remaining</td>
    <td>
      <button lvt-action="escalate_ticket" data-ticket_id="{{.ID}}">
        Escalate
      </button>
    </td>
  </tr>
  {{end}}
</div>

<!-- ML predictions -->
<div lvt-source="churn_predictions">
  {{range .Results}}
  <div class="{{.risk_level}}-risk">
    {{.name}}: {{.churn_probability | percent}} churn risk
    <button lvt-action="trigger_retention_campaign" data-customer_id="{{.id}}">
      Send Retention Email
    </button>
  </div>
  {{end}}
</div>
```

### When to Use What

| Complexity | Approach | Example |
|------------|----------|---------|
| Simple query | YAML | `SELECT * FROM users WHERE active = true` |
| Parameterized query | YAML | `SELECT * FROM tickets WHERE status = $1` |
| REST/HTTP API | YAML | GitHub issues, Slack channels |
| Multi-source aggregation | Code | Dashboard combining DB + Stripe + Analytics |
| Business logic | Code | SLA calculations, escalation workflows |
| ML/Data science | Code (Python) | Churn predictions, anomaly detection |
| Custom protocols | Code | gRPC, GraphQL, proprietary APIs |

### Security Benefits

| Aspect | Raw SQL in HTML | Named Sources (YAML/Code) |
|--------|----------------|---------------------------|
| Schema exposure | Visible to anyone | Hidden in config/code |
| SQL injection | Possible if not careful | Parameterized by default |
| Audit trail | Grep HTML files | Single location (YAML or code files) |
| Access control | Per-query checks | Role on source/action definition |
| Code review | Hard to spot issues | Clear diff in YAML or code |
| Complex logic | Not possible | Full language support |

---

## Integration Framework

### Structure (No compiled code, just YAML)

```ini
github.com/livepage-integrations/slack/
├── integration.yaml      # Declarative definition
├── README.md            # Usage docs
├── examples/
│   └── post-message.md  # Example usage
└── tests/
    └── test.md          # Integration tests
```

### Integration Definition Format

```yaml
# integration.yaml
name: slack
version: 1.0.0
description: Send messages and interact with Slack

auth:
  - name: SLACK_TOKEN
    type: env
    required: true

sources:
  channels:
    description: List Slack channels
    http:
      method: GET
      url: https://slack.com/api/conversations.list
      headers:
        Authorization: "Bearer {{auth.SLACK_TOKEN}}"

actions:
  post:
    description: Post message to channel
    params:
      - name: channel
        type: string
        required: true
      - name: message
        type: string
        required: true
    http:
      method: POST
      url: https://slack.com/api/chat.postMessage
      headers:
        Authorization: "Bearer {{auth.SLACK_TOKEN}}"
      body:
        channel: "{{params.channel}}"
        text: "{{params.message}}"
```

### Distribution (Git-based)

```yaml
# livepage.yaml
integrations:
  # Official (short syntax)
  - postgres
  - slack
  - github

  # Community (Git URL)
  - git: https://github.com/someone/livepage-jira
    version: v1.2.0

  # Private/company
  - git: git@github.com:company/livepage-internal-api
    ref: main

  # Local development
  - path: ./integrations/custom-api
```

---

## Authentication & Secrets

### User Authentication: OIDC-First

```yaml
# livepage.yaml
auth:
  provider: oidc
  issuer: https://company.okta.com
  client_id: ${OIDC_CLIENT_ID}
  client_secret: ${OIDC_CLIENT_SECRET}
  scopes: [openid, profile, email, groups]

  roles:
    admin: ["okta-admins", "platform-team"]
    editor: ["engineering"]
    viewer: ["*"]
```

**Why OIDC**:

- Declarative (config, not code)
- Enterprise-ready (Okta, Azure AD, Google Workspace)
- No password management
- MFA/conditional access handled by IdP

### Secrets: Three Tiers

| Tier | Method | Security Level | Use Case |
|------|--------|----------------|----------|
| Basic | Environment variables | Development | Local dev, demos |
| Standard | SOPS-encrypted in Git | Team/Startup | GitOps workflows |
| Enterprise | Vault/AWS Secrets Manager | Enterprise | Compliance, rotation |

```yaml
# Tier 1: Environment variables
integrations:
  postgres:
    connection: ${DATABASE_URL}

# Tier 2: SOPS-encrypted
secrets:
  file: secrets.enc.yaml

# Tier 3: External secrets manager
secrets:
  provider: vault
  address: https://vault.internal:8200
  path: secret/livepage/prod
```

---

## Minimum Viable Integrations

### Tier 1: Launch

| Integration | Type | Why Essential |
|-------------|------|---------------|
| PostgreSQL | Source + Action | Most common production DB |
| MySQL | Source + Action | Second most common |
| REST API | Source + Action | Connect to anything |
| Slack | Action | Notifications |
| SQLite | Source + Action | Already have via auto-persist |

### Tier 2: Month 1-2

| Integration | Type | Why Valuable |
|-------------|------|--------------|
| GitHub | Source + Action | Issues, PRs, deployments |
| Kubernetes | Source + Action | Pods, deployments, logs |
| Docker | Action | Container management |
| PagerDuty | Action | Incident response |
| AWS (S3, IAM) | Source + Action | Cloud operations |

### Tier 3: Community

MongoDB, Redis, Jira, Linear, Datadog, Grafana, Notion, Airtable, Google Sheets, Terraform, Vault

---

## Target Users

### Primary: Platform/DevOps Engineers

- Already use: Terraform, Kubernetes, Docker
- Already write: YAML, Markdown, Go
- Pain point: Need admin UIs but hate frontend work

### Secondary: Small Engineering Teams (5-20 people)

- Can't afford dedicated frontend engineers for internal tools
- Value Git workflows and code review
- Want self-hosted/open-source

### Anti-target

- Non-technical users (need drag-and-drop)
- Large enterprises with dedicated internal tools teams

---

## Positioning

### The Unique Value Proposition

LivePage is the **only tool** that combines:

| Capability | Runme | Retool | LivePage |
|------------|-------|--------|----------|
| Runnable code snippets | ✅ | ❌ | ✅ |
| Rich interactive UIs | ❌ | ✅ | ✅ |
| Git-native (clean diffs) | ✅ | ❌ | ✅ |
| Reusable data sources | ❌ | Partial | ✅ |
| No JavaScript required | ✅ | ❌ | ✅ |
| Server-side security | N/A | ❌ | ✅ |

**Tagline options**:

- "Runbooks that run. Dashboards that diff."
- "From documentation to application in markdown"
- "The missing link between docs and ops"

**One-liner**:

> Markdown files with runnable snippets AND interactive UIs. Git-native internal tools.

**Elevator pitch**:

> LivePage is runnable documentation meets internal tools. Start with a simple bash snippet—like Runme. Need a richer interface? Add HTML with data bindings—no React, no build step. Share reusable integrations across your org via Git. It's the docs-as-code approach applied to operations.

### The Journey

```sh
Runme (snippets only)
         ↓
LivePage (snippets + UIs + sources)
         ↓
Retool (UIs only, drag-and-drop)
```

**LivePage occupies the middle ground** that doesn't exist today:

- More powerful than pure documentation tools
- Simpler than full low-code platforms
- Git-native like infrastructure-as-code

---

## Roadmap

### Phase 1: Foundation (MVP)

- [ ] **Runnable snippets** - `bash run`, `python run`, etc.
- [ ] Snippet RBAC (who can run inline code)
- [ ] PostgreSQL + MySQL integrations
- [ ] REST API integration (generic)
- [ ] OIDC authentication
- [ ] Environment variable secrets
- [ ] Docker deployment

### Phase 2: Operations Focus

- [ ] __UI-enhanced snippets__ - `lvt-render="last_output"`
- [ ] Slack integration
- [ ] Kubernetes integration
- [ ] GitHub integration
- [ ] SOPS encrypted secrets
- [ ] Audit logging (snippets + sources)

### Phase 3: Enterprise & Ecosystem

- [ ] **Organization sources** - private Git-based integrations
- [ ] Vault/AWS Secrets Manager
- [ ] RBAC with IdP group mapping
- [ ] Integration registry/discovery
- [ ] Community contribution workflow
- [ ] Private integration support

---

## Business Model

**Self-hosted open-source** + **paid cloud hosting/support**

- **Free**: Self-host, community support
- **Paid**: Managed hosting, SSO, priority support, SLA

Similar to: Appsmith, GitLab, Supabase

---

## Open Questions

### Syntax & Semantics

1. **Attribute naming**: Is `lvt-source`, `lvt-action` the right pattern? Alternatives: `data-source`, `x-source`
2. **Snippet syntax**: Is `bash run` the right marker? Alternatives: `bash exec`, `bash live`, `bash!`
3. __Role syntax__: Standardize `require_role` (YAML) vs `lvt-require` (HTML) vs `allowed_roles`

### Snippets

4. **Snippet security**: How to sandbox snippets in multi-tenant environments?
5. **Snippet output capture**: How to pipe snippet output into UI rendering?
6. **Snippet vs Source boundary**: When should a snippet become a source?

### Sources & Integrations

7. **Complex queries**: How to handle joins, pagination, filtering in declarative syntax?
8. **Error handling**: How do users debug failed integrations?
9. **Real-time updates**: Should sources auto-refresh? What's the trigger model?

### Development & Testing

10. **Testing**: How do users test their markdown internal tools before deploying?
11. **Local development**: How to mock integrations for local testing?
12. **Preview mode**: How to preview markdown tools before publishing?

---

## Sources

- [Straits Research - Low-Code Market](https://straitsresearch.com/report/low-code-development-platform-market)
- [Mordor Intelligence - Low-Code Market](https://www.mordorintelligence.com/industry-reports/low-code-development-platform-market)
- [Appsmith - Retool Pricing](https://www.appsmith.com/blog/retool-pricing)
- [Superblocks - Source Control](https://www.superblocks.com/blog/introducing-source-control)
- [Runme - DevOps Notebooks](https://runme.dev/)
- [Phoenix LiveView](https://github.com/phoenixframework/phoenix_live_view)
- [SOPS - Secrets Management](https://github.com/getsops/sops)
- [GitGuardian - SOPS Guide](https://blog.gitguardian.com/a-comprehensive-guide-to-sops/)
