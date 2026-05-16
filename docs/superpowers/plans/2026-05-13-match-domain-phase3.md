# Match Domain Phase 3 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete the match bounded context with Scene Management, Turn/Round/Scene/Action persistence, and EnqueueMasterAction end-to-end flow.

**Architecture:** Three concerns layered sequentially: DB schema (migrations) → domain changes (Round/Scene IDs, MatchSession) → gateway (atomic persistence) → application (use cases) → WS layer (ChangeScene, EnqueueMasterAction). Each task produces working, tested, committed code.

**Tech Stack:** Go 1.23, PostgreSQL (goose migrations), pgx/v5, standard `testing` package, table-driven tests, `go test -tags=integration` for gateway tests.

---

## Key Conventions (read before every task)

- **Timestamps**: Always `time.Now()` in Go, passed as `$N` params. Never SQL `NOW()` in runtime queries.
- **Transaction rollback**:
  ```go
  defer func() {
      if p := recover(); p != nil {
          _ = tx.Rollback(ctx)
          panic(p)
      }
      _ = tx.Rollback(ctx) // no-op after Commit
  }()
  ```
- **Test packages**: External — `package round_test`, `package scene_test`, `package matchsession_test`, etc.
- **Integration tests**: Build tag `//go:build integration` on first line, file named `<pkg>_integration_test.go`.
- **Commits**: Include `Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>` in every commit.
- **Module**: `github.com/422UR4H/HxH_RPG_System`
- **Working dir root**: `internal/`, `migrations/`, `docs/` at repo root.

---

## Task 1: DB Migrations

**Goal:** Create 4 goose migration files for scenes, rounds, turns, and actions tables.

**No TDD needed** — these are SQL schema files. Verify with `go build ./...` (no DB required).

### Steps

- [ ] Create `migrations/20260513000001_add_scenes_table.sql`:

```sql
-- +goose Up
-- +goose StatementBegin
BEGIN;

CREATE TABLE IF NOT EXISTS scenes (
    id          SERIAL PRIMARY KEY,
    uuid        UUID NOT NULL DEFAULT gen_random_uuid(),
    match_uuid  UUID NOT NULL REFERENCES matches(uuid),
    category    VARCHAR(32) NOT NULL,
    brief_initial_description VARCHAR(255) NOT NULL DEFAULT '',
    brief_final_description   VARCHAR(255),
    created_at  TIMESTAMP NOT NULL,
    finished_at TIMESTAMP,
    UNIQUE (uuid)
);
CREATE INDEX IF NOT EXISTS idx_scenes_match_uuid ON scenes(match_uuid);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

DROP TABLE IF EXISTS scenes;

COMMIT;
-- +goose StatementEnd
```

- [ ] Create `migrations/20260513000002_add_rounds_table.sql`:

```sql
-- +goose Up
-- +goose StatementBegin
BEGIN;

CREATE TABLE IF NOT EXISTS rounds (
    id         SERIAL PRIMARY KEY,
    uuid       UUID NOT NULL DEFAULT gen_random_uuid(),
    scene_uuid UUID NOT NULL REFERENCES scenes(uuid),
    mode       VARCHAR(16) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    finished_at TIMESTAMP,
    UNIQUE (uuid)
);
CREATE INDEX IF NOT EXISTS idx_rounds_scene_uuid ON rounds(scene_uuid);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

DROP TABLE IF EXISTS rounds;

COMMIT;
-- +goose StatementEnd
```

- [ ] Create `migrations/20260513000003_add_turns_table.sql`:

```sql
-- +goose Up
-- +goose StatementBegin
BEGIN;

CREATE TABLE IF NOT EXISTS turns (
    id          SERIAL PRIMARY KEY,
    uuid        UUID NOT NULL DEFAULT gen_random_uuid(),
    round_uuid  UUID NOT NULL REFERENCES rounds(uuid),
    created_at  TIMESTAMP NOT NULL,
    finished_at TIMESTAMP NOT NULL,
    UNIQUE (uuid)
);
CREATE INDEX IF NOT EXISTS idx_turns_round_uuid ON turns(round_uuid);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

DROP TABLE IF EXISTS turns;

COMMIT;
-- +goose StatementEnd
```

- [ ] Create `migrations/20260513000004_add_actions_table.sql`:

```sql
-- +goose Up
-- +goose StatementBegin
BEGIN;

CREATE TABLE IF NOT EXISTS actions (
    id           SERIAL PRIMARY KEY,
    uuid         UUID NOT NULL DEFAULT gen_random_uuid(),
    turn_uuid    UUID NOT NULL REFERENCES turns(uuid),
    actor_uuid   UUID NOT NULL REFERENCES users(uuid),
    react_to_uuid UUID REFERENCES actions(uuid),
    target_ids   UUID[] NOT NULL DEFAULT '{}',
    type         VARCHAR(32) NOT NULL,
    speed        JSONB,
    skills       JSONB,
    move         JSONB,
    attack       JSONB,
    defense      JSONB,
    dodge        JSONB,
    feint        JSONB,
    trigger      JSONB,
    created_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (uuid)
);
CREATE INDEX IF NOT EXISTS idx_actions_turn_uuid ON actions(turn_uuid);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

DROP TABLE IF EXISTS actions;

COMMIT;
-- +goose StatementEnd
```

- [ ] Verify: `go build ./...` — must compile with no errors.

- [ ] Commit:
  ```
  feat(migrations): add scenes, rounds, turns, actions tables

  Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
  ```

---

## Task 2: Domain Entities — Round, Scene, Turn, MasterAction

**Goal:** Add `id`/`createdAt` to Round, `id` + `Close()` + `GetID()` to Scene, `AddMasterAction` to Turn, `SetHappenedAt` + exported `Skills` to MasterAction. Add `Reconstruct*` factory functions for DB hydration.

**TDD**: Write unit tests first for each new method, run `go test ./internal/domain/...` red→green.

### 2a — Round: add ID and createdAt

- [ ] Open `internal/domain/match/entity/round/round.go`. Add to `Round` struct:
  ```go
  id        uuid.UUID
  createdAt time.Time
  ```
  Update `NewRound`:
  ```go
  func NewRound(mode enum.RoundMode) *Round {
      return &Round{
          id:        uuid.New(),
          mode:      mode,
          turns:     []*turn.Turn{},
          events:    []GameEvent{},
          createdAt: time.Now(),
      }
  }
  ```
  Add methods:
  ```go
  func (r *Round) GetID() uuid.UUID      { return r.id }
  func (r *Round) GetCreatedAt() time.Time { return r.createdAt }
  ```
  Add reconstruct factory (for DB hydration — does NOT call `uuid.New()`):
  ```go
  func ReconstructRound(id uuid.UUID, mode enum.RoundMode, createdAt time.Time) *Round {
      return &Round{
          id:        id,
          mode:      mode,
          turns:     []*turn.Turn{},
          events:    []GameEvent{},
          createdAt: createdAt,
      }
  }
  ```
  Add import `"github.com/google/uuid"` to the import block.

- [ ] Write tests **first** in `internal/domain/match/entity/round/round_test.go` (append to existing file):
  ```go
  func TestRound_GetID(t *testing.T) {
      r := round.NewRound(enum.Free)
      if r.GetID() == (uuid.UUID{}) {
          t.Error("expected non-zero ID from NewRound")
      }
  }

  func TestRound_GetCreatedAt(t *testing.T) {
      before := time.Now()
      r := round.NewRound(enum.Free)
      after := time.Now()
      if r.GetCreatedAt().Before(before) || r.GetCreatedAt().After(after) {
          t.Errorf("createdAt %v not in [%v, %v]", r.GetCreatedAt(), before, after)
      }
  }

  func TestReconstructRound(t *testing.T) {
      id := uuid.New()
      now := time.Now()
      r := round.ReconstructRound(id, enum.Race, now)
      if r.GetID() != id {
          t.Errorf("expected ID %v, got %v", id, r.GetID())
      }
      if r.GetMode() != enum.Race {
          t.Errorf("expected mode Race, got %v", r.GetMode())
      }
      if !r.GetCreatedAt().Equal(now) {
          t.Errorf("expected createdAt %v, got %v", now, r.GetCreatedAt())
      }
      if r.GetFinishedAt() != nil {
          t.Error("expected nil finishedAt on reconstructed round")
      }
  }
  ```

### 2b — Scene: add ID, Close(), GetID()

- [ ] Open `internal/domain/match/entity/scene/scene.go`. Add to `Scene` struct:
  ```go
  id uuid.UUID
  ```
  Update `NewScene`:
  ```go
  func NewScene(category enum.SceneCategory, briefInitialDescription string) *Scene {
      return &Scene{
          id:                      uuid.New(),
          category:                category,
          BriefInitialDescription: briefInitialDescription,
          createdAt:               time.Now(),
      }
  }
  ```
  Add methods:
  ```go
  func (s *Scene) GetID() uuid.UUID { return s.id }

  // Close sets finishedAt without requiring a briefFinalDescription.
  // Used by the system when transitioning scenes via ChangeScene.
  func (s *Scene) Close(at time.Time) {
      if s.finishedAt == nil {
          s.finishedAt = &at
      }
  }
  ```
  Add reconstruct factory:
  ```go
  func ReconstructScene(id uuid.UUID, category enum.SceneCategory, briefInitialDesc string, createdAt time.Time) *Scene {
      return &Scene{
          id:                      id,
          category:                category,
          BriefInitialDescription: briefInitialDesc,
          createdAt:               createdAt,
      }
  }
  ```
  Add import `"github.com/google/uuid"`.

- [ ] Write tests **first** in `internal/domain/match/entity/scene/scene_test.go` (append to existing file):
  ```go
  func TestScene_GetID(t *testing.T) {
      s := scene.NewScene(enum.Roleplay, "start")
      if s.GetID() == (uuid.UUID{}) {
          t.Error("expected non-zero ID from NewScene")
      }
  }

  func TestScene_Close(t *testing.T) {
      s := scene.NewScene(enum.Battle, "Arena")
      at := time.Now()
      s.Close(at)
      if s.GetFinishedAt() == nil {
          t.Error("expected finishedAt to be set after Close")
      }
  }

  func TestScene_Close_Idempotent(t *testing.T) {
      s := scene.NewScene(enum.Battle, "Arena")
      first := time.Now()
      s.Close(first)
      second := first.Add(time.Second)
      s.Close(second)
      // Should not overwrite — finishedAt stays as first
      if !s.GetFinishedAt().Equal(first) {
          t.Errorf("expected finishedAt %v, got %v", first, *s.GetFinishedAt())
      }
  }

  func TestReconstructScene(t *testing.T) {
      id := uuid.New()
      now := time.Now()
      s := scene.ReconstructScene(id, enum.Battle, "Forest", now)
      if s.GetID() != id {
          t.Errorf("expected ID %v, got %v", id, s.GetID())
      }
      if s.GetCategory() != enum.Battle {
          t.Errorf("expected Battle, got %v", s.GetCategory())
      }
      if s.BriefInitialDescription != "Forest" {
          t.Errorf("expected 'Forest', got %q", s.BriefInitialDescription)
      }
      if !s.GetCreatedAt().Equal(now) {
          t.Errorf("expected createdAt %v, got %v", now, s.GetCreatedAt())
      }
      if s.GetFinishedAt() != nil {
          t.Error("expected nil finishedAt")
      }
  }
  ```
  Add import block to test file: `"time"`, `"github.com/google/uuid"`.

### 2c — Turn: add AddMasterAction, remove nolint

