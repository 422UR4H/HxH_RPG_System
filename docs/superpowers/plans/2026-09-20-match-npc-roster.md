# Plano — Rostering de NPC em partida

Branch: `feat/match-npc-roster` (a partir de `main`).
Spec de fronteira: `docs/superpowers/specs/2026-09-20-front-combat-phases.md` §13,
entrada "Sem NPC não há inimigo" (não está nesta branch; o essencial foi copiado abaixo).

## Problema

Não existe NPC em partida. `internal/gateway/pg/match/start_match.go` popula
`match_participants` apenas a partir de inscrições (`enrollments`) aceitas, então o mestre
não tem como pôr um inimigo em campo. As fichas de NPC já existem — são fichas de campanha,
com `master_uuid` preenchido e `player_uuid` nulo — e o front até as desenha como peças, mas
a `MatchSession` não sabe que elas existem.

## Solução

1. Caminho **REST** para o mestre pôr uma ficha de NPC na partida, sem inscrição: escreve em
   `match_participants`. REST e não WebSocket porque é ato de preparação, e porque mantém a
   sobreposição com o PR paralelo (pacote de preparativos da Fase 6) perto de zero.
2. `indexParticipants` (`internal/domain/match/matchsession/match_session.go`) passa a mapear
   NPC → mestre no `charToPlayer`, e `EnqueueAction` passa a aceitar a posse via `charToPlayer`
   como prova de participação.
3. Remoção de NPC da partida (hard delete).

Como este PR escreve em `match_participants` — o roster autoritativo —, o NPC sobrevive a
reinício de graça: `InitMatchSessionUC` recarrega dali.

## Global Constraints

Vinculantes para toda task. O reviewer lê esta seção.

- **G1 — Arquivos proibidos.** Não toque em `internal/app/game/message.go`,
  `internal/app/game/room.go`, `internal/domain/match/service/projection.go`,
  `internal/domain/match/service/damage.go`, `internal/domain/match/service/turn_resolver.go`,
  nem em `persist_turn_close` (`internal/gateway/pg/round/`). São do PR paralelo.
- **G2 — Sem reorganização.** Não reordene, reformate nem "limpe" bloco existente em arquivo
  compartilhado. É o que transforma conflito de 2 linhas em conflito de arquivo inteiro.
  Edições em arquivo existente são **aditivas** e no fim do bloco correspondente, exceto
  onde a task manda o contrário.
- **G3 — Sem refactor do carregamento de fichas.** Não mexa em
  `internal/application/match/init_match_session.go`. A ficha já é construída pela Factory
  (`internal/gateway/pg/sheet/read_character_sheet.go:136`, `factory.Build`) e o Init já é a
  resiliência a reinício.
- **G4 — Sem catálogo de NPCs.** A lista de quem *poderia* entrar não entra na `MatchSession`
  nem em endpoint novo: ela já vem de `campaign.characterSheets`, que é de onde o front monta
  o `npcMap`. A `MatchSession` guarda quem **está** na partida.
- **G5 — Front fora de escopo.** Nenhuma alteração em `System_X_System_React/`.
- **G6 — Convenções da casa.** Go idiomático, erros com `%w`. Testes com `testing` puro,
  table-driven com `t.Run()`, pacote de teste externo (`package foo_test`). Mocks em
  `mocks_test.go` por pacote de handler. **Nunca remova comentário `TODO`.**
- **G7 — Sem tag de validação do Huma** (`maxLength`, `minLength`, `enum`) em campo de body:
  o Huma intercepta antes do handler e devolve `"validation failed"` sem contexto. Validação
  de negócio fica no use case; o handler mapeia `domain.ErrValidation` →
  `huma.Error422UnprocessableEntity(err.Error())`. Tags permitidas: `required`, `doc`,
  `default`, `json`, `path`.
