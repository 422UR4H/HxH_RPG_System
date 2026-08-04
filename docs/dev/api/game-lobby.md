# Game Server — Lobby WebSocket Protocol

**Status:** In progress
**Server port:** 8081
**URL:** `ws://localhost:8081/ws?match_uuid=<uuid>&token=<jwt>&nickname=<name>`

## New Messages (Task 1: Lobby lifecycle)

### Server → Client

#### `lobby_closed`

Broadcast to all clients when the master cancels the lobby (via `cancel_lobby`).

```json
{ "type": "lobby_closed", "payload": "{}" }
```

#### `lobby_not_open`

Sent to a participant who tries to connect before the master has opened the lobby.

```json
{ "type": "lobby_not_open", "payload": "{}" }
```

The server immediately follows with a WebSocket close frame (code 4001, reason: "lobby not open") and closes the connection.

### Client → Server

#### `cancel_lobby`

Sent by the master to cancel the open lobby. Broadcasts `lobby_closed` to all connected clients and stops the room.

```json
{ "type": "cancel_lobby", "payload": "{}" }
```

Error if sender is not the master or room is not in lobby state:

```json
{ "type": "error", "payload": "{\"code\":\"forbidden\",\"message\":\"...\"}" }
```

#### `map_state_sync`

Enviada **apenas pelo mestre** para semear o tabuleiro em memória do game server.
O servidor não lê o mapa do banco: ele deriva a linha de visão de cada jogador a partir
das peças que recebe aqui. Sem peças, nenhum jogador tem origem de LOS e o fog cobre o
mapa inteiro.

```json
{
  "type": "map_state_sync",
  "payload": {
    "pieces": [
      {
        "piece_id": "<uuid>",
        "slot": { "kind": "square", "col": 18, "row": 19 },
        "character_id": "<sheet-uuid>",
        "visible": true
      }
    ],
    "walls": [ { "id": "<uuid>", "p1": [0, 0], "p2": [96, 0], "wall_type": "wall", "...": "..." } ],
    "grid": { "kind": "square", "cols": 35, "rows": 35, "cell_size": 96, "skew_ratio": 1 }
  }
}
```

**Semântica de `pieces`** — o campo é *nullable* e os dois casos são distintos:

| `pieces` | Efeito no tabuleiro |
|----------|---------------------|
| ausente / `null` | Nenhuma informação de peça neste sync — o tabuleiro atual é **preservado** |
| `[]` (presente e vazio) | O tabuleiro é **esvaziado** (autoritativo) |
| lista não vazia | O tabuleiro é **substituído** por essa lista |

Enviar `[]` quando o cliente simplesmente não carregou as peças ainda apaga todas as
origens de LOS. Por isso o cliente só deve emitir `map_state_sync` depois que o mapa REST
tiver chegado.

**Efeitos colaterais.** Com a partida em andamento (sessão viva), após semear o tabuleiro
o servidor recalcula a visibilidade de todos os jogadores e reenvia `map_full_state` para
cada cliente (filtrado por LOS para jogadores, completo para o mestre). Isso faz o estado
convergir independentemente da ordem de conexão — mestre antes ou depois dos jogadores, e
inclusive após reinício do servidor.

**Limite de tamanho.** O frame carrega o tabuleiro inteiro e cresce com o número de
paredes e peças; o limite de leitura do servidor é 1 MiB (`maxMessageSize` em
`internal/app/game/client.go`). Um limite menor faz o servidor fechar a conexão do mestre
no meio do sync — o mestre reconecta e reenvia em loop, e nenhum jogador recebe tabuleiro.

**Paredes que o jogador recebe.** Uma parede é enviada quando qualquer trecho dela está
na linha de visão do jogador — o teste amostra ao longo do segmento e desloca cada amostra
em direção ao observador. Isso é necessário porque uma parede que **bloqueia** a visão fica
exatamente sobre a borda do polígono de visibilidade: testar o ponto médio pela regra de
contenção responde "não visível", e a parede some da tela do jogador justamente quando ele
mais precisa dela (para abrir uma porta, arrombar, atacar). Paredes atrás de outra
continuam ocultas. Em modo `explored`, paredes já vistas permanecem visíveis.

Erro se o remetente não for o mestre:

```json
{ "type": "error", "payload": "{\"code\":\"forbidden\",\"message\":\"...\"}" }
```
