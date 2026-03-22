# <<.Title>>

A notes manager built with [Tinkerdown](https://github.com/livetemplate/tinkerdown).

## Quick Start

```bash
cd <<.ProjectName>>
tinkerdown serve
```

Open http://localhost:8080 in your browser.

## Features

- Markdown file as data source (no database needed)
- Add and delete notes through the web UI
- Data is stored as a markdown table — editable in any text editor
- Tag-based organization

## Project Structure

```
<<.ProjectName>>/
├── index.md          # Notes interface
├── _data/
│   └── notes.md      # Notes data (markdown table)
└── README.md         # This file
```

## Customization

Edit `_data/notes.md` directly to manage notes, or use the web UI. Add columns to the markdown table and update the template to display them.

## Learn More

- [Tinkerdown Documentation](https://github.com/livetemplate/tinkerdown)
- [Markdown Source Reference](https://github.com/livetemplate/tinkerdown/blob/main/docs/sources/markdown.md)
