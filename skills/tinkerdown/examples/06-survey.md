---
title: "Survey Form"
---

# Customer Survey

A multi-section survey demonstrating radio buttons, select dropdowns, and ratings.

**Features demonstrated:**
- Radio button groups
- Select dropdowns
- Range/rating inputs
- Multi-field forms
- Results display
- **No CSS classes needed** - PicoCSS styles semantic HTML automatically

```lvt
<main>
    <hgroup>
        <h1>Customer Satisfaction Survey</h1>
        <p>Help us improve by sharing your feedback</p>
    </hgroup>

    <!-- Survey Form -->
    <article>
        <form name="save" lvt-persist="responses">
            <label>
                Your Name
                <input type="text" name="name" required placeholder="Enter your name">
            </label>

            <label>
                Email
                <input type="email" name="email" required placeholder="you@example.com">
            </label>

            <!-- Overall Satisfaction (Radio) -->
            <fieldset>
                <legend>Overall Satisfaction</legend>
                <label><input type="radio" name="satisfaction" value="very_satisfied"> Very Satisfied</label>
                <label><input type="radio" name="satisfaction" value="satisfied"> Satisfied</label>
                <label><input type="radio" name="satisfaction" value="neutral"> Neutral</label>
                <label><input type="radio" name="satisfaction" value="dissatisfied"> Dissatisfied</label>
            </fieldset>

            <!-- How did you hear about us (Select) -->
            <label>
                How did you hear about us?
                <select name="source">
                    <option value="">Select an option</option>
                    <option value="search">Search Engine</option>
                    <option value="social">Social Media</option>
                    <option value="friend">Friend/Colleague</option>
                    <option value="ad">Advertisement</option>
                    <option value="other">Other</option>
                </select>
            </label>

            <!-- Rating (1-10) -->
            <label>
                Rating (1-10)
                <input type="range" name="rating" min="1" max="10" value="5">
            </label>

            <!-- Comments -->
            <label>
                Additional Comments
                <textarea name="comments" rows="4" placeholder="Tell us more about your experience..."></textarea>
            </label>

            <!-- Would Recommend (Checkbox) -->
            <label>
                <input type="checkbox" name="would_recommend">
                I would recommend this product to others
            </label>

            <button type="submit">Submit Survey</button>
        </form>
    </article>

    <!-- Survey Results -->
    <h2>Survey Responses</h2>

    {{if .Responses}}
    {{range .Responses}}
    <article>
        <header>
            <strong>{{.Name}}</strong>
            <small>{{.Email}}</small>
        </header>
        <p>
            Satisfaction: <strong>{{.Satisfaction}}</strong> |
            Rating: <strong>{{.Rating}}/10</strong> |
            Source: {{.Source}}
        </p>
        {{if .Comments}}<blockquote>{{.Comments}}</blockquote>{{end}}
        <footer>
            <button name="Delete" data-id="{{.Id}}" >Delete</button>
        </footer>
    </article>
    {{end}}
    {{else}}
    <p><em>No responses yet.</em></p>
    {{end}}
</main>
```

## How It Works

1. **Radio buttons** - Use `name="satisfaction"` with same name for grouping
2. **Select dropdown** - Use `<select>` with `<option>` elements
3. **Range input** - `type="range"` creates a slider
4. **Fieldset** - Groups related form elements with a legend

## Prompt to Generate This

> Build a customer survey with Livemdtools. Include name, email, satisfaction rating (radio buttons), how they heard about us (dropdown), a 1-10 rating slider, comments textarea, and a "would recommend" checkbox. Show results in cards. Use semantic HTML.
