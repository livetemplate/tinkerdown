# Documentation Site UX Improvements - Progress Tracker

## Overview
This document tracks the implementation of UX improvements for the LivePage documentation site based on the expert UX review conducted on November 21, 2025.

**Review Date**: November 21, 2025
**Screenshots**: `01_homepage.png`, `02_intro_page.png`, `03_installation.png`, `06_search_modal.png`
**Target**: examples/docs-site

---

## Phase 1: Critical UX Fixes (Priority 1)

### ✅ Task 1.1: Improve Typography & Content Layout
**Status**: 🔲 Not Started
**Estimated Time**: 2-3 hours
**Impact**: High - Affects readability across entire site

**Issues**:
- Content area too narrow (~700px)
- Insufficient line height (body text cramped)
- Weak heading hierarchy
- Tight paragraph spacing

**Implementation**:
- [ ] Increase max-width from 700px to 900px
- [ ] Improve line heights (body: 1.7, headings: 1.4)
- [ ] Strengthen heading hierarchy (font sizes, weights, spacing)
- [ ] Add proper content padding (3rem horizontal, 2rem vertical)
- [ ] Ensure mobile responsiveness

**Files to Modify**:
- `internal/assets/client/livepage-client.browser.css`

**Success Criteria**:
- Content feels spacious and readable
- Clear visual hierarchy in headings
- Comfortable reading experience

---

### ✅ Task 1.2: Enhance Active Page Indicator
**Status**: 🔲 Not Started
**Estimated Time**: 1 hour
**Impact**: High - Critical for navigation clarity

**Issues**:
- Active page in sidebar barely visible (subtle font weight change only)
- No background color or left border to indicate current location

**Implementation**:
- [ ] Add background color to active nav item
- [ ] Add left border (3-4px) in accent color
- [ ] Increase font weight contrast
- [ ] Add subtle padding/margin adjustments

**Files to Modify**:
- `internal/assets/client/livepage-client.browser.css`

**Success Criteria**:
- Active page immediately obvious at a glance
- Clear visual distinction from other nav items

---

### ✅ Task 1.3: Add Previous/Next Navigation
**Status**: 🔲 Not Started
**Estimated Time**: 3-4 hours
**Impact**: High - Essential for documentation UX

**Issues**:
- No prev/next buttons at bottom of content pages
- Users must return to sidebar to navigate sequentially

**Implementation**:
- [ ] Add nav order metadata to page structure
- [ ] Create prev/next component in Go templates
- [ ] Style prev/next buttons (left/right arrows)
- [ ] Add to page footer area
- [ ] Handle edge cases (first/last page)

**Files to Modify**:
- `internal/server/server.go` (nav logic)
- HTML template generation
- `internal/assets/client/livepage-client.browser.css`

**Success Criteria**:
- Clear prev/next buttons at bottom of each page
- Correct navigation order follows sidebar structure
- Proper styling with hover states

---

### ✅ Task 1.4: Improve Search Result Previews
**Status**: 🔲 Not Started
**Estimated Time**: 1-2 hours
**Impact**: Medium-High - Improves search usability

**Issues**:
- Search result previews very short (~40 chars)
- Hard to distinguish between results with similar titles
- Search modal shows "Introduction to LivePage Introduction to LivePage LivePage is a revolutionary framework..."

**Implementation**:
- [ ] Increase preview length to 120-150 characters
- [ ] Add ellipsis handling
- [ ] Ensure preview shows context around match
- [ ] Test with various query lengths

**Files to Modify**:
- Search result generation logic
- `internal/assets/client/livepage-client.browser.css` (preview styling)

**Success Criteria**:
- Meaningful context in search previews
- Easy to distinguish between results
- Preview helps users find right page

---

## Phase 2: Enhanced Navigation (Priority 2)

### ✅ Task 2.1: Add Table of Contents (ToC)
**Status**: 🔲 Not Started
**Estimated Time**: 3-4 hours
**Impact**: High - Major navigation enhancement

**Implementation**:
- [ ] Generate ToC from page headings (h2, h3)
- [ ] Add sticky ToC to right sidebar
- [ ] Implement smooth scroll to sections
- [ ] Highlight active section on scroll
- [ ] Make collapsible on mobile

---

### ✅ Task 2.2: Improve Header Hierarchy
**Status**: 🔲 Not Started
**Estimated Time**: 1-2 hours
**Impact**: Medium - Improves scanability

