# Plano — NPC ao vivo por WS + débito de LOS do arrasto de NPC

**Branch:** `feat/live-npc-ws` (a partir de `main` com PRs #73 e #74 mergeados).
**Pacotes:** `internal/domain/match/matchsession`, `internal/application/match`,
`internal/app/game`, `cmd/game`, `docs/`.

## Contexto

PR #73 abriu `POST/DELETE /matches/{uuid}/npcs` (REST, `cmd/api`, :5000): o NPC entra em
`match_participants` e `InitMatchSessionUC` o carrega quando a sala nasce. Uma sala **já viva**
não o vê: a `MatchSession` mora na memória do `cmd/game` (:8081), outro processo.
`indexParticipants` (match_session.go) mapeia NPC → mestre em `charToPlayer`, e isso criou um
custo no caminho quente: `relayPieceMove` (room.go) resolve o "dono" da peça por
`charToPlayer`, então o mestre arrastando um NPC dispara `RecomputeVisibility(mestre)` (todas as
peças de NPC × paredes, sob lock de escrita), cria uma `PlayerMemory` para o mestre que ninguém
lê, e reenvia `map_full_state` inteiro ao mestre — que já enxerga tudo.

## Decisões (registradas — vão para o contrato e o PR)

1. **O verbo WS convive com o REST.** REST = montar o roster antes da partida (preparação,
   sem sala). WS `add_npc` = pôr NPC no meio dela. **O WS não chama o REST**: o game server já
   tem o pool do Postgres, então o verbo executa o **mesmo** `AddMatchNPCUC` (mesmas guardas,
   mesma escrita em `match_participants`) e depois injeta a ficha na sessão viva. Um ato só,
   do ponto de vista do mestre; banco e memória não podem divergir por um segundo passo que o
   front esqueça.
2. **`ErrNPCAlreadyInMatch` do banco NÃO é erro para o verbo WS.** Todas as guardas do
   `AddMatchNPCUC` (mestre, partida não encerrada, ficha existe, é NPC, elegibilidade) rodam
   ANTES do INSERT que devolve a duplicata. Então "já está no banco" significa "passou em tudo,
   só a sessão está atrasada" — exatamente o caso de quem adicionou pelo REST no meio da partida.
   O verbo segue e injeta. Isso fecha o buraco do REST-no-meio-da-partida: o mestre reenvia
   `add_npc` e a sessão alcança o banco. O erro de duplicata do verbo é **da sessão**
   (`npc_already_in_match`), não do banco.
3. **Na sala sem sessão (lobby) o verbo funciona**: grava no banco (o `Init` o trará) e anuncia
   `npc_added`. Não há barras para publicar.
4. **Remoção ao vivo fica FORA.** Tirar um NPC de uma sessão viva esbarra em ação dele na fila,
   turno aberto com ele como ator/alvo, reação pendente — regras que ninguém decidiu. O REST
   `DELETE` continua valendo só para a próxima vez que a sala nascer. Registrar como lacuna.
5. **Propagação: `npc_added` (mesa inteira) + `bars_updated` (mesa inteira), NÃO
   `match_full_state`.** O único estado de combate que muda ao pôr um NPC é o conjunto de
   personagens nas barras. `broadcastBars` é o caminho documentado "depois de qualquer coisa que
   mexe nas barras" e **incrementa `seq`** — reenviar `match_full_state` com o `seq` corrente (é o
   que ele faz, por contrato) deixaria um `bars_updated` atrasado de mesmo `seq`, sem o NPC, ser
   aplicado por cima e apagar o NPC na tela. `match_full_state` continua sendo o que o nome diz:
   snapshot de quem conecta. `npc_added { characterId }` é o ack do mestre e o sinal para o front
   (re)buscar a ficha por REST — mesma forma de `scene_changed`/`master_action_enqueued`. Não
   vaza nada novo: `bars_updated.characters` já lista todo personagem, NPC incluído.
