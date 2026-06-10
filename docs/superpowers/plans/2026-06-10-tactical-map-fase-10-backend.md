# Tactical Map Fase 10 — Backend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the placeholder `Wall` entity with the full `WallSegment` model, wire it through mapper → validator → use case → HTTP handler, so `PUT /maps/:id` accepts and persists typed wall segments.

**Architecture:** `WallSegment` is a new domain entity in `internal/domain/map/entity/`. The existing `walls JSONB` column already exists — no SQL migration needed, only the Go shape changes. Validation lives in `map_validator.go`; the update use case grows a `Walls` input field; the HTTP handler and response type are updated accordingly.

**Tech Stack:** Go 1.23+, jackc/pgx v5 (JSONB), Huma v2 (HTTP), `github.com/google/uuid`, standard `encoding/json`, Go table-driven tests (`testing`), integration tests with `//go:build integration` tag.

---

## File Map

| File | Action | Responsibility |
|------|--------|---------------|
| `internal/domain/map/entity/wall_segment.go` | **Create** | `WallSegment` struct + all enum types |
| `internal/domain/map/entity/placeholders.go` | **Modify** | Remove old `Wall` struct |
| `internal/domain/map/entity/map.go` | **Modify** | `Walls []WallSegment` (was `[]Wall`) |
| `internal/domain/map/service/map_validator.go` | **Modify** | Add `ValidateWallSegments` + error vars |
| `internal/domain/map/service/map_validator_test.go` | **Modify** | Tests for `ValidateWallSegments` |
| `internal/gateway/pg/map/mapper.go` | **Modify** | Unmarshal into `[]entity.WallSegment` |
| `internal/gateway/pg/map/map_repository_test.go` | **Modify** | Add walls round-trip integration test |
| `internal/application/map/update_map.go` | **Modify** | `Walls *[]entity.WallSegment` in `UpdateMapInput`; validate + apply |
| `internal/app/api/map/update_map.go` | **Modify** | `Walls` field in `UpdateMapRequestBody` |
| `internal/app/api/map/map_response.go` | **Modify** | `Walls []entity.WallSegment` (was `any`) |
| `docs/dev/api/maps.md` | **Modify** | Show `walls` shape in `PUT /maps/:id` request + `MapResponse` |
| `docs/documentation-map.yaml` | **Modify** | Add entry for `internal/domain/map/` |

---

### Task 1: Write failing tests for `ValidateWallSegments`

**Files:**
- Modify: `internal/domain/map/service/map_validator_test.go`

The tests reference `service.ValidateWallSegments` and `entity.WallSegment`, which don't exist yet — they will fail to compile. That's correct for TDD.

- [ ] **Step 1: Add the failing tests**

Append to `internal/domain/map/service/map_validator_test.go` (keep the existing tests, add below):

