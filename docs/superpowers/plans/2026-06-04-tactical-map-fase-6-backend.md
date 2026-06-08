# Tactical Map Fase 6 — Backend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implementar a tabela `match_maps`, os endpoints REST (POST/GET/DELETE `/matches/:id/map`) e a extensão do WS do lobby com `lobby_piece_moved`.

**Architecture:** Segue o padrão DDD-lite do projeto: entity em `domain/matchmap`, use cases em `application/matchmap`, repository em `gateway/pg/matchmap`, handlers em `app/api/matchmap`. O lobby WS recebe um novo message type em `app/game/message.go` e `app/game/room.go`.

**Tech Stack:** Go, huma/v2, pgx/v5, goose migrations, gorilla/websocket, `github.com/422UR4H/HxH_RPG_System`.

---

## Mapa de arquivos

### Novos
| Arquivo | Responsabilidade |
|---|---|
| `migrations/20260604000000_create_match_maps_table.sql` | Tabela `match_maps` |
| `internal/domain/matchmap/entity/match_map.go` | Entidade de domínio |
| `internal/application/matchmap/errors.go` | Erros do domínio |
| `internal/application/matchmap/i_repository.go` | Interface do repositório |
| `internal/application/matchmap/i_match_repository.go` | Interface mínima de match |
| `internal/application/matchmap/attach_match_map.go` | UC: anexar mapa |
| `internal/application/matchmap/get_match_map.go` | UC: buscar mapa da partida |
| `internal/application/matchmap/detach_match_map.go` | UC: desanexar mapa |
| `internal/gateway/pg/matchmap/repository.go` | Struct Repository |
| `internal/gateway/pg/matchmap/error.go` | Erros de gateway |
| `internal/gateway/pg/matchmap/attach.go` | INSERT/UPSERT no DB |
| `internal/gateway/pg/matchmap/get.go` | SELECT no DB |
| `internal/gateway/pg/matchmap/detach.go` | DELETE no DB |
| `internal/app/api/matchmap/api.go` | Struct Api + RegisterRoutes |
| `internal/app/api/matchmap/response.go` | MatchMapResponse + toResponse |
| `internal/app/api/matchmap/attach.go` | Handler POST |
| `internal/app/api/matchmap/get.go` | Handler GET |
| `internal/app/api/matchmap/detach.go` | Handler DELETE |
| `internal/app/api/matchmap/attach_test.go` | Testes do handler Attach |
| `internal/app/api/matchmap/get_test.go` | Testes do handler Get |
| `internal/app/api/matchmap/detach_test.go` | Testes do handler Detach |
| `docs/dev/api/match-maps.md` | Contrato REST + WS |

### Modificados
| Arquivo | O que muda |
|---|---|
| `internal/app/api/api.go` | + `MatchMapHandler IApi` |
| `cmd/api/main.go` | + matchmapRepo, UCs, Api |
| `internal/app/game/message.go` | + `MsgTypeLobbyPieceMoved` + `LobbyPieceMovedPayload` |
| `internal/app/game/room.go` | + case `MsgTypeLobbyPieceMoved` em `handleClientMessage` |
| `docs/documentation-map.yaml` | + entrada match-maps |

---

## Task 1: Migração `match_maps`

**Files:**
- Create: `migrations/20260604000000_create_match_maps_table.sql`

- [ ] **Step 1: Criar o arquivo de migração**

```sql
-- migrations/20260604000000_create_match_maps_table.sql
-- +goose Up
-- +goose StatementBegin
BEGIN;

CREATE TABLE IF NOT EXISTS match_maps (
  match_uuid  UUID         PRIMARY KEY REFERENCES matches(uuid) ON DELETE CASCADE,
  map_uuid    UUID         NOT NULL    REFERENCES maps(uuid)    ON DELETE RESTRICT,
  attached_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

DROP TABLE IF EXISTS match_maps;

COMMIT;
-- +goose StatementEnd
```

- [ ] **Step 2: Aplicar a migração localmente (requer banco rodando)**

```bash
cd /path/to/System_X_System
goose -dir migrations postgres "$DATABASE_URL" up
```

Esperado: `OK   20260604000000_create_match_maps_table.sql`

- [ ] **Step 3: Commit**

```bash
git add migrations/20260604000000_create_match_maps_table.sql
git commit -m "feat(db): add match_maps table migration"
```

---

## Task 2: Entidade + erros + interfaces

**Files:**
- Create: `internal/domain/matchmap/entity/match_map.go`
- Create: `internal/application/matchmap/errors.go`
- Create: `internal/application/matchmap/i_repository.go`
- Create: `internal/application/matchmap/i_match_repository.go`

- [ ] **Step 1: Criar entidade de domínio**

```go
// internal/domain/matchmap/entity/match_map.go
package entity

import "time"

type MatchMap struct {
	MatchUUID  string
	MapUUID    string
	AttachedAt time.Time
}
```

- [ ] **Step 2: Criar erros**

```go
// internal/application/matchmap/errors.go
package matchmapuc

import "errors"

var (
	ErrMatchMapNotFound    = errors.New("match map not found")
	ErrMatchAlreadyStarted = errors.New("cannot change map after match has started")
	ErrNotMatchMaster      = errors.New("only the match master can perform this action")
	ErrMapNotFound         = errors.New("map not found")
	ErrMatchNotFound       = errors.New("match not found")
)
```

