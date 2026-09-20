# 03 — Fluxo de ação

> Estado do código na Fase 5. Não é design aspiracional: cada seta abaixo existe.

O coração do produto: jogadores **declaram intenção em paralelo**, o mestre **abre** as ações
na ordem certa, o sistema **resolve** os números. Ninguém espera a vez para *pensar* — é isso
que mata a latência de mesa.

## Os verbos de hoje

| Mensagem | Quem pode | O que faz |
|---|---|---|
| `enqueue_action` | jogador (do próprio personagem) | declara; rola os dados na chegada; entra na fila |
| `attach_reaction` | jogador alvo | responde ao ataque aberto; cobra barra; recalcula |
| `open_next_action` | mestre | fecha o turno anterior e abre o próximo da ordem |
| `pull_action` | mestre | abre **esta** action, fora de ordem, sem porteiro |
| `open_reaction` | mestre | dá a palavra a uma reaction anexada — **a ordem muda o resultado** |
| `edit_action` | mestre | edita a action aberta; recalcula sem re-sortear |
| `close_turn` | mestre | encerra de propósito; recusa se há reaction não aberta |
| `change_round_mode` | mestre | `Free` ⇄ `Race` |

> `enqueue_action` com `reactToId` preenchido **é** o `attach_reaction`: `room.go` desvia para
> `handleReaction`. Os dois campos viajam juntos ou nenhum — `reactToId` diz *"isto é reação"*
> e `reactionKind` diz *o que custa*.

## Visão geral

```mermaid
flowchart TB
    subgraph par["em paralelo, sem esperar a vez"]
        D1["jogador A<br/>enqueue_action"]
        D2["jogador B<br/>enqueue_action"]
        D3["jogador C<br/>enqueue_action"]
    end
    par --> Q["activeQueue<br/><i>lista simples — sem prioridade guardada</i>"]
    Q --> SEL{{"RoundScheduler<br/>chave calculada na hora"}}
    SEL -->|"nenhuma passa<br/>no porteiro"| RC["round_closed<br/>saldos liquidados"]
    SEL -->|"escolheu"| OPEN["mestre: open_next_action"]
    OPEN --> CLOSE0["fecha o turno anterior<br/>resolve · aplica dano · avança ledgers"]
    CLOSE0 --> T["turn_opened<br/>mecânica pública"]
    T --> REACT["alvos: attach_reaction"]
    REACT --> FLOOR["mestre: open_reaction<br/><i>a ordem de abertura muda o desfecho</i>"]
    FLOOR --> EDIT["mestre: edit_action<br/><i>opcional</i>"]
    EDIT --> CT["mestre: close_turn"]
    CT --> PERSIST[("actions · turns.resolution<br/>overridden_action_values")]
    CT --> OPEN

    style RC fill:#fff3cd,stroke:#856404
    style PERSIST fill:#e8e8e8,stroke:#555
```

**Não existe verbo de confirmação.** O mestre edita, a resolução recalcula na hora, e **passar
o bastão é a confirmação** — abrir a próxima, abrir uma reaction, fechar o turno. Não dá para
seguir em frente e continuar deliberando ao mesmo tempo.

## Tempo 1 — declarar

```mermaid
sequenceDiagram
    autonumber
    participant P as Cliente (jogador)
    participant R as Room
    participant UC as EnqueueActionUC
    participant S as MatchSession
    participant M as Mestre

    P->>R: enqueue_action {actorId, move?, attack?, skills?, ...}
    R->>R: reactToId ⊕ reactionKind? → erro
    R->>R: actorId vazio? → erro
    R->>R: session nil? → match_not_started
    alt reactToId preenchido
        R->>R: desvia para handleReaction (Tempo 3)
    end
    R->>R: buildAction(payload.ActorID, payload)
    opt tem Move com From
        R->>R: IsPathBlocked → move_blocked
    end
    Note over R: r.mu.Lock() ATRAVESSA o Execute
    R->>UC: Execute(session, userUUID, a)
    UC->>S: EnqueueAction(playerUUID, a)
    S->>S: charToPlayer[actorID] == playerUUID? senão ErrActionActorMismatch
    S->>S: rollActionDice(a) — os DOIS conjuntos, uma vez só
    S->>S: deriveSpeeds(a, systemBias)
    S->>S: activeQueue.Insert(a)
    R-->>P: action_enqueued  (só para quem enviou)
    R-->>M: action_queued {actionId, actorId, bars}  (só o mestre)
    R-->>P: bars_updated  (mesa inteira)
```