6. **`charToPlayer` passa a ser copy-on-write.** Até aqui ninguém escrevia nele depois do
   construtor, e `GetCharToPlayer()` devolve o próprio mapa — `publishResolution` e
   `buildMapFullState` o pegam sob `RLock` e iteram DEPOIS do `RUnlock`. Com `AddNPC` escrevendo
   nele, mutar in-place viraria `concurrent map read and map write` (fatal, não recuperável).
   `AddNPC` monta um mapa novo (cópia + a entrada) e troca a referência; quem já segurava o
   antigo continua lendo um mapa que ninguém mais muda. `statuses`/`charSheets` só são lidos por
   métodos da sessão sob `r.mu`, então escrita in-place sob lock de escrita basta.
7. **Débito de LOS:** em `relayPieceMove`, dono == mestre é tratado como "sem dono" — pula o
   recompute, a criação de memória e o `map_full_state` extra. O mestre continua recebendo
   `piece_moved` pelo ramo `isMaster` do dispatch (ou não recebe eco, se foi ele quem arrastou —
   o navegador dele já aplicou). Vale para os três caminhos que passam por `relayPieceMove`:
   arrasto do cliente, ação de turno e fuga de reação.

## Global Constraints

- Go 1.23; `testing` padrão, table-driven com `t.Run`; pacotes de teste externos `foo_test`
  quando o pacote já faz isso (siga o arquivo vizinho).
- **Nunca remova comentários TODO.**
- `room.go` é dono do lock (`r.mu`); `MatchSession` não tem lock próprio. Nada que envia para
  cliente roda dentro de seção crítica. I/O (banco) **fora** do lock.
- Wire format camelCase. Tipos de mensagem snake_case como os existentes.
- Commits terminam com:
  `Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>`
- Verificação por task: `go build ./... && go vet ./...` e os testes do pacote tocado.
  Task que toca `internal/app/game` roda também `go test -race ./internal/app/game/`.
- Comentários explicam **por quê**, no tom dos vizinhos (o arquivo é densamente comentado; siga).

---

## Task 1 — `MatchSession.AddNPC` (domínio)

**Arquivos:** `internal/domain/match/matchsession/match_session.go`,
`internal/domain/match/matchsession/error.go`,
`internal/domain/match/matchsession/match_session_test.go`.

Adicionar em `error.go` (siga o estilo dos erros existentes lá):

```go
ErrCharacterAlreadyInSession = errors.New("character is already in this match session")
```

Adicionar em `match_session.go`, perto de `indexParticipants`/`GetCharSheet`:

```go
// AddNPC puts a master-controlled character into a LIVE session — the in-memory half of
// what indexParticipants does at construction for an NPC row. ...
func (s *MatchSession) AddNPC(sheetUUID uuid.UUID, sheet *csSheet.CharacterSheet, masterUUID uuid.UUID) error
```

Comportamento:
- `sheetUUID == uuid.Nil` ou `sheet == nil` ou `masterUUID == uuid.Nil` → erro (use um erro
  existente adequado se houver; senão `errors.New` local descritivo). Não pode gravar lixo.
- Personagem já presente em `s.statuses` → `ErrCharacterAlreadyInSession`, **nada** muda.
- Senão: `s.charSheets[sheetUUID] = sheet`; `s.statuses[sheetUUID] = match.NewCharacterStatus()`;
  `charToPlayer` **copy-on-write** (Decisão 6): novo mapa com todas as entradas + a nova, e só
  então `s.charToPlayer = novo`. `s.participants` **não** é tocado (NPC fica fora dele, como em
  `indexParticipants`). Guarde contra `charSheets`/`statuses`/`charToPlayer` nil (inicialize).
- O comentário do método explica: por que NPC fica fora de `participants`; por que o
  charToPlayer é trocado e não mutado (quem lê fora do lock: `publishResolution`,
  `buildMapFullState` em room.go); que o chamador segura `r.mu` para escrita.

Testes (TDD — escreva antes):
1. Depois de `AddNPC`, `GetCharSheet(id)` devolve a ficha, `GetCharacterStatus(id)` existe,
   `CharacterIDs()` o contém, `GetCharToPlayer()[id.String()] == master`, e `PlayerIDs()` **não**
   contém o mestre.
2. `AddNPC` repetido → `ErrCharacterAlreadyInSession` (errors.Is) e o status original é o
   **mesmo ponteiro** (não foi resetado).
