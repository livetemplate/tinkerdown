# [[.Title]]

A REST API explorer built with tinkerdown.

## Quick Start

```bash
cd [[.ProjectName]]
tinkerdown serve
```

Then open http://localhost:8080

## Features

- Connect to any REST API
- Live data refresh
- Response caching
- Environment variable support for secrets

## Project Structure

```
[[.ProjectName]]/
├── index.md      # API explorer page
└── README.md     # This file
```

## Customization

Edit the `sources` section in `index.md` to connect to your own APIs:

```yaml
sources:
  myapi:
    type: rest
    url: https://api.example.com/data
    headers:
      Authorization: "Bearer ${API_TOKEN}"
    cache:
      ttl: 30s
```

## Environment Variables

Use `${VAR_NAME}` syntax for secrets:

```bash
export API_TOKEN="your-secret-token"
tinkerdown serve
```

## Learn More

- [tinkerdown Documentation](https://github.com/livetemplate/tinkerdown)
- [REST Source Reference](https://github.com/livetemplate/tinkerdown/blob/main/docs/sources/rest.md)
