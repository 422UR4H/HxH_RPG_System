# Avatar & Cover Visibility — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Exibir avatar e capa em todas as superfícies de ficha (lista própria, detalhe, sidebars de campanha e partida), padronizando o nome dos campos para `avatar_url`/`cover_url` (JSON) → `avatarUrl`/`coverUrl` (TypeScript).

**Architecture:** O backend passa a incluir `avatar_url`/`cover_url` nos summaries retornados por todos os endpoints que listam fichas; o frontend renomeia os campos existentes e refatora `CharacterSidebarItem` e `EnrollmentSidebarItem` para usar `CharacterSheetHeader` (modo `"card"`) com uma nova prop `showStatus` que oculta as barras de HP/SP quando o dado não existe.

**Tech Stack:** Go 1.23 (pgx/v5, huma/v2) — React/Vite/TypeScript — styled-components — React Query

---

## Repos e convenções

- Backend: `System_X_System/` — commits com `Co-authored-by: ...`
- Frontend: `System_X_System_React/` — commits com `Co-authored-by: ...`
- Verificação backend: `go vet -tags=integration ./internal/gateway/pg/...` após cada mudança de gateway; `go vet ./internal/...` após mudanças de API handler
- Verificação frontend: `npm run build` (executa `tsc -b && vite build`; falha em erros de tipo)
- Lint frontend: `npm run lint`

---

## File Map

### Backend — modificar

| Arquivo | O que muda |
|---------|------------|
| `internal/domain/entity/character_sheet/summary.go` | +`AvatarURL *string`, `CoverURL *string` |
| `internal/gateway/pg/sheet/read_character_sheet.go` | `ListCharacterSheetsByPlayerUUID`: +`cp.avatar_url, cp.cover_url` no SELECT + Scan |
| `internal/gateway/pg/campaign/read_campaign.go` | `pendingSheetsQuery` + `characterSheetsQuery`: idem |
| `internal/gateway/pg/enrollment/list_by_match_uuid.go` | `ListByMatchUUID`: idem |
| `internal/gateway/pg/match/read_participants.go` | `ListParticipantsByMatchUUID`: idem |
| `internal/app/api/sheet/character_sheet_sumary_response.go` | +`AvatarURL`, `CoverURL` em `CharacterBaseSummaryResponse`; atualizar `ToBaseSummaryResponse` |
| `docs/dev/api/character-sheet.md` | Documentar `avatar_url`/`cover_url` nos summaries |

### Frontend — modificar

| Arquivo | O que muda |
|---------|------------|
| `src/types/characterSheet.ts` | `Profile`: `cover`→`coverUrl`, `avatar`→`avatarUrl`; `CharacterSheetSummary`: idem; `CharacterBaseSummary`: +`avatarUrl?`, `coverUrl?` |
| `src/features/sheet/factories/profile.factory.ts` | `cover`→`coverUrl`, `avatar`→`avatarUrl` |
| `src/components/molecules/CharacterSheetHeader.tsx` | `profile?.cover`→`profile?.coverUrl`, `profile?.avatar`→`profile?.avatarUrl`; +prop `showStatus?: boolean` |
| `src/components/atoms/CharacterSheetCard.tsx` | `cover: character?.cover`→`coverUrl: character?.coverUrl`; `avatar: character?.avatar`→`avatarUrl: character?.avatarUrl` |
| `src/features/campaign/CharacterSidebarItem.tsx` | Refatorar: usar `CharacterSheetHeader` mode `"card"` + badges |
| `src/features/match/EnrollmentSidebarItem.tsx` | Refatorar: idem |

---

## Task 1 — Backend: adicionar `AvatarURL`/`CoverURL` ao domain `Summary`

**Arquivo:** `internal/domain/entity/character_sheet/summary.go`

- [ ] **1.1 Adicionar os campos ao struct**

Abrir `internal/domain/entity/character_sheet/summary.go`. O arquivo atual termina com `UpdatedAt time.Time`. Adicionar os dois campos ao final do struct `Summary`:

```go
type Summary struct {
	ID             int
	UUID           uuid.UUID
	PlayerUUID     *uuid.UUID
	MasterUUID     *uuid.UUID
	CampaignUUID   *uuid.UUID
	NickName       string
	FullName       string
	Alignment      string
	CharacterClass string
	Birthday       time.Time
	CategoryName   string
	CurrHexValue   *int
	Level          int
	Points         int
	TalentLvl      int
	PhysicalsLvl   int
	MentalsLvl     int
	SpiritualsLvl  int
	SkillsLvl      int
	Stamina        StatusBar
	Health         StatusBar
	Aura           StatusBar
	StoryStartAt   *time.Time
	StoryCurrentAt *time.Time
	DeadAt         *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	AvatarURL      *string
	CoverURL       *string
}
```

- [ ] **1.2 Verificar compilação**

```bash
cd System_X_System
go vet ./internal/domain/...
```

Saída esperada: nenhum erro.

- [ ] **1.3 Commit**

```bash
git add internal/domain/entity/character_sheet/summary.go
git commit -m "$(cat <<'EOF'
feat(domain): add AvatarURL and CoverURL to character sheet Summary

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 2 — Backend: adicionar `avatar_url`/`cover_url` nas 4 queries de gateway

**Arquivos:**
- `internal/gateway/pg/sheet/read_character_sheet.go`
- `internal/gateway/pg/campaign/read_campaign.go`
- `internal/gateway/pg/enrollment/list_by_match_uuid.go`
- `internal/gateway/pg/match/read_participants.go`

O padrão é sempre o mesmo: adicionar `cp.avatar_url, cp.cover_url` no final do SELECT (antes do FROM/JOIN), e `&s.AvatarURL, &s.CoverURL` no final do Scan correspondente.

- [ ] **2.1 `read_character_sheet.go` — `ListCharacterSheetsByPlayerUUID`**

Localizar a const `query` dentro de `ListCharacterSheetsByPlayerUUID` (linha ~189). Substituir o trecho de SELECT + Scan:

```go
// SELECT — trocar a linha que termina com cp.birthday por:
cp.nickname, cp.fullname, cp.alignment, cp.character_class, cp.birthday,
cp.avatar_url, cp.cover_url
```

```go
// Scan — trocar a linha que termina com &s.Birthday por:
&s.NickName, &s.FullName, &s.Alignment, &s.CharacterClass, &s.Birthday,
&s.AvatarURL, &s.CoverURL,
```

- [ ] **2.2 `read_campaign.go` — `pendingSheetsQuery`**

Localizar `pendingSheetsQuery` (~linha 67). Substituir o trecho de SELECT + Scan:

```go
// SELECT — trocar linha que termina com cp.birthday por:
cp.nickname, cp.fullname, cp.alignment, cp.character_class, cp.birthday,
cp.avatar_url, cp.cover_url
```

```go
// Scan — trocar linha que termina com &sheet.Birthday por:
&sheet.NickName, &sheet.FullName, &sheet.Alignment, &sheet.CharacterClass, &sheet.Birthday,
&sheet.AvatarURL, &sheet.CoverURL,
```

- [ ] **2.3 `read_campaign.go` — `characterSheetsQuery`**

Mesma mudança na const `characterSheetsQuery` (~linha 116):

```go
// SELECT
cp.nickname, cp.fullname, cp.alignment, cp.character_class, cp.birthday,
cp.avatar_url, cp.cover_url
```

```go
// Scan
&sheet.NickName, &sheet.FullName, &sheet.Alignment, &sheet.CharacterClass, &sheet.Birthday,
&sheet.AvatarURL, &sheet.CoverURL,
```

- [ ] **2.4 `list_by_match_uuid.go` — `ListByMatchUUID`**

Localizar a const `query` em `ListByMatchUUID` (~linha 14). Substituir:

```go
// SELECT — adicionar após cp.birthday (antes de u.uuid):
cp.nickname, cp.fullname, cp.alignment, cp.character_class, cp.birthday,
cp.avatar_url, cp.cover_url,
u.uuid, u.nick
```

```go
// Scan — adicionar após &s.Birthday (antes de &e.Player.UUID):
&s.NickName, &s.FullName, &s.Alignment, &s.CharacterClass, &s.Birthday,
&s.AvatarURL, &s.CoverURL,
&e.Player.UUID, &e.Player.Nick,
```

- [ ] **2.5 `read_participants.go` — `ListParticipantsByMatchUUID`**

Localizar a const `query` em `ListParticipantsByMatchUUID`. Substituir:

```go
// SELECT — adicionar após cp.birthday (antes de cs.story_start_at):
cp.nickname, cp.fullname, cp.alignment, cp.character_class, cp.birthday,
cp.avatar_url, cp.cover_url,
cs.story_start_at, cs.story_current_at, cs.dead_at
```

```go
// Scan — adicionar após &s.Birthday (antes de &s.StoryStartAt):
&s.NickName, &s.FullName, &s.Alignment, &s.CharacterClass, &s.Birthday,
&s.AvatarURL, &s.CoverURL,
&s.StoryStartAt, &s.StoryCurrentAt, &s.DeadAt,
```

- [ ] **2.6 Verificar compilação com integration tags**

```bash
go vet -tags=integration ./internal/gateway/pg/...
```

Saída esperada: nenhum erro.

- [ ] **2.7 Commit**

```bash
git add \
  internal/gateway/pg/sheet/read_character_sheet.go \
  internal/gateway/pg/campaign/read_campaign.go \
  internal/gateway/pg/enrollment/list_by_match_uuid.go \
  internal/gateway/pg/match/read_participants.go