- [ ] **Step 3: Criar interface do repositório matchmap**

```go
// internal/application/matchmap/i_repository.go
package matchmapuc

import (
	"context"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/matchmap/entity"
	"github.com/google/uuid"
)

type IRepository interface {
	AttachMap(ctx context.Context, matchUUID, mapUUID uuid.UUID) (*entity.MatchMap, error)
	GetMatchMap(ctx context.Context, matchUUID uuid.UUID) (*entity.MatchMap, error)
	DetachMap(ctx context.Context, matchUUID uuid.UUID) error
}
```

- [ ] **Step 4: Criar interface mínima de match (para verificar master e game_start_at)**

```go
// internal/application/matchmap/i_match_repository.go
package matchmapuc

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// MatchInfo holds the minimal match data needed by matchmap use cases.
type MatchInfo struct {
	MasterUUID  uuid.UUID
	GameStartAt *time.Time
}

type IMatchRepository interface {
	GetMatchInfo(ctx context.Context, matchUUID uuid.UUID) (*MatchInfo, error)
}
```

- [ ] **Step 5: Commit**

```bash
git add internal/domain/matchmap/ internal/application/matchmap/
git commit -m "feat(matchmap): add entity, errors, and repository interfaces"
```

---

## Task 3: Gateway — repository pg

**Files:**
- Create: `internal/gateway/pg/matchmap/error.go`
- Create: `internal/gateway/pg/matchmap/repository.go`
- Create: `internal/gateway/pg/matchmap/attach.go`
- Create: `internal/gateway/pg/matchmap/get.go`
- Create: `internal/gateway/pg/matchmap/detach.go`

- [ ] **Step 1: Criar erro de gateway**

```go
// internal/gateway/pg/matchmap/error.go
package pgmatchmap

import "errors"

var ErrMatchMapNotFound = errors.New("match map not found")
```

- [ ] **Step 2: Criar struct Repository**

```go
// internal/gateway/pg/matchmap/repository.go
package pgmatchmap

import pgfs "github.com/422UR4H/HxH_RPG_System/pkg"

type Repository struct {
	q pgfs.IQuerier
}

func NewRepository(q pgfs.IQuerier) *Repository {
	return &Repository{q: q}
}
```

- [ ] **Step 3: Implementar AttachMap (UPSERT)**

```go
// internal/gateway/pg/matchmap/attach.go
package pgmatchmap

import (
	"context"
	"fmt"
	"time"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/matchmap/entity"
	"github.com/google/uuid"
)

func (r *Repository) AttachMap(ctx context.Context, matchUUID, mapUUID uuid.UUID) (*entity.MatchMap, error) {
	const query = `
		INSERT INTO match_maps (match_uuid, map_uuid, attached_at)
		VALUES ($1, $2, now())
		ON CONFLICT (match_uuid) DO UPDATE
			SET map_uuid = EXCLUDED.map_uuid,
			    attached_at = now()
		RETURNING attached_at
	`
	var attachedAt time.Time
	err := r.q.QueryRow(ctx, query, matchUUID, mapUUID).Scan(&attachedAt)
	if err != nil {
		return nil, fmt.Errorf("attach match map: %w", err)
	}
	return &entity.MatchMap{
		MatchUUID:  matchUUID.String(),
		MapUUID:    mapUUID.String(),
		AttachedAt: attachedAt,
	}, nil
}
```

- [ ] **Step 4: Implementar GetMatchMap**

```go
// internal/gateway/pg/matchmap/get.go
package pgmatchmap

import (
	"context"
	"errors"
	"fmt"
	"time"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/matchmap/entity"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetMatchMap(ctx context.Context, matchUUID uuid.UUID) (*entity.MatchMap, error) {
	const query = `
		SELECT map_uuid, attached_at
		FROM match_maps
		WHERE match_uuid = $1
	`
	var mapUUIDStr uuid.UUID
	var attachedAt time.Time
	err := r.q.QueryRow(ctx, query, matchUUID).Scan(&mapUUIDStr, &attachedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMatchMapNotFound
		}
		return nil, fmt.Errorf("get match map: %w", err)
	}
	return &entity.MatchMap{
		MatchUUID:  matchUUID.String(),
		MapUUID:    mapUUIDStr.String(),
		AttachedAt: attachedAt,
	}, nil
}
```

- [ ] **Step 5: Implementar DetachMap**

```go
// internal/gateway/pg/matchmap/detach.go
package pgmatchmap

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) DetachMap(ctx context.Context, matchUUID uuid.UUID) error {
	const query = `DELETE FROM match_maps WHERE match_uuid = $1`
	tag, err := r.q.Exec(ctx, query, matchUUID)
	if err != nil {
		return fmt.Errorf("detach match map: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrMatchMapNotFound
	}
	return nil
}
```

- [ ] **Step 6: Implementar GetMatchInfo (para o IMatchRepository)**

Adicionar no arquivo `get.go` do gateway **match** existente (`internal/gateway/pg/match/read_match.go`), ou criar um arquivo separado no pacote matchmap. Como melhor prática, adicionar o método `GetMatchInfo` ao `Repository` do pacote `match` e declarar a satisfação da interface no `cmd/api/main.go`.

