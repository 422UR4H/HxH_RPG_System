# Delete Campaign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `DELETE /campaigns/{uuid}` following the same 3-layer pattern (gateway → use case → handler) as `DELETE /matches/{uuid}`.

**Architecture:** Gateway uses an atomic `DELETE … AND NOT EXISTS` query to guard against started matches, the use case pre-checks ownership via `GetCampaignMasterUUID`, and the handler maps domain errors to HTTP statuses. A migration adds `ON DELETE CASCADE` to the FK from `matches` and `submissions` to `campaigns` so unstarted matches cascade.

**Tech Stack:** Go 1.23, pgx/v5, huma/v2, goose migrations, humatest.

---

## File Map

| Action | Path |
|--------|------|
| Create | `migrations/20260525000001_add_cascade_to_campaign_fks.sql` |
| Create | `internal/gateway/pg/campaign/delete_campaign.go` |
| Modify | `internal/application/campaign/i_repository.go` |
| Modify | `internal/application/campaign/error.go` |
| Create | `internal/application/campaign/delete_campaign.go` |
| Modify | `internal/app/api/campaign/mocks_test.go` |
| Create | `internal/app/api/campaign/delete_campaign_test.go` |
| Create | `internal/app/api/campaign/delete_campaign.go` |
| Modify | `internal/app/api/campaign/routes.go` |
| Modify | `cmd/api/main.go` |
| Modify | `internal/gateway/pg/campaign/campaign_integration_test.go` |
| Create | `docs/dev/api/campaign.md` |
| Modify | `docs/documentation-map.yaml` |

---

### Task 1: Migration — ON DELETE CASCADE em matches e submissions

**Files:**
- Create: `migrations/20260525000001_add_cascade_to_campaign_fks.sql`

- [ ] **Step 1: Criar o arquivo de migration**

```sql
-- +goose Up
-- +goose StatementBegin
BEGIN;

ALTER TABLE matches DROP CONSTRAINT IF EXISTS matches_campaign_uuid_fkey;
ALTER TABLE matches
  ADD CONSTRAINT matches_campaign_uuid_fkey
  FOREIGN KEY (campaign_uuid) REFERENCES campaigns(uuid) ON DELETE CASCADE;

ALTER TABLE submissions DROP CONSTRAINT IF EXISTS submissions_campaign_uuid_fkey;
ALTER TABLE submissions
  ADD CONSTRAINT submissions_campaign_uuid_fkey
  FOREIGN KEY (campaign_uuid) REFERENCES campaigns(uuid) ON DELETE CASCADE;

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

ALTER TABLE matches DROP CONSTRAINT IF EXISTS matches_campaign_uuid_fkey;
ALTER TABLE matches
  ADD CONSTRAINT matches_campaign_uuid_fkey
  FOREIGN KEY (campaign_uuid) REFERENCES campaigns(uuid);

ALTER TABLE submissions DROP CONSTRAINT IF EXISTS submissions_campaign_uuid_fkey;
ALTER TABLE submissions
  ADD CONSTRAINT submissions_campaign_uuid_fkey
  FOREIGN KEY (campaign_uuid) REFERENCES campaigns(uuid);

COMMIT;
-- +goose StatementEnd
```

- [ ] **Step 2: Aplicar a migration localmente**

```bash
make migrate-up
```

Expected: migration aplicada sem erros.

- [ ] **Step 3: Commit**

```bash
git add migrations/20260525000001_add_cascade_to_campaign_fks.sql
git commit -m "feat(db): add ON DELETE CASCADE to matches and submissions → campaigns FK"
```

---

### Task 2: Gateway — DeleteCampaign

**Files:**
- Modify: `internal/application/campaign/i_repository.go`
- Create: `internal/gateway/pg/campaign/delete_campaign.go`
- Modify: `internal/gateway/pg/campaign/campaign_integration_test.go`

- [ ] **Step 1: Adicionar DeleteCampaign à interface IRepository**

Em `internal/application/campaign/i_repository.go`, adicionar ao final da interface:

```go
DeleteCampaign(ctx context.Context, uuid uuid.UUID) error
```