- **G8 — Wire format camelCase** nos dois lados. Tags `json` manuais em camelCase.
- **G9 — Verificação por task.** Ao fim de cada task, rode `go build ./...` e
  `go vet ./...`, `go vet -tags integration ./...` e `go vet -tags smoke ./...`
  (`go vet ./...` não enxerga arquivo com build tag). Testes de integração rodam com
  `-p 1` — em paralelo os pacotes compartilham o banco e se truncam (problema pré-existente).

## Decisões já tomadas (não reabra)

- **D1.** O endpoint exige `player_uuid IS NULL` na ficha. É endpoint de NPC. Aceitar ficha
  de jogador abriria backdoor no fluxo de enrollment: `indexParticipants` mapearia a ficha
  para aquele jogador, que passaria a agir por ela sem inscrição.
- **D2.** Autorização da ficha: `master_uuid == requisitante` **OU**
  `campaign_uuid == campanha da partida`.
- **D3.** Remoção é hard delete. `match_participants.left_at` existe, mas
  `ListParticipantsByMatchUUID` não filtra por ele — soft delete não tiraria o NPC da sessão,
  e passar a filtrar mudaria o comportamento para jogadores também, o que ninguém pediu.
- **D4.** A guarda de NPC vai **no SQL**, não só no use case. Um bug na camada de cima não
  pode conseguir apagar linha de jogador.
- **D5.** Não há guarda de `game_start_at`: pôr NPC é ato de preparação e vale antes e durante
  a partida. Há guarda de `story_end_at` (partida encerrada não recebe NPC).
- **D6.** Propagação ao vivo **não existe e não é deste PR**. `cmd/api` (:5000) e `cmd/game`
  (:8081) são processos separados; a `MatchSession` só nasce em `Room.StartMatch` ou no
  rehydrate de `internal/app/game/handler.go:184` quando `r.session == nil`. O NPC aparece
  quando a sala renasce. Um verbo de WS no game server vira fatia própria depois que o PR
  paralelo mergear.
- **D7.** Sem migration. `match_participants` já existe com
  `UNIQUE (match_uuid, character_sheet_uuid)`.

## Fatos do código (verificados, use sem reconferir)

- `internal/domain/entity/character_sheet/summary.go:15` — `Summary` tem `PlayerUUID`,
  `MasterUUID`, `CampaignUUID`, todos `*uuid.UUID`.
- `internal/gateway/pg/match/read_participants.go` — `ListParticipantsByMatchUUID` já
  seleciona `cs.master_uuid` e o escaneia em `s.MasterUUID`. Não precisa mudar.
- `internal/gateway/pg/sheet/read_character_sheet.go:243` —
  `GetCharacterSheetRelationshipUUIDs(ctx, sheetUUID) (csEntity.RelationshipUUIDs, error)`
  devolve `PlayerUUID`, `MasterUUID`, `CampaignUUID` e erra com
  `charactersheet.ErrCharacterSheetNotFound` (de `internal/application/character_sheet`)
  quando não acha.
- `internal/domain/match/match.go:10` — `Match` tem `MasterUUID`, `CampaignUUID`,
  `GameStartAt *time.Time`, `StoryEndAt *time.Time`.
- `internal/application/match/error.go` — já tem `ErrMatchNotFound`, `ErrNotMatchMaster`,
  `ErrMatchAlreadyFinished`, todos via `domain.NewValidationError(...)`.
- `internal/gateway/pg/match/error.go` — só tem `ErrMatchNotFound`.
- `internal/application/match/i_repository.go` — `IRepository` é a interface grande do
  repositório de partida. **Não a altere**: `internal/application/testutil/mock_match_repo.go`
  a implementa e é compartilhado. Use interfaces locais estreitas, como
  `CampaignParticipationChecker` em `internal/application/match/list_match_enrollments.go:24`.
- `migrations/20260504000000_add_match_participants_table.sql` — colunas
  `id, uuid, match_uuid, character_sheet_uuid, joined_at, left_at, created_at, updated_at`.
