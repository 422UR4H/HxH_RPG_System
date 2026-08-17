# Motor de Batalha — modelo técnico

> Consolidação da sessão de design de 2026-08-14/16. Registro completo, com as citações do
> dono do produto e as pontas soltas, em
> [`docs/superpowers/specs/2026-08-14-action-flow-design-notes.md`](../../superpowers/specs/2026-08-14-action-flow-design-notes.md).
>
> **Nada disto está implementado.** O esqueleto (Cena → Round → Turno → Action) existe e é
> correto; o miolo que calcula está vazio. Ver [`flows/05-lacunas.md`](flows/05-lacunas.md).

## Princípios

1. **Servidor autoritativo em tudo.** Todo cálculo roda no Go. O client do mestre envia
   inputs (CD, vantagem, edições) e renderiza. Nenhuma fórmula de jogo em TypeScript.
2. **"Abrir" é passar o microfone.** A operação é de regência da mesa — *"agora é a vez desta
   pessoa narrar"*. O cálculo é efeito colateral. Vale igual para action e para reaction.
3. **Sorteio uma vez; resultado derivado quantas vezes precisar.** Os dados caem quando a
   action ou reaction chega. O número final é refeito a cada edição do mestre e a cada
   reaction que colide. **O mestre nunca re-sorteia o dado do jogador.**
4. **O resultado de um teste é a margem, não um booleano.** Sucesso e falha são leituras da
   margem contra limiares.
5. **A action não carrega status.** Onde ela está diz em que ponto do ciclo está: na fila,
   no turno aberto, ou no histórico.

## Vocabulário

| Termo | Significado | Tipo |
|---|---|---|
| **Turno** | 1 action + suas reactions | `entity/turn.Turn` |
| **Round** | sequência de turnos | `entity/round.Round` |
| **Cena** | sequência de rounds | `entity/scene.Scene` |
| **Fila** | actions declaradas, ordenadas por actionSpeed; atravessa turnos | `action.PriorityQueue` |

`SceneCategory` (`Roleplay`/`Battle`) é organização **narrativa**. `RoundMode`
(`Free`/`Race`) é **regime de motor**. São eixos independentes: um round `Race` pode ficar
salvo numa cena de Roleplay. Pedir iniciativa **força** `Race`.

## Rolagem

- **Teste padrão = 2 D10 somados.** Vale para perícia, acerto e actionSpeed.
- **Crítico** = ambos os dados 10. **Erro crítico** = ambos 1. Não é a soma — é a combinação.
  **O motor precisa guardar os dados individuais**, não só o total. `RollContext.Dice` já faz.
- **Passivo vs. ativo.** Testes passivos usam o **valor médio do conjunto de dados** em vez de
  rolar: **11** para 2 D10, **10** para D20. Usado em: actionSpeed com `RoundMode == Free`,
  esquiva por reflexo (`Reflexo + 11`), defesa à mão livre, deslocamento Shift.
  > O passivo ser exatamente a média é intencional: rolar tem expectativa **zero** de ganho,
  > então o jogador só arrisca quando precisa de sorte acima da média.
- **Vantagem/desvantagem**: rola o conjunto duas vezes, vale o melhor/pior.
  `RollCondition.Bias` (−1/0/+1, acumula).
- **`RollCondition` é a struct do mestre**: `Bias` = dados; `Modifier` = ajuste manual
  (*"+3 porque teve criatividade estratégica"*).

⚠️ `docs/game/dados.md` foi corrigido nesta sessão; a implementação precisa bater com ele.

## Economia de turno — duas barras

Cada personagem tem **duas barras independentes**: `actionSpeed` (ataque, item, habilidade) e
`moveSpeed` (shift, dash, salto, rolamento).

**Fórmula de fechamento do round, por barra:**

```
barra_final = média(velocidades das ações daquela barra no round)
            − (nº de ações naquela barra × preço do round)
barra_final = min(barra_final, preço do round)          // teto
```

- **Preço do round** (`ActionBarCoast`) = a menor velocidade do round. Igual para todos.
- **Carry-over** atravessa para o round seguinte, como crédito ou débito, limitado ao teto.
- Quem não agiu carrega o piso. É legítimo e não é punido.
- **Agir de novo sempre acontece** — o que a segunda rolagem decide é o custo posterior. Uma
  rolagem fraca puxa a média para baixo, atrasa a própria segunda ação dentro do round e pode
  deixar débito para o seguinte. Não é uma aposta sobre *se* age; é sobre quanto custa depois.
