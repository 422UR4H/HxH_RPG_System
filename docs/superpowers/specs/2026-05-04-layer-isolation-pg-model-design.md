# Layer Isolation: eliminar dependências domain/entity → pg/model

**Data:** 2026-05-04  
**Status:** Aprovado para implementação

---

## Problema

Tipos definidos em `internal/gateway/pg/model` são importados por camadas internas (`entity`, `domain`), violando o isolamento de camadas definido em AGENTS.md:

> Dependency: entity ← domain ← app, entity ← gateway. Entities never import outer layers.

### Violações atuais

| Arquivo | Importa | Tipo(s) usado(s) |
|---|---|---|
| `entity/enrollment/enrollment.go` | `pg/model` | `CharacterSheetSummary` |
| `entity/campaign/campaign.go` | `pg/model` | `CharacterSheetSummary` |
| `domain/character_sheet/i_repository.go` | `pg/model` | `CharacterSheet`, `CharacterSheetSummary`, `StatusBar`, `CharacterSheetRelationshipUUIDs` |
| `domain/character_sheet/list_character_sheets.go` | `pg/model` | `CharacterSheetSummary` |
| `domain/character_sheet/get_character_sheet.go` | `pg/model` | `CharacterSheet`, `CharacterProfile`, `StatusBar` |
| `domain/character_sheet/create_character_sheet.go` | `pg/model` | `CharacterSheet`, `CharacterProfile`, `Proficiency`, `JointProficiency` |
| `domain/testutil/mock_character_sheet_repo.go` | `pg/model` | `CharacterSheet`, `CharacterSheetSummary`, `StatusBar`, `CharacterSheetRelationshipUUIDs` |
| `app/api/sheet/character_sheet_sumary_response.go` | `pg/model` | `CharacterSheetSummary` |

---

## Decisão de design

### O que permanece em `pg/model`

`CharacterSheet`, `CharacterProfile`, `Proficiency`, `JointProficiency` são modelos de tabela do banco (mapeamento 1:1 com colunas SQL) e pertencem ao gateway. Nenhum deles deve sair de `pg/model`.

### O que se move para `entity/character_sheet/`

Três tipos são projeções/value objects consumidos tanto pelo gateway (como targets de scan) quanto pelo domain/entity — o mesmo padrão já estabelecido por `entity/match/summary.go`:

| Tipo | Destino | Justificativa |
|---|---|---|
| `CharacterSheetSummary` | `entity/character_sheet/summary.go` | Projeção de leitura; segue padrão de `match.Summary` |
| `StatusBar` (simples `{Min, Curr, Max int}`) | `entity/character_sheet/summary.go` | Dependência direta de `CharacterSheetSummary`; não tem comportamento |
| `CharacterSheetRelationshipUUIDs` | `entity/character_sheet/relationship_uuids.go` | Gateway faz scan direto nele; domain/enrollment o consome |

### Como resolver `IRepository.CreateCharacterSheet` e `GetCharacterSheetByUUID`

`IRepository` vive no domain e não pode referenciar `pg/model.CharacterSheet`. A solução é a **Abordagem 1** (DDD canônico): o `IRepository` passa a usar a entidade de domínio rica `*sheet.CharacterSheet`. O gateway absorve toda a lógica de mapeamento que hoje vaza para o domain.

**Antes:**
```
domain use case → constrói model.CharacterSheet → IRepository.Create(*model.CharacterSheet)
IRepository.Get → model.CharacterSheet → domain use case hidrata sheet.CharacterSheet
```

**Depois:**
```
domain use case → IRepository.Create(*sheet.CharacterSheet)
IRepository.Get → *sheet.CharacterSheet   ← gateway faz todo o mapeamento internamente
```

### Como resolver `IRepository.UpdateStatusBars`

Troca `model.StatusBar` por `status.IStatusBar` (já em entity). O domain passa as barras diretamente; o gateway extrai `GetMin()`, `GetCurrent()`, `GetMax()` internamente. Elimina a conversão intermediária em `persistNormalizedStatus`.

---

## Mudanças por arquivo

### Novos arquivos em `entity/character_sheet/`

**`summary.go`** — pacote `character_sheet`
```go
type StatusBar struct {
    Min  int
    Curr int
    Max  int
}

type Summary struct {
    // todos os campos de pg/model.CharacterSheetSummary
    // Stamina, Health, Aura passam a ser character_sheet.StatusBar
}
```

