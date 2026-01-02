---
title: GitHub Repos Dashboard
sources:
  repos:
    type: rest
    url: https://api.github.com/users/torvalds/repos?sort=updated&per_page=10
---

# GitHub Repos Dashboard

Showing the 10 most recently updated repositories for **torvalds**.

```lvt
<table lvt-source="repos" lvt-columns="name:Repository,description:Description,stargazers_count:Stars,language:Language" lvt-empty="No repositories found">
</table>
```

---

## How It Works

This app fetches data from the GitHub API and displays it in a table.
No API key required for public repos.

To customize:
1. Change `torvalds` to any GitHub username
2. Adjust `per_page` for more/fewer results
3. Change `sort` to `stars`, `created`, or `pushed`
