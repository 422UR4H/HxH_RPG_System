# Handoff — "tentar abrir" a porta (esconder a tranca do jogador)

Estado em 2026-10-11 (back `feat/wall-lock-reach`, front `feat/wall-as-target`).

## O que o dono do produto quer

O jogador não deve saber, de cara, se uma porta está trancada. Ele **tenta abrir** qualquer porta:
- destrancada → abre;
- trancada → ele **descobre** que está trancada (seguindo o padrão que o jogo já usa para cenários
  análogos de informação revelada a quem agiu) e só então aparece "Arrombar fechadura" para ele.

## O que já está pronto

- **Servidor**: `open` (e `toggle`) de personagem numa passagem trancada **não é recusado no envio**;
  o turno resolve e a porta não abre (`ApplyWallInteract(w, kind, byMaster=false)` →
  `ok=false`, sem `WallResult`, sem `wall_state_changed`). Isto é de propósito: é o "tentar".
- **Mestre**: `lock`/`unlock` via `enqueue_master_action`; Abrir/Fechar ignoram a tranca.
- **Alcance**: abrir exige encostar (`out_of_reach`) — vale igual para a tentativa.

## O que falta

1. **Esconder `locked` do jogador.** Hoje vaza em três lugares (`internal/app/game`):
   - `toWallSegmentPayload` (`message.go`) põe `Locked: w.Locked` em toda parede de
     `map_full_state`/`match_full_state` e no `wall_revealed`;
   - `WallStateChangedPayload.Locked` (`broadcastWallStateChangedGated`, `room.go`) vai para todo
     jogador que vê a parede.
   Precisa de uma projeção por leitor: o mestre recebe `locked` real; o jogador recebe `locked`
   só se **ele sabe** (ver 2). O padrão de projeção por leitor já existe para fog
   (`dispatchPerPlayer`, `Viewer.SeesAllOf`, `projection.go`) — siga ele, não invente outro.
2. **Conhecimento por jogador (ou por personagem?)** — decisão de produto em aberto:
   quem descobriu, o personagem que tentou, o dono dele, ou a mesa toda que viu a tentativa?
   O estado vive na sessão (`MatchSession`) e precisa sobreviver a reconexão
   (`match_full_state`) e talvez ao reinício (persistir junto do board, como `revealed`?).
   Quando o mestre destranca/tranca de novo, o conhecimento caduca?
3. **Avisar quem tentou.** No fechamento/abertura do turno da tentativa frustrada, o servidor
   precisa produzir um resultado visível: hoje não há `WallResult` nenhum. Opções: um
   `WallResultKind` novo (`interact_failed`, motivo `locked`) que viaja na resolução do turno
   (`resolution_updated`, eixos tempo × classe, §6 do contrato) e no histórico
   (`match-history.md`). Projete o `locked` revelado só para quem deve saber.
4. **Front**: `actorWallVerbs` (`src/features/match/combat/wallVerbs.ts`) hoje mostra
   "Arrombar fechadura" quando `w.locked`. Com o payload projetado, o mesmo código passa a
   mostrar "Abrir" até o jogador descobrir — não muda nada nele se o `locked` que chega já for
   o "sabido". A linha do turno no histórico/`EventStream` precisa dizer "tentou abrir Porta —
   trancada".
5. **NPC do mestre**: o mestre vê tudo; para o NPC, faz sentido o composer do mestre continuar
   vendo a tranca real (é o mestre quem decide).

## Ordem sugerida

Contrato primeiro (`docs/dev/api/match-combat-ws.md` — `wall_state_changed`, `map_full_state`,
resolução do turno; `match-history.md`), depois back, depois front. Escopo: médio-alto — mexe em
projeção por leitor, que tem histórico de bugs (`docs/dev/fog-of-war-postmortem.md`).

## Arrombar (relacionado)

`lockpick` não rola nada: a corrente de testes de `skills` não é executada (contrato §10) e o
front não tem UI de classe de dificuldade. Decisão do dono: quando houver teste, arrombar é
**Destreza** se teste de atributo físico existir; senão **Accuracy**. Hoje não existe teste de
atributo (`SkillNameFrom` só aceita `enum.SkillName`; `Dexterity` é `AttributeName`). Até lá, o
mestre lê a tentativa e destranca pela mão dele.
