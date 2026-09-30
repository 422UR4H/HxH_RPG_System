# Match History API

## GET /matches/{uuid}/history — Histórico de ações da partida

**Auth:** JWT obrigatório

### Visibilidade

Mesma regra de acesso de `GET /matches/{uuid}/participants`:

- **Mestre da partida:** sempre vê.
- **Partida pública:** qualquer usuário autenticado vê.
- **Partida privada:** apenas participantes (jogadores com personagem na campanha) veem;
  demais recebem 403.

Dentro da resposta, porém, **cada campo tem sua própria visibilidade** — ver a seção
"A resposta já vem projetada" abaixo. Isso é ortogonal à autorização do endpoint: passar
na autorização só dá acesso ao histórico *como este usuário pode vê-lo*, nunca ao histórico
bruto.

### Response 200

Estrutura aninhada Scene → Round → Turn → Action, o mesmo formato que o domínio já
organiza internamente (`docs/dev/match/combat-engine.md`), sem achatamento: o front
renderiza os cards de ação dentro do escopo de cada cena. Cada round carrega também
`events` — o que aconteceu nele que **não é turno** — e cada turno, `masterActions`. Ver
"O que não é turno" abaixo.

```json
{
  "scenes": [
    {
      "uuid": "a0a0...",
      "category": "roleplay",
      "briefDesc": "Taverna",
      "createdAt": "2026-08-20T13:40:00Z",
      "finishedAt": "2026-08-20T13:59:00Z",
      "rounds": [
        {
          "uuid": "0f0f...",
          "mode": "Free",
          "createdAt": "2026-08-20T13:40:00Z",
          "finishedAt": "2026-08-20T13:59:00Z",
          "turns": [],
          "events": []
        }
      ]
    },
    {
      "uuid": "b3f1...",
      "category": "battle",
      "briefDesc": "Emboscada na floresta",
      "createdAt": "2026-08-20T14:00:00Z",
      "finishedAt": "2026-08-20T14:40:00Z",
      "rounds": [
        {
          "uuid": "1a2b...",
          "mode": "Race",
          "createdAt": "2026-08-20T14:00:05Z",
          "finishedAt": "2026-08-20T14:12:00Z",
          "events": [
            {
              "uuid": "e1e1...",
              "kind": "roundModeChanged",
              "createdAt": "2026-08-20T14:00:20Z",
              "payload": { "from": "Free", "to": "Race" }
            },
            {
              "uuid": "ma02...",
              "kind": "masterAction",
              "createdAt": "2026-08-20T14:02:10Z",
              "masterAction": {
                "uuid": "ma02...",
                "kind": "movePiece",
                "happenedAt": "2026-08-20T14:02:10Z",
                "content": { "characterId": "char-gon...", "pieceId": "piece-gon", "from": [4, 4, 0], "to": [6, 4, 0] }
              }
            }
          ],
          "turns": [
            {
              "uuid": "9c9c...",
              "createdAt": "2026-08-20T14:01:00Z",
              "finishedAt": "2026-08-20T14:01:30Z",
              "masterActions": [
                {
                  "uuid": "ma01...",
                  "kind": "wallInteract",
                  "turnId": "9c9c...",
                  "happenedAt": "2026-08-20T14:01:10Z",
                  "content": { "wallIds": ["wall-3"], "interact": "open" }
                }
              ],
              "action": {
                "uuid": "aa11...",
                "actorId": "char-gon...",
                "targetId": ["char-hisoka..."],
                "reactionKind": "",
                "skills": [
                  { "skillName": "Legerity", "rollCheck": { "skillName": "Legerity", "skillValue": 14, "attempts": { "primary": [6, 8] }, "result": 14 } }
                ],
                "speed": { "bar": 1, "rollCheck": { "skillName": "Legerity", "skillValue": 14, "attempts": { "primary": [6, 8] }, "result": 14 } },
                "move": {
                  "category": "Dash",
                  "from": [1, 1, 0],
                  "position": [4, 4, 0],
                  "speed": { "skillName": "Legerity", "skillValue": 4, "attempts": { "primary": [2, 5] }, "result": 13 },
                  "charge": { "skillName": "Legerity", "skillValue": 4, "attempts": { "primary": [1] }, "result": 6 },
                  "finalSpeed": 9
                },
                "attack": {
                  "weapon": "Fist",
                  "hit": { "skillName": "Legerity", "skillValue": 14, "attempts": { "primary": [6, 8] }, "result": 20 },
                  "damage": { "skillName": "Push", "skillValue": 10, "attempts": { "primary": [4] }, "result": 10 },
                  "relativeVelocity": 0
                }
              },
              "reactions": [
                {
                  "uuid": "bb22...",
                  "actorId": "char-hisoka...",
                  "reactToId": "aa11...",
                  "reactionKind": "dodge",
                  "systemBias": -1,
                  "skills": [
                    { "skillName": "Legerity", "rollCheck": { "skillName": "Legerity", "skillValue": 12, "attempts": { "primary": [5, 5] }, "result": 12 } }
                  ],
                  "speed": { "bar": 1, "rollCheck": { "skillName": "Legerity", "skillValue": 12, "attempts": { "primary": [5, 5] }, "result": 12 } },
                  "dodge": {
                    "rollCheck": { "skillName": "Legerity", "skillValue": 12, "attempts": { "primary": [5, 5] }, "result": 12 }
                  }
                }
              ],
              "resolution": {
                "isSettled": true,
                "action": { "skillName": "Legerity", "skillValue": 14, "diceRolled": [6, 8], "total": 20, "isCritical": false, "isCriticalFailure": false },
                "targets": [
                  {
                    "targetId": "char-hisoka...",
                    "avoided": false,
                    "defended": false,
                    "dodgeTotal": 12,
                    "defenseTotal": 0,
                    "rawDamage": 10,
                    "defenseApplied": 0,
                    "projectedDamage": 10,
                    "reaction": {
                      "kind": "dodge",
                      "total": 12,
                      "reactionId": "bb22...",
                      "stopsAttack": false
                    },
                    "payouts": [
                      {
                        "amount": -3,
                        "bias": 0,
                        "applies": "action_speed",
                        "source": "system",
                        "againstKind": "anyone",
                        "againstId": "00000000-0000-0000-0000-000000000000",
                        "expiresAt": "next_turn",
                        "reason": "repel: near miss penalty"
                      }
                    ]
                  }
                ],
                "errors": [
                  {
                    "subject": "char-desconhecido...",
                    "kind": "unknown_target",
                    "detail": "action target is neither a character nor a wall segment"
                  }
                ]
              }
            }
          ]
        }
      ]
    }
  ]
}
```

