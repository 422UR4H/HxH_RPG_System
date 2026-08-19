# Motor de Batalha — modelo técnico

> Consolidação da sessão de design de 2026-08-14/16. Registro completo, com as citações do
> dono do produto e as pontas soltas, em
> [`docs/superpowers/specs/2026-08-14-action-flow-design-notes.md`](../../superpowers/specs/2026-08-14-action-flow-design-notes.md).
>
> **Fases 1 a 3 implementadas** (`RollCalculator`, `CharacterStatus`, `MatchRules`, colisão de
> personagem, as duas barras com preço/média/porteiro duplo e o fechamento automático do
> round). Reações ativas, cadeia com vários alvos, regência da mesa e projeção por
> destinatário ainda não existem. Ver [`flows/05-lacunas.md`](flows/05-lacunas.md).

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

### A chave de prioridade

**A velocidade que rolou e a posição na fila deixaram de ser a mesma coisa.**

> **chave da n-ésima ação de um personagem = média(velocidades até n) − (n−1) × preço**

No exemplo canônico (p1=20, p2=23, p3=11, preço=11, segunda rolagem de p2 = 17):

| | n | média | chave |
|---|---|---|---|
| p2, 1ª | 1 | 23 | `23 − 0×11` = **23** |
| p1, 1ª | 1 | 20 | `20 − 0×11` = **20** |
| p3, 1ª | 1 | 11 | `11 − 0×11` = **11** |
| p2, 2ª | 2 | `(23+17)/2` = 20 | `20 − 1×11` = **9** |

Ordem: **p2 → p1 → p3 → p2**. A segunda ação de p2 **rolou 17 e entra na fila em 9** — um
número que não existe dentro da action. Ele nasce de estado do personagem: a média do round,
quantas ações já foram tomadas, e o preço.

Simetria útil: **a chave é o saldo antes de pagar aquela ação**; o saldo depois é
`chave − preço`. É a mesma conta lida em dois momentos.

⚠️ **Consequência estrutural: a fila não pode guardar a chave.** `PriorityQueue` ordena hoje
por `Action.Speed.Result`, um valor guardado. Mas a chave muda quando o personagem manda outra
ação (a média se move), e um heap não suporta re-chaveamento de item já inserido — quebra em
silêncio. **A chave passa a ser calculada na hora de escolher o próximo.** Com uma mesa de 4 a
6 personagens, varrer a lista custa menos que manter heap — e o `ExtractByID` já varre
linearmente, então o heap nunca pagou o próprio custo.

### Fechamento do round, por barra

```
saldo_final = média(velocidades das ações daquela barra que AGIRAM no round)
            − (nº dessas ações × preço do round)
saldo_final = min(saldo_final, preço do round)          // teto
```

- **A média não arredonda.** Ela é guardada com a fração (`float64` no código): truncar seria
  escolher uma política de arredondamento que a regra não pede, e a diferença se acumula ao
  longo dos rounds pelo carry. O exemplo canônico continua dando inteiro exato.
- **A média é por round**, e conta **só as ações que efetivamente agiram**. Uma ação enviada
  mas que não coube no round pertence ao round seguinte e **não entra na média deste**.
- **Preço do round** (`ActionBarCoast`) = a menor velocidade daquela barra na rodada.
- **Carry-over** atravessa para o round seguinte, como crédito ou débito, limitado ao teto.
- Quem não agiu carrega o piso. É legítimo e não é punido.
- **Agir de novo sempre acontece** quando há saldo — o que a segunda rolagem decide é o custo
  posterior. Uma rolagem fraca puxa a média para baixo, atrasa a própria segunda ação dentro
  do round e pode deixar débito para o seguinte.
- **A recalculação é forward-only.** A primeira action já aconteceu; o que a média move é
  apenas a posição da segunda na fila.

### O preço congela na primeira ação aberta

O preço é fixado quando o mestre **abre a primeira ação do round** — a menor velocidade entre
as que já estavam na fila — e **não muda mais até o round fechar**.

**Action atrasada ainda age neste round.** Se a velocidade dela for alta, ela entra na fila
no próprio valor e pode ser a **próxima a abrir**, passando na frente do que ainda não
ocorreu. O que não acontece é desfazer o que já foi jogado.

