---
title: Reading List
sources:
  toread:
    type: markdown
    anchor: "#to-read"
    readonly: false
  reading:
    type: markdown
    anchor: "#currently-reading"
    readonly: false
  finished:
    type: markdown
    anchor: "#finished"
    readonly: false
---

# My Reading List

A place to track articles, books, and papers I want to read.

## Add to Reading List

```lvt
<form lvt-submit="add" lvt-source="toread">
  <input name="title" placeholder="Title" required>
  <input name="url" type="url" placeholder="URL (optional)">
  <input name="source" placeholder="Where I found it">
  <input name="tags" placeholder="Tags (comma separated)">
  <button type="submit">Add</button>
</form>
```

---

## To Read {#to-read}

| title | url | source | tags |
|-------|-----|--------|------|
| How to Build a Second Brain | https://fortelabs.com/blog/basboverview/ | Twitter | productivity, notes | <!-- id:tr_001 -->
| The Architecture of Open Source Applications | https://aosabook.org | HN | programming, architecture | <!-- id:tr_002 -->

```lvt
<table lvt-source="toread" lvt-columns="title:Title,source:Found via,tags:Tags" lvt-actions="delete:×" lvt-empty="Nothing in queue!">
</table>
```

---

## Currently Reading {#currently-reading}

| title | started | notes |
|-------|---------|-------|
| Designing Data-Intensive Applications | 2024-01-10 | Chapter 3 on storage engines is excellent | <!-- id:cr_001 -->

```lvt
<table lvt-source="reading" lvt-columns="title:Title,started:Started,notes:Notes" lvt-actions="delete:×" lvt-empty="Start something!">
</table>
```

---

## Finished {#finished}

| title | finished | rating | takeaway |
|-------|----------|--------|----------|
| The Pragmatic Programmer | 2024-01-05 | 5 | Code reviews = learning opportunities | <!-- id:fn_001 -->

```lvt
<table lvt-source="finished" lvt-columns="title:Title,finished:Finished,rating:Rating,takeaway:Key Takeaway" lvt-actions="delete:×" lvt-empty="Finish something!">
</table>
```

---

## Stats

- **To read:** Check the queue above
- **This year's goal:** 24 books/papers
