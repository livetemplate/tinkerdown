---
title: Meeting Notes - Q1 Planning
sources:
  actions:
    type: markdown
    anchor: "#action-items"
    readonly: false
  decisions:
    type: markdown
    anchor: "#decisions"
    readonly: false
---

# Q1 Planning Meeting

**Date:** January 15, 2024
**Attendees:** Alice, Bob, Charlie
**Duration:** 1 hour

## Summary

Discussed Q1 priorities and resource allocation. Agreed to focus on
customer retention over new features.

## Discussion Notes

### Customer Feedback
- NPS dropped from 42 to 38 last quarter
- Top complaint: slow response times
- Bob's team will prioritize support tooling

### Engineering Capacity
- 3 new hires starting in February
- Alice proposes pairing program for onboarding
- Charlie concerned about sprint velocity during ramp-up

---

## Action Items {#action-items}

- [ ] Alice: Create onboarding schedule for new hires <!-- id:act_001 -->
- [ ] Bob: Draft support tooling proposal by Friday <!-- id:act_002 -->
- [ ] Charlie: Review velocity data from last 3 sprints <!-- id:act_003 -->

### Add Action Item

```lvt
<form lvt-submit="add" lvt-source="actions">
  <input name="text" placeholder="Action item..." required style="width: 80%">
  <button type="submit">Add</button>
</form>
```

### Track Progress

```lvt
<ul lvt-source="actions" lvt-field="text" lvt-actions="delete:×" lvt-empty="No action items yet">
</ul>
```

---

## Decisions {#decisions}

- Focus Q1 on retention, not acquisition <!-- id:dec_001 -->
- Delay API v2 launch to Q2 <!-- id:dec_002 -->

### Record Decision

```lvt
<form lvt-submit="add" lvt-source="decisions">
  <input name="text" placeholder="Decision made..." required style="width: 80%">
  <button type="submit">Record</button>
</form>
```

```lvt
<ol lvt-source="decisions" lvt-field="text" lvt-actions="delete:×">
</ol>
```

---

## Next Meeting

- **When:** January 22, 2024
- **Agenda:** Review Bob's proposal, onboarding plan