- `internal/gateway/pg/pgtest` — helpers `SetupTestDB`, `TruncateAll`, `InsertTestUser`,
  `InsertTestScenario`, `InsertTestCampaign`, `InsertTestMatch`, `InsertTestEnrollment`,
  `InsertTestMatchParticipant`, `InsertTestCharacterSheet(t, pool, playerUUID *string,
  masterUUID *string, campaignUUID *string, nick string) string`.
- Padrão de handler REST espelhável: `internal/app/api/matchmap/attach.go` (lê
  `auth.UserIDKey` do contexto, mapeia erro do UC para `huma.Error4xx`).

---

## Task 1 — Gateway: escrita e remoção do NPC no roster

**Arquivos:** cria `internal/gateway/pg/match/add_npc_participant.go` e
`internal/gateway/pg/match/remove_npc_participant.go`; edita (aditivo)
`internal/gateway/pg/match/error.go`; cria
`internal/gateway/pg/match/npc_roster_integration_test.go`.

Não edite `internal/gateway/pg/match/repository.go` a não ser que o build exija.

### Erros (aditivo em `error.go`, mantenha `ErrMatchNotFound` onde está)

```go
ErrNPCAlreadyInMatch = errors.New("npc already in match")
ErrNPCNotInMatch     = errors.New("npc not found in match")
```

### `AddNPCParticipant`

```go
func (r *Repository) AddNPCParticipant(
	ctx context.Context, matchUUID, sheetUUID uuid.UUID, joinedAt time.Time,
) (*matchEntity.Participant, error)
```

Um `INSERT` só, com a guarda de NPC no próprio SQL (D4) — a ficha tem de ter
`player_uuid IS NULL`, e só entra se a partida existir:

```sql
INSERT INTO match_participants
	(uuid, match_uuid, character_sheet_uuid, joined_at, created_at, updated_at)
SELECT gen_random_uuid(), $1, cs.uuid, $3, $4, $4
FROM character_sheets cs
WHERE cs.uuid = $2 AND cs.player_uuid IS NULL
ON CONFLICT (match_uuid, character_sheet_uuid) DO NOTHING
RETURNING uuid, match_uuid, character_sheet_uuid, joined_at, created_at, updated_at
```

Semântica do retorno — o `SELECT ... WHERE` e o `ON CONFLICT` fazem as duas causas de
"nenhuma linha" colapsarem em `pgx.ErrNoRows`, e elas precisam de erros diferentes. Quando
`QueryRow(...).Scan(...)` devolver `pgx.ErrNoRows`, faça **uma** consulta de desambiguação:

```sql
SELECT EXISTS (
	SELECT 1 FROM match_participants
	WHERE match_uuid = $1 AND character_sheet_uuid = $2
)
```

`true` → `ErrNPCAlreadyInMatch`. `false` → `ErrNPCNotInMatch` **não** serve aqui; devolva
`pgx.ErrNoRows` embrulhado com `%w` num erro que diga que a ficha não é NPC elegível:
adicione também

```go
ErrSheetNotEligibleNPC = errors.New("character sheet is not an npc")
```

e devolva `ErrSheetNotEligibleNPC`.

O `Participant` devolvido carrega só identidade — `UUID`, `MatchUUID`, `JoinedAt`,
`CreatedAt`, `UpdatedAt`, e `Sheet.UUID` recebendo `character_sheet_uuid`. Não preencha o
resto do `Summary`: quem quiser a ficha inteira chama
`GET /matches/{uuid}/participants`.

### `RemoveNPCParticipant`

```go
func (r *Repository) RemoveNPCParticipant(
	ctx context.Context, matchUUID, sheetUUID uuid.UUID,
) error
```

Hard delete com a guarda de NPC no SQL (D4):

```sql
DELETE FROM match_participants mp
USING character_sheets cs
WHERE mp.character_sheet_uuid = cs.uuid
  AND mp.match_uuid = $1
  AND mp.character_sheet_uuid = $2
  AND cs.player_uuid IS NULL
```