3. **Copy-on-write:** pegue `before := s.GetCharToPlayer()`, chame `AddNPC`, e afirme que
   `before` NÃO contém a chave nova e tem o mesmo `len` de antes. (É o teste que prova a
   Decisão 6 sem depender do race detector.)
4. Entradas inválidas (Nil sheetUUID, nil sheet, Nil master) → erro, nada gravado.
5. Um NPC adicionado pode enfileirar: `EnqueueAction(master, action com actorID = NPC)` passa
   (use o mesmo formato de action que os testes de NPC do PR #73 em `match_session_test.go`
   usam — procure por "NPC" no arquivo e reaproveite o helper).

Rode: `go test ./internal/domain/match/...` e `go vet ./...`.

---

## Task 2 — `AddLiveNPCUC` (aplicação)

**Arquivos:** novo `internal/application/match/add_live_npc.go` e `add_live_npc_test.go`
(siga `add_match_npc.go` / `add_match_npc_test.go` para estilo e mocks).

```go
type IAddLiveNPC interface {
	Execute(ctx context.Context, input *AddMatchNPCInput) (*csSheet.CharacterSheet, error)
}

type AddLiveNPCUC struct {
	roster      IAddMatchNPC     // o AddMatchNPCUC real em produção
	sheetLoader ICharSheetLoader // o mesmo que InitMatchSessionUC usa
}

func NewAddLiveNPCUC(roster IAddMatchNPC, sheetLoader ICharSheetLoader) *AddLiveNPCUC
```

`Execute`:
1. `roster.Add(ctx, input)`. Erro `ErrNPCAlreadyInMatch` → **segue** (Decisão 2; comentário
   explicando que todas as guardas rodaram antes do INSERT). Qualquer outro erro → devolve
   como está.
2. `sheetLoader.GetCharacterSheetByUUID(ctx, input.SheetUUID.String())`. O segundo retorno é
   `wasCorrected`, **não** "found" (ver o aviso em `init_match_session.go`). Erro
   `charactersheet.ErrCharacterSheetNotFound` → devolve `ErrCharacterSheetNotFound` do pacote
   `match` (o mesmo que o AddMatchNPCUC usa). Outro erro → devolve.
3. Devolve a ficha. **Não** toca em sessão: o UC faz I/O e regra; a sessão é mutada pelo room
   sob `r.mu` (Decisão: I/O fora do lock).
- `var _ IAddLiveNPC = (*AddLiveNPCUC)(nil)`.
- Registrar a interface também em `internal/app/game/room.go`? **Não nesta task** — Task 3.

Testes (mocks locais, table-driven):
- caminho feliz: Add ok → loader chamado com o UUID → ficha devolvida.
- `ErrNPCAlreadyInMatch` do roster → ainda carrega e devolve a ficha.
- `ErrNotMatchMaster`, `ErrSheetNotNPC`, `ErrMatchAlreadyFinished` do roster → devolvidos
  (errors.Is) e o loader **não** é chamado.
- loader `ErrCharacterSheetNotFound` → `ErrCharacterSheetNotFound` do pacote match.
- loader erro genérico → propagado.

Rode: `go test ./internal/application/match/...`, `go vet ./...`.

---

## Task 3 — verbo `add_npc` no game server + débito de LOS

**Arquivos:** `internal/app/game/message.go`, `room.go`, `hub.go`, `handler.go`,
`cmd/game/main.go`, os construtores de teste (`game_test.go`, `fog_dispatch_test.go`,
`handler_test.go`, `combat_e2e_test.go`, `fog_e2e_test.go`, `visibility_e2e_test.go`,
`reaction_chain_e2e_test.go`) e um novo `internal/app/game/add_npc_e2e_test.go` (ou dentro
do arquivo de teste mais próximo — siga o padrão de `edit_action_e2e_test.go`/`override_e2e_test.go`).

### 3a. Mensagens (`message.go`)
- `MsgTypeAddNPC MessageType = "add_npc"` (cliente → servidor), payload
  `AddNPCPayload { CharacterSheetUUID uuid.UUID `json:"characterSheetUuid"` }` — mesmo nome de
  campo do REST.
