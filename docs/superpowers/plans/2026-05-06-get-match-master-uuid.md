# Expose master_uuid in Match Responses — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `master_uuid` to the `MatchResponse` JSON returned by `GET /matches/{uuid}` and `POST /matches`.

**Architecture:** `MasterUUID` already exists on `matchEntity.Match` and is returned by both use cases — the only gap is the HTTP response struct and the two handler mappings. No domain, gateway, or entity changes needed.

**Tech Stack:** Go 1.23, github.com/danielgtaylor/huma/v2, github.com/danielgtaylor/huma/v2/humatest, github.com/google/uuid

---

### Task 1: Expose master_uuid in GetMatchHandler (TDD)

**Files:**
- Modify: `internal/app/api/match/get_match_test.go`
- Modify: `internal/app/api/match/create_match.go` (MatchResponse struct lives here)
- Modify: `internal/app/api/match/get_match.go`

- [ ] **Step 1: Write the failing assertion in TestGetMatchHandler**

In `internal/app/api/match/get_match_test.go`, inside the `"success"` sub-test, add an assertion for `master_uuid` after the existing `title` check:

```go
if tt.wantStatus == http.StatusOK {
    var result map[string]any
    if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
        t.Fatalf("failed to unmarshal response: %v", err)
    }
    matchData, ok := result["match"].(map[string]any)
    if !ok {
        t.Fatal("response missing 'match' field")
    }
    if matchData["title"] != "My Match" {
        t.Errorf("got title %v, want 'My Match'", matchData["title"])
    }
    if matchData["master_uuid"] != userUUID.String() {
        t.Errorf("got master_uuid %v, want %v", matchData["master_uuid"], userUUID.String())
    }
}
```

Note: the mock's `"success"` case already sets `MasterUUID: uid` (which equals `userUUID` passed as context), so the expected value is `userUUID.String()`.

- [ ] **Step 2: Run the test to confirm it fails**

```bash
go test ./internal/app/api/match/... -run TestGetMatchHandler/success -v
```

Expected: FAIL — `got master_uuid <nil>, want <some-uuid>`

- [ ] **Step 3: Add MasterUUID field to MatchResponse**

In `internal/app/api/match/create_match.go`, add the field to `MatchResponse` after `UUID`:

```go
type MatchResponse struct {
    UUID                    uuid.UUID `json:"uuid"`
    MasterUUID              uuid.UUID `json:"master_uuid"`
    CampaignUUID            uuid.UUID `json:"campaign_uuid"`
    Title                   string    `json:"title"`
    BriefInitialDescription string    `json:"brief_initial_description"`
    BriefFinalDescription   *string   `json:"brief_final_description,omitempty"`
    Description             string    `json:"description"`
    IsPublic                bool      `json:"is_public"`
    GameScheduledAt         string    `json:"game_scheduled_at"`
    GameStartAt             *string   `json:"game_start_at,omitempty"`
    StoryStartAt            string    `json:"story_start_at"`
    StoryEndAt              *string   `json:"story_end_at,omitempty"`
    CreatedAt               string    `json:"created_at"`
    UpdatedAt               string    `json:"updated_at"`
}
```

- [ ] **Step 4: Populate MasterUUID in GetMatchHandler**

In `internal/app/api/match/get_match.go`, add `MasterUUID` to the `MatchResponse` literal (the block starting at `response := MatchResponse{`):

```go
response := MatchResponse{
    UUID:                    match.UUID,
    MasterUUID:              match.MasterUUID,
    CampaignUUID:            match.CampaignUUID,
    Title:                   match.Title,
    BriefInitialDescription: match.BriefInitialDescription,
    BriefFinalDescription:   match.BriefFinalDescription,
    Description:             match.Description,
    IsPublic:                match.IsPublic,
    GameScheduledAt:         match.GameScheduledAt.Format(time.RFC3339),
    GameStartAt:             gameStartAtStr,
    StoryStartAt:            match.StoryStartAt.Format("2006-01-02"),
    StoryEndAt:              storyEndAtStr,
    CreatedAt:               match.CreatedAt.Format(http.TimeFormat),
    UpdatedAt:               match.UpdatedAt.Format(http.TimeFormat),
}
```

- [ ] **Step 5: Run all match handler tests**

```bash
go test ./internal/app/api/match/... -v
```

Expected: all PASS (including `TestGetMatchHandler/success`)

- [ ] **Step 6: Commit**

```bash
git add internal/app/api/match/create_match.go internal/app/api/match/get_match.go internal/app/api/match/get_match_test.go
git commit -m "feat: expose master_uuid in GetMatch response"
```

---

### Task 2: Expose master_uuid in CreateMatchHandler (TDD)

**Files:**
- Modify: `internal/app/api/match/create_match_test.go`
- Modify: `internal/app/api/match/create_match.go`

- [ ] **Step 1: Write the failing assertion in TestCreateMatchHandler**

In `internal/app/api/match/create_match_test.go`, inside the `"success"` sub-test, add an assertion for `master_uuid` after the existing `title` check:

```go
if tt.wantStatus == http.StatusCreated {
    var result map[string]any
    if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
        t.Fatalf("failed to unmarshal response: %v", err)
    }
    matchData, ok := result["match"].(map[string]any)
    if !ok {
        t.Fatal("response missing 'match' field")
    }
    if matchData["title"] != "Test Match" {
        t.Errorf("got title %v, want 'Test Match'", matchData["title"])
    }
    if matchData["master_uuid"] != userUUID.String() {
        t.Errorf("got master_uuid %v, want %v", matchData["master_uuid"], userUUID.String())
    }
}
```

Note: the mock already sets `MasterUUID: input.MasterUUID`, and `input.MasterUUID` is set from the context `userUUID` inside `CreateMatchHandler`.

- [ ] **Step 2: Run the test to confirm it fails**

```bash
go test ./internal/app/api/match/... -run TestCreateMatchHandler/success -v
```

Expected: FAIL — `got master_uuid <nil or zero-uuid>, want <userUUID>`

(The field exists in the struct from Task 1, but `CreateMatchHandler` doesn't populate it yet, so it serializes as the zero UUID `"00000000-0000-0000-0000-000000000000"`.)

- [ ] **Step 3: Populate MasterUUID in CreateMatchHandler**

In `internal/app/api/match/create_match.go`, add `MasterUUID` to the `MatchResponse` literal (the block starting at `response := MatchResponse{`):

```go
response := MatchResponse{
    UUID:                    match.UUID,
    MasterUUID:              match.MasterUUID,
    CampaignUUID:            match.CampaignUUID,
    Title:                   match.Title,
    BriefInitialDescription: match.BriefInitialDescription,
    BriefFinalDescription:   match.BriefFinalDescription,
    Description:             match.Description,
    IsPublic:                match.IsPublic,
    GameScheduledAt:         match.GameScheduledAt.Format(time.RFC3339),
    GameStartAt:             gameStartAtStr,
    StoryStartAt:            match.StoryStartAt.Format("2006-01-02"),
    CreatedAt:               match.CreatedAt.Format(http.TimeFormat),
    UpdatedAt:               match.UpdatedAt.Format(http.TimeFormat),
}
```

- [ ] **Step 4: Run all match handler tests**

```bash
go test ./internal/app/api/match/... -v
```

Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add internal/app/api/match/create_match.go internal/app/api/match/create_match_test.go
git commit -m "feat: expose master_uuid in CreateMatch response"
```