`RowsAffected() == 0` → `ErrNPCNotInMatch`.

### Testes de integração (`npc_roster_integration_test.go`)

Primeira linha `//go:build integration`, pacote `match_test`. Cada sub-teste começa com
`pgtest.TruncateAll(t, pool)`. Casos:

1. **Adiciona NPC** — ficha com `playerUUID=nil`, `masterUUID` e `campaignUUID` setados;
   `AddNPCParticipant` devolve participante com `Sheet.UUID` igual ao da ficha; em seguida
   `ListParticipantsByMatchUUID` traz a linha, com `Sheet.PlayerUUID == nil` e
   `Sheet.MasterUUID` igual ao do mestre.
2. **Duplicata** — chamar duas vezes devolve `ErrNPCAlreadyInMatch` na segunda, e o roster
   continua com uma linha só.
3. **Ficha de jogador é recusada** — ficha com `playerUUID` setado devolve
   `ErrSheetNotEligibleNPC` e nada é escrito.
4. **Remove NPC** — depois de adicionar, `RemoveNPCParticipant` zera o roster; chamar de novo
   devolve `ErrNPCNotInMatch`.
5. **Remoção não apaga jogador** — insira participante de jogador via
   `pgtest.InsertTestMatchParticipant` com ficha de jogador, chame `RemoveNPCParticipant` com
   aquele `sheetUUID`, espere `ErrNPCNotInMatch`, e confirme que a linha do jogador **continua**
   no roster. Este é o teste que prova D4.

Rode `go test -tags=integration -p 1 ./internal/gateway/pg/match/...` e relate a saída.
Se o banco de teste não subir, diga isso no relatório em vez de silenciar — não invente
resultado.

---

## Task 2 — Domain: NPC age pelo mestre na MatchSession

**Arquivo:** `internal/domain/match/matchsession/match_session.go` (duas funções, nada mais)
e `internal/domain/match/matchsession/match_session_test.go` (aditivo, funções novas no fim).

### 2a. `indexParticipants`

Hoje (linhas ~129-154) ela pula o NPC inteiro:

```go
		if p.Sheet.PlayerUUID == nil {
			continue // NPC: no player to authorize, no per-player fog memory
		}
```

O NPC passa a entrar no `charToPlayer` apontando para o mestre da ficha, e **só nisso**:
continua fora do `pMap`. `pMap` é o conjunto que ganha LOS e `PlayerMemory`
(`RecomputeAllVisibility`, `PlayerIDs`); pôr o mestre lá o transformaria em "jogador" em
vários lugares.

```go
		if p.Sheet.PlayerUUID == nil {
			// NPC: no player to authorize and no per-player fog memory, but the master
			// plays it — charToPlayer is what EnqueueAction and AttachReaction check.
			if p.Sheet.MasterUUID != nil && p.Sheet.UUID != uuid.Nil {
				charToPlayer[p.Sheet.UUID.String()] = *p.Sheet.MasterUUID
			}
			continue
		}
```

Atualize o comentário de doc da função para dizer que o NPC ganha entrada de autorização
apontando para o mestre, mas não ganha ponte de fog.

### 2b. `EnqueueAction`

Hoje (~linha 1053) tem dois portões, e o primeiro mata o mestre antes de o segundo rodar:

```go
	if _, ok := s.participants[playerUUID]; !ok {
		return ErrParticipantNotFound
	}
	owner, ok := s.charToPlayer[a.GetActorID().String()]
	if !ok || owner != playerUUID {
		return ErrActionActorMismatch
	}
```

Inverta: a posse via `charToPlayer` passa a valer como prova de participação, que é como
`AttachReaction` (~linha 854) já funciona.

