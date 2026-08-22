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

### Modificadores: o que cada um modifica, e contra quem

O sistema tem **mais de um tipo de reserva acumulada**, e por isso o `Modifier` precisa dizer
duas coisas que hoje não diz:

```go
type Modifier struct {
    Amount    int
    Bias      int
    Applies   Dimension    // ← novo: actionSpeed | dodge | …  o QUE ele modifica
    Source    Source       // system | master
    Against   Scope        // ← muda: todos | apenas X | todos MENOS X
    ExpiresAt Lifetime
    Reason    string
}
```

**`Applies`** existe porque a reserva do duelo (repelir/aparar) modifica `actionSpeed`,
enquanto a da esquiva fechada modifica **esquiva**. Sem esse campo o ledger não consegue
segurar as duas.

**`Against` deixa de ser um ponteiro** (`nil` = todos, senão um alvo). Falta a terceira forma:
**todos menos X** — que é exatamente o que a esquiva fechada produz, um bônus contra
*terceiros*, isto é, contra qualquer um que não seja o oponente do duelo.

### O bônus da esquiva fechada

A rolagem de **Evasão não soma** à esquiva ou ao escape. Ela entra na **lógica de
Desvantagem**: rola-se os dois e vale o **pior**.

> **O bônus é a diferença entre os dois valores** — a esquiva que o personagem não precisou
> gastar. Ele vale **contra terceiros**: qualquer um que tente pegá-lo num instante de guarda
> aberta.

É literalmente a estratégia do Kuroro: esquivar sem usar o máximo, e guardar a sobra para quem
vier de fora do duelo.

### A cadeia em área — o que passa de um alvo para o outro

**Não há regra rígida: é contextual, e o mestre pode alterar em qualquer ponto.** O que existe
é um **padrão por tipo de reação**, e o mestre sobrepõe quando a cena pedir.

O que a cadeia carrega é o **ataque residual** — o que sobrou do golpe depois de cada alvo.

| O alvo… | O que chega no próximo |
|---|---|
| **Esquivou** | o ataque **cheio**, sem alteração — desviar não gasta o golpe |
| **Repeliu com sucesso** | **nada — o ataque para aqui** |
| **Foi atingido** | reduzido pela **armadura** do alvo atingido |
| **Defendeu** | reduzido pela **defesa da arma** com que ele defendeu |

**Repelir encerra o ataque, mas não cancela as reações seguintes.** Elas **acontecem** — quem
tinha mandado scape se desloca —, só que sem chance de ser atingido. A reação é "desperdiçada"
no sentido mecânico, não no narrativo.

> ⚠️ **O mestre pode permitir que o ataque siga mesmo após um repelir bem-sucedido.** É
> sobreposição de regra padrão, não exceção codificada.

**A armadura reduz duas vezes:** para o alvo que a veste **e** para quem vem depois. Vale igual
para o **Nen**, que funciona como armadura — quando existir.

**A defesa da arma só entra aqui.** É neste ponto da cadeia que o campo `Weapon.defense` tem
função: ele reduz o que passa adiante quando alguém defende.

#### Ataque sequencial × simultâneo

O padrão acima descreve um ataque **sequencial**, que atravessa os alvos. Existe outro tipo:
o que **atinge todos ao mesmo tempo**.

No simultâneo, **o ataque não diminui** — todos recebem o mesmo. Mas narrativamente o mestre
**ainda abre um alvo por vez**, para cada um dizer como reagiu. A cadeia continua sendo o
gesto de mesa; só a aritmética muda.

> **Reservar na modelagem.** Isso será uma **configuração do tipo de habilidade**. As
> habilidades especiais **ainda não existem** no sistema e só chegam pós-MVP — mas o modelo do
> ataque precisa nascer com esse eixo previsto, senão a cadeia vira `if` retrofitado depois.

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

**O bônus do repelir e a penalidade do aparar são de `actionSpeed`, nunca de acerto.**

⚠️ **Isso vale para o acúmulo do duelo, não é lei global do sistema.** Uma versão anterior
generalizava para "todo bônus acumulado é de actionSpeed" — **falso**. Existem outras reservas
com outra natureza; a da esquiva fechada é de esquiva. Cada modificador diz o que modifica.

Disso emerge a mecânica de
duelo sem ninguém programar duelo: dois personagens que se enfrentam aceleram um contra o
outro e passam a trocar golpes mais rápido que o resto da batalha.

### O tipo da reação é declarado, não inferido

`Action.Bars()` deriva o custo de **quais componentes estão preenchidos**. Isso resolve as
actions — mover, agir, ou os dois — e **não consegue expressar o catálogo de reações**:

| Reação | Componentes | `Bars()` responde | Deveria responder |
|---|---|---|---|
| Escape padrão | `Dodge{Scape}` + `Move` | `[action, move]` | `[action, move]` ✔ |
| Escape defensivo | `Dodge{Scape}` + `Move` | `[action, move]` | `[action, move]` ✔ |
| Escape fechado | `Dodge{Scape}` + `Move` | `[action, move]` | `[move]` ❌ |
| Esquiva fechada | `Dodge{Close}` | `[action]` | **vazio** ❌ |
| Repelir | não existe campo | `[action]`, por vazio | `[action]` — por acidente |
| Não fazer nada | nada | `[action]` | **vazio** ❌ |

Os três escapes têm **exatamente a mesma forma** e custos diferentes. Nenhuma inferência sobre
campos preenchidos separa os três, porque a informação que falta não está na forma — está na
**intenção do jogador**. E intenção se declara.

**Decisão: a reação carrega o seu tipo.**

