# WebSocket Game Server — Spec de Design

**Data:** 29/04/2026
**Status:** Aprovado
**Escopo:** Infraestrutura MVP de comunicação em tempo real para execução de partidas

## Problema

A plataforma HxH RPG precisa de comunicação bidirecional em tempo real entre
o mestre e os jogadores durante uma partida. A API REST cuida das operações
CRUD (criar partidas, alistar personagens, etc.), mas quando uma partida está
em execução, todos os participantes precisam de entrega instantânea de
eventos: ações, reações, mudanças de turno, mensagens de chat e atualizações
de estado do jogo.

## Contexto: Arquitetura de Execução de uma Partida

Uma Partida se subdivide em **Cenas (Scenes)**, que contêm **Turnos (Turns)**,
que contêm **Rounds**. Essa hierarquia dirige todo o fluxo do jogo:

```
Partida (Match em execução)
├── Cena: Roleplay (interação/investigação)
│   └── Turnos (modo free — sem disputa de tempo)
│       └── Rounds (ação de cada personagem)
└── Cena: Battle (combate)
    └── Turnos (modo race — ordem por velocidade)
        └── Rounds (ação + reações, fila de prioridade)
```

**Categorias de cena** (roleplay vs battle) existem para classificação,
ordenação histórica e legibilidade narrativa. Elas NÃO determinam o modo do
turno — uma cena roleplay poderia teoricamente usar modo race e vice-versa.

**Modos de turno:**
- **Free** — sem pressão de tempo; jogadores agem em ordem natural
- **Race** — milissegundos importam; ações resolvidas por fila de prioridade de velocidade

A Turn Engine gerencia a execução dos turnos sem saber a categoria da cena.
Essa separação é intencional e deve ser preservada.

## Abordagem: gorilla/websocket + Hub/Room/Client

Usa a dependência existente `gorilla/websocket` (v1.5.3, já no go.mod) com
uma arquitetura Hub/Room/Client bem estruturada, inspirada no exemplo oficial
de chat da gorilla.

### Por que esta abordagem

- gorilla/websocket já é dependência do projeto
- O padrão Hub/Room/Client é o padrão da indústria para este caso de uso
- Documentação extensa e exemplos da comunidade
- A abstração Room pode ser extraída para processos separados futuramente

### Alternativa considerada

**nhooyr.io/websocket** — API Go mais idiomática (context.Context nativo,
escritas concurrent-safe). Rejeitada porque: o padrão Hub/Room já resolve
escritas concorrentes via write pumps por client, e gorilla tem vastamente
mais implementações de referência para este padrão exato.

## Arquitetura

### Visão Geral dos Componentes

```
┌─────────────────────────────────────────────────┐
│            API Server (cmd/api)                  │
│          REST — porta 5000                       │
│  POST /matches, GET /campaigns, etc.            │
└──────────────────────┬──────────────────────────┘
                       │ PostgreSQL compartilhado
┌──────────────────────▼──────────────────────────┐
│           Game Server (cmd/game)                 │
│         WebSocket — porta 8080                   │
│                                                  │
│  ┌────────────────────────────────────────────┐  │
│  │                   Hub                      │  │
│  │  rooms map[matchUUID]*Room                 │  │
│  │                                            │  │
│  │  ┌──────────────┐  ┌──────────────┐       │  │
│  │  │  Room #42    │  │  Room #87    │  ...  │  │
│  │  │  (lobby)     │  │  (playing)   │       │  │
│  │  │  👑 Mestre   │  │  👑 Mestre   │       │  │
│  │  │  🎮 Player1  │  │  🎮 Player1  │       │  │
│  │  │  🎮 Player2  │  │  🎮 Player2  │       │  │
│  │  └──────────────┘  └──────────────┘       │  │
│  └────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────┘
```

### Estrutura de Arquivos

```
internal/app/game/
├── hub.go          — Hub: gerencia todas as Rooms ativas
├── room.go         — Room: uma sessão de partida (lobby → playing → closed)
├── client.go       — Client: uma conexão WebSocket (readPump/writePump)
├── message.go      — Tipos de mensagem e protocolo JSON
├── handler.go      — Handler de upgrade HTTP com autenticação JWT
└── server.go       — Setup do router chi, CORS, configuração do upgrader
```

### Hub

Gerencia todas as rooms ativas. Singleton por processo do game server.

```go
type Hub struct {
    rooms      map[uuid.UUID]*Room  // matchUUID → Room
    register   chan *Client
    unregister chan *Client
    mu         sync.RWMutex
}
```

### Room

Uma por partida. Gerencia clients conectados e estado da sala.

```go
type Room struct {
    matchUUID  uuid.UUID
    masterUUID uuid.UUID
    state      RoomState              // lobby | playing | closed
    clients    map[uuid.UUID]*Client  // userUUID → Client
    broadcast  chan []byte
    register   chan *Client
    unregister chan *Client
}
```

Estados: `lobby` → `playing` → `closed`

### Client

Um por conexão WebSocket. Duas goroutines por client.

```go
type Client struct {
    userUUID uuid.UUID
    conn     *websocket.Conn
    room     *Room
    send     chan []byte
}
```

- `ReadPump()` — lê mensagens do WS, roteia para a room
- `WritePump()` — escreve mensagens do channel send para o WS