Criar `internal/gateway/pg/match/get_match_info.go`:

```go
// internal/gateway/pg/match/get_match_info.go
package match

import (
	"context"
	"errors"
	"fmt"
	"time"

	matchmapuc "github.com/422UR4H/HxH_RPG_System/internal/application/matchmap"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetMatchInfo(ctx context.Context, matchUUID uuid.UUID) (*matchmapuc.MatchInfo, error) {
	const query = `
		SELECT master_uuid, game_start_at
		FROM matches
		WHERE uuid = $1
	`
	var masterUUID uuid.UUID
	var gameStartAt *time.Time
	err := r.q.QueryRow(ctx, query, matchUUID).Scan(&masterUUID, &gameStartAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMatchNotFound
		}
		return nil, fmt.Errorf("get match info: %w", err)
	}
	return &matchmapuc.MatchInfo{
		MasterUUID:  masterUUID,
		GameStartAt: gameStartAt,
	}, nil
}
```

- [ ] **Step 7: Commit**

```bash
git add internal/gateway/pg/matchmap/ internal/gateway/pg/match/get_match_info.go
git commit -m "feat(matchmap): add pg repository (attach, get, detach) and GetMatchInfo"
```

---

## Task 4: Use cases

**Files:**
- Create: `internal/application/matchmap/attach_match_map.go`
- Create: `internal/application/matchmap/get_match_map.go`
- Create: `internal/application/matchmap/detach_match_map.go`

- [ ] **Step 1: UC AttachMatchMap**

```go
// internal/application/matchmap/attach_match_map.go
package matchmapuc

import (
	"context"
	"errors"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/matchmap/entity"
	pgmatchmap "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/matchmap"
	pgmatch "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

type IAttachMatchMap interface {
	Attach(ctx context.Context, input *AttachMatchMapInput) (*entity.MatchMap, error)
}

type AttachMatchMapInput struct {
	RequesterUUID uuid.UUID
	MatchUUID     uuid.UUID
	MapUUID       uuid.UUID
}

type AttachMatchMapUC struct {
	repo      IRepository
	matchRepo IMatchRepository
}

func NewAttachMatchMapUC(repo IRepository, matchRepo IMatchRepository) *AttachMatchMapUC {
	return &AttachMatchMapUC{repo: repo, matchRepo: matchRepo}
}

func (uc *AttachMatchMapUC) Attach(ctx context.Context, input *AttachMatchMapInput) (*entity.MatchMap, error) {
	info, err := uc.matchRepo.GetMatchInfo(ctx, input.MatchUUID)
	if err != nil {
		if errors.Is(err, pgmatch.ErrMatchNotFound) {
			return nil, ErrMatchNotFound
		}
		return nil, err
	}
	if info.MasterUUID != input.RequesterUUID {
		return nil, ErrNotMatchMaster
	}
	if info.GameStartAt != nil {
		return nil, ErrMatchAlreadyStarted
	}

	mm, err := uc.repo.AttachMap(ctx, input.MatchUUID, input.MapUUID)
	if err != nil {
		// map not found is surfaced as a FK violation from the DB
		if errors.Is(err, pgmatchmap.ErrMatchMapNotFound) {
			return nil, ErrMapNotFound
		}
		return nil, err
	}
	return mm, nil
}
```

- [ ] **Step 2: UC GetMatchMap**

```go
// internal/application/matchmap/get_match_map.go
package matchmapuc

import (
	"context"
	"errors"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/matchmap/entity"
	pgmatchmap "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/matchmap"
	"github.com/google/uuid"
)

type IGetMatchMap interface {
	Get(ctx context.Context, matchUUID uuid.UUID) (*entity.MatchMap, error)
}

type GetMatchMapUC struct {
	repo IRepository
}

func NewGetMatchMapUC(repo IRepository) *GetMatchMapUC {
	return &GetMatchMapUC{repo: repo}
}

// Get returns nil if no map is attached (not an error).
func (uc *GetMatchMapUC) Get(ctx context.Context, matchUUID uuid.UUID) (*entity.MatchMap, error) {
	mm, err := uc.repo.GetMatchMap(ctx, matchUUID)
	if err != nil {
		if errors.Is(err, pgmatchmap.ErrMatchMapNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return mm, nil
}
```

- [ ] **Step 3: UC DetachMatchMap**

```go
// internal/application/matchmap/detach_match_map.go
package matchmapuc

import (
	"context"
	"errors"

	pgmatchmap "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/matchmap"
	pgmatch "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

type IDetachMatchMap interface {
	Detach(ctx context.Context, input *DetachMatchMapInput) error
}

type DetachMatchMapInput struct {
	RequesterUUID uuid.UUID
	MatchUUID     uuid.UUID
}

type DetachMatchMapUC struct {
	repo      IRepository
	matchRepo IMatchRepository
}

func NewDetachMatchMapUC(repo IRepository, matchRepo IMatchRepository) *DetachMatchMapUC {
	return &DetachMatchMapUC{repo: repo, matchRepo: matchRepo}
}

func (uc *DetachMatchMapUC) Detach(ctx context.Context, input *DetachMatchMapInput) error {
	info, err := uc.matchRepo.GetMatchInfo(ctx, input.MatchUUID)
	if err != nil {
		if errors.Is(err, pgmatch.ErrMatchNotFound) {
			return ErrMatchNotFound
		}
		return err
	}
	if info.MasterUUID != input.RequesterUUID {
		return ErrNotMatchMaster
	}
	if info.GameStartAt != nil {
		return ErrMatchAlreadyStarted
	}

	if err := uc.repo.DetachMap(ctx, input.MatchUUID); err != nil {
		if errors.Is(err, pgmatchmap.ErrMatchMapNotFound) {
			return ErrMatchMapNotFound
		}
		return err
	}
	return nil
}
```

