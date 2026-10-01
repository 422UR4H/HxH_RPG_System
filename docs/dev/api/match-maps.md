# Match Maps API

## Overview

Endpoints to attach, retrieve, and detach a tactical map from a match. One map per match. Operations blocked after match starts.

## REST Endpoints

### POST /matches/{match_uuid}/map

Attach a map to a match. Replaces any previously attached map (upsert).

**Attaching a DIFFERENT map than the one currently attached deletes the match's own board**
(`match_boards` row and its `player_memories`, see "O tabuleiro da partida" below) — it was a
portrait of the old map and means nothing on the new one. Attaching the SAME map again leaves
the board untouched. Neither applies before the match's first board save; see "Ciclo de vida".

**Inheriting another match's board (B16).** Optionally, `inheritBoardFromMatchUuid` names a
match whose board (pieces, wall state, and each player's fog) becomes this match's starting
board instead of a fresh snapshot of `mapUuid` — "uma partida começa de onde outra terminou".
Requirements, checked in this order:
- the source match cannot be **this same match** (`ErrSourceMatchIsTheSameMatch`) —
  inheriting from yourself would have nothing left to copy from;
- the source match must be in the **same campaign** as this one (`ErrSourceMatchNotInCampaign`);
- the source match must actually **have** a board to inherit (`ErrSourceMatchHasNoBoard`);
- that board must already be on **`mapUuid`** — the map this request is attaching
  (`ErrSourceMatchOnAnotherMap`).

When inheritance is requested, the delete-on-different-map rule above does not run — the
inherited board replaces whatever was there instead.

**Auth:** required (JWT Bearer)  
**Role:** match master only

**Request body:**
```json
{
  "mapUuid": "uuid",
  "inheritBoardFromMatchUuid": "uuid"
}
```
`inheritBoardFromMatchUuid` is optional; omit it for a plain attach/re-attach.

**Example — inheriting from a previous session's match:**
```json
POST /matches/8f14e2.../map
{
  "mapUuid": "76987813-409d-4d61-92d7-9d86aaf824c8",
  "inheritBoardFromMatchUuid": "6b4f636c-dc55-4a85-a761-0be84357a54c"
}
```
```json
200 OK
{
  "matchMap": {
    "matchUuid": "8f14e2...",
    "mapUuid": "76987813-409d-4d61-92d7-9d86aaf824c8",
    "attachedAt": "2026-09-30T12:00:00Z"
  }
}
```
The response shape is unchanged — the inherited board is not echoed here; `GET
/matches/{match_uuid}/map` and the game server's own board load (`LoadMatchBoardUC`) are what
surface it.

**Responses:**
| Status | Description |
|--------|-------------|
| 200 | Map attached. Body: `{"matchMap": {"matchUuid": "...", "mapUuid": "...", "attachedAt": "ISO8601"}}` |
| 400 | Bad request (invalid UUID) |
| 401 | Unauthenticated |
| 403 | Not the match master |
| 404 | Match or map not found (also the source match of `inheritBoardFromMatchUuid`, when absent) |
| 422 | Match already started, or (B16) the source match is itself, is not in this campaign, has no board to inherit, or is on a different map |
| 500 | Internal server error |

---

### GET /matches/{match_uuid}/map

Get the map currently attached to a match.

**Auth:** required (JWT Bearer)

**Responses:**
| Status | Description |
|--------|-------------|
| 200 | Map attached. Body: `{"matchMap": {"matchUuid": "...", "mapUuid": "...", "attachedAt": "ISO8601"}}` |
| 204 | No map attached |
| 400 | Bad request |
| 401 | Unauthenticated |
| 500 | Internal server error |

---

### DELETE /matches/{match_uuid}/map

Detach the map from a match.

**Auth:** required (JWT Bearer)  
**Role:** match master only

**Responses:**
| Status | Description |
|--------|-------------|
| 204 | Map detached |
| 400 | Bad request |
| 401 | Unauthenticated |
| 403 | Not the match master |
| 404 | Match or map not found |
| 422 | Match already started |
| 500 | Internal server error |

---

## O tabuleiro da partida

Desde a Fase 6 de fechamento de combate (spec `2026-09-27-combat-closure-back-design.md`
§4.3), a partida deixou de olhar direto para o mapa da campanha depois que anexa um: ela
ganha o próprio **retrato**, gravado em `match_boards` (uma linha por `match_uuid`, chave
estrangeira para `matches ON DELETE CASCADE`).

**Estrutura da linha:** `match_uuid`, `map_uuid` (de qual mapa ele partiu), `grid`, `bg`
(NULL = herda o fundo do mapa), `pieces` (mesmo formato de `maps.pieces`), `walls` (as
paredes **inteiras**, com estado — aberta, trancada, HP, destruída, revelada) e
`updated_at`.

**É um retrato, não um diff.** Por isso:
- Editar o mapa da campanha com a partida rolando **não muda nada nela** — o retrato já
  se separou do mapa no instante em que foi salvo.
- Duas partidas no mesmo mapa **nunca se tocam** — cada uma tem sua própria linha.
- `bg` nulo herda o fundo do mapa **no momento em que é lido**, não uma cópia congelada:
  o editor de mapa da partida (trabalho futuro) é o único que escreveria um valor aqui,
  e ninguém o faz ainda.

**Ciclo de vida.** A linha nasce no **primeiro salvamento**. Antes disso, carregar =
devolver um retrato fresco do mapa anexado, sem gravar nada (`LoadMatchBoardUC`, spec
§4.3 "Quem carrega", B14). Anexar **outro** mapa antes do início da partida apaga a linha
velha — ela era de outro mapa; depois do início, anexar é recusado. Desanexar não apaga
nada, então desanexar e anexar outro mapa pode deixar a linha velha no banco: por isso
carregar só aceita a linha cujo `map_uuid` é o do mapa anexado **agora** — uma linha de
outro mapa conta como ausente, e o que vem é o retrato fresco do mapa novo. O `start_match`
recarrega o tabuleiro antes de gravá-lo, então a partida começa no mapa anexado no
momento do início, mesmo que ele tenha mudado com a sala do lobby aberta. Se o mapa foi
**desanexado** com o lobby aberto, essa recarga (ou a de uma reconexão do mestre) não acha
mapa nenhum e **esvazia** o tabuleiro da sala — peças, paredes, grade, fundo — e manda o
tabuleiro vazio (`map_full_state`) a quem está conectado; uma partida iniciada assim não
grava linha em `match_boards`.

**Herdar de outra partida (B16).** `POST /matches/{match_uuid}/map` aceita
`inheritBoardFromMatchUuid`: em vez de começar de um retrato fresco do mapa, a partida herda
o tabuleiro — posições, estado das paredes e fog — de uma partida de origem **diferente desta
mesma partida**, **da mesma campanha**, que **tem** um tabuleiro para herdar e cujo tabuleiro
já está no **mesmo mapa** sendo anexado (as quatro checagens, nessa ordem, cada uma com seu
erro 422 — `ErrSourceMatchIsTheSameMatch`, `ErrSourceMatchNotInCampaign`,
`ErrSourceMatchHasNoBoard`, `ErrSourceMatchOnAnotherMap`). A checagem de "é a mesma partida"
vem primeiro e sem tocar o banco: herdar de si mesmo apagaria, no meio da cópia, a própria
linha de onde o `INSERT … SELECT` ainda vai ler.

É literalmente um `INSERT … SELECT` da linha de `match_boards` e das linhas de
`player_memories` da origem **nesse mesmo mapa** (`map_id = <mapa do tabuleiro de origem>` —
a partida de origem é uma partida passada e pode ter memórias órfãs de um mapa anterior, que
não devem viajar), trocando o `match_uuid`/`match_id` (e, para cada memória, o `id`) —
`pgmatchboard.Repository.Copy`, numa única transação com a cópia do fog (dentro dela,
`fog.PlayerMemoryRepository.CopyMatch`). O tabuleiro antigo do destino, se havia algum, é
apagado por essa mesma cópia — não sobrevive misturado com o herdado.

**Anexe/herde com o lobby fechado.** Se a sala do lobby estiver aberta (o mestre conectado),
ela ainda guarda em memória o tabuleiro de antes, e o próximo salvamento dela — um
movimento ou remoção de peça no lobby — pode sobrescrever a linha recém-herdada (ou
recém-anexada, no mesmo mapa); anexe e herde com o lobby fechado.

**Quando persiste** (B3, spec §4.3 "Quando persiste"): a cada mudança definitiva do
tabuleiro — o movimento e a remoção de peça no lobby, `start_match` (antes de a sessão de
combate começar), os três verbos que fecham um turno (`close_turn`, `open_next_action`,
`pull_action`) e, **entre turnos**, as master actions de peça (mover/pôr/tirar) e a
interação/revelação de parede pelo mestre. Com um turno aberto nada é salvo antes do
fechamento dele (decisão do dono do produto, 2026-10-01): o fechamento salva o tabuleiro com
tudo o que aconteceu no turno, e um reinício no meio do turno volta ao último fechamento — ver
[`match-combat-ws.md`](match-combat-ws.md#reinício-recarga-queda). `Room.persistBoard(reason)`
é o único método que escreve; uma falha é logada e engolida — a jogada em memória continua
valendo, só a gravação que se perde (mesma política de `persistClosedTurn`).

**O fog do jogador vai junto.** Cada salvamento também grava a memória explorada de cada
jogador (`player_memories`, ver [`maps.md`](maps.md)) — a mesma gravação, para que um
reinício do servidor devolva tabuleiro e memória juntos e nunca um sem o outro.

**Reinício:** ver a tabela em [`match-combat-ws.md`](match-combat-ws.md#reinício-recarga-queda).

## WebSocket: `piece_moved` / `piece_removed` (client → server)

**Direction:** Client → Server, relayed per-recipient (fog-gated once a match has a session —
see [`match-combat-ws.md`](match-combat-ws.md#piece_moved-servidor); no fog in the lobby).
**When:** During the LOBBY phase only, as of B14 (spec §4.3, "Quem move o quê"). This pair used
to be named `lobby_piece_moved` here — the actual wire type has been `piece_moved` for a while;
this section was stale.

**Quem pode enviar o quê, e a validação do servidor:**

| Fase | Quem | `piece_moved` | `piece_removed` |
|---|---|---|---|
| lobby | mestre | qualquer peça | qualquer peça |
| lobby | jogador | só peça **já existente** do **próprio** personagem — não cria uma peça nova mandando um `pieceId` desconhecido, e não pode trocar o `characterId` de uma peça sua para outro | recusado sempre — remover é só do mestre |
| partida (sessão viva) | mestre | recusado — mover/pôr/tirar peça passa a ser `enqueue_master_action` (`move`/`remove`) | recusado, mesmo motivo |
| partida (sessão viva) | jogador | recusado — jogador só move peça agindo (`enqueue_action`) | recusado, mesmo motivo |

A posse do jogador no lobby é lida da ficha (`GetCharacterSheetRelationshipUUIDs`, via
`RoomDeps.SheetOwnership`), não de `charToPlayer` — esse mapa só existe com sessão viva. A
leitura é I/O e roda fora do lock da sala.

**Send payload:**
```json
{
  "type": "piece_moved",
  "payload": {
    "pieceId": "uuid-string",
    "characterId": "uuid-string",
    "slot": {
      "kind": "square",
      "col": 3,
      "row": 5
    },
    "z": 1.5
  }
}
```

Hex slot:
```json
{
  "slot": { "kind": "hex", "q": 2, "r": -1 }
}
```

```json
{ "type": "piece_removed", "payload": { "pieceId": "uuid-string" } }
```

**Relayed to other clients** (same shape, `senderId` is the mover's own UUID — see
[`match-combat-ws.md`](match-combat-ws.md#piece_moved-servidor) for the full per-recipient
table, which is shared with the combat engine's own server-authored moves):
```json
{
  "type": "piece_moved",
  "senderId": "user-uuid",
  "payload": {
    "pieceId": "...",
    "characterId": "...",
    "slot": { "kind": "square", "col": 3, "row": 5 },
    "z": 1.5
  }
}
```

**Notes:**
- `z` (elevation in metres, 0 = ground) is opaque passthrough, same as `slot`: the server
  never computes or defaults it, so the client must send it here if it wants the piece's
  elevation preserved.
- Refused with `error` `forbidden`. During a match, the message is one of
  `"during a match the master moves pieces with enqueue_master_action"` (master) or
  `"players move by action"` (player) — see the catalogue in
  [`match-combat-ws.md`](match-combat-ws.md#7-catálogo-de-erros). In the lobby, a player's
  disallowed move/removal is also `forbidden`, with a message describing the specific reason
  (piece not owned, piece does not exist yet, or trying to remove at all).
- Server-side piece ownership validation shipped in B14 (Task 4). The client SHOULD still
  restrict drag to allowed pieces for a responsive UI, but the server is now the actual
  authority — this replaces the earlier "no server-side validation, Phase 7+" note.