**`actorID` é o `sheetUUID`**, não o do jogador. A autorização é `charToPlayer[actorID] ==
playerUUID` — é assim que um mestre pode agir por um NPC sem que um jogador possa agir pelo
personagem de outro.

**`action_queued` é master-only e existe por um motivo estrutural:** sem ele o mestre não tem
como aprender o ID de uma action pendente, e `pull_action` fica inalcançável de um cliente
real. *Um ID que o cliente não consegue aprender é uma operação que o cliente não consegue
invocar.*

**A fila não guarda prioridade.** `activeQueue` é lista simples. A chave de ordenação muda
quando o personagem manda outra action (a média se move), e um heap não suporta re-chavear um
item já inserido — quebraria em silêncio. A chave é calculada na hora da seleção.

## Tempo 2 — abrir

```mermaid
sequenceDiagram
    autonumber
    participant M as Mestre
    participant R as Room
    participant UC as OpenNextActionUC
    participant S as MatchSession
    participant SCH as RoundScheduler
    participant DB as Postgres

    M->>R: open_next_action
    R->>R: IsMaster? senão forbidden
    Note over R: r.mu.Lock() ATRAVESSA o Execute
    R->>UC: Execute(session, masterUUID, userUUID)
    UC->>S: OpenNextAction()
    S->>SCH: FreezePrices — o preço congela na 1ª seleção
    alt modo Free
        S->>S: closeOpenTurn() → NextAction(fila)
    else modo Race
        S->>SCH: SelectNext — porteiro + chave
        alt nenhuma passa
            S->>S: closeOpenTurn() · RoundExhausted = true
            UC->>UC: CloseRoundUC — liquida saldos
            R-->>M: round_closed (mesa inteira)
        else escolheu
            S->>S: closeOpenTurn()
            S->>S: PullAction(id escolhido) · recordActed
        end
    end
    S-->>UC: TurnTransition {Closed, Opened, resoluções, Damaged}
    R->>R: persistClosedTurn(turno fechado)
    R->>DB: PersistTurnClose — action · reactions · resolution · overrides
    R-->>R: publishResolution(fechado) — SETTLED, projetado por destinatário
    R-->>R: turn_opened (mesa inteira)
    R-->>M: resolution_updated do aberto — master-only (não liquidado)
```

**São dois porteiros, não um:**

| | Porteiro |
|---|---|
| 1ª action do personagem no round | a **barra** alcança o preço: `carry + rolagem ≥ preço` |
| 2ª em diante | o **troco** das que já agiram alcança o preço |

**`closeOpenTurn` é o coração.** Ele resolve, **aplica o dano nas fichas** e avança os ledgers
— nessa ordem, para que um modificador que valia neste turno ainda conte. Toda rota que fecha
turno passa por ele.

**`pull_action` reusa a mesma operação, sem porteiro.** Antecipar uma action é prerrogativa do
mestre.

## Tempo 3 — reagir

```mermaid
sequenceDiagram
    autonumber
    participant A as Alvo
    participant R as Room
    participant UC as AttachReactionUC
    participant S as MatchSession
    participant M as Mestre

    A->>R: enqueue_action {reactToId, reactionKind, dodge?/repel?/move?}
    R->>R: handleReaction
    R->>UC: Execute(session, userUUID, reaction)
    UC->>S: AttachReaction(reaction)
    S->>S: turno fechado? → ErrTurnAlreadyClosed (ANTES de rolar ou cobrar)
    S->>S: o kind traz os componentes que exige? senão recusa
    S->>S: rollActionDice · deriveSpeeds
    S->>S: cobra as barras do KIND — no attach, nunca no open, nunca negada por saldo
    S->>S: desloca a action pendente daquela barra
    S->>S: ResolveTurn — recalcula a colisão inteira
    R-->>M: resolution_updated com pendingReactions (master-only)
    M->>R: open_reaction {reactionId}
    R->>S: OpenReaction — entra na cadeia, NA ORDEM DO MESTRE
    R-->>M: reaction_opened · resolution_updated
```

**O custo sai do tipo declarado, não da forma.** Os três escapes carregam exatamente os mesmos
campos e custam três coisas diferentes:

| `ReactionKind` | Barra de ação | Barra de movimento |
|---|---|---|
| `nothing`, `dodge`, `closedDodge` | — | — |
| `repel` | ✔ | — |
| `closedEscape` | — | ✔ |
| `escape`, `escapeGuard` | ✔ | ✔ |

**Uma reaction anexada e não aberta NÃO vira passo da cadeia.** É deliberado: se ela afetasse a
colisão antes de o mestre dar a palavra, a ordem de abertura deixaria de importar — e a ordem
importar *é* o poder de jogo desta fase.

## Tempo 4 — editar

```mermaid
sequenceDiagram
    autonumber
    participant M as Mestre
    participant R as Room
    participant S as MatchSession
    participant L as overrides (em memória)

    M->>R: edit_action {actionId, conditions?, skills?, targetIds?}
    R->>S: ApplyMasterAction(ma, masterUUID)
    S->>S: turno aberto? senão ErrNoActiveTurn
    S->>S: VALIDA TUDO numa cópia-sombra antes de mutar qualquer coisa
    S->>L: captura o valor ORIGINAL — uma linha por campo
    S->>S: muta a action AO VIVO · re-deriva sem re-sortear
    S-->>R: resolução recalculada
    R-->>M: action_edited
    R-->>R: publishResolution
```

**A action editada É a action.** Não existe versão paralela para mesclar na leitura — todo
consumidor lê um lugar só. O preço disso: *"o que o jogador mandou"* deixa de ser algo que se
lê e passa a ser algo que se **reconstrói**, pela tabela de sobrepostos.

**Valida antes de mutar.** Uma falha no meio do laço deixava meia edição aplicada e reportava
fracasso puro — o mestre acreditava não ter mudado nada.

**Reverter sai de graça.** Uma linha por **campo**, guardando o original. Editar de volta ao
valor original apaga a captura e não grava linha nenhuma. É isso que dispensa um verbo de
confirmação: o que um "confirmar" daria de real seria *cancelar*.

**A edição muda o desfecho, nunca a economia.** Barras cobradas, `Speeds` registrados e ordem
já jogada ficam como estão.

## Tempo 5 — fechar

```mermaid
flowchart TB
    CT["close_turn {confirm?}"] --> CHK{"há reaction<br/>anexada e não aberta?"}
    CHK -->|sim, sem confirm| REF["close_turn_refused<br/>+ a lista de quem ficou sem narrar"]
    CHK -->|não, ou confirm=true| CLOSE["closeOpenTurn()"]
    CLOSE --> RES["resolve · aplica dano · avança ledgers"]
    RES --> PUB["turn_closed (mesa)<br/>resolution_updated SETTLED, projetado"]
    RES --> DB[("PersistTurnClose")]
    DB --> D1["actions — action + reactions"]
    DB --> D2["turns.resolution — a colisão"]
    DB --> D3["overridden_action_values — o que o mestre atropelou"]

    style REF fill:#fff3cd,stroke:#856404
```

Essas reactions **entram no cálculo**; o que elas perdem é o momento de narrar. É por isso que
a confirmação existe — e ela é **computada pelo servidor**, não uma cortesia do cliente.

## Quem vê o quê

```mermaid
flowchart LR
    RES["TurnResolution"] --> SET{"IsSettled?"}
    SET -->|"não — turno aberto"| MO["só o mestre<br/><i>o cálculo é dele até fechar</i>"]
    SET -->|"sim — turno fechado"| PROJ["ProjectResolution<br/>por destinatário"]
    PROJ --> V1["mestre<br/>tudo"]
    PROJ --> V2["dono<br/>tudo o que é dele"]
    PROJ --> V3["todo o resto<br/>tudo menos a deny-list"]

    style MO fill:#e3e3ff,stroke:#4444aa
```

**Dois eixos, não um:** *tempo* (`IsSettled`) e *classe* (mestre / dono / resto). **O alvo não
é classe** — uma finta contra você não te conta que era finta.

**O rótulo é o vazamento:** `closedDodge` chega a terceiros como `dodge`, `closedEscape` como
`escape`. Deduzir da barra pública é legítimo — o escape fechado cobra uma barra onde o padrão
cobra duas. Ser avisado de graça pelo rótulo não é.

## Referências

Regras e o porquê de cada decisão: [`../combat-engine.md`](../combat-engine.md).
O que ainda está oco: [`05-lacunas.md`](05-lacunas.md).
