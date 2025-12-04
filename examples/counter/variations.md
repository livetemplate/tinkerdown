---
title: "Counter Variations - LiveTemplate Patterns"
type: tutorial
persist: localstorage
---

# Counter Variations: Common LiveTemplate Patterns

This tutorial demonstrates different counter patterns to teach key LiveTemplate concepts through practical examples.

## Basic Counter

Let's start with the simple counter from the main tutorial:

```go server id="basic-counter"
type BasicCounterState struct {
    Counter int `json:"counter"`
}

func (s *BasicCounterState) Increment(_ *livetemplate.ActionContext) error {
    s.Counter++
    return nil
}

func (s *BasicCounterState) Decrement(_ *livetemplate.ActionContext) error {
    s.Counter--
    return nil
}

func (s *BasicCounterState) Reset(_ *livetemplate.ActionContext) error {
    s.Counter = 0
    return nil
}
```

```lvt state="basic-counter"
<div class="counter-display {{if gt .Counter 0}}positive{{else if lt .Counter 0}}negative{{else}}zero{{end}}">
    {{.Counter}}
</div>
<div class="button-group">
    <button lvt-click="increment">+1</button>
    <button lvt-click="decrement">-1</button>
    <button lvt-click="reset">Reset</button>
</div>
```

## Bounded Counter (Validation)

This counter demonstrates **server-side validation** with minimum (0) and maximum (100) limits. Notice how the server enforces boundaries - client code can't bypass this!

```go server id="bounded-counter"
type BoundedCounterState struct {
    Counter int `json:"counter"`
    Min     int `json:"min"`
    Max     int `json:"max"`
}

func (s *BoundedCounterState) Init() error {
    // Initialize boundaries
    if s.Min == 0 && s.Max == 0 {
        s.Min = 0
        s.Max = 100
    }
    return nil
}

func (s *BoundedCounterState) Increment(_ *livetemplate.ActionContext) error {
    if s.Counter < s.Max {
        s.Counter++
    }
    return nil
}

func (s *BoundedCounterState) Decrement(_ *livetemplate.ActionContext) error {
    if s.Counter > s.Min {
        s.Counter--
    }
    return nil
}

func (s *BoundedCounterState) Reset(_ *livetemplate.ActionContext) error {
    s.Counter = s.Min
    return nil
}
```

```lvt state="bounded-counter"
<div class="counter-container">
    <div class="counter-header">
        <span class="bounds-label">Min: {{.Min}}</span>
        <span class="bounds-label">Max: {{.Max}}</span>
    </div>
    <div class="counter-display {{if eq .Counter .Max}}at-max{{else if eq .Counter .Min}}at-min{{else}}in-range{{end}}">
        {{.Counter}}
    </div>
    <div class="button-group">
        <button lvt-click="decrement" {{if eq .Counter .Min}}disabled{{end}}>-1</button>
        <button lvt-click="reset">Reset</button>
        <button lvt-click="increment" {{if eq .Counter .Max}}disabled{{end}}>+1</button>
    </div>
    <div class="bounds-bar">
        <div class="bounds-progress" style="width: {{.Counter}}%"></div>
    </div>
</div>
```

**Key Concept**: Validation happens on the server. The client UI shows disabled buttons for better UX, but the real enforcement is server-side.

## Step Counter (Action Parameters)

This counter shows how to handle **different action magnitudes** - increment by 1, 5, or 10.

```go server id="step-counter"
type StepCounterState struct {
    Counter int `json:"counter"`
}

// Add1 handles "add-1" action (hyphen converted to camelCase)
func (s *StepCounterState) Add1(_ *livetemplate.ActionContext) error {
    s.Counter += 1
    return nil
}

func (s *StepCounterState) Add5(_ *livetemplate.ActionContext) error {
    s.Counter += 5
    return nil
}

func (s *StepCounterState) Add10(_ *livetemplate.ActionContext) error {
    s.Counter += 10
    return nil
}

func (s *StepCounterState) Subtract1(_ *livetemplate.ActionContext) error {
    s.Counter -= 1
    return nil
}

func (s *StepCounterState) Subtract5(_ *livetemplate.ActionContext) error {
    s.Counter -= 5
    return nil
}

func (s *StepCounterState) Subtract10(_ *livetemplate.ActionContext) error {
    s.Counter -= 10
    return nil
}

func (s *StepCounterState) Reset(_ *livetemplate.ActionContext) error {
    s.Counter = 0
    return nil
}
```

```lvt state="step-counter"
<div class="counter-display">
    {{.Counter}}
</div>
<div class="step-buttons">
    <div class="button-row">
        <span class="row-label">Add:</span>
        <button lvt-click="add-1" class="step-btn">+1</button>
        <button lvt-click="add-5" class="step-btn">+5</button>
        <button lvt-click="add-10" class="step-btn">+10</button>
    </div>
    <div class="button-row">
        <span class="row-label">Subtract:</span>
        <button lvt-click="subtract-1" class="step-btn">-1</button>
        <button lvt-click="subtract-5" class="step-btn">-5</button>
        <button lvt-click="subtract-10" class="step-btn">-10</button>
    </div>
    <button lvt-click="reset" class="reset-btn">Reset</button>
</div>
```

**Key Concept**: Each action is a separate command. You can also pass parameters via `lvt-data` for more dynamic behavior.

## Dual Counter (State Isolation)

Two completely independent counters showing **state isolation** - each has its own server-side state instance.