- [ ] Open `internal/domain/match/entity/turn/turn.go`. Change field declaration from:
  ```go
  masterActions []action.MasterAction //nolint:unused // WIP: match system under development
  ```
  to:
  ```go
  masterActions []action.MasterAction
  ```
  Add method:
  ```go
  func (t *Turn) AddMasterAction(ma action.MasterAction) {
      t.masterActions = append(t.masterActions, ma)
  }
  ```

- [ ] Write test **first** in `internal/domain/match/entity/turn/turn_test.go` (append):
  ```go
  func TestTurn_AddMasterAction(t *testing.T) {
      a := action.NewAction(uuid.New(), nil, uuid.Nil, nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil)
      tRn := turn.NewTurn(*a)
      ma := action.NewMasterAction()

      tRn.AddMasterAction(*ma)

      got := tRn.GetMasterActions()
      if len(got) != 1 {
          t.Errorf("expected 1 master action, got %d", len(got))
      }
  }
  ```

### 2d — MasterAction: SetHappenedAt, export Skills

- [ ] Open `internal/domain/match/entity/action/master_action.go`. Change:
  ```go
  skills []Skill //nolint:unused // WIP: match system under development
  ```
  to:
  ```go
  Skills []Skill
  ```
  Update `GetSkills()` to use `ma.Skills`:
  ```go
  func (ma *MasterAction) GetSkills() []Skill {
      skillsCopy := make([]Skill, len(ma.Skills))
      copy(skillsCopy, ma.Skills)
      return skillsCopy
  }
  ```
  Add `SetHappenedAt`:
  ```go
  func (ma *MasterAction) SetHappenedAt(t time.Time) {
      ma.happenedAt = t
  }
  ```

- [ ] Write test **first** in a new file `internal/domain/match/entity/action/master_action_test.go`:
  ```go
  package action_test

  import (
      "testing"
      "time"

      "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
  )

  func TestMasterAction_SetHappenedAt(t *testing.T) {
      ma := action.NewMasterAction()
      if !ma.GetHappenedAt().IsZero() {
          t.Error("expected zero happenedAt on new MasterAction")
      }
      now := time.Now()
      ma.SetHappenedAt(now)
      if !ma.GetHappenedAt().Equal(now) {
          t.Errorf("expected happenedAt %v, got %v", now, ma.GetHappenedAt())
      }
  }

  func TestMasterAction_Skills(t *testing.T) {
      ma := action.NewMasterAction()
      ma.Skills = []action.Skill{{SkillName: "Gyo"}}
      got := ma.GetSkills()
      if len(got) != 1 {
          t.Fatalf("expected 1 skill, got %d", len(got))
      }
      if got[0].SkillName != "Gyo" {
          t.Errorf("expected SkillName 'Gyo', got %q", got[0].SkillName)
      }
  }
  ```

- [ ] Run: `go test ./internal/domain/...` — all must pass.

- [ ] Commit:
  ```
  feat(domain): add IDs, Close, Reconstruct factories to Round/Scene; AddMasterAction to Turn; SetHappenedAt to MasterAction

  Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
  ```

---

## Task 3: MatchSession — activeScene, flags, new methods

**Goal:** Add `activeScene *scene.Scene`, `scenePersisted bool`, `roundPersisted bool` to MatchSession. Add `ChangeScene`, `EnqueueMasterAction`, `NewMatchSessionWithState`, and flag accessors/setters.

**TDD**: Write failing tests first, then implement.

### 3a — Error sentinel

- [ ] Open `internal/domain/match/matchsession/error.go`. Add:
  ```go
  ErrNoActiveTurn = errors.New("no active turn in current round")
  ```

### 3b — New methods on MatchSession

- [ ] Open `internal/domain/match/matchsession/match_session.go`. Add imports for `scene` package. Add fields to struct:
  ```go
  activeScene    *scene.Scene
  scenePersisted bool
  roundPersisted bool
  ```

  Update `NewMatchSession` to init `activeScene`:
  ```go
  func NewMatchSession(
      matchUUID uuid.UUID,
      charSheets map[uuid.UUID]*csSheet.CharacterSheet,
      participants []*match.Participant,
  ) *MatchSession {
      pMap := make(map[uuid.UUID]*match.Participant, len(participants))
      for _, p := range participants {
          if p.Sheet.PlayerUUID != nil {
              pMap[*p.Sheet.PlayerUUID] = p
          }
      }
      return &MatchSession{
          matchUUID:    matchUUID,
          activeScene:  scene.NewScene(enum.Roleplay, ""),
          activeRound:  round.NewRound(enum.Free),
          activeQueue:  action.NewActionPriorityQueue(nil),
          charSheets:   charSheets,
          participants: pMap,
          roundOrch:    service.RoundOrchestrator{},
          combatRes:    service.CombatResolver{},
      }
  }
  ```

  Add `NewMatchSessionWithState` (used by InitMatchSessionUC when recovering persisted state):
  ```go
  func NewMatchSessionWithState(
      matchUUID uuid.UUID,
      charSheets map[uuid.UUID]*csSheet.CharacterSheet,
      participants []*match.Participant,
      activeScene *scene.Scene,
      activeRound *round.Round,
  ) *MatchSession {
      pMap := make(map[uuid.UUID]*match.Participant, len(participants))
      for _, p := range participants {
          if p.Sheet.PlayerUUID != nil {
              pMap[*p.Sheet.PlayerUUID] = p
          }
      }
      return &MatchSession{
          matchUUID:      matchUUID,
          activeScene:    activeScene,
          activeRound:    activeRound,
          activeQueue:    action.NewActionPriorityQueue(nil),
          charSheets:     charSheets,
          participants:   pMap,
          roundOrch:      service.RoundOrchestrator{},
          combatRes:      service.CombatResolver{},
          scenePersisted: true,
          roundPersisted: true,
      }
  }
  ```

  Add accessors:
  ```go
  func (s *MatchSession) GetMatchUUID() uuid.UUID      { return s.matchUUID }
  func (s *MatchSession) GetActiveScene() *scene.Scene { return s.activeScene }
  func (s *MatchSession) IsRoundPersisted() bool       { return s.roundPersisted }
  func (s *MatchSession) IsScenePersisted() bool       { return s.scenePersisted }

  // MarkRoundPersisted marks both scene and round as persisted after a successful PersistTurnClose call.
  func (s *MatchSession) MarkRoundPersisted() {
      s.scenePersisted = true
      s.roundPersisted = true
  }
  ```

  Add `ChangeScene`:
  ```go
  // ChangeScene closes the current scene and round, then starts a new scene with a new Free round.
  // Returns the old (closed) scene and old (closed) round.
  // Errors if the current round has an open turn.
  func (s *MatchSession) ChangeScene(category enum.SceneCategory, briefDesc string) (*scene.Scene, *round.Round, error) {
      if s.activeRound.HasOpenTurn() {
          return nil, nil, ErrRoundHasOpenTurn
      }
      now := time.Now()
      s.activeRound.Close(now)
      s.activeScene.Close(now)

      oldScene := s.activeScene
      oldRound := s.activeRound

      s.activeScene = scene.NewScene(category, briefDesc)
      s.activeRound = round.NewRound(enum.Free)
      s.scenePersisted = false
      s.roundPersisted = false

      return oldScene, oldRound, nil
  }
  ```

  Add `EnqueueMasterAction`:
  ```go
  // EnqueueMasterAction adds a master action to the current open turn.
  // Errors if there is no active (open) turn.
  func (s *MatchSession) EnqueueMasterAction(ma *action.MasterAction) error {
      t := s.activeRound.CurrentTurn()
      if t == nil || t.GetFinishedAt() != nil {
          return ErrNoActiveTurn
      }
      ma.SetHappenedAt(time.Now())
      t.AddMasterAction(*ma)
      return nil
  }
  ```

  Update `CloseRound` to preserve the scene:
  ```go
  func (s *MatchSession) CloseRound() (*round.Round, error) {
      if s.activeRound.HasOpenTurn() {
          return nil, ErrRoundHasOpenTurn
      }
      mode := s.activeRound.GetMode()
      closed := s.roundOrch.CloseRound(s.activeRound, time.Now())
      s.activeRound = round.NewRound(mode)
      s.roundPersisted = false
      return closed, nil
  }
  ```

### 3c — Tests

- [ ] Write all tests **before** implementing (red first). Append to `internal/domain/match/matchsession/match_session_test.go`:

  ```go
  func TestMatchSession_GetMatchUUID(t *testing.T) {
      id := uuid.New()
      s := matchsession.NewMatchSession(id, nil, nil)
      if s.GetMatchUUID() != id {
          t.Errorf("expected %v, got %v", id, s.GetMatchUUID())
      }
  }

  func TestMatchSession_GetActiveScene(t *testing.T) {
      s := matchsession.NewMatchSession(uuid.New(), nil, nil)
      if s.GetActiveScene() == nil {
          t.Fatal("expected non-nil active scene")
      }
      if s.GetActiveScene().GetCategory() != enum.Roleplay {
          t.Errorf("expected initial scene category Roleplay, got %v", s.GetActiveScene().GetCategory())
      }
  }

  func TestMatchSession_ChangeScene(t *testing.T) {
      t.Run("changes scene and resets round when no open turn", func(t *testing.T) {
          s := matchsession.NewMatchSession(uuid.New(), nil, nil)
          originalScene := s.GetActiveScene()
          originalRound := s.GetActiveRound()

          oldScene, oldRound, err := s.ChangeScene(enum.Battle, "Arena fight")
          if err != nil {
              t.Fatalf("unexpected error: %v", err)
          }
          if oldScene != originalScene {
              t.Error("expected returned old scene to be the original")
          }
          if oldRound != originalRound {
              t.Error("expected returned old round to be the original")
          }
          if oldScene.GetFinishedAt() == nil {
              t.Error("expected old scene to be closed")
          }
          if oldRound.GetFinishedAt() == nil {
              t.Error("expected old round to be closed")
          }
          if s.GetActiveScene() == originalScene {
              t.Error("expected new active scene after ChangeScene")
          }
          if s.GetActiveScene().GetCategory() != enum.Battle {
              t.Errorf("expected new scene category Battle, got %v", s.GetActiveScene().GetCategory())
          }
          if s.GetActiveRound() == originalRound {
              t.Error("expected new active round after ChangeScene")
          }
      })

      t.Run("returns ErrRoundHasOpenTurn when turn is open", func(t *testing.T) {
          playerA := uuid.New()
          s := sessionWithParticipants(playerA)
          s.EnqueueAction(playerA, makeActionWithSpeed(playerA, 5)) //nolint:errcheck
          s.OpenNextAction()                                         //nolint:errcheck

          _, _, err := s.ChangeScene(enum.Battle, "desc")
          if !errors.Is(err, matchsession.ErrRoundHasOpenTurn) {
              t.Errorf("expected ErrRoundHasOpenTurn, got %v", err)
          }
      })
  }

  func TestMatchSession_EnqueueMasterAction(t *testing.T) {
      t.Run("enqueues master action on current open turn", func(t *testing.T) {
          playerA := uuid.New()
          s := sessionWithParticipants(playerA)
          s.EnqueueAction(playerA, makeActionWithSpeed(playerA, 5)) //nolint:errcheck
          _, opened, _ := s.OpenNextAction()

          ma := action.NewMasterAction()
          if err := s.EnqueueMasterAction(ma); err != nil {
              t.Fatalf("unexpected error: %v", err)
          }
          if len(opened.GetMasterActions()) != 1 {
              t.Errorf("expected 1 master action on turn, got %d", len(opened.GetMasterActions()))
          }
          if ma.GetHappenedAt().IsZero() {
              t.Error("expected happenedAt to be set by EnqueueMasterAction")
          }
      })

      t.Run("returns ErrNoActiveTurn when no open turn", func(t *testing.T) {
          s := matchsession.NewMatchSession(uuid.New(), nil, nil)
          ma := action.NewMasterAction()
          err := s.EnqueueMasterAction(ma)
          if !errors.Is(err, matchsession.ErrNoActiveTurn) {
              t.Errorf("expected ErrNoActiveTurn, got %v", err)
          }
      })
  }

  func TestMatchSession_PersistenceFlags(t *testing.T) {
      t.Run("new session has flags false", func(t *testing.T) {
          s := matchsession.NewMatchSession(uuid.New(), nil, nil)
          if s.IsRoundPersisted() {
              t.Error("expected roundPersisted false on new session")
          }
          if s.IsScenePersisted() {
              t.Error("expected scenePersisted false on new session")
          }
      })

      t.Run("MarkRoundPersisted sets both flags", func(t *testing.T) {
          s := matchsession.NewMatchSession(uuid.New(), nil, nil)
          s.MarkRoundPersisted()
          if !s.IsRoundPersisted() {
              t.Error("expected roundPersisted true after MarkRoundPersisted")
          }
          if !s.IsScenePersisted() {
              t.Error("expected scenePersisted true after MarkRoundPersisted")
          }
      })

      t.Run("NewMatchSessionWithState has flags true", func(t *testing.T) {
          sc := scene.NewScene(enum.Battle, "Arena")
          r := round.NewRound(enum.Free)
          s := matchsession.NewMatchSessionWithState(uuid.New(), nil, nil, sc, r)
          if !s.IsRoundPersisted() {
              t.Error("expected roundPersisted true from WithState ctor")
          }
          if !s.IsScenePersisted() {
              t.Error("expected scenePersisted true from WithState ctor")
          }
          if s.GetActiveScene() != sc {
              t.Error("expected same scene pointer")
          }
          if s.GetActiveRound() != r {
              t.Error("expected same round pointer")
          }
      })

      t.Run("ChangeScene resets flags to false", func(t *testing.T) {
          sc := scene.NewScene(enum.Battle, "Arena")
          r := round.NewRound(enum.Free)
          s := matchsession.NewMatchSessionWithState(uuid.New(), nil, nil, sc, r)
          _, _, err := s.ChangeScene(enum.Roleplay, "Town")
          if err != nil {
              t.Fatalf("unexpected error: %v", err)
          }
          if s.IsRoundPersisted() {
              t.Error("expected roundPersisted false after ChangeScene")
          }
          if s.IsScenePersisted() {
              t.Error("expected scenePersisted false after ChangeScene")
          }
      })
  }
  ```
  Add imports for `scene` and `round` packages to test file.