```go
type ReactionKind string

const (
    ReactNothing      ReactionKind = "nothing"       // recusa até as passivas
    ReactDodge        ReactionKind = "dodge"         // esquiva ativa — arriscar a rolagem
    ReactClosedDodge  ReactionKind = "closedDodge"   // esquiva fechada
    ReactEscape       ReactionKind = "escape"        // escape padrão
    ReactEscapeGuard  ReactionKind = "escapeGuard"   // escape defensivo
    ReactClosedEscape ReactionKind = "closedEscape"  // escape fechado
    ReactRepel        ReactionKind = "repel"         // repelir
)
```

`Action` ganha o campo. **Não é uma struct aninhada** e não é o discriminador de "isto é uma
reação": esse já existe e é o `ReactToID`, que é `uuid.Nil` numa action. A regra de validação
é o par — `ReactToID != Nil` exige `ReactionKind != ""`, e o contrário.

Repelir ganha o componente que lhe falta, no mesmo formato da defesa:

```go
type Repel struct {
    Weapon *enum.WeaponName
    RollCheck
}
```

**O custo sai do tipo, não da forma:**

```go
func (k ReactionKind) Bars() []Bar {
    switch k {
    case ReactRepel:                    return []Bar{BarAction}
    case ReactClosedEscape:             return []Bar{BarMove}
    case ReactEscape, ReactEscapeGuard: return []Bar{BarAction, BarMove}
    default:                            return nil   // nothing, dodge, closedDodge
    }
}
```

E `Action.Bars()` ganha uma primeira porta: se há `ReactionKind`, é ela quem responde.

⚠️ **A invariante "`Bars()` nunca é vazio" passa a valer só para actions agendadas.** Ela
existia porque o escalonador precisa precificar toda action por alguma barra — e reação
**não é agendada**: não entra na `activeQueue`, não é aberta por `OpenNextAction`. Vazio é a
resposta *correta* para uma reação grátis. Nenhum caller quebra: os quatro do
`RoundScheduler` só veem a fila, e o `recordActed` itera o slice — iterar vazio é no-op.

**As passivas não são `ReactionKind`.** Esquiva por reflexo e defesa padrão não são enviadas
por ninguém: são o que o motor aplica quando nada chega. Não precisam de tipo porque não têm
remetente. `ReactNothing` existe justamente porque **recusar as passivas é um envio** — e a
única forma de distinguir "não mandou" de "mandou nada" é ter recebido alguma coisa.

**`enum.DodgeCategory` é absorvida.** Ela é hoje `{Evasive, Close, Scape}`, sem nenhum
consumidor além do passthrough em `action_mapper.go`. É o mesmo eixo do `ReactionKind`, só que
estritamente menos expressivo: `Scape` sozinho cobre os **três** escapes, que é exatamente a
distinção que falta. Mapa da absorção — `Evasive → ReactDodge`, `Close → ReactClosedDodge`,
`Scape → ReactEscape`; os outros dois escapes são novos. `Dodge` fica só com o `RollCheck`.

> Manter as duas seria estado redundante que pode discordar de si mesmo. Estado assim sempre
> discorda, mais cedo ou mais tarde.

### O custo da reação na economia de barra

Quatro perguntas, porque a reação não passa pela fila e a economia inteira foi escrita para
quem passa.

#### (a) Que velocidade a reação registra

`AttachReaction` roda `rollActionDice` mas **não** roda `deriveSpeeds`: a reação rola dados e
nunca vira número. Para as grátis, correto. Para as que cobram barra, não — `ResourceBar.Speeds`
é a lista das velocidades que **agiram**, e é ela que a média divide pela contagem. Uma reação
que cobra o preço sem registrar velocidade faria o personagem pagar por uma ação a mais do que
a média enxerga.

**Decisão:** reação que cobra barra passa por `deriveSpeeds` como qualquer action e registra a
velocidade que **ela mesma** rolou, em cada barra que ela cobra. Reação grátis não deriva nada
e não registra nada.

#### (b) O porteiro NÃO se aplica à reação

Os dois porteiros decidem **quem age dentro do round** — é escalonamento. Reação não é
escalonada: ela acontece agora, em resposta, ou não acontece nunca. Não há "próximo round" para
onde ela role.

**Decisão: reação nunca é negada por falta de saldo.** Ela debita a barra e o personagem começa
o round seguinte mais atrás. É a mesma regra da segunda ação — *"a ação acontece de qualquer
jeito; o que a rolagem decide não é **se** você age, e sim quanto isso vai te custar depois"* —
e evita a pior experiência possível de mesa: o sistema informar que você não tem permissão para
se defender.

> Derivada de uma regra já dada, não dada diretamente. Se estiver errada, o lugar de corrigir
> é aqui.

#### (c) Cobra-se no attach, não no open

Action cobra no **open** porque uma action que nunca alcança o preço rola para o round seguinte
intacta — o `Speeds` precisa significar "agiu de verdade". Reação não tem para onde rolar:
ela existe dentro do turno em que foi anexada, e só.

**Decisão: debita no attach.** Três razões que apontam para o mesmo lugar:

1. É o momento em que a reação vira real — `AttachReaction` já chama `ResolveTurn` ali, e a
   colisão é calculada ali.
2. O princípio já está escrito: *"o cálculo é feito no momento que a action/reaction chega"*.
3. Abrir uma reação é um evento de **narração**. Narrar não pode mexer em número.

Consequência que fecha certo com a Fase 5: reação anexada e **nunca aberta** já pagou. É o
comportamento que o diálogo de encerramento pressupõe — *ela entra no cálculo, mas perde o
momento de narrar*.

#### (d) A reação consome a action pendente da barra que ela cobra

