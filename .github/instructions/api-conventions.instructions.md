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

## Validation in Handlers

Never use Huma constraint tags (`maxLength`, `minLength`, `enum`) on request body fields. Huma intercepts before the handler and returns generic `"validation failed"` with no context.

All business validation belongs in the use case. The handler maps `domain.ErrValidation` → `huma.Error422UnprocessableEntity(err.Error())`.

```go
// ❌ Name string `json:"name" maxLength:"32"`
// ✅ Name string `json:"name" doc:"Name (5-32 characters)"`
```

Allowed tags: `required`, `doc`, `default`, `json`, `path`.
Exception: `enum` on path parameters is fine when the use case also validates.
