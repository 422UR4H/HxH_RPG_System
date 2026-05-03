---
applyTo: "internal/app/api/**"
---

# API Conventions

## Route Design

One resource identifier per URL path segment. Never group multiple IDs in a single segment.

```
✅ /enrollments/{uuid}/accept
❌ /enrollments/{sheet_uuid}/{match_uuid}/accept
```
