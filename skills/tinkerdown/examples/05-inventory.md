---
title: "Inventory Manager"
---

# Inventory Manager

An inventory system demonstrating PostgreSQL integration with `lvt-source`.

**Features demonstrated:**
- `lvt-source` with PostgreSQL
- Number inputs
- CRUD operations
- Conditional styling for low stock
- **No CSS classes needed** - PicoCSS styles semantic HTML automatically

**Configuration (tinkerdown.yaml):**
```yaml
title: "Inventory Manager"

sources:
  products:
    type: pg
    query: SELECT id, name, sku, quantity, price FROM products ORDER BY name
```

**Required environment:**
```bash
export DATABASE_URL="postgres://user:pass@localhost:5432/inventory"
```

```lvt
<main>
    <h1>Inventory Manager</h1>

    <!-- Add Product Form -->
    <article>
        <header>Add Product</header>
        <form name="save" lvt-persist="products">
            <fieldset role="group">
                <input type="text" name="name" required placeholder="Product name">
                <input type="text" name="sku" required placeholder="SKU-001">
            </fieldset>
            <fieldset role="group">
                <input type="number" name="quantity" required min="0" value="0" placeholder="Quantity">
                <input type="number" name="price" required min="0" step="0.01" placeholder="Price ($)">
            </fieldset>
            <button type="submit">Add Product</button>
        </form>
    </article>

    <!-- Products Table -->
    <article>
        <header>
            <span>Products</span>
            <button name="Refresh" class="outline">Refresh</button>
        </header>

        {{if .Products}}
        <table>
            <thead>
                <tr>
                    <th>Name</th>
                    <th>SKU</th>
                    <th>Quantity</th>
                    <th>Price</th>
                    <th>Actions</th>
                </tr>
            </thead>
            <tbody>
                {{range .Products}}
                <tr>
                    <td>{{.Name}}</td>
                    <td><code>{{.Sku}}</code></td>
                    <td>{{if lt .Quantity 10}}<mark>{{.Quantity}}</mark>{{else}}{{.Quantity}}{{end}}</td>
                    <td>${{.Price}}</td>
                    <td>
                        <button name="Delete" data-id="{{.Id}}" >Delete</button>
                    </td>
                </tr>
                {{end}}
            </tbody>
        </table>
        {{else}}
        <p><em>No products in inventory. Add one above!</em></p>
        {{end}}
    </article>
</main>
```

## How It Works

1. **PostgreSQL source** - Define query in `tinkerdown.yaml`, reference with `lvt-source`
2. **Number inputs** - `type="number"` with `min`, `step` attributes
3. **Low stock warning** - Conditional styling with `{{if lt .Quantity 10}}` using `<mark>` tag
4. **CRUD** - `lvt-persist` auto-generates Save/Delete actions

## Prompt to Generate This

> Build an inventory manager with Livemdtools. Connect to PostgreSQL for products. Show a table with name, SKU, quantity, price. Highlight low stock (under 10). Include add/delete functionality. Use semantic HTML.
