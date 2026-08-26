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
                    }
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
- `feint` e `trigger` são omitidos por completo quando o viewer não é dono nem mestre (ver
  abaixo); quando presentes, `feint` é o `RollCheck` da finta e `trigger` é um objeto vazio
  (o domínio ainda não tem campos em `action.Trigger`).
- `reactToId` só aparece em uma reaction (uma action raiz não reage a nada).
- `RollCheck.Context` (que carrega `RollCondition`, a vantagem/desvantagem que o mestre aplicou
  via `edit_action`) e `Action.SystemBias` (o bias que o próprio motor derivou) são detalhe
  interno do motor e não aparecem em superfície nenhuma — nem aqui, nem no WebSocket. O que o
  cliente vê são os números já resolvidos (`RollCheckResponse.result`, os totais em
  `resolution`); a condição ou o bias que os produziu não têm campo de saída.

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
  que era finta — só o dono da finta e o mestre veem `feint` não-nulo.
- **A esquiva fechada chega a terceiros indistinguível de uma esquiva comum.**
  `reactionKind: "closedDodge"` vira `"dodge"` (e `"closedEscape"` vira `"escape"`) para
  quem não é dono nem mestre — o rótulo é o vazamento; ver a nota em
  `internal/domain/match/service/projection.go`. A entrada de `Evasion` em `skills`
  desaparece junto, pela mesma razão.
- **Os números continuam públicos.** Dano, dados rolados, totais — nada disso é escondido,
  porque a dedução ("o adversário deduz dos números") depende deles estarem lá.

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
