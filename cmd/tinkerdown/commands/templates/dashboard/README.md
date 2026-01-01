# [[.Title]]

A multi-source dashboard built with tinkerdown.

## Quick Start

```bash
cd [[.ProjectName]]
tinkerdown serve
```

Then open http://localhost:8080

## Features

- Multiple data sources (tasks and team)
- Stats overview with live counts
- Add and delete tasks
- Responsive grid layout

## Project Structure

```
[[.ProjectName]]/
├── index.md          # Main dashboard page
├── _data/
│   ├── tasks.md      # Task data (editable)
│   └── team.md       # Team member data
└── README.md         # This file
```

## Customization

- Edit `_data/tasks.md` to modify initial tasks
- Edit `_data/team.md` to update team members
- Add more data sources in the frontmatter
- Add new stat cards to the overview

## Learn More

- [tinkerdown Documentation](https://github.com/livetemplate/tinkerdown)
- [Markdown Source Reference](https://github.com/livetemplate/tinkerdown/blob/main/docs/sources/markdown.md)