```go
	owner, ok := s.charToPlayer[a.GetActorID().String()]
	if !ok || owner != playerUUID {
		if _, isParticipant := s.participants[playerUUID]; !isParticipant {
			return ErrParticipantNotFound
		}
		return ErrActionActorMismatch
	}
```

Os quatro cenários atuais se preservam: jogador com o próprio personagem passa; jogador de
fora recebe `ErrParticipantNotFound`; jogador de dentro pedindo personagem alheio recebe
`ErrActionActorMismatch`; e o mestre continua **não** conseguindo agir por personagem de
jogador, porque ali `charToPlayer` aponta para o jogador e o mestre não está em
`participants`. O novo é o mestre por NPC, que passa.

Ponha um comentário curto acima do bloco explicando por que a posse vem antes — senão o
próximo leitor "conserta" a ordem de volta.

### Testes (aditivos, no fim de `match_session_test.go`)

Olhe os helpers que já existem no arquivo (`makeParticipant` por volta da linha 276) e
reaproveite-os; se precisar de um helper de NPC, escreva um novo ao lado, não altere o
existente.

1. **Mestre age pelo NPC** — sessão com um participante de jogador e um NPC
   (`PlayerUUID nil`, `MasterUUID` = mestre); `EnqueueAction(masterUUID, açãoComAtorNPC)`
   retorna `nil` e a ação entra em `PendingActions()`.
2. **Jogador não age pelo NPC** — `EnqueueAction(playerUUID, açãoComAtorNPC)` retorna
   `ErrActionActorMismatch`.
3. **Mestre não age por personagem de jogador** — `EnqueueAction(masterUUID,
   açãoComAtorDoJogador)` retorna `ErrParticipantNotFound`.
4. **Regressão dos dois portões** — jogador de fora da partida continua recebendo
   `ErrParticipantNotFound`; jogador de dentro pedindo personagem alheio continua recebendo
   `ErrActionActorMismatch`. (Os testes existentes nas linhas ~302 já cobrem parte disso;
   confirme que continuam passando e não os reescreva.)
5. **NPC fora do `pMap`** — `PlayerIDs()` não contém o UUID do mestre.

Rode `go test ./internal/domain/match/...` e relate.

---

## Task 3 — Application: use cases de adicionar e remover NPC

**Arquivos:** cria `internal/application/match/add_match_npc.go`,
`internal/application/match/remove_match_npc.go`,
`internal/application/match/add_match_npc_test.go`,
`internal/application/match/remove_match_npc_test.go`; edita (aditivo)
`internal/application/match/error.go`.

**Não altere `IRepository` nem `internal/application/testutil/mock_match_repo.go`.** Use
interfaces locais estreitas, declaradas no consumidor — padrão já usado em
`list_match_enrollments.go:24`.

### Erros novos (aditivo no fim do bloco `var` em `error.go`)

```go
ErrSheetNotNPC           = domain.NewValidationError(errors.New("character sheet is not an npc"))
ErrSheetNotOwnedByMaster = domain.NewValidationError(errors.New("character sheet does not belong to the master or the campaign"))
ErrNPCAlreadyInMatch     = domain.NewValidationError(errors.New("npc is already in this match"))
ErrNPCNotInMatch         = domain.NewValidationError(errors.New("npc is not in this match"))
ErrCharacterSheetNotFound = domain.NewValidationError(errors.New("character sheet not found"))
```

### Interfaces locais (declare em `add_match_npc.go`, reuse em `remove_match_npc.go`)

```go
// IMatchReader reads the match being rostered. Narrow on purpose: widening IRepository
// would drag every mock that implements it into this slice.
type IMatchReader interface {
	GetMatch(ctx context.Context, uuid uuid.UUID) (*match.Match, error)
}

// ISheetOwnershipReader answers who owns a sheet — the only thing the roster needs to know
// about it.
type ISheetOwnershipReader interface {
	GetCharacterSheetRelationshipUUIDs(ctx context.Context, uuid uuid.UUID) (csEntity.RelationshipUUIDs, error)
}

// INPCRoster writes the NPC side of match_participants.
type INPCRoster interface {
	AddNPCParticipant(ctx context.Context, matchUUID, sheetUUID uuid.UUID, joinedAt time.Time) (*match.Participant, error)
	RemoveNPCParticipant(ctx context.Context, matchUUID, sheetUUID uuid.UUID) error
}
```