Arquivo completo após edição:
```go
package campaign

import (
	"context"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	"github.com/google/uuid"
)

type IRepository interface {
	CreateCampaign(ctx context.Context, campaign *campaign.Campaign) error
	GetCampaign(ctx context.Context, uuid uuid.UUID) (*campaign.Campaign, error)
	GetCampaignMasterUUID(ctx context.Context, uuid uuid.UUID) (uuid.UUID, error)
	GetCampaignStoryDates(ctx context.Context, uuid uuid.UUID) (*campaign.Campaign, error)
	CountCampaignsByMasterUUID(ctx context.Context, masterUUID uuid.UUID) (int, error)
	ListCampaignsByMasterUUID(ctx context.Context, masterUUID uuid.UUID) ([]*campaign.Summary, error)
	ListPublicUpcomingCampaigns(ctx context.Context, after time.Time, userUUID uuid.UUID) ([]*campaign.PublicSummary, error)
	DeleteCampaign(ctx context.Context, uuid uuid.UUID) error
}
```

- [ ] **Step 2: Escrever o teste de integração (falha esperada)**

Adicionar ao final de `internal/gateway/pg/campaign/campaign_integration_test.go`:

```go
func TestDeleteCampaign(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgCampaign.NewRepository(pool)
	ctx := context.Background()

	t.Run("happy_path", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		masterUUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "delmaster", "delmaster@hunter.com", "pass"))
		campaignUUID := mustParseUUID(t, pgtest.InsertTestCampaign(t, pool, masterUUID.String(), "Campaign To Delete"))

		if err := repo.DeleteCampaign(ctx, campaignUUID); err != nil {
			t.Fatalf("DeleteCampaign() unexpected error: %v", err)
		}

		_, err := repo.GetCampaignMasterUUID(ctx, campaignUUID)
		if !errors.Is(err, pgCampaign.ErrCampaignNotFound) {
			t.Errorf("expected ErrCampaignNotFound after delete, got: %v", err)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		err := repo.DeleteCampaign(ctx, uuid.New())
		if !errors.Is(err, pgCampaign.ErrCampaignNotFound) {
			t.Errorf("expected ErrCampaignNotFound, got: %v", err)
		}
	})

	t.Run("has_started_match", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		masterUUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "delmaster2", "delmaster2@hunter.com", "pass"))
		campaignUUID := mustParseUUID(t, pgtest.InsertTestCampaign(t, pool, masterUUID.String(), "Campaign With Started Match"))
		matchUUIDStr := pgtest.InsertTestMatch(t, pool, masterUUID.String(), campaignUUID.String(), "Started Match")

		if _, err := pool.Exec(ctx, `UPDATE matches SET game_start_at = $1 WHERE uuid = $2`, time.Now(), matchUUIDStr); err != nil {
			t.Fatalf("failed to set game_start_at: %v", err)
		}

		err := repo.DeleteCampaign(ctx, campaignUUID)
		if !errors.Is(err, pgCampaign.ErrCampaignNotFound) {
			t.Errorf("expected ErrCampaignNotFound (campaign has started match), got: %v", err)
		}
	})

	t.Run("cascade_unstarted_match", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		masterUUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "delmaster3", "delmaster3@hunter.com", "pass"))
		campaignUUID := mustParseUUID(t, pgtest.InsertTestCampaign(t, pool, masterUUID.String(), "Campaign With Pending Match"))
		matchUUIDStr := pgtest.InsertTestMatch(t, pool, masterUUID.String(), campaignUUID.String(), "Pending Match")

		if err := repo.DeleteCampaign(ctx, campaignUUID); err != nil {
			t.Fatalf("DeleteCampaign() unexpected error: %v", err)
		}

		var count int
		if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM matches WHERE uuid = $1`, matchUUIDStr).Scan(&count); err != nil {
			t.Fatalf("failed to check match existence: %v", err)
		}
		if count != 0 {
			t.Errorf("expected match to be cascade deleted, but found %d row(s)", count)
		}
	})
}
```

O bloco de imports no topo do arquivo já tem `"errors"`, `"context"`, `"time"`, `uuid` e os packages do projeto — confirme e adicione o que faltar.

- [ ] **Step 3: Verificar que o teste falha por `DeleteCampaign` não existir**

```bash
go vet -tags=integration ./internal/gateway/pg/campaign/...
```

Expected: erro de compilação — `*Repository` does not implement `IRepository` (missing `DeleteCampaign`).

- [ ] **Step 4: Implementar `DeleteCampaign` no gateway**

Criar `internal/gateway/pg/campaign/delete_campaign.go`:

```go
package campaign

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (r *Repository) DeleteCampaign(ctx context.Context, campaignUUID uuid.UUID) error {
	const query = `
		DELETE FROM campaigns WHERE uuid = $1
		AND NOT EXISTS (
			SELECT 1 FROM matches
			WHERE campaign_uuid = $1 AND game_start_at IS NOT NULL
		)`
	result, err := r.q.Exec(ctx, query, campaignUUID)
	if err != nil {
		return fmt.Errorf("failed to delete campaign: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrCampaignNotFound
	}
	return nil
}
```

- [ ] **Step 5: Verificar vet passa**

```bash
go vet -tags=integration ./internal/gateway/pg/campaign/...
```

Expected: sem erros.

- [ ] **Step 6: Rodar os testes de integração**

```bash
go test -tags=integration ./internal/gateway/pg/campaign/...
```

Expected: todos passam, incluindo `TestDeleteCampaign`.

- [ ] **Step 7: Commit**

```bash
git add internal/application/campaign/i_repository.go \
        internal/gateway/pg/campaign/delete_campaign.go \
        internal/gateway/pg/campaign/campaign_integration_test.go
git commit -m "feat(campaign): implement DeleteCampaign gateway with NOT EXISTS guard"
```

---

### Task 3: Application Use Case — DeleteCampaignUC

**Files:**
- Modify: `internal/application/campaign/error.go`
- Create: `internal/application/campaign/delete_campaign.go`

- [ ] **Step 1: Adicionar `ErrCampaignHasStartedMatch` em error.go**

Em `internal/application/campaign/error.go`, adicionar ao final do bloco `var`:

```go
ErrCampaignHasStartedMatch = domain.NewValidationError(errors.New("campaign has a match that has already started"))
```

Arquivo completo após edição:
```go
package campaign

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrCampaignNotFound        = domain.NewValidationError(errors.New("campaign not found"))
	ErrNotCampaignOwner        = domain.NewValidationError(errors.New("master is not the owner of this campaign"))
	ErrMinNameLength            = domain.NewValidationError(errors.New("name must be at least 5 characters"))
	ErrMaxNameLength            = domain.NewValidationError(errors.New("name cannot exceed 32 characters"))
	ErrInvalidStartDate         = domain.NewValidationError(errors.New("story start date cannot be empty"))
	ErrMaxCampaignsLimit        = domain.NewValidationError(errors.New("user cannot have more than 10 campaigns"))
	ErrMaxBriefDescLength       = domain.NewValidationError(errors.New("brief description cannot exceed 64 characters"))
	ErrCampaignHasStartedMatch  = domain.NewValidationError(errors.New("campaign has a match that has already started"))
)
```

- [ ] **Step 2: Criar `delete_campaign.go` na camada de application**

Criar `internal/application/campaign/delete_campaign.go`:

```go
package campaign

