# Efeitos de status — desenho (fase 1)

Contrato do fio: [`../api/match-status-effects.md`](../api/match-status-effects.md). Spec:
`docs/superpowers/specs/2026-11-15-status-effects-design.md` (e `.pt-br.md`). Plano:
`docs/superpowers/plans/2026-11-17-status-effects-phase-1.md`.

A fase 1 entrega o **catálogo, o armazenamento, o comando do mestre, a visibilidade e os
bloqueios**. Tudo o que muda uma rolagem é a fase seguinte (ver "O que não está aqui").

## Mapa de pacotes

| Peça | Onde |
|---|---|
| Domínio puro: tipos, parâmetros, peças, conjunto, quem vê | `internal/domain/statuseffect/` |
| Sessão: guarda e consulta os efeitos de cada ficha; recusas e refazer pendências | `internal/domain/match/matchsession/status_effects.go` (+ `error.go`) |
| Caso de uso (sem I/O): aplicar, editar, remover | `internal/application/match/status_effects.go` (`StatusEffectsUC`, `StatusEffectChange`) |
| Carga ao iniciar a partida | `internal/application/match/init_match_session.go` (`WithStatusEffects`) |
| Gateway | `internal/gateway/pg/statuseffect/` (`ListBySheets`, `Write`); migration `20261117000000_character_status_effects.sql` |
| Sala: comandos, projeção por destinatário, gravação | `internal/app/game/status_effects.go`, `sight.go`, `room.go`, `master_board.go` |
| Fio | `internal/app/game/message.go`; forma compartilhada sala + histórico REST em `internal/app/wire/statuswire/` |
| Histórico: eventos, revelações, âncoras | `domain/statuseffect/event.go`; `gateway/pg/statuseffect/{events,reveals}.go`; `application/match/{get_match_history,reveal_status_effect}.go`; sala: `internal/app/game/status_anchor.go` |

O domínio `statuseffect` não importa nada de fora de `domain/entity`, exceto `domain/masteraction` (o `View` das `views` de um evento de histórico, `event.go`).

## As peças e a contenção

Um efeito não carrega regras: **compõe peças** (`Effect.Pieces()`, `piece.go`), as unidades que
o motor lê: `noAct`, `noReact`, `domainLocked`, `domainWarning`, `reactionRestricted`,
`testPenalty`, `autoFail`, `noDisplacement`, `losesHp`, `recovery`, `triggeredTest`, `conduct`,
`control`, `counter`. A **contenção** está no código, não no banco: Atordoado = Paralisado nos
quatro domínios + Atônito + falha de sentidos + Recuperação; Inconsciente herda o que o Atordoado
**faz**, não como ele **termina** (sem Recuperação por Tenacidade); Lesionado `disabled` também
carrega a desvantagem de `hurt`. Isso fecha a decisão 5 do plano: herda-se o que um status faz,
não como acaba.

**Só as peças que bloqueiam** têm consumidor hoje, e todas passam por **`Summary`**
(`summary.go`, `Summarize`): `CannotAct`, `CannotReact`, `NoDisplacement`, `LockedDomains`,
`ForbiddenReactions` e as `Notes` (aviso de domínio e conduta), cada um com a lista de **quem
causa** (`...By`, usada na mensagem de `blocked_by_status`). A sessão chama `StatusRefusal`
em `EnqueueAction` e `AttachReactionWithCancels`, depois de toda validação existente e antes de rolar
dado, consumir ou cobrar. `testPenalty`, `autoFail`, `losesHp`, `recovery`, `triggeredTest` e
`control` são **declaradas e testadas, e ainda não lidas** pelo motor.

Mapa reação ↔ peça (decisão 9 do plano): `default` = esquiva por reflexo + defesa padrão;
`dodge` = esquiva por reflexo; `closedDodge` = reflexo + evasão; `escape`/`escapeGuard` = fuga;
`closedEscape` = fuga + evasão; `repel` = aparo; `nothing` sempre. Domínios: esquiva →
reflexo, evasão; defesa → defesa padrão, aparo; fuga precisa de movimento **e** esquiva (spec
§5.4).