- [ ] **Step 4: Compilar para verificar erros de tipos**

```bash
cd /path/to/System_X_System
go build ./internal/application/matchmap/...
```

Esperado: sem erros.

- [ ] **Step 5: Commit**

```bash
git add internal/application/matchmap/
git commit -m "feat(matchmap): add attach, get, detach use cases"
```

---

## Task 5: API handlers

**Files:**
- Create: `internal/app/api/matchmap/api.go`
- Create: `internal/app/api/matchmap/response.go`
- Create: `internal/app/api/matchmap/attach.go`
- Create: `internal/app/api/matchmap/get.go`
- Create: `internal/app/api/matchmap/detach.go`

- [ ] **Step 1: Criar Api struct + RegisterRoutes**

```go
// internal/app/api/matchmap/api.go
package matchmapapi

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type Handler[I, O any] func(context.Context, *I) (*O, error)

type Api struct {
	AttachMatchMapHandler Handler[AttachMatchMapRequest, AttachMatchMapResponse]
	GetMatchMapHandler    Handler[GetMatchMapRequest, GetMatchMapResponse]
	DetachMatchMapHandler Handler[DetachMatchMapRequest, DetachMatchMapResponse]
}

func (a *Api) RegisterRoutes(_ *chi.Mux, api huma.API, _ *zap.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "attach-match-map",
		Method:      http.MethodPost,
		Path:        "/matches/{match_uuid}/map",
		Description: "Attach a tactical map to a match (master only, before game starts)",
		Tags:        []string{"match-maps"},
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusUnprocessableEntity,
			http.StatusInternalServerError,
		},
	}, a.AttachMatchMapHandler)

	huma.Register(api, huma.Operation{
		OperationID: "get-match-map",
		Method:      http.MethodGet,
		Path:        "/matches/{match_uuid}/map",
		Description: "Get the map attached to a match (204 if none)",
		Tags:        []string{"match-maps"},
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusInternalServerError,
		},
	}, a.GetMatchMapHandler)

	huma.Register(api, huma.Operation{
		OperationID: "detach-match-map",
		Method:      http.MethodDelete,
		Path:        "/matches/{match_uuid}/map",
		Description: "Detach the map from a match (master only, before game starts)",
		Tags:        []string{"match-maps"},
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusUnprocessableEntity,
			http.StatusInternalServerError,
		},
		DefaultStatus: http.StatusNoContent,
	}, a.DetachMatchMapHandler)
}
```

- [ ] **Step 2: Criar response types**

```go
// internal/app/api/matchmap/response.go
package matchmapapi

import (
	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/matchmap/entity"
)

type MatchMapResponse struct {
	MatchUUID  string `json:"match_uuid"`
	MapUUID    string `json:"map_uuid"`
	AttachedAt string `json:"attached_at"`
}

func toMatchMapResponse(mm *entity.MatchMap) MatchMapResponse {
	return MatchMapResponse{
		MatchUUID:  mm.MatchUUID,
		MapUUID:    mm.MapUUID,
		AttachedAt: mm.AttachedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
```

- [ ] **Step 3: Handler Attach**

```go
// internal/app/api/matchmap/attach.go
package matchmapapi

import (
	"context"
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	matchmapuc "github.com/422UR4H/HxH_RPG_System/internal/application/matchmap"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type AttachMatchMapRequestBody struct {
	MapUUID uuid.UUID `json:"map_uuid" required:"true" doc:"UUID of the map to attach"`
}

type AttachMatchMapRequest struct {
	MatchUUID uuid.UUID                 `path:"match_uuid"`
	Body      AttachMatchMapRequestBody `json:"body"`
}

type AttachMatchMapResponseBody struct {
	MatchMap MatchMapResponse `json:"match_map"`
}

type AttachMatchMapResponse struct {
	Body AttachMatchMapResponseBody
}

func AttachMatchMapHandler(uc matchmapuc.IAttachMatchMap) func(context.Context, *AttachMatchMapRequest) (*AttachMatchMapResponse, error) {
	return func(ctx context.Context, req *AttachMatchMapRequest) (*AttachMatchMapResponse, error) {
		userID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID")
		}

		mm, err := uc.Attach(ctx, &matchmapuc.AttachMatchMapInput{
			RequesterUUID: userID,
			MatchUUID:     req.MatchUUID,
			MapUUID:       req.Body.MapUUID,
		})
		if err != nil {
			switch {
			case errors.Is(err, matchmapuc.ErrNotMatchMaster):
				return nil, huma.Error403Forbidden(err.Error())
			case errors.Is(err, matchmapuc.ErrMatchNotFound),
				errors.Is(err, matchmapuc.ErrMapNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, matchmapuc.ErrMatchAlreadyStarted):
				return nil, huma.Error422UnprocessableEntity(err.Error())
			default:
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}
		return &AttachMatchMapResponse{Body: AttachMatchMapResponseBody{MatchMap: toMatchMapResponse(mm)}}, nil
	}
}
```