`*matchPg.Repository` satisfaz `IMatchReader` e `INPCRoster`; `*sheetPg.Repository` satisfaz
`ISheetOwnershipReader`.

### `AddMatchNPCUC`

```go
type IAddMatchNPC interface {
	Add(ctx context.Context, input *AddMatchNPCInput) (*match.Participant, error)
}

type AddMatchNPCInput struct {
	RequesterUUID uuid.UUID
	MatchUUID     uuid.UUID
	SheetUUID     uuid.UUID
}
```

Construtor `NewAddMatchNPCUC(matchReader IMatchReader, sheets ISheetOwnershipReader, roster INPCRoster) *AddMatchNPCUC`.

Ordem das regras — a autorização do requisitante vem antes de qualquer leitura de ficha, para
não vazar a existência de ficha alheia:

1. `matchReader.GetMatch` → se `errors.Is(err, matchPg.ErrMatchNotFound)` devolva
   `ErrMatchNotFound`; outro erro sobe.
2. `match.MasterUUID != input.RequesterUUID` → `ErrNotMatchMaster`.
3. `match.StoryEndAt != nil` → `ErrMatchAlreadyFinished`. (D5: **não** cheque `GameStartAt`.)
4. `sheets.GetCharacterSheetRelationshipUUIDs` → se
   `errors.Is(err, charactersheet.ErrCharacterSheetNotFound)` devolva
   `ErrCharacterSheetNotFound`; outro erro sobe.
5. `rel.PlayerUUID != nil` → `ErrSheetNotNPC` (D1).
6. Elegibilidade (D2): passa se
   (`rel.MasterUUID != nil && *rel.MasterUUID == input.RequesterUUID`) **ou**
   (`rel.CampaignUUID != nil && *rel.CampaignUUID == match.CampaignUUID`).
   Senão → `ErrSheetNotOwnedByMaster`.
7. `roster.AddNPCParticipant(ctx, matchUUID, sheetUUID, time.Now())` → traduza
   `matchPg.ErrNPCAlreadyInMatch` para `ErrNPCAlreadyInMatch` e
   `matchPg.ErrSheetNotEligibleNPC` para `ErrSheetNotNPC`.

### `RemoveMatchNPCUC`

```go
type IRemoveMatchNPC interface {
	Remove(ctx context.Context, input *RemoveMatchNPCInput) error
}

type RemoveMatchNPCInput struct {
	RequesterUUID uuid.UUID
	MatchUUID     uuid.UUID
	SheetUUID     uuid.UUID
}
```

Construtor `NewRemoveMatchNPCUC(matchReader IMatchReader, roster INPCRoster) *RemoveMatchNPCUC`.

Passos 1 e 2 idênticos ao Add. Depois `roster.RemoveNPCParticipant`, traduzindo
`matchPg.ErrNPCNotInMatch` para `ErrNPCNotInMatch`. **Sem** checagem de `StoryEndAt`: tirar
NPC de partida encerrada é inofensivo e recusar só atrapalha limpeza.
**Sem** releitura de ficha: a guarda de NPC já está no SQL (D4).

### Testes

Mocks locais em cada arquivo `_test.go` (structs com campos `...Fn`), no estilo de
`internal/application/testutil/mock_match_repo.go` mas **locais ao pacote de teste**.
Table-driven cobrindo cada regra acima, incluindo:

- partida inexistente → `ErrMatchNotFound`
- requisitante não é o mestre → `ErrNotMatchMaster` (e a ficha **não** é lida — assert que o
  mock de ficha não foi chamado)
