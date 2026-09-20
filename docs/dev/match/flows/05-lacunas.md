# 05 — O que ainda está oco

> Auditoria do código na Fase 5. Cada item foi verificado no arquivo, não lembrado.

## Bugs abertos

### A penalidade do aparar está escondida de quem pode aproveitar

`ProjectResolution` zera `Payouts` inteiro para terceiros, pela razão certa: a **reserva da
esquiva fechada** revela quanta Evasão foi embutida. Mas o mesmo campo carrega a **penalidade
do aparar**, que nasce `Against: ScopeAnyone()` — e `reacoes.md` é explícito: *"a penalidade de
quem aparou vale contra todo mundo — qualquer um pode aproveitar"*.

O número é dedutível (`cr.Ladder.Difference` viaja), mas aí o cliente faz álgebra — e o próprio
código evitou isso de propósito no `ReactionTotal`, com um comentário contra *"reconstruir o
número por álgebra a partir de `Ladder.Margin`"*.

> O discriminador já existe: esconder `Payouts` **só quando o rótulo foi rebaixado**. A reserva
> é segredo porque a fechada é segredo; repelir é público e declarado.

### `Interact` e `SystemBias` não são persistidos

A tabela `actions` não tem coluna para nenhum dos dois, e `deriveActionType` nem conhece
`Interact`: **abrir uma porta persiste como `"unspecified"`, sem payload**. `SystemBias` é
campo novo da Fase 5 — a Desvantagem de conversão some do histórico. O resultado final continua
certo; o que se perde é *por que* aquele número foi aquele, numa superfície cujo propósito é
deduzir dos números.

### `ReactionResults` é stub com o campo trocado

```go
res.ReactionResults[i] = ReactionResult{ReactorID: r.ReactToID}
```

`ReactToID` é o UUID da **action**; `ReactorID` devia ser o do **personagem**. Compila porque
os dois são `uuid.UUID`. Ninguém lê hoje — não vai ao wire, não é persistido, e a Fase 5
documentou a exclusão — então não é bug vivo. É mina: quem ler primeiro lê lixo com cara de
dado bom. Ou implementa, ou apaga o campo.

### Dois erros engolidos em silêncio

| Onde | O quê |
|---|---|
| `turn_resolver.go` — `case TargetKindUnknown` | `case` vazio: um alvo que o motor não classifica evapora sem ninguém saber |
| `resolveCharacterStep` | ficha faltando devolve `false` sem erro |

Ambos têm `TODO` pedindo para serem superficiados na resolução.

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
| Desempate `t.uuid` na ordenação do histórico | correto pela semântica do SQL, mas o teste não reproduz a falha anterior neste Postgres — registrado como *"correto por semântica, não verificado por teste"* |
| Smoke REST ponta a ponta | não existe fixture de seed campanha → cenário → partida → inscrição → início; substituído por três camadas reais mais checagem ao vivo |

## Fora do motor, por fase

| Item | Quando |
|---|---|
| Qualquer tela do fluxo | Fase 6 |
| Rostering de NPC — nada cria um NPC hoje | fatia própria, antes da Fase 6 |
| Exceção do percept no início de batalha | bloqueada: os subatributos mentais não existem |
| Posturas (condicionam o desconto do escape fechado) | pós-MVP |
| Override do desfecho da cadeia em área | regra de jogo ainda não escrita |
