# PATCH /campaigns/{uuid}

Update a campaign's editable fields. Two runtime modes apply based on whether
any match in the campaign has been started (`game_start_at IS NOT NULL`).

## Auth
JWT required (`Authorization: Bearer <token>`). Only the campaign's master may call this endpoint.

## Path Parameters
| Param | Type | Description |
|-------|------|-------------|
| uuid  | UUID | Campaign UUID |

## Request Body (all fields optional)

```json
{
  "name": "string (5-32 chars)",
  "brief_initial_description": "string (max 255)",
  "description": "string",
  "is_public": true,
  "call_link": "string (max 255)",
  "story_start_at": "YYYY-MM-DD",
  "story_current_at": "ISO 8601 (e.g. 2026-07-20T10:00:00Z)"
}
```

Empty body is a valid noop (returns current state).

## Field Availability by Mode

| Field | Free (no match started) | Restricted (match started) |
|-------|------------------------|---------------------------|
| `name` | ✅ editable | ❌ locked |
| `story_start_at` | ✅ editable | ❌ locked |
| `story_current_at` | ✅ any value | ✅ cannot go earlier than current value |
| `brief_initial_description` | ✅ | ✅ |
| `description` | ✅ | ✅ |
| `is_public` | ✅ | ✅ |
| `call_link` | ✅ | ✅ |

**`story_current_at` non-regression:** if the campaign already has a `story_current_at` value, the new value must be ≥ the current one. If the current value is null, any value is accepted.

## Success Response `200 OK`

```json
{
  "campaign": {
    "uuid": "uuid",
    "master_uuid": "uuid",
    "name": "string",
    "brief_initial_description": "string",
    "description": "string",
    "is_public": true,
    "call_link": "string",
    "story_start_at": "YYYY-MM-DD",
    "story_current_at": "ISO 8601 | omitted if null",
    "updated_at": "RFC 1123"
  }
}
```

## Error Responses

| Status | Condition |
|--------|-----------|
| 403 | Caller is not the campaign master |
| 404 | Campaign not found |
| 422 | Campaign has ended (story_end_at set) |
| 422 | `name` or `story_start_at` sent after match has started |
| 422 | `story_current_at` would go back in time |
| 422 | Validation error (name length, brief desc length, call link length) |
| 422 | Invalid date format |
| 500 | Unexpected server error |