- partida encerrada → `ErrMatchAlreadyFinished` (só no Add)
- ficha de jogador → `ErrSheetNotNPC`
- ficha de NPC de outro mestre e de outra campanha → `ErrSheetNotOwnedByMaster`
- ficha de NPC de outro mestre **mas da campanha da partida** → sucesso (D2)
- duplicata → `ErrNPCAlreadyInMatch`
- remoção de quem não está → `ErrNPCNotInMatch`

Rode `go test ./internal/application/match/...` e relate.

---

## Task 4 — Delivery REST + wiring

**Arquivos:** cria `internal/app/api/match/add_match_npc.go`,
`internal/app/api/match/remove_match_npc.go`,
`internal/app/api/match/add_match_npc_test.go`,
`internal/app/api/match/remove_match_npc_test.go`; edita (aditivo)
`internal/app/api/match/routes.go`, `internal/app/api/match/mocks_test.go` e
`cmd/api/main.go`.

### Rotas

`POST /matches/{uuid}/npcs` — `DefaultStatus: http.StatusCreated`.
`DELETE /matches/{uuid}/npcs/{sheet_uuid}` — `DefaultStatus: http.StatusNoContent`.

Um identificador por segmento de path (convenção da casa). Siga o nome de path param do
pacote: os outros handlers de `match` usam `{uuid}` para a partida.

Em `routes.go`: acrescente dois campos no fim do struct `Api` e dois `huma.Register` no fim de
`RegisterRoutes`. Não reordene nada do que já está lá (G2).

```go
	AddMatchNPCHandler    Handler[AddMatchNPCRequest, AddMatchNPCResponse]
	RemoveMatchNPCHandler Handler[RemoveMatchNPCRequest, RemoveMatchNPCResponse]
```

`Errors` das duas operações: 400, 401, 403, 404, 422, 500.

### Request/Response

```go
type AddMatchNPCRequestBody struct {
	CharacterSheetUUID uuid.UUID `json:"characterSheetUuid" required:"true" doc:"UUID of the NPC character sheet to put in the match"`
}

type AddMatchNPCRequest struct {
	UUID uuid.UUID `path:"uuid" required:"true" doc:"Match UUID"`
	Body AddMatchNPCRequestBody
}

type MatchNPCResponse struct {
	UUID               uuid.UUID `json:"uuid"`
	MatchUUID          uuid.UUID `json:"matchUuid"`
	CharacterSheetUUID uuid.UUID `json:"characterSheetUuid"`
	JoinedAt           string    `json:"joinedAt"` // RFC3339
}

type AddMatchNPCResponseBody struct {
	Participant MatchNPCResponse `json:"participant"`
}

type AddMatchNPCResponse struct {
	Body AddMatchNPCResponseBody
}

type RemoveMatchNPCRequest struct {
	UUID      uuid.UUID `path:"uuid" required:"true" doc:"Match UUID"`
	SheetUUID uuid.UUID `path:"sheet_uuid" required:"true" doc:"NPC character sheet UUID"`
}

type RemoveMatchNPCResponse struct{}
```

`JoinedAt` sai como `p.JoinedAt.Format(time.RFC3339)`, igual a
`internal/app/api/match/get_match_participants.go:85`.

### Mapa de erros (nos dois handlers)

| Erro do use case | HTTP |
|---|---|
| `ErrMatchNotFound`, `ErrCharacterSheetNotFound`, `ErrNPCNotInMatch` | 404 |
| `ErrNotMatchMaster` | 403 |
| `ErrSheetNotNPC`, `ErrSheetNotOwnedByMaster`, `ErrMatchAlreadyFinished`, `ErrNPCAlreadyInMatch` | 422 |
| resto | 500 |

`userUUID` vem de `ctx.Value(apiAuth.UserIDKey).(uuid.UUID)`; falha na asserção → 500, como
nos handlers vizinhos.

