# [[.Title]]

A CLI wrapper built with tinkerdown.

## Quick Start

```bash
cd [[.ProjectName]]
tinkerdown serve
```

Then open http://localhost:8080

## Features

- Wrap any CLI tool with a web form
- Auto-generated form fields from arguments
- Live output display
- Environment variable support

## Project Structure

```
[[.ProjectName]]/
├── index.md      # CLI wrapper interface
└── README.md     # This file
```

## Customization

Edit the `cmd` in `index.md` to wrap your own CLI tool:

```yaml
sources:
  command:
    type: exec
    cmd: your-cli-tool --arg1 value1 --arg2 value2
    manual: true
```

## Argument Types

Arguments are auto-detected:
- Text: `--name John`
- Number: `--count 5`
- Boolean: `--verbose true`

## Learn More

- [tinkerdown Documentation](https://github.com/livetemplate/tinkerdown)
- [Exec Source Reference](https://github.com/livetemplate/tinkerdown/blob/main/docs/sources/exec.md)
