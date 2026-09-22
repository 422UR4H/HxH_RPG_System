# Match NPCs API

Rostering de NPCs por partida: o mestre põe e tira fichas de NPC (`playerUuid == null`,
`masterUuid` preenchido) da lista de participantes (`match_participants`), sem depender de
inscrição/aceite como um jogador.

## POST /matches/{uuid}/npcs — Adicionar NPC à partida

**Auth:** JWT (apenas o mestre da partida)

### Request

```json
{
  "characterSheetUuid": "uuid-v4"
}
```

| Campo | Regra |
|---|---|
| `uuid` (path) | UUID da partida |
| `characterSheetUuid` | obrigatório; deve ser uma ficha NPC (`playerUuid == null`) pertencente ao mestre da partida **ou** à campanha da partida |

### Response 201

```json
{
  "participant": {
    "uuid": "...",
    "matchUuid": "...",
    "characterSheetUuid": "...",
    "joinedAt": "2026-09-20T19:30:00Z"
  }
}
```

### Respostas

| Status | Situação |
|---|---|
| 201 | NPC adicionado ao roster, retorna `{ "participant": MatchNPCResponse }` |
| 400 | Body malformado ou `uuid` de partida inválido |
| 401 | Sem JWT |
| 403 | Usuário autenticado não é o mestre da partida (`ErrNotMatchMaster`) |
| 404 | Partida não encontrada (`ErrMatchNotFound`) **ou** ficha de personagem não encontrada (`ErrCharacterSheetNotFound`) |
| 422 | `characterSheetUuid` ausente/schema inválido, **ou** validação de domínio: ficha não é NPC (`ErrSheetNotNPC`), ficha não pertence ao mestre nem à campanha (`ErrSheetNotOwnedByMaster`), partida já encerrada (`ErrMatchAlreadyFinished`), NPC já está na partida (`ErrNPCAlreadyInMatch`) |
| 500 | Erro interno |

A autorização do requisitante é checada **antes** de ler a ficha: se o usuário não é o
mestre da partida, a resposta é 403 mesmo que a ficha informada não exista — o endpoint não
vaza se a ficha de outra pessoa existe ou não.

---

## DELETE /matches/{uuid}/npcs/{sheet_uuid} — Remover NPC da partida

**Auth:** JWT (apenas o mestre da partida)

### Request

Sem body. Path params apenas.

| Campo | Regra |
|---|---|
| `uuid` (path) | UUID da partida |
| `sheet_uuid` (path) | UUID da ficha de personagem do NPC a remover |

### Response

`204 No Content`, sem body.

### Respostas

| Status | Situação |
|---|---|
| 204 | NPC removido do roster |
| 400 | `uuid` de partida ou `sheet_uuid` inválido |
| 401 | Sem JWT |
| 403 | Usuário autenticado não é o mestre da partida (`ErrNotMatchMaster`) |
| 404 | Partida não encontrada (`ErrMatchNotFound`) **ou** NPC não está na partida (`ErrNPCNotInMatch`) |
| 500 | Erro interno |

Ao contrário do POST, o handler de remoção **não** tem um caso 422: `RemoveMatchNPCUC` só
devolve `ErrMatchNotFound`, `ErrNotMatchMaster` e `ErrNPCNotInMatch`, então o `switch` do
handler foi enxugado para mapear só esses três — não existe branch 422 vivo aqui, mesmo que
a rota declare 422 na lista de erros do Huma. Também não há guarda de partida já encerrada:
tirar um NPC de uma partida encerrada é inofensivo e recusar isso só atrapalharia limpeza.

---

## O que este endpoint não faz

Este POST sozinho não coloca o NPC numa partida **já em andamento**. A `MatchSession` — o
estado vivo de combate, com `CharacterStatus`, barras e fog — mora na memória do processo do
game server (`cmd/game`, :8081); este endpoint REST roda no `cmd/api` (:5000), um processo
separado, sem pool nem memória compartilhada com o game server. Sozinho, o POST só grava
`match_participants`; o NPC entra na sessão viva quando a sala renasce — `Room.StartMatch` ou
o caminho de rehidratação — e `InitMatchSessionUC` recarrega o roster do zero, trazendo o NPC
junto com seu `CharacterStatus` e as duas barras.

**O caminho para o meio da partida é o verbo de WS `add_npc`** (ver
[`match-combat-ws.md`](match-combat-ws.md) §4), enviado pelo mestre direto ao game server.
Ele roda o MESMO `AddMatchNPCUC` deste endpoint e, havendo sessão viva, injeta a ficha nela em
seguida. Um NPC posto por este POST enquanto a partida já está em andamento fica só no banco
até o mestre reenviar `add_npc` pelo WS: o verbo tolera a "duplicata" que o banco reportaria
(o NPC já está em `match_participants`, `ErrNPCAlreadyInMatch`) e sincroniza a sessão mesmo
assim — é assim que o REST-no-meio-da-partida se resolve.

**Remoção continua só-na-próxima-sala.** O `DELETE` abaixo não tem par ao vivo: tirar um NPC
de uma sessão em andamento esbarra em regras de combate ainda não decididas (ação dele na
fila, turno aberto com ele como ator/alvo, reação pendente) e fica registrado como lacuna em
[`match-combat-ws.md`](match-combat-ws.md) §9.