```go
func validWallSegment() entity.WallSegment {
	return entity.WallSegment{
		ID:        "00000000-0000-0000-0000-000000000001",
		P1:        [2]float64{0, 0},
		P2:        [2]float64{1, 0},
		WallType:  entity.WallTypeWall,
		Material:  entity.WallMaterialStone,
		Move:      true,
		Sense:     entity.SenseFull,
		Direction: entity.WallDirectionBoth,
		HP:        100,
		MaxHP:     100,
		Resistance: 5,
	}
}

func TestValidateWallSegments_Empty(t *testing.T) {
	err := service.ValidateWallSegments([]entity.WallSegment{})
	if err != nil {
		t.Errorf("expected nil for empty walls, got %v", err)
	}
}

func TestValidateWallSegments_Valid(t *testing.T) {
	ws := validWallSegment()
	err := service.ValidateWallSegments([]entity.WallSegment{ws})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestValidateWallSegments_SameEndpoints(t *testing.T) {
	ws := validWallSegment()
	ws.P2 = ws.P1 // same point
	err := service.ValidateWallSegments([]entity.WallSegment{ws})
	if !errors.Is(err, service.ErrWallSameEndpoints) {
		t.Errorf("expected ErrWallSameEndpoints, got %v", err)
	}
}

func TestValidateWallSegments_InvalidWallType(t *testing.T) {
	ws := validWallSegment()
	ws.WallType = "invalid"
	err := service.ValidateWallSegments([]entity.WallSegment{ws})
	if !errors.Is(err, service.ErrWallInvalidType) {
		t.Errorf("expected ErrWallInvalidType, got %v", err)
	}
}

func TestValidateWallSegments_NegativeHP(t *testing.T) {
	ws := validWallSegment()
	ws.HP = -1
	err := service.ValidateWallSegments([]entity.WallSegment{ws})
	if !errors.Is(err, service.ErrWallNegativeHP) {
		t.Errorf("expected ErrWallNegativeHP, got %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify compilation failure**

```bash
cd /home/azzurah/Documentos/HxH_RPG_Environment_Project/System_X_System_Project/System_X_System
go test ./internal/domain/map/service/...
```

Expected: compilation error — `entity.WallSegment undefined`, `service.ValidateWallSegments undefined`, etc.

---

### Task 2: Create the `WallSegment` entity

**Files:**
- Create: `internal/domain/map/entity/wall_segment.go`

- [ ] **Step 1: Create the file**

```go
// internal/domain/map/entity/wall_segment.go
package entity

type WallType string

const (
	WallTypeWall       WallType = "wall"
	WallTypeDoor       WallType = "door"
	WallTypeWindow     WallType = "window"
	WallTypeSecretDoor WallType = "secret_door"
	WallTypeTerrain    WallType = "terrain"
)

type WallMaterial string

const (
	WallMaterialStone   WallMaterial = "stone"
	WallMaterialWood    WallMaterial = "wood"
	WallMaterialIron    WallMaterial = "iron"
	WallMaterialMagical WallMaterial = "magical"
)

type DoorSubtype string

const (
	DoorSubtypeBasic      DoorSubtype = "basic"
	DoorSubtypeDouble     DoorSubtype = "double"
	DoorSubtypePortcullis DoorSubtype = "portcullis"
	DoorSubtypeDrawbridge DoorSubtype = "drawbridge"
)

type WindowSubtype string

const (
	WindowSubtypeBasic     WindowSubtype = "basic"
	WindowSubtypeBarred    WindowSubtype = "barred"
	WindowSubtypeShuttered WindowSubtype = "shuttered"
)

type SenseKind string

const (
	SenseFull  SenseKind = "full"
	SenseSight SenseKind = "sight"
	SenseNone  SenseKind = "none"
)

type WallDirection string

const (
	WallDirectionBoth  WallDirection = "both"
	WallDirectionLeft  WallDirection = "left"
	WallDirectionRight WallDirection = "right"
)

