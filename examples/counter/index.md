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

```go server readonly id="counter-state"
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

```lvt interactive state="counter-state"
<div class="counter-demo">
    <h2>Count: {{.Counter}}</h2>
    <div class="controls">
        <button lvt-click="increment">+1</button>
        <button lvt-click="decrement">-1</button>
        <button lvt-click="reset">Reset</button>
    </div>
</div>
```

*Note: Full interactivity coming soon!*

## Step 3: Try It Yourself

Modify this Go code and click "Run" to see the output:

```go wasm editable
package main

import "fmt"

func main() {
    for i := 0; i < 5; i++ {
        fmt.Printf("Count: %d\n", i)
    }
}
```

*Note: WASM execution coming soon!*

## Next Steps

- Learn about [broadcasting](/advanced/broadcasting)
- Explore [form handling](/guides/forms)
- Build a [todo app](/tutorials/todos)