**`relationship_uuids.go`** — pacote `character_sheet`
```go
type RelationshipUUIDs struct {
    CampaignUUID *uuid.UUID
    PlayerUUID   *uuid.UUID
    MasterUUID   *uuid.UUID
}
```

> Nota: o tipo é renomeado de `CharacterSheetRelationshipUUIDs` para `RelationshipUUIDs` — o contexto do pacote já fornece o prefixo `character_sheet.`.

### `domain/character_sheet/i_repository.go`

Assinaturas que mudam:

```go
// antes
CreateCharacterSheet(ctx context.Context, sheet *model.CharacterSheet) error
GetCharacterSheetByUUID(ctx context.Context, uuid string) (*model.CharacterSheet, error)
ListCharacterSheetsByPlayerUUID(ctx context.Context, playerUUID string) ([]model.CharacterSheetSummary, error)
UpdateStatusBars(ctx context.Context, sheetUUID string, health, stamina, aura model.StatusBar) error
GetCharacterSheetRelationshipUUIDs(ctx context.Context, uuid uuid.UUID) (model.CharacterSheetRelationshipUUIDs, error)

// depois
CreateCharacterSheet(ctx context.Context, sheet *sheet.CharacterSheet) error
GetCharacterSheetByUUID(ctx context.Context, uuid string) (*sheet.CharacterSheet, error)
ListCharacterSheetsByPlayerUUID(ctx context.Context, playerUUID string) ([]csEntity.Summary, error)
UpdateStatusBars(ctx context.Context, sheetUUID string, health, stamina, aura status.IStatusBar) error
GetCharacterSheetRelationshipUUIDs(ctx context.Context, uuid uuid.UUID) (csEntity.RelationshipUUIDs, error)
```

### `domain/character_sheet/create_character_sheet.go`

- `CharacterSheetToModel` migra para `gateway/pg/sheet/` (renomeada para função interna do gateway).
- O use case passa a chamar `uc.repo.CreateCharacterSheet(ctx, characterSheet)` diretamente, sem construir `model.CharacterSheet`.
- Remove import de `pg/model`.

### `domain/character_sheet/get_character_sheet.go`

- `Wrap` migra para `gateway/pg/sheet/` — é lógica de mapeamento model→entity, pertence ao gateway.
- `ModelToProfile` migra para `gateway/pg/sheet/` pelo mesmo motivo.
- `hydrateCharacterSheet` deixa de existir no use case — `GetCharacterSheetByUUID` já retorna `*sheet.CharacterSheet`.
- `persistNormalizedStatus` simplifica: passa `allBars[enum.Health]` etc. diretamente, sem conversão para `model.StatusBar`.
- Remove import de `pg/model`. Remove import de `pg/sheet`: o gateway passa a retornar `ErrCharacterSheetNotFound` (domínio) diretamente, eliminando a conversão `pgSheet.ErrCharacterSheetNotFound → ErrCharacterSheetNotFound` no use case.

### `domain/enrollment/enroll_character_sheet.go`

- Atualiza import de `csEntity.RelationshipUUIDs` (automático — segue a assinatura do `IRepository`).
- Remove import de `pg/sheet`: pelo mesmo motivo acima, o gateway retorna `charactersheet.ErrCharacterSheetNotFound` diretamente; a conversão do erro some do use case.

### `domain/character_sheet/list_character_sheets.go`

- Tipo de retorno muda de `[]model.CharacterSheetSummary` para `[]csEntity.Summary`.
- Remove import de `pg/model`.

### `gateway/pg/sheet/` (implementações)

- `create_character_sheet.go`: absorve `CharacterSheetToModel` (renomeada como helper interno). Recebe `*sheet.CharacterSheet`, mapeia para `model.CharacterSheet`, executa SQL.
- `read_character_sheet.go`:
  - `GetCharacterSheetByUUID`: executa SQL, popula `model.CharacterSheet`, chama `Wrap` (agora local) para construir e retornar `*sheet.CharacterSheet`.
  - `GetCharacterSheetRelationshipUUIDs`: troca `model.CharacterSheetRelationshipUUIDs` por `csEntity.RelationshipUUIDs`.
  - `ListCharacterSheetsByPlayerUUID`: faz scan em `csEntity.Summary` em vez de `model.CharacterSheetSummary`.
