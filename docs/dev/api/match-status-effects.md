# Match / Efeitos de status — Protocolo WebSocket

**Status:** implementado (back, PR 1). Contrato de handoff para o front. Cada nome, campo e código
abaixo foi conferido contra `internal/app/game/message.go`, `status_effects.go`, `sight.go`,
`internal/domain/statuseffect/` e os testes e2e (`status_effects_e2e_test.go`,
`sight_e2e_test.go`, `status_block_e2e_test.go`). **Se achar divergência, o bug é do contrato**:
corrija este arquivo.

Complementa [`match-combat-ws.md`](match-combat-ws.md) (envelope, `error`, `match_full_state`,
`action_queued`, `npc_added`). O desenho e o porquê estão em
[`../match/status-effects.md`](../match/status-effects.md); o spec, em
`docs/superpowers/specs/2026-11-15-status-effects-design.pt-br.md`.

**Servidor:** `cmd/game/` · mesmo socket e mesmo envelope de `match-combat-ws.md` §1.

---

## 1. O que é um efeito de status, para o front

Um **efeito** é uma condição que o mestre põe num personagem da partida (PJ ou NPC): Paralisado,
Preso, Escondido… Cada efeito tem um `id`, um `kind` (tipo), parâmetros próprios do tipo, um
apelido opcional e uma **lista de quem o vê**. Nesta fase **só o mestre** aplica, edita e remove
(não há teste automático nem origem automática — ver [§10](#10-o-que-ainda-não-existe)).

O que o servidor **faz** com um efeito já nesta fase:

- **Bloqueia**: o personagem não pode declarar certas ações nem certas reações
  (`blocked_by_status`, [§6](#6-o-que-um-efeito-bloqueia)); a fila e as reações pendentes dele
  são refeitas quando o efeito entra.
- **Avisa o mestre** (`statusNotes` em `action_queued`): domínio inutilizado e conduta imposta.
- **Esconde** (`hidden`): o personagem some do mapa, das barras e dos status de quem não o
  percebeu ([§7](#7-quem-vê-o-quê-lista-e-visão)).

O que **não** faz: nada que mude uma rolagem (penalidade, desvantagem, falha automática), nenhum
dano por round, nenhuma recuperação. Os parâmetros dessas peças são guardados e mostrados, mas
ainda não pesam. Lista completa em [§10](#10-o-que-ainda-não-existe).

## 2. Índice

**Cliente → servidor — todas são só do mestre** (`forbidden` para os demais)

| Mensagem | O que faz |
|---|---|
| [`apply_status_effect`](#apply_status_effect) | aplica um efeito a um ou mais personagens (tudo ou nada) |
| [`edit_status_effect`](#edit_status_effect) | troca parâmetros, apelido e/ou lista de quem vê de um efeito |
| [`remove_status_effect`](#remove_status_effect) | tira um efeito (aplicado no mesmo turno aberto: desfaz) |
| [`relink_status_effect`](#relink_status_effect) | troca a âncora (`actionId`) das mudanças de um efeito no turno aberto |

**Servidor → cliente**

| Mensagem | Destino |
|---|---|
| [`status_effects_updated`](#8-status_effects_updated) | **por destinatário**: o mestre sempre; cada jogador só dos personagens que enxerga e só do que a lista dele inclui |
| `match_full_state.statusEffects` | quem conecta/reconecta — ver [§9](#9-reconexão) |
| [`action_queued.statusNotes`](#statusnotes-em-action_queued) | só o mestre |
| [`action_cancelled`](#action_cancelled-disparado-por-status) | o mestre + o dono da ação, quando um status derruba uma ação da fila |
| [`error` `blocked_by_status`](#blocked_by_status) | quem tentou agir ou reagir |

Também mudaram (detalhe em `match-combat-ws.md`): `npc_added` agora vai **só ao mestre**;
`bars_updated` e `match_full_state.bars` agora são **por jogador** (só os personagens que ele
enxerga); `resolution_updated.pendingReactions[].kind` pode ser `"nothing"` com `seated: true`.

Todo comando do mestre **sem sessão viva** (partida em lobby) responde `match_not_started`.

## 3. Valores (enums)

Todos os valores do fio são camelCase em inglês. **Os nomes em português são do front**; o
servidor nunca os manda.

### `kind`

| `kind` | Nome na mesa (PT) | Padrão de quem vê (sem lista) |
|---|---|---|
| `paralyzed` | Paralisado | escondido |
| `stupefied` | Atônito | escondido |
| `stunned` | Atordoado | **público** |
| `unconscious` | Inconsciente | **público** |
| `restrained` | Preso | **público** |
| `injured` | Lesionado | escondido |
| `slowed` | Lento | escondido |
| `sense` | Sentido afetado (ver `sense`) | escondido |
| `weakened` | Debilitado (ver `aptitude`) | escondido |
| `bleeding` | Sangrando | escondido |
| `burning` | Queimando | **público** |
| `freezing` | Congelando | escondido |
| `poisoned` | Envenenado | escondido |
| `asphyxiating` | Asfixiando | **público** |
| `prone` | Caído | **público** |
| `airborne` | No ar | **público** |
| `openGuard` | Guarda aberta | **público** |
| `compelled` | Compelido | escondido |
| `controlled` | Controlado | escondido |
| `hidden` | Escondido | escondido (é a presença do personagem — ver [§7](#hidden-escondido)) |

### Variantes

| Campo | Valores | Nomes na mesa (PT) |
|---|---|---|
| `sense` (só com `kind: "sense"`) | `vision` · `hearing` · `smell` · `tact` · `taste` | Visão · Audição · Olfato · Tato · Paladar |
| `aptitude` (só com `kind: "weakened"`) | `physical` · `mental` · `spiritual` | **Corpo debilitado** · **Mente debilitada** · **Espírito debilitado** |
| `params.level` (sentido) | `impaired` · `nullified` | Visão **prejudicada** / **anulada** (e o mesmo para os outros sentidos: "Audição prejudicada", "Olfato anulado"…) |
| `params.faces[]` (Tato) | `touch` · `balance` · `preemption` | toque · equilíbrio · preempção |
| `domains[]`, `injuries` (chaves) | `attack` · `defense` · `movement` · `dodge` | ataque · defesa · movimento · esquiva |
| `injuries` (valores) | `hurt` · `disabled` | machucado (desvantagem) · inutilizado |

`kind` e variante são a **identidade** do efeito: um personagem tem no máximo um efeito por
`(kind, variante)` — um por sentido, um por aptidão. Aplicar de novo o que ele já tem é recusado
(`status_effect_exists`); o mestre edita o existente.

### Sugestões de apelido (copy do front)

O apelido (`alias`) é texto livre, até 60 caracteres, mostrado à mesa junto do status. Tem peso
narrativo, não mecânico: o servidor não o interpreta. Sugestões por status (spec §8), para o
front oferecer:

> Visão — Cego, Ofuscado · Audição — Surdo, Zumbido · Tato
> (equilíbrio) — Tonto, Vertigem · Tato (toque) — Dormente · Corpo debilitado — Enjoado, Febril,
> Fraco · Mente debilitada — Abalado, Confuso · Espírito debilitado — Aura abalada, Desanimado · Paralisado — Em choque,
> Travado · Atônito — Em pânico, Horrorizado · Preso — Agarrado, Enredado, Soterrado, Pendurado ·
> Compelido — Amedrontado, Hipnotizado, Em fúria (sem "Enfeitiçado": não há feitiço no mundo
> hunter) · Escondido — Furtivo, Emboscado · Guarda aberta — Desprevenido, Surpreso ·
> Queimando — Em chamas · Asfixiando — Afogando, Estrangulado · Lesionado — livre ("braço quebrado").

Não há endpoint de catálogo: nomes em português e sugestões vivem no front.

## 4. Parâmetros por tipo

`params` é um objeto; **só os campos do tipo entram**. Um campo que o tipo não tem, mesmo com
valor neutro, é recusado (`status_effect_invalid`). Na resposta (`status_effects_updated`,
`match_full_state`) o servidor devolve só os campos do tipo, **com os padrões já aplicados**.

| `kind` | Campos de `params` | Obrigatório? | Padrão / regra |
|---|---|---|---|
| `paralyzed` | `domains` (lista), `points` (int) | `points` sim | `domains` vazio/ausente = **os quatro**; subconjunto permitido; sem repetição; sai sempre na ordem `attack, defense, movement, dodge`. `points` ≥ 1. |
| `stupefied` | `points` | sim | ≥ 1 |
| `stunned` | `points` | sim | ≥ 1 |
| `injured` | `injuries` (mapa `domínio → hurt\|disabled`) | sim | ao menos um domínio. `disabled` também carrega a desvantagem de `hurt`. |
| `slowed` | `penalty` (int) | sim | ≥ 1 |
| `sense` | `level`, `disadvantage`, `penalty`, `faces` | `level` sim; `faces` só no Tato | veja abaixo |
| `weakened` | `disadvantage`, `penalty` | — | veja abaixo |
| `bleeding` `burning` `freezing` `poisoned` | `hpPerRound` (int) | não | padrão **1**; ≥ 1 |
| `compelled` | `conduct` (texto), `irresistible` (bool) | `conduct` sim | `conduct` 1–255 caracteres depois de aparado; `irresistible` padrão `false` |
| `controlled` | `controllerId` (uuid) | não | se vier: personagem **da sessão** e **diferente do alvo** |
| `unconscious` `restrained` `prone` `airborne` `openGuard` `asphyxiating` `hidden` | — (nenhum) | — | enviar `params: {}`; qualquer campo é recusado |

**`sense`.** O envio exige `sense` no corpo do comando (não em `params`).

- `level: "impaired"` (prejudicado): `disadvantage` (padrão `true`) e `penalty` (≥ 0, padrão 0).
  Ao menos um dos dois precisa valer algo: `disadvantage: false` com `penalty` 0/ausente é
  recusado.
- `level: "nullified"` (anulado): **nenhum dos dois pode ser enviado — nem `false`, nem `0`**.
  Um sentido anulado recusa `disadvantage`/`penalty` mesmo quando mandados "neutros": omita-os.
  Na saída também não vêm.
- `faces`: **só com `sense: "tact"`**, obrigatório, subconjunto não vazio de
  `touch`/`balance`/`preemption`, sem repetição, sai na ordem `touch, balance, preemption`. Em
  outro sentido, `faces` é recusado.

**`weakened`.** O envio exige `aptitude`. `disadvantage` (padrão `true`) e `penalty` (≥ 0): ao
menos um dos dois precisa valer algo. Não existe "debilitado anulado".

**Sobre `controlled`:** `controllerId` aponta um personagem (de jogador ou NPC); a UI para
escolher quem controla é do front.

**Sobre os padrões de `params` na entrada de edição:** `edit_status_effect.params` **substitui o
conjunto inteiro** (não é um merge) e é normalizado do mesmo jeito: reenvie os campos
obrigatórios.

## 5. Cliente → servidor

### `apply_status_effect`

**Quem:** só o mestre. Aplica **o mesmo efeito** a um ou mais personagens. **Tudo ou nada**: se
qualquer personagem recusa, nenhum recebe.

```json
{
  "type": "apply_status_effect",
  "payload": {
    "characterIds": ["22222222-2222-4222-8222-222222222222"],
    "kind": "paralyzed",
    "params": { "domains": ["movement", "dodge"], "points": 5 },
    "alias": "Em choque",
    "visibility": { "mode": "players", "playerIds": ["99999999-9999-4999-8999-999999999999"] }
  }
}
```

| Campo | Tipo | Obrigatório | Notas |
|---|---|---|---|
| `characterIds` | uuid[] | sim | ids de **ficha** (`CharacterID`, o mesmo das peças e de `bars.characters`). Não vazio, sem repetição, sem uuid nulo. |
| `kind` | string | sim | [§3](#kind). Desconhecido → `status_effect_invalid`. |
| `sense` | string | só em `sense` | [§3](#variantes). Em outro `kind`, recusado. |
| `aptitude` | string | só em `weakened` | idem |
| `params` | objeto | sim (pode ser `{}`) | [§4](#4-parâmetros-por-tipo) |
| `alias` | string | não | até 60 caracteres depois de aparado; `""` = sem apelido |
| `visibility` | objeto | não | ausente = padrão do tipo. Ver [§7](#visibilidade-lista-de-quem-vê). |
| `actionId` | uuid | não | a action a que esta mudança pertence, no histórico: a **ação do turno aberto** ou **uma reação dele** (anexada, aberta ou sentada). Ausente = do turno (ou do round) como um todo. Sem turno aberto, ou um id que não é dele → `status_effect_invalid`. Os três comandos aceitam o mesmo campo. |

**Efeito colateral — Inconsciente traz Caído.** Aplicar `unconscious` num personagem que ainda
não tem `prone` aplica também um `prone` (parâmetros, apelido e lista padrão). O mestre pode
remover o Caído e manter só o Inconsciente (remover um não remove o outro). Os dois chegam em
`status_effects_updated`; nenhuma mensagem separada anuncia o segundo.

**O que mais acontece** ([§6](#6-o-que-um-efeito-bloqueia)): cada ação pendente do personagem que
o conjunto novo recusa sai da fila (`action_cancelled`); uma reação **não aberta** que ele não
pode mais dar é refeita.

**Dispara:** [`status_effects_updated`](#8-status_effects_updated) para quem vê.

**Erros:**

| `code` | Quando |
|---|---|
| `forbidden` | quem enviou não é o mestre |
| `match_not_started` | lobby, sem sessão |
| `match_already_ended` | partida encerrada |
| `invalid_payload` | JSON do `payload` malformado; `characterIds` vazio, com id nulo ou repetido; `sense`, `aptitude`, `level`, `faces[]`, `domains[]`, `injuries` (domínio ou nível) desconhecidos; `visibility.mode` desconhecido; **`visibility.playerIds` com `mode` diferente de `"players"`** |
| `unknown_character` | um id não é ficha da sessão |
| `status_effect_invalid` | `kind` desconhecido; parâmetro ausente, fora da faixa ou **estranho ao tipo**; `sense` anulado com `disadvantage`/`penalty` (mesmo `false`/`0`); `controllerId` igual ao alvo ou fora da partida; `alias` longo; `visibility.playerIds` com o mestre, alguém fora da partida, repetido ou uuid nulo; `actionId` sem turno aberto ou fora dele |
| `status_effect_exists` | um dos personagens já tem o `(kind, variante)` |
| `game_error` | `"status effects are not available"` (servidor sem o caso de uso) e qualquer outra falha |

### `edit_status_effect`

**Quem:** só o mestre. Edita um efeito **existente**; o `kind`, o `sense` e o `aptitude` não
mudam (remova e aplique outro).

```json
{
  "type": "edit_status_effect",
  "payload": {
    "statusEffectId": "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
    "params": { "domains": ["movement", "dodge"], "points": 3 },
    "alias": "",
    "visibility": { "mode": "default" }
  }
}
```

| Campo | Notas |
|---|---|
| `statusEffectId` | obrigatório (uuid não nulo; o servidor acha o personagem sozinho) |
| `params` | **ausente = intocado; presente = substitui todos** (normalizado como na aplicação: reenvie o que é obrigatório). ⚠️ Campos com padrão voltam ao padrão se omitidos: um `paralyzed` editado só com `points` volta a travar os **quatro** domínios; por isso o exemplo reenvia `domains`. |
| `alias` | ausente = intocado; `""` limpa |
| `visibility` | ausente = intocada; `{"mode":"default"}` volta ao padrão do tipo |
| `actionId` | opcional; mesma regra de `apply_status_effect` |

**Dispara:** `status_effects_updated` (lista inteira nova) para quem vê. Editar a lista de um
`hidden` re-avalia a peça, as barras e os status ([§7](#hidden-escondido)). Edição também refaz
a fila e as reações pendentes pelas mesmas regras da aplicação ([§6](#6-o-que-um-efeito-bloqueia)).

**Erros:** os de `apply_status_effect` que se aplicam, mais `status_effect_not_found` (nenhum
efeito com esse id).

### `remove_status_effect`

```json
{ "type": "remove_status_effect", "payload": { "statusEffectId": "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa" } }
```

`actionId` é opcional, com a mesma regra de `apply_status_effect`. `restoreCancelledActions`
(bool, opcional, ausente = `false`) pede que as ações da fila que o efeito cancelou voltem — só
vale no desfazer no turno (abaixo).

```json
{ "type": "remove_status_effect", "payload": { "statusEffectId": "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "restoreCancelledActions": true } }
```

**Dispara:** `status_effects_updated` com a lista sem o efeito (vazia se era o único). Remover
nunca recusa nada novo; reações sentadas como `nothing` por causa do efeito **voltam à padrão
inteira**.

**Desfazer no turno.** Remover, com o turno aberto, um efeito **aplicado nesse mesmo turno**
desfaz a aplicação: nada desse efeito é gravado no fechamento (nem a aplicação, nem edições, nem
evento `removed` no histórico). Ao vivo é uma remoção como as outras. As ações da fila que o
efeito já derrubou **continuam canceladas**, a não ser que o mestre peça a volta delas.
Qualquer outra remoção (efeito de antes do turno, ou sem turno aberto) é uma remoção normal,
gravada (com o turno aberto, no fechamento dele). Depois do fechamento não há volta.

**Devolver as ações (`restoreCancelledActions: true`).** Só no desfazer no turno: as ações que
**este** efeito tirou da fila neste turno (as que o mestre vê em `cancelledActionIds`, [§8](#8-status_effects_updated))
voltam **para o mesmo lugar da fila** (a ordem de chegada delas, que desempata a fila: empates
incluídos, contra as que chegaram antes e depois — também depois do cancelamento), com o **mesmo
id**, os mesmos dados já rolados e as **edições do mestre** que o cancelamento tinha descartado
(condições, velocidade editada, `damageSkill`). O servidor **revalida** cada uma contra o estado
de agora e recusa, com motivo, a que não cabe mais: outro status ainda a recusa; uma ação com
esse id já está na fila; o personagem saiu da partida; a cadeia de movimentos não está como
estava no cancelamento — um movimento que se apoiava numa base ainda na fila volta só se essa
base continua na fila e é o fim da cadeia; um que era cabeça (de fato, ou porque a base já
tinha saído da fila antes do status, aberta ou consumida) volta só se o personagem não tem outro
movimento pendente. Nunca volta meia cadeia. Uma devolvida some do registro de canceladas:
**nada dela é gravado** em `cancelled_actions` no fechamento.

Quando a recusa é porque **outro status aplicado neste mesmo turno** ainda a recusa, a ação passa
para o `cancelledActionIds` desse outro efeito: desfazer os dois a devolve, em qualquer ordem.
Recusada por um status de antes do turno, ela fica cancelada; a mensagem nomeia o tipo que
recusa.

Fora da janela do desfazer (efeito de antes do turno, sem turno aberto, depois do fechamento) a
flag é **ignorada** sem erro: não há o que devolver. Sem a flag, as ações continuam canceladas e
o registro de devolução do efeito é descartado.

**Dispara (devolvidas)**, por ação, na ordem da fila: [`action_restored`](#action_restored) ao
**mestre** e ao **dono** (ação de NPC: só ao mestre), e, ao mestre, logo depois, o `action_queued`
de sempre (a entrada na fila, com `statusNotes`). Depois, `bars_updated` e, se o personagem tem
reação no turno aberto, `resolution_updated` ao mestre. **Não** vem `action_enqueued`: ele é o
ack de um envio do cliente, casado pela ordem dos envios, e um ack sem pedido cairia no envio
errado.

### `action_restored`

**Direção:** servidor → cliente. **Destino:** o mestre e o dono da ação (NPC: só o mestre).

```json
{ "type": "action_restored", "payload": {
  "actionId": "33333333-3333-4333-8333-333333333333",
  "action": { "uuid": "33333333-3333-4333-8333-333333333333", "actorId": "…", "targetId": ["…"], "speed": { "bar": 0, "rollCheck": { "skillName": "Legerity" } }, "attack": { "weapon": "Sword", "hit": { "skillName": "Accuracy" } } },
  "beforeActionId": "44444444-4444-4444-8444-444444444444"
} }
```

| Campo | |
|---|---|
| `actionId` | o **mesmo** id de antes do cancelamento (o do `action_cancelled`) |
| `action` | a ação como o destinatário a vê: o mestre em `Full` (como `action_queued.action`), o dono no corte do `ownQueue` (`Declaration`: sem dados nem velocidades) |
| `beforeActionId` | omitido = ela vai para o fim; senão, a ação **que o destinatário já conhece na fila** (o mestre: qualquer uma; o dono: as dele) diante da qual ela voltou — o lugar dela, que desempata a fila. As outras devolvidas do mesmo comando não contam (ainda não chegaram) |

As devolvidas de um comando chegam **na ordem da fila**; o cliente põe cada uma de volta na
fila/`ownQueue` diante de `beforeActionId` (ou no fim), uma a uma, e todas caem no lugar. Para o mestre, o `action_queued`
que vem logo depois tem o mesmo `actionId` e só a redescreve (substitui no lugar).

**Dispara (recusadas)**, por ação, **só ao mestre**: `error` com `code: "restore_refused"` e a
mensagem `Não foi possível devolver a ação: <motivo>` (ex.: `Não foi possível devolver a ação:
Gon não pode agir: stunned`). A recusada continua cancelada (gravada no fechamento).

**Erros:** `forbidden` · `match_not_started` · `match_already_ended` · `invalid_payload`
(`statusEffectId` ausente ou nulo) · `status_effect_not_found` · `status_effect_invalid` (`actionId`
sem turno aberto ou fora dele).

### `relink_status_effect`

```json
{ "type": "relink_status_effect", "payload": { "statusEffectId": "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "actionId": "33333333-3333-4333-8333-333333333333" } }
```

Só o mestre, só com turno aberto. Troca a âncora de **toda** mudança desse efeito ainda pendente
no turno aberto: `actionId` = a ação do turno ou uma reação dele; `null` (ou ausente) = a
mudança é do turno, sem âncora. Depois do fechamento o histórico é definitivo.

A âncora também **segue sozinha**: quando a reação sentada de um personagem é substituída pela
que ele mandou (ou refeita por um status), o que estava preso à sentada passa para a nova; quando
a sentada sai sem substituta (o mestre trocou os alvos), passa para a ação do turno.

**Dispara:** `status_effects_updated` do personagem **ao mestre**, com o `turnActionId` novo
(ver [§8](#8-status_effects_updated)) — sempre, mesmo sem mudança.

**Erros:** `forbidden` · `match_not_started` · `invalid_payload` (`statusEffectId` ausente ou
nulo) · `status_effect_invalid` (sem turno aberto; `actionId` que não é do turno aberto) ·
`status_effect_not_found` (o efeito não tem mudança neste turno).

## 6. O que um efeito bloqueia

Um efeito é composto de peças. Nesta fase valem estas (as demais só são guardadas — [§10](#10-o-que-ainda-não-existe)):

| Peça | Vem de | Efeito |
|---|---|---|
| não age | `stupefied`, `stunned`, `unconscious` | recusa toda ação não-reação |
| não reage | `stupefied`, `stunned`, `unconscious` | recusa toda reação, menos `nothing` |
| domínio inutilizado | `paralyzed` (os `domains`), `stunned`/`unconscious` (os quatro) | recusa ação/reação que usa o domínio |
| sem deslocamento | `restrained`, `paralyzed` com `movement`, `stunned`/`unconscious` | recusa a ação de mover e as fugas |
| reação proibida | `restrained` (escape); `airborne` (esquiva por reflexo, escape); `openGuard` (esquiva por reflexo, aparo) | recusa a reação |
| aviso de domínio | `injured` com `disabled` | **não bloqueia**; vira `statusNotes` |
| conduta | `compelled` | **não bloqueia**; vira `statusNotes` |

`controlled`, `slowed`, `sense`, `weakened`, `hidden` e os de dano por round **não bloqueiam
nada** aqui.

**Ações** (`enqueue_action`): componentes `attack`/`feint` usam o domínio `attack`; `defense`,
`defense`; `dodge`, `dodge`; `move`, `movement`. `interact` e ação só de perícia só são
recusados por "não age".

**Reações** (`attach_reaction`, ou `enqueue_action` com `reactToId`), por `reactionKind`:

| Tipo | Precisa livre | Observação |
|---|---|---|
| `nothing` | — | **sempre permitida** |
| `default` | esquiva por reflexo (domínio esquiva) **ou** defesa padrão (domínio defesa) | recusada só se **as duas** estão fora (ou "não reage"); com uma fora é aceita e a outra roda sozinha |
| `dodge` | esquiva por reflexo | |
| `closedDodge` | esquiva por reflexo + evasão | |
| `escape`, `escapeGuard` | movimento **e** esquiva; sem "sem deslocamento" | |
| `closedEscape` | movimento e esquiva + evasão; sem "sem deslocamento" | |
| `repel` | aparo (domínio defesa) | |

Quando a defesa está inutilizada ou proibida, uma reação `dodge`/`closedDodge`/`escapeGuard`
também perde a defesa padrão de reserva. Isso não aparece no fio.

### `blocked_by_status`

`error` com `code: "blocked_by_status"` (envelope de `match-combat-ws.md` §1 / `error`), só para
quem enviou. Não muda nada na fila, nas barras nem nos dados.

```json
{ "type": "error", "payload": { "code": "blocked_by_status", "message": "Combatant não pode usar o domínio ataque: paralyzed" } }
```

`message` tem a forma `<nick> não pode <o que>: <causa>`. "O que" é um de `agir`,
`reagir`, `reagir com a reação padrão`, `se deslocar`, `usar o domínio <ataque|defesa|movimento|esquiva>`,
`usar <a esquiva por reflexo|a evasão|a defesa padrão|repelir|escapar>`. **A causa nomeia só os
`kind`s que o remetente pode ver**: o mestre vê todos; um jogador, os que a lista dele inclui;
se nenhum é visível a ele, a causa é `um efeito` — mas o "o que" continua dito (ex.:
`... não pode usar o domínio ataque: um efeito`). Não parseie a mensagem; mostre-a.

Sai de `enqueue_action`, `attach_reaction` e `enqueue_action` com `reactToId`.

### O que o efeito refaz nas pendências

Ao aplicar ou editar (e, para as sentadas, ao remover), na mesma operação:

1. Cada ação **pendente** do personagem que o conjunto novo recusa sai da fila, junto dos
   [movimentos empilhados](match-combat-ws.md#movimentos-empilhados) sobre ela. Cada uma é gravada como cancelada **pelo mestre**
   (mesma tabela e regra do `cancel_action`: com turno aberto, no fechamento dele).
2. Uma reação **ainda não aberta** do personagem no turno aberto que ele não pode mais dar é
   trocada pela reação padrão sentada (só as passivas permitidas) ou por `nothing` (quando nenhuma
   passiva resta ou ele não pode reagir). O que já foi **aberto** fica.
3. A reação trocada **não manda mensagem extra ao dono dela**: se o efeito é visível a ele,
   ele o ganha por `status_effects_updated`; se é escondido, **não é anunciado de propósito**
   (spec §9: o status privado pesa sem o jogador saber por quê).
4. O **mestre** recebe um `resolution_updated` novo (as reações pendentes mudaram) —
   `pendingReactions[].kind` pode agora ser `"nothing"`, com `seated: true`. `pendingReactions`
   continua **master-only**, mesmo num payload liquidado.

### `action_cancelled` disparado por status

Mesma mensagem do `cancel_action` (`{ "actionId": … }`), **um por ação** derrubada. Vai ao **mestre
e ao dono da ação** (ação de NPC: só ao mestre), seguido de `bars_updated`. Chegando, o cliente
tira a ação de `ownQueue`/fila como em qualquer cancelamento.

```json
{ "type": "action_cancelled", "payload": { "actionId": "33333333-3333-4333-8333-333333333333" } }
```

### `statusNotes` em `action_queued`

`action_queued` (e a mesma entrada em `match_full_state.queue`) ganha **`statusNotes`**, **só para
o mestre** — jogador nunca recebe a chave. Ausente quando não há nenhuma.

```json
{
  "type": "action_queued",
  "payload": {
    "actionId": "33333333-3333-4333-8333-333333333333",
    "actorId": "11111111-1111-4111-8111-111111111111",
    "bars": ["action"],
    "action": { "uuid": "33333333-3333-4333-8333-333333333333", "actorId": "11111111-1111-4111-8111-111111111111" },
    "statusNotes": [
      { "kind": "injured", "domain": "attack" },
      { "kind": "compelled", "conduct": "fugir do fogo", "irresistible": true }
    ]
  }
}
```

| Campo | Notas |
|---|---|
| `kind` | o tipo que originou a nota |
| `domain` | `attack`/`defense`/`movement`/`dodge`: aviso de **domínio inutilizado** (Lesionado `disabled`). Só vem quando a ação **usa** esse domínio. |
| `conduct`, `irresistible` | conduta imposta (Compelido); **sempre** vem, qualquer que seja a ação. `irresistible` ausente = `false`. |

(`action` abreviado acima; a forma inteira está em `match-combat-ws.md`.)

**`statusNotes` é um retrato do momento em que a ação entrou na fila** (e de `match_full_state`,
que o recalcula na conexão). O servidor **não reenvia** a nota quando um status muda depois: um
`status_effects_updated` posterior não altera o `action_queued` já mostrado. O front deriva as notas
**atuais** dos status (o mestre recebe todos) e usa `statusNotes` só como o valor daquele instante.

## 7. Quem vê o quê: lista e visão

Duas camadas **somadas**: a **lista** (quem o efeito deixa ver) e a **visão** (se o jogador
enxerga o personagem agora, pela névoa).

### Visibilidade (lista de quem vê)

`visibility` (na entrada, de `apply`/`edit`; na saída, **só ao mestre**):

| `mode` | `playerIds` | Significado |
|---|---|---|
| `default` | — | padrão do tipo (a tabela do [§3](#kind)) |
| `public` | — | todos os jogadores |
| `players` | lista de ids de **usuário** de jogadores da partida | só esses; **`[]` = só o mestre** |

`playerIds` com `mode` diferente de `players` é `invalid_payload`. O mestre sempre vê tudo e
não entra em `playerIds` (recusado).

**Regra do padrão** (sem lista, `mode: "default"`): kind **público** → todos. Kind
**escondido** → o mestre e o **dono**, num PJ; **só o mestre**, num NPC. (A configuração de mesa
`PlayerStatusEffectsPublic`, que abriria os escondidos de um PJ a todos, existe no servidor com padrão
`false` e **ainda não há como mudá-la pelo fio**: tratar como sempre `false`.) Quando há lista (`public`/`players`), **o dono só vê se
estiver nela** — o dono não entra implicitamente.

Na saída (só ao mestre), `visibility.playerIds` **ausente com `mode: "players"` = lista vazia (só o mestre)**.

Jogadores **nunca veem a lista**: `visibility` e `public` só chegam ao mestre; não há `updatedAt`
no fio.

### Visão (névoa)

Um jogador só recebe os efeitos de um personagem enquanto **enxerga** esse personagem. O dono e
o mestre sempre. Regras (decisão do dono 4):

- **Partida sem tabuleiro:** não há névoa, todos veem.
- **Com tabuleiro:** o personagem precisa ter peça, e o centro da peça dentro da visão do jogador.
  Personagem **sem peça** num tabuleiro não é visto pelos outros.
- **Ao sair da visão, o servidor não manda nada.** O jogador **fica com o último estado que
  viu** (não existe "deixou de ver"). Mudanças enquanto está fora **não chegam**.
- **Ao voltar à visão**, o servidor manda a lista **atual** (um `status_effects_updated`
  completo), logo depois do `piece_moved` que a trouxe.
- **Um personagem com lista visível vazia que o jogador nunca recebeu** não gera mensagem (não há
  o que apagar).

As barras seguem a mesma visão ([§11](#11-barras-e-npc_added-mudaram)).

### `hidden` (Escondido)

`hidden` tira o **personagem do mapa, das barras e dos status** de quem **não está na lista dele**
(a lista é "quem o percebeu"). Os dois mecanismos se somam; Escondido vale **com ou sem
tabuleiro**. O dono e o mestre nunca ficam escondidos.

Quando o `hidden` entra, sai ou tem a lista trocada, o servidor manda, nesta ordem:

1. a quem perdeu a peça: `piece_removed`; a quem a ganhou: `piece_moved` (mensagem do servidor);
2. `bars_updated` com o personagem entrando/saindo da lista de cada jogador;
3. `status_effects_updated` aos que ganham/perdem.

Enquanto Escondido, o personagem **não aparece** no `map_full_state` nem em
`match_full_state.bars`/`statusEffects` de quem não o percebe, e `piece_moved` dele não chega a
essa pessoa. As **ações** dele durante o Escondido chegam reduzidas (`turn_opened`/`reaction_opened`/`openTurn` com `actorHidden`, alvo cortado; ver [`match-combat-ws.md`](match-combat-ws.md#ator-escondido-a-mensagem-reduzida)) e o histórico respeita o que cada um viu ([§14](#14-histórico)).

### Ordem entre pistas e NPC ao vivo

`npc_added` vai ao mestre pela pista direta (`sendToMaster`) e o `syncSight`, que manda barras e
status, roda logo depois na mesma goroutine e na mesma pista: **o mestre sempre recebe
`npc_added` antes** dos status do NPC inscrito ao vivo (`add_npc`). Ainda assim, guardar os status
por `characterId` é o jeito mais simples de consumir. Já um `status_effects_updated` pode chegar
antes do `match_full_state` mais antigo na reconexão; a próxima mudança acerta.

## 8. `status_effects_updated`

**Direção:** servidor → cliente. **Destino:** por destinatário, só quem enxerga o personagem e só
com a lista que ele vê; o mestre sempre.

```json
{
  "type": "status_effects_updated",
  "payload": {
    "characterId": "22222222-2222-4222-8222-222222222222",
    "statusEffects": [
      {
        "id": "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
        "kind": "paralyzed",
        "params": { "domains": ["movement", "dodge"], "points": 5 },
        "alias": "Em choque",
        "appliedAt": "2026-11-17T14:03:22.123456Z",
        "visibility": { "mode": "players", "playerIds": ["99999999-9999-4999-8999-999999999999"] },
        "public": false
      }
    ]
  }
}
```

**`turnActionId`** (só o mestre, omitido quando não há): a âncora da última mudança desse efeito
ainda pendente no turno aberto — a ação ou a reação a que ela pertence. Chega quando muda
(`apply`/`edit`/`remove` com `actionId`, a âncora que segue a reação, `relink_status_effect`), some
quando o turno fecha (um `status_effects_updated` ao mestre sem ela) e vem no `match_full_state`.
Ordem no fechamento: esse `status_effects_updated` do mestre chega **antes** do
`character_hp_changed` e do `turn_closed`, na mesma pista direta.
Os jogadores nunca a recebem; veem a âncora no histórico, depois do fechamento.

**`cancelledActionIds`** (só o mestre, mesmo corte do `turnActionId`, omitido quando não há): os
ids das ações da fila que **esse** efeito tirou no turno aberto e que um `remove_status_effect`
com `restoreCancelledActions` ainda pode devolver ([§5](#remove_status_effect)). Só aparece num
efeito **aplicado neste turno** (a janela do desfazer); some quando o turno fecha (o mesmo
`status_effects_updated` ao mestre que tira o `turnActionId`) e vem no `match_full_state` do
mestre. É o gatilho do front para perguntar "devolver?" ao remover o efeito. Uma ação que voltou
sai da lista; uma que o desfazer de outro efeito não pôde devolver porque **este** a recusa
entra nela. Os jogadores nunca o recebem.

**É a lista INTEIRA** do personagem como o destinatário a vê: **substitua**, nunca acrescente. **Nunca
`null`**: `[]` = ele não vê mais nenhum. O servidor só manda quando a lista **difere** da última
que ele enviou àquele destinatário.

Visão de um **jogador** (sem `visibility`/`public`):

```json
{
  "type": "status_effects_updated",
  "payload": {
    "characterId": "22222222-2222-4222-8222-222222222222",
    "statusEffects": [
      { "id": "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", "kind": "prone", "params": {}, "appliedAt": "2026-11-17T14:03:22.123456Z" }
    ]
  }
}
```

| Campo | Tipo | Notas |
|---|---|---|
| `characterId` | uuid | ficha |
| `statusEffects[].id` | uuid | o `statusEffectId` de edit/remove |
| `.kind` | string | [§3](#kind) |
| `.sense` / `.aptitude` | string | só em `sense`/`weakened`; ausentes nos demais |
| `.params` | objeto | [§4](#4-parâmetros-por-tipo); sempre presente (`{}` sem parâmetros) |
| `.alias` | string | ausente = sem apelido |
| `.appliedAt` | string | instante **real** RFC 3339 (pode ir para `new Date`) |
| `.visibility` | objeto | **só o mestre**: `{ mode, playerIds? }` |
| `.public` | bool | **só o mestre**: `true` se o efeito está **listado para todo jogador** (modo `public`, padrão do tipo público, ou a regra de mesa); `players: [x]` dá `false`. A visão (névoa) ainda se aplica a cada jogador por cima; é a "visão à parte" para o mestre mostrar um selo "público" |

Exemplo de `params` por tipo na saída:

```json
{ "kind": "sense", "sense": "tact", "params": { "level": "impaired", "disadvantage": true, "penalty": 0, "faces": ["balance"] } }
{ "kind": "sense", "sense": "vision", "params": { "level": "nullified" } }
{ "kind": "weakened", "aptitude": "physical", "params": { "disadvantage": true, "penalty": 2 } }
{ "kind": "injured", "params": { "injuries": { "attack": "disabled", "dodge": "hurt" } } }
{ "kind": "bleeding", "params": { "hpPerRound": 1 } }
{ "kind": "compelled", "params": { "conduct": "fugir do fogo", "irresistible": false } }
{ "kind": "controlled", "params": { "controllerId": "55555555-5555-4555-8555-555555555555" } }
```

Numa edição de lista (de `hidden` para `public`, por exemplo) quem **ganha** o efeito recebe a
lista nova; quem o **perde** recebe a lista sem ele (talvez `[]`).

## 9. Reconexão

`match_full_state` ganha **`statusEffects`**: mapa `characterId → lista` (cada lista no formato
acima, já projetada para este destinatário), **só** dos personagens que ele enxerga agora e que
têm algo visível a ele. Ausente (`omitempty`) quando não há nenhum; **personagem com lista vazia
não entra** (ausente e `[]` querem dizer o mesmo aqui). O servidor guarda o que mandou, então
um `status_effects_updated` igual ao do `match_full_state` não é repetido.

```json
{
  "type": "match_full_state",
  "payload": {
    "roundMode": "Race",
    "statusEffects": {
      "22222222-2222-4222-8222-222222222222": [
        { "id": "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", "kind": "prone", "params": {}, "appliedAt": "2026-11-17T14:03:22.123456Z" }
      ]
    }
  }
}
```

(Abreviado: só o campo novo.) Os efeitos persistem entre sessões e reinícios; ver
[§12](#12-quando-é-gravado).

**Corrida conhecida:** como o `match_full_state` sai por uma pista e o `status_effects_updated`
por outra, uma mudança que acontece exatamente durante a reconexão pode chegar antes do estado
completo; o `statusEffects` mais antigo fica na tela até a próxima mudança do personagem.

## 10. O que ainda não existe

Para o front não procurar. **Nada disto está no fio**:

- **Tudo o que muda uma rolagem**: penalidade nos testes, "defesa só rolada" (`openGuard`),
  desvantagem em reação, falha automática de sentido/Atordoado, Lento, e a aplicação da
  desvantagem do Tato/equilíbrio (G1). Os parâmetros são guardados; **não pesam em dado nenhum**
  ainda (próxima fase, junto da regra nova de vantagem/desvantagem).
- **Testes automáticos** (resistência ao entrar, teste de equilíbrio do Tato, Recuperação).
  Quem aplica é só o mestre, à mão; ele digita os `points`.
- **Dano por round e Recuperação por round** (`hpPerRound`, `points` só são exibidos), HP
  negativo, Inconsciente automático, morte.
- **Asfixia** e Fadiga: `asphyxiating` existe como rótulo, sem mecânica.
- **"Não usa habilidades"** (Atônito/Atordoado/Paralisado): espera o Nen (TODO).
- **Revelar a um jogador só**: a revelação do histórico é **para todos** (ver [§14](#14-histórico)); por jogador é futuro.
- **Push no WebSocket depois de uma revelação**: não existe; o front relê o histórico.
- **Endpoint de catálogo** de efeitos: não existe; nomes e sugestões são do front.
- **Esconder o NPC no REST** de participantes (`GET` de participantes da partida): continua
  listando o NPC; o front precisa lidar com isso até a fase dele.
- **Configuração de mesa** `PlayerStatusEffectsPublic`: sem comando para ligar.
- O servidor não remove o efeito sozinho em nenhum caso (nem por troca de cena, nem por fim de
  round, nem por relógio).

## 11. Barras e `npc_added` mudaram

- **`bars_updated` e `match_full_state.bars` são por jogador.** A lista `characters` de um jogador
  traz só os personagens que **ele é dono ou enxerga** (mesma visão do [§7](#visão-névoa)); o
  mestre recebe todos. Um personagem que sai da visão **some** da lista dele no `bars_updated`
  seguinte; ao voltar, reaparece. O servidor não manda `bars_updated` a quem a vista não mudou.
- **`npc_added` vai só ao mestre.** Um NPC chega aos jogadores como qualquer peça: pela névoa
  (`piece_moved`/`map_full_state`) e, se enxergado, por `bars_updated`. Jogadores nunca recebem
  `npc_added` (nem no lobby).

## 12. Quando é gravado

Os efeitos persistem na tabela `character_status_effects` e voltam quando a sala nasce, ao iniciar a
partida (e, no NPC inscrito ao vivo, ao inscrevê-lo).

- **Fora de turno**: a mudança é gravada **na hora**, antes do `status_effects_updated`.
- **Dentro de um turno aberto**: a mudança vale **ao vivo** (bloqueios, mensagens, tudo) mas só
  fica **durável com o fechamento do turno**, na transação dele. Reiniciar o servidor no meio do
  turno perde o que foi feito nele (a regra da casa: nada dentro do turno é gravado antes de
  fechar).
- Uma gravação que falha **não derruba a mesa**: continua pendente e vai junto da próxima gravação
  que dá certo (próxima mudança de status fora de turno, próximo fechamento de turno ou o
  `end_match`).

Uma ficha viva em duas salas (a mesma ficha em duas partidas ao mesmo tempo) guarda duas
memórias: a última gravação ganha no banco.

**Os eventos do histórico vão nas mesmas gravações.** Cada mudança carrega o seu evento
(`status_effect_events`), carimbado pela sala no instante: fora de turno é gravado **na hora**,
junto do efeito; dentro de um turno, **com o fechamento**, na transação dele; uma gravação que
falha se cura junto com o efeito (o evento volta com os mesmos ids, e uma ficha ou partida
apagada no meio só o descarta, sem falhar o lote). Desfazer no turno (§5, `remove_status_effect`)
não grava evento algum.

## 13. Catálogo de erros desta feature

| `code` | Mensagens | Quando |
|---|---|---|
| `forbidden` | `only the master can perform this action` | os quatro comandos, por jogador |
| `match_not_started` | `match session not initialized` | lobby |
| `match_already_ended` | — | partida encerrada |
| `invalid_payload` | texto do campo | ver [§5](#apply_status_effect) |
| `unknown_character` | — | id de ficha fora da sessão |
| `status_effect_invalid` | texto do erro de domínio | parâmetro/lista inválidos; `actionId` sem turno aberto ou fora dele (`relink` inclusive) |
| `status_effect_exists` | `status effect already applied: <kind>[/<variante>]` | repetição de `(kind, variante)` |
| `status_effect_not_found` | — | edit/remove de id desconhecido; `relink` de efeito sem mudança neste turno |
| `blocked_by_status` | `<nick> não pode <o que>: <causa>` | `enqueue_action`, `attach_reaction` |
| `restore_refused` | `Não foi possível devolver a ação: <motivo>` | `remove_status_effect` com `restoreCancelledActions`, uma por ação recusada, só ao mestre |
| `game_error` | — | qualquer outra falha |

## 14. Histórico

O histórico (REST) registra a aplicação, a edição e a remoção de cada efeito, como cada leitor
pode vê-las, e esconde as ações de quem estava Escondido de quem não o percebia. O contrato
inteiro — formato do evento, `actionId` (a causa), regra de visibilidade, "Ações de quem estava
Escondido" e o endpoint do mestre `POST /matches/{matchUuid}/status-effects/{effectUuid}/reveal`
— está em [`match-history.md`](match-history.md#efeitos-de-status-no-histórico--kind-statuseffect-e-turnsstatuseffects).
Do lado do fio WS: `actionId` nos três comandos, `relink_status_effect`, `turnActionId` e
`cancelledActionIds` (só o mestre) e o desfazer no turno (com a devolução das ações), em §5 e §8.