import (
	"context"
	"errors"

	campaignPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	"github.com/google/uuid"
)

type IDeleteCampaign interface {
	Delete(ctx context.Context, input *DeleteCampaignInput) error
}

type DeleteCampaignInput struct {
	CampaignUUID uuid.UUID
	MasterUUID   uuid.UUID
}

type DeleteCampaignUC struct {
	repo IRepository
}

func NewDeleteCampaignUC(repo IRepository) *DeleteCampaignUC {
	return &DeleteCampaignUC{repo: repo}
}

func (uc *DeleteCampaignUC) Delete(ctx context.Context, input *DeleteCampaignInput) error {
	masterUUID, err := uc.repo.GetCampaignMasterUUID(ctx, input.CampaignUUID)
	if err != nil {
		if errors.Is(err, campaignPg.ErrCampaignNotFound) {
			return ErrCampaignNotFound
		}
		return err
	}

	if masterUUID != input.MasterUUID {
		return ErrNotCampaignOwner
	}

	err = uc.repo.DeleteCampaign(ctx, input.CampaignUUID)
	if err != nil {
		if errors.Is(err, campaignPg.ErrCampaignNotFound) {
			// Race condition: a match started between GetCampaignMasterUUID and DeleteCampaign
			return ErrCampaignHasStartedMatch
		}
		return err
	}

	return nil
}
```

- [ ] **Step 3: Verificar compilação**

```bash
go vet ./internal/application/campaign/...
```

Expected: sem erros.

- [ ] **Step 4: Commit**

```bash
git add internal/application/campaign/error.go \
        internal/application/campaign/delete_campaign.go