git commit -m "$(cat <<'EOF'
feat(gateway): include avatar_url and cover_url in all character sheet summary queries

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 3 — Backend: expor `avatar_url`/`cover_url` na response da API

**Arquivo:** `internal/app/api/sheet/character_sheet_sumary_response.go`

- [ ] **3.1 Adicionar campos em `CharacterBaseSummaryResponse`**

Substituir o struct `CharacterBaseSummaryResponse` completo (adicionar os dois últimos campos):

```go
type CharacterBaseSummaryResponse struct {
	UUID           uuid.UUID  `json:"uuid"`
	PlayerUUID     *uuid.UUID `json:"player_uuid,omitempty"`
	MasterUUID     *uuid.UUID `json:"master_uuid,omitempty"`
	CampaignUUID   *uuid.UUID `json:"campaign_uuid,omitempty"`
	NickName       string     `json:"nick_name"`
	StoryStartAt   *string    `json:"story_start_at,omitempty"`
	StoryCurrentAt *string    `json:"story_current_at,omitempty"`
	DeadAt         *string    `json:"dead_at,omitempty"`
	CreatedAt      string     `json:"created_at"`
	UpdatedAt      string     `json:"updated_at"`
	AvatarURL      *string    `json:"avatar_url,omitempty"`
	CoverURL       *string    `json:"cover_url,omitempty"`
}
```

- [ ] **3.2 Popular os novos campos em `ToBaseSummaryResponse`**

Substituir o `return` de `ToBaseSummaryResponse` (adicionar as duas últimas linhas):

```go
return CharacterBaseSummaryResponse{
	UUID:           sheet.UUID,
	PlayerUUID:     sheet.PlayerUUID,
	MasterUUID:     sheet.MasterUUID,
	CampaignUUID:   sheet.CampaignUUID,
	NickName:       sheet.NickName,
	StoryStartAt:   storyStartAtStr,
	StoryCurrentAt: storyCurrentAtStr,
	DeadAt:         deadAtStr,
	CreatedAt:      sheet.CreatedAt.Format(time.RFC3339),
	UpdatedAt:      sheet.UpdatedAt.Format(time.RFC3339),
	AvatarURL:      sheet.AvatarURL,
	CoverURL:       sheet.CoverURL,
}
```

- [ ] **3.3 Verificar compilação**

```bash
go vet ./internal/app/api/...
```

Saída esperada: nenhum erro.

- [ ] **3.4 Commit**

```bash
git add internal/app/api/sheet/character_sheet_sumary_response.go
git commit -m "$(cat <<'EOF'
feat(api): expose avatar_url and cover_url in CharacterBaseSummaryResponse

Propagates to CharacterPrivateSummaryResponse and CharacterPublicSummaryResponse
via embedding. Covers /charactersheets, /campaigns/:id, /matches/:id endpoints.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 4 — Backend: atualizar contrato de API

**Arquivo:** `docs/dev/api/character-sheet.md`

- [ ] **4.1 Adicionar nota de `avatar_url`/`cover_url` nos summaries**

Adicionar ao final do arquivo:

```markdown
---

## Campos de imagem nos summaries

Os endpoints que retornam listas de fichas (`GET /charactersheets`,
`GET /campaigns/:id`, enrollments e participants de partida) incluem
`avatar_url` e `cover_url` opcionais em cada summary de ficha:

