# Design da Arquitetura do Domínio Match

**Data:** 2026-05-11
**Status:** Aprovado
**Escopo:** `internal/domain/match/`, `internal/application/`, `internal/domain/service/`, `internal/app/game/`

---

## Contexto

O sistema de partidas (a execução in-game de uma sessão: cenas, rodadas, turnos, ações e reações)
é o domínio mais complexo deste projeto. Antes de implementá-lo, precisávamos estabelecer limites
arquiteturais claros para que o sistema permaneça manutenível, testável e reutilizável para outros
sistemas de RPG que possam ser construídos sobre a mesma base de código.

Este documento registra as decisões tomadas, explica o raciocínio por trás de cada uma e serve
como material de aprendizado para desenvolvedores que não estão familiarizados com esses padrões.

---

## Terminologia (Glossário Canônico)

Estes nomes são a fonte de verdade no código, na documentação e nas regras do jogo:

| Nome no Código | PT-BR | O que significa |
|----------------|-------|-----------------|
| `Match` | Partida | A sessão de jogo completa (raiz de tudo) |
| `Scene` | Cena | Um segmento narrativo contínuo: Roleplay ou Battle |
| `Round` | Rodada | Um ciclo de ações dentro de uma cena. Tem um modo (Free ou Race). |
| `Turn` | Turno | A ação de um personagem + todas as reações a ela. A unidade atômica do combate. |
| `Action` | Ação | O que um personagem faz no seu Turno |
| `Reaction` | Reação | A resposta de outro personagem a uma Ação, dentro do mesmo Turno |

**Hierarquia:**
```
Match → Scene → Round → Turn → (Action + Reactions)
```

**Modos de Round:**
- `Free` (Rodada Livre) — sem pressão de tempo; ordem narrativa; sem fila de prioridade
- `Race` (Rodada Disputada) — baseado em tempo; ações ordenadas por Velocidade via fila de prioridade

Ambas as categorias de Cena (Roleplay e Battle) usam a mesma estrutura de Round/Turn.
O modo é definido por Round, não derivado da categoria da Cena.

---

## Por que Esta Arquitetura?

### O problema central

O domínio match tem dois tipos diferentes de complexidade acontecendo simultaneamente:

1. **Estado rico em memória** — enquanto uma sessão está em execução, o Round atual, o cache de
   fichas de personagens e a fila de ações precisam viver em RAM. Buscar do banco de dados a cada
   ação seria lento e desnecessário.

2. **Regras puras e complexas** — calcular combate (dados + atributos + arma + reações) envolve
   múltiplas entidades, mas não pertence a nenhuma delas individualmente.

Uma abordagem ingênua misturaria ambas as preocupações em um único objeto "engine". Foi exatamente
o que os antigos `turn/engine.go` e `round/engine.go` fizeram — gerenciavam estado E aplicavam
regras, o que os tornava difíceis de testar e impossíveis de reutilizar.

### A solução: quatro responsabilidades separadas

Cada camada tem exatamente um trabalho. Isso não é uma ideia nova — segue o DDD Lite aplicado a
um servidor de jogo em tempo real.

---

## Camadas da Arquitetura

```
┌──────────────────────────────────────────────────────────┐
│  app/  (Camada de Entrega)                               │
│  Traduz mensagens HTTP/WebSocket em chamadas de          │
│  use cases. Conhece gorilla/websocket, huma, JSON.       │
│  NÃO contém regras de jogo ou lógica de negócio.         │
├──────────────────────────────────────────────────────────┤
│  application/  (Camada de Use Cases)                     │
│  Orquestra: busca no banco → chama domínio → persiste.   │
│  Conhece repositórios e MatchSession.                    │
│  NÃO conhece HTTP, WebSocket ou fórmulas de RPG.         │
├──────────────────────────────────────────────────────────┤
│  domain/match/matchsession/  (Estado In-Memory da Sessão)│
│  Guarda o estado de runtime ativo de uma partida:        │
│  Round atual, cache de fichas, fila de ações.            │
│  Vive dentro da Room durante a sessão WebSocket.         │
│  Usa domain services para calcular resultados.           │
├──────────────────────────────────────────────────────────┤
│  domain/match/service/  (Domain Services)                │
│  Structs stateless que aplicam regras puras de RPG.      │
│  Recebem entidades como parâmetros, retornam resultados. │
│  Não conhecem banco, HTTP, WS ou gerenciamento de RAM.   │
├──────────────────────────────────────────────────────────┤
│  domain/match/entity/  (Entidades)                       │
│  Structs de estado puro. Só métodos sobre o próprio      │
│  estado. Nunca importam nenhuma camada acima.            │
└──────────────────────────────────────────────────────────┘
       gateway/ implementa interfaces de repositório do domain/
```

