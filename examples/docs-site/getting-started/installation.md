---
title: "Installation"
---

# Installation

Get LivePage up and running on your machine.

## Prerequisites

- Go 1.21 or later
- A terminal/command line

## Install from Source

```bash
git clone https://github.com/livetemplate/livepage
cd livepage
go install ./cmd/livepage
```

## Verify Installation

Check that LivePage is installed correctly:

```bash
livepage --version
```

## Create Your First Page

Create a new directory and a simple markdown file:

```bash
mkdir my-tutorial
cd my-tutorial
echo "# Hello LivePage" > index.md
```

Start the development server:

```bash
livepage serve .
```

Open your browser to `http://localhost:8080` and you should see your page!

## Next Steps

Now that you have LivePage installed, learn how to [create pages](/guides/creating-pages).
