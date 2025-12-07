---
title: "{{.Title}}"
type: tutorial
persist: localstorage
steps: 3
---

# Welcome to {{.Title}}

This is a basic LiveTemplate tutorial to get you started.

## What is LiveTemplate?

LiveTemplate is a framework where **state lives on the server**. This means:
- ✅ State is secure and trusted (server-side)
- ✅ Business logic runs in Go
- ✅ Real-time updates via WebSocket
- ✅ Simple, declarative templates

## Step 1: Define Your State

First, let's define our application state in Go:

```go server
// State holds the application data on the server
type State struct {
	Message string `json:"message"`
	Count   int    `json:"count"`
}

// Increment handles the "increment" action
func (s *State) Increment(_ *livetemplate.Context) error {
	s.Count++
	return nil
}

// UpdateMessage handles the "update-message" action
func (s *State) UpdateMessage(ctx *livetemplate.Context) error {
	if msg, ok := ctx.Data["message"].(string); ok {
		s.Message = msg
	}
	return nil
}
```

> 💡 **Key Concept**: Each action has its own method (e.g., `Increment`, `UpdateMessage`). Method names match action names. This code runs on the server, keeping your business logic secure.

## Step 2: Build Your UI

Now let's create an interactive UI using Go templates:

```lvt
<div class="demo">
	<h2>Message: {{.Message}}</h2>
	<p>Count: {{.Count}}</p>

	<div class="controls">
		<button lvt-click="increment">Increment</button>
		<input
			type="text"
			value="{{.Message}}"
			lvt-change="update-message"
			lvt-change-data='{"message": "$value"}'
		/>
	</div>
</div>

<style>
.demo {
	padding: 2rem;
	border: 1px solid #ddd;
	border-radius: 8px;
	max-width: 500px;
	margin: 2rem auto;
}

.demo h2 {
	margin-top: 0;
	color: #333;
}

.controls {
	display: flex;
	gap: 1rem;
	margin-top: 1rem;
}

button {
	padding: 0.5rem 1rem;
	border: none;
	background: #007bff;
	color: white;
	border-radius: 4px;
	cursor: pointer;
	font-size: 1rem;
}

button:hover {
	background: #0056b3;
}

input {
	flex: 1;
	padding: 0.5rem;
	border: 1px solid #ddd;
	border-radius: 4px;
	font-size: 1rem;
}
</style>
```

> 💡 **Template Syntax**: Use `lvt-click` for button actions and `lvt-change` for input changes. The server automatically re-renders and pushes updates.

## Step 3: Try It Out!

Click the button and type in the input field to see real-time updates!

## How It Works

1. **User interacts** → Browser sends WebSocket message
2. **Server receives action** → Calls `Change()` method
3. **State updates** → Server modifies the state
4. **Re-render** → Server renders template with new state
5. **Push update** → New HTML sent to browser
6. **UI updates** → Browser displays changes instantly

## Next Steps

Now that you understand the basics:

- **Add validation** - Check inputs in the `Change()` method
- **Add more actions** - Handle different user interactions
- **Style it** - Customize the CSS to match your design
- **Explore advanced features** - Learn about persistence, routing, and more

Happy coding with LiveTemplate! 🚀