type WallSegment struct {
	ID            string         `json:"id"`
	P1            [2]float64     `json:"p1"`
	P2            [2]float64     `json:"p2"`
	WallType      WallType       `json:"wall_type"`
	Material      WallMaterial   `json:"material"`
	DoorSubtype   *DoorSubtype   `json:"door_subtype,omitempty"`
	WindowSubtype *WindowSubtype `json:"window_subtype,omitempty"`
	Move          bool           `json:"move"`
	Sense         SenseKind      `json:"sense"`
	Direction     WallDirection  `json:"direction"`
	Open          bool           `json:"open"`
	Locked        bool           `json:"locked"`
	HP            int            `json:"hp"`
	MaxHP         int            `json:"max_hp"`
	Resistance    int            `json:"resistance"`
	Destroyed     bool           `json:"destroyed"`
}
```

> **Note on `ID` type:** using `string` (not `uuid.UUID`) to match the same pattern as `Piece.ID` and `Piece.CharacterID` in this codebase (`piece.go` uses `string` for IDs). The frontend sends a UUID string generated by `crypto.randomUUID()`.

- [ ] **Step 2: Verify the package compiles**

```bash
go build ./internal/domain/map/entity/...
```

Expected: success.

---

### Task 3: Add `ValidateWallSegments` to the service

**Files:**
- Modify: `internal/domain/map/service/map_validator.go`

- [ ] **Step 1: Add error vars and function**

In `internal/domain/map/service/map_validator.go`, add to the existing `var (...)` block and append the new function:

```go
// add to existing var block:
ErrWallSameEndpoints = errors.New("wall p1 and p2 must be different points")
ErrWallInvalidType   = errors.New("invalid wall_type")
ErrWallNegativeHP    = errors.New("wall hp must be >= 0")
```

Append after `ValidateMap`:

```go
func ValidateWallSegments(walls []entity.WallSegment) error {
	for _, w := range walls {
		if w.P1 == w.P2 {
			return ErrWallSameEndpoints
		}
		switch w.WallType {
		case entity.WallTypeWall, entity.WallTypeDoor, entity.WallTypeWindow,
			entity.WallTypeSecretDoor, entity.WallTypeTerrain:
		default:
			return ErrWallInvalidType
		}
		if w.HP < 0 {
			return ErrWallNegativeHP
		}
	}
	return nil
}
```

- [ ] **Step 2: Run the service tests**

```bash
go test ./internal/domain/map/service/...
```

Expected: all tests PASS (including the 5 new wall tests + the 6 existing ones).

- [ ] **Step 3: Commit**

```bash
git add internal/domain/map/entity/wall_segment.go \
        internal/domain/map/service/map_validator.go \
        internal/domain/map/service/map_validator_test.go
git commit -m "feat(map): add WallSegment entity and ValidateWallSegments"
```

---

### Task 4: Replace `Wall` with `WallSegment` in entity + mapper + response

**Files:**
- Modify: `internal/domain/map/entity/placeholders.go`
- Modify: `internal/domain/map/entity/map.go`
- Modify: `internal/gateway/pg/map/mapper.go`
- Modify: `internal/app/api/map/map_response.go`

- [ ] **Step 1: Remove `Wall` from `placeholders.go`**

Delete the `Wall` struct (lines 3–7) from `internal/domain/map/entity/placeholders.go`. The file keeps `Decoration`, `MapItem`, and `BgImage`:

```go
package entity

type Decoration struct {
	ID       string  `json:"id"`
	URL      string  `json:"url"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Width    float64 `json:"width"`
	Height   float64 `json:"height"`
	Rotation float64 `json:"rotation"`
	ZOrder   int     `json:"z_order"`
	Opacity  float64 `json:"opacity"`
}

type MapItem struct {
	ID        string `json:"id"`
	ItemDefID string `json:"item_def_id"`
}

type BgImage struct {
	URL      string  `json:"url"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Width    float64 `json:"width"`
	Height   float64 `json:"height"`
	Rotation float64 `json:"rotation"`
	Opacity  float64 `json:"opacity"`
}
```

- [ ] **Step 2: Update `map.go` to use `[]WallSegment`**

In `internal/domain/map/entity/map.go`, change:
- Field: `Walls []Wall` → `Walls []WallSegment`
- In `NewTacticalMap`: `Walls: []Wall{}` → `Walls: []WallSegment{}`

Full updated file:

```go
package entity

import (
	"time"

	"github.com/google/uuid"
)