O exemplo mostra uma cena **sem turno** (a primeira: começou e terminou sem que um turno
fechasse nela — aparece mesmo assim, com `turns: []`), e uma master action de cada lugar:
a de parede dentro do turno em que foi aplicada, a de peça fora de turno, em `events`.

Notas sobre `scenes[]` e `rounds[]`:

- `category` é `battle` ou `roleplay`; `mode` é `Free` ou `Race` — valores de enum do
  domínio, como estão.
- **Cena e round aparecem desde que nascem**, não só quando o primeiro turno fecha neles
  (B15): são gravados no `start_match` (ou na reidratação depois de um reinício), no
  `change_scene` e quando um round fecha por exaustão e outro nasce. Uma cena em que se só
  conversou, um round que fechou sem turno — aparecem, com `turns: []`.
- `mode` do round é o **último** regime em que ele esteve. Por onde ele passou, e quando, está
  nos `events` (`roundModeChanged`).
- `finishedAt` ausente = round/cena ainda aberto.

Notas sobre os campos de `action`/`reactions`:

- `skills`, `move`, `attack`, `defense`, `dodge`, `repel`, `interact` só aparecem quando a
  action de fato os carrega — ausentes (não `null`), do contrário.
- `move.from` é a posição da PEÇA do ator no tabuleiro no instante do enfileiramento — nunca o
  que o cliente declarou em `move.from` do payload de `enqueue_action`, que o servidor sempre
  descarta (B6, spec §4.3 "B5, B6 e B10"). Ausente quando o ator não tinha peça no tabuleiro.
  Convenção de coordenada, igual em `move.position`: `[a, b, z]`, `(a, b) = (col, row)` numa
  grade quadrada ou `(q, r)` axial numa hexagonal; `z` não é lido pelo servidor. Uma reação
  (em `reactions[]`) nunca tem `move.from` — o campo não é derivado para o lado da reação.