**A regra de ouro:** dependências só fluem para baixo. Uma entidade nunca importa um service.
Um service nunca importa um use case. Um use case nunca importa um handler.

> **Para desenvolvedores júnior:** pense como uma cozinha. A entidade é o ingrediente bruto (só
> existe). O domain service é a receita (instruções puras). O use case é o chef (segue a receita
> e gerencia o que entra e sai da geladeira). O handler é o garçom (anota o pedido e traz o
> prato, não sabe nada de culinária).

---

## Por que `application/` NÃO é igual a `app/`

Um ponto comum de confusão:

| | `app/` | `application/` |
|---|---|---|
| **Papel** | Camada de entrega | Camada de use cases |
| **Conhece** | HTTP, WebSocket, serialização JSON | Entidades de domínio, repositórios, MatchSession |
| **Imports de framework** | `gorilla/websocket`, `huma` | Nenhum |
| **Testável sem servidor?** | Não | **Sim** |
| **Exemplos** | `room.go`, `handler.go` | `open_next_action.go`, `close_turn.go` |

Use cases devem ser testáveis em isolamento com apenas repositórios mock e uma MatchSession —
sem servidor HTTP, sem WebSocket, sem JSON. É por isso que ficam separados de `app/`.

Esse padrão é padrão em projetos Go com DDD Lite:
- Alguns projetos chamam de `internal/usecase/` (mais comum em Go open-source)
- Alguns chamam de `internal/application/` (termo DDD correto)
- Ambos são válidos; usamos `application/` aqui

---

## Camada de Entidades — O que Muda

### O que fica (estado puro, sem coordenação)

`Round` e `Turn` se tornam structs de estado puro. Eles registram o que aconteceu — não decidem
quando as coisas acontecem.

```go
// Round — registra o estado de um ciclo de ações
type Round struct {
    mode       enum.RoundMode
    turns      []*Turn       // log append-only dos Turns neste Round
    events     []GameEvent
    finishedAt *time.Time
}

// Métodos mínimos — apenas sobre o próprio estado:
func (r *Round) GetMode() enum.RoundMode
func (r *Round) AppendTurn(t *Turn)
func (r *Round) CurrentTurn() *Turn   // último turn da lista
func (r *Round) HasOpenTurn() bool    // CurrentTurn().FinishedAt == nil
func (r *Round) Close(at time.Time)

// Turn — registra a ação de um personagem + todas as reações
type Turn struct {
    action     action.Action
    reactions  []action.Action
    openedAt   time.Time
    finishedAt *time.Time
}
```

### O que é removido

| Arquivo | Ação | Motivo |
|---------|------|--------|
| `entity/match/turn/engine.go` | **Deletado** | Semântica antiga (Turn como ciclo), substituído pelo pacote `round/` |
| `entity/match/round/engine.go` | **Dissolvido** | Lógica vai para `RoundOrchestrator` (stateless) + `MatchSession` (stateful) |
| `entity/match/engine.go` | **Dissolvido** | Orquestração vai para `MatchSession` |

### Localização da ActionPriorityQueue

A `ActionPriorityQueue` **não** fica no `Round`. Motivos:
- Ações podem ser declaradas para um Round futuro antes do atual terminar
- Ações podem atravessar Rounds dentro da mesma Cena
- A fila é estado operacional (vivo durante a sessão), não um registro histórico

**Decisão:** a fila de prioridade vive na `MatchSession` como `activeQueue`. O Round registra
apenas os Turns que resultam do processamento da fila — ele é o log, não a fila.