**Implementation**:
- [ ] Increase h1 size (2.5rem → 3rem)
- [ ] Better spacing between headings
- [ ] Consistent font weights
- [ ] Add subtle color variations

---

### ✅ Task 2.3: Make Search More Prominent
**Status**: 🔲 Not Started
**Estimated Time**: 1 hour
**Impact**: Medium - Improves discoverability

**Implementation**:
- [ ] Increase search button size
- [ ] Add keyboard shortcut hint (⌘K / Ctrl+K)
- [ ] Improve hover states
- [ ] Consider adding search icon to header

---

### ✅ Task 2.4: Clarify Breadcrumb Clickability
**Status**: 🔲 Not Started
**Estimated Time**: 30 minutes
**Impact**: Low-Medium - UX polish

**Implementation**:
- [ ] Add underline on hover
- [ ] Change cursor to pointer
- [ ] Add subtle color change
- [ ] Ensure separator doesn't look clickable

---

## Phase 3: Polish & Features (Priority 3)

### ✅ Task 3.1: Add Code Copy Buttons
**Status**: 🔲 Not Started
**Estimated Time**: 2-3 hours
**Impact**: Medium - Developer convenience

**Implementation**:
- [ ] Add copy button to all code blocks
- [ ] Show "Copied!" feedback
- [ ] Position in top-right of code block
- [ ] Handle multi-line code properly

---

### ✅ Task 3.2: Add Sidebar Collapse/Expand
**Status**: 🔲 Not Started
**Estimated Time**: 2-3 hours
**Impact**: Medium - Screen space optimization

**Implementation**:
- [ ] Add collapse button to sidebar
- [ ] Animate collapse/expand
- [ ] Save state to localStorage
- [ ] Show icon-only version when collapsed

---

### ✅ Task 3.3: Improve Link Styling
**Status**: 🔲 Not Started
**Estimated Time**: 1 hour
**Impact**: Low - Visual polish

**Implementation**:
- [ ] Add subtle underline to content links
- [ ] Distinct color for links
- [ ] Hover states
- [ ] Visited link styling

---

### ✅ Task 3.4: Add Footer
**Status**: 🔲 Not Started
**Estimated Time**: 1-2 hours
**Impact**: Low - Completeness

**Implementation**:
- [ ] Add footer with version info
- [ ] Links to GitHub, docs, etc.
- [ ] Copyright/license info
- [ ] "Edit on GitHub" link

---

## Phase 4: Responsive & Accessibility (Priority 4)

### ✅ Task 4.1: Add Loading States
**Status**: 🔲 Not Started
**Estimated Time**: 2-3 hours
**Impact**: Medium - Perceived performance

**Implementation**:
- [ ] Skeleton screens for page loads
- [ ] Loading spinner for search
- [ ] Smooth transitions
- [ ] Handle slow connections

---

### ✅ Task 4.2: Full Accessibility Audit
**Status**: 🔲 Not Started
**Estimated Time**: 3-4 hours
**Impact**: High - Inclusivity

**Implementation**:
- [ ] ARIA labels for all interactive elements
- [ ] Keyboard navigation testing
- [ ] Screen reader testing
- [ ] Color contrast verification (WCAG AA)
- [ ] Focus indicators

---

## Testing & Verification

### E2E Tests to Create/Update
- [ ] Typography and layout responsive tests
- [ ] Active page indicator visibility test
- [ ] Prev/next navigation functionality
- [ ] Search preview length verification
- [ ] ToC generation and navigation
- [ ] Code copy button functionality
- [ ] Sidebar collapse/expand
- [ ] Keyboard navigation flows
- [ ] Screen reader compatibility

### Screenshot Verification
After each phase, capture new screenshots to verify improvements:
- [ ] Phase 1 - Before/after screenshots
- [ ] Phase 2 - Before/after screenshots
- [ ] Phase 3 - Before/after screenshots
- [ ] Phase 4 - Before/after screenshots

---

## Summary Statistics

**Total Tasks**: 14
**Phase 1 (Critical)**: 4 tasks
**Phase 2 (Enhanced Nav)**: 4 tasks
**Phase 3 (Polish)**: 4 tasks
**Phase 4 (Responsive/A11y)**: 2 tasks

**Current Status**: Phase 1 - In Progress
**Last Updated**: November 21, 2025
