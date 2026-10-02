# Fechamento da Fase 6 — pacote de back (B1–B16) — design

> **Escopo:** o §6A.5 inteiro de
> [`2026-09-20-front-combat-phases.md`](2026-09-20-front-combat-phases.md) — B1 a B16 —, na
> versão de `main` depois do **PR #80** (commit `e5fc3dd`), mais a decisão das master actions do **PR #81**. **Um PR, repo
> `System_X_System`.** O front (F1–F16) roda em paralelo, em outra sessão, e consome os contratos
> que este PR escreve.
>
> Plano: [`../plans/2026-09-27-combat-closure-back.md`](../plans/2026-09-27-combat-closure-back.md).

## 0. O workflow (cópia do §0.1 do documento mestre)

1. **Uma sessão por fase por repo.** Back e front rodam em paralelo quando não tocam arquivo em
   comum — são repos diferentes, então normalmente não tocam.
2. A sessão **lê este documento e o contrato** (`docs/dev/api/match-combat-ws.md`), escreve o
   **design spec** e o **plano**, e **para** para o dono do produto revisar.
3. **Lacuna ou contradição neste documento: liste e pare.** Ela volta para o autor deste
   documento, que corrige o texto. Não se contorna, e não se decide regra de jogo por conta.
4. Aprovado o spec, a sessão **compacta** e implementa **lendo o próprio plano do disco**. Se
   ela não conseguir implementar a partir do plano, o plano estava incompleto — é melhor
   descobrir nessa hora.
5. Implementação por **subagent-driven-development**, uma tarefa por subagente.
6. **Verificação no browser, com três contas** (`test@`, `test2@`, `test3@mail.com`, senha
   `12345678`): um mestre e dois jogadores, jogando o caminho que o usuário faz. A Fase 6 passou
   nos testes e falhou na mesa — o NPC do teste tinha sido inscrito por fora, e o caminho real
   nunca foi exercitado.
7. PR aberto dizendo o que foi verificado e **o que não foi**.

## 1. O que você precisa saber antes

- **Turno** = uma action e suas reactions. **Round** = sequência de turnos. **Cena** = sequência
  de rounds. `actorId` é o **sheetUUID**, nunca o do jogador.
- **Os dados caem quando a action chega** (`MatchSession.rollActionDice`). Nada rola de novo:
  tudo que vem depois **deriva**. Reconciliar nunca re-sorteia.
- `room.go` é dono do lock (`r.mu`). `MatchSession` não tem lock próprio. **Nada que envia a
  cliente roda com `r.mu` preso**; `dispatchPerPlayer` e `applyMove` pegam o lock sozinhos.
- Duas pistas de envio com ordens diferentes: `r.broadcast` (canal, entregue pelo goroutine do
  `Run`) e o envio direto por cliente (`client.SendMessage`, usado por `dispatchPerPlayer`).
  **Mensagens em pistas diferentes não têm ordem garantida entre si.** Isso pesa em B2 (§4.2).
- A causa de cada item está no documento mestre, com arquivo e linha. Este spec não a repete —
  diz **o desenho**.

## 2. O que foi fechado com o dono do produto depois da primeira versão