**G1 (Tato, face equilíbrio), registrado:** prejudicado → ataque, defesa, esquiva e movimento
recebem a desvantagem/número do próprio status; anulado → uma desvantagem nesses quatro.
Declarado como peças agora (`sensePieces`), aplicado na fase das rolagens.

### A reação padrão sentada segue os status

`seatDefaultReactions` agora senta `seatedReactionFor`: a `default` inteira; a `default` com
uma das passivas tirada (`WithoutReflexDodge` / `WithoutDefaultDefense` na `action.Action`, o
resolvedor as lê: sem reflexo pula a esquiva, sem defesa pula a defesa padrão); ou `nothing`,
quando o personagem não pode reagir ou perdeu as duas passivas. **Esses dois sinalizadores não
são persistidos** (`actioncodec` grava `DefaultDefense`, não eles, e nenhum mapper do fio os lê):
o histórico mostra uma `default` simples. O dano aplicado no fechamento é calculado em memória
**com** os sinalizadores, então está correto.

`DropRefusedByStatus` (a cada mudança de status do personagem, remoções inclusive) tira da fila
o que o conjunto recusa (com os movimentos empilhados, pela mesma cascata do `cancel_action`;
as edições do mestre descartadas; sem marca de troca, porque a decisão é do mestre) e refaz as
reações **não abertas** do turno aberto: sentada de mesmo tipo = só os sinalizadores; sentada de
tipo diferente ou enviada agora recusada = `DropUnopened` e nova sentada. Reação **aberta** nunca
é tocada. Uma reação enviada e substituída mantém o que já gastou (barras, ações consumidas):
nada é devolvido nem registrado.

## Quem vê (lista) e visão (névoa)

Armazenamento da lista: coluna `audience` JSONB **anulável** (`NULL` = padrão do tipo;
`{"everyone":true}`; `{"players":[…]}`; `{"players":[]}` = só o mestre). `statuseffect.Listed`
é a **única** conta de lista (`audience.go`): o mestre sempre; com lista, só quem está nela
(o dono não entra implicitamente); sem lista, público do tipo → todos, NPC → ninguém, PJ → só o
dono (ou todos, com `MatchRules.PlayerStatusEffectsPublic`).

A **visão** é somada pela sala em `seesCharacterLocked` (`sight.go`): o dono sempre; um
Escondido (`hiddenFromLocked`, que reusa `Listed` com o mesmo `Subject` da projeção —
`statusSubjectLocked` existe para as duas contas nunca divergirem) não é visto por quem está fora
da lista; sem tabuleiro, todos; com tabuleiro, o personagem precisa de peça, a peça não
escondida **para esse jogador** (`pieceHiddenForLocked`: `visible:false` global **ou** Escondido
por jogador, nunca escrito em `Visible`) e o centro dela dentro da visão do jogador.

**Projeção por destinatário** (`projectStatusEffectsLocked`): o mestre vê tudo com `visibility`
e `public`; os demais, o que `Listed` dá, e só se `seesCharacterLocked`. `syncStatusViews` compara
com o que cada um recebeu da última vez (`statusViews`, por jogador e personagem) e manda um
`status_effects_updated` só quando difere; quem não vê o personagem agora não recebe nem tem o
cache tocado (fica com o último estado visto). Lista vazia e sem entrada no cache valem o mesmo
(não há o que apagar).

**`syncSight`** é o ponto de reacerto depois de tudo que pode mudar a visão (peça que andou ou
saiu, parede, NPC que entrou, Escondido que mudou): `broadcastBars` (cada jogador, a vista dele) +
`syncStatusSight`. É adiado no fim de `relayPieceMove`/`relayPieceRemoved` e chamado em
`enrollLiveNPC`. Um Escondido que muda tira antes um retrato de quem vê cada peça
(`pieceSightLocked`), e `relayHiddenChange` manda `piece_removed`/`piece_moved` a quem perdeu ou
ganhou, antes de barras e status.

Barras também são por jogador (`BarsViewFor(isMaster, sees)`; cache de dedupe por jogador) e
`npc_added` vai só ao mestre (decisão 3).