*"Reagir ativamente consome a ação que você tinha na fila"* — e consome mesmo: sai da
`activeQueue`. Se ficasse, o personagem reagiria **e** ainda agiria, e a Desvantagem viraria
punição pura, sem troca.

**Qual, se houver várias:** a que **abriria primeiro para aquele personagem naquela barra** —
a de melhor chave. É a que teve o momento gasto. O escalonador já sabe ordenar; a escolha não
inventa critério novo.

| Reação | Consome |
|---|---|
| Repelir | a próxima pendente na barra de **ação** |
| Escape fechado | a próxima pendente na barra de **movimento** |
| Escape padrão / defensivo | a próxima pendente **em cada** barra (ação combinada conta uma vez, e vai) |
| Esquiva fechada, esquiva, não fazer nada, passivas | **nada** — é o desconto se pagando |

Se não havia pendente, a reação **vira** a sua ação, sem Desvantagem. Se havia, a Desvantagem
entra pelo `RollInput` (§ *Onde mora o viés de uma rolagem só*) — e note que Desvantagem é
**modo de rolagem**, não `Amount`: `RollAttempts` já rola os dois conjuntos, e a Desvantagem só
escolhe qual ler.

#### Repelir que falha por 10 ou mais

O escape abre mão explicitamente da rede de segurança. Repelir é a reação mais difícil e mais
recompensadora do catálogo, e o degrau do aparar já entrega **dano zero** numa faixa em que a
defesa entregaria dano reduzido.

**Decisão: repelir também abre mão das passivas.** Se as passivas ainda valessem, o último
degrau da escada quase nunca morderia — a defesa amaciaria toda falha grande, e a escada
deixaria de ser uma escolha de risco. Você comprometeu a arma com o golpe que vinha; não está
também se abaixando.

> Também derivada, e a mais discutível das quatro. É uma regra de jogo, não de código.

#### O timer de reação não precisa de relógio na Fase 4

O padrão é **desligado**, e o comportamento no estouro é exatamente o de um caminho que já
existe: o mestre encerra o turno sem reação anexada, e o motor aplica os padrões.

**Decisão: a Fase 4 guarda o número na regra de partida e não implementa relógio nenhum.** Com
o timer desligado, **encerrar o turno *é* o estouro**. A contagem visível é do front (Fase 6);
se um dia o servidor precisar forçar, ele força chamando o mesmo encerramento.

### O que a Fase 4 precisa consertar em código já escrito

| Onde | O quê |
|---|---|
| `match.Scope` | só tem `end_of_turn` / `end_of_round`. Um bônus criado no turno N com `end_of_turn` morre no fim do **próprio** N — e o bônus do repelir vale **no próximo turno**. Falta o degrau. |
| `CharacterStatus.ExpireModifiers` | **não tem caller nenhum**. Nada expira hoje. Precisa ser ligado no fechamento de turno e de round. |
| `Modifier` | ganha `Applies Dimension`, e `AgainstID *uuid.UUID` vira `Against Scope` com três formas — todos / apenas X / **todos menos X** (§ *Modificadores*). |
| `RollInput.Ledger` (comentário) | ainda afirma que o acumulado *"is always an actionSpeed adjustment, never a hit adjustment"*. Era a invariante generalizada demais. Quem decide a dimensão passa a ser `Modifier.Applies`, não o caller. |

### A edição do mestre

**A action editada É a action.** O mestre não constrói uma versão paralela que alguém precise
mesclar na leitura: o valor dele entra na própria action, e todo consumidor — resolução,
projeção, histórico — lê um lugar só.

Isso não é decisão nova: é a que o código já tinha tomado. `RollCondition` mora em
`RollContext`, dentro do `RollCheck`, dentro da `Action`. O viés do mestre sempre entrou na
action.

⚠️ **O preço desse modelo:** o valor original é destruído no objeto vivo. *"O que o jogador
mandou"* deixa de ser algo que se lê e passa a ser algo que se **reconstrói**, aplicando de
trás para frente a tabela dos valores deslocados (abaixo).

#### `MasterAction` não é sobreposição — é a ação do mestre

Ela existe porque **o mestre também age**, em dois níveis: dentro de uma action de jogador
(alvos, perícias, velocidade) e **acima da batalha**, no que só ele conduz. O segundo nível
ainda não está escrito, e é ele que impede `MasterAction` de ser um caso particular de
`Action`.

> `buildMasterAction` (`action_mapper.go`) mapeia só parte dela — `Move` e `Attack` caem em
> `TODO`. É o **mapper** que está incompleto, não a entidade.

#### A corrente de testes

Cada `Skill` de uma action é um teste com CD própria (`Skill.Difficulty`), e eles resolvem
**em corrente**:

| De onde vem a CD | Quando |
|---|---|
| o resultado do adversário, por subtração direta | teste direto contra personagem — acerto × esquiva, dano × defesa |
| **o mestre, à mão** | todo o resto — se o personagem consegue mesmo o mortal que descreveu |

**A margem atravessa.** O que sobra de um teste entra no próximo — folga ajuda, negativo
desconta. **Errar por 10 ou mais mata a corrente**: a action falha e os testes seguintes não
acontecem. O mestre pode mudar essa margem ou deixar a corrente seguir mesmo assim.

> ⭐ É a terceira aparição da mesma forma. O saldo da barra atravessa rounds, o dano atravessa
> alvos na cadeia em área, e a margem atravessa testes dentro de uma action. **Propagação de
> margem** parece ser a ideia central deste sistema, não um detalhe de três regras separadas.

