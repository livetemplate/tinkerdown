---
title: "<<.Title>>"
sources:
  products:
    type: csv
    file: products.csv
---

# <<.Title>>

A product inventory loaded from a CSV file.

## Product Inventory

```lvt
<table lvt-source="products" lvt-columns="id:ID,name:Product,category:Category,price:Price,stock:In Stock" lvt-empty="No products found.">
</table>
```

---

## How It Works

This app uses the **CSV file source** to load data from `products.csv`.

The `lvt-columns` attribute auto-renders the table — no Go template needed:

```yaml
sources:
  products:
    type: csv
    file: products.csv
```

### Adding Data

Edit `products.csv` directly to add or modify products. Changes appear on the next page load.

### Column Mapping

The `lvt-columns` format is `field:Label` — the field name comes from the CSV header row:

```
lvt-columns="id:ID,name:Product,category:Category,price:Price,stock:In Stock"
```