- **A recalculação é forward-only.** A primeira action já aconteceu; o que a média move é
  apenas a posição da segunda na fila.

### Ações compostas (move + attack)

Duas formas, e a escolha é do jogador:

| Forma | Resolução |
|---|---|
| **Duas actions separadas** | cada uma na sua barra, resolvida quando aquela barra chegar. A ordem sai do relógio, não da intenção. Quem quiser ordem garantida **não enfileira as duas de uma vez**: manda o movimento, espera resolver, e só então manda o ataque. |
| **Ação combinada** | as duas amarradas, resolvidas **juntas, no tempo da mais lenta** |

| Combinada | Ordem interna |
|---|---|
| **Cait** | livre — atacar antes, durante ou ao fim do movimento, conforme qual barra estiver na frente |
| **Arremetida** (1 slot) | movimento **obrigatoriamente antes** do ataque |
| **Investida** (2+ slots) | idem |

Decisão registrada: **não modelar as variações internas do cait** (atacar antes/durante/
depois). Quem quer controlar a sequência usa duas actions separadas.

### Escala das duas barras

`moveSpeed` e `actionSpeed` estão **na mesma escala** — ambas são `perícia + 2 D10`. A frase
do desenho sobre "quantos slots o personagem percorre" significa *por espaço de tempo*: mais
moveSpeed = mais rápido = mais slots ao longo dos turnos. **Distância é derivada; a barra é
tempo.** Logo o relógio único compara valor com valor, sem conversão.

⚠️ **Dois custos diferentes, não confundir:**

| Custo | O que é | Subtraído de |
|---|---|---|
| `ActionBarCoast` | preço do round = menor velocidade da rodada | da **barra** |
| `actionCoast` | vem do **Peso Excedente** (carga além do peso do personagem) | da **actionSpeed** da própria action |

A actionSpeed final ainda soma `moveSpeed` em arremetida/investida, e sofre cálculo
trigonométrico sobre o vetor `targetSpeed` quando o alvo está em movimento.

**A "regra do 2×" não existe como regra** — é consequência aritmética de o preço ser o menor
do round.

❓ **Em aberto:** cada barra tem o seu próprio preço, ou as duas compartilham um piso?

## Ciclo de uma action

```
enviada → validada → dados sorteados → enfileirada por actionSpeed
   → mestre abre (fecha o turno anterior)
   → mecânica pública, resultado só para o mestre; dono narra
   → alvos reagem
   → mestre abre cada reaction, uma por vez; dono narra
   → colisão recalculada a cada reaction
   → mestre encerra o turno → resultado aplicado e publicado
   → turno vai para o histórico
```

- **Não há confirmação de action.** Abrir vale como aval; editar recalcula e reavisa.
- **Depois do round 0 não existe fase de coleta.** A fila é permanentemente viva.
- **Só o mestre é notificado** de que alguém enfileirou uma action. A fila é secreta; a barra
  e a ordem são públicas.

### Resolução com vários alvos é uma cadeia

A colisão **não** é `f(action, reactions[])`. O estado do ataque sai alterado de cada
resolução e entra na próxima:

```
ataque₀ → resolve(alvo A) → ataque₁ → resolve(alvo B) → ataque₂ → …
```

As reactions chegam ao mestre ordenadas por actionSpeed, mas **ele abre na ordem que quiser —
e isso muda o resultado**. Não é preferência de ritmo: é a única ordem que o motor consegue
calcular, e é poder de jogo que a interface precisa tornar visível.

## Reações

**Passiva é grátis; ativa custa a action.**

| Situação | Efeito |
|---|---|
| Esquiva por reflexo, defesa padrão | passivas, sem custo. Ordem: esquiva → (se falhar) defesa |
| Reação ativa **sem** action enfileirada | a reaction **é** a action |
| Reação ativa **com** action enfileirada | **Desvantagem**: rola de novo, vale a pior das duas velocidades |

⚠️ Na conversão action→reaction **não há média** — há Desvantagem. Média é para múltiplas
actions no round; Desvantagem é para *trocar* o que se ia fazer.

**Três desfechos do lado do alvo:**

| Situação | Motor |
|---|---|
| Não envia nada (ou estoura o timer) | aplica os padrões |
| Envia "não fazer nada" | recebe o ataque **cru**, sem esquiva nem defesa |
| Envia reaction | colide |