- [ ] **Step 4: Handler Get**

```go
// internal/app/api/matchmap/get.go
package matchmapapi

import (
	"context"
	"net/http"

	matchmapuc "github.com/422UR4H/HxH_RPG_System/internal/application/matchmap"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type GetMatchMapRequest struct {
	MatchUUID uuid.UUID `path:"match_uuid"`
}

type GetMatchMapResponseBody struct {
	MatchMap MatchMapResponse `json:"match_map"`
}

type GetMatchMapResponse struct {
	Body   *GetMatchMapResponseBody
	Status int
}

func GetMatchMapHandler(uc matchmapuc.IGetMatchMap) func(context.Context, *GetMatchMapRequest) (*GetMatchMapResponse, error) {
	return func(ctx context.Context, req *GetMatchMapRequest) (*GetMatchMapResponse, error) {
		mm, err := uc.Get(ctx, req.MatchUUID)
		if err != nil {
			return nil, huma.Error500InternalServerError(err.Error())
		}
		if mm == nil {
			// No map attached — return 204 No Content
			return &GetMatchMapResponse{Status: http.StatusNoContent}, nil
		}
		return &GetMatchMapResponse{
			Body:   &GetMatchMapResponseBody{MatchMap: toMatchMapResponse(mm)},
			Status: http.StatusOK,
		}, nil
	}
}
```

- [ ] **Step 5: Handler Detach**

```go
// internal/app/api/matchmap/detach.go
package matchmapapi

import (
	"context"
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	matchmapuc "github.com/422UR4H/HxH_RPG_System/internal/application/matchmap"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type DetachMatchMapRequest struct {
	MatchUUID uuid.UUID `path:"match_uuid"`
}

type DetachMatchMapResponse struct{}

func DetachMatchMapHandler(uc matchmapuc.IDetachMatchMap) func(context.Context, *DetachMatchMapRequest) (*DetachMatchMapResponse, error) {
	return func(ctx context.Context, req *DetachMatchMapRequest) (*DetachMatchMapResponse, error) {
		userID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID")
		}

		err := uc.Detach(ctx, &matchmapuc.DetachMatchMapInput{
			RequesterUUID: userID,
			MatchUUID:     req.MatchUUID,
		})
		if err != nil {
			switch {
			case errors.Is(err, matchmapuc.ErrNotMatchMaster):
				return nil, huma.Error403Forbidden(err.Error())
			case errors.Is(err, matchmapuc.ErrMatchNotFound),
				errors.Is(err, matchmapuc.ErrMatchMapNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, matchmapuc.ErrMatchAlreadyStarted):
				return nil, huma.Error422UnprocessableEntity(err.Error())
			default:
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}
		return &DetachMatchMapResponse{}, nil
	}
}
```

- [ ] **Step 6: Compilar handlers**

```bash
go build ./internal/app/api/matchmap/...
```

Esperado: sem erros.

- [ ] **Step 7: Commit**

```bash
git add internal/app/api/matchmap/
git commit -m "feat(matchmap): add API handlers (attach, get, detach)"
```

---

## Task 6: Testes dos handlers

**Files:**
- Create: `internal/app/api/matchmap/attach_test.go`
- Create: `internal/app/api/matchmap/get_test.go`
- Create: `internal/app/api/matchmap/detach_test.go`

- [ ] **Step 1: Criar arquivo de testes Attach**