- `trigger` é omitido por completo quando o viewer não é dono nem mestre; quando presente, é
  um objeto vazio (o domínio ainda não tem campos em `action.Trigger`).
- `feint` segue uma regra **temporal**, não de classe: `ProjectAction`
  (`internal/domain/match/service/projection.go`) só o esconde de quem não é dono/mestre
  enquanto **aquele turno** ainda está **aberto** (`isSettled: false`, calculado por turno a
  partir de `FinishedAt` — não é uma constante global) — quem caiu na finta descobre dentro
  da resolução do MESMO turno, porque o sucesso dele foi contra um ataque falso e o de
  verdade vem em seguida; esconder depois do fechamento esconderia para sempre, que não é a
  regra. **Na prática de hoje**, todo turno que chega a este endpoint já está fechado
  (`FindMatchHistory` lê da tabela que `PersistTurnClose` grava, o único caminho de escrita
  atual) — então `feint`, quando presente na action, chega a todo viewer, não só a
  dono/mestre. Mas isso é uma consequência de `isSettled` ser sempre `true` aqui hoje, não
  uma regra separada codificada no endpoint: o dia em que um turno aberto atravessar este
  caminho (não acontece agora), a finta dele voltaria a ficar restrita a dono/mestre, turno a
  turno. Quando presente, `feint` é o `RollCheck` da finta. Ver
  [`match-combat-ws.md`](match-combat-ws.md), onde a mesma regra passou a valer também do lado
  do WebSocket desde B2 (design spec §4.2, PR de fechamento da Fase 6): `turn_opened.action`
  agora projeta a declaração de uma action de jogador, e `feint` segue exatamente este eixo
  do TEMPO ali — escondida de um terceiro enquanto o turno está aberto, visível (com ou sem
  números, conforme o destinatário) para o mestre e para o dono. Ver a seção de `turn_opened`
  nesse contrato.
- `reactToId` só aparece em uma reaction (uma action raiz não reage a nada).
- `systemBias` é o viés que o **próprio motor** impôs: `0` numa ação comum, `-1` numa reação
  que deslocou uma ação enfileirada (trocar o que você ia fazer custa Desvantagem). Vai para
  **todo** viewer, pela mesma razão que `attempts` vai (item abaixo): o viés é **público por
  omissão**. Se os dois conjuntos de dados e o `result` já viajam, QUAL conjunto o motor leu
  já é dedutível — esconder o campo só obrigaria o cliente à álgebra que este repo evita de
  propósito (ver `CharacterResult.ReactionTotal`). Omitido quando é `0`, que é a esmagadora
  maioria das actions.
- `RollCheck.Context` (que carrega `RollCondition`, a vantagem/desvantagem que o **mestre**
  aplicou via `edit_action`) **continua interno** e não aparece em superfície nenhuma — nem
  aqui, nem no WebSocket. Não é o mesmo caso de `systemBias`: a intervenção do mestre já tem
  superfície própria, em `overridden_action_values`, que registra o valor ANTERIOR junto com
  quem trocou e quando. O que o cliente vê aqui são os números já resolvidos
  (`actionwire.RollCheck.result`, os totais em `resolution`).
