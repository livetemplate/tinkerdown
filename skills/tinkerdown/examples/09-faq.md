---
title: "FAQ Page"
---

# FAQ Page

A frequently asked questions page demonstrating collapsible sections.

**Features demonstrated:**
- Accordion-style FAQ items using `<details>`
- Add new FAQ entries
- Category organization
- **No CSS classes needed** - PicoCSS styles semantic HTML automatically

```lvt
<main>
    <hgroup>
        <h1>Frequently Asked Questions</h1>
        <p>Find answers to common questions below</p>
    </hgroup>

    <!-- Add FAQ Form (Admin) -->
    <details>
        <summary>+ Add New FAQ</summary>
        <article>
            <form name="save" lvt-persist="faqs">
                <label>
                    Question
                    <input type="text" name="question" required placeholder="Enter the question">
                </label>
                <label>
                    Answer
                    <textarea name="answer" required rows="4" placeholder="Enter the answer"></textarea>
                </label>
                <label>
                    Category
                    <select name="category" required>
                        <option value="general">General</option>
                        <option value="billing">Billing</option>
                        <option value="technical">Technical</option>
                        <option value="account">Account</option>
                        <option value="shipping">Shipping</option>
                    </select>
                </label>
                <button type="submit">Add FAQ</button>
            </form>
        </article>
    </details>

    <!-- FAQ List -->
    {{if .Faqs}}
    {{range .Faqs}}
    <details>
        <summary>
            {{.Question}}
            <kbd>{{.Category}}</kbd>
        </summary>
        <p>{{.Answer}}</p>
        <button name="Delete" data-id="{{.Id}}" >Delete FAQ</button>
    </details>
    {{end}}

    <small>{{len .Faqs}} FAQ entries</small>
    {{else}}
    <article>
        <p><em>No FAQs yet. Click "Add New FAQ" above to create your first entry.</em></p>
    </article>
    {{end}}
</main>
```

## How It Works

1. **Accordion** - HTML `<details>` and `<summary>` elements create collapsible sections
2. **Category badges** - Use `<kbd>` tag for visual category labels
3. **Admin form** - Hidden by default in a `<details>` element
4. **Native behavior** - No JavaScript needed for expand/collapse

## Prompt to Generate This

> Build an FAQ page with Livemdtools. Use HTML details/summary for collapsible Q&A sections. Include a hidden admin form to add new FAQs with question, answer, and category. Show category badges. Include delete buttons. Use semantic HTML.