**Quem não alcança o preço fica de fora do round.** Não paga nada, não vai a saldo negativo:
leva **a rolagem cheia** para o round seguinte.

> Medido na **barra** (`carry + rolagem`), não na rolagem crua — ver "Quem pode agir". No
> round 0 dá no mesmo, porque o carry é zero.

> O teto não é violado nisso. Como o preço é a menor velocidade da mesa no congelamento, quem
> ficou de fora rolou **abaixo** dele — o que ele carrega já é menor que o teto por
> construção.

> Decisão do dono do produto, pela simplicidade: *"uma action que chegue depois com valor
> menor poderia simplesmente não ter custo suficiente pra agir neste turno, então esse valor
> seria um acúmulo para o próximo. Não teria problema algum nisso."*

Descartadas: preço dinâmico que só cai, e recálculo retroativo de tudo. Ambos fariam o mesmo
round cobrar valores diferentes de pessoas diferentes, ou reordenar o que já foi jogado — o
que já havia sido rejeitado antes.

**Corolário:** se depois de pagar o saldo ficou abaixo do preço, a próxima ação que o
personagem mandar **já pertence ao round seguinte**.

### Quem pode agir — são DOIS porteiros, não um

⚠️ **A chave de ordenação e o porteiro de elegibilidade são coisas diferentes.** A chave diz
**quando** a ação sai; o porteiro diz **se** ela sai. Confundir os dois quebra o exemplo
canônico por uma ação.

| | Porteiro |
|---|---|
| **1ª ação do personagem no round** | a **barra** alcança o preço: `carry + rolagem ≥ preço` |
| **2ª em diante** | o **troco** das que já agiram alcança o preço: `saldo após as anteriores ≥ preço` |

A **chave** — `carry + média(até n) − (n−1) × preço` — **só ordena**. Ela pode ficar abaixo do
preço sem que isso impeça a ação de acontecer.

**No exemplo canônico:** depois da 1ª ação, p2 tem troco `23 − 11 = 12` ≥ 11, então **ganha**
a segunda. Só então ela rola 17, a média cai para 20 e a chave dessa segunda ação fica em
`20 − 11 = 9` — abaixo do preço, e mesmo assim ela acontece, em quarto lugar.

> **A elegibilidade é decidida antes da nova rolagem; a chave, depois dela.** É o que sustenta
> *"agir de novo sempre acontece quando há saldo — o que a segunda rolagem decide é o custo
> posterior"*. **O direito, uma vez concedido, não é revogado** por uma rolagem ruim.

⚠️ **O porteiro da 1ª ação é a barra, não a rolagem crua.** A frase *"só quem rola abaixo do
preço fica de fora"* só é exata quando o carry é zero — isto é, no round 0. Com carry
positivo, um personagem que rolou abaixo do preço **pode ser resgatado pelo crédito**.

### Quando o round fecha

> **O round fecha quando nenhuma action pendente passa no porteiro que lhe cabe.**

Não é "quando as barras acabam". A barra **não zera** — termina em qualquer valor, inclusive
negativo. E não é sobre a chave: uma ação com chave abaixo do preço ainda acontece, se já
tinha passado no porteiro.

**Um personagem que não enviou action não tem saldo.** Saldo só significa alguma coisa se
houver action pendente para pagar com ele. Quem tem crédito sobrando de um round anterior mas
nunca mandou nada **não segura o round**: simplesmente não age, e a barra é limitada ao teto e
carregada para o seguinte.

Por isso **não existe deadlock e não é preciso "passar a vez"**. E o round fecha sozinho mesmo
com alguém muito rápido: cada ação extra derruba a média e o troco, até o porteiro fechar. É
auto-limitante.

### Como o carry-over entra no round seguinte

O saldo que atravessou **soma na barra** do round novo, antes de qualquer pagamento:

> **chave da n-ésima ação = carry + média(velocidades até n) − (n−1) × preço**

Vem do desenho: *"quando um jogador envia sua action, o sistema rola a actionSpeed somada à
iniciativa (ou não), que define o valor da barra"*, e o excedente *"é um bônus que fica na
barra de ação"*. É também o que faz o teto estabilizar em vez de crescer: quem fica parado
com rolagem 20 e preço 11 leva 9; no round seguinte tem `20 + 9 = 29`, paga 11, sobra 18 →
limitado a 11. Estabiliza.