**Escapar abre mão da defesa padrão** — falhou, toma dano cheio. Só o **escape defensivo**
mantém a rede de segurança.

**Custo por barra:**

| Reação | actionSpeed | moveSpeed |
|---|---|---|
| Esquiva por reflexo, defesa padrão | — | — |
| **Escape padrão** / escape defensivo | ✔ | ✔ |
| **Escape fechado** | — | ✔ |
| **Esquiva fechada** | — | — |
| **Repelir** | ✔ | — |

É aqui que as variantes fechadas se pagam: feitas no instante exato, sem abrir guarda, elas
devolvem a barra de ação.

### A escada de resultados

O degrau de **10** é o padrão do sistema, mas **não é universal**. A razão é de design:
*"deve ser mais fácil aparar do que acertar o alvo — este sistema já é muito punitivo."*

Repelir, com CD = resultado do ataque:

| Margem | Desfecho |
|---|---|
| `≥ CD + 10` | dano zero **e** bônus = a diferença, contra **aquele alvo**, no próximo turno |
| `CD … CD+9` | dano zero |
| `CD−10 … CD−1` | aparou: **dano zero** + penalidade = a diferença, contra **qualquer um** |
| `< CD − 10` | recebe o ataque |

**Assimetria intencional:** bônus é específico do alvo (você leu *aquele* oponente);
penalidade é geral (você ficou desequilibrado).

**O bônus acumulado é sempre de `actionSpeed`, nunca de acerto.** Disso emerge a mecânica de
duelo sem ninguém programar duelo: dois personagens que se enfrentam aceleram um contra o
outro e passam a trocar golpes mais rápido que o resto da batalha.

## Visibilidade

- Mecânica da action (alvos, arma, perícia) é **pública** ao abrir; o **cálculo é só do
  mestre** até o turno encerrar.
- **Nem todo campo é público, inclusive no histórico.** Uma esquiva fechada não pode revelar
  que embutiu um teste de Evasão — o adversário precisa **deduzir dos números**.
  O Action History é uma **superfície de jogo com visibilidade por campo**, não um log.
- Exceção do percept: no **início de batalha**, só vê os alvos quem percebeu (Percepção vs.
  Furtividade). Quem não viu não recebe os updates daquela action. Bloqueado — os
  subatributos mentais ainda não existem.

## Configuração de partida

Requisito explícito: a estrutura precisa comportar regras configuráveis **desde já**, com
padrões embutidos. Motivação: *"a comunidade não gosta de alguma coisa, então eu
flexibilizo."* Contrapeso: *"não gostaria de um código super complexo para isso."*

**Recomendação:** os **números** viram configuração (degrau de 10, conjunto de dados, valor
médio derivado, timer de reação, teto do carry-over). A **forma da escada** — quantos degraus
existem e o que cada um faz — fica em código, porque mudá-la muda o jogo.

| Regra | Padrão do MVP |
|---|---|
| Conjunto de dados | 2 D10 somados (D20 alternativo) |
| Valor médio | **derivado do conjunto** — 11 para 2 D10, 10 para D20 |
| Timer de reação | desligado |
| Reação padrão na omissão | ligada |
| `fog_mode` | `explored` |

> **Convergência:** o mecanismo de configuração de partida **não existe** no backend, e
> `fog_mode` já esbarra nele (hardcoded em `room.go`, listado nos Known Issues do
> `AGENTS.md`). São o mesmo buraco.

## Pendências estruturais

| Item | Situação |
|---|---|
| `RollCalculator` | retorna 0; ninguém chama |
| `TurnResolver` — ramo `character` | vazio |
| `CharacterStatus` | só comentário; precisa virar código (posturas, barras, livro-razão de bônus/penalidade) |
| `battle.Blow` | campos privados, sem construtor |
| `action.Initiative` | órfão; `ChangeMode` ignora o parâmetro |
| `buildAction` | descarta Skills, Speed, Feint, Attack, Defense |
| Tabela `SystemData` | auditoria de interferência do mestre — só no desenho |
| Onde mora a diferença acumulada | `RollContext`? `CharacterStatus`? derivada da action anterior? |
| Conflito no `Bias` | desvantagem gerada pelo sistema × ajuste do mestre disputam o campo |
| Tela de enviar action | **não existe no front** |