- [ ] Run: `go test ./internal/domain/match/matchsession/...` — all must pass.

- [ ] Commit:
  ```
  feat(matchsession): add activeScene, ChangeScene, EnqueueMasterAction, persistence flags

  Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
  ```

---

## Task 4: pgtest helpers — InsertTestScene, InsertTestRound, InsertTestTurn

**Goal:** Add three helper functions to `internal/gateway/pg/pgtest/setup.go` for integration test prerequisites.

- [ ] Open `internal/gateway/pg/pgtest/setup.go`. Append after `InsertTestMatchParticipant`:

  ```go
  func InsertTestScene(t *testing.T, pool *pgxpool.Pool, matchUUID, category string) string {
      t.Helper()
      ctx := context.Background()
      now := time.Now()

      var sceneUUID string
      err := pool.QueryRow(ctx,
          `INSERT INTO scenes (match_uuid, category, brief_initial_description, created_at)
           VALUES ($1, $2, $3, $4) RETURNING uuid`,
          matchUUID, category, "test scene", now,
      ).Scan(&sceneUUID)
      if err != nil {
          t.Fatalf("failed to insert test scene: %v", err)
      }
      return sceneUUID
  }

  func InsertTestRound(t *testing.T, pool *pgxpool.Pool, sceneUUID, mode string) string {
      t.Helper()
      ctx := context.Background()
      now := time.Now()

      var roundUUID string
      err := pool.QueryRow(ctx,
          `INSERT INTO rounds (scene_uuid, mode, created_at) VALUES ($1, $2, $3) RETURNING uuid`,
          sceneUUID, mode, now,
      ).Scan(&roundUUID)
      if err != nil {
          t.Fatalf("failed to insert test round: %v", err)
      }
      return roundUUID
  }

  func InsertTestTurn(t *testing.T, pool *pgxpool.Pool, roundUUID string) string {
      t.Helper()
      ctx := context.Background()
      now := time.Now()

      var turnUUID string
      err := pool.QueryRow(ctx,
          `INSERT INTO turns (round_uuid, created_at, finished_at) VALUES ($1, $2, $3) RETURNING uuid`,
          roundUUID, now, now,
      ).Scan(&turnUUID)
      if err != nil {
          t.Fatalf("failed to insert test turn: %v", err)
      }
      return turnUUID
  }
  ```

- [ ] Verify: `go vet -tags=integration ./internal/gateway/pg/...` — no errors.

- [ ] Update `TruncateAll` to include the new tables. The truncate must drop them in dependency order (actions → turns → rounds → scenes). Replace the existing `TruncateAll` body:
  ```go
  func TruncateAll(t *testing.T, pool *pgxpool.Pool) {
      t.Helper()
      ctx := context.Background()

      _, err := pool.Exec(ctx, `
          TRUNCATE TABLE actions, turns, rounds, scenes,
              enrollments, submissions, sessions,
              match_participants, matches, campaigns, scenarios,
              joint_proficiencies, proficiencies, character_profiles,
              character_sheets, users
          CASCADE
      `)
      if err != nil {
          t.Fatalf("failed to truncate tables: %v", err)
      }
  }
  ```

- [ ] Verify again: `go vet -tags=integration ./internal/gateway/pg/...`

- [ ] Commit:
  ```
  feat(pgtest): add InsertTestScene, InsertTestRound, InsertTestTurn helpers; include new tables in TruncateAll

  Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
  ```

---

## Task 5: IRoundRepository interface + gateway skeleton

**Goal:** Define `IRoundRepository` in the application layer and create the gateway package skeleton.

### 5a — IRoundRepository

- [ ] Open `internal/application/match/i_repository.go`. Add at the bottom (new interface, separate from `IRepository`):

  ```go
  import (
      // existing imports...
      "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
      "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
      "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/scene"
      "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
      roundrepo "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/round"
  )

  type IRoundRepository interface {
      PersistTurnClose(ctx context.Context, sc *scene.Scene, r *round.Round, t *turn.Turn, act *action.Action, matchUUID uuid.UUID) error
      FindActiveSession(ctx context.Context, matchUUID uuid.UUID) (*roundrepo.ActiveSessionData, error)
      CloseSceneAndRound(ctx context.Context, sceneUUID, roundUUID uuid.UUID, at time.Time) error
      CloseRound(ctx context.Context, roundUUID uuid.UUID, at time.Time) error
  }
  ```

### 5b — Gateway skeleton

- [ ] Create directory `internal/gateway/pg/round/`.

- [ ] Create `internal/gateway/pg/round/repository.go`:
  ```go
  package round

  import (
      "time"

      "github.com/google/uuid"
      "github.com/jackc/pgx/v5/pgxpool"
  )

  // ActiveSessionData holds reconstructed scene+round identifiers returned by FindActiveSession.
  type ActiveSessionData struct {
      SceneID        uuid.UUID
      Category       string
      BriefInitDesc  string
      SceneCreatedAt time.Time
      RoundID        uuid.UUID
      Mode           string
      RoundCreatedAt time.Time
  }

  // Repository implements persistence for round-scoped operations (scenes, rounds, turns, actions).
  type Repository struct {
      pool *pgxpool.Pool
  }

  func NewRepository(pool *pgxpool.Pool) *Repository {
      return &Repository{pool: pool}
  }
  ```

- [ ] Verify: `go build ./...` — no errors.

- [ ] Commit:
  ```
  feat(gateway/round): add IRoundRepository interface and repository skeleton with ActiveSessionData

  Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
  ```

---

## Task 6: Gateway — PersistTurnClose

**Goal:** Implement atomic persist of scene (ON CONFLICT DO NOTHING) + round (ON CONFLICT DO NOTHING) + turn + action in one transaction.

### 6a — Write integration test first

- [ ] Create `internal/gateway/pg/round/round_integration_test.go`:

```go
//go:build integration

package round_test

import (
    "context"
    "testing"
    "time"

    "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
    "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
    roundentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
    sceneentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/scene"
    turnentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
    roundrepo "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/round"
    "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/pgtest"
    "github.com/google/uuid"
)

func TestPersistTurnClose(t *testing.T) {
    pool := pgtest.SetupTestDB(t)
    repo := roundrepo.NewRepository(pool)
    ctx := context.Background()

    masterUUID := pgtest.InsertTestUser(t, pool, "gm1", "gm1@test.com", "pass")
    campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "Camp1")
    matchUUID := pgtest.InsertTestMatch(t, pool, masterUUID, campaignUUID, "Match1")
    matchUUIDParsed, _ := uuid.Parse(matchUUID)
    actorUUIDParsed, _ := uuid.Parse(masterUUID)

    t.Run("happy path — persists scene, round, turn, action atomically", func(t *testing.T) {
        pgtest.TruncateAll(t, pool)
        // Re-insert after truncate
        masterUUID2 := pgtest.InsertTestUser(t, pool, "gm2", "gm2@test.com", "pass")
        campaignUUID2 := pgtest.InsertTestCampaign(t, pool, masterUUID2, "Camp2")
        matchUUID2 := pgtest.InsertTestMatch(t, pool, masterUUID2, campaignUUID2, "Match2")
        matchUUIDParsed2, _ := uuid.Parse(matchUUID2)
        actorUUIDParsed2, _ := uuid.Parse(masterUUID2)

        sc := sceneentity.NewScene(enum.Battle, "Arena")
        r := roundentity.NewRound(enum.Free)
        act := action.NewAction(actorUUIDParsed2, nil, uuid.Nil, nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil)
        closedAt := time.Now()
        act2 := *act
        tRn := turnentity.NewTurn(act2)
        tRn.Close(closedAt)

        err := repo.PersistTurnClose(ctx, sc, r, tRn, act, matchUUIDParsed2)
        if err != nil {
            t.Fatalf("PersistTurnClose error: %v", err)
        }

        // Verify scene row exists
        var sceneCount int
        _ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM scenes WHERE uuid = $1`, sc.GetID()).Scan(&sceneCount)
        if sceneCount != 1 {
            t.Errorf("expected 1 scene row, got %d", sceneCount)
        }

        // Verify round row exists
        var roundCount int
        _ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM rounds WHERE uuid = $1`, r.GetID()).Scan(&roundCount)
        if roundCount != 1 {
            t.Errorf("expected 1 round row, got %d", roundCount)
        }

        // Verify turn row exists
        var turnCount int
        _ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM turns WHERE uuid = $1`, tRn.GetID()).Scan(&turnCount)
        if turnCount != 1 {
            t.Errorf("expected 1 turn row, got %d", turnCount)
        }

        // Verify action row exists
        var actionCount int
        _ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM actions WHERE actor_uuid = $1`, actorUUIDParsed2).Scan(&actionCount)
        if actionCount != 1 {
            t.Errorf("expected 1 action row, got %d", actionCount)
        }
    })

    t.Run("ON CONFLICT DO NOTHING — second call with same scene/round UUIDs is idempotent", func(t *testing.T) {
        pgtest.TruncateAll(t, pool)
        masterUUID3 := pgtest.InsertTestUser(t, pool, "gm3", "gm3@test.com", "pass")
        campaignUUID3 := pgtest.InsertTestCampaign(t, pool, masterUUID3, "Camp3")
        matchUUID3 := pgtest.InsertTestMatch(t, pool, masterUUID3, campaignUUID3, "Match3")
        matchUUIDParsed3, _ := uuid.Parse(matchUUID3)
        actorUUIDParsed3, _ := uuid.Parse(masterUUID3)

        sc := sceneentity.NewScene(enum.Roleplay, "Inn")
        r := roundentity.NewRound(enum.Free)

        act1 := action.NewAction(actorUUIDParsed3, nil, uuid.Nil, nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil)
        act1v := *act1
        tRn1 := turnentity.NewTurn(act1v)
        tRn1.Close(time.Now())
        if err := repo.PersistTurnClose(ctx, sc, r, tRn1, act1, matchUUIDParsed3); err != nil {
            t.Fatalf("first PersistTurnClose error: %v", err)
        }

        // Second call with same scene/round (simulating retry) — only new turn+action should insert
        act2 := action.NewAction(actorUUIDParsed3, nil, uuid.Nil, nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil)
        act2v := *act2
        tRn2 := turnentity.NewTurn(act2v)
        tRn2.Close(time.Now())
        if err := repo.PersistTurnClose(ctx, sc, r, tRn2, act2, matchUUIDParsed3); err != nil {
            t.Fatalf("second PersistTurnClose error: %v", err)
        }

        var sceneCount int
        _ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM scenes WHERE uuid = $1`, sc.GetID()).Scan(&sceneCount)
        if sceneCount != 1 {
            t.Errorf("expected exactly 1 scene row after two calls, got %d", sceneCount)
        }
    })

    _ = matchUUID
    _ = matchUUIDParsed
    _ = actorUUIDParsed
}
```

### 6b — Implement PersistTurnClose

- [ ] Create `internal/gateway/pg/round/persist_turn_close.go`:

```go
package round

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
    roundentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
    sceneentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/scene"
    turnentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
    "github.com/google/uuid"
)