❓ Derivado, não dito com todas as letras. Vetar em uma linha se o carry não somasse na barra
e servisse apenas para decidir quantas ações cabem.

### Ações compostas (move + attack)

Duas formas, e a escolha é do jogador:

| Forma | Resolução |
|---|---|
| **Duas actions separadas** | cada uma na sua barra, resolvida quando aquela barra chegar. A ordem sai do relógio, não da intenção. Quem quiser ordem garantida **não enfileira as duas de uma vez**: manda o movimento, espera resolver, e só então manda o ataque. |
| **Ação combinada** (cait, arremetida, investida) | **uma action só**, com `Move` e `Attack` preenchidos, que **cobra as duas barras** e acontece **no tempo da mais lenta** |

#### A ação combinada é UMA action

`Move` e `Attack` vivem dentro da mesma `Action`. O mestre **abre uma vez**, e é **um turno
só**. Não existe divisão em duas actions, nem duas entradas na fila, nem aresta de dependência
entre metades: a modelagem foi construída para haver uma action, e o movimento mora dentro dela.

**Ela ocorre no instante da metade mais lenta.** As duas velocidades continuam gravadas na
action — `Speed.Result` para a barra de ação, `Move.FinalSpeed` para a de movimento — e as duas
barras são cobradas. O que a combinação decide é apenas **quando** a action acontece:

```
chave da ação combinada = min(chave na barra de ação, chave na barra de movimento)
```

Como a chave maior age primeiro, o `min` é a mais lenta. Se o personagem é rápido de mão e
lento de pé, o golpe espera o pé; se é o contrário, o deslocamento fica "atrasado" até a vez do
ataque.

**Ela só age se passar no porteiro das duas barras.** Cobra as duas, então precisa poder pagar
as duas. Uma investida cuja barra de ação não alcança o preço fica de fora do round inteira —
não só o ataque. O contrário deixaria um personagem sem barra de ação atacar de carona no
movimento, e a economia deixaria de valer.

> Racional do dono do produto: *"um round é praticamente todo mundo agindo ao mesmo tempo. A
> resolução das actions é discreta, mas estamos numa simulação muito mais próxima da
> dinamicidade de uma batalha real. Na prática o personagem ainda está deslocando naquele
> slot e preparando seu ataque, na iminência de atacar."*

Vale igual para **cait**, **arremetida** (1 slot) e **investida** (2+ slots).

Decisão registrada: **não modelar as variações internas do cait** (atacar antes/durante/
depois). Quem quer controlar a sequência usa duas actions separadas.

#### ❌ Descartado: duas resoluções com aresta de dependência

Uma versão anterior desta seção dizia *"duas resoluções, com aresta de dependência"* e
*"não é no tempo da mais lenta"*: cada metade resolveria no seu próprio tempo, com outras
actions podendo cair entre elas. **Está descartado**, por decisão do dono do produto:

> *"eu tinha conversado a respeito de não ter problema uma investida poder ocorrer com um delay
> entre o movimento e o ataque... mas essa solução é ruim, então é melhor que ocorra no pior
> tempo de action mesmo."*

A complexidade técnica não se pagava — duas resoluções de uma action só significam dois turnos
carregando a mesma `Action`, e a tabela `actions` é chaveada pelo UUID da action. O ganho de
fidelidade era pequeno perto disso.

### Action enviada no meio do round

Uma action enviada durante a rodada **entra na fila com a actionSpeed que ela rolou**, mesmo
que esse valor seja maior que o de actions que **já foram resolvidas** naquele round. O
sistema **não** tenta reordenar retroativamente.

> *"Eu nunca tinha conseguido amarrar isso... acho que o ideal é ignorar e simplesmente
> enfileirar a action dele com a actionSpeed que ele obteve mesmo. Pelo que definimos sobre o
> mestre antecipar uma action, também não teria problema ficar na ordem 'errada'."*

Consistente com o `pull_action`, que já quebra a ordem por decisão do mestre.

❌ **Descartado:** o desenho antigo previa *"se o mestre abrir qualquer action do próximo
turno, as do turno atual são movidas para o próximo"*. Isso **nunca chegou ao código** e fica
descartado — o receio que o motivava foi resolvido aceitando que a ordem do round não é
garantia forte. **O histórico não necessariamente terá as actionSpeeds em ordem, e tudo bem.**

