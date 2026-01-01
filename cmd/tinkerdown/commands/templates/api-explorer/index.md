---
title: "{{.Title}}"
persist: localstorage
sources:
  repos:
    type: rest
    url: https://api.github.com/search/repositories?q=${query}&per_page=10
---

# {{.Title}}

Search GitHub repositories using the GitHub API.

## Search

```lvt
<main lvt-source="repos">
    <form lvt-submit="SetQuery" style="margin-bottom: 16px; display: flex; gap: 8px;">
        <input type="text" name="query" placeholder="Search repositories..." value="{{.Query}}"
               style="flex: 1; padding: 8px; border: 1px solid #ccc; border-radius: 4px;">
        <button type="submit"
                style="padding: 8px 16px; background: #007bff; color: white; border: none; border-radius: 4px; cursor: pointer;">
            Search
        </button>
    </form>

    {{if .Error}}
    <p><mark>Error: {{.Error}}</mark></p>
    {{else if .Data}}
    <p><small>Found {{len .Data.items}} repositories</small></p>
    <table>
        <thead>
            <tr>
                <th>Repository</th>
                <th>Stars</th>
                <th>Language</th>
                <th>Description</th>
            </tr>
        </thead>
        <tbody>
            {{range .Data.items}}
            <tr>
                <td><a href="{{.html_url}}" target="_blank">{{.full_name}}</a></td>
                <td>{{.stargazers_count}}</td>
                <td>{{.language}}</td>
                <td style="max-width: 300px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">{{.description}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>
    {{else}}
    <p>Enter a search term to find repositories.</p>
    {{end}}
</main>
```

## How It Works

1. Enter a search term (e.g., "golang cli")
2. Click Search to query GitHub's API
3. Results show repository name, stars, language, and description

**Note:** GitHub API has rate limits. For production use, add authentication.
