---
applyTo: "internal/gateway/**"
---

# Gateway Conventions

## Timestamps: Go, Not SQL

Generate all timestamps with `time.Now()` in Go and pass as query parameters (`$N`).
Never use `NOW()`, `CURRENT_DATE`, or `CURRENT_TIMESTAMP` in runtime SQL queries.

**Why:** Keeps timestamp source in the application layer (testable, consistent, timezone-aware).
DDL column defaults (`DEFAULT CURRENT_TIMESTAMP`) in migrations are fine — they're schema-level fallbacks.

```go
// ✅ correct
now := time.Now()
_, err = tx.Exec(ctx, `UPDATE matches SET updated_at = $1 WHERE uuid = $2`, now, matchUUID)

// ❌ wrong
_, err = tx.Exec(ctx, `UPDATE matches SET updated_at = NOW() WHERE uuid = $1`, matchUUID)
```

This applies to both production gateways AND test helpers (`pgtest/`).
