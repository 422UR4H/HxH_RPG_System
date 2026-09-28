# Match Maps API

## Overview

Endpoints to attach, retrieve, and detach a tactical map from a match. One map per match. Operations blocked after match starts.

## REST Endpoints

### POST /matches/{match_uuid}/map

Attach a map to a match. Replaces any previously attached map (upsert).

**Auth:** required (JWT Bearer)  
**Role:** match master only

**Request body:**
```json
{
  "mapUuid": "uuid"
}
```

**Responses:**
| Status | Description |
|--------|-------------|
| 200 | Map attached. Body: `{"matchMap": {"matchUuid": "...", "mapUuid": "...", "attachedAt": "ISO8601"}}` |
| 400 | Bad request (invalid UUID) |
| 401 | Unauthenticated |
| 403 | Not the match master |
| 404 | Match or map not found |
| 422 | Match already started |
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
velha — ela era de outro mapa; depois do início, anexar é recusado.

**Quando persiste** (B3, spec §4.3 "Quando persiste"): a cada mudança definitiva do
tabuleiro — o movimento e a remoção de peça no lobby, `start_match` (antes de a sessão de
combate começar), os três verbos que fecham um turno (`close_turn`, `open_next_action`,
`pull_action`), e a interação/revelação de parede pelo mestre. `Room.persistBoard(reason)`
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
