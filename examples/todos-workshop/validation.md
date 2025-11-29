# Todo App with Validation

This example demonstrates server-side validation using the `go-playground/validator` library.

## Server State

```go server id="validation-state"
package main

import (
    "github.com/go-playground/validator/v10"
    "github.com/livetemplate/livetemplate"
)

// Create a validator instance (reuse across requests)
var validate = validator.New()

type Todo struct {
    ID        int    `json:"id"`
    Text      string `json:"text"`
    Completed bool   `json:"completed"`
}

// TodoInput defines the shape of data when adding a todo
// This is separate from Todo because we validate before assigning an ID
type TodoInput struct {
    Text string `json:"text" validate:"required,min=3,max=100"`
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
            {ID: 1, Text: "Try adding an empty todo", Completed: false},
            {ID: 2, Text: "Try adding a todo with only 2 characters", Completed: false},
            {ID: 3, Text: "Try adding a todo with more than 100 characters to see validation in action!", Completed: false},
        },
        NextID: 4,
    }
}

func (s *TodoState) Change(ctx *livetemplate.ActionContext) error {
    switch ctx.Action {
    case "add":
        // Bind and validate the input
        var input TodoInput
        if err := ctx.BindAndValidate(&input, validate); err != nil {
            return err // Returns validation errors to client
        }

        // Validation passed! Create the todo
        newTodo := Todo{
            ID:        s.NextID,
            Text:      input.Text,
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
    }

    return nil
}
```

## Interactive UI

```lvt state="validation-state"
<div class="todo-app">
    <h2>Validated Todos</h2>

    {{with .ValidationError}}
    <div class="error-banner">{{.}}</div>
    {{end}}

    <form class="add-form" lvt-submit="add">
        <div class="form-group">
            <input
                type="text"
                name="text"
                placeholder="What needs to be done? (3-100 chars)"
                autocomplete="off"
                aria-invalid="{{if .ValidationError}}true{{else}}false{{end}}"
                aria-describedby="{{if .ValidationError}}error-message{{end}}"
            >
            {{with .ValidationError}}
            <div class="error-message" id="error-message">
                <span class="error-icon">⚠️</span>
                <span>{{.}}</span>
            </div>
            {{end}}
        </div>
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

<style>
.todo-app {
    max-width: 600px;
    margin: 2rem auto;
    padding: 2rem;
    background: #f9f9f9;
    border-radius: 8px;
    box-shadow: 0 2px 8px rgba(0,0,0,0.1);
}

.add-form {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 1.5rem;
}

.form-group {
    flex: 1;
    position: relative;
}

.form-group input {
    width: 100%;
    padding: 0.75rem 1rem;
    border: 2px solid #e5e7eb;
    border-radius: 4px;
    font-size: 1rem;
    box-sizing: border-box;
}

.form-group input:focus {
    outline: none;
    border-color: #3b82f6;
}

.form-group input[aria-invalid="true"] {
    border-color: #dc2626;
    background: #fef2f2;
}

.error-message {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-top: 0.5rem;
    padding: 0.5rem;
    background: #fee2e2;
    border-left: 3px solid #dc2626;
    border-radius: 4px;
    font-size: 0.9rem;
    color: #991b1b;
}

.error-icon {
    font-size: 1.1rem;
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
    white-space: nowrap;
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