- `MsgTypeNPCAdded MessageType = "npc_added"` (servidor → cliente), payload
  `NPCAddedPayload { CharacterID uuid.UUID `json:"characterId"` }`.

### 3b. Dependência
- `type IAddLiveNPC = appmatch.IAddLiveNPC` em room.go (como `IEditAction`).
- Novo parâmetro **no fim** de `NewRoom`, `NewHandler`, campo em `Room`/`Handler`, repassado em
  `hub.go`. `cmd/game/main.go`: `addMatchNPCUC := match.NewAddMatchNPCUC(matchRepository,
  sheetRepository, matchRepository)` (mesma montagem de `cmd/api/main.go:203`) e
  `addLiveNPCUC := match.NewAddLiveNPCUC(addMatchNPCUC, sheetRepository)`.
- Testes existentes: passam `nil` (ou um mock trivial se o construtor exigir); não mude a
  semântica de nenhum teste existente.

### 3c. O braço `case MsgTypeAddNPC` em `handleClientMessage`
Ordem:
1. Não-mestre → `NewErrorMessage("forbidden", ErrNotMaster.Error())`.
2. Unmarshal → falha ou `CharacterSheetUUID == uuid.Nil` → `invalid_payload`
   (`"invalid add_npc payload"`).
3. `sheet, err := r.addLiveNPCUC.Execute(ctx, &appmatch.AddMatchNPCInput{RequesterUUID:
   client.userUUID, MatchUUID: r.matchUUID, SheetUUID: payload.CharacterSheetUUID})` — **sem
   `r.mu`** (é I/O). Mapeamento de erro (errors.Is):
   - `ErrNotMatchMaster` → `forbidden`
   - `ErrMatchNotFound`, `ErrCharacterSheetNotFound` → `not_found`
   - `ErrSheetNotNPC`, `ErrSheetNotOwnedByMaster`, `ErrMatchAlreadyFinished` → `invalid_npc`
   - outro → `game_error`
   Mensagem = `err.Error()`.
4. `r.mu.Lock()`; `session := r.session`; se não-nil, `addErr = session.AddNPC(sheetUUID, sheet,
   r.masterUUID)`; `r.mu.Unlock()`. `ErrCharacterAlreadyInSession` → `npc_already_in_match`;
   outro erro → `game_error`. (Com sessão nil, pula — Decisão 3.)
5. Broadcast `npc_added { characterId }` para a mesa (via `r.broadcast` em goroutine, como
   `scene_changed`/`master_action_enqueued`).
6. Se havia sessão: `r.broadcastBars(session)`.

Comentário no braço: por que convive com o REST (Decisão 1), por que o duplicado do banco não é
erro (Decisão 2), por que `bars_updated` e não `match_full_state` (Decisão 5).

### 3d. Débito de LOS (`relayPieceMove`)
Depois de resolver `owner` a partir de `charToPlayer`, se `owner == r.masterUUID` trate como
`uuid.Nil` (sem dono) — Decisão 7. Comentário explicando: o mestre vê o tabuleiro sem filtro,
`buildMapFullState` descarta polígonos quando `isMaster`, e o recompute custava (NPCs × paredes)
sob lock de escrita + uma `PlayerMemory` que ninguém lê + o tabuleiro inteiro reenviado a cada
arrasto. Atualize também o comentário de `applyAndRelayPieceMove` que diz "Whoever owns the moved
character gets a fresh map_full_state" se ficar impreciso.

### 3e. Testes (e2e contra `Handler` real, clientes WS reais — siga `combat_e2e_test.go`)
Use um `IInitMatchSession` / sessão real cujo roster tenha um jogador; injete o NPC pelo verbo.
Mock do `IAddLiveNPC` que devolve uma ficha real (reaproveite o builder de ficha que os e2e já
usam) e registra as chamadas.
1. **Mestre adiciona em sala viva:** mestre recebe `npc_added` com o `characterId`; jogador
   também; ambos recebem `bars_updated` com `seq` maior que o anterior e o NPC em `characters`.
   E depois disso o **mestre consegue `enqueue_action` com `actorId` = NPC** (recebe
   `action_enqueued`, não `error`) — é a prova de que a sessão viva o conhece.