### Encerrar o turno com reactions pendentes

O mestre **nunca fica travado**. Se encerrar com reactions enviadas mas não abertas, elas
**entram no cálculo normalmente** — o jogador não é punido mecanicamente — mas perdem o
momento de narrar.

Requisito de interface: o sistema **confirma com uma mensagem explicando o que vai
acontecer** antes de encerrar nessa situação.

### Requisito de front — rascunho da action

Para o padrão "mando o movimento, espero, depois mando o ataque" ser usável, o front
**precisa preservar o rascunho** da configuração:

- Ao fechar a bottom sheet, a habilidade/arma selecionada **mantém** a configuração deixada,
  incluindo o alvo.
- Trocar de alvo ou de habilidade dentro da bottom sheet **migra** a configuração para o novo
  alvo/habilidade, em vez de resetar.

O alvo é escolhido clicando no personagem no campo — é isso que abre a bottom sheet. Alvo e
habilidade são ambos **configuração da action**.

### Movimento: como a velocidade nasce e se acumula

> Escrito para **não precisar ser explicado de novo**. Quando a fatia de movimento for
> implementada, referencie esta seção.

#### `Velocity` é um vetor; `Speed` é o escalar dentro dele

Já existe em código, e foi preparado exatamente para isto:

```go
// internal/domain/match/entity/action/velocity.go
type Velocity struct {
    Speed         float64 // o escalar — a velocidade de deslocamento
    DirectionPlan float64 // ângulo no plano (graus)
    DirectionAlt  float64 // ângulo vertical (graus) — saltos, voo
}
```

`CharacterStatus` carrega esse vetor. Os **dois ângulos** existem porque o movimento não é só
no plano: saltos e voo usam o vertical.

⚠️ **Não confundir com a perícia.** `action.Velocity` (este vetor) está **correto**. O que
está errado é `enum.Velocity`, uma `SkillName` — ver a nota de renomeação abaixo.

#### Todo movimento começa como Shift ou Dash

| | Perícia | Rola? |
|---|---|---|
| **Shift** — deslocamento controlado | `Brake` | não — valor passivo |
| **Dash** — arrancada | `Accelerate` | sim |

#### As transições não são simétricas

- **Shift → Dash**: basta enviar a próxima move action como Dash. O personagem acelera. Sem
  teste.
- **Dash → Shift**: exige um **teste de Brake**, e **o sistema o insere automaticamente** —
  o jogador não pede. Vale igual para **mudar de direção durante uma corrida**.

O motivo é físico: sair da inércia é uma escolha; interrompê-la ou desviá-la é uma disputa
contra a própria velocidade que você já carrega.

#### `Charge` acumula na `Speed`

O personagem pode **continuar acelerando**. Isso usa a perícia `Charge`, que **acumula na
velocidade** — a `Speed` sobe acima do que a perícia base entregaria numa arrancada só.

#### ⭐ Depois que o movimento começou, quem manda é a `Speed`

Este é o ponto que muda a implementação:

> Assim que o personagem está em movimento, a velocidade da move action **deixa de ser o
> teste de `Accelerate` ou `Brake`** e passa a ser **o resultado produzido pela aceleração —
> a própria `Speed` com que ele se desloca.**

Ou seja, há dois regimes:

| Momento | O que alimenta a velocidade da move action |
|---|---|
| **Iniciando o movimento** | o teste — `Accelerate` (dash) ou `Brake` passivo (shift) |
| **Já em movimento** | a **`Speed`** acumulada em `CharacterStatus.Velocity` |

E é por isso que `Charge` importa: ele empurra a `Speed`, e a `Speed` é o que passa a contar.

> ❓ O dono do produto escreveu "actionSpeed" ao descrever isto. Registrado como a velocidade
> da **move action** — a barra de movimento —, porque todo o contexto é deslocamento.
> Corrigir em uma linha se a intenção era a barra de ação.

#### ✅ Renomeação feita: `enum.Velocity` → `Quickness` (Fase 3)