- `update_status_bars.go`: recebe `status.IStatusBar`, extrai `GetMin()`, `GetCurrent()`, `GetMax()` internamente.
- `ModelToProfile` e `Wrap` tornam-se funções privadas do pacote `pg/sheet`.
- As implementações de `GetCharacterSheetByUUID` e `GetCharacterSheetRelationshipUUIDs` convertem `pgx.ErrNoRows` para `charactersheet.ErrCharacterSheetNotFound` (importando do domain) em vez de retornar `sheet.ErrCharacterSheetNotFound` — elimina a necessidade de os use cases importarem `pg/sheet` para fazer essa conversão.

### `entity/enrollment/enrollment.go`

```go
// antes
import sheetModel "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/model"
CharacterSheet sheetModel.CharacterSheetSummary

// depois
import csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
CharacterSheet csEntity.Summary
```

### `entity/campaign/campaign.go`

```go
// antes
import "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/model"
CharacterSheets []model.CharacterSheetSummary
PendingSheets   []model.CharacterSheetSummary

// depois
import csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
CharacterSheets []csEntity.Summary
PendingSheets   []csEntity.Summary
```

### `gateway/pg/campaign/read_campaign.go`

Scan de `CharacterSheets` e `PendingSheets` passa a usar `csEntity.Summary` (sem lógica nova — só troca o tipo de variável de scan).

### `gateway/pg/enrollment/list_by_match_uuid.go`

Scan do `e.CharacterSheet` já funciona diretamente em `csEntity.Summary` — sem mudança de lógica.

### `app/api/sheet/character_sheet_sumary_response.go`

Troca import de `pg/model` por `csEntity`. O campo de mapeamento `csEntity.Summary` → response HTTP permanece igual em estrutura.

### `domain/testutil/mock_character_sheet_repo.go`

Atualiza todas as assinaturas para usar os novos tipos (`*sheet.CharacterSheet`, `csEntity.Summary`, `status.IStatusBar`, `csEntity.RelationshipUUIDs`).

### `pg/model/`

Após a refatoração, nenhum arquivo fora de `gateway/` importa `pg/model`. Os arquivos `character_sheet_summary.go`, `status_bar.go`, `character_sheet_relationship_uuids.go` são removidos. Os demais permanecem intactos.

---

## Fluxo de dados pós-refatoração

```
Create:
  domain use case
    → IRepository.CreateCharacterSheet(*sheet.CharacterSheet)
    → gateway: CharacterSheetToModel(*sheet.CharacterSheet) → model.CharacterSheet
    → SQL INSERT

Get:
  domain use case
    → IRepository.GetCharacterSheetByUUID(uuid) → *sheet.CharacterSheet
    → gateway: SQL SELECT → model.CharacterSheet → Wrap → *sheet.CharacterSheet

List:
  domain use case
    → IRepository.ListCharacterSheetsByPlayerUUID → []csEntity.Summary
    → gateway: SQL SELECT → scan em []csEntity.Summary

UpdateStatusBars:
  domain use case
    → IRepository.UpdateStatusBars(uuid, health, stamina, aura status.IStatusBar)
    → gateway: .GetMin()/.GetCurrent()/.GetMax() → SQL UPDATE
```

---

## Testes

- `domain/testutil/mock_character_sheet_repo.go`: atualizar assinaturas.
- `domain/character_sheet/*_test.go`: atualizar mocks e tipos de retorno esperados.
- `app/api/sheet/*_test.go`: atualizar mocks.
- `gateway/pg/sheet/sheet_integration_test.go`: os testes de integração já cobrem o comportamento — verificar que passam após mover `Wrap` para o gateway.
- Não há novos cenários de teste; a lógica migra de camada sem mudança de comportamento.

---

## O que NÃO muda

- Queries SQL (nenhuma mudança).
- Comportamento observável das APIs.
- Estrutura de campos de `CharacterSheetSummary` / `RelationshipUUIDs` (mesmos campos, novo pacote).
- `pg/model.CharacterSheet`, `CharacterProfile`, `Proficiency`, `JointProficiency` (permanecem intocados).
- TODOs existentes no código (conforme AGENTS.md: nunca remover TODOs).
