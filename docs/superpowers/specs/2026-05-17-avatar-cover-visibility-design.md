# Avatar & Cover Visibility — Design Spec

**Date:** 2026-05-17  
**Feature:** Exibir avatar e capa nas listas de fichas (sidebar de campanha/partida) e na tela de detalhe da ficha  
**Stack:** Cross-stack (Go backend + React/TS frontend)  
**Status:** Approved

---

## Problem

Avatar e capa existem no domínio e no banco de dados, mas não aparecem em nenhuma das superfícies listadas abaixo:

1. **Listagem de fichas próprias** (`CharacterSheetsPage`) — `CharacterSheetCard` já usa `CharacterSheetHeader` em modo `card`, mas os campos `coverUrl`/`avatarUrl` nunca chegam do backend (ausentes na query SQL e na response type da listagem).
2. **Tela de detalhe da ficha** (`CharacterSheetPage`) — `CharacterSheetHeader` já renderiza capa e avatar, mas o backend envia `avatar_url`/`cover_url` no JSON do profile; após `objToCamelCase` viram `avatarUrl`/`coverUrl`, enquanto o tipo `Profile` do frontend espera `cover`/`avatar`. Resultado: sempre `undefined`, só placeholder aparece.
3. **Sidebar de campanha** (`CampaignPage` → `CharacterSidebarItem`) — sem estrutura visual de header; apenas nome e barras de HP/SP em texto.
4. **Sidebar de partida** (`MatchPage` → `EnrollmentSidebarItem`, `CharacterSidebarItem`) — mesma ausência.

---

## Goals

- Avatar e capa visíveis em todas as superfícies acima, para todos os usuários.
- HP/SP visíveis apenas para o dono da ficha ou para o mestre (já é o que o backend controla via `private`/`CharacterPrivateSummary`; basta não renderizar quando o dado não existe).
- Estrutura visual dos sidebar items idêntica ao `CharacterSheetHeader` existente, proporcional ao tamanho do container.
- Padronização do nome do campo: `avatar_url`/`cover_url` no JSON → `avatarUrl`/`coverUrl` no TypeScript.

---

## Non-Goals

- Nenhuma mudança na lógica de upload ou PATCH de imagens.
- Nenhuma migração de banco de dados — colunas `avatar_url`/`cover_url` já existem em `character_profiles`.
- Sem alteração na visibilidade de outros campos privados.

---

## Architecture

### Naming Convention

| Camada | Nome |
|--------|------|
| DB columns | `avatar_url`, `cover_url` |
| Go struct fields | `AvatarURL *string`, `CoverURL *string` |
| JSON response tags | `"avatar_url"`, `"cover_url"` |
| TypeScript (após `objToCamelCase`) | `avatarUrl`, `coverUrl` |

---

## Backend Changes (`System_X_System`)

### 1. `csEntity.Summary` — adicionar campos

**Arquivo:** `internal/domain/entity/character_sheet/summary.go`

Adicionar ao struct `Summary`:
```go
AvatarURL *string
CoverURL  *string
```

Esse struct é o portador de dados entre gateway e use case em todos os endpoints que listam fichas. A mudança propaga automaticamente.

### 2. `CharacterProfile` JSON tags — sem mudança

**Arquivo:** `internal/domain/entity/character_sheet/sheet/character_profile.go`

Já usa `json:"avatar_url"` e `json:"cover_url"`. Mantém como está.  
Isso já resolve o **Problema 2** (detalhe da ficha): o frontend precisará renomear `cover`/`avatar` → `coverUrl`/`avatarUrl`.

### 3. Queries SQL — adicionar `cp.avatar_url, cp.cover_url`

Quatro arquivos recebem `cp.avatar_url, cp.cover_url` no SELECT e os campos `&s.AvatarURL, &s.CoverURL` no Scan:

| Arquivo | Função |
|---------|--------|
| `internal/gateway/pg/sheet/read_character_sheet.go` | `ListCharacterSheetsByPlayerUUID` |
| `internal/gateway/pg/campaign/read_campaign.go` | `pendingSheetsQuery` + `characterSheetsQuery` |
| `internal/gateway/pg/enrollment/list_by_match_uuid.go` | `ListByMatchUUID` |
| `internal/gateway/pg/match/read_participants.go` | `ListParticipantsByMatchUUID` |

### 4. `CharacterBaseSummaryResponse` — adicionar campos

**Arquivo:** `internal/app/api/sheet/character_sheet_sumary_response.go`

```go
type CharacterBaseSummaryResponse struct {
    // ... campos existentes ...
    AvatarURL *string `json:"avatar_url,omitempty"`
    CoverURL  *string `json:"cover_url,omitempty"`
}
```

Atualizar `ToBaseSummaryResponse` para popular:
```go
AvatarURL: sheet.AvatarURL,
CoverURL:  sheet.CoverURL,
```

Como `CharacterPrivateSummaryResponse` e `CharacterPublicSummaryResponse` embeds `CharacterBaseSummaryResponse`, os campos aparecem automaticamente nos endpoints de listagem de fichas e de campanha.

---

## Frontend Changes (`System_X_System_React`)

### 5. Tipos TypeScript — renomear e adicionar campos

**Arquivo:** `src/types/characterSheet.ts`

- `Profile`: `cover?: string` → `coverUrl?: string`; `avatar?: string` → `avatarUrl?: string`
- `CharacterSheetSummary`: `cover?: string` → `coverUrl?: string`; `avatar?: string` → `avatarUrl?: string`
- `CharacterBaseSummary`: adicionar `coverUrl?: string` e `avatarUrl?: string`