```json
{
  "uuid": "...",
  "nick_name": "Gon",
  "avatar_url": "https://pub.r2.dev/avatar/uuid.webp",
  "cover_url": "https://pub.r2.dev/cover/uuid.webp"
}
```

Ambos são `omitempty` — ausentes quando o personagem ainda não tem imagem.
```

- [ ] **4.2 Commit**

```bash
git add docs/dev/api/character-sheet.md
git commit -m "$(cat <<'EOF'
docs(api): document avatar_url and cover_url in character sheet summaries

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 5 — Frontend: renomear campos `cover`/`avatar` → `coverUrl`/`avatarUrl`

**Arquivos:**
- `src/types/characterSheet.ts`
- `src/features/sheet/factories/profile.factory.ts`

Esta task atualiza os tipos e a factory. Após ela, o TypeScript vai apontar erros em todos os usos dos nomes antigos — esses erros são resolvidos nas Tasks 6 e 7.

- [ ] **5.1 Atualizar `src/types/characterSheet.ts`**

Substituir a interface `Profile` completa:

```ts
export type Profile = {
  nickname: string;
  fullname: string;
  description?: string;
  briefDescription: string;
  birthday: string;
  age: number;
  alignment: string;
  coverUrl?: string;
  avatarUrl?: string;
};
```

Substituir `CharacterSheetSummary` — apenas os dois campos renomeados:

```ts
export interface CharacterSheetSummary {
  uuid: string;
  playerUUID: string;
  masterUUID: string;
  campaignUUID: string;
  nickName: string;
  fullName: string;
  alignment: string;
  birthday?: string;
  age: number;
  coverUrl?: string;
  avatarUrl?: string;
  characterClass: string;
  categoryName: string;
  currHexValue: number | null;
  level: number;
  points: number;
  talentLvl: number;
  physicalsLvl: number;
  mentalsLvl: number;
  spiritualsLvl: number;
  skillsLvl: number;
  stamina: StatusBar;
  health: StatusBar;
  aura: StatusBar;
  createdAt: string;
  updatedAt: string;
}
```

Adicionar `avatarUrl`/`coverUrl` em `CharacterBaseSummary`:

```ts
export interface CharacterBaseSummary {
  uuid: string;
  playerUuid?: string;
  masterUuid?: string;
  campaignUuid?: string;
  nickName: string;
  avatarUrl?: string;
  coverUrl?: string;
  storyStartAt?: string;
  storyCurrentAt?: string;
  deadAt?: string;
  createdAt: string;
  updatedAt: string;
}
```

- [ ] **5.2 Atualizar `src/features/sheet/factories/profile.factory.ts`**

Substituir o conteúdo completo:

```ts
import type { Profile } from "../../../types/characterSheet";

export function createEmptyProfile(): Profile {
  const today = new Date();
  const mm = String(today.getMonth() + 1).padStart(2, "0");
  const dd = String(today.getDate()).padStart(2, "0");
  return {
    nickname: "",
    fullname: "",
    alignment: "",
    description: "",
    briefDescription: "",
    birthday: `0000-${mm}-${dd}T00:00:00.000Z`,
    age: 0,
    coverUrl: undefined,
    avatarUrl: undefined,
  };
}
```

- [ ] **5.3 Verificar erros de tipo (esperado que falhe)**

```bash
cd System_X_System_React
npm run build 2>&1 | grep "error TS"
```

Saída esperada: erros em `CharacterSheetHeader.tsx` e `CharacterSheetCard.tsx` referenciando `profile.cover`, `profile.avatar`, `character.cover`, `character.avatar`. Esses são corrigidos nas Tasks 6 e 7.

- [ ] **5.4 Commit parcial (só os dois arquivos desta task)**

