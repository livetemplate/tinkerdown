---
title: "<<.Title>>"
sources:
  notes:
    type: markdown
    file: "./_data/notes.md"
    anchor: "#notes"
    readonly: false
---

# <<.Title>>

A simple notes manager backed by a markdown file.

## Add a Note

```lvt
<form name="Add" lvt-source="notes" lvt-el:reset:on:success style="display: flex; gap: 8px; flex-wrap: wrap; align-items: center; margin-bottom: 16px;">
    <input name="Title" placeholder="Title" required style="padding: 8px; border: 1px solid #ccc; border-radius: 4px;">
    <input name="Content" placeholder="Note content..." required style="flex: 1; min-width: 200px; padding: 8px; border: 1px solid #ccc; border-radius: 4px;">
    <input name="Tag" placeholder="Tag" style="width: 120px; padding: 8px; border: 1px solid #ccc; border-radius: 4px;">
    <button type="submit" style="padding: 8px 16px; background: #007bff; color: white; border: none; border-radius: 4px; cursor: pointer;">
        Add Note
    </button>
</form>
```

## Your Notes

```lvt
<main lvt-source="notes">
    {{if .Error}}
    <p><mark>Error: {{.Error}}</mark></p>
    {{else if eq (len .Data) 0}}
    <p><em>No notes yet. Add your first note above!</em></p>
    {{else}}
    <table style="width: 100%; border-collapse: collapse;">
        <thead>
            <tr style="background: #f5f5f5;">
                <th style="text-align: left; padding: 12px; border-bottom: 2px solid #ddd;">Title</th>
                <th style="text-align: left; padding: 12px; border-bottom: 2px solid #ddd;">Content</th>
                <th style="text-align: left; padding: 12px; border-bottom: 2px solid #ddd;">Tag</th>
                <th style="width: 50px; padding: 12px; border-bottom: 2px solid #ddd;"></th>
            </tr>
        </thead>
        <tbody>
        {{range .Data}}
            <tr style="border-bottom: 1px solid #eee;">
                <td style="padding: 12px; font-weight: 500;">{{.Title}}</td>
                <td style="padding: 12px;">{{.Content}}</td>
                <td style="padding: 12px;">
                    {{if .Tag}}<span style="background: #e3f2fd; color: #1565c0; padding: 2px 8px; border-radius: 12px; font-size: 0.85em;">{{.Tag}}</span>{{end}}
                </td>
                <td style="padding: 12px; text-align: center;">
                    <button name="Delete" data-id="{{.Id}}"
                            style="padding: 4px 8px; color: #dc3545; border: 1px solid #dc3545; background: transparent; border-radius: 4px; cursor: pointer;"
                            title="Delete note">x</button>
                </td>
            </tr>
        {{end}}
        </tbody>
    </table>
    {{end}}
</main>
```

---

## How It Works

This app stores data in a **plain markdown file** (`_data/notes.md`). You can edit it with any text editor.

### Markdown Table Source

```yaml
sources:
  notes:
    type: markdown
    file: "./_data/notes.md"
    anchor: "#notes"
    readonly: false
```

- `anchor: "#notes"` — reads the table under the `# Notes` heading
- `readonly: false` — enables Add and Delete operations
- Data lives in `_data/notes.md` as a standard markdown table