---

## Camada de Domain Services

Localizada em `internal/domain/match/service/`. Três structs stateless.

> **Para desenvolvedores júnior:** "stateless" significa que o struct NÃO TEM campos — não guarda
> nenhum dado. Você pode criar uma instância e reutilizá-la para sempre. É essencialmente uma
> coleção de funções puras agrupadas sob um nome. Você testa chamando seus métodos com dados de
> teste — sem banco, sem rede, sem precisar mockar nada.

### `RoundOrchestrator`

Sabe tudo sobre o ciclo de vida de um Round/Turn: quando criar Turns, quando fechá-los, como
usar a fila de prioridade, como anexar reações.

```go
type RoundOrchestrator struct{} // stateless — sem campos

// Extrai a Action de maior velocidade da fila e cria um novo Turn no Round.
// Usado no modo Race (ou Free quando o mestre escolhe a próxima ação).
func (ro RoundOrchestrator) NextAction(r *round.Round, queue *action.PriorityQueue) (*turn.Turn, error)

// Extrai uma Action específica por UUID — quando o mestre escolhe qual resolver.
func (ro RoundOrchestrator) PullAction(r *round.Round, queue *action.PriorityQueue, id uuid.UUID) (*turn.Turn, error)

// Define finishedAt no Turn atual. Chamado antes de abrir a próxima ação.
func (ro RoundOrchestrator) CloseTurn(r *round.Round, at time.Time) *turn.Turn

// Define finishedAt no Round. Chamado quando o mestre encerra um Round.
func (ro RoundOrchestrator) CloseRound(r *round.Round, at time.Time) *round.Round

// Valida que a reaction aponta para a Action do Turn atual, depois a adiciona.
func (ro RoundOrchestrator) AttachReaction(r *round.Round, reaction *action.Action) error

// Alterna o modo do Round entre Free e Race.
func (ro RoundOrchestrator) ChangeMode(r *round.Round, initiative *action.Initiative)
```

### `CombatResolver`

Sabe como calcular o resultado de um Turn (ação + todas as reações até o momento).
Chamado sempre que o estado do Turn muda: quando ele é aberto E a cada reação anexada.
O mestre vê um snapshot de resolução atualizado após cada mudança.

```go
type CombatResolver struct{} // stateless

// Resolve o estado atual do Turn (ação + quantas reações existirem).
// Chamado na abertura do Turn (sem reações ainda) e re-chamado após cada AttachReaction.
// Retorna um snapshot que o use case transmite a todos os participantes.
func (cr CombatResolver) Resolve(
    t *turn.Turn,
    sheets map[uuid.UUID]*sheet.Sheet,
) *TurnResolution

type TurnResolution struct {
    ActionResult    RollResult
    ReactionResults []ReactionResult
    Blows           []*battle.Blow
    IsSettled       bool // false enquanto reações ainda podem chegar
}
```

### `RollCalculator`

Sabe como calcular o resultado final de uma rolagem: valores dos dados + perícia do personagem
+ modificadores.

```go
type RollCalculator struct{} // stateless

func (rc RollCalculator) Calculate(
    check action.RollCheck,
    sheet *sheet.Sheet,
) int
```

**Reutilização para outros sistemas de RPG:** para construir um RPG diferente (ex: sistema
Cyberpunk), você importa as mesmas entidades (`Action`, `Turn`, `Round`) e escreve novas
implementações de `CombatResolver` e `RollCalculator` específicas para as regras daquele sistema.
As entidades estruturais são genéricas; as regras são plugáveis.

---

## MatchSession

Localizada em `internal/domain/match/matchsession/match_session.go`.

> **Para desenvolvedores júnior:** `MatchSession` é a "memória viva" de uma partida enquanto
> ela está acontecendo. Pense como um quadro branco que o mestre e os jogadores escrevem durante
> a sessão. Quando a sessão termina, as partes importantes do quadro são salvas no banco e o
> resto é apagado. A `Room` (servidor WebSocket) guarda esse quadro branco.

### O que ela guarda

