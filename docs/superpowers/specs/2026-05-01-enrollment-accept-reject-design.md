# Enrollment Accept/Reject — Design Spec

## Problem

The master needs to accept or reject player enrollments in matches. Currently, enrollment is binary — a player enrolls a character sheet and it is immediately active. There is no approval step, so the master has no control over which characters participate.

## Solution

Add a `status` column to the `enrollments` table with three states: `pending`, `accepted`, `rejected`. The master can transition enrollments between states freely until the match starts.

## Status Lifecycle

```
    ┌──────────┐
    │ pending  │
    └────┬─────┘
         │
    ┌────┴────┐
    ▼         ▼
┌────────┐ ┌──────────┐
│accepted│◄►│ rejected │
└────────┘ └──────────┘
```

**Allowed transitions:**
- `pending → accepted`
- `pending → rejected`
- `rejected → accepted`
- `accepted → rejected`

**Idempotent behavior:** Accepting an already-accepted enrollment (or rejecting an already-rejected one) returns success without error.

**Temporal guard (TODO):** Once the match starts, no status transitions are allowed. The match start lifecycle is WIP — this guard will be implemented when the match start feature is ready.

## Schema Changes

### Migration: Add `status` column

```sql
-- +goose Up
ALTER TABLE enrollments ADD COLUMN status TEXT NOT NULL DEFAULT 'pending';
-- Existing rows receive 'pending' via DEFAULT

-- Remove the global UNIQUE constraint on character_sheet_uuid
ALTER TABLE enrollments DROP CONSTRAINT enrollments_character_sheet_uuid_key;

-- Partial unique: only one non-rejected enrollment per character sheet
CREATE UNIQUE INDEX idx_enrollments_active_sheet
ON enrollments (character_sheet_uuid)
WHERE status != 'rejected';
```

Existing enrollments are migrated to `pending` — the master must accept them explicitly.

## Domain Layer

### New Use Cases (`internal/domain/enrollment/`)

#### `AcceptEnrollmentUC`
Interface: `IAcceptEnrollment`
Method: `Accept(ctx, sheetUUID, matchUUID, masterUUID) error`

Flow:
1. Get enrollment status and match_uuid via `GetEnrollmentStatus(sheetUUID, matchUUID)`
2. If not found → `ErrEnrollmentNotFound`
3. If already `accepted` → return nil (idempotent)
4. Get campaign_uuid via `matchRepo.GetMatchCampaignUUID(matchUUID)`
5. Get campaign master via `campaignRepo.GetCampaignMasterUUID(campaignUUID)`
6. If masterUUID != campaign master → `ErrNotMatchMaster`
7. TODO: Check if match has started → `ErrMatchAlreadyStarted`
8. Call `repo.AcceptEnrollment(sheetUUID, matchUUID)`

#### `RejectEnrollmentUC`
Interface: `IRejectEnrollment`
Method: `Reject(ctx, sheetUUID, matchUUID, masterUUID) error`

Flow:
1. Get enrollment status and match_uuid via `GetEnrollmentStatus(sheetUUID, matchUUID)`
2. If not found → `ErrEnrollmentNotFound`
3. If already `rejected` → return nil (idempotent)
4. Get campaign_uuid via `matchRepo.GetMatchCampaignUUID(matchUUID)`
5. Get campaign master via `campaignRepo.GetCampaignMasterUUID(campaignUUID)`
6. If masterUUID != campaign master → `ErrNotMatchMaster`
7. TODO: Check if match has started → `ErrMatchAlreadyStarted`
8. Call `repo.RejectEnrollment(sheetUUID, matchUUID)`

### New Domain Errors (`internal/domain/enrollment/error.go`)

```go
ErrEnrollmentNotFound = domain.NewValidationError(
    errors.New("enrollment not found"))
ErrNotMatchMaster = domain.NewValidationError(
    errors.New("user is not the match's campaign master"))
```

### Expanded Repository Interface (`internal/domain/enrollment/i_repository.go`)

New methods:
- `GetEnrollmentStatus(ctx, sheetUUID, matchUUID) (string, error)` — returns current status
- `AcceptEnrollment(ctx, sheetUUID, matchUUID) error` — sets status to `accepted`
- `RejectEnrollment(ctx, sheetUUID, matchUUID) error` — sets status to `rejected`

## Gateway Layer (`internal/gateway/pg/enrollment/`)

### `error.go`
Gateway-level sentinel: `ErrEnrollmentNotFound` (domain UC maps it to its own domain error, following the submission pattern).

### `read_enrollment_status.go`
```sql
SELECT status FROM enrollments
WHERE character_sheet_uuid = $1 AND match_uuid = $2
```
Returns gateway `ErrEnrollmentNotFound` sentinel if no rows.

### `accept_enrollment.go`
```sql
UPDATE enrollments SET status = 'accepted'
WHERE character_sheet_uuid = $1 AND match_uuid = $2
```
Wrapped in transaction following existing pattern.

### `reject_enrollment.go`
```sql
UPDATE enrollments SET status = 'rejected'
WHERE character_sheet_uuid = $1 AND match_uuid = $2
```
Wrapped in transaction following existing pattern.

## App/Handler Layer (`internal/app/api/enrollment/`)

### `accept_enrollment.go`

```
POST /enrollments/{sheet_uuid}/{match_uuid}/accept
```

Request: path params `sheet_uuid`, `match_uuid`
Response: `{ "status": 200 }`

Error mapping:
- `ErrEnrollmentNotFound` → 404
- `ErrMatchNotFound` → 404
- `ErrCampaignNotFound` → 404
- `ErrNotMatchMaster` → 403
- default → 500

### `reject_enrollment.go`

```
POST /enrollments/{sheet_uuid}/{match_uuid}/reject
```

Request: path params `sheet_uuid`, `match_uuid`
Response: `{ "status": 204 }`

Error mapping: same as accept.

### Routes

Both handlers are added to the `Api` struct in `routes.go` and registered with Huma.

## Wiring (`cmd/api/main.go`)

Instantiate both UCs with dependencies:
- `enrollmentRepo` (for enrollment status read/update)
- `matchRepo` (for match → campaign lookup)
- `campaignRepo` (for campaign master ownership check)

Wire into `enrollmentApi` struct alongside the existing `EnrollCharacterHandler`.

## Testing

### Domain Tests (`internal/domain/enrollment/`)
Table-driven tests for each UC covering:
- Success case (pending → accepted/rejected)
- Success case (rejected → accepted, accepted → rejected)
- Idempotent case (already in target status)
- Enrollment not found
- Match not found
- Campaign not found
- Not campaign master
- Repository error

### Handler Tests (`internal/app/api/enrollment/`)
Humatest-based tests with mocks following the existing `mocks_test.go` pattern:
- Happy path (200/204)
- Error mapping (404, 403, 500)
- Invalid UUID in path (400)

## Implementation Phases

**Phase 1 — Accept Enrollment:** Migration + domain UC + gateway + handler + tests + wiring
**Phase 2 — Reject Enrollment:** Domain UC + gateway + handler + tests + wiring (migration shared from Phase 1)
