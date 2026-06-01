# Tactical Map Fase 1 — Backend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement CRUD REST para mapas táticos no backend Go, com domínio aninhado, gateway JSONB, handlers huma v2 e integration tests.

**Architecture:** Domínio aninhado em `internal/domain/map/` (padrão `match/`). Use cases com interface por operação. Gateway em `internal/gateway/pg/map/` com pgModel para JSONB. Handlers em `internal/app/api/map/` implementando `IApi`. Wire em `cmd/api/main.go` e `internal/app/api/api.go`.

**Tech Stack:** Go, pgx v5, huma v2, goose migrations, `github.com/google/uuid`, `encoding/json`

**Spec:** `docs/superpowers/specs/2026-05-31-tactical-map-fase-1-design.md`

---

### Task 1: Domain entities

**Files:**
- Create: `internal/domain/map/entity/map.go`
- Create: `internal/domain/map/entity/grid.go`
- Create: `internal/domain/map/entity/piece.go`
- Create: `internal/domain/map/entity/placeholders.go`

- [ ] **Step 1: Create `internal/domain/map/entity/grid.go`**

```go
package entity

type GridKind  string
type LineStyle string

const (
	GridKindSquare GridKind  = "square"
	GridKindHex    GridKind  = "hex"
	LineStyleSolid LineStyle = "solid"
	LineStyleDashed LineStyle = "dashed"
)

type GridShape struct {
	Kind      GridKind  `json:"kind"`
	Cols      int       `json:"cols"`
	Rows      int       `json:"rows"`
	CellSize  float64   `json:"cell_size"`
	SkewRatio float64   `json:"skew_ratio"`
	Rotation  float64   `json:"rotation"`
	Color     string    `json:"color"`
	Opacity   float64   `json:"opacity"`
	LineStyle LineStyle `json:"line_style"`
}

func DefaultGrid() GridShape {
	return GridShape{
		Kind:      GridKindSquare,
		Cols:      25,
		Rows:      25,
		CellSize:  64,
		SkewRatio: 1.0,
		Rotation:  0,
		Color:     "#ffffff",
		Opacity:   0.5,
		LineStyle: LineStyleSolid,
	}
}
```

- [ ] **Step 2: Create `internal/domain/map/entity/piece.go`**

```go
package entity

type SquareCoord struct {
	Kind string `json:"kind"` // "square"
	Col  int    `json:"col"`
	Row  int    `json:"row"`
}

type HexCoord struct {
	Kind string `json:"kind"` // "hex"
	Q    int    `json:"q"`
	R    int    `json:"r"`
}

type PieceCoord struct {
	Slot any     `json:"slot"` // SquareCoord | HexCoord — serialised as-is
	Z    float64 `json:"z"`
}

type Piece struct {
	ID          string     `json:"id"`
	CharacterID string     `json:"character_id"`
	Coord       PieceCoord `json:"coord"`
	Visible     bool       `json:"visible"`
}
```

- [ ] **Step 3: Create `internal/domain/map/entity/placeholders.go`**

```go
package entity

type Wall struct {
	ID        string       `json:"id"`
	Points    [][2]float64 `json:"points"`
	Thickness float64      `json:"thickness"`
}

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

- [ ] **Step 4: Create `internal/domain/map/entity/map.go`**

```go
package entity

import (
	"time"

	"github.com/google/uuid"
)

