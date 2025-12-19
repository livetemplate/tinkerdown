---
title: "Contact Form with Auto-Persist"
type: tutorial
persist: localstorage
---

# Contact Form

This example demonstrates the **auto-persist** feature. No Go code required - the framework automatically:
- Parses form fields from the template
- Creates SQLite table with matching schema
- Generates State struct and Save action
- Loads existing records on init

```lvt
<main>
    <h2>Submit a Contact</h2>

    <article>
        <form lvt-submit="save" lvt-persist="contacts">
            <label>
                Name
                <input type="text" name="name" required placeholder="Your name">
            </label>

            <label>
                Email
                <input type="email" name="email" required placeholder="your@email.com">
            </label>

            <label>
                Message
                <textarea name="message" rows="4" placeholder="Your message..."></textarea>
            </label>

            <label>
                <input type="checkbox" name="subscribe">
                Subscribe to newsletter
            </label>

            <button type="submit">Submit</button>
        </form>
    </article>

    <h2>Submitted Contacts</h2>

    {{if .Contacts}}
    <table>
        <thead>
            <tr>
                <th>ID</th>
                <th>Name</th>
                <th>Email</th>
                <th>Message</th>
                <th>Subscribe</th>
                <th>Created</th>
            </tr>
        </thead>
        <tbody>
            {{range .Contacts}}
            <tr>
                <td>{{.ID}}</td>
                <td>{{.Name}}</td>
                <td>{{.Email}}</td>
                <td>{{.Message}}</td>
                <td>{{if .Subscribe}}Yes{{else}}No{{end}}</td>
                <td>{{.CreatedAt}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>
    {{else}}
    <p><em>No contacts submitted yet.</em></p>
    {{end}}
</main>
```
