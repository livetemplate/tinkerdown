---
title: "Counter Tutorial"
type: tutorial
persist: localstorage
steps: 3
---

# Build a Counter with LiveTemplate

Learn how to build a reactive counter application using LiveTemplate - a framework where **state lives on the server**.

## Why LiveTemplate?

Traditional JavaScript frameworks keep state in the browser, which means:
- ❌ State can be manipulated by users
- ❌ Business logic runs in untrusted environment
- ❌ Validation happens client-side (can be bypassed)

LiveTemplate flips this around:
- ✅ State lives securely on the server
- ✅ Business logic runs in trusted Go code
- ✅ Validation is server-side and reliable
- ✅ Real-time updates via WebSocket

> 💡 **Key Concept**: In LiveTemplate, the server is the source of truth. The browser just displays what the server tells it to display.

## What You'll Build

By the end of this tutorial, you'll have a working counter that:
- Increments and decrements a number
- Shows visual feedback (green for positive, red for negative)
- Updates in real-time over WebSocket
- Persists state in localStorage

## Step 1: Define Your State

In LiveTemplate, application state is stored in **Go structs**. Here's our counter state:

```go server
// CounterState holds the application state on the server
type CounterState struct {
    Counter int `json:"counter"` // The current count value
}

// Change handles all state mutations
// This method is called whenever a user action is received
func (s *CounterState) Change(ctx *livetemplate.ActionContext) error {
    switch ctx.Action {
    case "increment":
        s.Counter++  // Increase by 1
    case "decrement":
        s.Counter--  // Decrease by 1
    case "reset":
        s.Counter = 0  // Reset to zero
    }
    return nil
}
```

> 💡 **Important**: This code runs **on the server**, not in the browser. Your business logic is secure and can't be manipulated by users.

**What's happening here:**
- `CounterState` struct holds our data
- `Change()` method handles all state mutations
- `ctx.Action` tells us which button was clicked
- Server-side validation can be added in `Change()` (e.g., min/max limits)

## Step 2: Build Your UI

The counter below is powered by the state above. Notice how the template uses Go's `template` syntax:

```lvt
<div class="counter-display {{if gt .Counter 0}}positive{{else if lt .Counter 0}}negative{{else}}zero{{end}}">
    {{.Counter}}
</div>
<div class="button-group">
    <button lvt-click="increment">+1</button>
    <button lvt-click="decrement">-1</button>
    <button lvt-click="reset">Reset</button>
</div>
```

> 💡 **Template Syntax**: `.Counter` references the `Counter` field from your `CounterState` struct. The server renders this template whenever state changes.

**UI Features:**
- `lvt-click="increment"` - sends "increment" action to server
- Conditional CSS classes show state visually
- Server re-renders and pushes updates automatically

## Step 3: See It In Action

Click the buttons below to interact with the counter! Open your **browser's DevTools** (F12 or right-click → Inspect) and go to the **Console** or **Network** tab to watch the WebSocket messages and state updates in real-time.

## How It Works

Here's the complete flow when you click a button:

1. **User clicks a button** → Browser sends WebSocket message with action name
2. **Server receives action** → Calls `Change()` method with the action
3. **State is updated** → Server increments/decrements the counter
4. **Template is rendered** → Server renders HTML with new state
5. **Update is pushed** → New HTML sent to browser via WebSocket
6. **UI updates instantly** → Browser displays the new counter value

> 🔍 **Try This**: Open browser DevTools (F12), go to the Network tab, filter by "WS" (WebSocket), and click the buttons. You'll see the action messages and state updates flowing between client and server in real-time!

## Key Takeaways

✅ **Server-side state** - Your data is secure and trusted
✅ **Go templates** - Familiar syntax, server-side rendering
✅ **Real-time updates** - WebSocket keeps UI in sync
✅ **Simple actions** - `lvt-click` sends action names to server
✅ **Auto-linking** - No manual wiring between state and UI

## Next Steps

Now that you understand the basics, explore:

- **🎨 Styling** - Notice how we use CSS classes for visual feedback?
- **🔄 Multiple counters** - Try adding a second counter with isolated state
- **✅ Validation** - Add min/max limits in the `Change()` method
- **📊 Advanced** - Learn about [broadcasting](/advanced/broadcasting) to sync state across users

Ready to build something bigger? Try our [todo app tutorial](/tutorials/todos)!