```go
// internal/app/api/matchmap/attach_test.go
package matchmapapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	matchmapapi "github.com/422UR4H/HxH_RPG_System/internal/app/api/matchmap"
	matchmapuc "github.com/422UR4H/HxH_RPG_System/internal/application/matchmap"
	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/matchmap/entity"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

type mockAttach struct {
	result *entity.MatchMap
	err    error
}

func (m *mockAttach) Attach(_ context.Context, _ *matchmapuc.AttachMatchMapInput) (*entity.MatchMap, error) {
	return m.result, m.err
}

func newTestMatchMap(matchUUID, mapUUID uuid.UUID) *entity.MatchMap {
	return &entity.MatchMap{
		MatchUUID:  matchUUID.String(),
		MapUUID:    mapUUID.String(),
		AttachedAt: time.Now().UTC(),
	}
}

func TestAttachMatchMapHandler_Success(t *testing.T) {
	userID := uuid.New()
	matchUUID := uuid.New()
	mapUUID := uuid.New()

	_, api := humatest.New(t)
	mock := &mockAttach{result: newTestMatchMap(matchUUID, mapUUID)}
	huma.Register(api, huma.Operation{Method: http.MethodPost, Path: "/matches/{match_uuid}/map"}, matchmapapi.AttachMatchMapHandler(mock))

	ctx := context.WithValue(context.Background(), auth.UserIDKey, userID)
	resp := api.PostCtx(ctx, "/matches/"+matchUUID.String()+"/map", map[string]any{"map_uuid": mapUUID.String()})

	if resp.Code != http.StatusOK {
		t.Errorf("got %d, want 200. body: %s", resp.Code, resp.Body.String())
	}
	var result map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	mm, ok := result["match_map"].(map[string]any)
	if !ok {
		t.Fatal("response missing 'match_map'")
	}
	if mm["map_uuid"] != mapUUID.String() {
		t.Errorf("got map_uuid %v, want %s", mm["map_uuid"], mapUUID.String())
	}
}

func TestAttachMatchMapHandler_NotMaster_Returns403(t *testing.T) {
	userID := uuid.New()
	matchUUID := uuid.New()

	_, api := humatest.New(t)
	mock := &mockAttach{err: matchmapuc.ErrNotMatchMaster}
	huma.Register(api, huma.Operation{Method: http.MethodPost, Path: "/matches/{match_uuid}/map"}, matchmapapi.AttachMatchMapHandler(mock))

	ctx := context.WithValue(context.Background(), auth.UserIDKey, userID)
	resp := api.PostCtx(ctx, "/matches/"+matchUUID.String()+"/map", map[string]any{"map_uuid": uuid.New().String()})

	if resp.Code != http.StatusForbidden {
		t.Errorf("got %d, want 403", resp.Code)
	}
}

func TestAttachMatchMapHandler_AlreadyStarted_Returns422(t *testing.T) {
	userID := uuid.New()
	matchUUID := uuid.New()

	_, api := humatest.New(t)
	mock := &mockAttach{err: matchmapuc.ErrMatchAlreadyStarted}
	huma.Register(api, huma.Operation{Method: http.MethodPost, Path: "/matches/{match_uuid}/map"}, matchmapapi.AttachMatchMapHandler(mock))

	ctx := context.WithValue(context.Background(), auth.UserIDKey, userID)
	resp := api.PostCtx(ctx, "/matches/"+matchUUID.String()+"/map", map[string]any{"map_uuid": uuid.New().String()})

	if resp.Code != http.StatusUnprocessableEntity {
		t.Errorf("got %d, want 422", resp.Code)
	}
}
```

- [ ] **Step 2: Criar arquivo de testes Get**

```go
// internal/app/api/matchmap/get_test.go
package matchmapapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	matchmapapi "github.com/422UR4H/HxH_RPG_System/internal/app/api/matchmap"
	matchmapuc "github.com/422UR4H/HxH_RPG_System/internal/application/matchmap"
	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/matchmap/entity"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

type mockGet struct {
	result *entity.MatchMap
	err    error
}

func (m *mockGet) Get(_ context.Context, _ uuid.UUID) (*entity.MatchMap, error) {
	return m.result, m.err
}

func TestGetMatchMapHandler_WithMap_Returns200(t *testing.T) {
	matchUUID := uuid.New()
	mapUUID := uuid.New()

	_, api := humatest.New(t)
	mock := &mockGet{result: newTestMatchMap(matchUUID, mapUUID)}
	huma.Register(api, huma.Operation{Method: http.MethodGet, Path: "/matches/{match_uuid}/map"}, matchmapapi.GetMatchMapHandler(mock))

	resp := api.Get("/matches/" + matchUUID.String() + "/map")
	if resp.Code != http.StatusOK {
		t.Errorf("got %d, want 200. body: %s", resp.Code, resp.Body.String())
	}
	var result map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["match_map"] == nil {
		t.Fatal("response missing 'match_map'")
	}
}

func TestGetMatchMapHandler_NoMap_Returns204(t *testing.T) {
	matchUUID := uuid.New()

	_, api := humatest.New(t)
	mock := &mockGet{result: nil, err: nil}
	huma.Register(api, huma.Operation{Method: http.MethodGet, Path: "/matches/{match_uuid}/map"}, matchmapapi.GetMatchMapHandler(mock))

	resp := api.Get("/matches/" + matchUUID.String() + "/map")
	if resp.Code != http.StatusNoContent {
		t.Errorf("got %d, want 204", resp.Code)
	}
}

func TestGetMatchMapHandler_Error_Returns500(t *testing.T) {
	matchUUID := uuid.New()

	_, api := humatest.New(t)
	mock := &mockGet{err: matchmapuc.ErrMatchMapNotFound}
	// ErrMatchMapNotFound returned from Get UC means "not found" — UC maps it to nil
	// So return a genuine internal error here
	mock.err = nil
	mock.result = nil // simulates "no map attached" → 204
	_ = mock
	mock2 := &mockGet{err: errInternal}
	huma.Register(api, huma.Operation{Method: http.MethodGet, Path: "/matches/{match_uuid}/map"}, matchmapapi.GetMatchMapHandler(mock2))

	resp := api.Get("/matches/" + matchUUID.String() + "/map")
	if resp.Code != http.StatusInternalServerError {
		t.Errorf("got %d, want 500", resp.Code)
	}
}

var errInternal = matchmapuc.ErrMatchNotFound // any non-nil error exercises 500 path
```

- [ ] **Step 3: Criar arquivo de testes Detach**

