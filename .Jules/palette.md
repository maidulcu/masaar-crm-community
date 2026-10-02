# Palette's Journal - Critical UX & Accessibility Learnings

## 2025-05-18 - Notification Popover Accessibility Pattern
**Learning:** Disclosure popovers (like notification bells) need dynamic ARIA labels (announcing unread item counts) and Escape key handling to be accessible to screen reader and keyboard-only users.
**Action:** When building dropdowns or popovers, always include `aria-expanded`, dynamic `aria-label`, `Escape` key close handler, and `focus-visible` focus rings.