```go
type MatchSession struct {
    matchUUID   uuid.UUID
    activeScene *scene.Scene
    activeRound *round.Round

    // Fila de ações — pertence à Cena ativa, não ao Round.
    // Sobrevive a mudanças de Round dentro da mesma Cena.
    activeQueue action.PriorityQueue

    // Cache de fichas de personagens. Carregado uma vez ao iniciar a sessão,
    // somente leitura durante o combate. Evita hits no banco a cada resolução de Turn.
    charSheets   map[uuid.UUID]*sheet.Sheet
    participants map[uuid.UUID]*match.Participant

    // Domain services injetados
    roundOrch service.RoundOrchestrator
    combatRes service.CombatResolver
}
```

### O que ela faz

```go
// Jogadores enfileiram a ação do próprio personagem.
func (s *MatchSession) EnqueueAction(playerUUID uuid.UUID, a *action.Action) error

// Mestre enfileira uma ação de NPC.
func (s *MatchSession) EnqueueMasterAction(npcUUID uuid.UUID, a action.MasterAction) error

// Mestre abre a ação de maior prioridade da fila.
// Se um Turn estiver aberto, ele é fechado primeiro.
// Retorna: o Turn fechado (se houver) e o novo Turn.
func (s *MatchSession) OpenNextAction() (closed *turn.Turn, opened *turn.Turn, err error)

// Mestre abre uma ação específica por UUID.
func (s *MatchSession) PullAction(id uuid.UUID) (closed *turn.Turn, opened *turn.Turn, err error)

// Jogador ou mestre anexa uma reação ao Turn atual.
// Também chama CombatResolver.Resolve para atualizar o snapshot de resolução.
func (s *MatchSession) AttachReaction(r *action.Action) (*TurnResolution, error)

// Fecha explicitamente o Turn atual (decisão do mestre).
func (s *MatchSession) CloseTurn() (*turn.Turn, error)

// Fecha o Round atual e prepara para o próximo.
func (s *MatchSession) CloseRound() (*round.Round, error)

// Acesso de leitura
func (s *MatchSession) GetActiveRound() *round.Round
func (s *MatchSession) GetCurrentTurn() *turn.Turn
func (s *MatchSession) GetCharSheet(playerUUID uuid.UUID) (*sheet.Sheet, error)
```

### Como a dependência cíclica é resolvida

As antigas engines compartilhavam um flag `closeRoundTriggered *bool` para comunicar entre si.
`MatchSession` torna o fluxo explícito:

```go
func (s *MatchSession) OpenNextAction() (closed *turn.Turn, opened *turn.Turn, err error) {
    // Fecha o Turn anterior antes de abrir o próximo — sem flag, sem estado compartilhado
    if s.activeRound != nil && s.activeRound.HasOpenTurn() {
        closed = s.roundOrch.CloseTurn(s.activeRound, time.Now())
    }
    opened, err = s.roundOrch.NextAction(s.activeRound, &s.activeQueue)
    return
}
```

O use case recebe tanto o Turn `closed` (para persistir) quanto o Turn `opened` (para transmitir)
de uma única chamada.

### Integração com a Room

`MatchSession` é criada quando a Room transita para `RoomStatePlaying` e é mantida durante toda
a sessão:

```go
type Room struct {
    // campos existentes...
    session *matchsession.MatchSession // nil até a partida começar

    // novos use cases injetados na construção
    openNextActionUC IOpenNextAction
    pullActionUC     IPullAction
    attachReactionUC IAttachReaction
    closeTurnUC      ICloseTurn
    closeRoundUC     ICloseRound
}
```

---

## Fluxo Completo de Dados — Exemplos

**Cenário: Mestre abre a próxima ação (modo Race)**

```
1. Mensagem WebSocket chega:
   { "type": "open_next_action" }

2. Room.handleClientMessage() despacha para:
   r.openNextActionUC.Execute(ctx, r.session, masterUUID)

3. Use case (OpenNextActionUC.Execute):
   a. Valida que o chamador é o mestre
   b. closed, opened, err := session.OpenNextAction()
      → MatchSession: fecha o Turn anterior (se houver) via RoundOrchestrator
      → MatchSession: extrai a próxima Action da activeQueue via RoundOrchestrator
      → MatchSession: adiciona novo Turn ao activeRound
   c. if closed != nil:
      → matchRepo.SaveTurn(ctx, closed)     ← um INSERT, registro imutável
   d. resolution := session.combatRes.Resolve(opened, session.charSheets)
   e. retorna opened, resolution

4. Room faz broadcast para todos os participantes:
   { "type": "turn_opened", "turn": opened, "resolution": resolution }
```

