# Prompt Used

```
Create a tinkerdown reading list app.

I want to track:
- Things I want to read (title, url, where I found it, tags)
- Things I'm currently reading (title, started date, notes)
- Things I've finished (title, finished date, rating, key takeaway)

Store everything in the markdown file.
Let me add new items via forms and delete items I don't need.
```

# Why This Pattern Works

**Personal knowledge management** meets **active documents**:

1. **Markdown storage** = Readable in any editor, grep-able, git-friendly
2. **Three lists** = Clear workflow (to read → reading → finished)
3. **Different fields per stage** = Captures what matters at each step
4. **Form input** = No need to manually format table rows

# The "Active Notes" Concept

Traditional notes are static. Tinkerdown notes are active:

| Static Note | Active Note |
|-------------|-------------|
| Text you read | Text + UI you interact with |
| Manual updates | Form submissions |
| Separate tools for tracking | Tracking built into the note |
| "I'll organize this later" | Already organized |

# LLM Success Rate

Tested with Claude 3.5 Sonnet: **9/10 successful first attempts**

Common issues:
- Sometimes creates one giant table instead of three stages
- May forget `readonly: false` on some sources

# Variations

- **Paper/Research tracker** - Add citation, abstract, relevance score
- **Course notes** - Track lectures, assignments, key concepts
- **Podcast queue** - Episodes to listen, timestamps, notes
- **Recipe collection** - Ingredients, instructions, ratings
