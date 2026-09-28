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

#### O tabuleiro é do servidor (B14, spec §4.3 "Quem carrega")

O game server carrega o tabuleiro sozinho — do banco (`match_boards`, a linha salva da
partida; ou, se a partida nunca salvou uma, um retrato do mapa anexado a ela), não do
cliente:

- **quando a sala nasce** — o primeiro cliente (sempre o mestre, único caminho que cria a
  `Room`) a se conectar dispara a leitura;
- **enquanto a partida ainda é lobby** (sem sessão viva), **a cada conexão do mestre** —
  reconectar (aba nova, F5) recarrega, o que cobre uma edição do mapa antes do primeiro
  movimento;
- **depois que a partida começa, só no nascimento** — um mestre que reconecta em pleno jogo
  não reseta o tabuleiro ao vivo por baixo da sessão.

O `map_full_state` que cada cliente recebe ao se conectar (ver `piece_moved`/`piece_removed`
em [`match-combat-ws.md`](match-combat-ws.md)) já reflete esse tabuleiro, filtrado por LOS do
mesmo jeito de sempre — completo para o mestre, recortado por linha de visão para o jogador.

#### `map_state_sync` (obsoleto desde B14)

**Não escreve mais nada no tabuleiro.** O servidor aceita a mensagem e **ignora o
payload inteiro** — `pieces`, `walls` e `grid` não têm efeito algum — e responde **só ao
remetente** (deve ser o mestre; `forbidden` para qualquer outro) com o `map_full_state`
**atual do servidor**, não o que o payload carregava. Mantido só para o front que ainda
manda esse sync não quebrar; será removido do contrato quando a Fase 13 tirar o envio do
front.

```json
{ "type": "map_state_sync", "payload": { "pieces": [], "walls": [], "grid": null } }
```

Antes de B14 este verbo era como o mestre semeava o tabuleiro em memória do servidor — o
servidor não lia o mapa do banco, e derivava a linha de visão de cada jogador a partir do
que chegava aqui. A seção anterior descreve como o servidor faz isso agora, sozinho.

Erro se o remetente não for o mestre:

```json
{ "type": "error", "payload": "{\"code\":\"forbidden\",\"message\":\"...\"}" }
```

#### Paredes que o jogador recebe em `map_full_state`

Uma parede é enviada quando qualquer trecho dela está na linha de visão do jogador — o
teste amostra ao longo do segmento e desloca cada amostra em direção ao observador. Isso é
necessário porque uma parede que **bloqueia** a visão fica exatamente sobre a borda do
polígono de visibilidade: testar o ponto médio pela regra de contenção responde "não
visível", e a parede some da tela do jogador justamente quando ele mais precisa dela (para
abrir uma porta, arrombar, atacar). Paredes atrás de outra continuam ocultas.

Em modo `explored`, toda parede que passou nesse teste é gravada na **memória** do
jogador e continua a ser enviada mesmo depois que ele sai da linha de visão. A memória
registra a parede observada (por id), não a região do mapa: um modelo por célula gerava
falso negativo (parede vista cujo centro de célula nunca entrou no polígono era
esquecida) e falso positivo (trecho ocluído numa célula iluminada era lembrado). Em modo
`live` não há memória — a parede some ao sair da visão.

O servidor **não** envia dado de memória ao cliente. O cliente desenha todas as paredes
que recebeu e usa o polígono de visibilidade para decidir o brilho de cada pixel: nítido
dentro da linha de visão, esmaecido fora dela.