Sob Agilidade existem **`Accelerate`, `Brake` e `Quickness`** — começar um movimento, interrompê-lo,
e o jogo de pés que move o personagem dentro do slot. A terceira se chamava `Velocity`, o que a
nomeava por uma grandeza em vez de pelo que ela testa, e colidia com `action.Velocity`, o vetor
de movimento, que é outra coisa e **continua como está**.

Alcance do que foi renomeado: o enum e seu valor serializado (`"Velocity"` → `"Quickness"`), 12
ocorrências em `character_class_factory.go`, o modelo e as queries do gateway
(`VelocityExp` → `QuicknessExp`, coluna `velocity_exp` → `quickness_exp` via
`migrations/20260818000000_rename_velocity_to_quickness.sql`), e as três chaves `velocity` no
front, que lê o nome serializado em minúscula — PR próprio no repo React.

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

**Cada barra tem o seu próprio preço**, calculado separadamente — mas as duas correm num
relógio só. Decidido; não reabrir.

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

⚠️ **Condicionado à postura — quando postura existir.** O desconto do escape fechado
(consumir só a barra de movimento) passará a exigir **postura evasiva**. Enquanto a regra de
posturas não existir, o desconto **vale sem essa condição**.

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
médio derivado, timer de reação). A **forma da escada** — quantos degraus
existem e o que cada um faz — fica em código, porque mudá-la muda o jogo.

| Regra | Padrão do MVP |
|---|---|
| Conjunto de dados | 2 D10 somados (D20 alternativo) |
| Valor médio | **derivado do conjunto** — 11 para 2 D10, 10 para D20 |
| ~~Teto do carry-over~~ | **não configurável** — é sempre o preço do round |
| Timer de reação | desligado |
| Reação padrão na omissão | ligada |
| `fog_mode` | `explored` |

> **Fase 1:** `MatchRules` existe como value object, com os padrões da tabela acima,
> recebido por parâmetro. Persistência, REST e o desbloqueio do `fog_mode` em `room.go`
> são fatia própria — `MatchRules.FogMode` é ponteiro, e a resolução é
> `partida ?? mapa ?? explored` em `MatchRules.ResolveFogMode`.

## O que a Fase 2 fixou no motor

### Sorteio uma vez, no lugar certo

Os dados caem em **`MatchSession.rollActionDice`**, chamado por `EnqueueAction` e por
`AttachReaction` — ou seja, no instante em que a action ou a reaction chega — e ficam em
`action.RollCheck.Attempts` (`RollAttempts{Primary, Secondary}`, movido do pacote `service`
para `action` para que um `RollCheck` possa carregá-lo).

A consequência é a propriedade que sustenta as fases seguintes: **`TurnResolver.Resolve` é
uma função pura do turno.** Ele deriva, nunca rola. Recalcular a colisão a cada reaction
(Fase 4) e a cada edição do mestre (Fase 5) sai de graça e sem re-sortear nada.

Um `RollCheck` cujos dados já caíram é deixado em paz, então chamar o sorteio duas vezes é
inofensivo.

⚠️ **Duas famílias de rolagem.** O teste usa `MatchRules.DiceSet` (2 D10). O **dano usa os
dados da própria arma** (`item.Weapon`, Espada = D10 + D4) e vai só em `Primary` — dano não
tem vantagem. Sem arma = `enum.Fist`, que é uma entrada real do catálogo.

### `RollSource` — o seam determinístico

`RollCalculator.Roll(rules, src)` e `RollCalculator.RollDice(sides, src)` recebem de onde vem
a face do dado. `nil` = `DiceRoller{}` (produção, crypto/rand); um teste passa uma fonte
roteirizada. `MatchSession.SetRollSource` existe só para os testes.

Sem isso, os critérios de pronto das Fases 3 e 4 — que citam números exatos — dependeriam de
sorte.

### A colisão, na ordem das regras

1. **Acerto** — teste ativo do atacante, derivado dos dados que já caíram. O `ModifierLedger`
   entra como `nil` aqui: a diferença acumulada é sempre de actionSpeed, nunca de acerto.
2. **Esquiva por reflexo** — passiva (`Reflexo + valor médio`), grátis, automática.
3. **Defesa** — só se a esquiva falhar, com CD um degrau menor que o ataque.
4. **Dano** — §4.7 do spec de design.

**Empate favorece o defensor** nos dois testes, como na tabela do repelir, onde cair
exatamente na CD já é linha de defensor.