- `systemBias` **agora tem equivalente no WebSocket** (desde B2, design spec §4.2) —
  `turn_opened.action.systemBias` e `match_full_state.openTurn.action.systemBias` carregam o
  mesmo valor. Antes de B2 isso era verdade por falta de superfície: `ActionPayload` só existe
  no sentido cliente→servidor, e `action_queued` (B1) — a primeira superfície a carregar a
  ação inteira — é master-only, então nunca precisou do argumento "já é dedutível" (o mestre
  já era dono de tudo). B2 é a primeira vez que esse argumento passa a valer para quem NÃO é
  mestre: `actionwire.Action.SystemBias` **não é cortado por nível** (ver
  `internal/app/wire/actionwire`), então sobrevive ao corte de `Opened` igual sobrevive ao
  `Full` do mestre — e o motivo é o mesmo de sempre, público por omissão. `master_action_enqueued`
  continua sem `ActionPayload` nem `systemBias` — o viés só existe em ações e reações de
  jogador (`buildAction`), nunca em `buildMasterAction`. Ver
  [`match-combat-ws.md`](match-combat-ws.md).
- `actionwire.RollCheck.attempts` (`primary` e, quando existir, `secondary`) vai para **todo**
  viewer, sem deny-list própria — isso não viola a política de visibilidade porque o viés é
  público por omissão: nada esconde QUAL conjunto o motor leu, então mostrar os dois não
  vaza mais do que o total já vaza. Mas é uma superfície de dados estritamente maior que o
  WebSocket: `resolution_updated` só emite `diceRolled`, o conjunto efetivamente lido —
  `attempts` do REST é o único lugar onde o conjunto NÃO lido também aparece.

Notas sobre `resolution.targets[]`:

- `payouts` é o que a reação **daquele alvo rendeu**: o bônus ou a penalidade do aparar, a
  reserva da esquiva fechada. Ausente quando não rendeu nada, que é a maioria das reações.
  Um payout é um modificador acumulado no personagem, escrito na ficha no fechamento do
  turno; `againstKind` (`anyone` · `only` · `all_but`) é o ponto dele — diz **quem** pode
  contá-lo — e `againstId` é o personagem em que os dois últimos se apoiam (zero UUID em
  `anyone`, que é o que esse caso significa). `amount` é ajuste plano; `bias` é
  vantagem/desvantagem nos dados (−1/0/+1), moeda diferente que não se soma ao total.
- `payouts` **está sujeito à projeção**, e por uma condição só: a mesma que rebaixa o rótulo.
  A reserva da esquiva fechada é a outra metade daquele segredo — o tamanho da esquiva não
  gasta diz quanta Evasão foi embutida — então sai junto com o nome. **A penalidade do aparar
  não sai:** nasce `againstKind: "anyone"`, um aparo nunca é rebaixado, e
  [`reacoes.md`](../../game/combate/reacoes.md) diz que *"vale contra todo mundo — qualquer um
  pode aproveitar"*. Quem pode aproveitar precisa conseguir ler.
- `applies`, `source`, `againstKind`, `expiresAt` e `reaction.rung` são **snake_case**: são
  valores de enum do domínio serializados como estão, não tags de struct.
