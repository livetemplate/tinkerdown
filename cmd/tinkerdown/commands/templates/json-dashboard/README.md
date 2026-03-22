# <<.Title>>

A metrics dashboard built with [Tinkerdown](https://github.com/livetemplate/tinkerdown).

## Quick Start

```bash
cd <<.ProjectName>>
tinkerdown serve
```

Open http://localhost:8080 in your browser.

## Features

- JSON file as data source
- Computed expressions (`=count()`, `=sum()`, `=max()`)
- Auto-rendered table via `lvt-columns`
- Inline stats that update with the data

## Project Structure

```
<<.ProjectName>>/
├── index.md       # Dashboard page with computed expressions
├── metrics.json   # Task data
└── README.md      # This file
```

## Customization

Edit `metrics.json` to track your own project data. Add or remove fields as needed — just update the `lvt-columns` attribute to match.

## Learn More

- [Tinkerdown Documentation](https://github.com/livetemplate/tinkerdown)
- [JSON Source Reference](https://github.com/livetemplate/tinkerdown/blob/main/docs/sources/json.md)
