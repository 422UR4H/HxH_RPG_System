# 05 — O que ainda está oco

> Auditoria do código na Fase 5. Cada item foi verificado no arquivo, não lembrado.
>
> **Atualizado em 2026-09-20:** os cinco bugs abertos foram fechados, e o desempate
> `t.uuid` deixou de ser "correto por semântica, não verificado por teste". O que segue
> em aberto são as regras escritas que o código ainda não executa, e o que está fora do
> motor por fase. O contrato WebSocket do combate agora vive em
> [`../../api/match-combat-ws.md`](../../api/match-combat-ws.md).
>
> **Atualizado em 2026-09-21 (backend-prep-front-combat):** três buracos que bloqueavam a
> Fase 6 do front — nenhuma mensagem movia a peça no tabuleiro, quem entrava/reconectava no
> meio de um combate não tinha snapshot nenhum, e `action_enqueued` chegava vazio — foram
> fechados. Nenhum dos três estava nomeado nesta lista (nasceram de uma auditoria do front,
> não deste inventário); a lista deles, e as três lacunas NOVAS que este trabalho descobriu
> (uma reação de escape não move a peça, a semântica de `Z` em `piece_moved` está em aberto,
> o movimento aplicado na abertura do turno não revalida parede), estão em
> [`../../api/match-combat-ws.md`](../../api/match-combat-ws.md) §9. O catálogo de combate
> por personagem (`GET /charactersheets/{uuid}/combat-catalogue`) também nasceu desta rodada
> — ver [`../../api/character-sheet.md`](../../api/character-sheet.md).

## Bugs que estavam abertos — fechados em 2026-09-20

> **Os cinco desta seção foram fechados.** Ficam registrados porque o *porquê* de cada
> conserto é a parte que não está no diff — e porque cada um deles nomeia uma classe de
> defeito que este motor produz com facilidade.

### ~~A penalidade do aparar está escondida de quem pode aproveitar~~ — fechado

`ProjectResolution` zerava `Payouts` inteiro para terceiros, pela razão certa: a **reserva da
esquiva fechada** revela quanta Evasão foi embutida. Mas o mesmo campo carrega a **penalidade
do aparar**, que nasce `Against: ScopeAnyone()` — e `reacoes.md` é explícito: *"a penalidade de
quem aparou vale contra todo mundo — qualquer um pode aproveitar"*.

O discriminador certo já existia: esconder `Payouts` **só quando o rótulo foi rebaixado** —
a mesma condição que já demota o kind. A reserva é segredo porque a fechada é segredo;
repelir é público e declarado. É isso que o código faz agora.

> Ainda **não há campo de `payouts` no wire**, nos dois caminhos. O conserto é de domínio e de
> persistência; expor o número é decisão de contrato, não descuido de mapeamento.

### ~~`Interact` e `SystemBias` não são persistidos~~ — fechado

`migrations/20260920000000_actions_interact_and_system_bias.sql` acrescenta as duas colunas;
`insertAction` grava, `decodeActionRow` lê de volta, e `deriveActionType` passou a conhecer
`Interact` — **antes de** a linha de `skills`, porque um arrombamento carrega os dois e o que
ele *é* é uma interação.

`Interact` já tinha campo de saída no histórico REST (`ActionResponse.Interact`), que vivia
sempre nulo por falta da coluna; agora funciona. **`SystemBias` continua sem superfície**, e
de propósito: `match-history.md` registra que ele e `RollCheck.Context` são internos do
motor. Persistir sem expor é o estado consistente — o dado deixou de ser *perdido*, e
mostrá-lo é uma decisão de contrato a tomar à parte.

### ~~`ReactionResults` é stub com o campo trocado~~ — **apagado**

Das duas saídas — implementar ou apagar — foi apagar.

Tudo que uma `ReactionResult` poderia carregar já é reportado pelo único caminho que sabe
qual rolagem cada tipo de reação lê: o tipo, o total, o ID próprio, a escada e o veredito de
parada de uma reação **aberta** caem no `CharacterResult` do alvo que a enviou
(`ReactionKind`, `ReactionTotal`, `ReactionID`, `Ladder`, `ReactionStopsAttack`); uma
**anexada e não aberta** é nomeada em `PendingReactions`. Uma segunda lista, chaveada de
outro jeito, seria uma verdade paralela para manter em sincronia com aquelas — e sem leitor.