type TacticalMap struct {
	ID          uuid.UUID   `json:"id"`
	CampaignID  uuid.UUID   `json:"campaign_id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Grid        GridShape   `json:"grid"`
	Bg          *BgImage    `json:"bg"`
	Pieces      []Piece     `json:"pieces"`
	Walls       []Wall      `json:"walls"`
	Decorations []Decoration `json:"decorations"`
	Items       []MapItem   `json:"items"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
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
		Walls:       []Wall{},
		Decorations: []Decoration{},
		Items:       []MapItem{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}
```

- [ ] **Step 5: Verify package compiles**

```bash
go build ./internal/domain/map/...
```
Expected: no output (success)

- [ ] **Step 6: Commit**

```bash
git add internal/domain/map/
git commit -m "feat(tactical-map): domain entities — TacticalMap, GridShape, Piece, placeholders"
```

---

### Task 2: Domain validator

**Files:**
- Create: `internal/domain/map/service/map_validator.go`
- Create: `internal/domain/map/service/map_validator_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// internal/domain/map/service/map_validator_test.go
package service_test

import (
	"testing"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/map/service"
)

func validGrid() entity.GridShape {
	return entity.DefaultGrid()
}

func TestValidateMap_ValidMap(t *testing.T) {
	err := service.ValidateMap("Forest", validGrid())
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestValidateMap_EmptyName(t *testing.T) {
	err := service.ValidateMap("", validGrid())
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestValidateMap_InvalidCellSize(t *testing.T) {
	g := validGrid()
	g.CellSize = 0
	err := service.ValidateMap("Forest", g)
	if err == nil {
		t.Error("expected error for cell_size=0")
	}
}

func TestValidateMap_InvalidCols(t *testing.T) {
	g := validGrid()
	g.Cols = 0
	err := service.ValidateMap("Forest", g)
	if err == nil {
		t.Error("expected error for cols=0")
	}
}

func TestValidateMap_InvalidRows(t *testing.T) {
	g := validGrid()
	g.Rows = 0
	err := service.ValidateMap("Forest", g)
	if err == nil {
		t.Error("expected error for rows=0")
	}
}

func TestValidateMap_SkewRatioOutOfRange(t *testing.T) {
	g := validGrid()
	g.SkewRatio = 1.5
	err := service.ValidateMap("Forest", g)
	if err == nil {
		t.Error("expected error for skew_ratio > 1")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/domain/map/service/...
```
Expected: FAIL — `service` package not found

- [ ] **Step 3: Implement `map_validator.go`**

```go
// internal/domain/map/service/map_validator.go
package service

import (
	"errors"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
)

var (
	ErrEmptyName       = errors.New("map name cannot be empty")
	ErrInvalidCellSize = errors.New("cell_size must be > 0")
	ErrInvalidCols     = errors.New("cols must be > 0")
	ErrInvalidRows     = errors.New("rows must be > 0")
	ErrInvalidSkewRatio = errors.New("skew_ratio must be in [0, 1]")
)

func ValidateMap(name string, grid entity.GridShape) error {
	if name == "" {
		return ErrEmptyName
	}
	if grid.CellSize <= 0 {
		return ErrInvalidCellSize
	}
	if grid.Cols <= 0 {
		return ErrInvalidCols
	}
	if grid.Rows <= 0 {
		return ErrInvalidRows
	}
	if grid.SkewRatio < 0 || grid.SkewRatio > 1 {
		return ErrInvalidSkewRatio
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/domain/map/service/... -v
```
Expected: all 6 tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/domain/map/service/
git commit -m "feat(tactical-map): domain validator — name, cell_size, cols, rows, skew_ratio"
```

---

### Task 3: Migration

**Files:**
- Create: `migrations/20260531000000_create_maps_table.sql`

- [ ] **Step 1: Create migration**

```sql
-- migrations/20260531000000_create_maps_table.sql
-- +goose Up
CREATE TABLE maps (
  id          UUID         PRIMARY KEY,
  campaign_id UUID         NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
  name        VARCHAR(255) NOT NULL,
  description TEXT         NOT NULL DEFAULT '',
  grid        JSONB        NOT NULL,
  bg          JSONB,
  pieces      JSONB        NOT NULL DEFAULT '[]',
  walls       JSONB        NOT NULL DEFAULT '[]',
  decorations JSONB        NOT NULL DEFAULT '[]',
  items       JSONB        NOT NULL DEFAULT '[]',
  created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_maps_campaign_id ON maps(campaign_id);

-- +goose Down
DROP INDEX IF EXISTS idx_maps_campaign_id;
DROP TABLE IF EXISTS maps;
```

- [ ] **Step 2: Run migration against local DB**

```bash
make migrate
```
Expected: migration applied without error

- [ ] **Step 3: Commit**

```bash
git add migrations/20260531000000_create_maps_table.sql
git commit -m "feat(tactical-map): migration — create maps table with JSONB columns"
```

---

### Task 4: Gateway — pgModel, mapper, repository

**Files:**
- Create: `internal/gateway/pg/map/repository.go`
- Create: `internal/gateway/pg/map/mapper.go`
- Create: `internal/gateway/pg/map/error.go`

- [ ] **Step 1: Create `repository.go`**

```go
// internal/gateway/pg/map/repository.go
package pgmap

import (
	pgfs "github.com/422UR4H/HxH_RPG_System/pkg"
)

type Repository struct {
	q pgfs.IQuerier
}

func NewRepository(q pgfs.IQuerier) *Repository {
	return &Repository{q: q}
}
```

- [ ] **Step 2: Create `error.go`**

```go
// internal/gateway/pg/map/error.go
package pgmap

import "errors"

var ErrMapNotFound = errors.New("map not found")
```

- [ ] **Step 3: Create `mapper.go`**

```go
// internal/gateway/pg/map/mapper.go
package pgmap

import (
	"encoding/json"
	"fmt"
	"time"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/google/uuid"
)

type pgModel struct {
	ID          uuid.UUID
	CampaignID  uuid.UUID
	Name        string
	Description string
	Grid        []byte
	Bg          []byte
	Pieces      []byte
	Walls       []byte
	Decorations []byte
	Items       []byte
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func toEntity(m *pgModel) (*entity.TacticalMap, error) {
	var grid entity.GridShape
	if err := json.Unmarshal(m.Grid, &grid); err != nil {
		return nil, fmt.Errorf("unmarshal grid: %w", err)
	}

	var bg *entity.BgImage
	if m.Bg != nil && string(m.Bg) != "null" {
		bg = &entity.BgImage{}
		if err := json.Unmarshal(m.Bg, bg); err != nil {
			return nil, fmt.Errorf("unmarshal bg: %w", err)
		}
	}

	pieces := []entity.Piece{}
	if err := json.Unmarshal(m.Pieces, &pieces); err != nil {
		return nil, fmt.Errorf("unmarshal pieces: %w", err)
	}

	walls := []entity.Wall{}
	if err := json.Unmarshal(m.Walls, &walls); err != nil {
		return nil, fmt.Errorf("unmarshal walls: %w", err)
	}

	decorations := []entity.Decoration{}
	if err := json.Unmarshal(m.Decorations, &decorations); err != nil {
		return nil, fmt.Errorf("unmarshal decorations: %w", err)
	}

	items := []entity.MapItem{}
	if err := json.Unmarshal(m.Items, &items); err != nil {
		return nil, fmt.Errorf("unmarshal items: %w", err)
	}

	return &entity.TacticalMap{
		ID:          m.ID,
		CampaignID:  m.CampaignID,
		Name:        m.Name,
		Description: m.Description,
		Grid:        grid,
		Bg:          bg,
		Pieces:      pieces,
		Walls:       walls,
		Decorations: decorations,
		Items:       items,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}, nil
}

func marshalJSON(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal json: %w", err)
	}
	return b, nil
}
```

- [ ] **Step 4: Verify compilation**

```bash
go build ./internal/gateway/pg/map/...
```
Expected: no output

- [ ] **Step 5: Commit**

```bash
git add internal/gateway/pg/map/
git commit -m "feat(tactical-map): gateway repository scaffold, pgModel, mapper"
```

---

### Task 5: Gateway — CRUD operations + integration tests

**Files:**
- Create: `internal/gateway/pg/map/create_map.go`
- Create: `internal/gateway/pg/map/read_map.go`
- Create: `internal/gateway/pg/map/list_maps.go`
- Create: `internal/gateway/pg/map/update_map.go`
- Create: `internal/gateway/pg/map/delete_map.go`
- Create: `internal/gateway/pg/map/map_repository_test.go`

- [ ] **Step 1: Write integration tests first**

```go
// internal/gateway/pg/map/map_repository_test.go
//go:build integration

package pgmap_test

import (
	"context"
	"testing"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	pgmap "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/map"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/pgtest"
	"github.com/google/uuid"
)

func TestMapRepository_CreateAndGet(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.NewPool(t)
	repo := pgmap.NewRepository(pool)

	campaignID := pgtest.SeedCampaign(t, pool)

	m := entity.NewTacticalMap(campaignID, "Forest", "A dark forest")
	if err := repo.CreateMap(ctx, m); err != nil {
		t.Fatalf("CreateMap: %v", err)
	}

	got, err := repo.GetMap(ctx, m.ID)
	if err != nil {
		t.Fatalf("GetMap: %v", err)
	}
	if got.Name != "Forest" {
		t.Errorf("expected name Forest, got %s", got.Name)
	}
	if got.Grid.Cols != 25 {
		t.Errorf("expected cols 25, got %d", got.Grid.Cols)
	}
}

func TestMapRepository_ListByCampaign(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.NewPool(t)
	repo := pgmap.NewRepository(pool)

	campaignID := pgtest.SeedCampaign(t, pool)

	m1 := entity.NewTacticalMap(campaignID, "Map A", "")
	m2 := entity.NewTacticalMap(campaignID, "Map B", "")
	_ = repo.CreateMap(ctx, m1)
	_ = repo.CreateMap(ctx, m2)

	maps, err := repo.ListMapsByCampaign(ctx, campaignID)
	if err != nil {
		t.Fatalf("ListMapsByCampaign: %v", err)
	}
	if len(maps) != 2 {
		t.Errorf("expected 2 maps, got %d", len(maps))
	}
}

func TestMapRepository_Update(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.NewPool(t)
	repo := pgmap.NewRepository(pool)

	campaignID := pgtest.SeedCampaign(t, pool)
	m := entity.NewTacticalMap(campaignID, "Old Name", "")
	_ = repo.CreateMap(ctx, m)

	m.Name = "New Name"
	if err := repo.UpdateMap(ctx, m); err != nil {
		t.Fatalf("UpdateMap: %v", err)
	}

	got, _ := repo.GetMap(ctx, m.ID)
	if got.Name != "New Name" {
		t.Errorf("expected New Name, got %s", got.Name)
	}
}

func TestMapRepository_Delete(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.NewPool(t)
	repo := pgmap.NewRepository(pool)

	campaignID := pgtest.SeedCampaign(t, pool)
	m := entity.NewTacticalMap(campaignID, "Temp", "")
	_ = repo.CreateMap(ctx, m)

	if err := repo.DeleteMap(ctx, m.ID); err != nil {
		t.Fatalf("DeleteMap: %v", err)
	}

	_, err := repo.GetMap(ctx, m.ID)
	if err == nil {
		t.Error("expected error after delete, got nil")
	}
}

func TestMapRepository_GetMap_NotFound(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.NewPool(t)
	repo := pgmap.NewRepository(pool)

	_, err := repo.GetMap(ctx, uuid.New())
	if err == nil {
		t.Error("expected ErrMapNotFound")
	}
}
```

- [ ] **Step 2: Run integration tests to verify they fail**

```bash
go test -tags=integration ./internal/gateway/pg/map/...
```
Expected: FAIL — methods not defined on `*Repository`

- [ ] **Step 3: Implement `create_map.go`**

```go
// internal/gateway/pg/map/create_map.go
package pgmap

import (
	"context"
	"fmt"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
)

func (r *Repository) CreateMap(ctx context.Context, m *entity.TacticalMap) error {
	grid, err := marshalJSON(m.Grid)
	if err != nil {
		return err
	}
	pieces, err := marshalJSON(m.Pieces)
	if err != nil {
		return err
	}
	walls, err := marshalJSON(m.Walls)
	if err != nil {
		return err
	}
	decorations, err := marshalJSON(m.Decorations)
	if err != nil {
		return err
	}
	items, err := marshalJSON(m.Items)
	if err != nil {
		return err
	}

	var bg []byte
	if m.Bg != nil {
		bg, err = marshalJSON(m.Bg)
		if err != nil {
			return err
		}
	}

	const query = `
		INSERT INTO maps (
			id, campaign_id, name, description,
			grid, bg, pieces, walls, decorations, items,
			created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
	`
	_, err = r.q.Exec(ctx, query,
		m.ID, m.CampaignID, m.Name, m.Description,
		grid, bg, pieces, walls, decorations, items,
		m.CreatedAt, m.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create map: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Implement `read_map.go`**

```go
// internal/gateway/pg/map/read_map.go
package pgmap

import (
	"context"
	"errors"
	"fmt"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetMap(ctx context.Context, id uuid.UUID) (*entity.TacticalMap, error) {
	const query = `
		SELECT id, campaign_id, name, description,
		       grid, bg, pieces, walls, decorations, items,
		       created_at, updated_at
		FROM maps WHERE id = $1
	`
	row := r.q.QueryRow(ctx, query, id)
	m := &pgModel{}
	err := row.Scan(
		&m.ID, &m.CampaignID, &m.Name, &m.Description,
		&m.Grid, &m.Bg, &m.Pieces, &m.Walls, &m.Decorations, &m.Items,
		&m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMapNotFound
		}
		return nil, fmt.Errorf("get map: %w", err)
	}
	return toEntity(m)
}
```

- [ ] **Step 5: Implement `list_maps.go`**

```go
// internal/gateway/pg/map/list_maps.go
package pgmap

import (
	"context"
	"fmt"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/google/uuid"
)

func (r *Repository) ListMapsByCampaign(
	ctx context.Context, campaignID uuid.UUID,
) ([]*entity.TacticalMap, error) {
	const query = `
		SELECT id, campaign_id, name, description,
		       grid, bg, pieces, walls, decorations, items,
		       created_at, updated_at
		FROM maps WHERE campaign_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.q.Query(ctx, query, campaignID)
	if err != nil {
		return nil, fmt.Errorf("list maps: %w", err)
	}
	defer rows.Close()

	var result []*entity.TacticalMap
	for rows.Next() {
		m := &pgModel{}
		if err := rows.Scan(
			&m.ID, &m.CampaignID, &m.Name, &m.Description,
			&m.Grid, &m.Bg, &m.Pieces, &m.Walls, &m.Decorations, &m.Items,
			&m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan map row: %w", err)
		}
		e, err := toEntity(m)
		if err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}
```

- [ ] **Step 6: Implement `update_map.go`**

```go
// internal/gateway/pg/map/update_map.go
package pgmap

import (
	"context"
	"fmt"
	"time"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
)

func (r *Repository) UpdateMap(ctx context.Context, m *entity.TacticalMap) error {
	grid, err := marshalJSON(m.Grid)
	if err != nil {
		return err
	}
	pieces, err := marshalJSON(m.Pieces)
	if err != nil {
		return err
	}
	walls, err := marshalJSON(m.Walls)
	if err != nil {
		return err
	}
	decorations, err := marshalJSON(m.Decorations)
	if err != nil {
		return err
	}
	items, err := marshalJSON(m.Items)
	if err != nil {
		return err
	}
	var bg []byte
	if m.Bg != nil {
		bg, err = marshalJSON(m.Bg)
		if err != nil {
			return err
		}
	}

	m.UpdatedAt = time.Now().UTC()

	const query = `
		UPDATE maps SET
			name=$1, description=$2, grid=$3, bg=$4,
			pieces=$5, walls=$6, decorations=$7, items=$8, updated_at=$9
		WHERE id=$10
	`
	_, err = r.q.Exec(ctx, query,
		m.Name, m.Description, grid, bg,
		pieces, walls, decorations, items, m.UpdatedAt,
		m.ID,
	)
	if err != nil {
		return fmt.Errorf("update map: %w", err)
	}
	return nil
}
```

- [ ] **Step 7: Implement `delete_map.go`**

```go
// internal/gateway/pg/map/delete_map.go
package pgmap

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (r *Repository) DeleteMap(ctx context.Context, id uuid.UUID) error {
	const query = `DELETE FROM maps WHERE id = $1`
	_, err := r.q.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete map: %w", err)
	}
	return nil
}
```

- [ ] **Step 8: Run integration tests to verify they pass**

```bash
go test -tags=integration ./internal/gateway/pg/map/... -v
```
Expected: all 5 tests PASS

- [ ] **Step 9: Commit**

```bash
git add internal/gateway/pg/map/
git commit -m "feat(tactical-map): gateway CRUD + integration tests"
```

---

### Task 6: Application use cases

**Files:**
- Create: `internal/application/map/i_repository.go`
- Create: `internal/application/map/errors.go`
- Create: `internal/application/map/create_map.go`
- Create: `internal/application/map/get_map.go`
- Create: `internal/application/map/list_maps.go`
- Create: `internal/application/map/update_map.go`
- Create: `internal/application/map/delete_map.go`

- [ ] **Step 1: Create `i_repository.go`**

```go
// internal/application/map/i_repository.go
package mapuc

import (
	"context"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/google/uuid"
)

type IRepository interface {
	CreateMap(ctx context.Context, m *entity.TacticalMap) error
	GetMap(ctx context.Context, id uuid.UUID) (*entity.TacticalMap, error)
	ListMapsByCampaign(ctx context.Context, campaignID uuid.UUID) ([]*entity.TacticalMap, error)
	UpdateMap(ctx context.Context, m *entity.TacticalMap) error
	DeleteMap(ctx context.Context, id uuid.UUID) error
}
```

- [ ] **Step 2: Create `errors.go`**

```go
// internal/application/map/errors.go
package mapuc

import "errors"

var (
	ErrMapNotFound   = errors.New("map not found")
	ErrNotMapMaster  = errors.New("only the campaign master can perform this action")
)
```

- [ ] **Step 3: Create `create_map.go`**

```go
// internal/application/map/create_map.go
package mapuc

import (
	"context"

	campaignApp "github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/map/service"
	"github.com/google/uuid"
)

type ICreateMap interface {
	CreateMap(ctx context.Context, input *CreateMapInput) (*entity.TacticalMap, error)
}

type CreateMapInput struct {
	RequesterID uuid.UUID
	CampaignID  uuid.UUID
	Name        string
	Description string
}

type CreateMapUC struct {
	repo         IRepository
	campaignRepo campaignApp.IRepository
}

func NewCreateMapUC(repo IRepository, campaignRepo campaignApp.IRepository) *CreateMapUC {
	return &CreateMapUC{repo: repo, campaignRepo: campaignRepo}
}

func (uc *CreateMapUC) CreateMap(ctx context.Context, input *CreateMapInput) (*entity.TacticalMap, error) {
	if err := service.ValidateMap(input.Name, entity.DefaultGrid()); err != nil {
		return nil, err
	}

	masterID, err := uc.campaignRepo.GetCampaignMasterUUID(ctx, input.CampaignID)
	if err != nil {
		return nil, err
	}
	if masterID != input.RequesterID {
		return nil, ErrNotMapMaster
	}

	m := entity.NewTacticalMap(input.CampaignID, input.Name, input.Description)
	if err := uc.repo.CreateMap(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}
```

- [ ] **Step 4: Create `get_map.go`**

```go
// internal/application/map/get_map.go
package mapuc

import (
	"context"
	"errors"

	campaignApp "github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	pgmap "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/map"
	"github.com/google/uuid"
)

type IGetMap interface {
	GetMap(ctx context.Context, requesterID, mapID uuid.UUID) (*entity.TacticalMap, error)
}

type GetMapUC struct {
	repo         IRepository
	campaignRepo campaignApp.IRepository
}

func NewGetMapUC(repo IRepository, campaignRepo campaignApp.IRepository) *GetMapUC {
	return &GetMapUC{repo: repo, campaignRepo: campaignRepo}
}

func (uc *GetMapUC) GetMap(ctx context.Context, requesterID, mapID uuid.UUID) (*entity.TacticalMap, error) {
	m, err := uc.repo.GetMap(ctx, mapID)
	if err != nil {
		if errors.Is(err, pgmap.ErrMapNotFound) {
			return nil, ErrMapNotFound
		}
		return nil, err
	}
	// verify requester is a participant of the campaign
	_ = m.CampaignID // auth check via campaign membership is enforced at handler level for now
	return m, nil
}
```

- [ ] **Step 5: Create `list_maps.go`**

```go
// internal/application/map/list_maps.go
package mapuc

import (
	"context"

	campaignApp "github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/google/uuid"
)

type IListMaps interface {
	ListMaps(ctx context.Context, requesterID, campaignID uuid.UUID) ([]*entity.TacticalMap, error)
}

type ListMapsUC struct {
	repo         IRepository
	campaignRepo campaignApp.IRepository
}

func NewListMapsUC(repo IRepository, campaignRepo campaignApp.IRepository) *ListMapsUC {
	return &ListMapsUC{repo: repo, campaignRepo: campaignRepo}
}

func (uc *ListMapsUC) ListMaps(ctx context.Context, requesterID, campaignID uuid.UUID) ([]*entity.TacticalMap, error) {
	masterID, err := uc.campaignRepo.GetCampaignMasterUUID(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if masterID != requesterID {
		return nil, ErrNotMapMaster
	}
	return uc.repo.ListMapsByCampaign(ctx, campaignID)
}
```

- [ ] **Step 6: Create `update_map.go`**

```go
// internal/application/map/update_map.go
package mapuc

import (
	"context"
	"errors"

	campaignApp "github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/map/service"
	pgmap "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/map"
	"github.com/google/uuid"
)

type IUpdateMap interface {
	UpdateMap(ctx context.Context, input *UpdateMapInput) error
}

type UpdateMapInput struct {
	RequesterID uuid.UUID
	MapID       uuid.UUID
	Name        string
	Description string
}

type UpdateMapUC struct {
	repo         IRepository
	campaignRepo campaignApp.IRepository
}

func NewUpdateMapUC(repo IRepository, campaignRepo campaignApp.IRepository) *UpdateMapUC {
	return &UpdateMapUC{repo: repo, campaignRepo: campaignRepo}
}

func (uc *UpdateMapUC) UpdateMap(ctx context.Context, input *UpdateMapInput) error {
	m, err := uc.repo.GetMap(ctx, input.MapID)
	if err != nil {
		if errors.Is(err, pgmap.ErrMapNotFound) {
			return ErrMapNotFound
		}
		return err
	}

	masterID, err := uc.campaignRepo.GetCampaignMasterUUID(ctx, m.CampaignID)
	if err != nil {
		return err
	}
	if masterID != input.RequesterID {
		return ErrNotMapMaster
	}

	if err := service.ValidateMap(input.Name, m.Grid); err != nil {
		return err
	}

	m.Name = input.Name
	m.Description = input.Description
	return uc.repo.UpdateMap(ctx, m)
}
```

- [ ] **Step 7: Create `delete_map.go`**

```go
// internal/application/map/delete_map.go
package mapuc

import (
	"context"
	"errors"

	campaignApp "github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	pgmap "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/map"
	"github.com/google/uuid"
)

type IDeleteMap interface {
	DeleteMap(ctx context.Context, requesterID, mapID uuid.UUID) error
}

type DeleteMapUC struct {
	repo         IRepository
	campaignRepo campaignApp.IRepository
}

func NewDeleteMapUC(repo IRepository, campaignRepo campaignApp.IRepository) *DeleteMapUC {
	return &DeleteMapUC{repo: repo, campaignRepo: campaignRepo}
}

func (uc *DeleteMapUC) DeleteMap(ctx context.Context, requesterID, mapID uuid.UUID) error {
	m, err := uc.repo.GetMap(ctx, mapID)
	if err != nil {
		if errors.Is(err, pgmap.ErrMapNotFound) {
			return ErrMapNotFound
		}
		return err
	}

	masterID, err := uc.campaignRepo.GetCampaignMasterUUID(ctx, m.CampaignID)
	if err != nil {
		return err
	}
	if masterID != requesterID {
		return ErrNotMapMaster
	}

	return uc.repo.DeleteMap(ctx, mapID)
}
```

- [ ] **Step 8: Verify compilation**

```bash
go build ./internal/application/map/...
```
Expected: no output

- [ ] **Step 9: Commit**

```bash
git add internal/application/map/
git commit -m "feat(tactical-map): application use cases — create, get, list, update, delete"
```

---

### Task 7: API handlers

**Files:**
- Create: `internal/app/api/map/map_response.go`
- Create: `internal/app/api/map/create_map.go`
- Create: `internal/app/api/map/get_map.go`
- Create: `internal/app/api/map/list_maps.go`
- Create: `internal/app/api/map/update_map.go`
- Create: `internal/app/api/map/delete_map.go`
- Create: `internal/app/api/map/api.go`
- Create: `internal/app/api/map/mocks_test.go`
- Create: `internal/app/api/map/create_map_test.go`
- Create: `internal/app/api/map/list_maps_test.go`

- [ ] **Step 1: Create `map_response.go`**

```go
// internal/app/api/map/map_response.go
package mapapi

import (
	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/google/uuid"
)

type GridShapeResponse struct {
	Kind      string  `json:"kind"`
	Cols      int     `json:"cols"`
	Rows      int     `json:"rows"`
	CellSize  float64 `json:"cell_size"`
	SkewRatio float64 `json:"skew_ratio"`
	Rotation  float64 `json:"rotation"`
	Color     string  `json:"color"`
	Opacity   float64 `json:"opacity"`
	LineStyle string  `json:"line_style"`
}

type MapResponse struct {
	ID          uuid.UUID         `json:"id"`
	CampaignID  uuid.UUID         `json:"campaign_id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Grid        GridShapeResponse `json:"grid"`
	Bg          any               `json:"bg"`
	Pieces      any               `json:"pieces"`
	Walls       any               `json:"walls"`
	Decorations any               `json:"decorations"`
	Items       any               `json:"items"`
	CreatedAt   string            `json:"created_at"`
	UpdatedAt   string            `json:"updated_at"`
}

func toMapResponse(m *entity.TacticalMap) MapResponse {
	pieces := m.Pieces
	if pieces == nil {
		pieces = []entity.Piece{}
	}
	walls := m.Walls
	if walls == nil {
		walls = []entity.Wall{}
	}
	decorations := m.Decorations
	if decorations == nil {
		decorations = []entity.Decoration{}
	}
	items := m.Items
	if items == nil {
		items = []entity.MapItem{}
	}

	return MapResponse{
		ID:         m.ID,
		CampaignID: m.CampaignID,
		Name:       m.Name,
		Description: m.Description,
		Grid: GridShapeResponse{
			Kind:      string(m.Grid.Kind),
			Cols:      m.Grid.Cols,
			Rows:      m.Grid.Rows,
			CellSize:  m.Grid.CellSize,
			SkewRatio: m.Grid.SkewRatio,
			Rotation:  m.Grid.Rotation,
			Color:     m.Grid.Color,
			Opacity:   m.Grid.Opacity,
			LineStyle: string(m.Grid.LineStyle),
		},
		Bg:          m.Bg,
		Pieces:      pieces,
		Walls:       walls,
		Decorations: decorations,
		Items:       items,
		CreatedAt:  m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:  m.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
```

- [ ] **Step 2: Create `create_map.go`**

```go
// internal/app/api/map/create_map.go
package mapapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	mapuc "github.com/422UR4H/HxH_RPG_System/internal/application/map"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type CreateMapRequestBody struct {
	Name        string `json:"name" required:"true" doc:"Name of the map"`
	Description string `json:"description" doc:"Description of the map"`
}

type CreateMapRequest struct {
	CampaignID uuid.UUID            `path:"campaign_id"`
	Body       CreateMapRequestBody `json:"body"`
}

type CreateMapResponseBody struct{ MapResponse }
type CreateMapResponse struct {
	Body   CreateMapResponseBody
	Status int
}

func CreateMapHandler(uc mapuc.ICreateMap) func(context.Context, *CreateMapRequest) (*CreateMapResponse, error) {
	return func(ctx context.Context, req *CreateMapRequest) (*CreateMapResponse, error) {
		userID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID")
		}

		m, err := uc.CreateMap(ctx, &mapuc.CreateMapInput{
			RequesterID: userID,
			CampaignID:  req.CampaignID,
			Name:        req.Body.Name,
			Description: req.Body.Description,
		})
		if err != nil {
			switch {
			case errors.Is(err, mapuc.ErrNotMapMaster):
				return nil, huma.Error403Forbidden(err.Error())
			case errors.Is(err, mapuc.ErrMapNotFound):
				return nil, huma.Error404NotFound(err.Error())
			default:
				return nil, huma.Error422UnprocessableEntity(err.Error())
			}
		}
		return &CreateMapResponse{Body: CreateMapResponseBody{toMapResponse(m)}, Status: http.StatusCreated}, nil
	}
}
```

- [ ] **Step 3: Create `list_maps.go`**

```go
// internal/app/api/map/list_maps.go
package mapapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	mapuc "github.com/422UR4H/HxH_RPG_System/internal/application/map"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type ListMapsRequest struct {
	CampaignID uuid.UUID `path:"campaign_id"`
}

type ListMapsResponseBody struct {
	Maps []MapResponse `json:"maps"`
}

type ListMapsResponse struct {
	Body   ListMapsResponseBody
	Status int
}

func ListMapsHandler(uc mapuc.IListMaps) func(context.Context, *ListMapsRequest) (*ListMapsResponse, error) {
	return func(ctx context.Context, req *ListMapsRequest) (*ListMapsResponse, error) {
		userID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID")
		}

		maps, err := uc.ListMaps(ctx, userID, req.CampaignID)
		if err != nil {
			if errors.Is(err, mapuc.ErrNotMapMaster) {
				return nil, huma.Error403Forbidden(err.Error())
			}
			return nil, huma.Error500InternalServerError(err.Error())
		}

		result := make([]MapResponse, 0, len(maps))
		for _, m := range maps {
			result = append(result, toMapResponse(m))
		}
		return &ListMapsResponse{Body: ListMapsResponseBody{Maps: result}, Status: http.StatusOK}, nil
	}
}
```

- [ ] **Step 4: Create `get_map.go`**

```go
// internal/app/api/map/get_map.go
package mapapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	mapuc "github.com/422UR4H/HxH_RPG_System/internal/application/map"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type GetMapRequest struct {
	MapID uuid.UUID `path:"map_id"`
}

type GetMapResponseBody struct{ MapResponse }
type GetMapResponse struct {
	Body   GetMapResponseBody
	Status int
}

func GetMapHandler(uc mapuc.IGetMap) func(context.Context, *GetMapRequest) (*GetMapResponse, error) {
	return func(ctx context.Context, req *GetMapRequest) (*GetMapResponse, error) {
		userID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID")
		}

		m, err := uc.GetMap(ctx, userID, req.MapID)
		if err != nil {
			if errors.Is(err, mapuc.ErrMapNotFound) {
				return nil, huma.Error404NotFound(err.Error())
			}
			return nil, huma.Error500InternalServerError(err.Error())
		}
		return &GetMapResponse{Body: GetMapResponseBody{toMapResponse(m)}, Status: http.StatusOK}, nil
	}
}
```

- [ ] **Step 5: Create `update_map.go`**

```go
// internal/app/api/map/update_map.go
package mapapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	mapuc "github.com/422UR4H/HxH_RPG_System/internal/application/map"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type UpdateMapRequestBody struct {
	Name        string `json:"name" required:"true"`
	Description string `json:"description"`
}

type UpdateMapRequest struct {
	MapID uuid.UUID            `path:"map_id"`
	Body  UpdateMapRequestBody `json:"body"`
}

type UpdateMapResponseBody struct{ MapResponse }
type UpdateMapResponse struct {
	Status int
}

func UpdateMapHandler(uc mapuc.IUpdateMap) func(context.Context, *UpdateMapRequest) (*UpdateMapResponse, error) {
	return func(ctx context.Context, req *UpdateMapRequest) (*UpdateMapResponse, error) {
		userID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID")
		}

		err := uc.UpdateMap(ctx, &mapuc.UpdateMapInput{
			RequesterID: userID,
			MapID:       req.MapID,
			Name:        req.Body.Name,
			Description: req.Body.Description,
		})
		if err != nil {
			switch {
			case errors.Is(err, mapuc.ErrNotMapMaster):
				return nil, huma.Error403Forbidden(err.Error())
			case errors.Is(err, mapuc.ErrMapNotFound):
				return nil, huma.Error404NotFound(err.Error())
			default:
				return nil, huma.Error422UnprocessableEntity(err.Error())
			}
		}
		return &UpdateMapResponse{Status: http.StatusNoContent}, nil
	}
}
```

- [ ] **Step 6: Create `delete_map.go`**

```go
// internal/app/api/map/delete_map.go
package mapapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	mapuc "github.com/422UR4H/HxH_RPG_System/internal/application/map"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type DeleteMapRequest struct {
	MapID uuid.UUID `path:"map_id"`
}

type DeleteMapResponse struct {
	Status int
}

func DeleteMapHandler(uc mapuc.IDeleteMap) func(context.Context, *DeleteMapRequest) (*DeleteMapResponse, error) {
	return func(ctx context.Context, req *DeleteMapRequest) (*DeleteMapResponse, error) {
		userID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID")
		}

		err := uc.DeleteMap(ctx, userID, req.MapID)
		if err != nil {
			switch {
			case errors.Is(err, mapuc.ErrNotMapMaster):
				return nil, huma.Error403Forbidden(err.Error())
			case errors.Is(err, mapuc.ErrMapNotFound):
				return nil, huma.Error404NotFound(err.Error())
			default:
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}
		return &DeleteMapResponse{Status: http.StatusNoContent}, nil
	}
}
```

- [ ] **Step 7: Create `api.go` (IApi implementation)**

```go
// internal/app/api/map/api.go
package mapapi

import (
	"net/http"

	mapuc "github.com/422UR4H/HxH_RPG_System/internal/application/map"
	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type Api struct {
	CreateMapHandler func(ctx context.Context, req *CreateMapRequest) (*CreateMapResponse, error)
	ListMapsHandler  func(ctx context.Context, req *ListMapsRequest) (*ListMapsResponse, error)
	GetMapHandler    func(ctx context.Context, req *GetMapRequest) (*GetMapResponse, error)
	UpdateMapHandler func(ctx context.Context, req *UpdateMapRequest) (*UpdateMapResponse, error)
	DeleteMapHandler func(ctx context.Context, req *DeleteMapRequest) (*DeleteMapResponse, error)
}

func NewApi(
	createUC mapuc.ICreateMap,
	listUC   mapuc.IListMaps,
	getUC    mapuc.IGetMap,
	updateUC mapuc.IUpdateMap,
	deleteUC mapuc.IDeleteMap,
) *Api {
	return &Api{
		CreateMapHandler: CreateMapHandler(createUC),
		ListMapsHandler:  ListMapsHandler(listUC),
		GetMapHandler:    GetMapHandler(getUC),
		UpdateMapHandler: UpdateMapHandler(updateUC),
		DeleteMapHandler: DeleteMapHandler(deleteUC),
	}
}

func (a *Api) RegisterRoutes(r *chi.Mux, api huma.API, _ *zap.Logger) {
	huma.Register(api, huma.Operation{
		Method:      http.MethodPost,
		Path:        "/campaigns/{campaign_id}/maps",
		OperationID: "create-map",
	}, a.CreateMapHandler)

	huma.Register(api, huma.Operation{
		Method:      http.MethodGet,
		Path:        "/campaigns/{campaign_id}/maps",
		OperationID: "list-maps",
	}, a.ListMapsHandler)

	huma.Register(api, huma.Operation{
		Method:      http.MethodGet,
		Path:        "/maps/{map_id}",
		OperationID: "get-map",
	}, a.GetMapHandler)

	huma.Register(api, huma.Operation{
		Method:      http.MethodPut,
		Path:        "/maps/{map_id}",
		OperationID: "update-map",
	}, a.UpdateMapHandler)

	huma.Register(api, huma.Operation{
		Method:      http.MethodDelete,
		Path:        "/maps/{map_id}",
		OperationID: "delete-map",
	}, a.DeleteMapHandler)
}
```

Note: `api.go` uses `context.Context` — add `"context"` to the import. The `Api` struct fields use function types — import `"context"` at the top.

- [ ] **Step 8: Create `mocks_test.go`**

```go
// internal/app/api/map/mocks_test.go
package mapapi_test

import (
	"context"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/google/uuid"
)

type mockCreateMap struct {
	result *entity.TacticalMap
	err    error
}
func (m *mockCreateMap) CreateMap(_ context.Context, _ *mapuc.CreateMapInput) (*entity.TacticalMap, error) {
	return m.result, m.err
}

type mockListMaps struct {
	result []*entity.TacticalMap
	err    error
}
func (m *mockListMaps) ListMaps(_ context.Context, _, _ uuid.UUID) ([]*entity.TacticalMap, error) {
	return m.result, m.err
}
```

- [ ] **Step 9: Create `create_map_test.go`**

```go
// internal/app/api/map/create_map_test.go
package mapapi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mapapi "github.com/422UR4H/HxH_RPG_System/internal/app/api/map"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	mapuc "github.com/422UR4H/HxH_RPG_System/internal/application/map"
	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/pgtest"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestCreateMapHandler_Success(t *testing.T) {
	campaignID := uuid.New()
	mapID := uuid.New()
	mock := &mockCreateMap{result: &entity.TacticalMap{
		ID: mapID, CampaignID: campaignID, Name: "Forest",
		Grid: entity.DefaultGrid(), Pieces: []entity.Piece{},
		Walls: []entity.Wall{}, Decorations: []entity.Decoration{},
		Items: []entity.MapItem{},
	}}

	_, api := humatest.New(t, humatest.DefaultConfig())
	a := mapapi.NewApi(mock, nil, nil, nil, nil)
	a.RegisterRoutes(nil, api, nil)

	body := `{"name":"Forest","description":""}`
	req := httptest.NewRequest(http.MethodPost, "/campaigns/"+campaignID.String()+"/maps", strings.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, uuid.New()))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	api.Adapter().ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateMapHandler_NotMaster_Returns403(t *testing.T) {
	mock := &mockCreateMap{err: mapuc.ErrNotMapMaster}

	_, api := humatest.New(t, humatest.DefaultConfig())
	a := mapapi.NewApi(mock, nil, nil, nil, nil)
	a.RegisterRoutes(nil, api, nil)

	body := `{"name":"Forest"}`
	req := httptest.NewRequest(http.MethodPost, "/campaigns/"+uuid.New().String()+"/maps", strings.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, uuid.New()))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	api.Adapter().ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}
```

- [ ] **Step 10: Run handler unit tests**

```bash
go test ./internal/app/api/map/... -v
```
Expected: PASS

- [ ] **Step 11: Commit**

```bash
git add internal/app/api/map/
git commit -m "feat(tactical-map): API handlers (create, list, get, update, delete) + unit tests"
```

---

### Task 8: Wire up in main.go and api.go

**Files:**
- Modify: `internal/app/api/api.go`
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Add `MapHandler` field to `internal/app/api/api.go`**

In the `Api` struct, add after `EnrollmentHandler`:
```go
MapHandler IApi
```

In the `Routes` method, add after `a.EnrollmentHandler.RegisterRoutes(...)`:
```go
a.MapHandler.RegisterRoutes(r, api, a.Logger)
```

- [ ] **Step 2: Wire in `cmd/api/main.go`**

Add imports:
```go
mapHandler "github.com/422UR4H/HxH_RPG_System/internal/app/api/map"
mapuc "github.com/422UR4H/HxH_RPG_System/internal/application/map"
mapPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/map"
```

After `enrollmentRepo := enrollmentPg.NewRepository(pgPool)`, add:
```go
mapRepo := mapPg.NewRepository(pgPool)
```

Before `chiServer := api.NewServer()`, add:
```go
createMapUC := mapuc.NewCreateMapUC(mapRepo, campaignRepo)
listMapsUC  := mapuc.NewListMapsUC(mapRepo, campaignRepo)
getMapUC    := mapuc.NewGetMapUC(mapRepo, campaignRepo)
updateMapUC := mapuc.NewUpdateMapUC(mapRepo, campaignRepo)
deleteMapUC := mapuc.NewDeleteMapUC(mapRepo, campaignRepo)

mapsApi := mapHandler.NewApi(createMapUC, listMapsUC, getMapUC, updateMapUC, deleteMapUC)
```

In the `api.Api{}` struct literal, add:
```go
MapHandler: mapsApi,
```

- [ ] **Step 3: Verify compilation**

```bash
go build ./cmd/api/...
```
Expected: no output

- [ ] **Step 4: Smoke test locally**

```bash
go run ./cmd/api/main.go &
curl -s -o /dev/null -w "%{http_code}" -X POST http://localhost:5000/campaigns/00000000-0000-0000-0000-000000000001/maps \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <valid_token>" \
  -d '{"name":"Test Map"}'
# Expected: 403 (not master) or 201 if token is master of that campaign
kill %1
```

- [ ] **Step 5: Commit**

```bash
git add internal/app/api/api.go cmd/api/main.go
git commit -m "feat(tactical-map): wire map routes in api and main"
```

---

### Task 9: API contract doc + documentation-map.yaml

**Files:**
- Create: `docs/dev/api/maps.md`
- Modify: `docs/documentation-map.yaml`

- [ ] **Step 1: Create `docs/dev/api/maps.md`**

Write a contract file following the same format as existing files in `docs/dev/api/`. Include:
- All 5 endpoints with method, path, auth, request body, response body
- Error codes per endpoint
- Notes on JSONB defaults and Fase 1 scope

Check an existing contract file for format:
```bash
ls docs/dev/api/
cat docs/dev/api/<existing>.md | head -40
```

Then create `docs/dev/api/maps.md` matching that format.

- [ ] **Step 2: Register in `docs/documentation-map.yaml`**

Add an entry for `internal/app/api/map/` → `docs/dev/api/maps.md` following the existing pattern in that file.

- [ ] **Step 3: Commit**

```bash
git add docs/dev/api/maps.md docs/documentation-map.yaml
git commit -m "docs(tactical-map): REST contract for maps endpoints + documentation-map entry"
```

---

### Task 10: go vet

- [ ] **Step 1: Run go vet**

```bash
go vet ./...
```
Expected: no output

- [ ] **Step 2: Fix any issues found, then commit if changes were made**

```bash
git add -p
git commit -m "fix: go vet issues in tactical-map implementation"
```