**Cenário: Jogador anexa uma reação**

```
1. { "type": "attach_reaction", "payload": { "react_to_id": "...", ... } }

2. Room → attachReactionUC.Execute(ctx, session, playerUUID, reactionData)

3. Use case:
   a. Valida que o jogador possui o personagem
   b. Cria action.Action com ReactToID definido
   c. resolution, err := session.AttachReaction(reaction)
      → MatchSession: RoundOrchestrator.AttachReaction valida ReactToID
      → MatchSession: CombatResolver.Resolve recalcula o Turn com a nova reação
   d. retorna resolution

4. Room envia o resolution atualizado APENAS AO MESTRE — não faz broadcast geral
   { "type": "resolution_updated", "resolution": resolution }
   Os jogadores só ficam sabendo da reação quando o mestre a revela (evento separado,
   fluxo a ser definido). Esta é uma regra de visibilidade central do jogo.
```

> **Regra de visibilidade (fluxo completo a definir):** reações são submetidas de forma privada
> — apenas o mestre as recebe no `attach_reaction`. Todos os jogadores recebem as informações
> da reação somente quando o mestre a revela explicitamente. O fluxo completo de "revelar
> reação" e seu impacto no estado do `Turn` (reações submetidas vs reveladas) será detalhado
> em um spec de acompanhamento.

---

## Modelo de Persistência

**Turns são persistidos apenas uma vez: quando são encerrados.** `openedAt` vive na entidade em
memória até então. Sem `UPDATE` — cada escrita é um `INSERT`. Tabelas são imutáveis após a
escrita inicial. Sem colunas `updated_at`.

```
Hierarquia no banco (cada nível é um log append-only):

matches
└── scenes        (fk: match_uuid)
     └── rounds   (fk: scene_uuid, col: mode)
          └── turns (fk: round_uuid, cols: action_data, reactions_data, opened_at, finished_at)
```

O histórico completo de uma partida é reconstruído lendo todos os Turns em ordem — esse é o log
de eventos natural que o modelo de dados fornece, sem precisar de infraestrutura de Event Sourcing.

> **Sobre Event Sourcing:** Event Sourcing completo (event store, projeções, replay) seria
> over-engineering para este projeto. A tabela de Turns append-only já fornece a trilha de
> auditoria e o registro histórico. Domain Events (pub/sub simples para broadcasting) podem ser
> adicionados depois sem alterar este modelo.

---

## Estrutura de Pastas

```
internal/
├── app/
│   ├── api/                        ← Handlers REST HTTP (sem mudança)
│   └── game/                       ← WebSocket: Hub, Room, Client
│       └── room.go                 ← adiciona: session *matchsession.MatchSession
│
├── application/                    ← Use Cases (movidos de domain/)
│   ├── match/
│   │   ├── create_match.go
│   │   ├── start_match.go
│   │   ├── list_matches.go
│   │   ├── get_match.go
│   │   ├── open_next_action.go     ← NOVO
│   │   ├── pull_action.go          ← NOVO
│   │   ├── enqueue_action.go       ← NOVO
│   │   ├── attach_reaction.go      ← NOVO
│   │   ├── close_turn.go           ← NOVO
│   │   ├── close_round.go          ← NOVO
│   │   └── i_repository.go
│   ├── auth/
│   ├── campaign/
│   ├── character_sheet/
│   ├── enrollment/
│   ├── submission/
│   ├── session/
│   └── scenario/
│
├── domain/
│   ├── match/                      ← Bounded context de Partida (NOVA estrutura)
│   │   ├── entity/
│   │   │   ├── action/             ← Action, Attack, Defense, Dodge, etc. (sem mudança)
│   │   │   ├── round/              ← Round (entity pura — engine.go REMOVIDO)
│   │   │   ├── turn/               ← Turn (entity pura — engine.go DELETADO)
│   │   │   ├── scene/              ← Scene (sem mudança)
│   │   │   ├── battle/             ← Blow (sem mudança)
│   │   │   ├── match.go
│   │   │   ├── participant.go
│   │   │   ├── character_status.go
│   │   │   ├── game_event.go
│   │   │   └── summary.go
│   │   ├── service/
│   │   │   ├── round_orchestrator.go
│   │   │   ├── combat_resolver.go
│   │   │   └── roll_calculator.go
│   │   └── matchsession/
│   │       └── match_session.go
│   │
│   └── entity/                     ← Localização legada (estável — migrar separadamente)
│       ├── character_sheet/        ← Estável, totalmente testado — NÃO refatorar agora
│       ├── character_class/
│       ├── campaign/
│       ├── scenario/
│       ├── user/
│       ├── enrollment/
│       ├── item/
│       ├── enum/                   ← Enums compartilhados (usados por todos os contextos)
│       └── die/                    ← Dados compartilhados (usados por todo o sistema)
│
└── gateway/                        ← Repositórios PostgreSQL (estrutura sem mudança)
```