### Dry-run: calcula sempre, aplica uma vez

`TurnResolution` carrega `CharacterResults` com o dano **projetado** — nada tocou ficha
nenhuma. A aplicação de verdade acontece em `MatchSession.closeOpenTurn`, no fechamento
implícito que `OpenNextAction`/`PullAction` já faziam, e devolve `[]DamagedCharacter` para o
use case persistir com `sheet.Repository.UpdateStatusBars`.

### Serialização: `r.mu` atravessa o `Execute`

`MatchSession` não tem lock próprio — `room.go` é a única serialização. As rotas
`enqueue_action`, `attach_reaction`, `open_next_action` e `pull_action` **soltavam o
`RLock` antes de chamar o use case**, então só o ponteiro da sessão estava protegido, não o
que o use case fazia com ela. Um jogador enfileirando corria com o mestre abrindo, na mesma
`PriorityQueue`.

A Fase 2 passou a segurar o **write lock durante o `Execute`** nas quatro rotas. Ficou mais
grave com esta fase porque essas rotas passaram a sortear dados, resolver o turno e escrever
HP em ficha. Provado pelo `TestE2E_AttackAgainstACharacterProducesDamage` sob `-race`.

### `actorID` é o `sheetUUID`

`Action.actorID` deixou de ser o jogador autenticado e passou a ser o **personagem** que age —
o mesmo ID que a peça do tabuleiro carrega como `CharacterID` e o mesmo que um `TargetID`
carrega. `ActionPayload.actorId` é obrigatório no wire. A autorização continua por jogador:
`EnqueueAction` verifica `charToPlayer[actorCharID] == playerUUID`.

## O que a Fase 3 fixou no motor

### A chave não mora na action

`PriorityQueue` deixou de ser heap — `entity/action/priority_queue.go` agora é uma lista
simples, em ordem de inserção. A razão é estrutural, não de gosto: a chave de uma ação
pendente,

```
carry + média(velocidades até n) − (n−1) × preço
```

é estado do **personagem**, não da action, e se move toda vez que esse personagem manda outra
ação — a média do round desliza sob ela. Um heap não sabe re-chavear um item que já guarda;
continuaria devolvendo uma ordem obsoleta, em silêncio. Por isso a chave passa a ser calculada
**na hora da seleção**, por `service.RoundScheduler` (`round_scheduler.go`, privados
`keyOf`/`keyOnBar`) — nunca guardada. Com 4 a 6 personagens numa mesa, varrer a lista custa
menos que manter o heap, e `ExtractByID` já varria linearmente desde sempre.

### Os dois porteiros em código

`service.BarEconomy.IsEligible` (`bar_economy.go`) é literalmente os dois porteiros da seção
"Quem pode agir" acima: primeira ação do personagem no round mede a **barra** contra o preço
(`carry + rolagem ≥ preço`); segunda em diante mede o **troco** das que já agiram
(`Balance(...) ≥ preço`). `BarEconomy.Key` é a chave — ordena, nunca decide se a ação
acontece. Os dois vivem separados no código exatamente como vivem separados na regra: nada
chama `Key` para decidir elegibilidade, nem `IsEligible` para decidir ordem.

### Onde o preço mora

`Round.prices` (`entity/round/round.go`) substituiu o antigo campo único `coast` por um mapa
`action.Bar → int`, um preço por barra. `RoundScheduler.FreezePrices` o congela na primeira
seleção que enxerga trabalho pendente **naquela barra** — não no round inteiro — e
`Round.FreezePrice` é idempotente: a primeira chamada vence, toda chamada seguinte é
ignorada. Uma barra ausente do mapa ainda não precificou; é também a leitura constante de um
round `Free`, que não tem preço nenhum.

### `Speeds` são as que agiram

`MatchSession.recordActed` grava a velocidade de uma ação em cada barra que ela cobrou, mas só
quando a ação **abre** (dentro de `OpenNextAction`/`PullAction`) — nunca quando ela chega à
fila. `deriveSpeeds`, chamado por `EnqueueAction`, só calcula o número da velocidade; não o
registra em `ResourceBar.Speeds`. Essa separação é o que compra a regra "action atrasada ainda
age inteira": uma ação que nunca alcançou o preço nunca entrou em `acted`, então não há nada
para desfazer quando ela rola para o round seguinte — ela simplesmente carrega o valor que já
tinha.

