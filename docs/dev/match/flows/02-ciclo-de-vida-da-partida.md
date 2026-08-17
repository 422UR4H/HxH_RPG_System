# 02 — Ciclo de vida da partida

## Estados

```mermaid
stateDiagram-v2
    [*] --> Criada: POST /matches (CreateMatchUC)
    Criada --> Lobby: mestre abre o lobby (WS connect)
    Lobby --> EmAndamento: MsgTypeStartMatch → StartMatchUC + InitMatchSessionUC
    Lobby --> Cancelada: MsgTypeCancelLobby
    EmAndamento --> EmAndamento: cenas / rounds / turnos
    EmAndamento --> [*]: (fim de partida — ainda não implementado)
```

Antes de `StartMatch`, `Room.session == nil`. Toda mensagem de jogo responde
`match_not_started`. O lobby já mexe em peças e paredes (`piece_moved`, `map_state_sync`),
mas isso vive em `Room`, não na sessão.

## `start_match`: a fábrica da sessão

```mermaid
sequenceDiagram
    autonumber
    participant M as Cliente (mestre)
    participant R as Room
    participant SUC as StartMatchUC
    participant IUC as InitMatchSessionUC
    participant Repo as IRepository / IRoundRepository / ICharSheetLoader
    participant DB as Postgres
    participant S as MatchSession

    M->>R: start_match
    R->>SUC: Start(ctx, matchUUID, userUUID)
    SUC->>Repo: StartMatch (grava gameStartAt)
    Repo->>DB: UPDATE matches
    R->>IUC: Init(ctx, matchUUID)
    IUC->>Repo: ListParticipantsByMatchUUID
    loop cada participante com PlayerUUID
        IUC->>Repo: GetCharacterSheetByUUID(sheet.UUID)
    end
    IUC->>Repo: FindActiveSession(matchUUID)
    alt existe cena/round aberto no DB (retomada)
        Repo-->>IUC: *ActiveSessionData
        IUC->>S: NewMatchSessionWithState(...ReconstructScene, ReconstructRound)
        Note over S: scenePersisted = roundPersisted = true
    else partida nova
        IUC->>S: NewMatchSession(...)
        Note over S: cena Roleplay vazia + round Free<br/>scenePersisted = roundPersisted = false
    end
    IUC-->>R: *MatchSession
    R->>S: SetPieceSource(r)
    R->>S: SyncMapState(walls, grid)
    R->>S: SyncPlayerMemories(nil, fog.FogModeExplored)
    R->>S: RecomputeAllVisibility()
    R-->>M: match_started + map_full_state por jogador
```

Dois detalhes que importam:

- **`charSheets` (e `statuses`) é chaveado por `sheetUUID`, não por `playerUUID`.** As
  **peças do tabuleiro carregam o `sheetUUID`** como `CharacterID`, e é por esse eixo que a
  ficha e o estado de combate vivo são buscados. `participants` continua chaveado por
  `playerUUID` (autorização é por jogador); o mapa auxiliar `charToPlayer[sheetUUID] =
  playerUUID` é a ponte entre os dois eixos toda vez que o fluxo cruza tabuleiro ↔ jogador.
- **A sessão nasce com uma cena e um round já abertos.** Não existe estado "sem cena".
  Cena inicial: `enum.Roleplay` com descrição vazia; round inicial: `enum.Free`.

## Cena → Round → Turno: a hierarquia viva

```mermaid
flowchart TB
    S["Scene<br/>category · briefInitialDescription<br/>turns[] · createdAt · finishedAt"]
    R["Round<br/>mode (Free|Race)<br/>turns[] · events[] · createdAt · finishedAt"]
    T["Turn<br/>action (1, imutável)<br/>reactions[] · masterActions[]<br/>finishedAt"]
    A["Action (a do ator)"]
    RE["Action (reações)"]
    MA["MasterAction (intervenções do mestre)"]

    S --> R
    R --> T
    T --> A
    T --> RE
    T --> MA
```

A sessão mantém **exatamente um** `activeScene` e **exatamente um** `activeRound`.
O "turno atual" é sempre `activeRound.CurrentTurn()` — o último da fatia. Não há índice
nem ponteiro separado: `HasOpenTurn()` é `CurrentTurn() != nil && finishedAt == nil`.

## `change_scene`

```mermaid
sequenceDiagram
    participant M as Mestre
    participant R as Room
    participant UC as ChangeSceneUC
    participant S as MatchSession
    participant DB as Postgres

    M->>R: change_scene {category, briefInitialDescription}
    R->>UC: Execute(ctx, session, masterUUID, callerUUID, cat, desc)
    UC->>UC: callerUUID == masterUUID? senão ErrNotMatchMaster
    UC->>S: ChangeScene(cat, desc)
    alt activeRound.HasOpenTurn()
        S-->>UC: ErrRoundHasOpenTurn
    else
        S->>S: fecha activeRound e activeScene (now)
        S->>S: activeScene = NewScene(cat, desc)<br/>activeRound = NewRound(enum.Free)
        S->>S: scenePersisted = roundPersisted = false
        S-->>UC: (oldScene, oldRound)
    end
    R->>DB: CloseSceneAndRound(oldScene.ID, oldRound.ID, now) — só se já estavam persistidos
    R-->>M: scene_changed (broadcast)
```

Note que a cena nova **sempre volta para `Free`**, mesmo que a anterior estivesse em `Race`.

## `close_round`

`CloseRoundUC` fecha o round atual e **abre um novo já preservando o `mode`**
(`round.NewRound(mode)`), marcando `roundPersisted = false`. Persiste via
`IRoundRepository.CloseRound` apenas se o round fechado já existia no banco.
Erro de persistência é engolido (`_ = dbErr // log in production`) — a partida em memória
segue em frente.