⛔ **Nada disso está em código.** `match_session.go` rola cada `Skill` e **ninguém lê o
resultado**; a única leitura de `Skills` em toda a colisão é a `Evasion` da esquiva fechada,
por nome. O que acontece com quem falha (guarda aberta, caído, dano igual à diferença) segue
em aberto por decisão do dono do produto — o sistema propõe um padrão, o mestre substitui.

#### Perícia removida: dados vivem na memória, nunca no banco

Enquanto o turno está aberto, os dados de uma perícia removida ficam guardados — senão tirar e
recolocar seria re-rolagem grátis. **Eles não são persistidos.** Um teste que não aconteceu
sujaria o histórico, e o histórico é superfície de jogo, não log.

Acrescentar perícia, ao contrário, **rola dados novos** — e isso não fere *"o mestre nunca
re-rola o dado de um jogador"*: não é re-rolagem, é a primeira rolagem de um teste que não
existia.

#### A edição muda o desfecho, nunca a economia

Barras cobradas, `Speeds` registradas e ordem já jogada não se refazem — mesmo quando a edição
mudaria o que `Bars()` responde (`chargesActionBar()` lê `len(a.Skills) > 0`). A economia é
artefato **público e sequencial**: `bars_updated` já foi ao ar. Refazer o preço reordenaria o
que já foi jogado.

#### Os valores deslocados

O que a auditoria guarda **não é a edição** — é o **valor que a edição descartou**. A action
já carrega o valor novo; guardar os dois seria duplicação que diverge.

Duas origens, um sobrescritor:

| O valor deslocado veio de | Quem sobrescreveu |
|---|---|
| cálculo do sistema | o mestre |
| envio do jogador | o mestre |

Uma linha por valor deslocado, não um retrato da action inteira: **o mestre edita um campo por
vez, pela tela**, então quem escreve já sabe o que mudou e não há diff a calcular — que era a
única vantagem do retrato. Editar o mesmo campo duas vezes deixa duas linhas, e ler de trás
para frente devolve o original.

Identidade em coluna (qual action, qual campo, quando, qual mestre); o valor deslocado em
`JSONB`, porque o formato varia de verdade — um inteiro, uma lista de perícias, um conjunto de
alvos — e ninguém vai consultar dentro dele.

⚠️ **O nome `SystemData` está reprovado** e o substituto ainda não foi escolhido. O problema é
semântico e é real: um nome genérico não consegue **recusar** nada, e vira depósito. O nome
precisa dizer *valores de action descartados por sobrescrita*.

## Visibilidade

- Mecânica da action (alvos, arma, perícia) é **pública** ao abrir; o **cálculo é só do
  mestre** até o turno encerrar.
- **Nem todo campo é público, inclusive no histórico.** Uma esquiva fechada não pode revelar
  que embutiu um teste de Evasão — o adversário precisa **deduzir dos números**.
  O Action History é uma **superfície de jogo com visibilidade por campo**, não um log.
- Exceção do percept: no **início de batalha**, só vê os alvos quem percebeu (Percepção vs.
  Furtividade). Quem não viu não recebe os updates daquela action. Bloqueado — os
  subatributos mentais ainda não existem.

### A política: público por omissão, deny-list explícita, mesa inteira

As duas metades vêm de regras que já existiam: **o dano é público, o HP não**, e *"o
adversário precisa deduzir dos números"*. Deduzir exige ver os números — então a omissão é
mostrar, e o que se esconde é lista fechada.

**Três classes de destinatário, não quatro:** mestre (vê tudo), **dono** da action ou reaction
(vê tudo o que é dele), e todo o resto (vê tudo menos a deny-list). **O alvo não é classe
privilegiada** — uma finta contra você não te conta que era finta.

| Oculto de terceiros | Por quê |
|---|---|
| HP | o dano é público, o HP não |
| `Feint` | uma finta revelada não é finta |
| `Trigger` | idem, até disparar |
| a entrada de `Evasion` na esquiva fechada, e a reserva que ela gera | o adversário deduz |
| **o próprio `ReactionKind`, nas variantes fechadas** | ⬇ |

⚠️ **O rótulo é o vazamento.** Se `closedDodge` chega público, ninguém precisa deduzir nada: o
rótulo já contou que havia Evasão embutida. Uma esquiva fechada chega aos terceiros
**indistinguível de uma esquiva**; um escape fechado, de um escape.

A dedução continua possível, que é o ponto: `bars_updated` é público, e o escape fechado cobra
**uma** barra enquanto o padrão cobra duas. Quem olha a barra percebe. **Deduzir da barra é
legítimo; ser informado não é** — a política inteira cabe nessa frase.

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
   entra como `nil` aqui: a diferença acumulada do duelo modifica `actionSpeed`, não o acerto.
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
fila — e **só em `Race`**. `Free` não congela preço nenhum e `settleBars` pula toda barra que
não precificou, então uma velocidade gravada em `Free` seria cobrada por nada e zerada por nada:
sobreviveria à troca de regime e faria `IsEligible` ler o personagem como quem "já agiu",
negando a **primeira** ação dele no round disputado. O porteiro fica dentro do próprio
`recordActed`, fechando para os dois chamadores de uma vez. `deriveSpeeds`, chamado por `EnqueueAction`, só calcula o número da velocidade; não o
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

⚠️ **`ProjectOrder` e `SelectNext` compartilham a pontuação inteira, não só o desempate.** Os
dois passam pelo mesmo `RoundScheduler.best` — a chave é `keyOf`, e o empate vai para quem
entrou primeiro (só troca o melhor quando a chave é **estritamente** maior).