### 6. `CharacterSheetHeader` — renomear referências + prop `showStatus`

**Arquivo:** `src/components/molecules/CharacterSheetHeader.tsx`

- `profile?.cover` → `profile?.coverUrl`
- `profile?.avatar` → `profile?.avatarUrl`
- Adicionar prop `showStatus?: boolean` (default `true`). Quando `false`, o `StatusBarsContainer` não é renderizado.

```tsx
{showStatus !== false && (
  <StatusBarsContainer>
    <HpBar current={health?.current} max={health?.max} />
    <SpBar current={stamina?.current} max={stamina?.max} />
  </StatusBarsContainer>
)}
```

### 7. `CharacterSheetCard` — renomear referências

**Arquivo:** `src/components/atoms/CharacterSheetCard.tsx`

```tsx
charSheet.profile = {
  // ...
  coverUrl: character?.coverUrl,
  avatarUrl: character?.avatarUrl,
};
```

### 8. `CharacterSidebarItem` — refatorar para usar `CharacterSheetHeader`

**Arquivo:** `src/features/campaign/CharacterSidebarItem.tsx`

Substitui o conteúdo atual (nome + barras textuais) por `CharacterSheetHeader` em modo `"card"`.  
Constrói `charSheet` mínimo a partir do summary (mesmo padrão de `CharacterSheetCard`).  
`showStatus={!!character.health}` — visível apenas quando o dado existe (mestre ou dono da ficha, via `CharacterPrivateSummary`).  
Badges (Dead, NPC, Pending, Saiu) permanecem `position: absolute` sobre o header via `ItemContainer` que mantém `position: relative`.

```tsx
const charSheet = createEmptyCharacterSheet();
charSheet.characterClass = character.characterClass ?? "";
charSheet.profile = {
  nickname: character.nickName,
  fullname: character.fullName ?? "",
  coverUrl: character.coverUrl,
  avatarUrl: character.avatarUrl,
  // demais campos com defaults
};
charSheet.status = {
  health: character.health ?? { min: 0, current: 0, max: 0 },
  stamina: character.stamina ?? { min: 0, current: 0, max: 0 },
};
```

Interface do componente atualizada (`coverUrl`/`avatarUrl` já vêm de `CharacterBaseSummary`):
```ts
interface CharacterSidebarItemProps {
  character: CharacterBaseSummary & {
    isPending?: boolean;
    fullName?: string;
    characterClass?: string;
    health?: StatusBar;
    stamina?: StatusBar;
  };
  isMaster: boolean;
  hasLeft?: boolean;
  onClick: () => void;
}
```

### 9. `EnrollmentSidebarItem` — refatorar para usar `CharacterSheetHeader`

**Arquivo:** `src/features/match/EnrollmentSidebarItem.tsx`

Mesma lógica. `showStatus={!!priv}` onde `priv = enrollment.characterSheet.private`.  
Botões de aceitar/rejeitar permanecem abaixo do header.

---

## Data Flow

```
DB (avatar_url, cover_url)
  └─ Gateway: SELECT cp.avatar_url, cp.cover_url → csEntity.Summary.AvatarURL/CoverURL
       └─ Response: CharacterBaseSummaryResponse.AvatarURL/CoverURL (json:"avatar_url")
            └─ objToCamelCase → avatarUrl / coverUrl (TypeScript)
                 └─ CharacterSheetHeader: profile?.avatarUrl / profile?.coverUrl
```

---

## Visibility Rules

| Superfície | Avatar/Capa | HP/SP |
|------------|-------------|-------|
| Lista de fichas próprias | ✅ sempre | ✅ sempre (são fichas do próprio usuário) |
| Detalhe da ficha | ✅ sempre | ✅ sempre |
| Sidebar campanha — mestre | ✅ sempre | ✅ (`CharacterPrivateSummary` tem health/stamina) |
| Sidebar campanha — jogador | ✅ sempre | ❌ (`CharacterPublicSummary` não tem health/stamina) |
| Sidebar partida (enrollment) | ✅ sempre | ✅ só para mestre (`private != null`) |
| Sidebar partida (participant) | ✅ sempre | ✅ só para mestre (`private != null`) |

---

## API Contract Update

Atualizar `docs/dev/api/character-sheet.md` para documentar que `GET /charactersheets` e os endpoints de campanha/partida passam a incluir `avatar_url` e `cover_url` nos summaries de ficha.

---

## Testing

- `go vet -tags=integration ./internal/gateway/pg/...` após cada mudança de gateway
- Verificar visualmente nas 4 superfícies após implementação

---

## Files Affected

### Backend (`System_X_System`)
- `internal/domain/entity/character_sheet/summary.go`
- `internal/gateway/pg/sheet/read_character_sheet.go`
- `internal/gateway/pg/campaign/read_campaign.go`
- `internal/gateway/pg/enrollment/list_by_match_uuid.go`
- `internal/gateway/pg/match/read_participants.go`
- `internal/app/api/sheet/character_sheet_sumary_response.go`
- `docs/dev/api/character-sheet.md`

### Frontend (`System_X_System_React`)
- `src/types/characterSheet.ts`
- `src/components/molecules/CharacterSheetHeader.tsx`
- `src/components/atoms/CharacterSheetCard.tsx`
- `src/features/campaign/CharacterSidebarItem.tsx`
- `src/features/match/EnrollmentSidebarItem.tsx`