git commit -m "feat(campaign): add DeleteCampaignUC with ownership and started-match guard"
```

---

### Task 4: Handler, Testes e Route

**Files:**
- Modify: `internal/app/api/campaign/mocks_test.go`
- Create: `internal/app/api/campaign/delete_campaign_test.go`
- Create: `internal/app/api/campaign/delete_campaign.go`
- Modify: `internal/app/api/campaign/routes.go`

- [ ] **Step 1: Adicionar mock `mockDeleteCampaign` em mocks_test.go**

Em `internal/app/api/campaign/mocks_test.go`, adicionar ao final:

```go
type mockDeleteCampaign struct {
	fn func(ctx context.Context, input *campaign.DeleteCampaignInput) error
}

func (m *mockDeleteCampaign) Delete(ctx context.Context, input *campaign.DeleteCampaignInput) error {
	return m.fn(ctx, input)
}
```

- [ ] **Step 2: Escrever o teste unitário do handler (falha esperada)**

Criar `internal/app/api/campaign/delete_campaign_test.go`:

```go
package campaign_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/campaign"
	campaignUC "github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestDeleteCampaignHandler(t *testing.T) {
	userUUID := uuid.New()
	campaignUUID := uuid.New()

	tests := []struct {
		name       string
		uuidPath   string
		mockFn     func(ctx context.Context, input *campaignUC.DeleteCampaignInput) error
		wantStatus int
	}{
		{
			name:     "success",
			uuidPath: campaignUUID.String(),
			mockFn: func(_ context.Context, input *campaignUC.DeleteCampaignInput) error {
				if input.CampaignUUID != campaignUUID {
					t.Errorf("campaign uuid not forwarded: got %v", input.CampaignUUID)
				}
				if input.MasterUUID != userUUID {
					t.Errorf("master uuid not forwarded: got %v", input.MasterUUID)
				}
				return nil
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "invalid_uuid",
			uuidPath:   "not-a-uuid",
			mockFn:     func(_ context.Context, _ *campaignUC.DeleteCampaignInput) error { return nil },
			wantStatus: http.StatusBadRequest,
		},
		{
			name:     "campaign_not_found",
			uuidPath: campaignUUID.String(),
			mockFn: func(_ context.Context, _ *campaignUC.DeleteCampaignInput) error {
				return campaignUC.ErrCampaignNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:     "not_owner",
			uuidPath: campaignUUID.String(),
			mockFn: func(_ context.Context, _ *campaignUC.DeleteCampaignInput) error {
				return campaignUC.ErrNotCampaignOwner
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name:     "has_started_match",
			uuidPath: campaignUUID.String(),
			mockFn: func(_ context.Context, _ *campaignUC.DeleteCampaignInput) error {
				return campaignUC.ErrCampaignHasStartedMatch
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:     "internal_server_error",
			uuidPath: campaignUUID.String(),
			mockFn: func(_ context.Context, _ *campaignUC.DeleteCampaignInput) error {
				return errors.New("db error")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)
			handler := campaign.DeleteCampaignHandler(&mockDeleteCampaign{fn: tt.mockFn})

			huma.Register(api, huma.Operation{
				Method: http.MethodDelete,
				Path:   "/campaigns/{uuid}",
				Errors: []int{
					http.StatusBadRequest, http.StatusNotFound,
					http.StatusForbidden, http.StatusUnprocessableEntity,
					http.StatusInternalServerError,
				},
				DefaultStatus: http.StatusNoContent,
			}, handler)

			ctx := context.WithValue(context.Background(), auth.UserIDKey, userUUID)
			resp := api.DeleteCtx(ctx, "/campaigns/"+tt.uuidPath)

			if resp.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d. Body: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}
		})
	}
}
```

- [ ] **Step 3: Confirmar falha de compilação**

```bash
go vet ./internal/app/api/campaign/...
```

Expected: erro — `campaign.DeleteCampaignHandler` undefined.

- [ ] **Step 4: Implementar o handler**

Criar `internal/app/api/campaign/delete_campaign.go`:

```go
package campaign

import (
	"context"
	"errors"
	"net/http"

	apiAuth "github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	campaignUC "github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type DeleteCampaignRequest struct {
	UUID string `path:"uuid" required:"true"`
}

type DeleteCampaignResponse struct {
	Status int
}

func DeleteCampaignHandler(
	uc campaignUC.IDeleteCampaign,
) func(context.Context, *DeleteCampaignRequest) (*DeleteCampaignResponse, error) {
	return func(ctx context.Context, req *DeleteCampaignRequest) (*DeleteCampaignResponse, error) {
		userUUID, ok := ctx.Value(apiAuth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		campaignUUID, err := uuid.Parse(req.UUID)
		if err != nil {
			return nil, huma.Error400BadRequest("invalid uuid")
		}

		err = uc.Delete(ctx, &campaignUC.DeleteCampaignInput{
			CampaignUUID: campaignUUID,
			MasterUUID:   userUUID,
		})
		if err != nil {
			switch {
			case errors.Is(err, campaignUC.ErrCampaignNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, campaignUC.ErrNotCampaignOwner):
				return nil, huma.Error403Forbidden(err.Error())
			case errors.Is(err, campaignUC.ErrCampaignHasStartedMatch):
				return nil, huma.Error422UnprocessableEntity(err.Error())
			default:
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}

		return &DeleteCampaignResponse{Status: http.StatusNoContent}, nil
	}
}
```

- [ ] **Step 5: Registrar rota e adicionar campo na struct Api**

Em `internal/app/api/campaign/routes.go`:

Adicionar campo à struct `Api`:
```go
DeleteCampaignHandler Handler[DeleteCampaignRequest, DeleteCampaignResponse]
```

Adicionar bloco de registro em `RegisterRoutes`, após o bloco `ListPublicUpcomingCampaigns`:
```go
huma.Register(api, huma.Operation{
    Method:      http.MethodDelete,
    Path:        "/campaigns/{uuid}",
    Description: "Delete a campaign by UUID",
    Tags:        []string{"campaigns"},
    Errors: []int{
        http.StatusBadRequest,
        http.StatusUnauthorized,
        http.StatusForbidden,
        http.StatusNotFound,
        http.StatusUnprocessableEntity,
        http.StatusInternalServerError,
    },
    DefaultStatus: http.StatusNoContent,
}, a.DeleteCampaignHandler)
```

- [ ] **Step 6: Rodar os testes unitários do handler**

```bash
go test ./internal/app/api/campaign/...
```

Expected: todos os 6 casos de `TestDeleteCampaignHandler` passam.

- [ ] **Step 7: Commit**

```bash
git add internal/app/api/campaign/delete_campaign.go \
        internal/app/api/campaign/delete_campaign_test.go \
        internal/app/api/campaign/mocks_test.go \
        internal/app/api/campaign/routes.go
git commit -m "feat(campaign): add DeleteCampaignHandler with unit tests and route registration"
```

---

### Task 5: Wiring

**Files:**
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Instanciar DeleteCampaignUC e registrar no Api**

Em `cmd/api/main.go`, após a linha `listPublicUpcomingCampaignsUC := ...`, adicionar:

```go
deleteCampaignUC := campaign.NewDeleteCampaignUC(campaignRepo)
```

No bloco `campaignsApi := campaignHandler.Api{...}`, adicionar o campo:

```go
DeleteCampaignHandler: campaignHandler.DeleteCampaignHandler(deleteCampaignUC),
```

O bloco completo ficará:
```go
campaignsApi := campaignHandler.Api{
    CreateCampaignHandler:              campaignHandler.CreateCampaignHandler(createCampaignUC),
    GetCampaignHandler:                 campaignHandler.GetCampaignHandler(getCampaignUC, listPlayerEnrollmentsForCampaignUC),
    ListCampaignsHandler:               campaignHandler.ListCampaignsHandler(listCampaignsUC),
    ListPublicUpcomingCampaignsHandler: campaignHandler.ListPublicUpcomingCampaignsHandler(listPublicUpcomingCampaignsUC),
    DeleteCampaignHandler:              campaignHandler.DeleteCampaignHandler(deleteCampaignUC),
}
```

- [ ] **Step 2: Verificar compilação do binário**

```bash
go vet ./cmd/api/...
```

Expected: sem erros.

- [ ] **Step 3: Commit**

```bash
git add cmd/api/main.go
git commit -m "feat(campaign): wire DeleteCampaignUC and handler in main"
```

---

### Task 6: Contrato de API e Documentation Map

**Files:**
- Create: `docs/dev/api/campaign.md`
- Modify: `docs/documentation-map.yaml`

- [ ] **Step 1: Criar `docs/dev/api/campaign.md`**

```markdown
# Campaign API

## DELETE /campaigns/{uuid} — Deletar campanha

**Auth:** JWT (master da campanha)

### Path Parameters

| Parâmetro | Tipo | Descrição |
|---|---|---|
| `uuid` | UUID v4 | UUID da campanha a deletar |

### Request

Sem body.

### Respostas

| Status | Situação |
|---|---|
| 204 | Campanha deletada com sucesso. Partidas não-iniciadas e submissions cascadeiam. |
| 400 | UUID inválido no path |
| 401 | Sem JWT |
| 403 | Usuário não é o master da campanha |
| 404 | Campanha não encontrada |
| 422 | Campanha possui ao menos uma partida que já foi iniciada (`game_start_at IS NOT NULL`) |
| 500 | Erro interno |

### Notas

- A deleção é atômica: se qualquer partida associada tiver `game_start_at != null`, a query não deleta e retorna 422.
- Partidas não-iniciadas e suas submissions são removidas via `ON DELETE CASCADE`.
- O check de `game_start_at IS NOT NULL` é feito diretamente no SQL (`NOT EXISTS` subquery), garantindo atomicidade contra race conditions onde uma partida pode iniciar entre a verificação de ownership e a deleção.
```

- [ ] **Step 2: Registrar em `docs/documentation-map.yaml`**

Adicionar entrada no `documentation-map.yaml` (seguindo o padrão das entradas existentes para `app/api/campaign`):

```yaml
- code_path: internal/app/api/campaign/delete_campaign.go
  dev_docs:
    - path: docs/dev/api/campaign.md
      confidence: directly_affected
  game_docs: []
  notes: "DELETE /campaigns/{uuid} handler"

- code_path: internal/application/campaign/delete_campaign.go
  dev_docs:
    - path: docs/dev/api/campaign.md
      confidence: directly_affected
  game_docs: []
  notes: "DeleteCampaignUC"

- code_path: internal/gateway/pg/campaign/delete_campaign.go
  dev_docs:
    - path: docs/dev/api/campaign.md
      confidence: directly_affected
  game_docs: []
  notes: "DeleteCampaign gateway"
```

- [ ] **Step 3: Commit**

```bash
git add docs/dev/api/campaign.md docs/documentation-map.yaml
git commit -m "docs(campaign): add DELETE /campaigns/{uuid} API contract and map entries"
```

---

## Verificação Final

- [ ] `go vet ./...` sem erros
- [ ] `go test ./internal/app/api/campaign/...` todos passam
- [ ] `go test -tags=integration ./internal/gateway/pg/campaign/...` todos passam (requer DB local)
- [ ] `make build` ou `go build ./cmd/api/` compila sem erros