E `ProjectOrder` **simula o round para a frente**: escolhe o melhor, registra numa cópia do
estado das barras que ele abriu, tira da lista e repete. Pontuar tudo de uma vez contra o mesmo
`acted` só acerta a **primeira** posição — a segunda ação pendente de um personagem seria
chaveada como se a primeira não tivesse aberto. No exemplo canônico (preço 11; p2 pendente em 23
e 17, p1 em 20, p3 em 11) a passada única publica p2 → p1 → p2 → p3, e a mesa joga
p2 → p1 → p3 → p2, porque a chave da segunda de p2 cai para `média(23,17) − 11 = 9` assim que a
primeira abre. Nada real é mutado: a fila entrega cópia e o estado das barras é lido por um
overlay que morre com a chamada, então `RoundScheduler{}` continua valendo como zero value.

### As barras são públicas

`bars_updated` (`BarsUpdatedPayload`, `internal/app/game/message.go`) é broadcast, não
projeção por destinatário: carrega `seq`, os preços congelados, o saldo e o histórico de
velocidades de cada personagem em ambas as barras, e a ordem projetada
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
- **`Feint`, cada `Skill`, `Move.Charge`, `Defense`, `Dodge`, `Attack.Hit`, `Attack.Charge` e
  o dano da arma** (a outra família de rolagem — os dados da própria arma, só `Primary`, sem
  vantagem) rolam **sempre que o campo correspondente existe na action**, em qualquer regime —
  nenhum deles depende de `Race`/`Free` nem de categoria de movimento. Uma action sem
  `Defense`, por exemplo, não sorteia nada por `Defense`; uma que o carrega, sorteia sempre.
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

⚠️ **Esta lista é da Fase 3 e não cobre reações — está incompleta hoje.** Na Fase 3, `Repel`
não tinha ramo nenhum em `rollActionDice`: repelir nunca sorteava dado. A Fase 4 fechou essa
lacuna (era um defeito, não uma omissão de design — ver "Repelir nunca rolava" abaixo) e
`Repel` entra no sorteio como `Defense`/`Dodge`. A lista completa, com a aritmética exata de
quantas faces cada peça consome, está em "Quantos dados cada reação consome", na seção da
Fase 4.

## O que a Fase 4 fixou no motor

### O tipo mora em `action.ReactionKind`, e `Bars()` responde pelo tipo

`entity/action/reaction_kind.go` traz os sete valores do catálogo (`nothing`, `dodge`,
`closedDodge`, `escape`, `escapeGuard`, `closedEscape`, `repel`), exatamente como fechado em
"O tipo da reação é declarado, não inferido" acima. `ReactionKind.Bars()` responde o custo
lendo o **tipo declarado**, não a forma do payload — é a decisão registrada ali chegando ao
código.