## Quando se grava, e como se cura

Regra da casa (spec §10): **dentro de um turno aberto nada é gravado antes do fechamento.** O
**momento da gravação é decidido pela sala**, não pelo caso de uso (que não faz I/O e só devolve
`StatusEffectChange.Writes`): `holdStatusEffectWritesLocked`.

- Turno aberto: as gravações vão para `turnWrites.statusEffects` e o `persistClosedTurn` as leva
  em `TurnCloseData.StatusEffects`, na transação do turno.
- Fora de turno, com repositório: entram no fim de `unwrittenStatusEffects` e o chamador, já sem
  trava, chama `flushStatusEffects` (**grava antes de avisar**, como as master actions: o aviso
  serve de barreira nos testes).
- Sem repositório: não grava nem acumula.

**Cura:** uma gravação que falha continua pendente (`unwrittenStatusEffects`, em `unwritten.go`)
e vai **na frente** da próxima gravação que dá certo: a próxima mudança de status fora de turno,
o próximo fechamento de turno (que também devolve as do turno que falhou, logo depois do prefixo
que copiou: a ordem real) **ou o `end_match`** (`MatchEndData.StatusEffects`, na transação do fim:
`PersistMatchEnd`). Cada gravador descarta **exatamente o prefixo que copiou**. Uma sala que
fecha com gravações pendentes (ou com um turno aberto) registra no log (`logDroppedTurn`).

Limite conhecido de ordem: entre o caso de uso fechar o turno em memória e o `persistClosedTurn`
tirar o retrato, um **segundo socket do mestre** pode fazer uma mudança fora de turno que entra
em `unwrittenStatusEffects` na frente das do turno que fecha; se ela edita o mesmo efeito, a
ordem de gravação seria edit → apply. Com um socket só não acontece (o fechamento e o próximo
comando dividem a bomba de leitura).

### Cadeia de trava

`persistMu` → `r.mu` é o aninhamento permitido de tudo que grava (tabuleiro, fechamento de
turno, efeitos de status). **Só o `end_match`** segura a cadeia maior: `clockPersistMu` →
`persistMu` → `r.mu` (precisa de `persistMu` para a cópia e o descarte do prefixo de status não
correrem contra um `flushStatusEffects`). Não há deadlock porque nada que segura `persistMu`
toma `clockPersistMu` (`persistClosedTurn`, `flushStatusEffects`, `savePersistedBoard` e o
recarregar+salvar do `StartMatch` não chamam `persistClock` nem o checkpoint). Nunca se segura
`r.mu` durante I/O nem ao chamar `dispatchPerPlayer` (que toma `r.mu`).

## NPC ao vivo

`enrollLiveNPC` (add_npc com sessão viva): sessão recebe o NPC, `npc_added` vai ao mestre,
`loadLiveNPCStatusEffects` lê os efeitos (`ListBySheets`, sem trava; falha = NPC sem status, com
log) e só então `syncSight()` manda barras e status. O `npc_added` (`sendToMaster`) e o `syncSight`
seguinte correm na mesma goroutine e na mesma pista direta: o mestre recebe `npc_added` **antes**
dos efeitos do NPC. A leitura usa
`ReplaceStatusEffects`: um status aplicado por um segundo socket entre a inscrição e a leitura
seria sobrescrito (desprezível).

## Histórico: eventos, listas, revelações e âncoras

Contrato: [`../api/match-history.md`](../api/match-history.md).

**Eventos.** Cada mudança (`applied`/`edited`/`removed`) carrega um `statuseffect.Event`
(`Write.Event`), carimbado pela sala no instante e gravado **na mesma operação** do efeito
(`status_effect_events`: `snapshot`, `listing`, `views`, `hidden_from`, `action_uuid`, `seq` para
desempatar). O caso de uso continua sem I/O. `ResolveListing` resolve a lista do instante (o
padrão do tipo já aplicado, nunca "padrão"); `views` = jogadores que enxergavam o personagem logo
antes ou logo depois, mais o dono (o mestre nunca entra).

