# Tinkerdown: Planned Example Apps

**Date:** 2025-12-31
**Status:** Planned (not yet functional)
**Depends on:** Markdown-native implementation from unified design

---

These examples use the planned markdown-native syntax. They will become functional after implementing the features in weeks 1-8 of the implementation plan.

---

## 1. Minimal: Two-Line Todo

**Features required:** Heading as anchor, task list parsing

```markdown
# Todo
- [ ] Try tinkerdown
```

---

## 2. Simple: Expense Tracker

**Features required:** Heading as anchor, table parsing, schema inference, computed expressions

```markdown
# Expenses

## Log
| date | category | amount | note |
|------|----------|--------|------|
| 2024-01-15 | Food | $45.50 | Groceries |
| 2024-01-14 | Transport | $12.00 | Uber |

## Summary
**Total:** `sum(log.amount)`
**This week:** `sum(log.amount where date >= @start-of-week)`
**Food:** `sum(log.amount where category = "Food")`
```

---

## 3. Productivity: Team Tasks

**Features required:** Tabs, @mentions, #tags, status banners, computed expressions

```markdown
# Sprint Board

> 📊 `count(in-progress)` in progress | `count(done)` done

## [All] | [Mine] @me | [Urgent] #urgent

## Backlog
- [ ] User authentication #feature — @alice
- [ ] Fix memory leak #bug #urgent — @bob

## In Progress
- [ ] Dashboard redesign — @alice @friday

## Done
- [x] Database migration
- [x] CI/CD setup
```

---

## 4. Runbook: Incident Response

**Features required:** Status banners, ordered task lists, action buttons, operator identity

```markdown
# Database Incident

> ⚠️ Started: `now` | Operator: `operator`

## Checklist
1. [ ] Check database connectivity
2. [ ] Review error logs
3. [ ] Identify affected queries
4. [ ] Implement fix
5. [ ] Verify resolution
6. [ ] Update status page

## Execution Log
| time | action | who |
|------|--------|-----|
| | | |

[Resolve Incident](action:resolve)
[Escalate](action:escalate)
```

---

## 5. Automation: Daily Standup Bot

**Features required:** Output declarations, @schedule triggers

```markdown
# Standup Bot

> Slack: #team-standup

## Questions
1. What did you do yesterday?
2. What will you do today?
3. Any blockers?

Post questions @daily:9am @weekdays
Remind non-responders @daily:11am @weekdays
```

---

## 6. Automation: Health Monitor

**Features required:** Output declarations, @schedule triggers, conditional alerts

```markdown
# Health Monitor

> Slack: #ops-alerts
> Email: oncall@company.com

## Services
| service | status | last_check |
|---------|--------|------------|
| API | healthy | 2024-01-15 14:30 |
| DB | healthy | 2024-01-15 14:30 |
| Cache | degraded | 2024-01-15 14:28 |

Check all services @every:5min
Alert if status != "healthy"
```

---

## 7. Full Example: Personal CRM

**Features required:** All markdown-native features

```markdown
# My CRM

> 📊 `count(deals where stage != "Closed")` active | `sum(deals.value where stage != "Closed")` pipeline

## [All] Deals | [Hot] stage = "Negotiation" | [Won] stage = "Closed Won"

## Deals
| company | value | stage | owner | follow_up |
|---------|-------|-------|-------|-----------|
| Acme Corp | $50,000 | Proposal | @me | @friday |
| Globex | $25,000 | Negotiation | @me | @tomorrow @2pm |

## Follow-ups Due
> ⚠️ `count(deals where follow_up <= @today)` due today

## Activity Log
| date | company | action |
|------|---------|--------|
| 2024-01-15 | Acme Corp | Sent proposal |
| 2024-01-14 | Globex | Discovery call |

---

Remind of follow-ups @daily:9am
```

---

## 8. Habit Tracker

**Features required:** Task lists, computed expressions, @schedule triggers

```markdown
# Habits

> 🔥 Best streak: `max(habits.streak)` days

## Habits
- [x] Exercise — 🔥 12
- [x] Read 30 min — 🔥 8
- [ ] Meditate — 🔥 0
- [x] No sugar — 🔥 5

Reset streaks @daily:midnight where not done
Mark all undone @daily:midnight
```

---

## 9. Meeting Notes

**Features required:** @user mentions, task lists, @date triggers

```markdown
# Meetings

## 2024-01-15 Sprint Planning

**Attendees:** @alice, @bob, @charlie

### Decisions
- [x] Move to weekly releases — @alice
- [ ] Adopt new testing framework — @bob
- [ ] Hire two engineers — @charlie @next-month

### Action Items
1. [ ] Draft release schedule — @alice @friday
2. [ ] Jest migration spike — @bob @next-week
3. [x] Post job descriptions — @charlie

### Notes
> "Ship smaller, ship faster" — @alice
```

---

## 10. Inventory System

**Features required:** Computed expressions, conditional banners, action buttons

```markdown
# Inventory

> ⚠️ `count(items where quantity < reorder_level)` items need reorder

## Items
| sku | name | quantity | reorder_level | location |
|-----|------|----------|---------------|----------|
| A001 | Widget | 50 | 20 | Shelf A |
| A002 | Gadget | 5 | 10 | Shelf B |
| A003 | Gizmo | 100 | 25 | Shelf A |

### Adjust Stock
- SKU: ___
- Change: +/- ___
- Reason: Received | Sold | Damaged | Count

[Update Stock]

---

[Generate PO](action:create-po)
[Export CSV](export:csv)
```

---

## Implementation Milestones

| Example | Functional After |
|---------|------------------|
| 1. Two-Line Todo | Week 2 |
| 2. Expense Tracker | Week 4 |
| 3. Team Tasks | Week 6 |
| 4. Incident Runbook | Week 6 |
| 5. Standup Bot | Week 8 |
| 6. Health Monitor | Week 8 |
| 7. Personal CRM | Week 8 |
| 8. Habit Tracker | Week 8 |
| 9. Meeting Notes | Week 6 |
| 10. Inventory | Week 6 |

After each milestone, move the corresponding example to `examples/` as a working demo.
