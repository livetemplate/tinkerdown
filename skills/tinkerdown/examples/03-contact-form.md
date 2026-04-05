---
title: "Contact Form"
---

# Contact Form

A contact form demonstrating form handling with multiple field types.

**Features demonstrated:**
- `lvt-persist` with multiple field types
- Text, email, textarea, checkbox inputs
- Form validation (HTML5)
- Table display of submissions
- **No CSS classes needed** - PicoCSS styles semantic HTML automatically

```lvt
<main>
    <h1>Contact Us</h1>

    <!-- Contact Form -->
    <article>
        <form name="save" lvt-persist="contacts">
            <label>
                Name
                <input type="text" name="name" required minlength="2" placeholder="Your name">
            </label>

            <label>
                Email
                <input type="email" name="email" required placeholder="you@example.com">
            </label>

            <label>
                Subject
                <input type="text" name="subject" required placeholder="What's this about?">
            </label>

            <label>
                Message
                <textarea name="message" required rows="4" placeholder="Your message..."></textarea>
            </label>

            <label>
                <input type="checkbox" name="subscribe">
                Subscribe to our newsletter
            </label>

            <button type="submit">Send Message</button>
        </form>
    </article>

    <!-- Submissions List -->
    <h2>Recent Submissions</h2>

    {{if .Contacts}}
    <table>
        <thead>
            <tr>
                <th>Name</th>
                <th>Email</th>
                <th>Subject</th>
                <th>Newsletter</th>
                <th>Actions</th>
            </tr>
        </thead>
        <tbody>
            {{range .Contacts}}
            <tr>
                <td>{{.Name}}</td>
                <td>{{.Email}}</td>
                <td>{{.Subject}}</td>
                <td>{{if .Subscribe}}Yes{{else}}No{{end}}</td>
                <td>
                    <button name="Delete" data-id="{{.Id}}" >Delete</button>
                </td>
            </tr>
            {{end}}
        </tbody>
    </table>
    {{else}}
    <p><em>No submissions yet.</em></p>
    {{end}}
</main>
```

## How It Works

1. **Form fields** - Each input name becomes a column in the auto-generated table
2. **Field types** - `text`, `email`, `textarea`, `checkbox` are all supported
3. **Validation** - Use HTML5 attributes like `required`, `minlength`, `type="email"`
4. **Auto-CRUD** - `lvt-persist="contacts"` creates table and generates Save/Delete actions

## Prompt to Generate This

> Build a contact form with Livemdtools. Include name, email, subject, message, and a newsletter checkbox. Show submitted contacts in a table with delete buttons. Use form validation. Use semantic HTML.
