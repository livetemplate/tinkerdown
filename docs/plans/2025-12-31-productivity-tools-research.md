# Tinkerdown for Personal & Team Productivity

**Date:** 2025-12-31
**Status:** Research
**Goal:** Deeply explore productivity tools as a use case for tinkerdown

---

## Table of Contents

1. [The Opportunity](#the-opportunity)
2. [Why Tinkerdown for Productivity](#why-tinkerdown-for-productivity)
3. [App Categories](#app-categories)
4. [Detailed App Catalog](#detailed-app-catalog)
5. [Pattern Library](#pattern-library)
6. [LLM Generation Strategies](#llm-generation-strategies)
7. [Distribution & Sharing](#distribution--sharing)
8. [Gaps & Required Features](#gaps--required-features)
9. [Roadmap](#roadmap)

---

## The Opportunity

### The Problem with Current Productivity Tools

```
┌─────────────────────────────────────────────────────────────────┐
│                    TODAY'S PRODUCTIVITY STACK                   │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Note Apps       Todo Apps      Kanban Boards  Spreadsheets     │
│  ────────        ─────────      ────────────   ────────────     │
│  • $10/mo        • $5/mo        • $10/mo       • Free but ugly  │
│  • Vendor lock   • Limited      • Too visual   • No interactivity│
│  • Heavy         • No custom    • No code      • Export hell    │
│  • Slow          • Sync issues  • Inflexible   • No version ctrl│
│                                                                 │
│  Problems:                                                      │
│  • Too many accounts                                            │
│  • Data spread across services                                  │
│  • Can't customize for specific needs                           │
│  • Sync conflicts and data loss                                 │
│  • No version history                                           │
│  • Switching costs are high                                     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### The Tinkerdown Alternative

```
┌─────────────────────────────────────────────────────────────────┐
│                    TINKERDOWN PRODUCTIVITY                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                     ~/productivity/                       │  │
│  │                                                           │  │
│  │  habits.md         job-search.md      team-standup.md    │  │
│  │  expenses.md       reading-list.md    meeting-notes/     │  │
│  │  goals-2025.md     recipes.md         retrospectives/    │  │
│  │                                                           │  │
│  └──────────────────────────────────────────────────────────┘  │
│                             │                                   │
│                        git commit                               │
│                             │                                   │
│                      Version history                            │
│                      Sync via git                               │
│                      Backup for free                            │
│                                                                 │
│  Benefits:                                                      │
│  • Free forever                                                 │
│  • Own your data (plain text)                                   │
│  • Git = sync + backup + history                                │
│  • LLM generates custom tools in seconds                        │
│  • Edit markdown OR use UI                                      │
│  • Works offline                                                │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Why Tinkerdown for Productivity

### Core Value Propositions

| Value | How Tinkerdown Delivers |
|-------|-------------------------|
| **Free** | No SaaS subscriptions, self-hosted |
| **Portable** | Plain markdown files, no vendor lock-in |
| **Versioned** | Git provides full history |
| **Customizable** | Edit the markdown to change behavior |
| **LLM-friendly** | AI generates custom tools instantly |
| **Dual-mode** | Edit text OR use click-based UI |
| **Offline** | Local files, no internet required |
| **Composable** | Connect to APIs, databases, scripts |

### The "Throwaway App" Concept

Traditional apps are built to last forever. Tinkerdown enables **throwaway apps**:

```
┌─────────────────────────────────────────────────────────────────┐
│                     THROWAWAY APP LIFECYCLE                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Need arises        LLM generates        Use for days/weeks    │
│  ───────────        ─────────────        ──────────────────    │
│  "I need to         "Create a            Track applications,   │
│   track my job       tinkerdown app       update status,       │
│   applications"      for job search"      add notes            │
│                                                                 │
│       │                   │                      │              │
│       ▼                   ▼                      ▼              │
│                                                                 │
│  ┌─────────┐        ┌─────────┐           ┌─────────┐          │
│  │  5 sec  │───────▶│  1 min  │──────────▶│ Weeks   │          │
│  │  prompt │        │ working │           │ of use  │          │
│  └─────────┘        └─────────┘           └─────────┘          │
│                                                  │              │
│                                                  ▼              │
│                                                                 │
│  Need ends          Archive or delete     Data remains         │
│  ─────────          ─────────────────     ────────────         │
│  Got a job!         job-search.md →       Markdown is still    │
│                     archive/ or delete    readable forever     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Key insight:** You don't need a "job search app" forever. You need it for 3 months. LLM generates it, you use it, you're done. The data stays readable because it's markdown.

### Dual Interface Advantage

```
┌─────────────────────────────────────────────────────────────────┐
│                       DUAL INTERFACE                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  EDIT MODE (Markdown)              UI MODE (Browser)            │
│  ────────────────────              ─────────────────            │
│                                                                 │
│  ## Tasks                          ┌─────────────────────┐      │
│                                    │ Tasks               │      │
│  | Task | Done |                   │                     │      │
│  |------|------|                   │ ☑ Buy groceries     │      │
│  | Buy groceries | x |             │ ☐ Call mom          │      │
│  | Call mom | |                    │ ☐ Finish report     │      │
│  | Finish report | |               │                     │      │
│                                    │ [Add task: _____ ]  │      │
│                                    └─────────────────────┘      │
│                                                                 │
│  Power users:                      Casual users:                │
│  • Bulk edit in vim                • Click to toggle            │
│  • Regex find/replace              • Form to add                │
│  • Script automation               • No markdown knowledge      │
│  • Git diff changes                • Share with non-tech folks  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## App Categories

### Personal Productivity (Individual Use)

| Category | Apps | Complexity | Data Source |
|----------|------|------------|-------------|
| **Task Management** | Todo list, GTD inbox, project tasks | Low | markdown |
| **Habit Tracking** | Daily habits, streaks, habit stacking | Low-Med | markdown |
| **Time Tracking** | Pomodoro, time logs, project hours | Low | markdown |
| **Journaling** | Daily journal, mood tracker, gratitude | Low | markdown |
| **Finance** | Expense tracker, budget, net worth | Low-Med | markdown/sqlite |
| **Health** | Workout log, meal tracker, sleep log | Low | markdown |
| **Learning** | Reading list, course tracker, flashcards | Low-Med | markdown |
| **Life Admin** | Contacts, passwords*, home inventory | Low | markdown/json |

### Team Productivity (Collaborative Use)

| Category | Apps | Complexity | Data Source |
|----------|------|------------|-------------|
| **Meetings** | Meeting notes, action items, decisions | Low | markdown |
| **Standups** | Daily sync, blockers, updates | Low | markdown |
| **Retrospectives** | What went well, improvements, actions | Low | markdown |
| **Project Tracking** | Kanban, sprint board, milestones | Med | markdown/sqlite |
| **Knowledge Base** | Team wiki, runbooks, documentation | Low | markdown |
| **People** | Team directory, org chart, 1:1 notes | Low | markdown |
| **Hiring** | Interview feedback, candidate tracking | Med | markdown |
| **OKRs** | Goals, key results, progress tracking | Med | markdown |

### Personal Knowledge Management

| Category | Apps | Complexity | Data Source |
|----------|------|------------|-------------|
| **Notes** | Zettelkasten, linked notes, tags | Med | markdown |
| **Research** | Paper tracker, citation manager | Med | markdown |
| **Bookmarks** | Link saving, read-later, annotations | Low | markdown |
| **Learning** | Spaced repetition, vocabulary | Med | markdown |
| **Collections** | Quotes, recipes, gift ideas | Low | markdown |

---

## Detailed App Catalog

### 1. Habit Tracker

**Use case:** Track daily habits, build streaks, see patterns.

**Why tinkerdown:**
- Visual streaks motivate
- Edit habits by editing markdown
- Git history shows long-term patterns
- No subscription for habit apps

**Structure:**
```markdown
---
sources:
  habits:
    type: markdown
    anchor: "#habits"
    readonly: false
  log:
    type: markdown
    anchor: "#log"
    readonly: false
---

# Habit Tracker

## Habits {#habits}

| habit | frequency | streak |
|-------|-----------|--------|
| Exercise | daily | 5 |
| Read 30min | daily | 12 |
| No social media | daily | 3 |
| Weekly review | weekly | 8 |

## Today's Log

<form lvt-submit="add" lvt-source="log">
  <select name="habit" lvt-source="habits" lvt-value="habit" lvt-label="habit">
  </select>
  <button type="submit">✓ Done</button>
</form>

## Log {#log}

| date | habit | done |
|------|-------|------|
```

**Features needed:**
- Auto-date on form submit
- Streak calculation (computed field)
- Weekly/monthly view

---

### 2. Time Tracker / Pomodoro

**Use case:** Track time on projects, pomodoro sessions, billable hours.

**Why tinkerdown:**
- Simple logging without app overhead
- Export time data easily (it's markdown)
- Connect to calendar via exec source
- Custom categories per project

**Structure:**
```markdown
---
sources:
  sessions:
    type: markdown
    anchor: "#sessions"
    readonly: false
    auto_fields:
      start: "{{now:15:04}}"
---

# Time Tracker

## Active Session

<div id="timer">
  <span id="time">25:00</span>
  <button lvt-click="start_pomodoro">▶ Start</button>
  <button lvt-click="stop">⏹ Stop</button>
</div>

## Log Session

<form lvt-submit="add" lvt-source="sessions">
  <input name="project" placeholder="Project">
  <input name="duration" placeholder="Minutes" type="number">
  <input name="notes" placeholder="What did you do?">
  <button type="submit">Log</button>
</form>

## Sessions {#sessions}

| start | project | duration | notes |
|-------|---------|----------|-------|

## Weekly Summary

```lvt
<!-- Computed: sum duration by project -->
<table lvt-source="sessions" lvt-group-by="project" lvt-aggregate="duration:sum">
</table>
```
```

**Features needed:**
- Timer component (JavaScript in lvt block)
- Aggregation functions (sum, count, avg)
- Date filtering

---

### 3. Daily Journal / Mood Tracker

**Use case:** Daily reflection, mood tracking, gratitude practice.

**Why tinkerdown:**
- Journal stays readable forever
- Git history = journal history
- Add structure (mood, gratitude) or free-form
- Private (local files)

**Structure:**
```markdown
---
sources:
  entries:
    type: markdown
    anchor: "#entries"
    auto_fields:
      date: "{{now:2006-01-02}}"
      time: "{{now:15:04}}"
---

# Daily Journal

## New Entry

<form lvt-submit="add" lvt-source="entries">
  <select name="mood">
    <option value="great">😊 Great</option>
    <option value="good">🙂 Good</option>
    <option value="okay">😐 Okay</option>
    <option value="bad">😔 Bad</option>
  </select>
  <textarea name="gratitude" placeholder="3 things I'm grateful for..."></textarea>
  <textarea name="reflection" placeholder="Today's reflection..."></textarea>
  <button type="submit">Save Entry</button>
</form>

## Entries {#entries}

| date | time | mood | gratitude | reflection |
|------|------|------|-----------|------------|
```

**Features needed:**
- Rich text / multiline in tables
- Mood visualization (emoji or chart)
- Search/filter entries

---

### 4. Expense Tracker

**Use case:** Track spending, categorize expenses, monthly budgets.

**Why tinkerdown:**
- No financial app has your data
- Export to CSV trivially
- Custom categories
- Connect to bank via API (advanced)

**Structure:**
```markdown
---
sources:
  expenses:
    type: markdown
    anchor: "#expenses"
    auto_fields:
      date: "{{now:2006-01-02}}"
  budget:
    type: markdown
    anchor: "#budget"
---

# Expense Tracker - January 2025

## Budget {#budget}

| category | budget | spent |
|----------|--------|-------|
| Food | 500 | 0 |
| Transport | 200 | 0 |
| Entertainment | 150 | 0 |

## Add Expense

<form lvt-submit="add" lvt-source="expenses">
  <input name="amount" type="number" step="0.01" placeholder="Amount">
  <select name="category">
    <option>Food</option>
    <option>Transport</option>
    <option>Entertainment</option>
    <option>Other</option>
  </select>
  <input name="description" placeholder="What for?">
  <button type="submit">Add</button>
</form>

## Expenses {#expenses}

| date | amount | category | description |
|------|--------|----------|-------------|

## Summary

```lvt
<!-- Sum by category, compare to budget -->
```
```

**Features needed:**
- Aggregation (sum by category)
- Computed fields (budget - spent)
- Charts (optional)

---

### 5. Team Standup

**Use case:** Async daily standups, track blockers, see team updates.

**Why tinkerdown:**
- No Slack clutter
- Searchable history
- See patterns over time
- Works for remote/async teams

**Structure:**
```markdown
---
sources:
  updates:
    type: markdown
    anchor: "#updates"
    auto_fields:
      date: "{{now:2006-01-02}}"
      time: "{{now:15:04}}"
      who: "{{operator}}"
---

# Team Standup - Week of Jan 6

## Post Update

<form lvt-submit="add" lvt-source="updates">
  <textarea name="yesterday" placeholder="What I did yesterday..."></textarea>
  <textarea name="today" placeholder="What I'm doing today..."></textarea>
  <textarea name="blockers" placeholder="Any blockers?"></textarea>
  <button type="submit">Post Update</button>
</form>

## Updates {#updates}

| date | time | who | yesterday | today | blockers |
|------|------|-----|-----------|-------|----------|
```

**Features needed:**
- Filter by date (today, this week)
- Filter by person
- Highlight blockers

---

### 6. Interview Feedback Tracker

**Use case:** Track candidates, collect feedback, make hiring decisions.

**Why tinkerdown:**
- Structured feedback format
- All interviewers contribute
- Git PR for hiring decision
- No expensive ATS

**Structure:**
```markdown
---
sources:
  candidates:
    type: markdown
    anchor: "#candidates"
  feedback:
    type: markdown
    anchor: "#feedback"
    auto_fields:
      date: "{{now:2006-01-02}}"
      interviewer: "{{operator}}"
---

# Hiring: Senior Engineer

## Candidates {#candidates}

| name | stage | status |
|------|-------|--------|
| Alice Smith | Onsite | In Progress |
| Bob Jones | Phone Screen | Passed |
| Carol White | Offer | Accepted |

## Add Feedback

<form lvt-submit="add" lvt-source="feedback">
  <select name="candidate" lvt-source="candidates" lvt-value="name" lvt-label="name">
  </select>
  <select name="round">
    <option>Phone Screen</option>
    <option>Technical</option>
    <option>System Design</option>
    <option>Behavioral</option>
  </select>
  <select name="rating">
    <option value="strong_yes">Strong Yes</option>
    <option value="yes">Yes</option>
    <option value="neutral">Neutral</option>
    <option value="no">No</option>
    <option value="strong_no">Strong No</option>
  </select>
  <textarea name="notes" placeholder="Feedback..."></textarea>
  <button type="submit">Submit Feedback</button>
</form>

## Feedback {#feedback}

| date | interviewer | candidate | round | rating | notes |
|------|-------------|-----------|-------|--------|-------|
```

**Features needed:**
- Filter feedback by candidate
- Aggregate ratings
- Decision workflow

---

### 7. OKR Tracker

**Use case:** Set goals, track key results, measure progress.

**Why tinkerdown:**
- Company-wide visibility via git
- Progress updates are commits
- Historical OKRs preserved
- Custom scoring

**Structure:**
```markdown
---
sources:
  objectives:
    type: markdown
    anchor: "#objectives"
  keyresults:
    type: markdown
    anchor: "#keyresults"
  checkins:
    type: markdown
    anchor: "#checkins"
    auto_fields:
      date: "{{now:2006-01-02}}"
---

# Q1 2025 OKRs - Engineering

## Objectives {#objectives}

| id | objective | owner | status |
|----|-----------|-------|--------|
| O1 | Improve system reliability | @alice | On Track |
| O2 | Reduce deploy time | @bob | At Risk |

## Key Results {#keyresults}

| objective | kr | target | current | confidence |
|-----------|-----|--------|---------|------------|
| O1 | Reduce p99 latency to <100ms | 100ms | 150ms | 70% |
| O1 | Achieve 99.9% uptime | 99.9% | 99.7% | 60% |
| O2 | Deploy time < 10 min | 10min | 25min | 40% |

## Weekly Check-in

<form lvt-submit="add" lvt-source="checkins">
  <select name="kr" lvt-source="keyresults" lvt-value="kr" lvt-label="kr">
  </select>
  <input name="current" placeholder="Current value">
  <input name="confidence" type="number" min="0" max="100" placeholder="Confidence %">
  <textarea name="notes" placeholder="Update notes..."></textarea>
  <button type="submit">Update</button>
</form>

## Check-ins {#checkins}

| date | kr | current | confidence | notes |
|------|-----|---------|------------|-------|
```

---

### 8. 1:1 Meeting Notes

**Use case:** Track recurring 1:1s, action items, career growth.

**Why tinkerdown:**
- Private (not in cloud services)
- Long-term history
- Template for consistency
- Action item tracking

**Structure:**
```markdown
---
sources:
  meetings:
    type: markdown
    anchor: "#meetings"
    auto_fields:
      date: "{{now:2006-01-02}}"
  actions:
    type: markdown
    anchor: "#actions"
---

# 1:1 Notes: Alice <> Bob

## Quick Add

<form lvt-submit="add" lvt-source="meetings">
  <textarea name="discussed" placeholder="What we discussed..."></textarea>
  <textarea name="feedback" placeholder="Feedback given/received..."></textarea>
  <button type="submit">Save Notes</button>
</form>

<form lvt-submit="add" lvt-source="actions">
  <input name="action" placeholder="Action item">
  <input name="owner" placeholder="Owner">
  <input name="due" type="date">
  <button type="submit">Add Action</button>
</form>

## Open Actions {#actions}

| action | owner | due | done |
|--------|-------|-----|------|

## Meeting History {#meetings}

| date | discussed | feedback |
|------|-----------|----------|
```

---

### 9. Personal CRM

**Use case:** Track contacts, interactions, follow-ups.

**Why tinkerdown:**
- Own your contact data
- Log interactions over years
- Custom fields per context
- Export easily

**Structure:**
```markdown
---
sources:
  contacts:
    type: markdown
    anchor: "#contacts"
  interactions:
    type: markdown
    anchor: "#interactions"
    auto_fields:
      date: "{{now:2006-01-02}}"
---

# Personal CRM

## Contacts {#contacts}

| name | company | email | last_contact | notes |
|------|---------|-------|--------------|-------|

## Log Interaction

<form lvt-submit="add" lvt-source="interactions">
  <select name="contact" lvt-source="contacts" lvt-value="name" lvt-label="name">
  </select>
  <select name="type">
    <option>Email</option>
    <option>Call</option>
    <option>Meeting</option>
    <option>Coffee</option>
  </select>
  <textarea name="notes" placeholder="What did you discuss?"></textarea>
  <input name="follow_up" type="date" placeholder="Follow up date">
  <button type="submit">Log</button>
</form>

## Recent Interactions {#interactions}

| date | contact | type | notes | follow_up |
|------|---------|------|-------|-----------|

## Due for Follow-up

```lvt
<!-- Filter: follow_up <= today AND not done -->
```
```

---

### 10. Retrospective

**Use case:** Sprint retros, project post-mortems, team reflection.

**Why tinkerdown:**
- Permanent record in git
- Action items tracked
- Anonymous input (optional)
- Compare retros over time

**Structure:**
```markdown
---
sources:
  good:
    type: markdown
    anchor: "#good"
  improve:
    type: markdown
    anchor: "#improve"
  actions:
    type: markdown
    anchor: "#actions"
---

# Sprint 42 Retrospective

**Date:** 2025-01-03
**Facilitator:** Alice
**Attendees:** Bob, Carol, Dave

## What Went Well {#good}

| item | votes |
|------|-------|

<form lvt-submit="add" lvt-source="good">
  <input name="item" placeholder="Something that went well...">
  <button type="submit">Add</button>
</form>

## What Could Improve {#improve}

| item | votes |
|------|-------|

<form lvt-submit="add" lvt-source="improve">
  <input name="item" placeholder="Something to improve...">
  <button type="submit">Add</button>
</form>

## Action Items {#actions}

| action | owner | due | status |
|--------|-------|-----|--------|

<form lvt-submit="add" lvt-source="actions">
  <input name="action" placeholder="Action to take">
  <input name="owner" placeholder="Owner">
  <input name="due" type="date">
  <button type="submit">Add Action</button>
</form>
```

---

## Pattern Library

### Pattern 1: Simple CRUD Tracker

**When to use:** Any list with add/edit/delete.

```markdown
---
sources:
  items:
    type: markdown
    anchor: "#items"
    readonly: false
---

## Items {#items}

| id | name | status |
|----|------|--------|

<form lvt-submit="add" lvt-source="items">
  <input name="name">
  <button type="submit">Add</button>
</form>

<table lvt-source="items" lvt-actions="delete:Delete">
</table>
```

---

### Pattern 2: Log with Auto-Timestamp

**When to use:** Activity logs, journals, time tracking.

```markdown
---
sources:
  log:
    type: markdown
    anchor: "#log"
    auto_fields:
      timestamp: "{{now:2006-01-02 15:04}}"
      who: "{{operator}}"
---

## Log {#log}

| timestamp | who | entry |
|-----------|-----|-------|

<form lvt-submit="add" lvt-source="log">
  <input name="entry" placeholder="What happened?">
  <button type="submit">Log</button>
</form>
```

---

### Pattern 3: Kanban / Status Board

**When to use:** Task status, hiring pipeline, deal stages.

```markdown
---
sources:
  items:
    type: markdown
    anchor: "#items"
---

## Items {#items}

| id | name | status |
|----|------|--------|
| 1 | Task A | todo |
| 2 | Task B | doing |
| 3 | Task C | done |

## Board View

```lvt
<div style="display: flex; gap: 20px;">
  <div class="column">
    <h3>Todo</h3>
    <ul lvt-source="items" lvt-filter="status:todo" lvt-field="name">
    </ul>
  </div>
  <div class="column">
    <h3>Doing</h3>
    <ul lvt-source="items" lvt-filter="status:doing" lvt-field="name">
    </ul>
  </div>
  <div class="column">
    <h3>Done</h3>
    <ul lvt-source="items" lvt-filter="status:done" lvt-field="name">
    </ul>
  </div>
</div>
```
```

---

### Pattern 4: Dashboard with Multiple Sources

**When to use:** Aggregate data from multiple files/sources.

```markdown
---
sources:
  tasks:
    type: markdown
    file: "./_data/tasks.md"
    anchor: "#tasks"
  team:
    type: markdown
    file: "./_data/team.md"
    anchor: "#members"
  metrics:
    type: exec
    cmd: "./scripts/metrics.sh"
---

# Dashboard

## Tasks
<table lvt-source="tasks" lvt-columns="task,status,owner"></table>

## Team
<ul lvt-source="team" lvt-field="name"></ul>

## System Metrics
<table lvt-source="metrics" lvt-columns="metric,value"></table>
```

---

### Pattern 5: Template-Based Creation

**When to use:** Recurring events (meetings, sprints, reviews).

```
templates/
  weekly-meeting.md
  sprint-retro.md
  1-on-1.md

instances/
  2025-01-03-weekly.md    # cp templates/weekly-meeting.md
  sprint-42-retro.md       # cp templates/sprint-retro.md
```

Shell helper:
```bash
meeting() {
  local template="templates/$1.md"
  local name="${2:-$(date +%Y-%m-%d)}-$1"
  cp "$template" "instances/$name.md"
  tinkerdown serve "instances/$name.md"
}

# Usage
meeting weekly
meeting sprint-retro sprint-43
```

---

## LLM Generation Strategies

### Prompt Templates by Category

#### Personal Tracker
```
Create a tinkerdown app to track [THING].
I want to: [add new items, mark as done, delete, filter by status].
Store data in the markdown file.
Include: [date tracking, categories, notes].
```

#### Team Tool
```
Create a tinkerdown app for team [ACTIVITY].
Features: [list members, add entries, track status, assign owners].
Auto-fill: date and current user.
Include a summary view showing [AGGREGATION].
```

#### Dashboard
```
Create a tinkerdown dashboard that shows:
- [DATA SOURCE 1] from [API/file/command]
- [DATA SOURCE 2] from [API/file/command]
Display as [tables/lists/cards].
Refresh every [N] seconds.
```

### LLM Success Metrics

| Metric | Target | How to Measure |
|--------|--------|----------------|
| First-try success | >80% | App runs without edits |
| Data displays | 100% | Tables/lists populated |
| Forms work | 100% | Submit adds data |
| Self-documenting | Yes | Markdown explains purpose |

### Common LLM Mistakes to Avoid

```yaml
# DON'T: Complex nested sources
sources:
  items:
    type: markdown
    transforms:
      - filter: status != 'done'
      - sort: date desc
      - limit: 10

# DO: Simple flat sources
sources:
  items:
    type: markdown
    anchor: "#items"
```

```html
<!-- DON'T: Complex JavaScript -->
<script>
  const data = await fetch('/api/items');
  renderChart(data);
</script>

<!-- DO: Declarative lvt attributes -->
<table lvt-source="items" lvt-columns="name,status">
</table>
```

---

## Distribution & Sharing

### Personal Use

```
~/productivity/
├── habits.md
├── expenses.md
├── reading.md
└── .git/
```

Sync via git to any cloud (GitHub, GitLab, self-hosted).

### Team Use

```
team-productivity/
├── standups/
│   ├── 2025-01-06.md
│   └── 2025-01-07.md
├── retros/
│   └── sprint-42.md
├── templates/
│   ├── standup.md
│   └── retro.md
└── README.md
```

Share via:
- Git repo (private/public)
- `tinkerdown serve --host 0.0.0.0` for LAN
- `tinkerdown build` for static HTML

### Template Marketplace

```
community-templates/
├── personal/
│   ├── habit-tracker.md
│   ├── expense-tracker.md
│   └── reading-list.md
├── team/
│   ├── standup.md
│   ├── retro.md
│   └── okr.md
└── README.md
```

Users browse, copy, customize.

---

## Gaps & Required Features

### Must Have for Productivity

| Feature | Description | Priority |
|---------|-------------|----------|
| **Auto-timestamp** | Fill date/time on submit | P0 |
| **Operator identity** | Fill current user | P0 |
| **Aggregations** | Sum, count, avg by group | P1 |
| **Date filtering** | Today, this week, this month | P1 |
| **Computed fields** | budget - spent = remaining | P1 |
| **Rich text** | Multiline in table cells | P2 |

### Nice to Have

| Feature | Description | Priority |
|---------|-------------|----------|
| **Charts** | Simple bar/line charts | P2 |
| **Calendar view** | Dates as calendar | P2 |
| **Drag-and-drop** | Reorder, kanban boards | P2 |
| **Notifications** | Due date reminders | P3 |
| **Mobile view** | Responsive design | P2 |

### Already Exists

- ✅ Markdown data source (add/delete/edit)
- ✅ Exec source (run commands)
- ✅ REST source (fetch APIs)
- ✅ SQLite source (database)
- ✅ Auto-rendering tables/lists
- ✅ Forms with submit
- ✅ Button actions

---

## Roadmap

### Phase 1: Foundation (Already Done)

- ✅ Markdown data source
- ✅ CRUD operations
- ✅ Auto-rendering components
- ✅ Forms and actions

### Phase 2: Productivity Essentials

| Feature | Effort | Impact |
|---------|--------|--------|
| Auto-timestamp on submit | Small | High |
| Operator identity | Small | High |
| Filter by field value | Medium | High |
| Sort by field | Medium | Medium |

### Phase 3: Power Features

| Feature | Effort | Impact |
|---------|--------|--------|
| Aggregations (sum, count) | Medium | High |
| Computed fields | Medium | High |
| Date range filtering | Medium | Medium |
| Multiline table cells | Medium | Medium |

### Phase 4: Delight

| Feature | Effort | Impact |
|---------|--------|--------|
| Simple charts | Large | Medium |
| Calendar view | Large | Medium |
| Template gallery | Medium | Medium |
| Mobile responsive | Medium | Medium |

---

## Summary

### Why Productivity is the Killer Use Case

1. **Everyone needs it** - Universal problem
2. **Personal + Team** - Both audiences
3. **LLM-friendly** - Simple patterns, clear prompts
4. **Throwaway is OK** - Don't need forever apps
5. **Markdown fits** - Notes + interactivity = perfect
6. **Git fits** - Version control for personal data
7. **Low stakes** - Experiment freely

### The Pitch

```
Traditional:
"I need a habit tracker"
→ Research apps → Sign up → Configure → Maybe use → Data locked

Tinkerdown:
"I need a habit tracker"
→ Ask LLM → Get markdown file → tinkerdown serve → Use immediately
→ Data is yours → Edit anytime → Git backup → Free forever
```

### Next Steps

1. Build Phase 2 features (auto-timestamp, operator, filtering)
2. Create 10+ polished example apps
3. Write LLM prompt templates for each category
4. Build template gallery/marketplace
5. Create "Getting Started with Productivity" guide