```go
// internal/app/api/matchmap/detach_test.go
package matchmapapi_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	matchmapapi "github.com/422UR4H/HxH_RPG_System/internal/app/api/matchmap"
	matchmapuc "github.com/422UR4H/HxH_RPG_System/internal/application/matchmap"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

type mockDetach struct {
	err error
}

func (m *mockDetach) Detach(_ context.Context, _ *matchmapuc.DetachMatchMapInput) error {
	return m.err
}

func TestDetachMatchMapHandler_Success(t *testing.T) {
	userID := uuid.New()
	matchUUID := uuid.New()

	_, api := humatest.New(t)
	mock := &mockDetach{}
	huma.Register(api, huma.Operation{
		Method:        http.MethodDelete,
		Path:          "/matches/{match_uuid}/map",
		DefaultStatus: http.StatusNoContent,
	}, matchmapapi.DetachMatchMapHandler(mock))

	ctx := context.WithValue(context.Background(), auth.UserIDKey, userID)
	resp := api.DeleteCtx(ctx, "/matches/"+matchUUID.String()+"/map")
	if resp.Code != http.StatusNoContent {
		t.Errorf("got %d, want 204. body: %s", resp.Code, resp.Body.String())
	}
}

func TestDetachMatchMapHandler_NotMaster_Returns403(t *testing.T) {
	userID := uuid.New()
	matchUUID := uuid.New()

	_, api := humatest.New(t)
	mock := &mockDetach{err: matchmapuc.ErrNotMatchMaster}
	huma.Register(api, huma.Operation{
		Method:        http.MethodDelete,
		Path:          "/matches/{match_uuid}/map",
		DefaultStatus: http.StatusNoContent,
	}, matchmapapi.DetachMatchMapHandler(mock))

	ctx := context.WithValue(context.Background(), auth.UserIDKey, userID)
	resp := api.DeleteCtx(ctx, "/matches/"+matchUUID.String()+"/map")
	if resp.Code != http.StatusForbidden {
		t.Errorf("got %d, want 403", resp.Code)
	}
}

func TestDetachMatchMapHandler_AlreadyStarted_Returns422(t *testing.T) {
	userID := uuid.New()
	matchUUID := uuid.New()

	_, api := humatest.New(t)
	mock := &mockDetach{err: matchmapuc.ErrMatchAlreadyStarted}
	huma.Register(api, huma.Operation{
		Method:        http.MethodDelete,
		Path:          "/matches/{match_uuid}/map",
		DefaultStatus: http.StatusNoContent,
	}, matchmapapi.DetachMatchMapHandler(mock))

	ctx := context.WithValue(context.Background(), auth.UserIDKey, userID)
	resp := api.DeleteCtx(ctx, "/matches/"+matchUUID.String()+"/map")
	if resp.Code != http.StatusUnprocessableEntity {
		t.Errorf("got %d, want 422", resp.Code)
	}
}
```

- [ ] **Step 4: Rodar os testes**

```bash
go test ./internal/app/api/matchmap/... -v
```

Esperado: todos passam.

- [ ] **Step 5: Commit**

```bash
git add internal/app/api/matchmap/*_test.go
git commit -m "test(matchmap): add handler unit tests for attach, get, detach"
```

---

## Task 7: Wiring — `api.go` e `cmd/api/main.go`

**Files:**
- Modify: `internal/app/api/api.go`
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Adicionar `MatchMapHandler` à struct `Api` em `internal/app/api/api.go`**

No arquivo `internal/app/api/api.go`, adicionar campo na struct `Api`:

```go
// Após MapHandler IApi:
MatchMapHandler IApi
```

E em `Routes(...)`, registrar o handler após `MapHandler`:

```go
a.MatchMapHandler.RegisterRoutes(r, api, a.Logger)
```

- [ ] **Step 2: Adicionar matchmap ao `cmd/api/main.go`**

Adicionar o import do pacote matchmap:

```go
matchmapHandler "github.com/422UR4H/HxH_RPG_System/internal/app/api/matchmap"
matchmapuc "github.com/422UR4H/HxH_RPG_System/internal/application/matchmap"
matchmapPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/matchmap"
```

Dentro de `main()`, após a criação de `mapRepo`:

```go
matchmapRepo := matchmapPg.NewRepository(pgPool)
attachMatchMapUC := matchmapuc.NewAttachMatchMapUC(matchmapRepo, matchRepo)
getMatchMapUC := matchmapuc.NewGetMatchMapUC(matchmapRepo)
detachMatchMapUC := matchmapuc.NewDetachMatchMapUC(matchmapRepo, matchRepo)

matchMapsApi := matchmapHandler.Api{
    AttachMatchMapHandler: matchmapHandler.AttachMatchMapHandler(attachMatchMapUC),
    GetMatchMapHandler:    matchmapHandler.GetMatchMapHandler(getMatchMapUC),
    DetachMatchMapHandler: matchmapHandler.DetachMatchMapHandler(detachMatchMapUC),
}
```

E adicionar à struct `api.Api`:

```go
MatchMapHandler: &matchMapsApi,
```

- [ ] **Step 3: Compilar o servidor inteiro**

```bash
go build ./cmd/api/...
```

Esperado: sem erros de compilação.

- [ ] **Step 4: Commit**

```bash
git add internal/app/api/api.go cmd/api/main.go
git commit -m "feat(matchmap): wire match_maps API into server"
```

---

## Task 8: WS lobby — `lobby_piece_moved`

**Files:**
- Modify: `internal/app/game/message.go`
- Modify: `internal/app/game/room.go`

- [ ] **Step 1: Adicionar message type e payload em `message.go`**

Após a constante `MsgTypeCancelLobby`, adicionar:

