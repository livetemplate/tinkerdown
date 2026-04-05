---
title: "Simple Blog"
---

# Simple Blog

A blog demonstrating `lvt-persist` for posts.

**Features demonstrated:**
- `lvt-persist` for blog posts
- Textarea for long content
- Date display
- Delete functionality
- **No CSS classes needed** - PicoCSS styles semantic HTML automatically

```lvt
<main>
    <!-- Header -->
    <hgroup>
        <h1>My Blog</h1>
        <p>Thoughts and ideas</p>
    </hgroup>

    <!-- New Post Form -->
    <article>
        <header>Write a Post</header>
        <form name="save" lvt-persist="posts">
            <input type="text" name="title" required placeholder="Post title">
            <textarea name="content" required rows="6" placeholder="Write your post content here..."></textarea>
            <input type="text" name="author" placeholder="Your name (optional)">
            <button type="submit">Publish Post</button>
        </form>
    </article>

    <!-- Posts List -->
    {{if .Posts}}
    {{range .Posts}}
    <article>
        <header>
            <hgroup>
                <h2>{{.Title}}</h2>
                <p>{{if .Author}}By {{.Author}} - {{end}}{{.CreatedAt}}</p>
            </hgroup>
        </header>
        <p>{{.Content}}</p>
        <footer>
            <button name="Delete" data-id="{{.Id}}" >Delete Post</button>
        </footer>
    </article>
    {{end}}
    {{else}}
    <article>
        <p><em>No posts yet. Write your first post above!</em></p>
    </article>
    {{end}}
</main>
```

## How It Works

1. **Posts table** - `lvt-persist="posts"` creates a table with Title, Content, Author columns
2. **Auto-generated fields** - Id and CreatedAt are added automatically
3. **Long content** - `<textarea>` is stored as text in SQLite
4. **Template display** - Posts are available as `.Posts` array

## Prompt to Generate This

> Build a simple blog with Livemdtools. Let users write posts with a title, content, and optional author name. Display posts in a card layout with timestamps. Include delete buttons. Use semantic HTML.
