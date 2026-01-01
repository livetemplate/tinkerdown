---
title: "{{.Title}}"
sources:
  submissions:
    type: sqlite
    db: "./submissions.db"
    table: submissions
    readonly: false
---

# {{.Title}}

A contact form with SQLite persistence.

## Submit a Message

```lvt
<main lvt-source="submissions">
    <form lvt-submit="Add" style="max-width: 500px; display: flex; flex-direction: column; gap: 12px;">
        <div>
            <label for="name" style="display: block; margin-bottom: 4px; font-weight: bold;">Name</label>
            <input type="text" id="name" name="name" required
                   style="width: 100%; padding: 8px; border: 1px solid #ccc; border-radius: 4px;">
        </div>
        <div>
            <label for="email" style="display: block; margin-bottom: 4px; font-weight: bold;">Email</label>
            <input type="email" id="email" name="email" required
                   style="width: 100%; padding: 8px; border: 1px solid #ccc; border-radius: 4px;">
        </div>
        <div>
            <label for="message" style="display: block; margin-bottom: 4px; font-weight: bold;">Message</label>
            <textarea id="message" name="message" rows="4" required
                      style="width: 100%; padding: 8px; border: 1px solid #ccc; border-radius: 4px;"></textarea>
        </div>
        <button type="submit"
                style="padding: 10px 20px; background: #28a745; color: white; border: none; border-radius: 4px; cursor: pointer; align-self: flex-start;">
            Submit
        </button>
    </form>

    <hr style="margin: 24px 0;">

    <h2>Submissions</h2>
    {{if .Error}}
    <p><mark>Error: {{.Error}}</mark></p>
    {{else if not .Data}}
    <p>No submissions yet.</p>
    {{else}}
    <table>
        <thead>
            <tr>
                <th>Name</th>
                <th>Email</th>
                <th>Message</th>
                <th>Actions</th>
            </tr>
        </thead>
        <tbody>
            {{range .Data}}
            <tr>
                <td>{{.Name}}</td>
                <td>{{.Email}}</td>
                <td>{{.Message}}</td>
                <td>
                    <button lvt-click="Delete" lvt-data-id="{{.Id}}"
                            style="color: red; border: 1px solid red; background: transparent; border-radius: 4px; cursor: pointer; padding: 2px 8px;">
                        Delete
                    </button>
                </td>
            </tr>
            {{end}}
        </tbody>
    </table>
    {{end}}
</main>
```
