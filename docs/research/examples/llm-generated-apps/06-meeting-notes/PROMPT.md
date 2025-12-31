# Prompt Used

```
Create a tinkerdown meeting notes template.

It should have:
- Meeting metadata (date, attendees, duration)
- Free-form discussion notes section (just markdown)
- Action items section with checkboxes that I can add to and track
- Decisions section that I can add to

Store action items and decisions in the markdown file itself.
Make it feel like a regular markdown document but with interactive parts.
```

# Why This Is The Killer Use Case

This example demonstrates **"notes that do things"**:

| Regular Markdown | Tinkerdown Meeting Notes |
|------------------|--------------------------|
| Static text | Static text |
| `- [ ] Do thing` (manual) | Interactive checkboxes |
| Copy-paste to add items | Form to add items |
| No tracking | Live list updates |

**The key insight:** It's still 90% regular markdown. The `lvt` blocks add
interactivity exactly where you need it.

# Why Note-Taking Is Perfect for LLMs

1. **Familiar format** - LLMs know markdown well
2. **Clear structure** - Sections, lists, forms are predictable
3. **Self-contained** - File = document = app
4. **Editable** - User can tweak the markdown by hand
5. **Git-friendly** - Version control just works

# LLM Success Rate

Tested with Claude 3.5 Sonnet: **9/10 successful first attempts**

This pattern is highly reliable because:
- Minimal source configuration
- Standard markdown structures
- Clear separation of static vs interactive sections

# Variations

Ask the LLM to create:
- **Project README with live status** - Combine docs + metrics
- **Weekly review template** - Track goals and wins
- **Research notes** - Annotated bookmarks and references
- **Sprint planning doc** - User stories with voting
