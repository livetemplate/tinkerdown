---
title: "[[.Title]]"
sources:
  messages:
    type: sqlite
    db: "./messages.db"
    table: messages
    readonly: false
---

# [[.Title]]

A contact form with SQLite persistence.

## Contact Us

```lvt
<main lvt-source="messages">
    <div class="form-container">
        <form class="contact-form" lvt-submit="Add">
            <div class="form-group">
                <label for="name">Name</label>
                <input type="text" id="name" name="name" placeholder="Your name" required>
            </div>

            <div class="form-group">
                <label for="email">Email</label>
                <input type="email" id="email" name="email" placeholder="your@email.com" required>
            </div>

            <div class="form-group">
                <label for="subject">Subject</label>
                <select id="subject" name="subject">
                    <option value="general">General Inquiry</option>
                    <option value="support">Support</option>
                    <option value="feedback">Feedback</option>
                    <option value="other">Other</option>
                </select>
            </div>

            <div class="form-group">
                <label for="message">Message</label>
                <textarea id="message" name="message" rows="5" placeholder="Your message..." required></textarea>
            </div>

            <button type="submit" class="submit-btn">Send Message</button>
        </form>
    </div>
</main>

<style>
.form-container {
    max-width: 500px;
    margin: 0 auto;
}

.contact-form {
    background: #f8f9fa;
    padding: 2rem;
    border-radius: 8px;
}

.form-group {
    margin-bottom: 1rem;
}

.form-group label {
    display: block;
    margin-bottom: 0.5rem;
    font-weight: 500;
    color: #333;
}

.form-group input,
.form-group select,
.form-group textarea {
    width: 100%;
    padding: 0.75rem;
    border: 1px solid #ddd;
    border-radius: 4px;
    font-size: 1rem;
    box-sizing: border-box;
}

.form-group input:focus,
.form-group select:focus,
.form-group textarea:focus {
    outline: none;
    border-color: #007bff;
    box-shadow: 0 0 0 3px rgba(0, 123, 255, 0.1);
}

.form-group textarea {
    resize: vertical;
    min-height: 100px;
}

.submit-btn {
    width: 100%;
    padding: 0.75rem 1.5rem;
    background: #007bff;
    color: white;
    border: none;
    border-radius: 4px;
    font-size: 1rem;
    font-weight: 500;
    cursor: pointer;
    transition: background 0.2s;
}

.submit-btn:hover {
    background: #0056b3;
}
</style>
```

---

## Submitted Messages

```lvt
<section lvt-source="messages">
    {{if .Error}}
    <p class="error">Error: {{.Error}}</p>
    {{else if eq (len .Data) 0}}
    <p class="empty">No messages yet.</p>
    {{else}}
    <div class="messages-list">
        {{range .Data}}
        <div class="message-card">
            <div class="message-header">
                <strong>{{.Name}}</strong>
                <span class="badge">{{.Subject}}</span>
                <button class="btn-delete" lvt-click="Delete" lvt-data-id="{{.Id}}">x</button>
            </div>
            <div class="message-email">{{.Email}}</div>
            <div class="message-body">{{.Message}}</div>
        </div>
        {{end}}
    </div>
    <p class="message-count">{{len .Data}} message(s) received</p>
    {{end}}
</section>

<style>
.messages-list {
    display: flex;
    flex-direction: column;
    gap: 1rem;
}

.message-card {
    background: #fff;
    border: 1px solid #e9ecef;
    padding: 1rem;
    border-radius: 8px;
}

.message-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 0.5rem;
}

.message-header strong {
    flex: 1;
}

.badge {
    background: #e9ecef;
    padding: 0.25rem 0.5rem;
    border-radius: 4px;
    font-size: 0.8em;
    color: #495057;
}

.btn-delete {
    padding: 0.25rem 0.5rem;
    color: #dc3545;
    border: 1px solid #dc3545;
    background: transparent;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.8em;
}

.message-email {
    color: #666;
    font-size: 0.9em;
    margin-bottom: 0.5rem;
}

.message-body {
    color: #333;
    white-space: pre-wrap;
}

.message-count {
    color: #666;
    font-size: 0.9em;
    margin-top: 1rem;
}

.error {
    color: #dc3545;
    background: #f8d7da;
    padding: 0.75rem;
    border-radius: 4px;
}

.empty {
    color: #666;
    font-style: italic;
    text-align: center;
    padding: 2rem;
}
</style>
```
