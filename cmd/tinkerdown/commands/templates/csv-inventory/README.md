# <<.Title>>

A product inventory app built with [Tinkerdown](https://github.com/livetemplate/tinkerdown).

## Quick Start

```bash
cd <<.ProjectName>>
tinkerdown serve
```

Open http://localhost:8080 in your browser.

## Features

- CSV file as data source (no database needed)
- Auto-rendered table via `lvt-columns`
- Edit `products.csv` to update inventory

## Project Structure

```
<<.ProjectName>>/
├── index.md       # Inventory page
├── products.csv   # Product data
└── README.md      # This file
```

## Customization

Edit `products.csv` to add your own data. The CSV header row defines the column names used in `lvt-columns`.

## Learn More

- [Tinkerdown Documentation](https://github.com/livetemplate/tinkerdown)
- [CSV Source Reference](https://github.com/livetemplate/tinkerdown/blob/main/docs/sources/csv.md)