```go
// Client → Server (lobby map sync)
// Prefixo lobby_ distingue de eventos in-game futuros (Fase 7).
MsgTypeLobbyPieceMoved MessageType = "lobby_piece_moved"
```

Após o type `PlayerKickedPayload`, adicionar:

```go
// SlotPayload representa uma coordenada de slot (square ou hex).
type SlotPayload struct {
	Kind string `json:"kind"`             // "square" | "hex"
	Col  *int   `json:"col,omitempty"`    // square only
	Row  *int   `json:"row,omitempty"`    // square only
	Q    *int   `json:"q,omitempty"`      // hex only
	R    *int   `json:"r,omitempty"`      // hex only
}

type LobbyPieceMovedPayload struct {
	PieceID string      `json:"piece_id"`
	Slot    SlotPayload `json:"slot"`
}
```

- [ ] **Step 2: Adicionar case em `room.handleClientMessage`**

Em `room.go`, dentro do `switch incoming.Type {`, adicionar após `case MsgTypeCancelLobby:`:

```go
case MsgTypeLobbyPieceMoved:
    // Broadcast piece move to all OTHER participants in the lobby.
    // No server-side piece ownership validation in Phase 6 — client restricts
    // drag to allowed pieces. TODO: validate piece ownership per user (Phase 7+)
    var payload LobbyPieceMovedPayload
    if err := json.Unmarshal(incoming.Payload, &payload); err != nil {
        client.SendMessage(NewErrorMessage("invalid_payload", "invalid lobby_piece_moved payload"))
        return
    }
    outMsg := NewClientMessage(MsgTypeLobbyPieceMoved, client.userUUID, payload)
    data, _ := json.Marshal(outMsg)
    // Broadcast to all except sender.
    r.mu.RLock()
    for id, c := range r.clients {
        if id == client.userUUID {
            continue
        }
        select {
        case c.send <- data:
        default:
            log.Printf("dropping lobby_piece_moved for slow client %s", id)
        }
    }
    r.mu.RUnlock()
```

- [ ] **Step 3: Compilar o game server**

```bash
go build ./cmd/game/...
```

Esperado: sem erros.

- [ ] **Step 4: Commit**

```bash
git add internal/app/game/message.go internal/app/game/room.go
git commit -m "feat(game): add lobby_piece_moved WS message broadcast"
```

---

## Task 9: Smoke tests manuais e PR

- [ ] **Step 1: Rodar todos os testes unitários**

```bash
go test ./...
```

Esperado: todos passam.

- [ ] **Step 2: Smoke test curl (requer servidor rodando e banco com dados)**

```bash
# Substituir <TOKEN>, <MATCH_UUID>, <MAP_UUID> por valores reais

# Attach
curl -s -X POST http://localhost:5000/matches/<MATCH_UUID>/map \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"map_uuid": "<MAP_UUID>"}' | jq .

# Get
curl -s http://localhost:5000/matches/<MATCH_UUID>/map \
  -H "Authorization: Bearer <TOKEN>" | jq .

# Detach
curl -s -X DELETE http://localhost:5000/matches/<MATCH_UUID>/map \
  -H "Authorization: Bearer <TOKEN>"
# Esperado: 204 No Content
```

- [ ] **Step 3: Escrever contrato da API em `docs/dev/api/match-maps.md`**

O arquivo já foi definido no spec (§4.5 e §4.6). Criar com o conteúdo do spec.

```bash
# Criar o arquivo docs/dev/api/match-maps.md
# Conteúdo: contratos REST (POST/GET/DELETE /matches/:id/map) + WS (lobby_piece_moved)
# Referência: spec 2026-06-04-tactical-map-fase-6-design.md §4
```

- [ ] **Step 4: Atualizar `docs/documentation-map.yaml`**

Adicionar a entrada para `match-maps.md` no mapa de documentação.

```yaml
# Dentro de docs/documentation-map.yaml, adicionar:
- file: docs/dev/api/match-maps.md
  description: Match-Maps REST (POST/GET/DELETE /matches/{match_uuid}/map) + lobby_piece_moved WS
  touched_by:
    - internal/app/api/matchmap/
    - internal/gateway/pg/matchmap/
    - internal/application/matchmap/
    - internal/app/game/message.go
    - internal/app/game/room.go
```

- [ ] **Step 5: Commit e push**

```bash
git add docs/dev/api/match-maps.md docs/documentation-map.yaml
git commit -m "docs(match-maps): add API contract and documentation-map entry"

git push -u origin feat/tactical-map-fase-6
```

- [ ] **Step 6: Abrir PR**

```bash
gh pr create \
  --title "feat(tactical-map): fase 6 — match_maps REST + lobby_piece_moved WS" \
  --body "$(cat <<'EOF'
## Summary
- Tabela `match_maps` (migração goose)
- Endpoints REST: `POST/GET/DELETE /matches/:id/map`
- Extensão do WS do lobby: `lobby_piece_moved` (broadcast sem sender)
- Contrato: `docs/dev/api/match-maps.md`

## Test plan
- [ ] `go test ./...` passa
- [ ] Smoke test curl: attach, get (200 e 204), detach
- [ ] Smoke test WS: lobby_piece_moved recebido por outros clientes na sala

Cross-link: PR frontend feat/tactical-map-fase-6

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```
