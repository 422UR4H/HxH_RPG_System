# Participantes de Partida — Spec de Design

**Data:** 2026-05-04  
**Branch:** `refactor/layer-isolation-pg-model`  
**Status:** Aprovado

## Contexto

A página de detalhes da partida (front-end, fora do escopo) precisa dos summaries dos `CharacterSheet`s participantes. Isso requer uma tabela n:m entre partidas e fichas, populada quando a partida inicia e atualizada ao longo dela.

A tabela `enrollments` é o fluxo de aprovação pré-partida. `match_participants` é o registro de participação dentro da partida — um conceito de domínio distinto, com ciclo de vida próprio.

**Nota futura (fora do escopo):** `match_participants` também servirá como seed para carregar objetos `CharacterSheet` completos em um struct `MatchSession` em memória (`app/game/`) quando a camada WS do jogo for implementada.

---

## Decisões

| Tópico | Decisão |
|--------|---------|
| Endpoint | Separado `GET /matches/{uuid}/participants` — não embutido no `GetMatch` |
| Visibilidade | Mesmo padrão `ViewerIsMaster` do `list_match_enrollments` |
| Rastreamento de participação | Três timestamps: `joined_at`, `left_at`, `died_at` |
| Integração com StartMatch | Abordagem A — operação atômica no gateway dentro do `StartMatchUC` |
| Entidade `Participant` | `domain/entity/match/participant.go` — mesmo pacote, sem pacote novo |
| Gateway | Novos arquivos no pacote existente `pg/match/` — mesma `Repository` |
| `CharacterSheetWithVisibilityResponse` | Mover de `api/match/` → `api/sheet/` (usado por dois handlers) |
| Entidade `Match` | Inalterada — sem campo `CharacterSheets` |
| Timestamp `game_start_at` | Gerado no `StartMatchUC` (não no gateway) — passado para `StartMatch` e `RegisterFromAcceptedEnrollments` |

---

## Schema

```sql
CREATE TABLE match_participants (
    id   SERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),

    match_uuid           UUID NOT NULL REFERENCES matches(uuid),
    character_sheet_uuid UUID NOT NULL REFERENCES character_sheets(uuid),

    -- Rastreamento de participação por timestamps (todos ortogonais):
    --   joined_at vs match.game_start_at → se entrou depois do início
    --   left_at IS NULL + match.story_end_at IS NOT NULL → completou normalmente
    --   died_at IS NOT NULL → morreu durante a partida (independente de left_at)
    joined_at TIMESTAMP NOT NULL,
    left_at   TIMESTAMP,
    died_at   TIMESTAMP,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (uuid),
    UNIQUE (match_uuid, character_sheet_uuid)
);
CREATE INDEX idx_match_participants_match_uuid ON match_participants(match_uuid);
```

`UNIQUE (match_uuid, character_sheet_uuid)` — uma ficha participa no máximo uma vez por partida. Diferente de `enrollments`, não é necessário unique condicional.

---

## Entidade de Domínio

**`internal/domain/entity/match/participant.go`** — mesmo pacote que `Match`, `Summary`, `GameEvent`.

```go
package match

import (
    "time"
    csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
    "github.com/google/uuid"
)

type Participant struct {
    UUID      uuid.UUID
    MatchUUID uuid.UUID
    Sheet     csEntity.Summary
    JoinedAt  time.Time
    LeftAt    *time.Time
    DiedAt    *time.Time
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

Sem métodos derivados na entidade — `JoinedLate` e `IsActive` são preocupações de apresentação, calculadas pelo handler a partir dos timestamps.

---

## Casos de Uso

### `StartMatchUC` — extensão

**Mudança de assinatura de `IRepository.StartMatch`** (alinha com gateway-conventions — Go gera timestamps):

```go
// internal/domain/match/i_repository.go
StartMatch(ctx context.Context, matchUUID uuid.UUID, gameStartAt time.Time) error
```

**Nova interface estreita em `start_match.go`:**

```go
type IMatchParticipantWriter interface {
    RegisterFromAcceptedEnrollments(
        ctx context.Context, matchUUID uuid.UUID, gameStartAt time.Time,
    ) error
}
```

**Fluxo atualizado do `Start()`:**

```go
func (uc *StartMatchUC) Start(ctx context.Context, matchUUID, masterUUID uuid.UUID) error {
    // ... validações existentes (GetMatch, check MasterUUID, AlreadyStarted, AlreadyFinished) ...

    gameStartAt := time.Now()
    if err := uc.matchRepo.StartMatch(ctx, matchUUID, gameStartAt); err != nil {
        return err
    }
    if err := uc.enrollmentRepo.RejectPendingEnrollments(ctx, matchUUID); err != nil {
        return err
    }
    return uc.participantRepo.RegisterFromAcceptedEnrollments(ctx, matchUUID, gameStartAt)
}
```

### `GetMatchParticipantsUC` — novo

**`internal/domain/match/get_match_participants.go`**

Espelha o `ListMatchEnrollmentsUC` exatamente na lógica de autorização.

```go
type IMatchParticipantReader interface {
    ListParticipantsByMatchUUID(
        ctx context.Context, matchUUID uuid.UUID,
    ) ([]*matchEntity.Participant, error)
}

type GetMatchParticipantsResult struct {
    Participants   []*matchEntity.Participant
    ViewerIsMaster bool
}

