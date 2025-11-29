# Todo App with Persistence

This example demonstrates client-side persistence using localStorage.

## Server State

```server id="persistence-state"
package main

import (
    "github.com/livetemplate/livetemplate"
)

type Todo struct {
    ID        int    `json:"id"`
    Text      string `json:"text"`
    Completed bool   `json:"completed"`
}

type TodoState struct {
    Todos  []Todo `json:"todos"`
    NextID int    `json:"nextId"`
}

// ActiveCount returns the number of active (not completed) todos
func (s *TodoState) ActiveCount() int {
    count := 0
    for _, todo := range s.Todos {
        if !todo.Completed {
            count++
        }
    }
    return count
}

func NewTodoState() *TodoState {
    return &TodoState{
        Todos: []Todo{
            {ID: 1, Text: "This todo persists in localStorage", Completed: false},
            {ID: 2, Text: "Try refreshing the page after adding todos", Completed: false},
            {ID: 3, Text: "Your changes will persist!", Completed: false},
        },
        NextID: 4,
    }
}

func (s *TodoState) Change(ctx *livetemplate.ActionContext) error {
    switch ctx.Action {
    case "add":
        text := ctx.GetString("text")
        if text == "" {
            return nil
        }

        newTodo := Todo{
            ID:        s.NextID,
            Text:      text,
            Completed: false,
        }
        s.Todos = append(s.Todos, newTodo)
        s.NextID++

    case "toggle":
        id := ctx.GetInt("id")
        for i := range s.Todos {
            if s.Todos[i].ID == id {
                s.Todos[i].Completed = !s.Todos[i].Completed
                break
            }
        }

    case "delete":
        id := ctx.GetInt("id")
        for i, todo := range s.Todos {
            if todo.ID == id {
                s.Todos = append(s.Todos[:i], s.Todos[i+1:]...)
                break
            }
        }

    case "restore":
        // Restore state from localStorage
        var restored TodoState
        if err := ctx.Bind(&restored); err != nil {
            return err
        }
        s.Todos = restored.Todos
        s.NextID = restored.NextID
    }

    return nil
}
```

## Interactive UI

