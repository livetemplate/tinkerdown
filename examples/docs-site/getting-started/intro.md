---
title: "Introduction to LivePage"
---

# Introduction to LivePage

LivePage is a revolutionary framework for building interactive documentation and tutorials.

## Core Concepts

### Server-Side State

Unlike traditional JavaScript frameworks, LivePage keeps all state on the server. This means:

- State cannot be manipulated by users
- Business logic runs in a trusted environment
- Validation is reliable and secure

### Real-Time Updates

LivePage uses WebSockets to push updates from server to client instantly. When state changes, the server re-renders the affected components and pushes the updates to all connected clients.

### Go Templates

Write your UI using Go's built-in `html/template` syntax. No JSX, no build step required.

## Next Steps

Ready to get started? Head over to [Installation](/getting-started/installation) to set up your development environment.