```go server id="dual-counter-a"
type DualCounterStateA struct {
    Counter int    `json:"counter"`
    Label   string `json:"label"`
}

func (s *DualCounterStateA) Init() error {
    if s.Label == "" {
        s.Label = "Counter A"
    }
    return nil
}

func (s *DualCounterStateA) Increment(_ *livetemplate.ActionContext) error {
    s.Counter++
    return nil
}

func (s *DualCounterStateA) Decrement(_ *livetemplate.ActionContext) error {
    s.Counter--
    return nil
}

func (s *DualCounterStateA) Reset(_ *livetemplate.ActionContext) error {
    s.Counter = 0
    return nil
}
```

```go server id="dual-counter-b"
type DualCounterStateB struct {
    Counter int    `json:"counter"`
    Label   string `json:"label"`
}

func (s *DualCounterStateB) Init() error {
    if s.Label == "" {
        s.Label = "Counter B"
    }
    return nil
}

func (s *DualCounterStateB) Increment(_ *livetemplate.ActionContext) error {
    s.Counter++
    return nil
}

func (s *DualCounterStateB) Decrement(_ *livetemplate.ActionContext) error {
    s.Counter--
    return nil
}

func (s *DualCounterStateB) Reset(_ *livetemplate.ActionContext) error {
    s.Counter = 0
    return nil
}
```

<div class="dual-counter-container">
<div class="dual-counter-item">

```lvt state="dual-counter-a"
<div class="counter-label">{{.Label}}</div>
<div class="counter-display {{if gt .Counter 0}}positive{{else if lt .Counter 0}}negative{{else}}zero{{end}}">
    {{.Counter}}
</div>
<div class="button-group">
    <button lvt-click="decrement">-</button>
    <button lvt-click="reset">Reset</button>
    <button lvt-click="increment">+</button>
</div>
```

</div>
<div class="dual-counter-item">

```lvt state="dual-counter-b"
<div class="counter-label">{{.Label}}</div>
<div class="counter-display {{if gt .Counter 0}}positive{{else if lt .Counter 0}}negative{{else}}zero{{end}}">
    {{.Counter}}
</div>
<div class="button-group">
    <button lvt-click="decrement">-</button>
    <button lvt-click="reset">Reset</button>
    <button lvt-click="increment">+</button>
</div>
```

</div>
</div>

**Key Concept**: Each server block creates a separate state instance. Counter A and Counter B don't share state - they're completely isolated.

## Shopping Cart Quantity (Real-World Example)

A practical example: product quantity selector like you'd see in an e-commerce site.

```go server id="shopping-cart"
type ProductState struct {
    ProductName string `json:"productName"`
    Quantity    int    `json:"quantity"`
    Price       float64 `json:"price"`
    Total       float64 `json:"total"`
}

func (s *ProductState) Init() error {
    // Initialize product
    if s.ProductName == "" {
        s.ProductName = "LiveTemplate T-Shirt"
        s.Price = 29.99
        s.Quantity = 1
    }
    s.Total = float64(s.Quantity) * s.Price
    return nil
}

func (s *ProductState) Increase(_ *livetemplate.ActionContext) error {
    if s.Quantity < 10 {
        s.Quantity++
    }
    s.Total = float64(s.Quantity) * s.Price
    return nil
}

func (s *ProductState) Decrease(_ *livetemplate.ActionContext) error {
    if s.Quantity > 1 {
        s.Quantity--
    }
    s.Total = float64(s.Quantity) * s.Price
    return nil
}

func (s *ProductState) Remove(_ *livetemplate.ActionContext) error {
    s.Quantity = 0
    s.Total = 0
    return nil
}
```

```lvt state="shopping-cart"
<div class="product-card">
    <div class="product-header">
        <h4>{{.ProductName}}</h4>
        <span class="product-price">${{printf "%.2f" .Price}}</span>
    </div>

    {{if gt .Quantity 0}}
    <div class="quantity-selector">
        <span class="quantity-label">Quantity:</span>
        <div class="quantity-controls">
            <button lvt-click="decrease" class="qty-btn" {{if eq .Quantity 1}}disabled{{end}}>−</button>
            <span class="quantity-display">{{.Quantity}}</span>
            <button lvt-click="increase" class="qty-btn" {{if eq .Quantity 10}}disabled{{end}}>+</button>
        </div>
    </div>

    <div class="product-total">
        <span class="total-label">Subtotal:</span>
        <span class="total-amount">${{printf "%.2f" .Total}}</span>
    </div>

    <button lvt-click="remove" class="remove-btn">Remove from Cart</button>
    {{else}}
    <div class="removed-message">
        Item removed from cart
    </div>
    {{end}}
</div>
```

**Key Concept**: Real applications combine state, validation, and calculations. The server manages quantity limits and computes totals - the client just displays and sends actions.

## Pattern Summary

| Pattern | Teaches | Use When |
|---------|---------|----------|
| **Basic Counter** | Core concepts | Learning LiveTemplate basics |
| **Bounded Counter** | Validation | Need min/max limits, server-side rules |
| **Step Counter** | Multiple actions | Different operation magnitudes |
| **Dual Counter** | State isolation | Multiple independent components |
| **Shopping Cart** | Real-world app | Combining state + validation + calculations |

## Key Takeaways

✅ **Server-side validation** - Boundaries and rules enforced where they can't be bypassed
✅ **Multiple actions** - Different buttons can trigger different server behaviors
✅ **State isolation** - Each component has independent server-side state
✅ **Calculated fields** - Server computes derived values (like totals)
✅ **Conditional rendering** - Templates adapt based on state

Try modifying these examples to experiment with LiveTemplate patterns!