```bash
git add src/types/characterSheet.ts src/features/sheet/factories/profile.factory.ts
git commit -m "$(cat <<'EOF'
feat(types): rename cover/avatar to coverUrl/avatarUrl; add fields to CharacterBaseSummary

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 6 — Frontend: atualizar `CharacterSheetHeader` (renomear + prop `showStatus`)

**Arquivo:** `src/components/molecules/CharacterSheetHeader.tsx`

- [ ] **6.1 Adicionar `showStatus` à interface e renomear referências de imagem**

Substituir a interface `CharacterSheetHeaderProps`:

```tsx
interface CharacterSheetHeaderProps {
  data: Data;
  mode: HeaderMode;
  showStatus?: boolean;
}
```

Atualizar a desestruturação do componente:

```tsx
export default function CharacterSheetHeader({
  mode,
  showStatus = true,
  data: { charSheet, setCharSheet, charClasses, onAvatarSelected, onCoverSelected },
}: CharacterSheetHeaderProps) {
```

Substituir a linha do `Cover`:

```tsx
<Cover src={profile?.coverUrl || coverPlaceholder} alt={`cover`} />
```

Substituir a linha do `Avatar`:

```tsx
<Avatar src={profile?.avatarUrl || avatarPlaceholder} alt={`avatar`} />
```

Substituir o bloco `<StatusBarsContainer>` (torná-lo condicional):

```tsx
{showStatus && (
  <StatusBarsContainer>
    <HpBar current={health?.current} max={health?.max} />
    <SpBar current={stamina?.current} max={stamina?.max} />
  </StatusBarsContainer>
)}
```

- [ ] **6.2 Verificar compilação**

```bash
npm run build 2>&1 | grep "error TS"
```

Saída esperada: ainda há erro em `CharacterSheetCard.tsx` (resolvido na Task 7).

- [ ] **6.3 Commit**

```bash
git add src/components/molecules/CharacterSheetHeader.tsx
git commit -m "$(cat <<'EOF'
feat(CharacterSheetHeader): rename cover/avatar to coverUrl/avatarUrl; add showStatus prop

showStatus=false hides the HP/SP bars (used by sidebar items for non-owner/non-master viewers).

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 7 — Frontend: atualizar `CharacterSheetCard`

**Arquivo:** `src/components/atoms/CharacterSheetCard.tsx`

- [ ] **7.1 Renomear `cover`/`avatar` no profile que é passado ao charSheet**

Substituir o bloco de atribuição do `charSheet.profile`:

```tsx
charSheet.profile = {
  nickname: character.nickName,
  fullname: character.fullName,
  age: character.age,
  briefDescription: "",
  birthday: character.birthday ?? "0000-01-01T00:00:00.000Z",
  alignment: character.alignment,
  coverUrl: character?.coverUrl,
  avatarUrl: character?.avatarUrl,
};
```

- [ ] **7.2 Verificar compilação limpa**

```bash
npm run build
```

Saída esperada: build bem-sucedido, zero erros de tipo.

- [ ] **7.3 Commit**

```bash
git add src/components/atoms/CharacterSheetCard.tsx
git commit -m "$(cat <<'EOF'
fix(CharacterSheetCard): use coverUrl/avatarUrl from CharacterSheetSummary

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 8 — Frontend: refatorar `CharacterSidebarItem`

**Arquivo:** `src/features/campaign/CharacterSidebarItem.tsx`

Substitui o conteúdo textual atual por `CharacterSheetHeader` em modo `"card"`. O `ItemContainer` mantém as bordas coloridas por estado e as badges em posição absoluta. `showStatus` recebe `true` apenas quando `character.health` existe (mestre ou quando é a própria ficha do jogador).

- [ ] **8.1 Substituir o conteúdo completo do arquivo**

```tsx
import styled from "styled-components";
import type { CharacterBaseSummary, StatusBar } from "../../types/characterSheet";
import { createEmptyCharacterSheet } from "../../features/sheet/factories/characterSheet.factory";
import CharacterSheetHeader from "../../components/molecules/CharacterSheetHeader";

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

export default function CharacterSidebarItem({
  character,
  isMaster,
  hasLeft,
  onClick,
}: CharacterSidebarItemProps) {
  const isDead = !!character.deadAt;
  const isPending = !!character.isPending;
  const isNpc = !character.playerUuid;

  const charSheet = createEmptyCharacterSheet();
  charSheet.characterClass = character.characterClass ?? "";
  charSheet.profile = {
    ...charSheet.profile,
    nickname: character.nickName,
    fullname: character.fullName ?? "",
    coverUrl: character.coverUrl,
    avatarUrl: character.avatarUrl,
  };
  if (character.health && character.stamina) {
    charSheet.status = {
      health: character.health,
      stamina: character.stamina,
    };
  }

  return (
    <ItemContainer
      $isDead={isDead}
      $isPending={isPending}
      $isNpc={isNpc}
      $clickable={isMaster}
      onClick={isMaster ? onClick : undefined}
    >
      <CharacterSheetHeader
        mode="card"
        data={{ charSheet }}
        showStatus={!!character.health}
      />
      {isPending && <PendingBadge>Pendente</PendingBadge>}
      {isNpc && <NpcBadge>NPC</NpcBadge>}
      {isDead && <DeadBadge>Morto</DeadBadge>}
      {hasLeft && <LeftBadge>Saiu</LeftBadge>}
    </ItemContainer>
  );
}

const ItemContainer = styled.div<{
  $isDead?: boolean;
  $isPending?: boolean;
  $isNpc?: boolean;
  $clickable?: boolean;
}>`
  position: relative;
  overflow: hidden;
  border-radius: 8px;
  opacity: ${({ $isDead }) => ($isDead ? 0.7 : 1)};
  cursor: ${({ $clickable }) => ($clickable ? "pointer" : "default")};
  border-left: 4px solid
    ${({ $isDead, $isPending, $isNpc }) =>
      $isDead
        ? "#e74c3c"
        : $isPending
        ? "#3498db"
        : $isNpc
        ? "#2ecc71"
        : "#ffa216"};

  &:hover {
    filter: ${({ $clickable }) => ($clickable ? "brightness(1.05)" : "none")};
  }
`;

const Badge = styled.span`
  position: absolute;
  top: 10px;
  right: 10px;
  border-radius: 4px;
  padding: 2px 6px;
  font-size: 12px;
  font-weight: bold;
  z-index: 10;
`;

const PendingBadge = styled(Badge)`
  background-color: #3498db;
  color: white;
`;

const DeadBadge = styled(Badge)`
  background-color: #e74c3c;
  color: white;
`;

const NpcBadge = styled(Badge)`
  background-color: #2ecc71;
  color: white;
`;

const LeftBadge = styled(Badge)`
  background-color: #555;
  color: #ccc;
`;
```

- [ ] **8.2 Verificar compilação**

```bash
npm run build
```

Saída esperada: build limpo.

- [ ] **8.3 Verificar lint**

```bash
npm run lint
```

Saída esperada: zero erros.

- [ ] **8.4 Commit**

```bash
git add src/features/campaign/CharacterSidebarItem.tsx
git commit -m "$(cat <<'EOF'
feat(CharacterSidebarItem): replace text layout with CharacterSheetHeader in card mode

HP/SP bars shown only when health data is present (master or sheet owner).
Badges remain absolutely positioned above the header.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 9 — Frontend: refatorar `EnrollmentSidebarItem`

**Arquivo:** `src/features/match/EnrollmentSidebarItem.tsx`

O enrollment tem um `characterSheet: CharacterSheetWithVisibility` onde `private` existe apenas quando o viewer é o mestre. O status badge (Pendente/Aceito/Rejeitado) fica posicionado absolutamente no topo direito, sobre o header. Os botões de aceitar/rejeitar ficam abaixo do header (não são absolutamente posicionados para não sobreapor o header).

- [ ] **9.1 Substituir o conteúdo completo do arquivo**

```tsx
import styled from "styled-components";
import type { Enrollment } from "../../types/match";
import { createEmptyCharacterSheet } from "../../features/sheet/factories/characterSheet.factory";
import CharacterSheetHeader from "../../components/molecules/CharacterSheetHeader";

interface EnrollmentSidebarItemProps {
  enrollment: Enrollment;
  isMaster: boolean;
  isLoading: boolean;
  onAccept: (enrollmentId: string) => void;
  onReject: (enrollmentId: string) => void;
  onClick: () => void;
}

export default function EnrollmentSidebarItem({
  enrollment,
  isMaster,
  isLoading,
  onAccept,
  onReject,
  onClick,
}: EnrollmentSidebarItemProps) {
  const { characterSheet, status } = enrollment;
  const priv = characterSheet.private;

  const charSheet = createEmptyCharacterSheet();
  charSheet.characterClass = priv?.characterClass ?? "";
  charSheet.profile = {
    ...charSheet.profile,
    nickname: characterSheet.nickName,
    fullname: priv?.fullName ?? "",
    coverUrl: characterSheet.coverUrl,
    avatarUrl: characterSheet.avatarUrl,
  };
  if (priv?.health && priv?.stamina) {
    charSheet.status = {
      health: priv.health,
      stamina: priv.stamina,
    };
  }

  return (
    <ItemContainer $clickable={isMaster} onClick={isMaster ? onClick : undefined}>
      <CharacterSheetHeader
        mode="card"
        data={{ charSheet }}
        showStatus={!!priv}
      />
      <StatusBadge $status={status}>
        {status === "pending" && "Pendente"}
        {status === "accepted" && "Aceito"}
        {status === "rejected" && "Rejeitado"}
      </StatusBadge>

      {isMaster && (
        <Actions>
          <ActionButton
            $variant="accept"
            disabled={isLoading}
            onClick={(e) => {
              e.stopPropagation();
              onAccept(enrollment.uuid);
            }}
          >
            ✓
          </ActionButton>
          <ActionButton
            $variant="reject"
            disabled={isLoading}
            onClick={(e) => {
              e.stopPropagation();
              onReject(enrollment.uuid);
            }}
          >
            ✗
          </ActionButton>
        </Actions>
      )}
    </ItemContainer>
  );
}

const ItemContainer = styled.div<{ $clickable: boolean }>`
  position: relative;
  overflow: hidden;
  border-radius: 8px;
  border-left: 4px solid #ffa216;
  cursor: ${({ $clickable }) => ($clickable ? "pointer" : "default")};

  &:hover {
    filter: ${({ $clickable }) => ($clickable ? "brightness(1.05)" : "none")};
  }
`;

const StatusBadge = styled.span<{ $status: string }>`
  position: absolute;
  top: 10px;
  right: 10px;
  z-index: 10;
  border-radius: 4px;
  padding: 2px 8px;
  font-size: 12px;
  font-weight: bold;
  background-color: ${({ $status }) =>
    $status === "pending"
      ? "#3498db"
      : $status === "accepted"
      ? "#2ecc71"
      : "#e74c3c"};
  color: white;
`;

const Actions = styled.div`
  display: flex;
  gap: 6px;
  padding: 10px 15px;
  background-color: rgba(0, 0, 0, 0.6);
`;

const ActionButton = styled.button<{ $variant: "accept" | "reject" }>`
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 6px;
  font-size: 16px;
  font-weight: bold;
  cursor: pointer;
  background-color: ${({ $variant }) =>
    $variant === "accept" ? "#27ae60" : "#c0392b"};
  color: white;
  transition: filter 0.2s;

  &:hover:not(:disabled) {
    filter: brightness(1.15);
  }
  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
`;
```

- [ ] **9.2 Verificar compilação**

```bash
npm run build
```

Saída esperada: build limpo.

- [ ] **9.3 Verificar lint**

```bash
npm run lint
```

Saída esperada: zero erros.

- [ ] **9.4 Commit**

```bash
git add src/features/match/EnrollmentSidebarItem.tsx
git commit -m "$(cat <<'EOF'
feat(EnrollmentSidebarItem): replace text layout with CharacterSheetHeader in card mode

HP/SP shown only when master (private data present). Status badge and action
buttons remain visible above/below the header.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Verificação Final

Após todas as tasks, testar manualmente as 4 superfícies:

- [ ] **Lista de fichas próprias** (`/charactersheets`): cards mostram capa e avatar vindos da API
- [ ] **Detalhe da ficha** (`/charactersheets/:id`): capa e avatar aparecem (não mais placeholders quando há imagem)
- [ ] **Sidebar de campanha** (mestre): header com capa, avatar, HP/SP visíveis para todos os personagens
- [ ] **Sidebar de campanha** (jogador): header com capa e avatar; HP/SP ocultos para outros personagens
- [ ] **Sidebar de partida** (enrollment e participant): idem às regras de campanha

---

## Notas de implementação

- `CharacterSheetHeader` usa container queries (`cqi`) — escala automaticamente conforme a largura do `ItemContainer`. Nenhum ajuste manual de tamanho necessário.
- `CharacterBaseSummary` é re-exportado de `src/types/campaign.ts` via re-export — a mudança em `characterSheet.ts` propaga sem alterações adicionais.
- A prop `showStatus` tem default `true` — todos os usos existentes de `CharacterSheetHeader` (detalhe, lista, create, edit) não precisam passar a prop.