- `escape` é o veredito de uma **fuga** (`escape`, `escapeGuard`, `closedEscape`) — **ausente**
  em todo alvo que não fugiu. Mesma forma do `targets[].escape` do `resolution_updated` do
  WebSocket ([`match-combat-ws.md`](match-combat-ws.md#resolution_updated)):

  ```json
  "escape": { "escaped": false, "movePassed": false, "dodgePassed": true, "awaitsMaster": false, "landing": [7, 6, 0] }
  ```

  `escaped` = `movePassed` **e** `dodgePassed`, os dois contra o acerto do atacante. É o que
  diz, depois, por que uma peça andou ou não: escapou → foi ao destino da fuga; falhou com
  `landing` → foi para onde o mestre escolheu; falhou sem `landing` (`awaitsMaster: true`, que
  num turno fechado se lê "ficou") → não saiu do lugar. `landing` (`[col, row, z]`) só aparece
  numa fuga que falhou. Persistido com a resolução do turno
  (`internal/gateway/pg/round/resolution_record.go`); turnos gravados antes deste campo não o
  trazem.

- `errors` só aparece quando o motor **não conseguiu** calcular parte da colisão, o que é
  raro — então a presença dela é o sinal. **Não é mensagem de erro:** o request não falhou e
  o turno não falhou; os números ao lado são reais e falta um pedaço da colisão neles.
  `kind` é o discriminador estável (`unknown_target` · `missing_sheet` · `no_attack`),
  `subject` é o UUID que o motor não resolveu, e `detail` é prosa para humano — não parseie.
  Um `missing_sheet` quer dizer que um alvo **não produziu entrada em `targets`**: é
  exatamente o silêncio que este campo existe para quebrar, e é por isso que ele está aqui e
  não só no WebSocket. O caminho ao vivo é efêmero — "o mestre recebe pelo WS" pressupõe
  mestre conectado e olhando naquele instante; o histórico existe porque isso não se pode
  pressupor. Ver `internal/gateway/pg/round/resolution_record.go`, que persiste as faltas
  pela mesma razão.

### O que não é turno — `rounds[].events` e `turns[].masterActions`

Dois tipos de coisa acontecem dentro de um round sem serem o turno de alguém:

- **`roundModeChanged`** — o mestre trocou o regime do round. Gravado em `match_events` no
  instante em que a troca é aplicada (`change_round_mode`); uma troca para o regime em que o
  round já estava não é troca e não grava nada. **Público**, como o próprio regime.
- **master action** — tudo o que o mestre aplicou pelo `enqueue_master_action` e foi aceito:
  arrastar, pôr e tirar peça (`movePiece`, `placePiece`, `removePiece`), interagir com uma
  parede e revelá-la (`wallInteract`, `revealWall`), e as genéricas que só se penduram no
  turno aberto (`turnNote`). Gravadas em `master_actions`, tabela própria (spec §4.8), **no
  instante** em que são aplicadas, com ou sem turno aberto.

**`edit_action` não é master action** e não aparece em lugar nenhum desta resposta: a edição
do mestre continua registrada só em `overridden_action_values` (o valor que ela deslocou), e
o histórico mostra a action já editada, que **é** a action. Um turno editado tem
`masterActions: []`.

**Onde uma master action entra:**

| A master action foi gravada… | Aparece em |
|---|---|
| com o turno aberto, e esse turno está no histórico | `turns[].masterActions` daquele turno |
| sem turno aberto (o arrastar entre turnos é o caso comum) | `rounds[].events`, `kind: "masterAction"` |
| com um turno que **não foi gravado** (um turno só é gravado ao fechar; um reinício com o turno aberto o perde) | `rounds[].events`, `kind: "masterAction"` — o `turnId` continua lá, apontando um turno que não existe na árvore |

A terceira linha é deliberada: a master action aconteceu, o tabuleiro salvo a confirma, e
sumir com ela porque o turno não sobreviveu apagaria algo que a mesa viu.

**Formato de `events[]`** — em ordem de tempo, os dois `kind` intercalados:

| Campo | |
|---|---|
| `uuid` | do evento (`roundModeChanged`) ou da master action (`masterAction`) |
| `kind` | `"roundModeChanged"` \| `"masterAction"` |
| `createdAt` | quando aconteceu |
| `payload` | só em `roundModeChanged`: `{ "from": "Free", "to": "Race" }` |
| `masterAction` | só em `masterAction`: o mesmo objeto de `turns[].masterActions[]` |

**Formato de uma master action** (`turns[].masterActions[]` e `events[].masterAction`):

| Campo | |
|---|---|
| `uuid` | |
| `kind` | `movePiece` · `placePiece` · `removePiece` · `wallInteract` · `revealWall` · `turnNote` |
| `turnId` | o turno aberto quando foi aplicada; **ausente** fora de turno |
| `happenedAt` | quando foi aplicada |
| `content` | o que ela fez — peça: `{ characterId, pieceId, from?, to? }` (`from` ausente num `placePiece`, `to` ausente num `removePiece`); parede: `{ wallIds, interact }` (só as paredes que de fato mudaram); `turnNote`: o payload do `enqueue_master_action` como o mestre o mandou |

`events` e `masterActions` são **sempre listas** — `[]` quando não há nada — nunca `null`.
Dentro de cada uma, a ordem do array é a ordem do tempo; `createdAt`/`happenedAt` têm
precisão de segundo, então o array é quem desempata.

**Projeção — cada leitor vê a master action como a viu ao vivo.** No instante da aplicação o
servidor já decide, jogador a jogador, o que cada um recebe (o portão de fog do
`piece_moved`, a parede que muda às vistas ou não). Essa mesma decisão é gravada com a
master action, para **todo jogador da sessão**, conectado ou não — o que conta é o fog dele
naquele instante —, e o histórico devolve a cada um exatamente aquilo:

| O jogador, ao vivo, … | No histórico ele vê |
|---|---|
| recebeu a master action (o `piece_moved`, a parede mudando, a remoção) | a master action inteira |
| recebeu só o `piece_removed` — viu a peça sair e não viu para onde | a master action **sem o destino** (`content.to` ausente) |
| não recebeu nada | **nada** — a entrada não existe para ele, nem dentro do turno nem em `events` |

O mestre vê todas, inteiras. Revelar parede vai a todos ao vivo, então vai a todos aqui.
`turnNote` não chega à mesa ao vivo (pendura-se no turno e não emite nada), então é só do
mestre. **O registro de quem viu o quê nunca sai no wire** — para ninguém, nem para o mestre:
é como a projeção é decidida, não dado de mesa.

### A resposta já vem projetada — não filtre no cliente

**Este é o ponto central deste endpoint.** O Action History é uma superfície de jogo com
visibilidade por campo, não um log — a mesma política que `resolution_updated` já aplica no
WebSocket (ver `docs/dev/match/combat-engine.md#visibilidade`), rodada aqui pelas MESMAS
funções (`service.ProjectAction`, `service.ProjectResolution`) — e as master actions por
`masteraction.Record.ProjectFor` (seção acima).

Isso significa, na prática:

- **O mesmo turno retorna JSON diferente para viewers diferentes.** O mestre vê tudo; o
  dono de uma action ou reaction vê tudo que é seu; qualquer outro participante vê tudo
  **menos** a deny-list. Não existe uma "verdade única" que o front possa cachear e
  reutilizar entre usuários.
- **O alvo de um ataque não é uma classe privilegiada.** Uma finta contra você não avisa
  que era finta **enquanto aquele turno ainda está aberto**. A regra que esconde `feint` de
  quem não é dono/mestre é **temporal** (`isSettled`, por turno), não de classe — e como hoje
  todo turno que chega a este endpoint já está fechado, `feint`, quando presente, chega **a
  todo viewer** na prática atual. Ver a nota sobre `feint` mais acima.
- **A esquiva fechada chega a terceiros indistinguível de uma esquiva comum.**
  `reactionKind: "closedDodge"` vira `"dodge"` (e `"closedEscape"` vira `"escape"`) para
  quem não é dono nem mestre — o rótulo é o vazamento; ver a nota em
  `internal/domain/match/service/projection.go`. A entrada de `Evasion` em `skills`
  desaparece junto, pela mesma razão.
- **Os números continuam públicos.** Dano, dados rolados, totais — nada disso é escondido,
  porque a dedução ("o adversário deduz dos números") depende deles estarem lá.
- **Dois campos de `resolution` são MASTER-ONLY**, e não por classe de dono: saem para todo
  mundo que não seja o mestre, inclusive do dono do personagem em questão.
  - `pendingReactions` — reações anexadas e ainda não abertas. É a lista de tarefas do
    mestre, não estado de mesa.
  - `errors` — as faltas do **motor** ao calcular aquele turno (ver abaixo). São diagnóstico
    sobre a engine, não fato sobre a ficção, e o mestre é o único que pode fazer algo a
    respeito.

**Não implemente um segundo filtro no front.** O servidor já entrega exatamente o que este
usuário pode ver; uma filtragem client-side redundante só cria uma segunda cópia da
deny-list para divergir da primeira da próxima vez que ela mudar.

### Erros

| Status | Situação |
|---|---|
| 200 | Histórico retornado (`{ "scenes": [] }` para uma partida que nunca começou; uma partida começada tem ao menos a cena e o round em que começou) |
| 400 | UUID inválido |
| 401 | Sem JWT |
| 403 | Partida privada e usuário não é mestre nem participante |
| 404 | Partida não encontrada |
| 500 | Erro interno |
