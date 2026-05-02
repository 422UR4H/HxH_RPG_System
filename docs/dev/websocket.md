# Servidor WebSocket (Game Server)

## Arquitetura

Três camadas concorrentes: **Hub → Room → Client**.

```
Hub (1)
 └─ Room (por partida)
     ├─ Client (jogador 1)
     ├─ Client (jogador 2)
     └─ ...
```

- **Hub** — gerencia salas em `map[uuid.UUID]*Room` protegido por `sync.RWMutex`. `GetOrCreateRoom` cria a sala (recebendo UCs de domínio) e inicia sua goroutine (`go room.Run()`). `Stop()` encerra todas as salas.
- **Room** — event loop baseado em `select` com canais: `broadcast`, `register`, `unregister`, `stop`. Cada sala roda em sua própria goroutine. Possui UCs injetados (`IStartMatch`, `IKickPlayer`) para operações de domínio com persistência.
- **Client** — encapsula `gorilla/websocket.Conn` com duas goroutines: `ReadPump` (lê mensagens do WS e delega a `room.handleClientMessage`) e `WritePump` (escreve do canal `send` e envia pings).

Constantes do Client:

| Constante        | Valor   |
|------------------|---------|
| `writeWait`      | 10s     |
| `pongWait`       | 60s     |
| `pingPeriod`     | 54s     |
| `maxMessageSize` | 4096 B  |

## Máquina de Estados

```
Lobby → Playing → Closed
```

- **Lobby** → **Playing**: Apenas o mestre pode iniciar (`StartMatch`). Valida `userUUID == masterUUID` e `state == RoomStateLobby`. Chama `StartMatchUC.Start` que persiste `game_start_at` no banco e rejeita enrollments pendentes.
- **Playing** → **Closed**: Sala fecha quando o último client desconecta ou via `Hub.Stop()`.

Estados definidos como `RoomState string`: `"lobby"`, `"playing"`, `"closed"`.

## Protocolo de Mensagens

Struct base:

```go
type Message struct {
    Type      MessageType
    Payload   json.RawMessage
    SenderID  uuid.UUID
    Timestamp time.Time
}
```

### Server → Client

| Tipo             | Payload                | Quando                                  |
|------------------|------------------------|-----------------------------------------|
| `room_state`     | `RoomStatePayload`     | Ao conectar (estado completo)           |
| `player_joined`  | `PlayerPayload`        | Novo jogador entra                      |
| `player_left`    | `PlayerPayload`        | Jogador sai                             |
| `player_kicked`  | `PlayerKickedPayload`  | Mestre expulsa jogador do lobby         |
| `match_started`  | —                      | Mestre inicia a partida                 |
| `chat_message`   | `ChatPayload`          | Mensagem de chat                        |
| `error`          | `ErrorPayload`         | Erro (code + message)                   |

### Client → Server

| Tipo           | Descrição                            |
|----------------|--------------------------------------|
| `start_match`  | Mestre solicita início               |
| `kick_player`  | Mestre expulsa jogador               |
| `chat`         | Mensagem de chat                     |

Payloads auxiliares:
- `ErrorPayload { Code, Message string }`
- `PlayerPayload { UUID, Nickname }`
- `RoomStatePayload { MatchUUID, State, Players []PlayerInfo }`
- `PlayerInfo { UUID, Nickname, IsMaster, IsOnline }`
- `ChatPayload { Message string }`
- `KickPlayerPayload { PlayerUUID uuid.UUID }` — enviado pelo mestre
- `PlayerKickedPayload { UUID, Nickname, Reason }` — broadcast para todos

## Ciclo de Vida da Conexão

`Handler.HandleWebSocket` executa:

1. **Autenticação** — token via query param (`?token=`) ou header (`Authorization: Bearer`).
2. **Parse** — extrai `match_uuid` do query param.
3. **Autorização** — verifica se é mestre (`GetMatchMaster`) ou jogador inscrito (`EnrollmentChecker`).
4. **Upgrade** — HTTP → WebSocket via gorilla upgrader.
5. **Client** — cria `Client` com `userUUID`, `nickname`, `conn`, canal `send`.
6. **Registro** — registra client na Room (canal `register`).
7. **Goroutines** — inicia `go ReadPump()` e `go WritePump()`.

```go
// TODO: IN PRODUCTION, IMPLEMENT ORIGIN CHECKING (in upgrader.CheckOrigin)
```

Dependências do handler:
- `MatchRepository` — busca dados da partida e mestre.
- `EnrollmentChecker` — verifica inscrição do jogador.
- `IStartMatch` — UC de domínio para iniciar partida (persiste no banco).
- `IKickPlayer` — UC de domínio para expulsar jogador (rejeita enrollment no banco).

Servidor (`Server`) usa chi router com duas rotas:
- `GET /ws` — HandleWebSocket
- `GET /health` — health check

Timeouts: Read=15s, Write=15s, Idle=60s.

## Integração com Domínio

A Room delega operações de negócio a use cases de domínio via interfaces locais:

```go
type IStartMatch interface {
    Start(ctx context.Context, matchUUID uuid.UUID, masterUUID uuid.UUID) error
}
type IKickPlayer interface {
    Kick(ctx context.Context, matchUUID uuid.UUID, playerUUID uuid.UUID, masterUUID uuid.UUID) error
}
```

- **StartMatch**: Valida permissões localmente (mestre + estado lobby), depois chama `StartMatchUC.Start` que persiste `game_start_at` e rejeita todos os enrollments pendentes. Apenas após sucesso no banco a sala transiciona para `Playing`.
- **KickPlayer**: Valida permissão de mestre, chama `KickPlayerUC.Kick` que rejeita o enrollment no banco, depois desconecta o client e envia `player_kicked` para todos.

Chamadas ao banco usam `context.Background()` dentro das goroutines da Room (o `r.Context()` do HTTP já foi encerrado no momento da chamada).

## Referências de Código

| Arquivo           | Responsabilidade                                |
|-------------------|-------------------------------------------------|
| `game/hub.go`     | `Hub` (gerencia salas, mutex, stop)             |
| `game/room.go`    | `Room` (event loop, estados, StartMatch)        |
| `game/client.go`  | `Client` (ReadPump, WritePump, ping/pong)       |
| `game/message.go` | `Message`, tipos de payload, MessageType        |
| `game/handler.go` | `HandleWebSocket` (auth, upgrade, registro)     |
| `game/server.go`  | `Server` (chi router, timeouts, health check)   |
