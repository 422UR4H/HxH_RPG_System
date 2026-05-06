# Expose master_uuid in Match Responses

## Context

`GET /matches/{uuid}` and `POST /matches` both return `MatchResponse`. The `MasterUUID` field already exists on the `Match` entity and propagates through the domain and use-case layers untouched — it was simply never mapped into the HTTP response struct.

## Change

Add `master_uuid` to the shared `MatchResponse` struct and populate it in both handlers.

### Affected files

| File | Change |
|------|--------|
| `internal/app/api/match/create_match.go` | Add `MasterUUID uuid.UUID \`json:"master_uuid"\`` to `MatchResponse`; populate in `CreateMatchHandler` |
| `internal/app/api/match/get_match.go` | Populate `MasterUUID` in `GetMatchHandler` |
| `internal/app/api/match/get_match_test.go` | Assert `master_uuid` field in success case |
| `internal/app/api/match/create_match_test.go` | Assert `master_uuid` field in success case |

### No changes required in

- Domain entities (`match.Match.MasterUUID` already exists)
- Use cases (`GetMatchUC` and `CreateMatchUC` already return full `*Match`)
- Gateway / repository layer

## JSON contract

```json
{
  "match": {
    "uuid": "...",
    "master_uuid": "...",
    "campaign_uuid": "...",
    ...
  }
}
```