```lvt state="persistence-state"
<div class="todo-app">
    <h2>Persistent Todos</h2>

    <div class="persistence-info">
        <span class="info-icon">💾</span>
        <span>Changes are saved to localStorage automatically</span>
    </div>

    <form class="add-form" lvt-submit="add">
        <input
            type="text"
            name="text"
            placeholder="What needs to be done?"
            autocomplete="off"
        >
        <button type="submit">Add Todo</button>
    </form>

    {{$totalCount := len .Todos}}
    {{$activeCount := .ActiveCount}}
    <div class="stats-row">
        <div class="stat">
            <span class="stat-label">Total</span>
            <span class="stat-value">{{$totalCount}}</span>
        </div>
        <div class="stat">
            <span class="stat-label">Active</span>
            <span class="stat-value">{{$activeCount}}</span>
        </div>
    </div>

    {{if eq (len .Todos) 0}}
    <div class="empty-state">
        <p>No todos yet!</p>
        <p class="hint">Add one above to get started 👆</p>
    </div>
    {{else}}
    <ul class="todo-list">
        {{range .Todos}}
        <li class="todo-item {{if .Completed}}completed{{end}}">
            <input
                type="checkbox"
                id="todo-{{.ID}}"
                {{if .Completed}}checked{{end}}
                lvt-click="toggle"
                lvt-data-id="{{.ID}}"
            >
            <label for="todo-{{.ID}}">
                <span class="todo-text">{{.Text}}</span>
            </label>
            <button
                class="delete-btn"
                lvt-click="delete"
                lvt-data-id="{{.ID}}"
                aria-label="Delete todo"
            >
                ×
            </button>
        </li>
        {{end}}
    </ul>
    {{end}}
</div>

<script>
// Persistence logic - runs after state updates
(function() {
    const STORAGE_KEY = 'livepage-todos-persistence';

    // Save state to localStorage after each update
    function saveState(state) {
        try {
            localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
            console.log('State saved to localStorage:', state);
        } catch (e) {
            console.error('Failed to save state:', e);
        }
    }

    // Load state from localStorage on page load
    function loadState() {
        try {
            const saved = localStorage.getItem(STORAGE_KEY);
            if (saved) {
                const state = JSON.parse(saved);
                console.log('Restoring state from localStorage:', state);

                // Send restore action to server
                fetch('/livepage/action', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        blockId: 'persistence-state',
                        action: 'restore',
                        data: state
                    })
                }).then(() => {
                    // Reload page to show restored state
                    window.location.reload();
                });
            }
        } catch (e) {
            console.error('Failed to load state:', e);
        }
    }

    // Listen for state updates from WebSocket
    if (window.LivePage) {
        const originalOnMessage = window.LivePage.ws.onmessage;
        window.LivePage.ws.onmessage = function(event) {
            try {
                const msg = JSON.parse(event.data);
                if (msg.type === 'update' && msg.blockId === 'persistence-state') {
                    // Save the updated state
                    saveState(msg.state);
                }
            } catch (e) {
                console.error('Failed to parse WebSocket message:', e);
            }

            // Call original handler
            if (originalOnMessage) {
                originalOnMessage.call(this, event);
            }
        };
    }

    // Load state on page load
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', loadState);
    } else {
        loadState();
    }
})();
</script>

<style>
.todo-app {
    max-width: 600px;
    margin: 2rem auto;
    padding: 2rem;
    background: #f9f9f9;
    border-radius: 8px;
    box-shadow: 0 2px 8px rgba(0,0,0,0.1);
}

.persistence-info {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.75rem 1rem;
    background: #dbeafe;
    border-left: 3px solid #3b82f6;
    border-radius: 4px;
    margin-bottom: 1.5rem;
    font-size: 0.9rem;
    color: #1e40af;
}

.info-icon {
    font-size: 1.2rem;
}

.add-form {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 1.5rem;
}

.add-form input {
    flex: 1;
    padding: 0.75rem 1rem;
    border: 2px solid #e5e7eb;
    border-radius: 4px;
    font-size: 1rem;
}

.add-form input:focus {
    outline: none;
    border-color: #3b82f6;
}

.add-form button {
    padding: 0.75rem 1.5rem;
    background: #3b82f6;
    color: white;
    border: none;
    border-radius: 4px;
    font-size: 1rem;
    font-weight: 600;
    cursor: pointer;
    transition: background 0.2s;
}

.add-form button:hover {
    background: #2563eb;
}

.stats-row {
    display: flex;
    gap: 1rem;
    margin-bottom: 1.5rem;
    padding: 1rem;
    background: white;
    border-radius: 4px;
}

.stat {
    flex: 1;
    text-align: center;
}

.stat-label {
    display: block;
    font-size: 0.85rem;
    color: #666;
    margin-bottom: 0.25rem;
}

.stat-value {
    display: block;
    font-size: 1.5rem;
    font-weight: bold;
    color: #3b82f6;
}

.empty-state {
    text-align: center;
    padding: 3rem 1rem;
    color: #999;
}

.todo-list {
    list-style: none;
    padding: 0;
    margin: 0;
}

.todo-item {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 1rem;
    background: white;
    border-radius: 4px;
    border-left: 4px solid #cbd5e1;
    margin-bottom: 0.5rem;
    transition: all 0.2s;
}

.todo-item.completed {
    border-left-color: #10b981;
    background: #f0fdf4;
}

.todo-item input[type="checkbox"] {
    width: 20px;
    height: 20px;
    cursor: pointer;
}

.todo-item label {
    flex: 1;
    cursor: pointer;
}

.todo-text {
    font-size: 1.05rem;
}

.todo-item.completed .todo-text {
    text-decoration: line-through;
    color: #999;
}

.delete-btn {
    width: 32px;
    height: 32px;
    border: none;
    background: #fee2e2;
    color: #dc2626;
    border-radius: 4px;
    font-size: 1.5rem;
    line-height: 1;
    cursor: pointer;
    transition: all 0.2s;
    display: flex;
    align-items: center;
    justify-content: center;
}

.delete-btn:hover {
    background: #fecaca;
    transform: scale(1.1);
}
</style>
```
