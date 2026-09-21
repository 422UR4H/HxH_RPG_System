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
renderiza os cards de ação dentro do escopo de cada cena.

```json
{
  "scenes": [
    {
      "uuid": "b3f1...",
      "category": "combat",
      "briefDesc": "Emboscada na floresta",
      "createdAt": "2026-08-20T14:00:00Z",
      "finishedAt": "2026-08-20T14:40:00Z",
      "rounds": [
        {
          "uuid": "1a2b...",
          "mode": "combat",
          "createdAt": "2026-08-20T14:00:05Z",
          "finishedAt": "2026-08-20T14:12:00Z",
          "turns": [
            {
              "uuid": "9c9c...",
              "createdAt": "2026-08-20T14:01:00Z",
              "finishedAt": "2026-08-20T14:01:30Z",
              "action": {
                "uuid": "aa11...",
                "actorId": "char-gon...",
                "targetId": ["char-hisoka..."],
                "reactionKind": "",
                "skills": [
                  { "skillName": "Legerity", "rollCheck": { "skillName": "Legerity", "skillValue": 14, "attempts": { "primary": [6, 8] }, "result": 14 } }
                ],
                "speed": { "bar": 1, "rollCheck": { "skillName": "Legerity", "skillValue": 14, "attempts": { "primary": [6, 8] }, "result": 14 } },
                "attack": {
                  "weapon": "fist",
                  "hit": { "skillName": "Legerity", "skillValue": 14, "attempts": { "primary": [6, 8] }, "result": 20 },
                  "damage": { "skillName": "Strength", "skillValue": 10, "attempts": { "primary": [4] }, "result": 10 },
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

Notas sobre os campos de `action`/`reactions`:

- `skills`, `move`, `attack`, `defense`, `dodge`, `repel`, `interact` só aparecem quando a
  action de fato os carrega — ausentes (não `null`), do contrário.
- `trigger` é omitido por completo quando o viewer não é dono nem mestre; quando presente, é
  um objeto vazio (o domínio ainda não tem campos em `action.Trigger`).
- `feint` segue uma regra **temporal**, não de classe: `ProjectAction`
  (`internal/domain/match/service/projection.go`) só o esconde de quem não é dono/mestre
  enquanto o turno ainda está **aberto** (`isSettled: false`) — quem caiu na finta descobre
  dentro da resolução do MESMO turno, porque o sucesso dele foi contra um ataque falso e o de
  verdade vem em seguida; esconder depois do fechamento esconderia para sempre, que não é a
  regra. Como este endpoint só devolve turnos **fechados** (`FindMatchHistory` lê da tabela
  que `PersistTurnClose` grava, o único caminho de escrita), essa condição nunca se aplica
  aqui: **`feint`, quando presente na action, chega a todo viewer deste endpoint**, não só a
  dono/mestre. Quando presente, `feint` é o `RollCheck` da finta. Ver
  [`match-combat-ws.md`](match-combat-ws.md), onde a mesma regra é descrita pelo lado do
  WebSocket (que nunca expõe `feint` — não há mensagem servidor→cliente que projete a
  declaração de uma action).
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
  (`RollCheckResponse.result`, os totais em `resolution`).
- `systemBias` **não tem equivalente no WebSocket**, e não por política: nenhuma mensagem
  servidor→cliente projeta a declaração de uma `action.Action` (`ActionPayload` só existe no
  sentido cliente→servidor). O argumento do "já é dedutível" também não valeria lá —
  `resolution_updated` emite só `diceRolled`, o conjunto efetivamente lido. Ver
  [`match-combat-ws.md`](match-combat-ws.md).
- `RollCheckResponse.attempts` (`primary` e, quando existir, `secondary`) vai para **todo**
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

### A resposta já vem projetada — não filtre no cliente

**Este é o ponto central deste endpoint.** O Action History é uma superfície de jogo com
visibilidade por campo, não um log — a mesma política que `resolution_updated` já aplica no
WebSocket (ver `docs/dev/match/combat-engine.md#visibilidade`), rodada aqui pelas MESMAS
funções (`service.ProjectAction`, `service.ProjectResolution`).

Isso significa, na prática:

- **O mesmo turno retorna JSON diferente para viewers diferentes.** O mestre vê tudo; o
  dono de uma action ou reaction vê tudo que é seu; qualquer outro participante vê tudo
  **menos** a deny-list. Não existe uma "verdade única" que o front possa cachear e
  reutilizar entre usuários.
- **O alvo de um ataque não é uma classe privilegiada.** Uma finta contra você não avisa
  que era finta **enquanto o turno ainda está aberto**. A regra que esconde `feint` de quem
  não é dono/mestre é **temporal** (`isSettled`), não de classe — e como este endpoint só
  devolve turnos já fechados, `feint`, quando presente, chega **a todo viewer** deste
  endpoint. Ver a nota sobre `feint` mais acima.
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
| 200 | Histórico retornado (pode ser `{ "scenes": [] }` para uma partida sem turnos fechados) |
| 400 | UUID inválido |
| 401 | Sem JWT |
| 403 | Partida privada e usuário não é mestre nem participante |
| 404 | Partida não encontrada |
| 500 | Erro interno |