type IGetMatchParticipants interface {
    Get(ctx context.Context, matchUUID, userUUID uuid.UUID) (*GetMatchParticipantsResult, error)
}
```

Autorização: igual ao `ListMatchEnrollmentsUC` — busca a partida, verifica `MasterUUID == userUUID`, se privada e não é master → `ExistsSheetInCampaign`. Reutiliza a interface `CampaignParticipationChecker` já declarada no pacote.

---

## Gateway — `pg/match/`

Novos arquivos dentro do pacote existente (sem pacote novo, mesma `Repository`):

### `pg/match/register_participants.go`

```go
func (r *Repository) RegisterFromAcceptedEnrollments(
    ctx context.Context, matchUUID uuid.UUID, gameStartAt time.Time,
) error {
    now := time.Now()
    // INSERT ... SELECT é atômico no PostgreSQL — transação explícita desnecessária.
    const query = `
        INSERT INTO match_participants
            (uuid, match_uuid, character_sheet_uuid, joined_at, created_at, updated_at)
        SELECT gen_random_uuid(), match_uuid, character_sheet_uuid, $2, $3, $3
        FROM enrollments
        WHERE match_uuid = $1 AND status = 'accepted'
        ON CONFLICT (match_uuid, character_sheet_uuid) DO NOTHING
    `
    _, err := r.q.Exec(ctx, query, matchUUID, gameStartAt, now)
    if err != nil {
        return fmt.Errorf("failed to register match participants: %w", err)
    }
    return nil
}
```

`ON CONFLICT DO NOTHING` — idempotente em caso de chamada duplicada.

### `pg/match/read_participants.go`

`ListParticipantsByMatchUUID` — JOIN em `character_sheets`, `character_profiles`, `LEFT JOIN users` (fichas de NPC pertencem ao mestre e não têm `player_uuid`; o gateway de enrollment usa INNER JOIN e excluiria NPCs silenciosamente — participantes devem usar LEFT JOIN).

O scan inclui `cs.story_start_at`, `cs.story_current_at`, `cs.dead_at` — campos usados por `ToBaseSummaryResponse` e que o gateway de enrollment atualmente omite. Summaries de participantes devem ser completos.

**`Participant.DiedAt` vs `csEntity.Summary.DeadAt`:** são campos distintos. `DiedAt` em `Participant` registra quando o personagem morreu nesta partida específica (evento scoped à partida). `DeadAt` em `csEntity.Summary` (de `character_sheets.dead_at`) é a data canônica de morte do personagem na história, definida separadamente. Ambos podem ser não-nil simultaneamente.

### `pg/match/start_match.go` — atualizado

Recebe `gameStartAt time.Time` como parâmetro em vez de gerar `time.Now()` internamente.

---

## Handler de API

### `CharacterSheetWithVisibilityResponse` — mover

De `internal/app/api/match/list_match_enrollments.go` → `internal/app/api/sheet/` (onde vivem todos os tipos de apresentação de ficha). Usado por ambos os handlers de enrollment e participants.

### Novos arquivos em `internal/app/api/match/`

- `get_match_participants.go`
- `get_match_participants_test.go`

### Shape da resposta

```go
type ParticipantResponse struct {
    UUID     uuid.UUID                                     `json:"uuid"`
    JoinedAt string                                        `json:"joined_at"`
    LeftAt   *string                                       `json:"left_at,omitempty"`
    DiedAt   *string                                       `json:"died_at,omitempty"`
    Sheet    apiSheet.CharacterSheetWithVisibilityResponse `json:"character_sheet"`
}
```

`JoinedLate` omitido da resposta — o front-end computa a partir de `joined_at` vs `match.game_start_at` (já presente no `GET /matches/{uuid}`).

Handler espelha `ListMatchEnrollmentsHandler`: mesmo switch de erros, mesmo `ViewerIsMaster` → `private: null` vs dados completos.

### Rota

`GET /matches/{uuid}/participants` registrada em `internal/app/api/match/routes.go`.

### Wiring (`api.go`)

`pg/match.Repository` satisfaz tanto `IMatchParticipantWriter` quanto `IMatchParticipantReader` via structural typing — uma instância injetada em ambos os UCs.

---

## Testes

| Camada | Tipo | Arquivo |
|--------|------|---------|
| `pg/match/` — register | Integração | `match_integration_test.go` |
| `pg/match/` — read participants | Integração | `match_integration_test.go` |
| `domain/match/StartMatchUC` | Unit (mock) | `start_match_test.go` |
| `domain/match/GetMatchParticipantsUC` | Unit (mock) | novo `get_match_participants_test.go` |
| `app/api/match/` handler | Unit (humatest) | `get_match_participants_test.go` |

Novo helper pgtest: `InsertTestMatchParticipant(t, pool, matchUUID, sheetUUID string, joinedAt time.Time) string`

---

## Limite de Escopo

| Dentro do escopo | Fora do escopo |
|-----------------|----------------|
| Migration `match_participants` | `MatchSession` em `app/game/` |
| Entidade `Participant` | Carregamento de `CharacterSheet` completos no WS Room |
| Extensão do `StartMatchUC` | Endpoints de join/leave/death durante a partida |
| `GetMatchParticipantsUC` + handler | |
| Refactor de `CharacterSheetWithVisibilityResponse` | |