### A média não trunca

`BarEconomy.Mean` — uma função, sem `math.Floor` nem divisão inteira — devolve
`float64(soma)/float64(n)` e guarda a fração. Truncar seria escolher uma política de
arredondamento que a regra nunca pediu, e o erro se acumularia round a round pelo carry; o
exemplo canônico continua fechando em número inteiro exato porque os números do exemplo foram
escolhidos para isso, não porque o código arredonda.

### A ação combinada é UMA action, na prática

O desenho já registrado em "Ações compostas" acima chegou ao código sem desvio: `action.Bars()`
(`entity/action/bar.go`) devolve as duas barras quando a action carrega `Move` e algo que cobra
a barra de ação; `SpeedOn(bar)` devolve a velocidade certa para cada uma, sem re-rolar nada; e
`RoundScheduler.keyOf` agenda pelo `min` das duas chaves, gatilhando o porteiro **das duas
barras** — não só uma. O mestre abre uma vez, é um turno só, e a versão descartada com duas
resoluções e aresta de dependência entre metades não foi construída: a modelagem inteira
pressupõe uma `Action`, e não haveria onde pendurar uma segunda resolução sem duas entradas na
fila — o que a tabela `actions`, chaveada pelo UUID da action, não comporta de graça.

### `Race` é alcançável

`ChangeRoundModeUC` (`application/match/change_round_mode.go`) troca o regime do round ativo
via `MatchSession.SetRoundMode` → `Round.SetMode`, restrito ao mestre (`ErrNotMatchMaster`
para qualquer outro chamador). Trocar no meio do round é permitido de propósito: a economia
simplesmente recomeça a contar dali — ninguém "já agiu" do ponto de vista das barras, e os
preços congelam na próxima seleção. **Iniciativa continua de fora**: é a regra de jogo que
normalmente forçaria `Race`, e fica para uma fatia futura; esta fase só entrega o regime em
si, ligável pelo mestre.

### O round fecha sozinho

O predicado é `RoundScheduler.AnyEligible` — sua negação, não "as barras acabarem". Quando
nenhuma ação pendente passa no porteiro que lhe cabe, `MatchSession.OpenNextAction` marca
`TurnTransition.RoundExhausted = true` em vez de abrir algo, e é aí que `CloseRoundUC`
finalmente ganha um chamador: o caminho de auto-fechamento em
`application/match/open_next_action.go` o executa na hora, loga e segue adiante se falhar — a
mesa não pode ficar sem o bastão por causa de uma falha de fechamento. `bars_updated` e
`round_closed` saem de `room.go` nessa mesma passada.

⚠️ **`ProjectOrder` e `SelectNext` compartilham o desempate.** Os dois usam "quem entrou
primeiro vence o empate" — `SelectNext` só troca o melhor quando a chave é **estritamente**
maior; `ProjectOrder` ordena com `sort.SliceStable`. Divergir os dois faria a barra geral
mostrar uma ordem que a mesa depois não vê acontecer.

### As barras são públicas

`bars_updated` (`BarsUpdatedPayload`, `internal/app/game/message.go`) é broadcast, não
projeção por destinatário: carrega os preços congelados, o saldo e o histórico de velocidades
de cada personagem em ambas as barras, e a ordem projetada
(`MatchSession.ProjectedOrder`/`RoundScheduler.ProjectOrder`) — quem age a seguir e em qual
barra. **Nada que identifique a ação em si** entra no payload: sem ID de ação, arma, alvo ou
perícia. Isso é o que sustenta "a fila é secreta; a barra e a ordem são públicas" — um jogador
sem visão da barra geral só descobre que era a vez dele depois que passou.

### Um gotcha de teste: `rollActionDice` rola quase tudo, mas não é uniforme

`MatchSession.rollActionDice` sorteia, na chegada, todo teste que a action carrega — mas
**quantos dados caem depende do regime do round e da categoria de movimento**, não é "tudo,
sempre":

