---
title: Team Feedback Collector
sources:
  feedback:
    type: sqlite
    database: ./feedback.db
    table: feedback
---

# Team Feedback Collector

Anonymous feedback form for your team. Responses are stored locally.

## Submit Feedback

```lvt
<form lvt-submit="add" lvt-source="feedback">
  <label>How are you feeling today?</label>
  <select name="mood" required>
    <option value="great">Great</option>
    <option value="good">Good</option>
    <option value="okay">Okay</option>
    <option value="struggling">Struggling</option>
  </select>

  <label>What's going well?</label>
  <textarea name="going_well" rows="3" placeholder="Share what's working..."></textarea>

  <label>What could be better?</label>
  <textarea name="could_improve" rows="3" placeholder="Share suggestions..."></textarea>

  <label>Any blockers?</label>
  <input name="blockers" placeholder="What's slowing you down?">

  <button type="submit">Submit Feedback</button>
</form>
```

---

## All Feedback

```lvt
<table lvt-source="feedback" lvt-columns="mood:Mood,going_well:Going Well,could_improve:Could Improve,blockers:Blockers" lvt-empty="No feedback yet. Be the first to share!">
</table>
```
