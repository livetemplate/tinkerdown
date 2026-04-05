# Tinkerdown

## AI builds tools in one file. No React. No build step. Just run.

---

### The Problem

Every AI app builder today outputs the same thing: a React project with 10+ files, npm dependencies, and a build pipeline. For a simple admin panel or internal tool, that's massive overkill.

**You asked for a todo app. You got:**

```
package.json
src/App.tsx
src/components/TodoList.tsx
src/components/TodoItem.tsx
src/hooks/useTodos.ts
tailwind.config.js
...
```

Now you need to `npm install`, configure your environment, set up hosting, and debug build errors. All for a simple tool.

---

### The Solution

**Tinkerdown outputs a single markdown file that just works.**

```markdown
# Todo App

<form name="add" lvt-persist="todos">
  <input name="task" placeholder="New task">
  <button>Add</button>
</form>

<div lvt-source="todos">
{{range .}}
  <li>{{.task}} <button name="delete" data-id="{{.id}}">×</button></li>
{{end}}
</div>
```

**One file. 12 lines. Run it:**

```bash
tinkerdown serve todo.md
```

**That's it.** No npm. No build. No deploy. It just works.

---

### How It Works

| You Write | Tinkerdown Handles |
|-----------|------------------|
| `lvt-persist="todos"` | Database table, CRUD operations, persistence |
| `lvt-source="pg:users"` | PostgreSQL queries, real-time updates |
| `name="delete"` (on button) | WebSocket actions, server-side logic |
| `{{template "lvt:datatable"}}` | Production-ready UI components |

**The Declarative Trifecta:**

- **Components** — Pre-built UI (datatables, dropdowns, modals)
- **Sources** — Connect to PostgreSQL, REST APIs, CSV files, or any script
- **Persistence** — Auto-generated CRUD with zero backend code

---

### Why Tinkerdown?

| | React Builders | Tinkerdown |
|--|----------------|----------|
| **Output** | 10+ files | 1 file |
| **Build step** | Required | None |
| **Hosting** | Required | Single binary |
| **Time to working app** | Minutes | Seconds |
| **Self-hosted** | Complex | `./tinkerdown serve` |
| **Cost** | $25-50/mo | Free & open source |

---

### Use Any Language

Don't know Go? No problem. Tinkerdown's "polyglot" mode lets you use **any language** as a data source:

```yaml
# tinkerdown.yaml
sources:
  sales:
    type: exec
    cmd: python scripts/fetch_sales.py
```

```html
<div lvt-source="sales">
  {{range .}}
    <tr><td>{{.product}}</td><td>{{.revenue}}</td></tr>
  {{end}}
</div>
```

**Build a UI for your Python, Bash, or Node scripts in 30 seconds.**

---

### Perfect For

- **Internal tools** — Admin panels, dashboards, data viewers
- **Quick utilities** — One-off tools that don't need a full stack
- **AI-generated apps** — LLMs excel at single-file formats
- **Self-hosted tools** — Data stays on your machine, no SaaS required

---

### Get Started

```bash
# Install
go install github.com/livetemplate/livemdtools@latest

# Create your first app
echo '# Hello World

```lvt
<h1>Hello, {{.Name}}!</h1>
<form name="greet">
  <input name="name" placeholder="Your name">
  <button>Greet</button>
</form>
```' > hello.md

# Run it
tinkerdown serve hello.md
```

Open `http://localhost:8080` and you have a working app.

---

### Links

- **GitHub**: github.com/livetemplate/livemdtools
- **Docs**: livemdtools.dev/docs
- **Examples**: livemdtools.dev/examples

---

*Tinkerdown is open source (Apache 2.0). Built with Go.*

**Stop building React apps for simple tools. Just write a Tinkerdown.**