### Protocolo de Mensagens

Todas as mensagens são JSON com envelope comum:

```json
{
  "type": "start_match",
  "payload": { ... },
  "sender_id": "user-uuid",
  "timestamp": "2026-04-29T01:00:00Z"
}
```

`sender_id` e `timestamp` são sempre preenchidos pelo servidor.

#### Mensagens Server → Client

| Tipo | Payload | Descrição |
|------|---------|-----------|
| `room_state` | `{match_uuid, state, players: [...]}` | Estado atual (enviado ao entrar) |
| `player_joined` | `{uuid, nickname}` | Jogador conectou |
| `player_left` | `{uuid, nickname}` | Jogador desconectou |
| `match_started` | `{}` | Partida iniciou |
| `chat_message` | `{message}` | Chat broadcast |
| `error` | `{code, message}` | Erro (unicast para quem enviou) |

#### Mensagens Client → Server

| Tipo | Payload | Papel Requerido | Descrição |
|------|---------|-----------------|-----------|
| `start_match` | `{}` | Apenas mestre | Transiciona room para playing |
| `chat` | `{message}` | Qualquer | Envia mensagem de chat |

### Fluxo de Conexão

1. Client conecta: `GET /ws?match_uuid=XXX` com `Authorization: Bearer <JWT>`
2. Handler valida JWT → extrai userUUID
3. Handler valida parâmetro match_uuid
4. Handler consulta DB: match existe? User é master ou está enrolled?
5. Falha na validação: retorna erro HTTP (antes do upgrade)
6. Sucesso: upgrade para WebSocket
7. Hub registra client na Room apropriada (cria se necessário)
8. Room envia `room_state` para o novo client
9. Room faz broadcast de `player_joined` para os demais

### Validação na Conexão (pré-upgrade)

| Checagem | Falha | HTTP Status |
|----------|-------|-------------|
| JWT válido? | Token inválido/expirado | 401 |
| match_uuid presente e UUID válido? | Ausente/malformado | 400 |
| Match existe no DB? | Não encontrado | 404 |
| User é master ou enrolled? | Não autorizado | 403 |

### Máquina de Estados da Room

```
            Mestre conecta
                 │
                 ▼
        ┌─────────────┐
        │    LOBBY     │ ◄── jogadores podem entrar/sair
        │              │     chat disponível
        └──────┬───────┘
               │ Mestre envia start_match
               ▼
        ┌─────────────┐
        │   PLAYING    │ ◄── partida em curso
        │              │     chat disponível
        └──────┬───────┘     (futuro: ações de jogo)
               │ Todos desconectam ou mestre encerra
               ▼
        ┌─────────────┐
        │   CLOSED     │ → Room removida do Hub
        └─────────────┘
```

### Autenticação

A conexão WebSocket reutiliza o mesmo JWT da API REST. O token é validado uma
vez no momento da conexão (HTTP upgrade). Uma vez estabelecido o WebSocket, a
conexão persiste independente da expiração do token.

**Melhoria futura:** Refresh de token via WebSocket — servidor envia novo JWT
antes do atual expirar.

### Reconexão

Abordagem MVP: se um client desconecta e reconecta, recebe um Client novo na
mesma Room. A Room envia `room_state` atual ao entrar, então o client fica
imediatamente atualizado. Sem replay de mensagens no MVP.

### Integração com Código Existente

**Pacotes reutilizados:**
- `pkg/auth` — `ValidateToken()` para autenticação JWT
- `pkg` (pgfs) — Pool de conexão PostgreSQL
- `internal/config` — `LoadCORS()` para configuração CORS
- `go-chi/chi` — Router HTTP (consistência com servidor API)
- `gorilla/websocket` — Implementação WebSocket (já no go.mod)
- `internal/gateway/pg/match` — Validar existência do match
- `internal/gateway/pg/enrollment` — Validar enrollment do jogador

**Reescrita do entry point:** `cmd/game/main.go` será reescrito do protótipo
atual para um servidor com injeção de dependências, seguindo o mesmo padrão
de `cmd/api/main.go`.

### Tamanho Estimado

| Arquivo | Linhas | Propósito |
|---------|--------|-----------|
| hub.go | ~80 | Gerenciamento de rooms |
| room.go | ~120 | Gerenciamento de clients, máquina de estados, broadcast |
| client.go | ~100 | Goroutines readPump/writePump |
| message.go | ~50 | Tipos de mensagem, marshal/unmarshal |
| handler.go | ~60 | Upgrade HTTP→WS, auth, validação |
| server.go | ~30 | Router chi, setup do servidor |
| **Total** | **~440** | |

## Estratégia de Testes

- **Testes unitários:** Hub, Room, Client com conexões mock
- **Testes de integração:** Ciclo de vida completo de conexão WebSocket
- **Helpers de teste gorilla/websocket** para simular conexões de clientes

## Melhorias Futuras (fora do escopo MVP)

- Refresh de token via WebSocket
- Adicionar/remover jogadores em runtime (controle do mestre)
- Escalar para processo separado por Room
- Ações de jogo (turnos, rounds, combate) sobre a mesma infra de mensagens
- Replay de mensagens na reconexão
- Modo espectador
