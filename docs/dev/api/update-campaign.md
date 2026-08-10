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
  "briefInitialDescription": "string (max 255)",
  "description": "string",
  "isPublic": true,
  "callLink": "string (max 255)",
  "storyStartAt": "YYYY-MM-DD",
  "storyCurrentAt": "ISO 8601 (e.g. 2026-07-20T10:00:00Z)"
}
```

Empty body is a valid noop (returns current state).

## Field Availability by Mode

| Field | Free (no match started) | Restricted (match started) |
|-------|------------------------|---------------------------|
| `name` | ✅ editable | ❌ locked |
| `storyStartAt` | ✅ editable | ❌ locked |
| `storyCurrentAt` | ✅ any value | ✅ cannot go earlier than current value |
| `briefInitialDescription` | ✅ | ✅ |
| `description` | ✅ | ✅ |
| `isPublic` | ✅ | ✅ |
| `callLink` | ✅ | ✅ |

**`storyCurrentAt` non-regression:** if the campaign already has a `storyCurrentAt` value, the new value must be ≥ the current one. If the current value is null, any value is accepted.

## Success Response `200 OK`

```json
{
  "campaign": {
    "uuid": "uuid",
    "masterUuid": "uuid",
    "name": "string",
    "briefInitialDescription": "string",
    "description": "string",
    "isPublic": true,
    "callLink": "string",
    "storyStartAt": "YYYY-MM-DD",
    "storyCurrentAt": "ISO 8601 | omitted if null",
    "updatedAt": "RFC 1123"
  }
}
```

## Error Responses

| Status | Condition |
|--------|-----------|
| 403 | Caller is not the campaign master |
| 404 | Campaign not found |
| 422 | Campaign has ended (storyEndAt set) |
| 422 | `name` or `storyStartAt` sent after match has started |
| 422 | `storyCurrentAt` would go back in time |
| 422 | Validation error (name length, brief desc length, call link length) |
| 422 | Invalid date format |
| 500 | Unexpected server error |