`Bars()` **pode devolver vazio**, e só para reações. A invariante antiga ("`Bars()` nunca é
vazio") continua valendo para `Action.Bars()`, porque o escalonador precisa precificar toda
action que agenda — e reação **não é agendada**: não entra na `activeQueue`, não é escolhida
por `RoundScheduler`. Vazio é a resposta correta para `nothing`, `dodge` e `closedDodge`, que
não cobram bar nenhuma. Nenhum caller quebra: os quatro métodos de `RoundScheduler` só veem a
fila, e `recordActed`/`chargeReactionBars` iteram o slice que `Bars()` devolve — iterar vazio é
no-op.

`ReactionKind.RequiredComponents()` e `Displaces()` vivem ao lado de `Bars()` no mesmo tipo,
não duplicadas em outro lugar: um kind sabe sozinho do que precisa para ser bem-formado e se
desloca ou não.

### `action.Repel` chega, `enum.DodgeCategory` sai

`Repel` (`entity/action/repel.go`) tem a mesma forma de `Defense` — arma e teste — porque é o
mesmo gesto lido contra uma escada diferente. A arma importa duas vezes: é com o que se repele,
e num quase-acerto (aparar) a defesa dela é o que reduz o golpe que segue para o próximo alvo.

`enum.DodgeCategory` foi **removida** — checado: nenhuma referência sobra no código. Era o
mesmo eixo do `ReactionKind`, só que estritamente menos expressivo (`Scape` sozinho não
distinguia os três escapes), e mantê-la ao lado seria estado redundante capaz de discordar de
si mesmo.

### A fronteira recusa reação sem o componente que o tipo exige

`action_mapper.go` lê `ReactionKind.RequiredComponents()` e confere cada componente contra o
que o payload realmente carregou — `dodge` exige `Dodge`, `repel` exige `Repel`, os escapes
exigem `Dodge` **e** `Move` (via `Displaces()`). Faltando um, o WS devolve erro
(`"reaction %q must carry a dodge"` etc.) **antes** de a reação chegar ao domínio.

Isso substitui o que a Fase 4 herdava: uma reação malformada não é mais derivada contra um
`RollCheck` vazio — o que silenciosamente virava o pior resultado possível para quem a mandou
(`Total = 0`, pior que a passiva que substituía, ou `RungFailure` garantido num repelir). Era
um bug de cliente disfarçado de resultado de jogo; agora é um erro de WS, no mesmo padrão que
uma categoria de movimento não suportada já usava.

### `service.ResolveReaction` — as sete linhas do catálogo, e a escada do repelir

`reaction_collision.go` deriva as sete reações, pura como tudo desde a Fase 2 — todo dado já
caiu na chegada (attach), `ResolveReaction` só lê `RollCheck.Attempts`. `ReactNothing` recusa
até as passivas e retorna cedo. `ReactRepel` vai para `resolveRepel`, que lê a escada de margem
que a Fase 2 escreveu como função pura e nunca ligou — é aqui que ela liga. As outras cinco
passam por `dodgeAndReserve`: reflexo sempre, Evasão só nas variantes fechadas, com o pior dos
dois contando como "a esquiva" e a diferença virando reserva.

### A cadeia: `ChainState`, `Reduce`, e a ordem de abertura do mestre

Com vários alvos a colisão não é `f(action, reactions[])` — é uma caminhada:
`ataque₀ → resolve(alvo A) → ataque₁ → resolve(alvo B) → …`. `service.ChainState` (`Residual`,
`Stopped`) é o que uma resolução deixa para a próxima, e `Reduce` aplica o desfecho de um alvo
sobre o golpe que segue: esquivou não gasta nada; repeliu para tudo; defendeu subtrai a defesa
da arma que defendeu; acertou subtrai a armadura do alvo. Parado não é cancelado — quem vem
depois de um repel bem-sucedido ainda resolve e ainda narra, só não pode mais ser atingido.

`Reduce` checa **`Defended` antes de `Avoided`**, de propósito: um quase-acerto no repelir
(`RungNearMiss`) é os dois ao mesmo tempo — dano zero para quem aparou **e** conta como
defendido para quem vem depois, porque é a arma de quem aparou que reduz o golpe adiante. Checar
`Avoided` primeiro deixaria o golpe seguir inteiro e a defesa da arma nunca faria o único
trabalho que lhe cabe nessa linha.

`buildChainOrder` é a ordem da caminhada em si: primeiro toda reação **aberta**, na ordem em
que **o mestre abriu**, depois quem sobrou, na ordem em que o ataque nomeou os alvos.
`Turn.reactions` é ordenado por **attach**, que não é a ordem de abertura — um alvo pode
anexar muito antes de o mestre chegar nele — e andar pela ordem de anexação em vez da ordem de
abertura destruiria a única coisa que esta fase entrega: que a ordem em que o mestre abre muda
o resultado. `Turn.OpenReaction` é idempotente por ID de reação, não por ator, então
`buildChainOrder` guarda um `covered` por ator dentro do próprio loop — sem ele, um segundo
`open_reaction` idempotente do mesmo ator produziria um segundo `chainStep` para o mesmo alvo:
um `CharacterResult` a mais, dano aplicado duas vezes, payout duplicado no fechamento.

⚠️ **Não existe regra rígida aqui — só o default por tipo de reação.** `Reduce` é comentado
explicitamente como a tabela padrão; o mestre pode sobrepor a qualquer momento, e essa
superfície de override (`SystemData`) é da Fase 5. Não a construa aqui.

### Cobrança no attach, nunca no open, nunca negada por saldo

`AttachReaction` roda `rollActionDice` na chegada. Se a reação **não** é grátis
(`!r.ReactionKind.IsFree()`), ela consome a action pendente, deriva a velocidade com o viés de
Desvantagem quando havia algo para trocar, e cobra a barra — tudo ali, não na abertura.
`chargeReactionBars` só grava em `Race` (pela mesma razão que `recordActed`: uma barra que
nunca precificou não tem o que cobrar), e nunca checa saldo antes de debitar: **reação nunca é
recusada por falta de barra**, ela só deixa o personagem mais atrasado no round seguinte. É a
mesma regra já dada para a segunda ação — o que a rolagem decide não é *se* você age, é *quanto
isso custa depois*.

⚠️ **Em aberto para o dono do produto — sem teste que cubra hoje.** Quando um personagem tem
**os dois** pendentes ao mesmo tempo — uma ação combinada (`Move`+`Attack`, uma `Action` só,
ocupando as duas barras) **e** uma move action separada — e reage com um escape (que cobra as
duas barras), `consumePendingFor` varre bar por bar, independentemente: para `BarAction` acha
e extrai a combinada; para `BarMove`, com a combinada já fora da fila, acha e extrai a move
separada que sobrou. Pelo código, isso consome **as duas**, não só a combinada — o que não é o
que a frase "ação combinada conta uma vez, e vai" (na seção "Reações" acima) parece prometer.
Essa frase é um parêntese, não uma regra escrita, e nenhum teste força esse cenário. **Pergunta
para o dono do produto:** o escape deveria consumir só a combinada, deixando a move separada na
fila? Até essa resposta existir, o comportamento é o que o código faz, não o que a frase supõe.

### `match.Modifier` diz o quê, contra quem, e por quanto tempo

Três campos novos substituem o que a Fase 1 tinha: `Applies Dimension` diz **o quê** —
`DimActionSpeed` (a reserva de duelo) ou `DimDodge` (a reserva da esquiva fechada), a mesma
distinção que fechou o "conflito no Bias" registrado no fim da seção "Reações" acima. `Against
Scope` diz **contra quem**, com três formas — `ScopeAnyone`, `ScopeOnly(id)`,
`ScopeAllBut(id)` — porque um ponteiro nulo-ou-um-alvo não consegue expressar "todo mundo menos
o duelista atual", que é exatamente a reserva da esquiva fechada. `ExpiresAt Lifetime` diz **por
quanto tempo**, e ganhou o degrau que faltava: `LifetimeNextTurn`, ao lado de `EndOfTurn` e
`EndOfRound`.

`next_turn` **não usa relógio nenhum** — é demoção. `ModifierLedger.AdvanceTurn` (chamado por
`MatchSession.advanceLedgers`, depois de `applyResolution`, no fechamento do turno) descarta
todo `EndOfTurn` e rebaixa todo `NextTurn` para `EndOfTurn`. Um bônus nascido no turno N fica
vivo até o fechamento de N+1: exatamente o intervalo que o bônus do repelir precisa. `Expire`
continua existindo para `EndOfRound`, chamado por `CloseRound`.

`ExpireModifiers`/`AdvanceTurn` **têm chamador agora** — a Fase 3 os deixou órfãos; a Fase 4 os
ligou nos dois fechamentos, turno e round.

### `open_reaction` / `reaction_opened`, e o `PendingReactions` que os liga

`open_reaction` é master-only, abre uma reação por ID e devolve `reaction_opened`
(`TurnID`, `ReactionID`) em broadcast — público porque de quem é a vez de narrar é público; o
cálculo continua não sendo, até a Fase 5. A projeção completa (`CharacterResults`, dano
projetado) só vai para o mestre, em `resolution_updated`.

`TurnResolution.PendingReactions` é o que fecha o ciclo: cada reação **anexada e ainda não
aberta**, com `ReactionID`, `ActorID` e `Kind`. Sem ela, `open_reaction` era inalcançável a
partir de um cliente de verdade — nenhuma mensagem publicava o ID de uma reação antes de ela
ser aberta. `CharacterResult.ReactionID` só é preenchido a partir do `chainStep`, e
`buildChainOrder` só produz `chainStep` para reação já **aberta** — de propósito, porque
arrastar uma reação não-aberta para a caminhada deixaria o mestre nunca ter dado a palavra a
ela antes de ela já ter afetado a colisão. `PendingReactions` é o único lugar que nomeia o ID
de uma reação anexada antes dela ser aberta — sem ele, era um ID que o cliente não tinha como
aprender, para uma operação que ele não conseguia invocar.

### Repelir nunca rolava — a reação mais difícil era a pior escolha possível

`rollActionDice` tinha ramo para `Speed`, `Feint`, cada `Skill`, `Move.Speed`, `Move.Charge`,
`Defense`, `Dodge`, `Attack.Hit`, `Attack.Charge` e o dano da arma — e nenhum para `Repel`.
`Repel.RollCheck.Attempts` ficava no zero value para sempre, `RollCalculator.Derive` totalizava
`perícia + 0` toda vez, a margem contra qualquer acerto real saía profundamente negativa, e
`ClimbLadder` sempre devolvia `RungFailure` — que é justamente o degrau que **também** abre mão
das passivas. Repelir não era só fraco: era estritamente a pior escolha possível, sempre, e três
dos quatro degraus da escada (`RungGreatSuccess`, `RungSuccess`, `RungNearMiss`) eram
inalcançáveis em produção, embora a escada em si estivesse correta desde a Fase 2. Nenhum teste
pegava isso porque todo teste de repelir existente anexava a reação sem componente `Repel` —
correto para o que aqueles testes checavam (contabilidade de barra), não a rolagem.

**Corrigido**: `rollActionDice` ganhou `if a.Repel != nil { test(&a.Repel.RollCheck) } }`, ao
lado do ramo de `Dodge` a que pertence.

### A reserva da esquiva fechada era escrita e nunca lida

`dodgeAndReserve` sempre bancou a reserva (`Dimension: DimDodge`, `Against:
ScopeAllBut(atacante)`) no ledger do alvo — mas `reaction_collision.go` nunca lia `in.Ledger`
de volta, e `resolveCharacterStep` nunca preenchia `ReactionInput.Ledger`. A mecânica inteira
descrita em "A escada de resultados" acima — a reserva de quem esquivou fechado valendo contra
quem vem de fora do duelo — não fazia nada.

**Corrigido**: `ResolveInput` ganhou `Statuses map[uuid.UUID]*match.CharacterStatus`
(read-only), `MatchSession.ResolveTurn` passa `s.statuses`, e `resolveCharacterStep` lê
`in.Statuses[step.targetID]` para popular `ReactionInput.Ledger`. `deriveReflex` **e**
`deriveEvasion` agora leem essa reserva — as duas, porque as variantes fechadas contam o pior
dos dois testes como "a esquiva", e a reserva precisa alcançar qualquer um dos dois que acabe
sendo esse.

### Armadura reduz zero

`ChainState.Reduce` subtrai `armour` da linha "acertou" — mas não existe entidade de armadura
no código, nem campo de ficha para ela, então `turn_resolver.go` declara `const armour = 0` e é
isso que entra na conta hoje. A linha está codificada porque a **forma** é o que importa,
exatamente como `ApplicableDefense` já codifica as linhas de tipo de dano que ainda não sabe
ler. Não construa um modelo de armadura para preencher isto — é trabalho de uma fase futura.

### Quantos dados cada reação consome

Como a Fase 3 fez para actions, mas para reações — a próxima pessoa escrevendo um teste
roteirizado precisa exatamente disto, ou vai achar por instrumentação como esta fase achou.

⚠️ **A armadilha real não é "quais campos rolam" — é que um teste 2D10 consome QUATRO faces,
não duas.** `RollCalculator.Roll` (chamado por `rollActionDice`, para **todo** teste do
conjunto de dados da partida — `MatchRules.DiceSet`, 2 D10) sempre sorteia **os dois
conjuntos**, `Primary` e `Secondary`, 2 dados cada, **mesmo quando nada usa Vantagem ou
Desvantagem** e só `Primary` acaba sendo lido. Um plano que reserva 2 faces por teste 2D10 vai
sortear a metade do que o código realmente consome. Só o dano da arma (`RollCalculator
.RollDice`, chamado à parte, fora do laço de `test()`) rola um único conjunto — 2 dados para
uma Espada (D10+D4), sem Secondary, porque dano não tem Vantagem.

Por reação:

| `ReactionKind` | O que rola | Faces (teste 2D10 = 4, sempre) |
|---|---|---|
| `nothing` | nada — recusa até as passivas | 0 |
| `dodge` | `Dodge` (Reflexo) | 4 |
| `closedDodge` | `Dodge` (Reflexo) + a perícia `Evasion` em `Skills` | 8 |
| `escape` / `escapeGuard` | `Dodge` (Reflexo) + `Move.Speed` se a categoria não for `Shift` | 4, ou 8 se `Move.Category == Dash` |
| `closedEscape` | `Dodge` (Reflexo) + `Evasion` em `Skills` + `Move.Speed` se não for `Shift` | 8, ou 12 se `Dash` |
| `repel` | `Repel` | 4 |

Um alvo que não anexa nem abre reação nenhuma (silêncio, ou `nothing`) não consome dado algum —
`resolveCharacterStep` aplica a passiva sem tocar a fonte de dados.

**Exemplo real** (`TestE2E_AreaAttackWithThreeTargetsReactingDifferently`): ataque de Espada
contra três alvos, A repele. O plano original previa 6 faces (`6,4,7,3,7,4`); o consumo real é
**10**: `Attack.Hit` (2D10, 4 faces) + dano da Espada (D10+D4, 2 faces, conjunto único) + o
`Repel` de A (2D10, 4 faces) = 10. B (`nothing`) e C (silêncio) não consomem nada.

### O caminho de escrita estava quebrado desde a Fase 2

`actions.actor_uuid` referenciava `users(uuid)`. Mas o `actorID` virou o **sheetUUID** na Fase
2, e a FK nunca acompanhou: **todo** `PersistTurnClose` de uma partida real falhava com
`23503`, e `room.go` loga e segue. Nada de partida nenhuma foi persistido desde então.

O teste de integração passava porque **ele mesmo** passava um user UUID como ator — testava a
semântica antiga. É o gotcha durável desta seção: um teste que constrói o dado errado não
prova nada, e não há como notar isso olhando só se ele está verde.

| O que estava errado | O que passou a valer |
|---|---|
| FK de `actor_uuid` → `users` | → `character_sheets` (`NOT VALID`, porque as linhas antigas não são legíveis por ninguém) |
| Reações nunca escritas | `PersistTurnClose` grava a action **e** `t.GetReactions()`, nessa ordem — `react_to_uuid` aponta para a action, na mesma transação |
| `ReactionKind` e `Repel` sem coluna | colunas `reaction_kind` e `repel` |
| `deriveActionType` inferia tipo de reação pela forma | reação nunca é classificada por forma: `type = "reaction"`, e `reaction_kind` diz qual |

A FK apontar para `character_sheets` é também o que permite persistir turno de **NPC** — uma
ficha sem `player_uuid`. Com a FK em `users`, isso era impossível por construção, e teria
mordido o rostering de NPC bem depois, longe daqui.

⚠️ **O resultado do turno continua não sendo persistido** — nem dano, nem margem, nem
desfecho, só a declaração. E recalcular depois é impossível: o ledger daquele instante não
existe mais. O Action History da Fase 5 precisa disso, e é **decisão de forma dela**, não
consertável aqui.

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
| Escada de margem | ✅ Fase 2 — `service.ClimbLadder` como função pura, sem reação ligada nela. ✅ Fase 4 — `resolveRepel` liga o repelir; os quatro degraus são alcançáveis (antes, `RungFailure` era o único possível) |
| Aplicação do dano na ficha | ✅ Fase 2 — dry-run em toda resolução, aplicado uma vez no fechamento do turno e persistido via `UpdateStatusBars` |
| `PriorityQueue` | ✅ Fase 3 — deixou de ser heap; virou lista simples, chave calculada em `RoundScheduler` na hora da seleção |
| `BarEconomy` / `RoundScheduler` | ✅ Fase 3 — preço por barra, média sem truncar, porteiro duplo (`IsEligible`), chave (`Key`), carry-over com teto (`CloseBalance`), projeção da ordem (`ProjectOrder`) |
| Fechamento do round | ✅ Fase 3 — `RoundScheduler.AnyEligible` nega, `OpenNextActionUC` chama `CloseRoundUC` (primeiro chamador que ele ganha), `room.go` emite `round_closed` |
| `RoundMode.Race` | ✅ Fase 3 — alcançável via `ChangeRoundModeUC`/`change_round_mode`, master only. Iniciativa continua fora — `action.Initiative` segue órfão |
| `bars_updated` | ✅ Fase 3 — broadcast com `seq`, preços, saldos/velocidades por personagem e a ordem projetada; nada que identifique a action |
| `ReactionKind` | ✅ Fase 4 — sete valores declarados no envio, `Bars()`/`RequiredComponents()`/`Displaces()` no próprio tipo; `enum.DodgeCategory` removida |
| `service.ResolveReaction` | ✅ Fase 4 — as sete linhas do catálogo, pura, mais a escada do repelir ligada |
| Cadeia com vários alvos | ✅ Fase 4 — `ChainState.Reduce`/`buildChainOrder`, andando por reação aberta na ordem do mestre, depois pelos alvos restantes. Override do mestre é Fase 5 |
| Custo da reação | ✅ Fase 4 — cobrado no attach, nunca no open, nunca negado por saldo; `match.Modifier` ganhou `Applies`/`Against`/`ExpiresAt: next_turn` (demoção, sem relógio) |
| `open_reaction` | ✅ Fase 4 — `open_reaction`/`reaction_opened`, e `PendingReactions` em `resolution_updated` (master-only) fecha o ciclo — sem isso o ID de uma reação anexada era inalcançável por um cliente real |
| Persistência do turno | ✅ Fase 4 (fix) — FK de `actor_uuid` para `character_sheets`, reações gravadas com a action, colunas `reaction_kind`/`repel`. **O resultado do turno segue fora** — Fase 5 |
| Armadura | reduz zero — não existe entidade nem campo de ficha; a linha está codificada em `ChainState.Reduce`, o valor não. Fase futura |