2. **Jogador tenta** → `error` `forbidden`, UC **não** chamado.
3. **Duplicado na sessão:** segundo `add_npc` do mesmo NPC → `error` `npc_already_in_match`,
   sem segundo `npc_added`.
4. **UC recusa** (`ErrSheetNotNPC`) → `error` `invalid_npc`, sessão intocada.
5. **Payload inválido** (uuid zero) → `invalid_payload`.
6. **Lobby (sem sessão):** UC chamado, `npc_added` sai, nenhum `bars_updated`.
7. **Débito de LOS:** com um NPC no `charToPlayer` e uma peça dele no tabuleiro, o mestre
   arrasta a peça (`piece_moved` do cliente mestre) → o mestre **não** recebe `map_full_state`;
   um jogador que vê o destino recebe `piece_moved`; e `session.GetPlayerMemory(master)`
   devolve `ok == false` (nenhuma memória criada). Use o padrão de barreira de ordenação que os
   testes de movimento já usam para asserções negativas (procure "ordering barrier" /
   "barrier" nos testes do pacote) — não durma.
8. Controle do 7: arrastar a peça de um **jogador** continua mandando `map_full_state` ao dono.

Rode: `go build ./...`, `go vet ./...`, `go test ./internal/app/game/`,
`go test -race ./internal/app/game/`.

---

## Task 4 — Documentação

**Arquivos:** `docs/dev/api/match-combat-ws.md`, `docs/dev/api/match-npcs.md`,
`docs/dev/match/flows/05-lacunas.md`, `docs/documentation-map.yaml`, `AGENTS.md` só se citar
algo que ficou falso.

- `match-combat-ws.md`:
  - §2: o bloco "**NPC hoje não age**" está **obsoleto desde o PR #73** — substituir pelo estado
    real: NPC é do mestre (`charToPlayer` NPC → mestre), o mestre age por ele com `actorId` = a
    ficha do NPC; ficha de jogador continua negada ao mestre. Remover a linha "NPC não age" da
    tabela do §9.
  - §3 índice: `add_npc` (mestre) e `npc_added` (mesa inteira).
  - §4: seção `add_npc` — payload, pré-condições, o que acontece com e sem sessão, a regra do
    duplicado (Decisão 2), erros; e por que convive com o REST (Decisão 1).
  - §5: seção `npc_added` — payload, destino, quando; e a nota de que as barras chegam por
    `bars_updated` (com `seq` novo), não por `match_full_state` (Decisão 5).
  - §7 catálogo de erros: `not_found`, `invalid_npc`, `npc_already_in_match`; acrescentar
    `add_npc` na linha de `forbidden`.
  - Seção `piece_moved` servidor (~linha 1117): o texto "uma peça de NPC ... não tem dono" volta
    a ser verdade por outro motivo — reescrever: a peça de NPC pertence ao mestre, e o mestre não
    recebe o `map_full_state` extra porque enxerga o tabuleiro inteiro (Decisão 7).
  - §9: lacuna nova "**Remoção de NPC ao vivo não existe**" (Decisão 4).
- `match-npcs.md`, seção "O que este endpoint não faz": apontar para `add_npc` como o caminho no
  meio da partida, e dizer que reenviar `add_npc` para um NPC já posto pelo REST sincroniza a
  sessão; remoção continua só-na-próxima-sala.
- `05-lacunas.md`: a linha "Rostering de NPC — nada cria um NPC hoje" está obsoleta — marcar como
  resolvida (#73 + este PR), seguindo o formato que o arquivo usa para itens resolvidos.
- `documentation-map.yaml`: nota do `add_match_npc.go` atualizada (não diz mais que a sala viva
  não vê); entrada nova para `internal/application/match/add_live_npc.go` →
  `docs/dev/api/match-combat-ws.md` e `match-npcs.md`. **Não** mexer nos caminhos obsoletos
  (isso é outro PR).
- Tudo em PT-BR, no tom dos documentos existentes.