Nada está aberto. Os três pontos devolvidos ao autor do documento mestre foram fechados, e o
documento os registra (PR #81):

| Ponto | Decisão |
|---|---|
| **B9 — o `attack` da master action.** Nada lê o conteúdo de uma master action; o que um "ataque do mestre" faria é regra de jogo | **O `attack` sai do payload.** O mestre ataca pelo NPC, com `enqueue_action`. Um ataque do mestre só faria sentido como efeito de ambiente (armadilha), que é **futuro**, não pendência. O servidor recusa uma master action com `attack`, dizendo o caminho certo (T17) |
| **B14 — "mesma persistência"** de uma master action que nunca foi persistida | Master actions ganham **tabela própria**, entram no histórico e são projetadas pelo que cada leitor viu ao vivo — §4.8 |
| **"Hoje o `AttachMap` permite"** trocar o mapa depois do início — erro da sessão, na rodada anterior | O use case já recusa (`ErrMatchAlreadyStarted`). O documento foi corrigido; não há o que implementar |

## 3. A forma geral

Três ideias atravessam os dezesseis itens:

1. **O servidor é o dono do tabuleiro** (B14). Ele carrega o tabuleiro do banco, persiste cada
   mudança definitiva (B3), e o cliente só desenha. Tudo que mexe em peça — movimento de ação,
   escape, arrastar do mestre, pôr, tirar — passa pelo mesmo `applyMove`/`relayPieceMove` e pelo
   mesmo ponto de persistência.
2. **Um formato de action, três níveis de corte** (B1, B2, B12). O formato é o do histórico REST;
   o que muda é quanto dele cada destinatário recebe. Um pacote só monta os três.
3. **O que descreve a partida vem do servidor a cada conexão** (§0.2 do mestre). Cada item diz o
   que acontece num reinício (§5 deste spec).

## 4. O desenho, item a item

### 4.1 O formato de action compartilhado — base de B1, B2 e B12

Hoje o formato mora em `internal/app/api/match/get_match_history.go` (`ActionResponse` e
vizinhos). O pacote WS não importa o pacote REST, e não deve.

**Novo pacote `internal/app/wire/actionwire`** (delivery, sem I/O): os tipos `Action`,
`RollCheck`, `RollAttempts`, `Skill`, `ActionSpeed`, `Move`, `Attack`, `Defense`, `Dodge`,
`Repel`, `Interact`, `Trigger` — os mesmos campos e tags JSON de hoje — e uma função:

```go
type Level int

const (
	Full        Level = iota // tudo: dados, totais, velocidades
	Opened                   // mecânica + velocidades; sem dados/result de acerto, dano, perícias, finta, defesas
	Declaration              // só o que o dono declarou; nenhum dado, nenhum total, nenhuma velocidade
)

func From(a action.Action, lvl Level) Action
```

- O REST do histórico passa a usar `actionwire.From(a, actionwire.Full)`. **O JSON do REST não
  muda um byte** — há um teste de ouro que prova isso (plano, T6).
- Para caber os cortes sem mudar o JSON cheio: `RollCheck.SkillValue` e `RollCheck.Result`
  viram `*int` com `omitempty`, `RollCheck.Attempts` vira ponteiro com `omitempty`, e
  `Move.FinalSpeed` vira `*int` com `omitempty`. No nível `Full` eles são sempre preenchidos
  (inclusive com zero), então saem iguais a hoje.
- **O que cada nível corta** (a tabela do B2 do mestre, e o B1/B12):

| Campo | `Full` | `Opened` | `Declaration` |
|---|---|---|---|
| alvos, arma, `move` (categoria, origem, destino — origem e destino ainda sujeitos à fog da peça para quem não é mestre nem dono, ver abaixo), nomes das perícias, `reactionKind`, `interact`, `spread`, `relativeVelocity`, `systemBias` | ✔ | ✔ | ✔ |
| `speed.rollCheck` (actionSpeed) — perícia, dados, total | ✔ | ✔ | só o nome da perícia |
| `move.speed` e `move.finalSpeed` (moveSpeed) | ✔ | ✔ | só o nome da perícia |
| dados, `skillValue` e `result` de `attack.hit`, `attack.damage`, `attack.charge`, `move.charge`, `skills[]`, `feint`, `defense`, `dodge`, `repel` | ✔ | só o nome da perícia | só o nome da perícia |

- A **deny-list** (finta e gatilho escondidos com turno aberto, rótulo fechado rebaixado,
  `Evasion` fora) continua sendo do `service.ProjectAction`, que roda **antes** do `From`. O
  `From` não decide quem vê o quê: ele só corta números. Os dois eixos ficam cada um no seu
  lugar.
- **`move.from`/`move.position` têm um terceiro eixo, a fog da peça** (decisão do dono do
  produto, 2026-10-01): no WebSocket, para quem não é mestre nem dono do ator, eles passam
  ainda pelo gate do relay ao vivo (`pieceMoveView`) — destino visível → os dois; só a origem
  → `from` (só se `move.from` for a origem julgada); nenhum, peça `visible: false` ou ator sem
  peça → nenhum (a categoria fica). A origem julgada no `turn_opened` ao vivo é a casa da peça
  na abertura (a do relay), não o `move.from` do enfileiramento; na reconexão, `move.from`.
  Quem aplica é o `room.go` (`turnActionWireLocked`), depois do `From`, que continua sem
  decidir visibilidade.
  Para caber, `Move.Position` vira `*[3]int` com `omitempty`. O `escape.landing` da resolução
  liquidada segue o mesmo gate. **O REST segue o que foi visto ao vivo** (decisão do dono do
  produto, 2026-10-01, depois desta): o veredito do gate é gravado por jogador da sessão na
  abertura (`actions.move_views`) e no fechamento (`landingViews` no `escape` — onde a peça da
  fuga parou), com o turno, e o histórico corta `from`/`position`/`landing` e o `move.position`
  das reações (só o de uma fuga que escapou, a quem a viu chegar) por ele — linhas antigas falham
  fechado.

### 4.2 B1, B2 e B12 — as três superfícies

**B1 — a fila do mestre.** `ActionQueuedPayload` ganha `action` (nível `Full`, sem projeção:
as duas superfícies já são master-only). O `newActionQueuedPayload` que alimenta o
`action_queued` **e** o `match_full_state.queue` é o mesmo, então os dois não divergem. Os campos
`actionId`, `actorId` e `bars` ficam (o front atual os lê).

**B2 — `turn_opened` projetado.** `TurnOpenedPayload` ganha `action`:

- mestre: `From(a, Full)`;
- **todos os outros, inclusive o dono**: `From(ProjectAction(a, viewer, false), Opened)`.

Deixa de ser broadcast: passa a sair por `dispatchPerPlayer`, um payload por destinatário.

⚠️ **Ordem.** Hoje o `turn_closed` sai por `r.broadcast` (síncrono no canal) e o `turn_opened`
também. Se só o `turn_opened` passasse para a pista direta, ele poderia **chegar antes** do
`turn_closed` do turno anterior. Por isso **os dois passam para a pista direta**, na mesma
ordem de hoje: `piece_moved` → `turn_closed` → `resolution_updated` → `turn_opened`. Numa pista
só e num goroutine só, a ordem de envio é a de chegada. O contrato passa a prometer essa ordem.

O `match_full_state.openTurn` ganha `actionId` (**hoje não tem**, e o front precisa dele para
casar com a lista de declaradas) e `action`, com a mesma projeção.

**B12 — reconciliar.** `MatchFullStatePayload` ganha `ownQueue` para quem **não** é o mestre:

```go
type OwnQueuedActionPayload struct {
	ActionID uuid.UUID          `json:"actionId"`
	Action   actionwire.Action  `json:"action"` // Declaration: sem velocidade (B1 ⚠️)
}
// MatchFullStatePayload.OwnQueue []OwnQueuedActionPayload `json:"ownQueue"` — sem omitempty
```

- São as ações da fila cujo ator pertence ao destinatário (`charToPlayer`), na ordem da fila.
- **Sempre presente** (`[]` quando vazia) para não-mestre com sessão viva. Assim o front
  distingue "o servidor diz que você não tem nada" de "servidor antigo que não manda o campo".
- Regra de reconciliação no contrato: uma declarada é conhecida se o `actionId` dela está em
  `ownQueue` **ou** é o `openTurn.actionId`. O resto sai com aviso, e o rascunho volta.
  **O cliente nunca reenvia sozinho.**

**Por que só reconciliar** (decisão registrada no §6A.9 do mestre): as barras são cobradas no
enqueue e vivem em memória. Uma fila persistida sobre barras zeradas seria um estado que nunca
existiu. Perder a fila num reinício é aceitável; divergir, não.

### 4.3 O tabuleiro da partida — B14, B3, B16, B11 e o `move` de B9

#### Estrutura (B3, com B16 e o editor futuro em mente)

Tabela nova `match_boards`, uma linha por partida:

| Coluna | Tipo | Por quê |
|---|---|---|
| `match_uuid` | `UUID PK → matches ON DELETE CASCADE` | um tabuleiro por partida |
| `map_uuid` | `UUID → maps ON DELETE RESTRICT` | de qual mapa ele **partiu** |
| `grid` | `JSONB` | as coordenadas das paredes e peças dependem dela |
| `bg` | `JSONB NULL` | **`NULL` = herda o fundo do mapa**; o editor de mapa da partida (futuro) escreve aqui |
| `pieces` | `JSONB` | as peças, no formato de `maps.pieces` |
| `walls` | `JSONB` | as paredes **inteiras**, com estado (aberta, trancada, HP, destruída, revelada) |
| `updated_at` | `TIMESTAMPTZ` | |

- É um **retrato**, não um diff. Por isso editar o mapa da campanha com a partida rolando não
  muda nada nela, e duas partidas no mesmo mapa não se tocam — exatamente a decisão do dono.
- O fog explorado vai para `player_memories`, que já existe (`match_id`, `map_id`, `player_id`,
  `seen_features`) e cujo repositório é um stub com `TODO` (`gateway/pg/fog`). O stub é
  implementado.
- **B16 fica trivial:** herdar é `INSERT … SELECT` da linha de `match_boards` e das linhas de
  `player_memories` da partida de origem, trocando o `match_uuid`. Uma transação.
- **Ciclo de vida.** A linha nasce no primeiro salvamento. Carregar = a linha, se existe; senão, um
  retrato fresco do mapa anexado (sem gravar). Anexar **outro** mapa (antes do início — depois
  já é recusado) apaga a linha velha, que era de outro mapa.

#### Quem carrega (B14)

- Pacote de aplicação novo `internal/application/matchboard`: `LoadMatchBoardUC` (anexo →
  linha ou mapa) e `SaveMatchBoardUC` (linha + memórias). Entidade `internal/domain/matchboard`
  com `Board{MatchUUID, MapUUID, Grid, Bg, Pieces, Walls}`.
- O `cmd/game` passa a ter os repositórios de mapa, anexo, tabuleiro e memória.
- A `Room` carrega o tabuleiro **quando nasce** e, **enquanto é lobby, toda vez que o mestre
  conecta** — é o momento em que hoje o `map_state_sync` chegava, e é o que faz uma edição de
  mapa antes do primeiro movimento de lobby aparecer. Depois do início, só no nascimento.
- O `map_state_sync` **deixa de escrever qualquer coisa**: é aceito, ignorado, e responde ao
  remetente com um `map_full_state`. Fica marcado como obsoleto no contrato até o F13 tirá-lo do
  front (recusar agora quebraria o front que está no ar).
- **Última defesa** (pedida pelo mestre): interagir ou revelar uma parede que o servidor não
  conhece responde `error` `unknown_wall` ao mestre, em vez do silêncio de hoje.

#### Quando persiste (B3)

Em todo momento em que o tabuleiro muda de forma definitiva: movimento e remoção no lobby,
`start_match`, os **três** verbos que fecham turno (depois dos escapes), e — **entre turnos** —
as master actions de peça e a interação/revelação de parede. Com um turno aberto, essas duas
não salvam: o fechamento do turno salva, com elas dentro (decisão do dono do produto,
2026-10-01, abaixo). Um só retrato, `Room.boardSnapshotLocked()`, e dois escritores: o
fechamento, que o entrega a `PersistTurnClose` para gravar na transação do turno (abaixo), e
`Room.persistBoard()` para todo o resto (com `persistBoardOutsideTurn()`, a variante das duas
que podem cair dentro de um turno — decide "há turno aberto?" na mesma seção crítica do
retrato). Os dois:

- tiram o retrato sob `r.mu`, soltam, gravam fora do lock;
- são serializados por um `persistMu` próprio (ordem `persistMu` → `r.mu`), que envolve retrato **e** gravação — dois
  salvamentos concorrentes gravam na ordem em que os retratos foram tirados;
- falha é logada dizendo o que se perdeu, e não derruba a jogada (a política de sempre do
  `persistClosedTurn`).

⭐ **Por que o fechamento de turno, e não a abertura.** A peça da action anda na abertura, mas o
turno só é persistido no fechamento. Se o servidor cair no meio, o turno se perde — e a peça,
salva no fechamento anterior, volta junto com ele. Tabuleiro e histórico concordam.

**Decisão do dono do produto (2026-10-01): dentro de um turno aberto, nada é salvo antes do
fechamento.** Até então, um salvamento feito com o turno aberto — master action de peça,
interação/revelação de parede — retratava o tabuleiro como estava, já com o movimento da
abertura, e um reinício antes do fechamento deixava tabuleiro e histórico discordando. Agora
tudo o que acontece dentro do turno aberto (o movimento da abertura, as master actions do
mestre) fica durável **junto** com o fechamento dele: o fechamento grava, numa transação só
(`PersistTurnClose`), o turno, as master actions feitas nele (§4.8) e o tabuleiro com o fog de
cada jogador (`TurnCloseData.Board`/`Memories`, retratados por `boardSnapshotLocked` na mesma
seção crítica que drena o turno, depois dos escapes e antes de o próximo turno andar). Os três
verbos de fechamento não chamam mais `persistBoard`. Um reinício no meio do turno volta o
turno **inteiro** ao último fechamento. A sala fechar porque todos saíram (ou o hub parar) é,
para a persistência, um reinício: a próxima sala reidrata do banco, e nada do turno é salvo
naquele momento — só um log diz o que se perdeu. Nenhum verbo com a sala viva encerra um turno
sem fechá-lo: `change_scene` e o fechamento do round são recusados com turno aberto,
`change_round_mode` troca o regime do mesmo round e `kick_player` não toca no turno.

**Decisão do dono do produto (2026-10-02): um comando do mestre, uma transação.** O HP que o
fechamento aplica entra na mesma transação: os casos de uso de fechamento não escrevem mais
fichas (saiu o `persistDamage`, que fazia um `UPDATE character_sheets` por ficha atingida, fora
de transação e sob `r.mu`); a `Room` copia as barras (vida, estamina, aura) de cada ficha
atingida na seção crítica do `persistClosedTurn` (`TurnCloseData.StatusBars`) e o
`PersistTurnClose` as grava com o `UpdateStatusBars` do gateway da ficha rodando no `tx`. Se a
transação falhar, o dano continua na ficha em memória e a `Room` a guarda como não gravada
(`unwritten.go`): o próximo fechamento bem-sucedido a grava junto com as fichas que ele mesmo
atingiu. Nunca todas as fichas da sessão — isso atropelaria uma edição feita por REST no meio da
partida. Só um reinício antes desse próximo fechamento perde o HP não gravado.

Ficam de fora do "tudo junto": a troca de regime (`match_events` e `rounds.mode`, gravados na
hora — pertencem ao round, não ao turno); e a inscrição de NPC, por `add_npc` ou pelo pôr de um
NPC que não participava (B11 — `match_participants` na hora, sobrevive sem peça). Ver
`docs/dev/api/match-combat-ws.md` §9.

#### Quem move o quê

| Momento | Quem | Como | Validação |
|---|---|---|---|
| **lobby** | mestre | `piece_moved` / `piece_removed` | qualquer peça |
| **lobby** | jogador | `piece_moved` | só peça **existente** de personagem **dele**; não cria, não remove |
| **partida** | mestre | `enqueue_master_action` com `move` (arrastar ou **pôr**) ou `remove` (**tirar**) | ver abaixo |
| **partida** | jogador | só por ação | `piece_moved`/`piece_removed` recusados com `forbidden` |

A posse do jogador no lobby vem da ficha (`GetCharacterSheetRelationshipUUIDs`), porque no lobby
não existe `charToPlayer`.

**Master action de peça** (B14, e o `move` de B9):

```json
{ "type": "enqueue_master_action", "payload": { "targetIds": ["<sheetUUID>"], "move": { "position": [3, 4, 0] } } }
{ "type": "enqueue_master_action", "payload": { "targetIds": ["<sheetUUID>"], "remove": {} } }
```

- `targetIds[0]` é o personagem; `move.category`, `speed` e `charge` são ignorados (o arrastar
  não é movimento de jogo, não rola nem cobra barra).
- **Aplica na hora, com ou sem turno aberto** (ao vivo). Com turno aberto, também é pendurada
  no turno (`EnqueueMasterAction`); sem turno, o `ErrNoActiveTurn` não impede nada.
- **Pôr**: personagem sem peça ganha uma peça nova (`id` novo, forma do slot pela grade,
  visível, `z` 0). Se não é participante e é NPC do mestre → inscrição de B11 + `npc_added` +
  `bars_updated`. Personagem de jogador que não é participante → `error` `not_participant`.
- **Tirar**: remove a peça. Não desinscreve, não mata, não apaga histórico.
- Sai `piece_moved`/`piece_removed` com projeção de fog **para todos, inclusive o mestre** (a
  tela dele espera a confirmação, F12, então não há eco a suprimir). O mestre recebe
  `master_action_enqueued` — **só ele**: o eco carrega a posição, e para peça escondida isso
  vazaria para a mesa.
- Persistência: o tabuleiro + uma linha em `master_actions` (§4.8), com o `turnId` quando
  há turno aberto e o que cada jogador viu dela — na hora sem turno aberto; com turno aberto,
  as duas esperam o fechamento dele (`persistBoardOutsideTurn`, "Quando persiste" acima).

#### B11 — NPC no mapa é NPC da partida

- **Na sessão que nasce** (`InitMatchSessionUC.Init`, que roda no `start_match` **e** na
  reidratação depois de um reinício): antes de listar os participantes, lê o tabuleiro e, para
  cada peça cujo personagem não é participante, chama o `AddMatchNPCUC.Add` que já existe.
  `ErrNPCAlreadyInMatch` é sucesso; `ErrSheetNotNPC`, `ErrSheetNotOwnedByMaster` e
  `ErrCharacterSheetNotFound` são pulados com log (peça de jogador não inscrito não é NPC). É
  idempotente, e conserta as partidas que já existem sem tocar no banco à mão.
- No `start_match`, a `Room` salva o tabuleiro do lobby **antes** do `Init`, para que ele leia o
  que está na tela.
- **Com a sala viva**: pelo "pôr" acima. O braço do `add_npc` e o "pôr" chamam o **mesmo**
  método da `Room` (`enrollLiveNPC`), então os dois avisam a mesa igual: `npc_added`.

#### B5, B6 e B10 — geometria do movimento

- **B6.** A origem deixa de vir do cliente. Com o servidor dono do tabuleiro, `from` é **a
  posição da peça do ator no servidor**, no enqueue. `Move.From` vira `*[3]int` (nil = ator sem
  peça, e aí não há o que checar). O `from` do payload é ignorado; o contrato diz isso. Linhas
  antigas do banco com `[0,0,0]` continuam lendo como origem `(0,0)` — é só exibição de
  histórico.
- **B5 e B10.** A checagem de parede passa a usar `mapservice.SlotCenterToWorld` com a grade da
  sessão — o **centro** do slot, e a conversão certa para grade hexagonal (hoje multiplica
  `col × cellSize`, o que está errado nas duas grades). A convenção fica escrita no contrato:
  `position`/`from` = `[a, b, z]`, com `(a, b)` = `(col, row)` na grade quadrada e `(q, r)`
  axial na hexagonal, e `z` ainda não lido pelo servidor (o `applyMove` preserva o `z` da peça).
  Testes com as duas grades.

### 4.4 B13 — o escape

**Motor** (`service.ResolveReaction`, puro, sem I/O):

- Para todo `reactionKind` com `Displaces()`: `movePassed = Move != nil && Move.FinalSpeed >=
  HitTotal` (o `FinalSpeed` já é o Accelerate rolado no Dash e o Brake passivo no Shift —
  `deriveSpeeds`); `dodgePassed = Dodge.Total >= HitTotal`; **`Avoided = dodgePassed &&
  movePassed`**.
- Não evitou → `escapeGuard` cai para a defesa (é `KeepsDefault`); os outros tomam o golpe
  inteiro. É o fluxo que já existe depois do `Avoided`; só a condição muda.
- `ReactionOutcome` e `CharacterResult` ganham `Escape *EscapeResult{MovePassed, DodgePassed,
  Escaped, Landing *[3]int}` — `nil` fora dos escapes.
- Comentário no ponto onde a soma do movimento à esquiva entraria (regra conhecida, não
  implementada), apontando para o §6A.5 B13 do mestre.

**A escolha do mestre — caminho completo, não o fallback.** Ele escolhe onde a peça cai pelo
`edit_action`, que **já é** a superfície de edição da resolução do turno aberto (é onde a Fase 8
vai editar rolagens):

```json
{ "type": "edit_action", "payload": { "actionId": "<reactionId>", "escapeLanding": { "position": [5, 2, 0] } } }
{ "type": "edit_action", "payload": { "actionId": "<reactionId>", "escapeLanding": { "position": null } } }
```

- Guardada no `Turn` (`SetEscapeLanding`/`ClearEscapeLanding`), por `reactionId`. O resolvedor a
  copia para `CharacterResult.Escape.Landing`.
- Só vale para uma reação do turno aberto com `Displaces()`; posição dentro da grade. Pode ser
  escolhida a qualquer momento com o turno aberto — o escape pode passar a falhar ou a passar
  quando o mestre abre outras reações —, e **só é usada se, no fechamento, ele falhou**.
- Volta o `resolution_updated` recalculado, como toda edição.
- Por que não o fallback: a superfície (`edit_action`), o lugar de guardar (`Turn`) e o de
  persistir (`turns.resolution`) já existem. O custo é um campo em cada um.

**Tabuleiro:**

- `open_reaction` **não desloca mais nenhum escape** (sai o ramo do Shift, `room.go` ~843).
- `applyClosedEscapes`, nos três verbos que fecham: `Escaped` → destino; senão, `Landing` →
  onde o mestre escolheu; senão, fica. O ramo da falha tem o comentário pedido, apontando para
  o §6A.5 B13.
- **Wire:** `CharacterResultPayload.escape` (`escaped`, `movePassed`, `dodgePassed`,
  `awaitsMaster` = falhou e sem `landing`, `landing`). Enquanto o turno está aberto, isso é
  master-only como o resto da resolução; fechado, vai projetado para todos (números públicos).
  Persistido em `turns.resolution` (`resolution_record.go`) e devolvido no histórico.
- **Contrato:** sai *"Dano e deslocamento são desfechos INDEPENDENTES"*, sai a tabela "quando a
  peça anda" por categoria; entra a regra única.

### 4.5 B15 — o histórico guarda o que não é turno

- **Cena e round são persistidos quando nascem**, não no primeiro turno fechado: no
  `start_match`/reidratação (os ativos), no `change_scene` e quando o round acaba (nenhuma ação
  na fila consegue mais pagar o preço) e outro nasce. O insert idempotente que hoje mora dentro
  de `PersistTurnClose` vira um método do repositório (`EnsureSceneAndRound`), chamado dos dois
  lugares. O `finished_at` da cena é gravado por `CloseSceneAndRound`; o do round, pelo mesmo
  upsert idempotente (o `COALESCE` do `finished_at` é o SQL de fechar um round). Com o
  "persistir ao nascer", o `if sceneWasPersisted` deixa de pular fechamentos.
- **O fim do round e o nascimento do seguinte vão juntos** (dono do produto, 2026-10-02 — um
  comando do mestre, uma transação). Quando o `open_next_action` fecha o último turno e acaba o
  round, os dois entram na transação do turno (`TurnCloseData.NextRound`); quando acaba o round
  sem fechar turno, numa transação só deles (`PersistRoundClose`). Se a transação do turno
  falhar, o turno se perde (logado), mas o fim do round — que aconteceu na mesa — ainda vai com o
  seguinte pelo `PersistRoundClose`. Se também esse falhar (ou o `PersistRoundClose` do caminho
  sem turno), a `Room` guarda o fim como não gravado e o põe na transação da próxima gravação do
  round seguinte (`TurnCloseData.UnwrittenRoundEnds`, ou `PersistRoundClose` com as pontas no
  lugar do `EnsureSceneAndRound`): o round seguinte nunca vira linha aberta ao lado de um anterior
  ainda aberto, e o banco nunca fica com duas rodadas abertas na cena. O que o
  `CloseRound` liquida em memória (saldo das barras, modificadores de fim de round) não é durável
  em lugar nenhum — não há o que mais pôr na transação.
- **Tabela nova `match_events`** para o que acontece **dentro** de um round e não é turno:
  `(uuid, match_uuid, scene_uuid, round_uuid, turn_uuid NULL, kind, payload JSONB, created_at)`.
  `turn_uuid` não tem FK: o turno só é gravado no fechamento, e pode nunca ser (reinício).
  Tipo: `roundModeChanged {from, to}`. (As master actions **não** vão aqui: têm tabela própria,
  §4.8. A tabela de eventos fica para o que não é ação de ninguém.)
- **REST:** o `GET /matches/{uuid}/history` passa a usar `LEFT JOIN`, então cena e round sem turno
  aparecem; cada round ganha `events: [...]`, em ordem de `createdAt` — os `roundModeChanged` e as
  **master actions fora de turno** (§4.8), no mesmo mecanismo, cada entrada com um `kind`. Vêm de
  consultas separadas, costuradas na árvore por `round_uuid`, para não multiplicar o join grande.
  Round fechado = `finishedAt` do round; troca de cena = a própria cena.
- **Visibilidade:** `roundModeChanged` é público (o regime é público). As master actions seguem a
  projeção do §4.8. O `move` de cada action e o `escape.landing` saem a cada leitor como ele os
  viu ao vivo, pelo veredito gravado no turno (§4.1; dono do produto, 2026-10-01).
- **Conserto do `match-history.md`**: `category` é `battle`/`roleplay`, `mode` é `Free`/`Race`,
  e o `move` passa a ser mostrado.

### 4.6 B4 e B7 — a conexão

- **B4 — mesma pessoa, duas conexões: a última vence.** No `register`, se já há um cliente
  daquele usuário e não é este, o velho recebe `error` `connection_replaced` e é fechado
  (`Close()`, que sinaliza `done` — nunca fechar `send`). O guarda de ponteiro do `unregister`
  que já existe impede que a saída do velho tire o novo da sala.
- **B7 — `Register` numa sala fechada não bloqueia.** A `Room` ganha `done`, fechado quando o
  `Run` retorna. `Register` e o `unregister` do `ReadPump` fazem `select` com `done`. `Register`
  devolve `ErrRoomClosed`; o handler tenta uma vez mais com `GetOrCreateRoom` (mestre) ou
  responde `lobby_not_open` (jogador), como hoje sem sala.

### 4.7 B8 — o dono recebe o próprio `private`

`toParticipantResponse` recebe o UUID de quem pede: `private` sai quando quem pede é o mestre
**ou** é o jogador dono da ficha. O NPC continua sem dono. Contrato em `match.md` (participantes).

### 4.8 Master actions persistidas — decisão do dono do produto

**Tabela própria, separada de `actions`.** Não é preferência, é o modelo: `actions.actor_uuid`
referencia `character_sheets` (PR #69) e o ator de uma master action é o **mestre**, que é
usuário; `actions.turn_uuid` é `NOT NULL` e a master action acontece **fora de turno** (o
arrastar entre turnos é o caso comum). Misturar obrigaria a afrouxar as duas colunas e a filtrar
um tipo do outro em toda leitura.

```sql
CREATE TABLE master_actions (
  uuid         UUID        PRIMARY KEY,
  match_uuid   UUID        NOT NULL REFERENCES matches(uuid) ON DELETE CASCADE,
  scene_uuid   UUID        NOT NULL REFERENCES scenes(uuid),
  round_uuid   UUID        NOT NULL REFERENCES rounds(uuid),
  turn_uuid    UUID,                    -- NULL = fora de turno; sem FK (ver abaixo)
  master_uuid  UUID        NOT NULL REFERENCES users(uuid),
  kind         VARCHAR(32) NOT NULL,    -- movePiece · placePiece · removePiece · wallInteract · revealWall · turnNote
  content      JSONB       NOT NULL,    -- o que ela carrega: alvos, move (de/para), interact, …
  views        JSONB       NOT NULL,    -- o que cada jogador viu dela ao vivo (abaixo)
  happened_at  TIMESTAMPTZ NOT NULL
);
```

- **O que entra:** toda `enqueue_master_action` **aceita** — as de peça (§4.3), as de parede
  (interagir, revelar) e as genéricas que hoje só se penduram no turno aberto (`turnNote`).
  **`edit_action` não é master action** e não entra: a edição continua registrada só em
  `overridden_action_values`, como hoje. (A sessão pendura as duas coisas em
  `Turn.masterActions`; por isso a gravação **não** lê `t.GetMasterActions()` — ela acontece no
  braço do `enqueue_master_action`, onde só as de verdade passam.)
- **Quando grava** (decisão do dono do produto, 2026-10-01): **sem turno aberto, no instante**
  em que é aplicada — o mesmo momento em que B3 salva o tabuleiro e B15 grava os eventos.
  **Com turno aberto, no fechamento desse turno**, na mesma transação do turno: o registro é
  montado no instante (views, `turn_uuid`, `happened_at` daquele momento) e guardado pela sala
  (`turnWrites`, por turno) até um dos três verbos de fechamento entregá-lo a
  `PersistTurnClose` (`TurnCloseData.MasterActions`), que o insere com o `Insert` do próprio
  gateway de master actions rodando na transação — a mesma em que vai o tabuleiro (§4.3).
  Turno, master actions e tabuleiro entram juntos ou nenhum entra; uma falha é logada dizendo
  que se perderam o turno, as master actions dele e o tabuleiro que ele deixou.
  Um reinício (ou a sala fechar) com o turno aberto perde o turno e as master actions dele —
  é o ponto: o turno inteiro volta ao último fechamento. `scene_uuid`/`round_uuid` são os ativos
  (garantidos por `EnsureSceneAndRound`, §4.5; dentro do turno nenhum dos dois muda, e
  `PersistTurnClose` garante o mesmo par na sua transação). `turn_uuid` continua sem FK: o turno
  só é gravado no fechamento, e linhas gravadas antes desta decisão podem apontar um turno que um
  reinício perdeu — o histórico as mostra fora de turno (abaixo).
- **Onde aparece no `GET /history`:** com `turn_uuid` de um turno que está na árvore → dentro
  daquele turno, em `turns[].masterActions`, na ordem do tempo — e, desde 2026-10-01, só a partir
  do fechamento (antes ela não está em lugar nenhum). Sem turno (ou, em linhas antigas, com um
  turno que não foi gravado) → em `rounds[].events`, como entrada `kind: "masterAction"`, na
  ordem do tempo junto com os `roundModeChanged`.
- **Projeção — cada leitor vê como viu ao vivo.** No instante da aplicação, o servidor já decide,
  jogador a jogador, o que cada um recebe (o portão de fog do `relayPieceMove`, o
  `broadcastWallStateChangedGated`). A mesma decisão é **gravada** em `views`, para **todo jogador
  da sessão** (conectado ou não — o que conta é o fog dele naquele instante, lido do cache que já
  existe para todos):

| `views[jogador]` | ao vivo ele recebeu | no histórico ele vê |
|---|---|---|
| `full` | o `piece_moved` (ou a parede mudando, ou a remoção) | a master action inteira |
| `left` | só o `piece_removed` — viu a peça sair e não viu para onde | a master action **sem o destino** |
| ausente | nada | **nada** — a entrada não existe para ele |

  O mestre vê tudo. Revelar parede vai a todos ao vivo, então é `full` para todos. `turnNote` não
  chega à mesa ao vivo (é pendurada no turno e não emite nada), então fica só para o mestre. A
  projeção roda no `GetMatchHistoryUC`, ao lado do `ProjectAction` — o REST não filtra nada.

## 5. Reinício, recarga, queda (§0.2 do mestre)

| Estado | Recarregar o cliente / reconectar | Reiniciar o servidor |
|---|---|---|
| Tabuleiro (posições, paredes, fog) | `map_full_state` do servidor | **volta** do `match_boards` + `player_memories` (B3) |
| Fila | mestre: `queue` (B1) · dono: `ownQueue` (B12) | **perdida**; `ownQueue: []` faz o front descartar com aviso e devolver o rascunho. Ninguém reenvia |
| Turno aberto, reações anexadas, master actions feitas dentro dele | `openTurn` com `action` (B2); mestre: `resolution` | **perdido inteiro** (o turno só persiste ao fechar, e o que aconteceu dentro dele vai junto — decisão de 2026-10-01, §4.3); a peça da action e o que o mestre mudou no tabuleiro voltam ao último fechamento, e as master actions feitas no turno não chegam a ser gravadas. Exceção: a inscrição de um NPC posto no turno (B11) sobrevive |
| Escolha do mestre para um escape (B13) | vem na `resolution` do mestre | perdida com o turno aberto |
| Histórico | REST | REST (B15) — nada que já fechou se perde; master actions fora de turno gravadas no instante, as de dentro de um turno com o fechamento dele (§4.8) |
| Barras | `bars` com o `seq` atual | zeradas (o de sempre: perdido, não divergente) |
| HP (vida, estamina, aura da ficha) | `character_hp_changed` ao vivo; a ficha por REST | **volta** de `character_sheets` como o último fechamento **gravado** o deixou — gravado na transação do turno (decisão de 2026-10-02, §4.3); o HP de um fechamento cuja transação falhou vai com o próximo fechamento bem-sucedido, e só se perde num reinício antes dele |
| NPC do mapa | — | reinscrito, idempotente, no `Init` (B11) |
| Duas abas | a última vence (B4) | — |

A verificação inclui recarregar no meio de um turno e reiniciar o `cmd/game` no meio de uma fila
(§7).

## 6. Contratos que mudam

| Arquivo | O quê |
|---|---|
| `docs/dev/api/match-combat-ws.md` | B1 (`action_queued.action`, `queue`), B2 (`turn_opened.action`, `openTurn.actionId/action`, a ordem nova), B12 (`ownQueue` + regra), B13 (escape, `edit_action.escapeLanding`, `escape` na resolução), B14 (master action `move`/`remove`, `piece_moved` só no lobby, `map_state_sync` obsoleto, `unknown_wall`, `not_participant`), B6/B10 (convenção de coordenada, `from` derivado), B4 (`connection_replaced`), B9 (`attack` recusado) |
| `docs/dev/api/game-lobby.md` | `piece_moved`/`piece_removed` validados no lobby; o servidor carrega o tabuleiro |
| `docs/dev/api/match-history.md` | B15 (`events`, cena/round sem turno), master actions (`turns[].masterActions`, `events` com `kind: "masterAction"`, projeção por `views`), `escape` na resolução, `from` opcional, exemplos corrigidos; sai a frase *"as master actions nunca são persistidas"* |
| `docs/dev/api/match-maps.md` | B16 (`inheritBoardFromMatchUuid`), anexar outro mapa apaga o tabuleiro velho |
| `docs/dev/api/match.md` (participantes) | B8 |
| `docs/dev/api/match-npcs.md` | B11 (inscrição pela peça e pelo `Init`) |
| `docs/documentation-map.yaml` | os pacotes novos (`actionwire`, `matchboard` ×3, eventos) |
| `AGENTS.md` / `docs/dev/match/combat-engine.md` | Known Issues atualizados: tabuleiro persistido, escape novo |

Cada tarefa do plano atualiza o contrato **no mesmo commit** que muda o wire.

## 7. Testes e verificação

- **TDD por camada** (`integration-tests.instructions.md`): domínio com testes unitários
  (escape); gateways novos com testes de integração `//go:build integration` (`match_boards`,
  `player_memories`, `match_events`, histórico com `LEFT JOIN`, herança de B16); `room.go` com os
  e2e do pacote `game` (padrão `combat_e2e_test.go`), usando repositórios falsos.
- **Reinício simulado nos e2e:** uma `Room` nova sobre o **mesmo** repositório falso de
  tabuleiro e rodada — posições, paredes e memórias voltam; `ownQueue` vem vazio.
- **Teste de ouro do REST** (T6): o JSON do histórico antes e depois do `actionwire` é idêntico.
- Por tarefa: `go build ./...`, `go vet ./...`, `go vet -tags integration ./...`, `go vet -tags
  smoke ./...`, `go test ./internal/...`, `go test -race ./internal/app/game/` quando toca
  `room.go`, e a suíte de integração com `-p 1` quando toca gateway.
- **Ponta a ponta (entrega):** `make run-dev`, smoke por `curl` nos REST afetados (histórico,
  participantes, anexar com herança) e por um cliente WS de linha de comando nos fluxos de
  mestre + dois jogadores, **incluindo reiniciar o `cmd/game` no meio de uma fila e de um
  turno**. A verificação no browser com três contas é do PR de front (F1/F10 dependem deste); o
  PR de back diz isso e deixa o ambiente pronto com `./dev-checkout.sh`.

## 8. Effort e modelos

- Esta sessão planejou em **high**, como o mestre recomenda. Não mudei.
- Implementação: um subagente por tarefa, **`sonnet`**, despachado sem tipo customizado (sem
  arriscar uma chave de frontmatter de effort que eu não conferi). As tarefas com mais
  integração em `room.go` (T5, T8, T12) pedem revisão de código por um subagente separado antes
  de seguir.

## 9. Decisões desta sessão

| Decisão | Por quê |
|---|---|
| Um pacote `actionwire` com três níveis de corte | o mestre pede um formato só; o corte é número, a deny-list continua no `ProjectAction` |
| `turn_opened` e `turn_closed` na pista direta | projetar o `turn_opened` sem isso inverteria a ordem com o `turn_closed` |
| `ownQueue` sempre presente para não-mestre | o front precisa distinguir "nada" de "servidor antigo" |
| Tabuleiro como retrato inteiro em `match_boards`, com `bg` nulo = herda | B16 vira uma cópia de linha; editar o mapa da campanha não toca partida; o editor futuro tem onde escrever |
| Persistir o tabuleiro no fechamento de turno, não na abertura | tabuleiro e histórico caem juntos num reinício |
| O HP do fechamento na transação do turno (2026-10-02) | **dono do produto** — um comando do mestre, uma transação; e nenhum I/O de ficha sob `r.mu` |
| O fim do round e o round seguinte na transação do turno que o mesmo comando fechou, ou numa só deles (2026-10-02) | **dono do produto** — um comando do mestre, uma transação; nunca duas rodadas abertas na cena |
| `map_state_sync` aceito e ignorado, não recusado | não quebrar o front que está no ar antes do F13 |
| Lobby recarrega o tabuleiro a cada conexão do mestre | é quando o `map_state_sync` chegava; edição de mapa antes do primeiro movimento aparece |
| `from` derivado da peça no servidor | mata o sentinela de B6 em vez de trocá-lo por outro |
| Escolha do escape pelo `edit_action`, guardada no `Turn` | é a superfície de edição da resolução; caminho completo, sem fallback |
| Master actions em tabela própria, gravadas no instante, projetadas pelo que cada jogador viu ao vivo (`views`) | **dono do produto** (tabelas separadas, as duas no histórico, projeção "como viu ao vivo"); o formato é desta sessão |
| Duas conexões: a última vence | recarregar a aba é o caso comum |
| `attack` sai do `MasterActionPayload` | dono do produto — o mestre ataca pelo NPC; efeito de ambiente é futuro |

## 10. Fora de escopo

O editor de mapa da partida (só não fecho a porta: `bg`); a soma do movimento à esquiva (fica o
comentário); regras de colisão; o verbo de cancelar ação; tudo do front.