### ~~Dois erros engolidos em silêncio~~ — fechados

`TurnResolution.Errors` é a superfície que os dois `TODO`s pediam. As faltas são do **motor**,
nunca desfechos de jogo — uma esquiva que falhou não é erro; um alvo que o motor não
classifica é.

| `Kind` | Antes | Agora |
|---|---|---|
| `unknown_target` | `case TargetKindUnknown` vazio | entrada em `Errors` nomeando o alvo |
| `missing_sheet` | `return false` pelado | entrada em `Errors` nomeando a ficha — do **alvo** ou do **ator**, que são casos distintos e agora se distinguem |

`Resolve` continua **total**: sempre devolve uma resolução, e as faltas viajam dentro dela.
Uma colisão com um buraco ainda vale todos os outros números que calculou — a mesma contenção
que `FindMatchHistory` já faz por uma linha ilegível.

`Errors` é **master-only** no wire (`ProjectResolution` o tira de todo mundo mais, como faz
com `PendingReactions`) e **é persistido** em `turns.resolution`: um `missing_sheet` quer
dizer que um alvo não produziu `CharacterResult` nenhum, e um histórico que guardasse o
silêncio leria, um ano depois, como um turno que simplesmente não mirou naquela pessoa.

## Regras escritas que o código não executa

### A corrente de testes — a maior

`match_session.go` rola cada `Skill` da action e **ninguém lê o resultado**. A única leitura de
`Skills` em toda a colisão é a `Evasion` da esquiva fechada, por nome.

A regra está em [`docs/game/combate/acoes.md`](../../../game/combate/acoes.md): cada perícia é
um teste com CD própria, a margem atravessa de um teste para o próximo, errar por 10 ou mais
mata a corrente. **Nada disso existe em código** — então a edição de perícias que a Fase 5
entregou muda uma lista que não decide nada.

> O que fica em aberto de propósito: a consequência para quem falha (guarda aberta, caído, dano
> igual à diferença). É sistema de status, não existe, e a decisão de produto é explícita —
> o sistema propõe um padrão, o mestre substitui.

### `ReboundDamage`

Calculado em `structural_damage.go`, persistido no record, **nunca aplicado ao ator**. Quatro
`TODO`s no mesmo arquivo: aplicar só se for corpo a corpo, subtrair a Defesa do ator, zerar se
for ataque à distância, incluir no broadcast.

### Armadura reduz zero

`ChainState.Reduce` subtrai `armour` — e `turn_resolver.go` declara `const armour = 0`, porque
não existe entidade de armadura nem campo de ficha. **A linha está codificada porque a forma é
o que importa.** Não construa um modelo de armadura para preencher isto.

### `action.Initiative` é órfão

`RoundOrchestrator.ChangeMode` recebe o parâmetro e **ignora**. O assento está reservado para a
regra de jogo que normalmente forçará `Race`; quem troca o regime hoje é o mestre, à mão, por
`change_round_mode`.

## Verificação que ficou devendo

| Item | Situação |
|---|---|
| ~~Desempate `t.uuid` na ordenação do histórico~~ | **verificado.** `TestFindMatchHistoryKeepsEachTiedTurnsReactionWithItsOwnTurn` força o empate de `finished_at` com **os dois** turnos carregando reação — e é isso que faltava: com uma reação só, a ordem restante ainda funciona por acaso, que é por que o teste anterior passava dos dois jeitos. Tire `t.uuid` do `ORDER BY` e ele acusa quatro turnos onde há dois |
| Smoke REST ponta a ponta | não existe fixture de seed campanha → cenário → partida → inscrição → início; substituído por três camadas reais mais checagem ao vivo |

## Fora do motor, por fase

| Item | Quando |
|---|---|
| Qualquer tela do fluxo | Fase 6 |
| Rostering de NPC — nada cria um NPC hoje | fatia própria, antes da Fase 6 |
| Exceção do percept no início de batalha | bloqueada: os subatributos mentais não existem |
| Posturas (condicionam o desconto do escape fechado) | pós-MVP |
| Override do desfecho da cadeia em área | regra de jogo ainda não escrita |