### Por que bounded context para match mas não para character_sheet?

`character_sheet/` está estável e totalmente testado. Migrá-lo para `domain/character_sheet/entity/`
exigiria atualizar todos os seus imports sem ganho funcional imediato. Está diferido para um PR
de refatoração dedicado. O novo código de `match/` começa com a estrutura correta desde o início.

---

## Log de Decisões

| Decisão | Justificativa |
|---------|---------------|
| `MatchSession` em `domain/match/matchsession/` | Específico ao contexto de partida; claramente separado da `session/` de auth |
| `ActionPriorityQueue` na `MatchSession`, não no `Round` | A fila é estado operacional que sobrevive a mudanças de Round dentro de uma Cena |
| Domain services são structs stateless | Permite testes isolados com dados Go puros; permite substituição de regras para futuros sistemas de RPG |
| `RoundOrchestrator` (não Coordinator) | Mais expressivo para o papel de dirigir o fluxo de execução de um Round |
| Método `Resolve` único no `CombatResolver` | Chamado na abertura do Turn e após cada reação; mais simples que `Resolve`+`Recalculate` com comportamento equivalente |
| Turn persistido apenas no fechamento (um INSERT) | Log append-only — sem UPDATE, sem updated_at; consistente com o modelo de log de eventos |
| Use cases movidos para `application/` | Separação clara da entrega (`app/`) e do domínio puro (`domain/`); testável sem nenhum framework |
| Estrutura de bounded context para `match/` | Agrupa tudo sobre o domínio de partida; permite extração futura como módulo reutilizável |
| `character_sheet/` permanece em `entity/` por ora | Estável, testado, sem disrupção justificada; refatoração diferida |
| Sem Event Sourcing | A tabela de Turns append-only JÁ É o log de eventos; ES completo seria over-engineering para o MVP |
| Ambas as categorias de Cena usam Round/Turn | Roleplay = Round no modo Free, Battle = Round no modo Race; modelo unificado, modo definido por Round |

---

## Trabalho Diferido

Estes são adiamentos intencionais — não itens esquecidos:

1. **Migração do bounded context `character_sheet/`** — mover `domain/entity/character_sheet/`
   para `domain/character_sheet/entity/` em um PR dedicado após esta arquitetura estar estável.

2. **Shared kernel para `enum/` e `die/`** — mover para `domain/shared/` quando múltiplos
   bounded contexts estiverem estabelecidos.

3. **Domain Events (pub/sub)** — emitir eventos no fechamento de Turn para futuros sistemas de
   analytics ou notificação. Não necessário para o MVP.

4. **Otimização de memória do cache `charSheets`** — para o MVP, todas as fichas são carregadas
   no início da sessão. Futuro: carregamento lazy ou limite de tamanho para partidas grandes.

5. **`Initiative` em `RoundOrchestrator.ChangeMode`** — marcado como TODO no código existente;
   completar quando as regras de Iniciativa forem definidas.