- **`Speed` (actionSpeed) só rola em `Race`.** Em `Free` ela é passiva — não há disputa sobre
  quem age primeiro, então não há o que rolar — e nenhum dado cai para ela
  (`match_session.go:590-594`). Um teste em `Free` que reserva uma face para a velocidade
  sorteia uma a mais do que o código consome, e todo número depois dela sai deslocado.
- **Feint, cada `Skill`, `Move.Charge`, `Attack.Hit`, `Attack.Charge` e o dano da arma**
  (a outra família de rolagem — os dados da própria arma, só `Primary`, sem vantagem) rolam
  **incondicionalmente**, em qualquer regime, sempre que o campo correspondente existe na
  action.
- **`Move.Speed` rola sempre, exceto em `Shift`** — decidido por `Move.Category`, não pelo
  round: `Dash` rola, `Shift` toma o valor passivo e não consome dado nenhum.

Um teste com fonte de dados roteirizada precisa contar exatamente esses sorteios, na ordem
certa, para o regime e a categoria de movimento da action em questão; foi achado instrumentando
a fonte depois que um teste passou com um número que só batia por coincidência. Do outro lado,
um teste **passivo não pode consumir dado nenhum** — um sorteio fantasma é inofensivo em
produção e venenoso em teste: drena a fonte roteirizada e desloca todo número que vem depois. É
por isso que tanto `Speed` em `Free` quanto `Move.Speed` em `Shift` pulam o sorteio — ambas são
passivas por definição, e testá-las mesmo assim quebraria qualquer script de dados a partir
dali.

## Pendências estruturais

| Item | Situação |
|---|---|
| `RollCalculator` | ✅ Fase 1 — `Roll` sorteia os dois conjuntos uma vez, `Derive` recalcula quantas vezes o mestre editar. ✅ Fase 2 — a `MatchSession` o chama na chegada da action, e ganhou o seam `RollSource` |
| `TurnResolver` — ramo `character` | ✅ Fase 2 — acerto, esquiva por reflexo e defesa passivas, dano e `CharacterResult` |
| `CharacterStatus` | ✅ Fase 1 — `ResourceBar` (duas barras), `ModifierLedger`, `Stance` reservado |
| `battle.Blow` | ✅ Fase 2 — construtor e acessores; `defense` virou ponteiro (nil = defesa passiva) |
| `action.Initiative` | órfão; `ChangeMode` ignora o parâmetro |
| `buildAction` | ✅ Fase 2 — mapeia o payload inteiro, com a fronteira `string → enum.SkillName` e `→ enum.WeaponName` |
| Tabela `SystemData` | auditoria de interferência do mestre — só no desenho |
| Onde mora a diferença acumulada | ✅ Fase 1 — `ModifierLedger` no `CharacterStatus`, com `AgainstID`, `ExpiresAt` e `Source` |
| Conflito no `Bias` | ✅ Fase 1 — `RollCondition.Bias` é do mestre; o viés do sistema é um `Modifier` de `Source: system`, e o `RollCalculator` soma os dois em `Derive` |
| Tela de enviar action | **não existe no front** — Fase 6 |
| Escada de margem | ✅ Fase 2 — `service.ClimbLadder` como função pura, sem reação ligada nela. A Fase 4 liga o repelir |
| Aplicação do dano na ficha | ✅ Fase 2 — dry-run em toda resolução, aplicado uma vez no fechamento do turno e persistido via `UpdateStatusBars` |
| `PriorityQueue` | ✅ Fase 3 — deixou de ser heap; virou lista simples, chave calculada em `RoundScheduler` na hora da seleção |
| `BarEconomy` / `RoundScheduler` | ✅ Fase 3 — preço por barra, média sem truncar, porteiro duplo (`IsEligible`), chave (`Key`), carry-over com teto (`CloseBalance`), projeção da ordem (`ProjectOrder`) |
| Fechamento do round | ✅ Fase 3 — `RoundScheduler.AnyEligible` nega, `OpenNextActionUC` chama `CloseRoundUC` (primeiro chamador que ele ganha), `room.go` emite `round_closed` |
| `RoundMode.Race` | ✅ Fase 3 — alcançável via `ChangeRoundModeUC`/`change_round_mode`, master only. Iniciativa continua fora — `action.Initiative` segue órfão |
| `bars_updated` | ✅ Fase 3 — broadcast com preços, saldos/velocidades por personagem e a ordem projetada; nada que identifique a action |
