---
title: "Counter Tutorial"
type: tutorial
persist: localstorage
steps: 3
---

# Build a Counter with LiveTemplate

Learn how to build a reactive counter application using LiveTemplate.

## What You'll Build

By the end of this tutorial, you'll have a working counter that updates in real-time.

## Step 1: Understanding State

In LiveTemplate, application state is stored in Go structs. Here's our counter state:

```go server
type CounterState struct {
    Counter int `json:"counter"`
}

func (s *CounterState) Change(ctx *livetemplate.ActionContext) error {
    switch ctx.Action {
    case "increment":
        s.Counter++
    case "decrement":
        s.Counter--
    case "reset":
        s.Counter = 0
    }
    return nil
}
```

This code runs on the server and handles all state changes.

## Step 2: Interactive Demo

The counter below is powered by the state above:

```lvt
<div class="counter-demo">
    <h2>Count: {{.Counter}}</h2>
    <div class="controls">
        <button lvt-click="increment">+1</button>
        <button lvt-click="decrement">-1</button>
        <button lvt-click="reset">Reset</button>
    </div>
</div>
```

Click the buttons above to interact with the counter! The state is managed on the server and updates are sent in real-time over WebSocket.

## How It Works

1. **User clicks a button** - The client sends an action (e.g., "increment") to the server
2. **Server updates state** - The `Change` method updates the counter value
3. **Server sends update** - The new state is sent back to the client
4. **UI updates** - The counter display updates automatically

## Next Steps

- Learn about [broadcasting](/advanced/broadcasting)
- Explore [form handling](/guides/forms)
- Build a [todo app](/tutorials/todos)