type TacticalMap struct {
	ID          uuid.UUID     `json:"id"`
	CampaignID  uuid.UUID     `json:"campaign_id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Grid        GridShape     `json:"grid"`
	Bg          *BgImage      `json:"bg"`
	Pieces      []Piece       `json:"pieces"`
	Walls       []WallSegment `json:"walls"`
	Decorations []Decoration  `json:"decorations"`
	Items       []MapItem     `json:"items"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

func NewTacticalMap(campaignID uuid.UUID, name, description string) *TacticalMap {
	now := time.Now().UTC()
	return &TacticalMap{
		ID:          uuid.New(),
		CampaignID:  campaignID,
		Name:        name,
		Description: description,
		Grid:        DefaultGrid(),
		Bg:          nil,
		Pieces:      []Piece{},
		Walls:       []WallSegment{},
		Decorations: []Decoration{},
		Items:       []MapItem{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}
```

- [ ] **Step 3: Update `mapper.go` to unmarshal `[]WallSegment`**

In `internal/gateway/pg/map/mapper.go`, change the `walls` unmarshal block from `[]entity.Wall{}` to `[]entity.WallSegment{}`:

```go
walls := []entity.WallSegment{}
if err := json.Unmarshal(m.Walls, &walls); err != nil {
    return nil, fmt.Errorf("unmarshal walls: %w", err)
}
```

- [ ] **Step 4: Update `map_response.go`**

In `internal/app/api/map/map_response.go`, change `Walls any` to `Walls []entity.WallSegment` and fix the nil-guard:

```go
type MapResponse struct {
	ID          uuid.UUID              `json:"id"`
	CampaignID  uuid.UUID              `json:"campaign_id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Grid        GridShapeResponse      `json:"grid"`
	Bg          any                    `json:"bg"`
	Pieces      any                    `json:"pieces"`
	Walls       []entity.WallSegment   `json:"walls"`
	Decorations any                    `json:"decorations"`
	Items       any                    `json:"items"`
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at"`
}
```

And in `toMapResponse`, replace the walls nil-guard with:

```go
walls := m.Walls
if walls == nil {
    walls = []entity.WallSegment{}
}
```

The rest of `toMapResponse` stays the same; just change the `Walls: walls` field assignment.

- [ ] **Step 5: Build to verify no compilation errors**

```bash
go build ./...
```

Expected: success. If the build fails, fix the import issue (ensure `internal/app/api/map/map_response.go` imports `entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"`).

- [ ] **Step 6: Run unit tests**

```bash
go test ./internal/domain/map/...
```

Expected: all PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/domain/map/entity/placeholders.go \
        internal/domain/map/entity/map.go \
        internal/gateway/pg/map/mapper.go \
        internal/app/api/map/map_response.go
git commit -m "refactor(map): replace Wall placeholder with WallSegment throughout"
```

---

### Task 5: Wire `Walls` into the update use case and HTTP handler

**Files:**
- Modify: `internal/application/map/update_map.go`
- Modify: `internal/app/api/map/update_map.go`

- [ ] **Step 1: Add `Walls` to `UpdateMapInput`**

In `internal/application/map/update_map.go`, add field to `UpdateMapInput`:

```go
type UpdateMapInput struct {
	RequesterID uuid.UUID
	MapID       uuid.UUID
	Name        *string
	Description string
	Grid        *entity.GridShape
	Bg          *entity.BgImage
	Pieces      *[]entity.Piece
	Walls       *[]entity.WallSegment // nil = keep existing; empty slice = clear all
}
```

Then, in `UpdateMapUC.UpdateMap`, add wall validation and application after the existing `if input.Pieces != nil` block:

```go
if input.Walls != nil {
    if err := service.ValidateWallSegments(*input.Walls); err != nil {
        return err
    }
    m.Walls = *input.Walls
}
```

- [ ] **Step 2: Add `Walls` to `UpdateMapRequestBody`**

In `internal/app/api/map/update_map.go`, add to `UpdateMapRequestBody`:

```go
Walls *[]entity.WallSegment `json:"walls" required:"false" doc:"Wall segments; omit to keep existing, send [] to clear all"`
```

And in `UpdateMapHandler`, add to the `UpdateMapInput` literal:

```go
Walls: req.Body.Walls,
```

- [ ] **Step 3: Build**

```bash
go build ./...
```

Expected: success.

- [ ] **Step 4: Run unit tests**

```bash
go test ./internal/...
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/application/map/update_map.go \
        internal/app/api/map/update_map.go
git commit -m "feat(map): accept walls in PUT /maps/:id"
```

---

### Task 6: Integration test — wall segment round-trip

**Files:**
- Modify: `internal/gateway/pg/map/map_repository_test.go`

- [ ] **Step 1: Add the integration test**

Append to `internal/gateway/pg/map/map_repository_test.go`:

```go
func TestMapRepository_WallsRoundTrip(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	repo := pgmap.NewRepository(pool)

	masterStr := pgtest.InsertTestUser(t, pool, "master_wrt", "master_wrt@hunter.com", "pass")
	campaignStr := pgtest.InsertTestCampaign(t, pool, masterStr, "Test Campaign WRT")
	campaignID, err := uuid.Parse(campaignStr)
	if err != nil {
		t.Fatalf("parse campaign uuid: %v", err)
	}

	m := entity.NewTacticalMap(campaignID, "Walled Room", "")
	if err := repo.CreateMap(ctx, m); err != nil {
		t.Fatalf("CreateMap: %v", err)
	}

	doorSubtype := entity.DoorSubtypeBasic
	m.Walls = []entity.WallSegment{
		{
			ID:          "wall-uuid-0001",
			P1:          [2]float64{0, 0},
			P2:          [2]float64{64, 0},
			WallType:    entity.WallTypeWall,
			Material:    entity.WallMaterialStone,
			Move:        true,
			Sense:       entity.SenseFull,
			Direction:   entity.WallDirectionBoth,
			HP:          100,
			MaxHP:       100,
			Resistance:  5,
		},
		{
			ID:          "wall-uuid-0002",
			P1:          [2]float64{64, 0},
			P2:          [2]float64{64, 64},
			WallType:    entity.WallTypeDoor,
			Material:    entity.WallMaterialWood,
			DoorSubtype: &doorSubtype,
			Move:        true,
			Sense:       entity.SenseFull,
			Direction:   entity.WallDirectionBoth,
			Open:        false,
			Locked:      true,
			HP:          40,
			MaxHP:       40,
			Resistance:  2,
		},
	}
	if err := repo.UpdateMap(ctx, m); err != nil {
		t.Fatalf("UpdateMap with walls: %v", err)
	}

	got, err := repo.GetMap(ctx, m.ID)
	if err != nil {
		t.Fatalf("GetMap: %v", err)
	}
	if len(got.Walls) != 2 {
		t.Fatalf("expected 2 walls, got %d", len(got.Walls))
	}
	if got.Walls[0].WallType != entity.WallTypeWall {
		t.Errorf("expected WallTypeWall, got %s", got.Walls[0].WallType)
	}
	if got.Walls[1].WallType != entity.WallTypeDoor {
		t.Errorf("expected WallTypeDoor, got %s", got.Walls[1].WallType)
	}
	if got.Walls[1].DoorSubtype == nil || *got.Walls[1].DoorSubtype != entity.DoorSubtypeBasic {
		t.Errorf("expected DoorSubtypeBasic, got %v", got.Walls[1].DoorSubtype)
	}
	if !got.Walls[1].Locked {
		t.Error("expected wall[1].Locked = true")
	}
}
```

- [ ] **Step 2: Run integration test (requires a running Postgres)**

```bash
go test -tags=integration ./internal/gateway/pg/map/... -run TestMapRepository_WallsRoundTrip -v
```

Expected:

```
--- PASS: TestMapRepository_WallsRoundTrip (0.05s)
PASS
```

If the Postgres test DB is not running, start it first: check `internal/gateway/pg/pgtest/setup.go` for the connection string used by `pgtest.SetupTestDB`.

- [ ] **Step 3: Run all map integration tests**

```bash
go test -tags=integration ./internal/gateway/pg/map/... -v
```

Expected: all PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/gateway/pg/map/map_repository_test.go
git commit -m "test(map): add wall segment round-trip integration test"
```

---

### Task 7: Update API contract — `maps.md`

**Files:**
- Modify: `docs/dev/api/maps.md`

- [ ] **Step 1: Add `walls` to the `PUT /maps/:id` request table**

In the `PUT /maps/:id` section, add a row to the field rules table:

```markdown
| `walls` | opcional; omitir mantém existente; `[]` remove todas as paredes |
```

Update the PUT request body example to include walls:

```json
{
  "name": "Floresta do Norte — Revisado",
  "description": "Nova descrição do mapa",
  "grid": { "kind": "square", "cols": 25, "rows": 25, "cell_size": 64, "skew_ratio": 1.0, "rotation": 0, "color": "#ffffff", "opacity": 0.5, "line_style": "solid" },
  "bg": null,
  "pieces": [
    { "id": "uuid", "character_id": "uuid", "coord": { "slot": { "kind": "square", "col": 3, "row": 5 }, "z": 0 }, "visible": true }
  ],
  "walls": [
    {
      "id": "uuid",
      "p1": [0, 0],
      "p2": [64, 0],
      "wall_type": "wall",
      "material": "stone",
      "move": true,
      "sense": "full",
      "direction": "both",
      "open": false,
      "locked": false,
      "hp": 100,
      "max_hp": 100,
      "resistance": 5,
      "destroyed": false
    }
  ]
}
```

Add `walls` to the `PUT /maps/:id` 422 response table note (it now validates walls too).

- [ ] **Step 2: Update `MapResponse` to show `WallSegment` shape**

In the `## MapResponse` section, replace `"walls": []` with a full example:

```json
"walls": [
  {
    "id": "uuid",
    "p1": [0, 0],
    "p2": [64, 0],
    "wall_type": "wall",
    "material": "stone",
    "move": true,
    "sense": "full",
    "direction": "both",
    "open": false,
    "locked": false,
    "hp": 100,
    "max_hp": 100,
    "resistance": 5,
    "destroyed": false
  }
]
```

- [ ] **Step 3: Add WallSegment schema section**

Append a new `## WallSegment — Formato do segmento de parede` section at the bottom of `maps.md`:

```markdown
## WallSegment — Formato do segmento de parede

```json
{
  "id": "uuid-string",
  "p1": [0.0, 0.0],
  "p2": [64.0, 0.0],
  "wall_type": "wall | door | window | secret_door | terrain",
  "material": "stone | wood | iron | magical",
  "door_subtype": "basic | double | portcullis | drawbridge",
  "window_subtype": "basic | barred | shuttered",
  "move": true,
  "sense": "full | sight | none",
  "direction": "both | left | right",
  "open": false,
  "locked": false,
  "hp": 100,
  "max_hp": 100,
  "resistance": 5,
  "destroyed": false
}
```

| Campo | Tipo | Descrição |
|---|---|---|
| `id` | string (UUID) | Identificador único do segmento, gerado pelo frontend via `crypto.randomUUID()` |
| `p1`, `p2` | `[number, number]` | Endpoints em coordenadas de mundo (pré-transform); `p1 ≠ p2` |
| `wall_type` | enum | Comportamento funcional |
| `material` | enum | Propriedades físicas (HP, resistência, cor) |
| `door_subtype` | enum? | Presente apenas quando `wall_type = "door"` |
| `window_subtype` | enum? | Presente apenas quando `wall_type = "window"` |
| `move` | bool | Bloqueia movimento físico |
| `sense` | enum | O que bloqueia em termos de percepção |
| `direction` | enum | Direção de bloqueio (both = nos dois sentidos) |
| `open` | bool | Porta/janela está aberta (só relevante para door/window) |
| `locked` | bool | Porta trancada |
| `hp` | int | Pontos de vida atuais (≥ 0) |
| `max_hp` | int | Pontos de vida máximos |
| `resistance` | int | Dano absorvido por ataque |
| `destroyed` | bool | Segmento destruído (visual alterado) |

### Defaults por tipo (aplicados pelo frontend ao criar o segmento)

| `wall_type` | `move` | `sense` | `direction` | `material` padrão |
|---|---|---|---|---|
| `wall` | `true` | `full` | `both` | `stone` |
| `door` | `true` | `full` | `both` | `wood` |
| `window` | `true` | `none` | `both` | `wood` |
| `secret_door` | `true` | `full` | `both` | `stone` |
| `terrain` | `true` | `none` | `left` | — |

### Notas gerais de validação (backend)

- `PUT /maps/:id` aceita `walls` como campo opcional. `null` ou ausente = mantém as paredes existentes. `[]` = remove todas.
- Validações: `p1 ≠ p2`; `wall_type` deve ser um dos 5 valores válidos; `hp ≥ 0`.
- O backend não calcula defaults — o frontend envia o objeto completo.
```

- [ ] **Step 4: Update the "Notas gerais" section**

In the existing `### Notas gerais` at the bottom, update the line about walls:

Old:
```
- Criação e atualização aceitam `name`, `description`, `grid`, `bg` e `pieces`. `walls`, `decorations` e `items` são gerenciados por endpoints futuros.
```

New:
```
- `POST /campaigns/:id/maps` (criação) aceita `name`, `description`, `grid`, `bg` e `pieces`. `walls`, `decorations` e `items` ficam `[]` na criação.
- `PUT /maps/:id` aceita adicionalmente `walls` como lista de `WallSegment`. `decorations` e `items` ainda não são suportados no request (gerenciados por fases futuras).
```

- [ ] **Step 5: Commit**

```bash
git add docs/dev/api/maps.md
git commit -m "docs(api): update maps.md with WallSegment schema in PUT /maps/:id"
```

---

### Task 8: Update `documentation-map.yaml`

**Files:**
- Modify: `docs/documentation-map.yaml`

- [ ] **Step 1: Add entries for the map domain**

In `docs/documentation-map.yaml`, find the `# ─── Map: REST API ───` block (around line 326). Add a new block for the map domain **before** it:

```yaml
  # ─── Map: Domain Entity + Service ───
  - code_path: internal/domain/map/
    dev_docs:
      - path: docs/dev/api/maps.md
        confidence: directly_affected
    notes: Map entity (TacticalMap, WallSegment, GridShape, Piece, etc.) and validator — any shape change affects the REST contract
```

- [ ] **Step 2: Commit**

```bash
git add docs/documentation-map.yaml
git commit -m "docs: register internal/domain/map/ in documentation-map.yaml"
```

---

## Self-Review

**Spec coverage check:**

| Spec requirement | Covered by |
|---|---|
| `wall_segment.go` with all fields from §2.3 + enums | Task 2 |
| `maps_mapper.go` — serialize/deserialize `[]WallSegment` | Task 4 (mapper.go) |
| `map_validator.go` — validate p1≠p2, wallType valid, hp≥0 | Task 3 |
| `maps.md` updated — `walls` shape in PUT /maps/:id | Task 7 |
| `documentation-map.yaml` registration | Task 8 |
| `TacticalMap.Walls []Wall → []WallSegment` | Task 4 (map.go) |
| Handler accepts `walls` in request body | Task 5 |
| Integration test for wall round-trip | Task 6 |

**Placeholder scan:** none found — all steps have complete code.

**Type consistency check:**
- `WallSegment.WallType` uses `WallType` type defined in Task 2; validator in Task 3 switches on `entity.WallTypeWall`, `entity.WallTypeDoor`, etc. — all match.
- `[]entity.WallSegment` used consistently across mapper (Task 4), use case (Task 5), handler (Task 5), response type (Task 4), and integration test (Task 6).
- `ValidateWallSegments` exported name used identically in Task 1 (test), Task 3 (implementation), and Task 5 (use case call).
- `ErrWallSameEndpoints`, `ErrWallInvalidType`, `ErrWallNegativeHP` declared in Task 3 and referenced in Task 1 tests — consistent.

---

**Plan complete and saved to `docs/superpowers/plans/2026-06-10-tactical-map-fase-10-backend.md`.**

**Two execution options:**

**1. Subagent-Driven (recommended)** — disparo de subagente fresco por task, revisão entre tasks, iteração rápida.

**2. Inline Execution** — execução das tasks nesta sessão via `executing-plans`, com checkpoints de revisão.

**Qual abordagem?**
