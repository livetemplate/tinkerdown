---
title: Job Application Tracker
sources:
  jobs:
    type: markdown
    anchor: "#applications"
    readonly: false
---

# Job Application Tracker

Track your job applications in one place. Data is stored in this file.

## Add Application

```lvt
<form lvt-submit="add" lvt-source="jobs">
  <input name="company" placeholder="Company" required>
  <input name="position" placeholder="Position" required>
  <input name="status" placeholder="Status (applied/interview/offer/rejected)" value="applied">
  <input name="date" type="date" required>
  <input name="notes" placeholder="Notes">
  <button type="submit">Add Application</button>
</form>
```

## Applications

```lvt
<table lvt-source="jobs" lvt-columns="company:Company,position:Position,status:Status,date:Date,notes:Notes" lvt-actions="delete:×" lvt-empty="No applications yet. Add your first one above!">
</table>
```

---

## Applications {#applications}

| company | position | status | date | notes |
|---------|----------|--------|------|-------|
| Acme Corp | Software Engineer | applied | 2024-01-15 | Referral from John | <!-- id:job_001 -->
| TechStart | Senior Developer | interview | 2024-01-18 | Phone screen scheduled | <!-- id:job_002 -->