### Wiring em `cmd/api/main.go`

Ao lado de `getMatchParticipantsUC`, construa os dois UCs e acrescente os dois campos no
literal `matchHandler.Api{...}`. `matchRepo` satisfaz `IMatchReader` e `INPCRoster`;
`characterSheetRepo` satisfaz `ISheetOwnershipReader`. Edição aditiva: não reordene o literal.

### Testes de handler

`humatest`, no estilo de `internal/app/api/matchmap/attach_test.go`. Mocks novos em
`mocks_test.go` (aditivos, no fim do arquivo). Cubra: sucesso (201 no Add com corpo correto,
204 no Remove), encaminhamento correto do `RequesterUUID`/`MatchUUID`/`SheetUUID` ao UC, e
um caso por faixa de status (403, 404, 422).

Rode `go test ./internal/app/api/match/... && go build ./...` e relate.

---

## Task 5 — Prova de que o NPC sobrevive ao Init, contrato e mapa de docs

**Arquivos:** edita (aditivo) `internal/application/match/init_match_session_test.go`; cria
`docs/dev/api/match-npcs.md`; edita (aditivo) `docs/documentation-map.yaml`.

Lembre de G3: **não** altere `init_match_session.go`, só o teste.

### Teste do Init — o critério de pronto deste PR

Acrescente ao fim de `init_match_session_test.go` um teste que prove o que D6 diz que é o
pronto: um roster que contém um NPC produz uma sessão em que o NPC tem `CharacterStatus` e as
duas barras, e em que o mestre pode agir por ele.

Olhe primeiro como os testes existentes no arquivo montam os mocks de
`IRepository`/`ICharSheetLoader`/`IRoundRepository` e reaproveite. O teste:

1. `ListParticipantsByMatchUUID` devolve dois participantes: um de jogador e um NPC
   (`Sheet.PlayerUUID == nil`, `Sheet.MasterUUID` = mestre).
2. `FindActiveSession` devolve `nil, nil` (sessão nova).
3. `Init` roda; a sessão resultante tem `CharacterStatus` para o sheet UUID do NPC, com as
   duas barras presentes (use o acessor que a `MatchSession` já expõe para status — procure
   por `GetCharacterStatus` ou equivalente em `match_session.go`; se não houver acessor
   público, verifique pelo caminho que os testes existentes usam, **sem** adicionar acessor
   novo só para o teste).
4. `session.GetCharToPlayer()` mapeia o sheet UUID do NPC para o UUID do mestre.

Se o `ICharSheetLoader` mockado precisar devolver uma ficha montada, siga exatamente o que os
testes vizinhos fazem.

### `docs/dev/api/match-npcs.md`

Em PT-BR, no formato de `docs/dev/api/match.md` (título, seção por endpoint, **Auth**,
Request em bloco `json`, tabela de regras, tabela de Respostas com status × situação).
Documente os dois endpoints com o mapa de erros da Task 4, e inclua exemplo de request e de
response.

Acrescente, ao fim do arquivo, uma seção curta **"O que este endpoint não faz"** dizendo, em
duas ou três linhas: a partida em andamento **não** vê o NPC na hora, porque a `MatchSession`
vive no processo do game server (`cmd/game`, :8081) e o endpoint vive no `cmd/api` (:5000);
o NPC aparece quando a sala renasce e o `InitMatchSessionUC` recarrega o roster; a propagação
ao vivo é fatia futura, um verbo de WS no game server.

### `docs/documentation-map.yaml`

Acrescente uma entrada (no estilo das existentes, veja a de `internal/app/api/matchmap/` por
volta da linha 405) mapeando `internal/app/api/match/add_match_npc.go` →
`docs/dev/api/match-npcs.md`, com `notes` descrevendo
`POST/DELETE /matches/{uuid}/npcs` — rostering de NPC pelo mestre.
Não reordene as entradas existentes.

Rode `go test ./internal/application/match/...` e relate.