// PersistTurnClose atomically writes scene (idempotent), round (idempotent), turn, and action
// to the database. Idempotency on scene/round allows retries without duplicates.
func (r *Repository) PersistTurnClose(
    ctx context.Context,
    sc *sceneentity.Scene,
    rnd *roundentity.Round,
    t *turnentity.Turn,
    act *action.Action,
    matchUUID uuid.UUID,
) error {
    tx, err := r.pool.Begin(ctx)
    if err != nil {
        return fmt.Errorf("PersistTurnClose begin tx: %w", err)
    }
    defer func() {
        if p := recover(); p != nil {
            _ = tx.Rollback(ctx)
            panic(p)
        }
        _ = tx.Rollback(ctx)
    }()

    // Insert scene (idempotent — ON CONFLICT DO NOTHING means a retry won't fail)
    _, err = tx.Exec(ctx,
        `INSERT INTO scenes (uuid, match_uuid, category, brief_initial_description, created_at)
         VALUES ($1, $2, $3, $4, $5)
         ON CONFLICT (uuid) DO NOTHING`,
        sc.GetID(), matchUUID, string(sc.GetCategory()), sc.BriefInitialDescription, sc.GetCreatedAt(),
    )
    if err != nil {
        return fmt.Errorf("PersistTurnClose insert scene: %w", err)
    }

    // Insert round (idempotent)
    _, err = tx.Exec(ctx,
        `INSERT INTO rounds (uuid, scene_uuid, mode, created_at)
         VALUES ($1, $2, $3, $4)
         ON CONFLICT (uuid) DO NOTHING`,
        rnd.GetID(), sc.GetID(), string(rnd.GetMode()), rnd.GetCreatedAt(),
    )
    if err != nil {
        return fmt.Errorf("PersistTurnClose insert round: %w", err)
    }

    // Insert turn (not idempotent — each turn is unique)
    if t.GetFinishedAt() == nil {
        return fmt.Errorf("PersistTurnClose: turn must be closed before persisting")
    }
    _, err = tx.Exec(ctx,
        `INSERT INTO turns (uuid, round_uuid, created_at, finished_at)
         VALUES ($1, $2, $3, $4)`,
        t.GetID(), rnd.GetID(), t.GetFinishedAt(), t.GetFinishedAt(), // created_at approximated by finishedAt
    )
    if err != nil {
        return fmt.Errorf("PersistTurnClose insert turn: %w", err)
    }

    // Derive action type
    actionType := deriveActionType(act)

    // Marshal optional JSONB fields
    speedJSON, err := marshalNullable(act.Speed)
    if err != nil {
        return fmt.Errorf("PersistTurnClose marshal speed: %w", err)
    }
    skillsJSON, err := marshalNullable(act.Skills)
    if err != nil {
        return fmt.Errorf("PersistTurnClose marshal skills: %w", err)
    }
    moveJSON, err := marshalNullable(act.Move)
    if err != nil {
        return fmt.Errorf("PersistTurnClose marshal move: %w", err)
    }
    attackJSON, err := marshalNullable(act.Attack)
    if err != nil {
        return fmt.Errorf("PersistTurnClose marshal attack: %w", err)
    }
    defenseJSON, err := marshalNullable(act.Defense)
    if err != nil {
        return fmt.Errorf("PersistTurnClose marshal defense: %w", err)
    }
    dodgeJSON, err := marshalNullable(act.Dodge)
    if err != nil {
        return fmt.Errorf("PersistTurnClose marshal dodge: %w", err)
    }
    feintJSON, err := marshalNullable(act.Feint)
    if err != nil {
        return fmt.Errorf("PersistTurnClose marshal feint: %w", err)
    }
    triggerJSON, err := marshalNullable(act.Trigger)
    if err != nil {
        return fmt.Errorf("PersistTurnClose marshal trigger: %w", err)
    }

    // react_to_uuid: nil if zero
    var reactToUUID *uuid.UUID
    if act.ReactToID != uuid.Nil {
        v := act.ReactToID
        reactToUUID = &v
    }

    _, err = tx.Exec(ctx,
        `INSERT INTO actions
         (uuid, turn_uuid, actor_uuid, react_to_uuid, target_ids, type,
          speed, skills, move, attack, defense, dodge, feint, trigger, created_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
        act.GetID(), t.GetID(), act.GetActorID(), reactToUUID, act.TargetID, actionType,
        speedJSON, skillsJSON, moveJSON, attackJSON, defenseJSON, dodgeJSON, feintJSON, triggerJSON,
        t.GetFinishedAt(),
    )
    if err != nil {
        return fmt.Errorf("PersistTurnClose insert action: %w", err)
    }

    if err := tx.Commit(ctx); err != nil {
        return fmt.Errorf("PersistTurnClose commit: %w", err)
    }
    return nil
}

func deriveActionType(act *action.Action) string {
    if act.Attack != nil {
        return "attack"
    }
    if act.Move != nil {
        return "move"
    }
    if act.Defense != nil {
        return "defense"
    }
    if act.Dodge != nil {
        return "dodge"
    }
    if act.Feint != nil {
        return "feint"
    }
    if len(act.Skills) > 0 {
        return "skill"
    }
    return "unspecified"
}

// marshalNullable returns nil (SQL NULL) when v is a nil pointer or empty slice, else JSON bytes.
func marshalNullable(v any) ([]byte, error) {
    if v == nil {
        return nil, nil
    }
    return json.Marshal(v)
}
```

**Note on turn created_at**: The `turns.created_at` column is set to `t.GetFinishedAt()` as an approximation because `Turn` does not currently expose a `createdAt` field. If a `createdAt` field is added to `Turn` in a future phase, update this query to use it.

- [ ] Run: `go test -tags=integration ./internal/gateway/pg/round/...` — must pass.

- [ ] Commit:
  ```
  feat(gateway/round): implement PersistTurnClose with atomic scene/round/turn/action insert

  Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
  ```

---

## Task 7: Gateway — FindActiveSession

**Goal:** Query for the active (unfinished) scene+round pair for a match.

### 7a — Write integration test first (append to `round_integration_test.go`)

```go
func TestFindActiveSession(t *testing.T) {
    pool := pgtest.SetupTestDB(t)
    repo := roundrepo.NewRepository(pool)
    ctx := context.Background()

    t.Run("returns nil when no active session exists", func(t *testing.T) {
        pgtest.TruncateAll(t, pool)
        masterUUID := pgtest.InsertTestUser(t, pool, "gm1", "gm1@test.com", "pass")
        campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "Camp1")
        matchUUID := pgtest.InsertTestMatch(t, pool, masterUUID, campaignUUID, "Match1")
        matchUUIDParsed, _ := uuid.Parse(matchUUID)

        data, err := repo.FindActiveSession(ctx, matchUUIDParsed)
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if data != nil {
            t.Errorf("expected nil, got %+v", data)
        }
    })

    t.Run("returns active session when scene and round are open", func(t *testing.T) {
        pgtest.TruncateAll(t, pool)
        masterUUID := pgtest.InsertTestUser(t, pool, "gm2", "gm2@test.com", "pass")
        campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "Camp2")
        matchUUID := pgtest.InsertTestMatch(t, pool, masterUUID, campaignUUID, "Match2")
        matchUUIDParsed, _ := uuid.Parse(matchUUID)

        sceneUUID := pgtest.InsertTestScene(t, pool, matchUUID, "Battle")
        roundUUID := pgtest.InsertTestRound(t, pool, sceneUUID, "Free")

        data, err := repo.FindActiveSession(ctx, matchUUIDParsed)
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if data == nil {
            t.Fatal("expected non-nil ActiveSessionData")
        }
        if data.SceneID.String() != sceneUUID {
            t.Errorf("expected SceneID %s, got %s", sceneUUID, data.SceneID)
        }
        if data.RoundID.String() != roundUUID {
            t.Errorf("expected RoundID %s, got %s", roundUUID, data.RoundID)
        }
        if data.Category != "Battle" {
            t.Errorf("expected category Battle, got %q", data.Category)
        }
        if data.Mode != "Free" {
            t.Errorf("expected mode Free, got %q", data.Mode)
        }
    })

    t.Run("ignores finished scenes", func(t *testing.T) {
        pgtest.TruncateAll(t, pool)
        masterUUID := pgtest.InsertTestUser(t, pool, "gm3", "gm3@test.com", "pass")
        campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "Camp3")
        matchUUID := pgtest.InsertTestMatch(t, pool, masterUUID, campaignUUID, "Match3")
        matchUUIDParsed, _ := uuid.Parse(matchUUID)

        sceneUUID := pgtest.InsertTestScene(t, pool, matchUUID, "Roleplay")
        pgtest.InsertTestRound(t, pool, sceneUUID, "Free")

        // Close the scene
        pool.Exec(ctx, `UPDATE scenes SET finished_at = $1 WHERE uuid = $2`, time.Now(), sceneUUID) //nolint:errcheck

        data, err := repo.FindActiveSession(ctx, matchUUIDParsed)
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if data != nil {
            t.Errorf("expected nil for finished scene, got %+v", data)
        }
    })
}
```

### 7b — Implement FindActiveSession

- [ ] Create `internal/gateway/pg/round/find_active_session.go`:

```go
package round

import (
    "context"
    "errors"
    "fmt"

    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"
)

// FindActiveSession returns the current unfinished scene+round for the given match,
// or nil if none exists. The caller reconstructs domain entities from the returned data.
func (r *Repository) FindActiveSession(ctx context.Context, matchUUID uuid.UUID) (*ActiveSessionData, error) {
    row := r.pool.QueryRow(ctx,
        `SELECT s.uuid, s.category, s.brief_initial_description, s.created_at,
                ro.uuid, ro.mode, ro.created_at
         FROM scenes s
         JOIN rounds ro ON ro.scene_uuid = s.uuid
         WHERE s.match_uuid = $1
           AND s.finished_at IS NULL
           AND ro.finished_at IS NULL
         LIMIT 1`,
        matchUUID,
    )

    data := &ActiveSessionData{}
    err := row.Scan(
        &data.SceneID, &data.Category, &data.BriefInitDesc, &data.SceneCreatedAt,
        &data.RoundID, &data.Mode, &data.RoundCreatedAt,
    )
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, nil
        }
        return nil, fmt.Errorf("FindActiveSession: %w", err)
    }
    return data, nil
}
```

- [ ] Run: `go test -tags=integration ./internal/gateway/pg/round/...` — all must pass.

- [ ] Commit:
  ```
  feat(gateway/round): implement FindActiveSession

  Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
  ```

---

## Task 8: Gateway — CloseSceneAndRound + CloseRound

**Goal:** Implement two UPDATE operations for closing scenes and rounds.

### 8a — Write integration tests first (append to `round_integration_test.go`)

```go
func TestCloseSceneAndRound(t *testing.T) {
    pool := pgtest.SetupTestDB(t)
    repo := roundrepo.NewRepository(pool)
    ctx := context.Background()

    t.Run("happy path — sets finished_at on both scene and round", func(t *testing.T) {
        pgtest.TruncateAll(t, pool)
        masterUUID := pgtest.InsertTestUser(t, pool, "gm1", "gm1@test.com", "pass")
        campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "Camp1")
        matchUUID := pgtest.InsertTestMatch(t, pool, masterUUID, campaignUUID, "Match1")

        sceneUUID := pgtest.InsertTestScene(t, pool, matchUUID, "Battle")
        roundUUID := pgtest.InsertTestRound(t, pool, sceneUUID, "Free")

        sceneUUIDParsed, _ := uuid.Parse(sceneUUID)
        roundUUIDParsed, _ := uuid.Parse(roundUUID)
        at := time.Now().Truncate(time.Microsecond)

        err := repo.CloseSceneAndRound(ctx, sceneUUIDParsed, roundUUIDParsed, at)
        if err != nil {
            t.Fatalf("CloseSceneAndRound error: %v", err)
        }

        var sceneFinishedAt, roundFinishedAt time.Time
        pool.QueryRow(ctx, `SELECT finished_at FROM scenes WHERE uuid = $1`, sceneUUID).Scan(&sceneFinishedAt) //nolint:errcheck
        pool.QueryRow(ctx, `SELECT finished_at FROM rounds WHERE uuid = $1`, roundUUID).Scan(&roundFinishedAt) //nolint:errcheck

        if !sceneFinishedAt.Truncate(time.Microsecond).Equal(at) {
            t.Errorf("scene finished_at: got %v, want %v", sceneFinishedAt, at)
        }
        if !roundFinishedAt.Truncate(time.Microsecond).Equal(at) {
            t.Errorf("round finished_at: got %v, want %v", roundFinishedAt, at)
        }
    })
}

func TestCloseRound(t *testing.T) {
    pool := pgtest.SetupTestDB(t)
    repo := roundrepo.NewRepository(pool)
    ctx := context.Background()

    t.Run("happy path — sets finished_at on round only", func(t *testing.T) {
        pgtest.TruncateAll(t, pool)
        masterUUID := pgtest.InsertTestUser(t, pool, "gm1", "gm1@test.com", "pass")
        campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "Camp1")
        matchUUID := pgtest.InsertTestMatch(t, pool, masterUUID, campaignUUID, "Match1")

        sceneUUID := pgtest.InsertTestScene(t, pool, matchUUID, "Battle")
        roundUUID := pgtest.InsertTestRound(t, pool, sceneUUID, "Race")

        roundUUIDParsed, _ := uuid.Parse(roundUUID)
        at := time.Now().Truncate(time.Microsecond)

        err := repo.CloseRound(ctx, roundUUIDParsed, at)
        if err != nil {
            t.Fatalf("CloseRound error: %v", err)
        }

        var roundFinishedAt time.Time
        pool.QueryRow(ctx, `SELECT finished_at FROM rounds WHERE uuid = $1`, roundUUID).Scan(&roundFinishedAt) //nolint:errcheck

        if !roundFinishedAt.Truncate(time.Microsecond).Equal(at) {
            t.Errorf("round finished_at: got %v, want %v", roundFinishedAt, at)
        }

        // Scene must NOT be closed
        var sceneFinishedAt *time.Time
        pool.QueryRow(ctx, `SELECT finished_at FROM scenes WHERE uuid = $1`, sceneUUID).Scan(&sceneFinishedAt) //nolint:errcheck
        if sceneFinishedAt != nil {
            t.Error("expected scene finished_at to remain NULL")
        }
    })
}
```

### 8b — Implement

- [ ] Create `internal/gateway/pg/round/close_scene_and_round.go`:

```go
package round

import (
    "context"
    "fmt"
    "time"

    "github.com/google/uuid"
)

// CloseSceneAndRound sets finished_at on both scene and round atomically.
// Used when ChangeScene transitions to a new scene.
func (r *Repository) CloseSceneAndRound(ctx context.Context, sceneUUID, roundUUID uuid.UUID, at time.Time) error {
    tx, err := r.pool.Begin(ctx)
    if err != nil {
        return fmt.Errorf("CloseSceneAndRound begin tx: %w", err)
    }
    defer func() {
        if p := recover(); p != nil {
            _ = tx.Rollback(ctx)
            panic(p)
        }
        _ = tx.Rollback(ctx)
    }()

    if _, err := tx.Exec(ctx,
        `UPDATE scenes SET finished_at = $1 WHERE uuid = $2`,
        at, sceneUUID,
    ); err != nil {
        return fmt.Errorf("CloseSceneAndRound update scene: %w", err)
    }

    if _, err := tx.Exec(ctx,
        `UPDATE rounds SET finished_at = $1 WHERE uuid = $2`,
        at, roundUUID,
    ); err != nil {
        return fmt.Errorf("CloseSceneAndRound update round: %w", err)
    }

    if err := tx.Commit(ctx); err != nil {
        return fmt.Errorf("CloseSceneAndRound commit: %w", err)
    }
    return nil
}
```

- [ ] Create `internal/gateway/pg/round/close_round.go`:

```go
package round

import (
    "context"
    "fmt"
    "time"

    "github.com/google/uuid"
)

// CloseRound sets finished_at on a single round row.
// Used by CloseRoundUC when the round was already persisted via PersistTurnClose.
func (r *Repository) CloseRound(ctx context.Context, roundUUID uuid.UUID, at time.Time) error {
    _, err := r.pool.Exec(ctx,
        `UPDATE rounds SET finished_at = $1 WHERE uuid = $2`,
        at, roundUUID,
    )
    if err != nil {
        return fmt.Errorf("CloseRound: %w", err)
    }
    return nil
}
```

- [ ] Run: `go test -tags=integration ./internal/gateway/pg/round/...` — all must pass.

- [ ] Verify: `go vet -tags=integration ./internal/gateway/pg/...`

- [ ] Commit:
  ```
  feat(gateway/round): implement CloseSceneAndRound and CloseRound

  Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
  ```

---

## Task 9: Application — EnqueueMasterActionUC

**Goal:** New use case that validates master identity and delegates to `session.EnqueueMasterAction`.

### 9a — Write unit test first

- [ ] Create `internal/application/match/enqueue_master_action_test.go`:

```go
package match_test

import (
    "context"
    "errors"
    "testing"

    appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
    "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
    "github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
    "github.com/google/uuid"
)

func TestEnqueueMasterActionUC(t *testing.T) {
    masterUUID := uuid.New()

    t.Run("returns ErrNotMatchMaster when caller is not master", func(t *testing.T) {
        s := matchsession.NewMatchSession(uuid.New(), nil, nil)
        uc := appmatch.NewEnqueueMasterActionUC()
        caller := uuid.New() // different from masterUUID
        ma := action.NewMasterAction()

        err := uc.Execute(context.Background(), s, masterUUID, caller, ma)
        if !errors.Is(err, appmatch.ErrNotMatchMaster) {
            t.Errorf("expected ErrNotMatchMaster, got %v", err)
        }
    })

    t.Run("returns ErrNoActiveTurn when no open turn", func(t *testing.T) {
        s := matchsession.NewMatchSession(uuid.New(), nil, nil)
        uc := appmatch.NewEnqueueMasterActionUC()
        ma := action.NewMasterAction()

        err := uc.Execute(context.Background(), s, masterUUID, masterUUID, ma)
        if !errors.Is(err, matchsession.ErrNoActiveTurn) {
            t.Errorf("expected ErrNoActiveTurn, got %v", err)
        }
    })
}
```

### 9b — Implement

- [ ] Create `internal/application/match/enqueue_master_action.go`:

```go
package match

import (
    "context"

    "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
    "github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
    "github.com/google/uuid"
)

type IEnqueueMasterAction interface {
    Execute(ctx context.Context, session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID, ma *action.MasterAction) error
}

type EnqueueMasterActionUC struct{}

func NewEnqueueMasterActionUC() *EnqueueMasterActionUC {
    return &EnqueueMasterActionUC{}
}

func (uc *EnqueueMasterActionUC) Execute(
    ctx context.Context,
    session *matchsession.MatchSession,
    masterUUID, callerUUID uuid.UUID,
    ma *action.MasterAction,
) error {
    if callerUUID != masterUUID {
        return ErrNotMatchMaster
    }
    return session.EnqueueMasterAction(ma)
}

var _ IEnqueueMasterAction = (*EnqueueMasterActionUC)(nil)
```

- [ ] Run: `go test ./internal/application/match/...` — all must pass.

- [ ] Commit:
  ```
  feat(application/match): add EnqueueMasterActionUC

  Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
  ```

---

## Task 10: Application — ChangeSceneUC

**Goal:** New use case validating master identity and calling `session.ChangeScene`.

### 10a — Write unit test first

- [ ] Create `internal/application/match/change_scene_test.go`:

```go
package match_test

import (
    "context"
    "errors"
    "testing"

    appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
    "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
    "github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
    "github.com/google/uuid"
)

func TestChangeSceneUC(t *testing.T) {
    masterUUID := uuid.New()

    t.Run("returns ErrNotMatchMaster when caller is not master", func(t *testing.T) {
        s := matchsession.NewMatchSession(uuid.New(), nil, nil)
        uc := appmatch.NewChangeSceneUC()
        caller := uuid.New()

        _, _, err := uc.Execute(context.Background(), s, masterUUID, caller, enum.Battle, "Arena")
        if !errors.Is(err, appmatch.ErrNotMatchMaster) {
            t.Errorf("expected ErrNotMatchMaster, got %v", err)
        }
    })

    t.Run("changes scene when master calls with valid args", func(t *testing.T) {
        s := matchsession.NewMatchSession(uuid.New(), nil, nil)
        uc := appmatch.NewChangeSceneUC()

        oldScene, oldRound, err := uc.Execute(context.Background(), s, masterUUID, masterUUID, enum.Battle, "Arena fight")
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if oldScene == nil {
            t.Fatal("expected non-nil old scene")
        }
        if oldRound == nil {
            t.Fatal("expected non-nil old round")
        }
        if oldScene.GetFinishedAt() == nil {
            t.Error("expected old scene to be closed")
        }
        if s.GetActiveScene().GetCategory() != enum.Battle {
            t.Errorf("expected active scene category Battle, got %v", s.GetActiveScene().GetCategory())
        }
    })
}
```

### 10b — Implement

- [ ] Create `internal/application/match/change_scene.go`:

```go
package match

import (
    "context"

    "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
    roundentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
    sceneentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/scene"
    "github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
    "github.com/google/uuid"
)

type IChangeScene interface {
    Execute(ctx context.Context, session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID, category enum.SceneCategory, briefDesc string) (*sceneentity.Scene, *roundentity.Round, error)
}

type ChangeSceneUC struct{}

func NewChangeSceneUC() *ChangeSceneUC {
    return &ChangeSceneUC{}
}

func (uc *ChangeSceneUC) Execute(
    ctx context.Context,
    session *matchsession.MatchSession,
    masterUUID, callerUUID uuid.UUID,
    category enum.SceneCategory,
    briefDesc string,
) (*sceneentity.Scene, *roundentity.Round, error) {
    if callerUUID != masterUUID {
        return nil, nil, ErrNotMatchMaster
    }
    return session.ChangeScene(category, briefDesc)
}

var _ IChangeScene = (*ChangeSceneUC)(nil)
```

- [ ] Run: `go test ./internal/application/match/...` — all must pass.

- [ ] Commit:
  ```
  feat(application/match): add ChangeSceneUC

  Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
  ```

---

## Task 11: Application — InitMatchSessionUC recovery + CloseRoundUC update

**Goal:** InitMatchSessionUC recovers persisted scene+round from DB. CloseRoundUC persists round close when already persisted.

### 11a — Write tests first

- [ ] Create `internal/application/match/init_match_session_recovery_test.go`:

```go
package match_test

import (
    "context"
    "testing"
    "time"

    appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
    "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
    roundrepo "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/round"
    "github.com/google/uuid"
)

// mockRoundRepo implements IRoundRepository for unit tests.
type mockRoundRepo struct {
    findActiveFn func(ctx context.Context, matchUUID uuid.UUID) (*roundrepo.ActiveSessionData, error)
    closeRoundFn func(ctx context.Context, roundUUID uuid.UUID, at time.Time) error
    closeSceneAndRoundFn func(ctx context.Context, sceneUUID, roundUUID uuid.UUID, at time.Time) error
    persistTurnCloseFn func() error
}

func (m *mockRoundRepo) FindActiveSession(ctx context.Context, matchUUID uuid.UUID) (*roundrepo.ActiveSessionData, error) {
    if m.findActiveFn != nil {
        return m.findActiveFn(ctx, matchUUID)
    }
    return nil, nil
}

func (m *mockRoundRepo) CloseRound(ctx context.Context, roundUUID uuid.UUID, at time.Time) error {
    if m.closeRoundFn != nil {
        return m.closeRoundFn(ctx, roundUUID, at)
    }
    return nil
}

func (m *mockRoundRepo) CloseSceneAndRound(ctx context.Context, sceneUUID, roundUUID uuid.UUID, at time.Time) error {
    if m.closeSceneAndRoundFn != nil {
        return m.closeSceneAndRoundFn(ctx, sceneUUID, roundUUID, at)
    }
    return nil
}

func (m *mockRoundRepo) PersistTurnClose(ctx context.Context, sc any, r any, t any, act any, matchUUID uuid.UUID) error {
    if m.persistTurnCloseFn != nil {
        return m.persistTurnCloseFn()
    }
    return nil
}

func TestInitMatchSessionUC_Recovery(t *testing.T) {
    t.Run("returns NewMatchSessionWithState when active session found", func(t *testing.T) {
        sceneID := uuid.New()
        roundID := uuid.New()
        now := time.Now()

        roundRepo := &mockRoundRepo{
            findActiveFn: func(_ context.Context, _ uuid.UUID) (*roundrepo.ActiveSessionData, error) {
                return &roundrepo.ActiveSessionData{
                    SceneID:        sceneID,
                    Category:       string(enum.Battle),
                    BriefInitDesc:  "Forest",
                    SceneCreatedAt: now,
                    RoundID:        roundID,
                    Mode:           string(enum.Free),
                    RoundCreatedAt: now,
                }, nil
            },
        }

        matchRepo := &testutil_MockMatchRepo_ForInit{}
        sheetLoader := &testutil_MockSheetLoader{}
        uc := appmatch.NewInitMatchSessionUC(matchRepo, sheetLoader, roundRepo)

        session, err := uc.Init(context.Background(), uuid.New())
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if !session.IsRoundPersisted() {
            t.Error("expected IsRoundPersisted true when recovering state")
        }
        if !session.IsScenePersisted() {
            t.Error("expected IsScenePersisted true when recovering state")
        }
        if session.GetActiveScene().GetID() != sceneID {
            t.Errorf("expected scene ID %v, got %v", sceneID, session.GetActiveScene().GetID())
        }
    })

    t.Run("returns NewMatchSession when no active session found", func(t *testing.T) {
        roundRepo := &mockRoundRepo{} // returns nil, nil

        matchRepo := &testutil_MockMatchRepo_ForInit{}
        sheetLoader := &testutil_MockSheetLoader{}
        uc := appmatch.NewInitMatchSessionUC(matchRepo, sheetLoader, roundRepo)

        session, err := uc.Init(context.Background(), uuid.New())
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if session.IsRoundPersisted() {
            t.Error("expected IsRoundPersisted false for fresh session")
        }
    })
}

// Minimal mock implementations for the test helpers used above.
// These satisfy the IRepository and ICharSheetLoader interfaces.
type testutil_MockMatchRepo_ForInit struct{}

func (m *testutil_MockMatchRepo_ForInit) ListParticipantsByMatchUUID(ctx context.Context, matchUUID uuid.UUID) ([]*match.Participant, error) {
    return nil, nil
}
// Implement remaining IRepository methods as no-ops (only ListParticipants is called by Init).
// ... (all other methods return zero values)

type testutil_MockSheetLoader struct{}

func (m *testutil_MockSheetLoader) GetCharacterSheetByUUID(ctx context.Context, id string) (*csSheet.CharacterSheet, bool, error) {
    return nil, false, nil
}
```

**Note**: The `testutil_MockMatchRepo_ForInit` must fully implement `IRepository`. Use the existing `testutil.MockMatchRepo` from `internal/application/testutil` instead if it satisfies the interface, to avoid duplication. Check `internal/application/testutil/` for the existing mock.

- [ ] Check existing testutil mocks:

```bash
ls internal/application/testutil/
```

Use the existing mock if it satisfies `IRepository`. If `testutil.MockMatchRepo` already has all required methods, import it directly into the test.

### 11b — Implement InitMatchSessionUC changes

- [ ] Open `internal/application/match/init_match_session.go`. Add `roundRepo IRoundRepository` field. Update constructor and `Init` method:

```go
type InitMatchSessionUC struct {
    matchRepo   IRepository
    sheetLoader ICharSheetLoader
    roundRepo   IRoundRepository
}

func NewInitMatchSessionUC(matchRepo IRepository, sheetLoader ICharSheetLoader, roundRepo IRoundRepository) *InitMatchSessionUC {
    return &InitMatchSessionUC{matchRepo: matchRepo, sheetLoader: sheetLoader, roundRepo: roundRepo}
}

func (uc *InitMatchSessionUC) Init(ctx context.Context, matchUUID uuid.UUID) (*matchsession.MatchSession, error) {
    participants, err := uc.matchRepo.ListParticipantsByMatchUUID(ctx, matchUUID)
    if err != nil {
        return nil, err
    }

    charSheets := make(map[uuid.UUID]*csSheet.CharacterSheet, len(participants))
    for _, p := range participants {
        if p.Sheet.PlayerUUID == nil {
            continue
        }
        sheet, found, err := uc.sheetLoader.GetCharacterSheetByUUID(ctx, p.Sheet.UUID.String())
        if err != nil {
            return nil, err
        }
        if found {
            charSheets[*p.Sheet.PlayerUUID] = sheet
        }
    }

    data, err := uc.roundRepo.FindActiveSession(ctx, matchUUID)
    if err != nil {
        return nil, err
    }
    if data != nil {
        sc := scene.ReconstructScene(data.SceneID, enum.SceneCategory(data.Category), data.BriefInitDesc, data.SceneCreatedAt)
        r := round.ReconstructRound(data.RoundID, enum.RoundMode(data.Mode), data.RoundCreatedAt)
        return matchsession.NewMatchSessionWithState(matchUUID, charSheets, participants, sc, r), nil
    }

    return matchsession.NewMatchSession(matchUUID, charSheets, participants), nil
}
```

Add imports: `scene`, `round`, `roundrepo "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/round"` — wait, the application layer must not import the gateway. The `IRoundRepository.FindActiveSession` returns `*roundrepo.ActiveSessionData`. Since that type is defined in `internal/gateway/pg/round`, and the application layer imports it via the interface, this creates a dependency on the gateway package from the application. 

**Resolution**: Move `ActiveSessionData` to the application layer. Define it in `internal/application/match/i_repository.go` instead:

```go
// ActiveSessionData is the data returned by IRoundRepository.FindActiveSession.
// Defined here (application layer) so the gateway package does not leak into the domain.
type ActiveSessionData struct {
    SceneID        uuid.UUID
    Category       string
    BriefInitDesc  string
    SceneCreatedAt time.Time
    RoundID        uuid.UUID
    Mode           string
    RoundCreatedAt time.Time
}
```

Update `IRoundRepository.FindActiveSession` signature to return `*ActiveSessionData` (in the same package). Update `internal/gateway/pg/round/repository.go` to remove the `ActiveSessionData` type and make the repository implement the application-layer interface (the gateway's `Repository` must return `*appmatch.ActiveSessionData` — but that creates a circular import).

**Correct resolution**: Define `ActiveSessionData` in a standalone package `internal/domain/match/matchsession/session_data.go`:

```go
package matchsession

import (
    "time"
    "github.com/google/uuid"
)

// ActiveSessionData is the DB-hydration DTO returned by the round repository.
type ActiveSessionData struct {
    SceneID        uuid.UUID
    Category       string
    BriefInitDesc  string
    SceneCreatedAt time.Time
    RoundID        uuid.UUID
    Mode           string
    RoundCreatedAt time.Time
}
```

Then:
- `IRoundRepository.FindActiveSession` returns `*matchsession.ActiveSessionData`
- `internal/gateway/pg/round/repository.go` drops its `ActiveSessionData` and imports `matchsession`
- No circular imports: gateway imports matchsession (allowed by dependency rules: gateway → domain)

Update `internal/gateway/pg/round/find_active_session.go` to return `*matchsession.ActiveSessionData` and update `repository.go` to remove the type.

- [ ] After implementing the `ActiveSessionData` move, update all files consistently:
  1. `internal/domain/match/matchsession/session_data.go` — new file with `ActiveSessionData`
  2. `internal/gateway/pg/round/repository.go` — remove `ActiveSessionData`, import matchsession
  3. `internal/gateway/pg/round/find_active_session.go` — return `*matchsession.ActiveSessionData`
  4. `internal/application/match/i_repository.go` — `IRoundRepository.FindActiveSession` returns `*matchsession.ActiveSessionData`
  5. `internal/application/match/init_match_session.go` — use `*matchsession.ActiveSessionData`

### 11c — Implement CloseRoundUC changes

- [ ] Open `internal/application/match/close_round.go`. Add `roundRepo IRoundRepository`. Update constructor and `Execute`:

```go
type CloseRoundUC struct {
    roundRepo IRoundRepository
}

func NewCloseRoundUC(roundRepo IRoundRepository) *CloseRoundUC {
    return &CloseRoundUC{roundRepo: roundRepo}
}

func (uc *CloseRoundUC) Execute(
    ctx context.Context,
    session *matchsession.MatchSession,
    masterUUID, callerUUID uuid.UUID,
) (*round.Round, error) {
    if callerUUID != masterUUID {
        return nil, ErrNotMatchMaster
    }
    closedRound, err := session.CloseRound()
    if err != nil {
        return nil, err
    }
    if session.IsRoundPersisted() && closedRound.GetFinishedAt() != nil {
        if dbErr := uc.roundRepo.CloseRound(ctx, closedRound.GetID(), *closedRound.GetFinishedAt()); dbErr != nil {
            // Log but don't fail — round is already closed in memory
            _ = dbErr
        }
    }
    return closedRound, nil
}
```

**Note**: `session.IsRoundPersisted()` is checked on the session state AFTER `CloseRound()` — but `CloseRound()` resets `roundPersisted` to false. The check must happen BEFORE the session call. Fix:

```go
func (uc *CloseRoundUC) Execute(
    ctx context.Context,
    session *matchsession.MatchSession,
    masterUUID, callerUUID uuid.UUID,
) (*round.Round, error) {
    if callerUUID != masterUUID {
        return nil, ErrNotMatchMaster
    }
    wasPersisted := session.IsRoundPersisted()
    closedRound, err := session.CloseRound()
    if err != nil {
        return nil, err
    }
    if wasPersisted && closedRound.GetFinishedAt() != nil {
        if dbErr := uc.roundRepo.CloseRound(ctx, closedRound.GetID(), *closedRound.GetFinishedAt()); dbErr != nil {
            _ = dbErr // log in production
        }
    }
    return closedRound, nil
}
```

- [ ] Update existing `close_round_test.go` to pass a mock `roundRepo` to `NewCloseRoundUC`. Read the existing test and add the mock:

```go
// in close_round_test.go — add mockRoundRepoNoOp (or reuse mockRoundRepo from Task 11a)
type mockCloseRoundRepo struct{}
func (m *mockCloseRoundRepo) CloseRound(_ context.Context, _ uuid.UUID, _ time.Time) error { return nil }
func (m *mockCloseRoundRepo) CloseSceneAndRound(_ context.Context, _, _ uuid.UUID, _ time.Time) error { return nil }
func (m *mockCloseRoundRepo) FindActiveSession(_ context.Context, _ uuid.UUID) (*matchsession.ActiveSessionData, error) { return nil, nil }
func (m *mockCloseRoundRepo) PersistTurnClose(_ context.Context, _ any, _ any, _ any, _ any, _ uuid.UUID) error { return nil }
```

Update all `NewCloseRoundUC()` calls to `NewCloseRoundUC(&mockCloseRoundRepo{})`.

- [ ] Run: `go test ./internal/application/match/...` — all must pass.

- [ ] Update wire-up in `cmd/game/` where `NewCloseRoundUC` is called — find and update to pass the round repo. Run `go build ./...`.

- [ ] Commit:
  ```
  feat(application/match): InitMatchSessionUC recovers persisted scene/round; CloseRoundUC persists close; add ActiveSessionData to matchsession

  Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
  ```

---

## Task 12: WS Layer — ChangeScene messages + handler

**Goal:** Add `MsgTypeChangeScene` / `MsgTypeSceneChanged` message types, `IChangeScene` interface, `changeSceneUC` field on Room, and the handler case.

### 12a — Update message.go

- [ ] Open `internal/app/game/message.go`. Add to the constants block:

```go
// Client → Server (scene management)
MsgTypeChangeScene MessageType = "change_scene"

// Server → Client (scene events)
MsgTypeSceneChanged MessageType = "scene_changed"
```

Add payload structs:

```go
type ChangeScenePayload struct {
    Category                string `json:"category"`
    BriefInitialDescription string `json:"brief_initial_description"`
}

type SceneChangedPayload struct {
    SceneID                 uuid.UUID `json:"scene_id"`
    Category                string    `json:"category"`
    BriefInitialDescription string    `json:"brief_initial_description"`
}
```

### 12b — Update Room

- [ ] Open `internal/app/game/room.go`. Add interface and field:

```go
type IChangeScene interface {
    Execute(ctx context.Context, session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID, category enum.SceneCategory, briefDesc string) (*scene.Scene, *round.Round, error)
}
```

Add to `Room` struct:
```go
changeSceneUC    IChangeScene
```

Add to `NewRoom` parameter list (add after `attachReactionUC`):
```go
changeSceneUC IChangeScene,
```

Set in body:
```go
changeSceneUC: changeSceneUC,
```

Add imports: `enum`, `scene`, `round` from domain packages.

Add handler case in `handleClientMessage`:

```go
case MsgTypeChangeScene:
    if !r.IsMaster(client.userUUID) {
        client.SendMessage(NewErrorMessage("forbidden", ErrNotMaster.Error()))
        return
    }
    var payload ChangeScenePayload
    if err := json.Unmarshal(incoming.Payload, &payload); err != nil {
        client.SendMessage(NewErrorMessage("invalid_payload", "invalid change_scene payload"))
        return
    }
    r.mu.RLock()
    session := r.session
    r.mu.RUnlock()

    newScene, _, err := r.changeSceneUC.Execute(
        context.Background(), session,
        r.masterUUID, client.userUUID,
        enum.SceneCategory(payload.Category), payload.BriefInitialDescription,
    )
    if err != nil {
        client.SendMessage(NewErrorMessage("game_error", err.Error()))
        return
    }
    out := NewServerMessage(MsgTypeSceneChanged, SceneChangedPayload{
        SceneID:                 newScene.GetID(),
        Category:                string(newScene.GetCategory()),
        BriefInitialDescription: newScene.BriefInitialDescription,
    })
    data, _ := json.Marshal(out)
    go func() { r.broadcast <- data }()
```

**Note**: `newScene` here is the NEW active scene (returned by `ChangeScene`). But `ChangeScene` returns the OLD scene and OLD round. The new active scene is `session.GetActiveScene()`. Fix the handler to use `session.GetActiveScene()` after the call:

```go
newScene, oldRound, err := r.changeSceneUC.Execute(...)
if err != nil { ... }

// Persist old scene/round close if they were already in DB
r.mu.RLock()
sceneWasPersisted := session.IsScenePersisted()
r.mu.RUnlock()

if sceneWasPersisted && oldRound != nil {
    // oldScene is what was returned — its ID matches what is in DB
    // oldRound has its finishedAt set by ChangeScene
    if oldScene != nil && oldRound.GetFinishedAt() != nil {
        if dbErr := r.roundRepo.CloseSceneAndRound(
            context.Background(),
            oldScene.GetID(), oldRound.GetID(), *oldRound.GetFinishedAt(),
        ); dbErr != nil {
            log.Printf("CloseSceneAndRound error: %v", dbErr)
        }
    }
}

r.mu.RLock()
activeScene := session.GetActiveScene()
r.mu.RUnlock()

out := NewServerMessage(MsgTypeSceneChanged, SceneChangedPayload{
    SceneID:                 activeScene.GetID(),
    Category:                string(activeScene.GetCategory()),
    BriefInitialDescription: activeScene.BriefInitialDescription,
})
data, _ := json.Marshal(out)
go func() { r.broadcast <- data }()
```

Adjust the handler signature and variables: the `IChangeScene.Execute` returns `(oldScene *scene.Scene, oldRound *round.Round, err error)`. The variable binding is:

```go
oldScene, oldRound, err := r.changeSceneUC.Execute(...)
```

Add `roundRepo IRoundRepository` field to `Room`. Add to `NewRoom` params and body. Add import `appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"` for `IRoundRepository`.

### 12c — Write mock tests

- [ ] Open `internal/app/game/game_test.go`. Add mock and update `newTestRoom`:

```go
type mockChangeSceneUC struct{}

func (m *mockChangeSceneUC) Execute(_ context.Context, s *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID, category enum.SceneCategory, briefDesc string) (*scene.Scene, *round.Round, error) {
    return scene.NewScene(category, briefDesc), round.NewRound(enum.Free), nil
}

type mockRoundRepoGame struct{}

func (m *mockRoundRepoGame) PersistTurnClose(_ context.Context, _ *sceneentity.Scene, _ *roundentity.Round, _ *turnentity.Turn, _ *action.Action, _ uuid.UUID) error {
    return nil
}
func (m *mockRoundRepoGame) FindActiveSession(_ context.Context, _ uuid.UUID) (*matchsession.ActiveSessionData, error) {
    return nil, nil
}
func (m *mockRoundRepoGame) CloseSceneAndRound(_ context.Context, _, _ uuid.UUID, _ time.Time) error {
    return nil
}
func (m *mockRoundRepoGame) CloseRound(_ context.Context, _ uuid.UUID, _ time.Time) error {
    return nil
}
```

Update `newTestRoom` to add `&mockChangeSceneUC{}` and `&mockRoundRepoGame{}` parameters.

Add a test:

```go
func TestChangeSceneMessage(t *testing.T) {
    matchUUID := uuid.New()
    masterUUID := uuid.New()
    room := newTestRoom(matchUUID, masterUUID)
    go room.Run()
    defer room.Stop()

    // Verify MsgTypeChangeScene constant exists and message type serialization works
    payload := game.ChangeScenePayload{Category: "Battle", BriefInitialDescription: "Arena fight"}
    msg := game.NewServerMessage(game.MsgTypeChangeScene, payload)
    if msg.Type != game.MsgTypeChangeScene {
        t.Errorf("expected type change_scene, got %s", msg.Type)
    }
}
```

- [ ] Run: `go test ./internal/app/game/...` — all must pass.

- [ ] Update `handler.go` and `handler_test.go` to include `changeSceneUC` and `roundRepo` parameters in `NewHandler`, `GetOrCreateRoom`, and their mocks. Follow the same pattern as existing parameters.

- [ ] Run: `go test ./internal/app/game/...` again after handler updates.

- [ ] Commit:
  ```
  feat(game): add ChangeScene message type, handler, and Room/Handler wiring

  Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
  ```

---

## Task 13: WS Layer — EnqueueMasterAction messages + buildMasterAction + handler

**Goal:** Add `MsgTypeEnqueueMasterAction` / `MsgTypeMasterActionEnqueued`, `buildMasterAction` in `action_mapper.go`, `enqueueMasterActionUC` on Room, and the handler case.

### 13a — Update message.go

- [ ] Append to `internal/app/game/message.go`:

```go
// Client → Server (master NPC actions)
MsgTypeEnqueueMasterAction MessageType = "enqueue_master_action"

// Server → Client
MsgTypeMasterActionEnqueued MessageType = "master_action_enqueued"
```

Add payload structs:

```go
type MasterActionPayload struct {
    TargetIDs   []uuid.UUID          `json:"target_ids"`
    Skills      []ActionSkillPayload  `json:"skills,omitempty"`
    Move        *MovePayload          `json:"move,omitempty"`
    Attack      *AttackPayload        `json:"attack,omitempty"`
    ActionSpeed *RollCheckPayload     `json:"action_speed,omitempty"`
}

type MasterActionEnqueuedPayload struct {
    TargetIDs   []uuid.UUID          `json:"target_ids"`
    Skills      []ActionSkillPayload  `json:"skills,omitempty"`
    Move        *MovePayload          `json:"move,omitempty"`
    Attack      *AttackPayload        `json:"attack,omitempty"`
    ActionSpeed *RollCheckPayload     `json:"action_speed,omitempty"`
}
```

### 13b — Update action_mapper.go

- [ ] Open `internal/app/game/action_mapper.go`. Append `buildMasterAction`:

```go
// buildMasterAction maps a MasterActionPayload to a MasterAction domain entity.
// masterUUID is the authenticated master's UUID — never trusted from the payload.
func buildMasterAction(masterUUID uuid.UUID, p MasterActionPayload) *action.MasterAction {
    ma := action.NewMasterAction()
    ma.TargetID = p.TargetIDs

    if p.Move != nil {
        // TODO: map Move fully once frontend contract is finalized
        _ = p.Move
    }
    if p.Attack != nil {
        // TODO: map Attack once frontend contract is finalized
        _ = p.Attack
    }
    if p.ActionSpeed != nil {
        ma.ActionSpeed = &action.RollCheck{SkillName: p.ActionSpeed.SkillName}
    }
    for _, s := range p.Skills {
        ma.Skills = append(ma.Skills, action.Skill{SkillName: s.SkillName})
    }
    return ma
}
```

Add import for `action` package if not already present.

### 13c — Update Room

- [ ] Add to `Room` struct:
  ```go
  enqueueMasterActionUC IEnqueueMasterAction
  ```
  
  Add interface in `room.go`:
  ```go
  type IEnqueueMasterAction interface {
      Execute(ctx context.Context, session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID, ma *action.MasterAction) error
  }
  ```

  Add to `NewRoom` params and body. Add handler case:

  ```go
  case MsgTypeEnqueueMasterAction:
      if !r.IsMaster(client.userUUID) {
          client.SendMessage(NewErrorMessage("forbidden", ErrNotMaster.Error()))
          return
      }
      var payload MasterActionPayload
      if err := json.Unmarshal(incoming.Payload, &payload); err != nil {
          client.SendMessage(NewErrorMessage("invalid_payload", "invalid enqueue_master_action payload"))
          return
      }
      r.mu.RLock()
      session := r.session
      r.mu.RUnlock()
      ma := buildMasterAction(client.userUUID, payload)
      if err := r.enqueueMasterActionUC.Execute(context.Background(), session, r.masterUUID, client.userUUID, ma); err != nil {
          client.SendMessage(NewErrorMessage("game_error", err.Error()))
          return
      }
      out := NewServerMessage(MsgTypeMasterActionEnqueued, MasterActionEnqueuedPayload{
          TargetIDs:   payload.TargetIDs,
          Skills:      payload.Skills,
          Move:        payload.Move,
          Attack:      payload.Attack,
          ActionSpeed: payload.ActionSpeed,
      })
      data, _ := json.Marshal(out)
      go func() { r.broadcast <- data }()
  ```

- [ ] Add mock and test in `game_test.go`:

```go
type mockEnqueueMasterActionUC struct{}

func (m *mockEnqueueMasterActionUC) Execute(_ context.Context, _ *matchsession.MatchSession, _, _ uuid.UUID, _ *action.MasterAction) error {
    return nil
}
```

Update `newTestRoom` to include `&mockEnqueueMasterActionUC{}`.

Add test:

```go
func TestEnqueueMasterActionMessage(t *testing.T) {
    payload := game.MasterActionPayload{TargetIDs: []uuid.UUID{uuid.New()}}
    msg := game.NewServerMessage(game.MsgTypeEnqueueMasterAction, payload)
    if msg.Type != game.MsgTypeEnqueueMasterAction {
        t.Errorf("expected type enqueue_master_action, got %s", msg.Type)
    }
}
```

- [ ] Update `handler.go` and `handler_test.go` with `enqueueMasterActionUC` parameter. Follow existing pattern.

- [ ] Run: `go test ./internal/app/game/...` — all must pass.

- [ ] Commit:
  ```
  feat(game): add EnqueueMasterAction message type, buildMasterAction, and Room/Handler wiring

  Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
  ```

---

## Task 14: Room — persistence integration (PersistTurnClose in handleOpenNextAction/handlePullAction)

**Goal:** After each turn close in `handleOpenNextAction` and `handlePullAction`, call `roundRepo.PersistTurnClose`. `roundRepo` is already on Room from Task 12.

### 14a — Update handleClientMessage cases

- [ ] In `internal/app/game/room.go`, find `case MsgTypeOpenNextAction`. After the `openNextActionUC.Execute` call, add:

```go
result, err := r.openNextActionUC.Execute(context.Background(), session, r.masterUUID, client.userUUID)
if err != nil {
    client.SendMessage(NewErrorMessage("game_error", err.Error()))
    return
}

// Persist the closed turn (if any) asynchronously — best-effort
if result.ClosedTurn != nil {
    closedTurn := result.ClosedTurn
    act := closedTurn.GetAction()
    r.mu.RLock()
    activeScene := session.GetActiveScene()
    activeRound := session.GetActiveRound()
    matchUUID := session.GetMatchUUID()
    r.mu.RUnlock()
    // Note: PersistTurnClose uses the round that was active during the turn,
    // which is the current activeRound (the turn belongs to it before close).
    // However, CloseRound swaps the round — so we capture *before* any round close.
    // OpenNextAction does NOT close the round; it only closes the turn.
    if err2 := r.roundRepo.PersistTurnClose(context.Background(), activeScene, activeRound, closedTurn, &act, matchUUID); err2 != nil {
        log.Printf("PersistTurnClose error: %v", err2)
    } else {
        r.mu.Lock()
        session.MarkRoundPersisted()
        r.mu.Unlock()
    }
}

act := result.OpenedTurn.GetAction()
out := NewServerMessage(MsgTypeTurnOpened, TurnOpenedPayload{
    TurnID:  result.OpenedTurn.GetID(),
    ActorID: act.GetActorID(),
})
data, _ := json.Marshal(out)
go func() { r.broadcast <- data }()
```

- [ ] Apply the same pattern to `case MsgTypePullAction`.

**Important**: `OpenNextActionResult.ClosedTurn` may be nil (first turn has no predecessor). Always guard with `if result.ClosedTurn != nil`.

### 14b — Update tests

- [ ] In `game_test.go` and `handler_test.go`, the `mockRoundRepoGame` is already in place. Verify `newTestRoom` passes it to `NewRoom`. All existing tests should pass unchanged.

- [ ] Run: `go test ./...` — all must pass.

- [ ] Commit:
  ```
  feat(game): integrate PersistTurnClose after turn close in OpenNextAction and PullAction handlers

  Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
  ```

---

## Task 15: Documentation update

**Goal:** Update `domain-map.instructions.md` and `AGENTS.md` to reflect Phase 3 completion.

### 15a — domain-map.instructions.md

- [ ] Open `.github/instructions/domain-map.instructions.md`. Update the Current State section:

Change:
```
- ✅ `domain/match/matchsession/` — In-memory match state: MatchSession + 6 session methods (Phase 2 complete)
```
To:
```
- ✅ `domain/match/matchsession/` — In-memory match state: MatchSession + 9 session methods, persistence flags, NewMatchSessionWithState (Phase 3 complete)
```

Add note about IDs:
```
- ✅ `domain/match/entity/round/` — Round entity with UUID id and createdAt; ReconstructRound for DB hydration
- ✅ `domain/match/entity/scene/` — Scene entity with UUID id, Close(), GetID(); ReconstructScene for DB hydration
- ✅ `gateway/pg/round/` — PersistTurnClose (atomic), FindActiveSession, CloseSceneAndRound, CloseRound
```

### 15b — AGENTS.md

- [ ] Open `AGENTS.md`. Find the Known Issues section. Replace:

```
## Known Issues

(Phase 2 complete — no outstanding issues)

**Deferred to Phase 3:**
- Turn/Round DB persistence (no schema yet — turns close automatically on NextAction/PullAction; round close is system-triggered)
- Reaction visibility: players see reactions only when master reveals (currently master-only)
- Scene management (`activeScene`, `ChangeScene`)
- Initiative handling in `ChangeMode`
- `EnqueueMasterAction` (NPC queue)
```

With:

```
## Known Issues

(Phase 3 complete — no outstanding issues)

**Deferred to Phase 4:**
- Reaction visibility: players see reactions only when master reveals (currently master-only)
- Initiative handling in `ChangeMode`
- `Turn.createdAt` field (turns currently use `finishedAt` as approximation for `created_at` in DB)
- Full Move/Attack mapping in `buildMasterAction` (pending frontend contract finalization)
```

- [ ] Run: `go build ./...` — must compile.

- [ ] Commit:
  ```
  docs: update domain-map and AGENTS.md to reflect Phase 3 completion

  Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
  ```

---

## Verification Checklist

After all tasks complete, run the following commands in order:

```bash
# 1. Build — no errors
go build ./...

# 2. Unit tests
go test ./internal/domain/...
go test ./internal/application/match/...
go test ./internal/app/game/...

# 3. Integration vet (no DB needed)
go vet -tags=integration ./internal/gateway/pg/...

# 4. Full test suite (requires PostgreSQL)
go test ./...
go test -tags=integration ./internal/gateway/pg/round/...
```

Expected: all tests pass, `go build ./...` exits 0.

---

## File Map

| Task | Files Created / Modified |
|------|--------------------------|
| 1 | `migrations/20260513000001_add_scenes_table.sql` (new) |
| 1 | `migrations/20260513000002_add_rounds_table.sql` (new) |
| 1 | `migrations/20260513000003_add_turns_table.sql` (new) |
| 1 | `migrations/20260513000004_add_actions_table.sql` (new) |
| 2 | `internal/domain/match/entity/round/round.go` (modified) |
| 2 | `internal/domain/match/entity/round/round_test.go` (modified) |
| 2 | `internal/domain/match/entity/scene/scene.go` (modified) |
| 2 | `internal/domain/match/entity/scene/scene_test.go` (modified) |
| 2 | `internal/domain/match/entity/turn/turn.go` (modified) |
| 2 | `internal/domain/match/entity/turn/turn_test.go` (modified) |
| 2 | `internal/domain/match/entity/action/master_action.go` (modified) |
| 2 | `internal/domain/match/entity/action/master_action_test.go` (new) |
| 3 | `internal/domain/match/matchsession/match_session.go` (modified) |
| 3 | `internal/domain/match/matchsession/error.go` (modified) |
| 3 | `internal/domain/match/matchsession/match_session_test.go` (modified) |
| 3 | `internal/domain/match/matchsession/session_data.go` (new — ActiveSessionData DTO) |
| 4 | `internal/gateway/pg/pgtest/setup.go` (modified) |
| 5 | `internal/application/match/i_repository.go` (modified) |
| 5 | `internal/gateway/pg/round/repository.go` (new) |
| 6 | `internal/gateway/pg/round/persist_turn_close.go` (new) |
| 6 | `internal/gateway/pg/round/round_integration_test.go` (new) |
| 7 | `internal/gateway/pg/round/find_active_session.go` (new) |
| 7 | `internal/gateway/pg/round/round_integration_test.go` (modified) |
| 8 | `internal/gateway/pg/round/close_scene_and_round.go` (new) |
| 8 | `internal/gateway/pg/round/close_round.go` (new) |
| 8 | `internal/gateway/pg/round/round_integration_test.go` (modified) |
| 9 | `internal/application/match/enqueue_master_action.go` (new) |
| 9 | `internal/application/match/enqueue_master_action_test.go` (new) |
| 10 | `internal/application/match/change_scene.go` (new) |
| 10 | `internal/application/match/change_scene_test.go` (new) |
| 11 | `internal/application/match/init_match_session.go` (modified) |
| 11 | `internal/application/match/close_round.go` (modified) |
| 11 | `internal/application/match/close_round_test.go` (modified) |
| 12 | `internal/app/game/message.go` (modified) |
| 12 | `internal/app/game/room.go` (modified) |
| 12 | `internal/app/game/hub.go` (modified) |
| 12 | `internal/app/game/handler.go` (modified) |
| 12 | `internal/app/game/game_test.go` (modified) |
| 12 | `internal/app/game/handler_test.go` (modified) |
| 13 | `internal/app/game/message.go` (modified) |
| 13 | `internal/app/game/action_mapper.go` (modified) |
| 13 | `internal/app/game/room.go` (modified) |
| 13 | `internal/app/game/game_test.go` (modified) |
| 14 | `internal/app/game/room.go` (modified) |
| 15 | `.github/instructions/domain-map.instructions.md` (modified) |
| 15 | `AGENTS.md` (modified) |