**Quem vê um evento** é uma regra só, `statuseffect.VisibleInHistory`: mestre; ou (âncora visível
**e** (revelação do efeito cobre o instante **ou** (leitor em `views`, ou em `hidden_from` com o
Escondido já revelado, **e** a última lista do efeito inclui o leitor))). A última lista
(`LatestListings`) vem de qualquer partida; sem evento anterior, nenhuma → falha fechada.

**`hidden_from`** (sala → banco): esparso, `NULL` = ninguém. `cancelled_actions` **não tem**
`hidden_from`: a action cancelada só chega ao mestre e ao dono do ator (dono do produto,
2026-11-20), que enxergam através do Escondido, então nenhum leitor dependeria de uma lista ali.
Em `status_effect_events` é
`{playerId: hiddenEffectId}` (jogadores da sessão para quem o personagem estava Escondido) — lá
ele só **abre** o evento a quem estava de fora; a lista de permissão do evento é `views`, então
nomear só a sessão não vaza nada. Em `actions` é uma **lista de permissão**
(`statuseffect.HiddenFrom`, `hiddenRecordLocked`): `{charId: {effect, perceivedBy: [playerId…]}}`,
por personagem Escondido com lista que não é "todos", com a lista resolvida do instante
(`ResolveListing`) — quem não está nela não o percebia, inclusive quem não era jogador da sessão
(leitor de fora da partida pública, membro da campanha não inscrito, inscrito depois). O dono e o
mestre são decididos na leitura. Cobre o ator e cada alvo na linha da action (abertura do turno)
e o reator na da reação (na abertura dela; nunca aberta, no fechamento). A sentinela de linha
ilegível é o registro zero (efeito `uuid.Nil`, ninguém): esconde de todos e não é revelável. O
codec é um só (`actioncodec.EncodeHiddenFrom`/`DecodeHiddenFrom`). Nunca sai no wire.

**Revelações** (`status_effect_reveals`, `reveals.go`): só o REST do mestre insere, e só um efeito
revelável na partida (`RevealableInMatch`: evento em qualquer partida + ficha participante desta —
um Escondido trazido de outra partida também); o histórico lê
o `MAX(revealed_at)` por efeito (`RevealedAt`) uma vez por requisição, junto de `LatestListings`.
`RevealLookup.Covers` = revelado em ou depois do instante. Ampliar a lista nunca revela.

**Âncoras** (`actionId`, a causa; `status_anchor.go` + `matchsession/anchor_moves.go`). Só o mestre
informa, e só uma action/reação do turno aberto. Enquanto o turno está aberto a sala as mantém
sob `r.mu`: **seguem** (reação sentada substituída pela do jogador, ou refeita por status → a nova;
sentada que sai sem substituta → a action do turno); **relink** (`relink_status_effect`) as troca
à mão; o mestre vê a âncora viva em `turnActionId`. **Desfazer** (`remove_status_effect` de um
efeito aplicado neste mesmo turno) apaga todas as gravações e eventos pendentes dele e tira o id
dele de `hidden_from` das ações/eventos do turno (`purgeEffectFromTurnLocked`: nas ações, a
entrada do personagem sai inteira); as ações da fila
que ele cancelou continuam canceladas. Depois do fechamento o histórico é definitivo.

**Horários reais.** O turno guarda a abertura (`Turn.GetOpenedAt`, relógio da sala, µs):
`turns.created_at` é ela; cada reação ganha `actions.opened_at` (NULL = nunca aberta). A regra de
"escondido" compara a revelação com esses instantes.

## Limitação: uma ficha viva em duas salas

Cada sala guarda os efeitos **na própria sessão** e grava o que mudou nela. Uma mesma ficha viva
em duas partidas ao mesmo tempo tem **duas memórias**: a última gravação ganha no banco, sem
merge. Não é tratado nesta fase.

## O que não está aqui

Rolagens (penalidades, desvantagem, falha automática, Lento, G1), testes automáticos, dano e
recuperação por round, asfixia, "não usa habilidades" (Nen), revelar a um jogador só (a revelação
é para todos), push de WS depois de revelar e desfazer depois do fechamento do turno. Lista no
contrato, §10.
