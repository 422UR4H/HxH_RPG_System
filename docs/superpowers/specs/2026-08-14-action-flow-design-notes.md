# Fluxo de Actions — notas de brainstorm (EM ANDAMENTO)

> **Isto não é uma spec.** É o caderno da sessão de design de 2026-08-14, escrito
> incrementalmente para sobreviver a reinícios. A spec sai daqui quando o desenho fechar.
>
> Fontes: `fluxos.excalidraw` (caixa **Battle Running**, nível de produto — válida) e a
> descrição do dono do produto nesta sessão. A caixa **Match (Partida)** do mesmo desenho
> está **deprecada** (foi desenhada sobre MatchEngine/TurnEngine/ActionEngine, que a
> refatoração DDD-lite dissolveu). A faixa de lobby/"fluxo MVP" está desatualizada — ignorar.

## Vocabulário (fechado)

| Termo | Significado | Código |
|---|---|---|
| **Turno** | 1 action + suas reactions | `entity/turn.Turn` ✅ |
| **Round** | sequência de turnos | `entity/round.Round` ✅ |
| **Cena** | sequência de rounds | `entity/scene.Scene` ✅ |
| **Fila** | actions declaradas, ordenadas por actionSpeed; atravessa turnos | `action.PriorityQueue` ✅ |
| **CD** | classe de dificuldade, igual a D&D — alvo numérico do teste | `Skill.Difficulty` |

A espinha Cena → Round → Turno → Action sobreviveu inteira ao refactor. O que não
sobreviveu foi o miolo que calcula.

## Decisões fechadas

1. **Servidor autoritativo em tudo.** Todo cálculo roda no Go. O client do mestre envia
   inputs (CD, vantagem, edições) e renderiza. Nada de fórmula de jogo em TS.
2. **A action não carrega campo de status.** Onde ela está diz em que ponto do ciclo está:
   na fila = declarada; no turno aberto = em jogo; turno fechado = histórico. Marcas de
   tempo cobrem o que a posição não expressa (e dão o Action History de graça).
3. **Sorteio na chegada, resultado derivado quantas vezes precisar.** Os dados caem uma vez,
   no momento em que a action **ou a reaction** chega (obrigatório: a fila é ordenada por
   actionSpeed, e ordenar exige calcular). O número final — `dados + valor da perícia +
   vantagem/desvantagem + CD` — é refeito a cada edição do mestre e a cada reaction que
   chega. **O mestre nunca re-sorteia o dado do jogador.**
4. **O mestre sempre encerra o turno.** Um gesto só, com ou sem reaction.
5. **Não há confirmação de action.** Abrir já vale como aval; o mestre só toca na action se
   quiser mudar algo, e mexer recalcula e reavisa. O controle dele está no encerramento do
   turno, que continua sendo sempre seu. Um gesto a menos num fluxo que terá muitos.
6. **"Abrir" = passar o microfone.** A operação é de regência da mesa — "agora é a vez desta
   pessoa narrar". O cálculo é efeito colateral. Vale igual para action e para reaction.

## Descobertas estruturais

- **Reaction tem exatamente o mesmo ciclo que action**: enviada → mestre abre → balão sobe →
  dono narra. Logo "abrir" é uma operação genérica sobre qualquer coisa narrável.
- **Narração é paralela e fora do sistema.** Enquanto um jogador narra, outros setam ações
  e o mestre edita. O sistema não registra o texto narrado.

### Os três desfechos do lado do alvo

Não são dois. A diferença entre o primeiro e o terceiro é intencional e é regra de jogo:

| Situação | O que o motor faz |
|---|---|
| **Não envia nada** (ou estoura o timer) | Aplica as reações padrão: **esquiva por reflexo → (se falhar) defesa** |
| **Envia "não fazer nada"** | Recebe o ataque **cru** — sem esquiva, sem defesa. É uma escolha deliberada |
| **Envia uma reaction** | Colide action × reaction |

O **timer de reação é regra de partida** (configurável, pode estar desligada). Estourado o
tempo, cai no primeiro caso.

## Fluxo momento a momento

✅ existe · 🔨 falta construir · ❓ decisão em aberto

| # | Momento | Estado |
|---|---|---|
| 1 | Jogador clica no alvo → bottom sheet de action → escolhe arma/habilidade → escolhe perícias → envia | 🔨 **front inteiro** (não existe tela) |
| 2 | Back valida identidade (é participante, é o dono da ficha) | ✅ |
| 3 | Back valida regra (alvo existe, alcance, recursos) | 🔨 |
| 4 | Sorteia os dados e calcula actionSpeed | 🔨 |
| 5 | Enfileira por actionSpeed | ✅ estrutura; hoje todos entram com 0 |
| 6 | **Só o mestre** é notificado de que fulano setou sua ação | 🔨 |
| 7 | Mestre define CD dos testes antes de abrir | 🔨 |
| 8 | Jogador troca a action na fila → **Desvantagem** | 🔨 |
| 9 | Mestre abre a 1ª da fila (fecha o turno anterior) | ✅ |
| 10 | Todos veem as **mecânicas** (alvos, arma, habilidade); **só o mestre vê o cálculo** | 🔨 |
| 11 | Balão sobe no ícone → é a vez do dono narrar | 🔨 front |
| 12 | Jogador narra | fora do sistema |
| 13 | Mestre ajusta a action já aberta (troca perícia, remove teste, dá bônus/penalidade) — todos veem | 🔨 |
| 14 | ~~Mestre confirma a action~~ — **não existe**; abrir já vale como aval | decidido |
| 15 | Ícones de reação aparecem ao lado do(s) alvo(s) | 🔨 front |
| 16 | Alvo escolhe reação (a qualquer momento, até o mestre encerrar o turno) | ✅ parcial — hoje qualquer um pode |
| 16b | Timer de reação estoura (se a regra de partida estiver ligada) → padrões | 🔨 |
| 17 | Reações complexas abrem bottom sheet; escapes exigem slot do grid | 🔨 |
| 18 | Mestre **abre a reaction** → broadcast → balão sobe → dono narra | 🔨 |
| 19 | Sistema colide action × reaction | 🔨 **buraco central** |
| 20 | Mestre vê a colisão antes de confirmar; pode editar a reaction | 🔨 |
| 21 | Mestre encerra o turno | 🔨 |
| 22 | Resultado aplicado na batalha (dano, status, posição) | 🔨 nada altera ficha hoje |
| 23 | Todos veem o resultado do turno | 🔨 hoje só o mestre, e vem vazio |
| 24 | Turno vai pro histórico (round → cena) | ✅ parcial |
| 25 | Mestre abre a próxima action → volta ao 9 | ✅ |

## Catálogo de reações

| Reação | Efeito | Complexidade |
|---|---|---|
| **Esquivar** (reflexo) | Passiva: `reflexo + 11`. Sem sair da posição, sem penalidade. **Padrão** — sempre tentada primeiro. Se falhar, o jogador pode arriscar `reflexo + 2D10`. | simples |
| **Defender** | **Padrão — mas só depois da esquiva por reflexo.** CD = CD do ataque − 10 (defender é um degrau mais fácil que esquivar). Clicar = "não farei nada além disso". | simples |
| **Escapar** (scape) | Força a esquiva gastando a **ação de movimento**. Rola `esquiva 2D10 + aceleração (dash)` — **nunca o passivo 11**. Exige informar o slot de destino. ⚠️ **Abre mão da defesa padrão**: se falhar, toma **dano cheio**. | abre bottom sheet |
| **Escape defensivo** | Escapar mantendo a defesa como último recurso. É a única forma de ter rede de segurança depois de forçar a esquiva. | abre bottom sheet |
| **Esquiva fechada** | Em desenvolvimento. Ideia: atrelar a perícia **Evasão**. | abre bottom sheet |
| **Repelir** | "Atacar o ataque", geralmente com arma, para que ele não atinja. | abre bottom sheet |
| **Não fazer nada** | Recebe o ataque **cru** — recusa até os padrões. Não confundir com omissão. | simples |

Composição de uma action é livre: movimento + ataque + perícias atreladas. Exemplo real —
*saltar sobre o alvo dando um mortal e atingir pelas costas com a espada* = ação de
movimento (salto) + ação de ataque (espada) + teste de Acrobacia.

## Rolagem

- **Desvantagem**: rola duas vezes, fica o pior. **Vantagem**: o inverso.
  `RollCondition.Bias` (−1/0/+1, acumula) já existe no código para isso.
- Rolagem com vantagem/desvantagem usa **2 D10** por teste. ❓ *Confirmar se o teste normal
  é 1 D10.*
- Pós-MVP a quantidade e o tipo de dado viram **regra de partida**.
- Trocar uma action já enfileirada custa Desvantagem — o desenho é intencional: não
  compensa abandonar uma ação para fazer outra que exija muitos testes.

## Percept (início de batalha)

Regra geral: ao abrir a action, **todos veem os alvos**. Exceção: **início de batalha**,
onde só vê quem percebeu — teste de Percepção contra a Furtividade do atacante. Quem não
viu **não recebe os updates daquela action**. Detalhes ficam para depois do MVP.

## Resolução com vários alvos é uma **cadeia**, não um lote

Regra dada pelo dono do produto: *"em ataques em área, a primeira reaction pode influenciar
em como o ataque chega no segundo alvo — melhorando ou piorando a situação dele, e assim
progressivamente. Só é possível calcular isso após a resolução da primeira reaction."*

Consequência forte para o motor: **a colisão não é uma função de `(action, reactions[])`.**
É uma dobra sequencial — o estado do ataque sai alterado de cada resolução e entra na
próxima:

```
estadoDoAtaque₀ ─▶ resolve(alvo A) ─▶ estadoDoAtaque₁ ─▶ resolve(alvo B) ─▶ estadoDoAtaque₂ …
```

E como **o mestre escolhe a ordem de abertura**, ele determina o resultado. As reactions
chegam a ele **ordenadas por actionSpeed**, mas ele pode abrir na ordem que quiser — e isso
não é um detalhe de UI, é uma decisão de jogo com consequência mecânica.

Fluxo com vários alvos:
- Mestre abre **uma reaction por vez**; o dono narra a sua.
- Idealmente encerra o turno só depois de abrir todas.
  ❓ *Encerrar antes de abrir todas ainda não foi pensado — decidir se é permitido.*
- Quem não respondeu leva os padrões no encerramento.
- ❓ **Escolha do mestre**: revelar o cálculo **progressivamente** (a cada reaction
  resolvida) ou **só ao encerrar o turno** (tudo de uma vez). Provavelmente vira regra de
  partida ou preferência do mestre.

## Posturas (o diagrama de interseção)

Quatro domínios sobrepostos, e as ações vivem dentro ou nas interseções:

| Domínio | Ações |
|---|---|
| **move** | rolar, deslizar, mover furtivamente, salto baixo/rasante, acelerar→correr |
| **attack** | saltar, investir, arremeter, fintar, atacar `[proficiência]` |
| **dodge** (evasiva) | escape, esquiva acrobática, forçar esquiva, rolamento, reflexo `[passiva]`, evasão |
| **defense** | defender `[à mão — passiva]`, aparar/parry `[armado]`, esquiva defensiva (reflexo + defesa) |
| **∩ attack + defense** | **repelir** |

Regra de postura: o padrão é **"em guarda"** — pronto para ações defensivas ou evasivas.
**Atacar coloca o personagem na postura ofensiva nem que seja por um instante, e isso "abre
a guarda"** para ataques de adversários. Ataques-surpresa têm vantagem, e é por isso que
estar em guarda importa.

> Isto é exatamente o `match.CharacterStatus` que hoje está órfão no código (só comentário).
> A postura é estado de personagem que o motor **precisa** conhecer para calcular
> vantagem/desvantagem da ação seguinte.

## Passivo vs. ativo: um padrão geral

Aparece em pelo menos dois lugares, e provavelmente é uma abstração do motor inteiro:

| Teste | Versão passiva (valor fixo) | Versão ativa (rolada) |
|---|---|---|
| actionSpeed | **11** (quando `RoundMode == Free`) | 2 D10 (`Race`) |
| Reflexo (esquiva) | **reflexo + 11** | reflexo + 2 D10 |
| Defesa à mão livre | passiva | — |

> **O passivo é 11, não 10.** O desenho dizia `reflexo +10`, sem nenhuma regra que o
> justificasse (verificado: a única outra ocorrência de `+10` por perto é `brake+10`, regra
> de movimento, sem relação). Corrigido para **11** por decisão do dono do produto — e a
> correção conserta o design: com passivo 10 contra média 11, rolar seria melhor na média e
> ninguém usaria o passivo. Com **11**, rolar tem expectativa **zero** de ganho: troca-se
> certeza por variância. O jogador rola só quando precisa de sorte acima da média.

O desenho prevê
até avisar o jogador: *"posso notificar o jogador se seu personagem perder no reflexo — ele
tem a possibilidade de arriscar (reflexo + 2D10), ou de fazer outra coisa (forçar esquiva,
repelir). Posso rolar internamente outro teste para notificar se ele for PERDER mesmo
arriscando — daí ele pode receber o ataque ou fazer algo."*

## ⭐ A diferença acumulada — o motor econômico do sistema

**Esta é a mecânica central, e reorganiza tudo o mais.** Nas palavras do dono do produto:

> *"Esse sistema trabalha muito com essa ideia de **diferença entre testes e resultados**. A
> diferença costuma ser **cumulativa**, e isso é válido para quase tudo. O próprio turno é
> dinâmico por causa disso."*

O resultado de um teste **não é sucesso/falha**. É a **margem** — quanto passou ou faltou
para a CD. E essa margem **não se perde no turno**: ela vira crédito ou débito que atravessa
turnos e rounds.

### Como a margem circula

| Origem | Vira | Escopo | Quando vale |
|---|---|---|---|
| Repelir com sucesso grande | **bônus** = a diferença | **só contra aquele alvo** | próximo turno |
| Aparar (falhou em repelir por pouco) | **penalidade** = a diferença | **geral**, contra qualquer atacante | próximo turno |
| Diferença positiva na rodada | **bônus** | — | passa para o **próximo round** |

Repare na assimetria, que é intencional: **o bônus do atacante é específico daquele alvo; a
penalidade do defensor é geral.** Você aprendeu a leitura *daquele* oponente; mas ficar
desequilibrado te expõe a todo mundo.

### ⚠️ O bônus é de **actionSpeed**, nunca de acerto

Decisão do dono do produto, e ela é o freio do sistema:

> *"Não dar acerto, mas sim actionSpeed, garante que o outro personagem atacará mais rápido,
> mas não que terá diretamente mais chance de acertar."*

O acúmulo é **proposital** — quem é mais rápido vai atacar mais mesmo. Mas convertê-lo em
velocidade em vez de precisão traz dois ganhos:

1. **Emerge uma mecânica de duelo sem precisar programar duelo.** Dois personagens que se
   enfrentam acumulam velocidade um contra o outro e passam a trocar golpes num ritmo
   próprio, mais rápido que o resto da batalha ao redor. Ninguém declarou "duelo" — a regra
   produziu isso sozinha.
2. **O sistema fica menos apelativo.** Mais ataques dão mais chances de acertar, claro, mas
   indiretamente. O personagem mais fraco precisa de estratégia para sair da situação, e não
   fica simplesmente sem saída.

E o acúmulo **precisa** existir em algum lugar porque os turnos são dinâmicos: truncar a
diferença tornaria o turno menos dinâmico, que é justamente a graça.

A frase "a diferença é cumulativa e vale para quase tudo" **continua valendo** — para o
turno (actionSpeed) e para outros testes de perícia **quando há uma cadeia/corrente de
testes**.

### Onde a diferença acumulada NÃO mora

**`RollCondition` é a struct do mestre**, não do motor de acúmulo:

| Campo | Significado |
|---|---|
| `Bias` | vantagem/desvantagem **dos dados** (rolar duas vezes, pegar melhor/pior) |
| `Modifier` | bônus/penalidade que o **mestre atribui à mão** — ex.: *"+3 porque teve criatividade estratégica"* |

A diferença acumulada é outra coisa e mora em outro lugar. Candidatos levantados:
`RollContext`, `CharacterStatus`, ou nem ser armazenada — ser derivada da action anterior.
**Decisão técnica delegada; a resolver no desenho da implementação.** Preferência declarada:
o caminho mais simples.

### A ActionBar — a economia do round

Fonte: descrição do dono do produto + caixa do desenho em `(-1989,-2136)` e `(2876,-2333)`.

**Regras:**

1. **Cada action tem a sua própria actionSpeed**, rolada quando a action chega (somada à
   iniciativa, quando houver).
2. **O personagem só tem direito a uma actionSpeed por round.** Se agir mais de uma vez, a
   actionSpeed do round é a **média** das actionSpeeds das suas actions naquele round.
3. **O preço de agir é o mesmo para todos: a menor actionSpeed do round** (`ActionBarCoast`).
4. Ao abrir uma action, o preço é **subtraído da barra**. O que sobra é bônus; se ficar
   negativo, é penalidade.
5. **A barra não é a actionSpeed da próxima action** — ela só indica **quem age em seguida**.
   A próxima action rola a sua própria.
6. Quem tem a menor actionSpeed **age por último e começa o próximo round em zero**.
7. O que sobrar na barra ao fim do round **atravessa para o próximo**, como bônus ou débito.

### A fórmula (fechada)

> **barra ao fim do round = média(actionSpeeds das suas actions) − nº de actions × preço**

**Exemplo** — p1=20, p2=23, p3=11, preço = 11. O **round 0** é o único sem ninguém narrando:
é o round em que todos setam a primeira action.

| # | Quem | Base | Conta | Resultado |
|---|---|---|---|---|
| 1º | **p2** | 23 | age | sobra 12 ≥ 11 → **ganha direito a agir de novo** |
| — | p2 | — | manda a 2ª action, que rola a **sua própria** actionSpeed (ex.: 17) | média = (23+17)/2 = **20** → barra recalculada: 20 − 11 = **9** |
| 2º | **p1** | 20 | age | 20 − 11 = **+9** ao próximo round |
| 3º | **p3** | 11 | age | 11 − 11 = **0** — começa o próximo round zerado |
| 4º | **p2** | 9 | age a 2ª vez | 20 − 2×11 = **−2** → começa o próximo round **devendo** |

**A posição é recalculada, mas nada é desfeito.** A primeira action de p2 já aconteceu e
continua tendo acontecido; o que muda é **onde a segunda cai na fila**. Como a média puxou p2
de 12 para 9, ele passa a agir **depois de p3** — que tinha 11. Só o futuro se move.

**Agir de novo é uma aposta.** O jogador decide mandar a segunda action **antes** de saber o
quanto ela vai rolar. Vindo fraca, ela puxa a média para baixo, atrasa a própria segunda ação
e ainda pode deixá-lo em débito no round seguinte. É isso que impede o acúmulo de virar
espiral — sem teto artificial nenhum.

### ⭐ São DUAS barras por personagem, não uma

Explicitado pelo dono do produto e não estava claro nas notas até aqui:

| Tipo de action | Exemplos | Barra |
|---|---|---|
| **Genérica** | ataque, puxar algo da mochila, usar item, habilidade | barra de **actionSpeed** |
| **Movimento tático** | shift, dash, salto, rolamento | barra de **moveSpeed** |

Cada uma tem a **sua própria barra**, alimentada pela sua própria rolagem. Bate com a nota do
desenho: *"o sistema define o custo final das actions (seja geral ou por barra de moveSpeed e
actionSpeed)"* e com *"em cada turno um personagem pode realizar 1 move action e +1 action
(genérica)"*.

❓ Em aberto: cada barra tem o **seu próprio preço** (o piso do round calculado
separadamente para movimento e para ação)?

### A reaction dentro da economia

**Reação passiva é grátis; reação ativa custa a action.**

| Situação | O que acontece |
|---|---|
| Esquiva por reflexo, defesa padrão | passivas — não custam nada |
| Reação ativa **sem** action enfileirada | a reaction **é** a action dele; entra normalmente |
| Reação ativa **com** action já enfileirada | **Desvantagem**, igual a trocar de action |

**Como a Desvantagem funciona na conversão:** a actionSpeed é rolada de novo (para a
reaction), e vale a **pior das duas** — a da action original ou a da reaction. Se a da
reaction for menor, usa-se ela; se a da action for menor, registra-se o valor da action e
usa-se ele na reaction.

⚠️ **Aqui não há média.** A média vale para múltiplas *actions* no round; a conversão
action→reaction usa **Desvantagem** (pior das duas).

### Esquiva fechada — a referência é Kuroro × Zeno e Silva

O dono do produto ancora o desenho nessa luta, e ela define dois itens do catálogo:

- **Escape defensivo** — Silva pula para trás escapando da lâmina envenenada de Kuroro, mas o
  corte ainda pega o braço. Escapou *e* defendeu; o dano passou reduzido.
- **Esquiva fechada** — Kuroro esquiva do *Dragon Head* de Zeno esperando até o último
  instante, **de propósito, para não abrir brecha** para Silva atacar. Silva espera uma
  abertura que nunca vem, e só depois percebe que é intencional.

> *"O sistema precisa suportar esse tipo de criatividade e estratégia nas batalhas."*

**Custo:** por ser "fechada" — feita no momento exato, sem abrir guarda — **não custa a action
genérica**. Um *escape* fechado custa a **action de movimento**, como qualquer escape, mas
não a genérica. É uma esquiva mais difícil que a normal justamente por não deixar brecha;
exige confiança nas próprias habilidades, e por isso não carrega penalidade quando
bem-sucedida. Combina esquiva + **Evasão**.

❓ Ambiguidade a confirmar: "escape fechado" e "esquiva fechada" são a mesma coisa ou dois
itens distintos (um com deslocamento, outro sem)? A frase *"o escape fechado tem menos
ainda"* sugere que são dois.

**Observação de design:** a cena só é jogável porque duas regras já fechadas se encaixam —
ficar parado é legítimo e não punido (Silva esperando a brecha), e o **Action History** é
público. Silva não faz um teste para descobrir a intenção de Kuroro: ele **lê o padrão** das
ações anteriores. Isso faz do histórico não um log, mas uma **superfície de jogo**.

### Esquiva fechada ≠ escape fechado (resolvido)

São **coisas diferentes**:

| | O que é | Custo |
|---|---|---|
| **Esquiva fechada** | ação de esquiva com a perícia **Evasão** embutida. Sem deslocamento. | não custa action genérica |
| **Escape fechado** | escape (com deslocamento) executado "fechado" | custa a barra de **movimento**, não a genérica |

No front: **clicar** na reaction envia direto; **clicar e segurar** abre o bottom sheet para
customizar. Esquiva fechada = segurar em Esquiva e adicionar Evasão.

No diagrama do desenho, `forçar esquiva`, `escape`, `rolamento`, `esquiva fechada` e
`escape defensivo` estão todos na **interseção entre `dodge` e `move`** — e há seta de
`acelerar → correr (move)` entrando nessa interseção.

### Como a reaction funciona no front

Ao ser alvejado, aparecem **botões** ao lado do personagem para decidir **se** e **como** reagir.

| Gesto | Efeito |
|---|---|
| **Clicar** em Esquiva ou Defesa | reaction enviada direto, sem configuração |
| **Clicar e segurar** | abre o bottom sheet de action para customizar a mecânica (a narração vem depois) |
| **Clicar e segurar em Scape** | bottom sheet já vem com a perícia **Accelerate** pré-setada |

**A perícia é o que define o escape** — sem deslocamento, é apenas uma esquiva padrão. No
bottom sheet dá para trocar:

| Perícia | Deslocamento tático | Comportamento |
|---|---|---|
| **Accelerate** | **Dash** | rápido; durante o dash o personagem está "no ar" e **não consegue esquivar** — fica exposto |
| **Brake** | **Shift** | movimento controlado; **não rola dado — usa o valor base 11** (ou 10 se a partida usar D20) |

> **No escape, o movimento precisa ser Shift**, justamente porque durante o Dash o personagem
> não consegue esquivar.

**Por que Brake é a perícia base do Shift:** ela define a capacidade de frear. Se o
personagem acelera ao máximo e esse máximo está além do seu brake, ele não consegue frear de
uma vez — *é rápido, mas não ágil*. Se freia num instante, brake e accelerate estão
equilibrados.

**Cânone:** ficar exposto durante o próprio ataque é o instante em que Gon — sem saber o que
é Nen, mas já dominando Zetsu — rouba a plaqueta de Hisoka no Exame Hunter, exatamente
enquanto Hisoka atacava outro personagem.

### O ganho das esquivas não-padrão

Esquiva fechada e escape fechado são **propositalmente trabalhosas de configurar**. O que
elas dão em troca:

> **Bônus para reagir a qualquer action vinda de um terceiro** — alguém de fora do duelo.

Mecânica: a rolagem de **Evasão** nesse contexto **não soma** à esquiva ou ao escape — ela
entra na **lógica de Desvantagem** (vale a pior das duas). E **o bônus é exatamente a
diferença entre os dois valores**.

Ou seja: o personagem esquiva **sem usar seu máximo**, e converte a esquiva "desnecessária"
em reserva contra qualquer terceiro que tente pegá-lo num instante de guarda aberta. É
literalmente a estratégia do Kuroro.

**Cânone (o desfecho):** Kuroro é pego exatamente quando Silva percebe que ele esquiva no
último instante sem deixar tempo para atacar, e decide ir com tudo — usa o Ren com todo o
poder. Ao fazer isso, chama a atenção de Kuroro por um instante, o que abre a brecha para
Zeno atacar com o dragão e capturá-lo.

### ⭐ Nem todo campo de uma action é público

Insight novo, e ele muda o desenho do histórico:

> *"Não necessariamente todos os campos das actions são públicos. Podem ser depois que a
> partida terminar, mas durante ela não — inclusive no histórico."*

**Exemplo canônico:** a action de escape fechado **não pode revelar** que houve um teste de
Evasão embutido. O jogador do Silva precisa **deduzir dos números** que as esquivas de Kuroro
vieram só um pouco acima dos acertos de Zeno. Kuroro não quer ser atingido, mas está
arriscando de propósito — e para isso **precisa estar em postura fechada**.

Consequência: o **Action History** é uma superfície de jogo com **visibilidade por campo**,
não um log público. A dedução tem que ser possível; a leitura direta, não.

### `SystemData` — auditoria de interferência do mestre

Existe no desenho, **não existe em código**. Toda interferência do mestre (edição,
gerenciamento, bônus concedido) é persistida numa tabela de auditoria.

**Por quê:** *"auditar as interferências dos mestres se for necessário, caso jogadores
reclamem de 'roubo' ou coisa do tipo."* Pouco importante no início; **essencial** quando
chegar a **Mesa Livre** (abaixo).

⚠️ **Tensão de modelagem a resolver:** o mestre pode **anular a Desvantagem** que o sistema
aplicou. O campo natural para isso é `RollCondition.Bias` — mas já ficou definido que `Bias`
é o campo *do mestre*. Ou seja, a desvantagem gerada pelo sistema e o ajuste do mestre
disputariam o mesmo campo. **Decidir onde cada uma mora.**

### Horizonte: Scenario e Mesa Livre (contexto, fora de escopo)

A cadeia hoje é `Match → Campaign`. No futuro entra o **Scenario**:
`Match → Campaign → Scenario`. O cenário é **compartilhado**: vários mestres criam campanhas
dentro dele, e os jogadores deixam de estar presos a uma mesa.

**Mesa Livre:** um jogador pode se juntar a outros jogadores e outro mestre, e transitar
entre as mesas que quiser — desde que estejam no **mesmo cenário e na mesma linha temporal**.

É o que torna a auditoria (`SystemData`) importante: com jogadores circulando entre mesas, a
reputação de um mestre passa a importar.

### O mestre pode perdoar o custo

> *"Pode ser importante existir um controle para o mestre decidir, em última instância, se
> essa reaction vai MESMO custar a action do jogador. Dependendo da criatividade narrativa do
> jogador, ele pode reagir de forma a criar espaço para sua própria action — e o mestre pode
> permitir que ele ainda a realize, e até antecipá-la, passando na frente das outras."*

Antecipar uma action já existe em código: é o `pull_action` / `PullActionUC`. ❓ O que não
existe é a **consequência** — o desenho antigo previa mover todas as outras actions para o
próximo round, e o dono do produto está **em dúvida se isso é bom**.

### As duas barras

| Barra | De quem | O que mostra |
|---|---|---|
| **Geral** | da partida; explícita para o mestre | linha do tempo de **quando cada um age**, respeitando progressão e proporção — incluindo as ações múltiplas de um mesmo personagem |
| **Individual** | por personagem | o actionSpeed que "vazou" — o que será creditado ou cobrado no próximo round |

### Teto do carry-over (fechado)

**O carry-over é limitado ao preço do round.** Quem não agiu carrega o piso daquele round.

Foi levantada a objeção de que isso premia ficar parado (rolagem 20, preço 11: quem age
carrega 9, quem fica parado carrega 11). **A objeção não procede**, por dois motivos:

1. **Quem ficou parado já está atrasado em ação.** Perdeu um turno inteiro de jogo — o que
   vale muito mais que 2 pontos de tempo. É uma troca, não um ganho: tempo por ação.
2. **A vantagem dura um round e some.** Quem age também converge para o teto em poucos
   rounds (9 → 20+9−11 = 18 → limitado a 11). Não existe estoque: ficar parado dois rounds
   seguidos não rende nada além do piso.

Leitura narrativa, e é a intenção declarada: **ficar parado lendo a luta compra reflexo um
pouco mais rápido no round seguinte.** *"Se ele tivesse agido, teria gastado mais tempo,
então tudo bem 11 em vez de 9."*

### ⭐ Depois do round 0 não existe mais fase de coleta

> *"Após o round 0 nunca mais haverá espaço para os jogadores enviarem suas actions. Eles
> precisarão fazer isso em runtime mesmo, ao longo dos turnos de outros jogadores. A ideia é
> que os jogadores fiquem espertos: já ter em mente a próxima action logo após finalizar a
> atual, quase sempre com uma action enfileirada."*

**O round 0 é o único momento de coleta da batalha inteira.** Dali em diante a fila é
permanentemente viva e a batalha nunca para para esperar ninguém. Quem não pensa à frente
simplesmente não age — e carrega o piso.

É a tese anti-latência levada ao limite: a pressão de tempo **é** o mecanismo.

Consequência de interface: a **barra geral é visível para os jogadores**, não só para o
mestre. Se o jogador precisa compor a próxima action *durante* os turnos alheios, ele precisa
enxergar quanto tempo tem. (Era ponto em aberto; fechado como visível.)

⚠️ Correção registrada: **o balão só sobe se o personagem enviou action.** Não existe "o
balão avisa que chegou sua vez" para quem não enfileirou nada.

**A regra do 2× não é uma regra separada.** *"Se alguma iniciativa for 2× maior que a menor
do turno, o servidor notifica o dono do personagem que pode enviar outra action"* é apenas a
consequência aritmética de o preço ser o menor: se você tem 2× o menor, depois de pagar uma
vez ainda sobra o suficiente para pagar de novo. Uma regra a menos no sistema.

**Fim do round:** *"quando a barra de ação de todos os jogadores acaba, o sistema pode rolar
iniciativa novamente para a batalha continuar naturalmente."*

### ⚠️ Dois custos diferentes — não confundir

| Custo | O que é | De onde é subtraído |
|---|---|---|
| **`ActionBarCoast`** | o preço do round = a menor actionSpeed da rodada | da **barra** |
| **`actionCoast`** / `actionSpeedCoast` | vem do **Peso Excedente** (carga que o personagem leva além do próprio peso) | da **actionSpeed** da própria action |

E `actionSpeed` final ainda sofre: soma do `moveSpeed` se for arremetida/investida, e cálculo
trigonométrico sobre `targetSpeed` (vetor) se o alvo estiver em movimento.

### O piso da régua — duas formulações do mesmo medo

O medo é o mesmo: se o mais lento tirar um número muito baixo, o preço fica baixo e o mais
rápido age muitas vezes. Existem duas formulações:

- **No desenho** — limite sobre a própria barra. *"Racional do limite de 1/3 da barra…
  utilizando no mínimo 1/3, permitindo mais de uma action por personagem no turno, quem
  ataca apenas 1× tem a sensação mais neutra, enquanto quem ataca mais vezes tem uma sensação
  gratificante."* Considerou 1/2 (só 1 action) e 1/4 (até 3 actions). Concluiu: *"ainda não
  estou achando interessante esse limite fixo… é melhor deixar aberto para o mestre no
  futuro."*
- **Nesta sessão** — piso sobre o preço do round: *"a menor actionSpeed permitida é 1/3 da
  maior actionSpeed do round."*

Para o personagem mais rápido as duas dão quase o mesmo teto (≈3 actions). ❓ Escolher uma.

### Visibilidade da barra

> *"É importante que os jogadores consigam ver/saber a sua própria action_bar, pelo menos.
> Eles precisam saber quanto foi o custo da sua ação para decidirem se já enviam a próxima.
> Seria interessante eles irem vendo as actions de quem vai agindo também."*

E ao abrir uma action, *"todos veem a intenção e a sua posição na action_bar"*. Note que isso
**não conflita** com "só o mestre é notificado de quem setou ação": o que é secreto é a fila
antes de abrir; a barra e a ordem são públicas.

### O turno dinâmico sai daqui

> *"Um jogador que tira grandes números numa rodada **pode atacar mais de uma vez na
> rodada** — se for **2× ou mais** que o jogador que tirou o menor número da rodada. E toda
> a diferença positiva é passada para o próximo round como bônus."*

Isso bate com a nota solta do desenho: *"se alguma iniciativa for 2× maior que a menor do
turno, o servidor notifica o client do dono do personagem que pode enviar outra action"*.

**Consequência para o motor:** existe um **livro-razão de bônus e penalidades por
personagem**, parte dele endereçado a um alvo específico, parte geral, com validade de
turno ou de round. É mais um pedaço do `CharacterStatus` órfão.

### A escada de resultados

**O degrau de 10 é o padrão do sistema, mas não é universal.** A razão é de design, não de
matemática:

> *"É importante para duelos serem mais flexíveis — conseguir aparar um ataque mesmo
> perdendo por pouco, mas acumulando desvantagem para o próximo turno. Caso contrário um
> jogador um pouco mais fraco receberia muitos golpes e o duelo ficaria menos dinâmico e
> divertido. **Deve ser mais fácil aparar do que acertar o alvo.** Este sistema já é muito
> punitivo, então isso entra para equilibrar a diversão."*

**Repelir** (CD = resultado do ataque):

| Margem | Desfecho |
|---|---|
| `≥ CD + 10` | Não recebe dano **e** ganha bônus = a diferença, para atacar **aquele alvo** no próximo turno |
| `CD … CD+9` | Não recebe dano |
| `CD−10 … CD−1` | Falhou em repelir mas **apara**: **dano zero**, e leva penalidade = a diferença no próximo turno (contra qualquer um) |
| `< CD − 10` | Recebe o ataque |

⚠️ Aparar **não é dano reduzido — é dano zero**. O preço é a penalidade acumulada.

## Economia de turno

- **Por turno, um personagem faz 1 move action + 1 action genérica.** Podem ser enviadas
  juntas.
- **Forçar esquiva**: não se moveu → sem penalidade e ainda usa sua action. Já setou o
  movimento → desvantagem para trocar de direção, **ou** vantagem se aproveitar a direção já
  escolhida. Já se moveu → sacrifica a action.
- **Repelir**: não agiu → sem penalidade e ainda usa sua action. Já setou a action →
  desvantagem para trocar a trajetória, ou vantagem se aproveitá-la. Já agiu → só repele se
  a trajetória fizer sentido, e pode exigir outros testes.
- **Ataques consecutivos "roubam turno"** e custam guarda aberta.

## Ainda não modelado (ciência, não escopo agora)

Levantado pelo dono do produto: **as actions ainda não existem de forma consolidada.** O
desenho tem muito material bruto que não entrou nestas notas: mecânica de movimento
(accelerate/brake/charge, fórmula de curva `f(v,x) = vel·|sen(x)|`), arremetida × investida,
finta com direcionalidade, golpe simultâneo e combos, tipos de dano (concusivo/cortante/
perfurante/ultra perfurante), consumo de energy, desengaje. Fica registrado que existe —
não entra no recorte agora.

## Questões em aberto

1. ❓ Encerrar o turno **antes** de abrir todas as reactions — permitido?
2. ❓ Revelar o cálculo progressivamente ou só ao encerrar — regra de partida ou botão do
   mestre a cada turno?
3. ❓ Como o mestre é avisado do resultado sem que a mesa veja — canal separado ou o mesmo
   evento com campos filtrados?
4. ❓ Bônus/penalidade acumulados: duram exatamente um turno/round ou até serem gastos?
   Vários bônus contra o mesmo alvo somam? Bônus e penalidade se anulam?
5. ❓ A regra do 2× é medida sobre actionSpeed? Comparada com o menor da rodada em que
   instante — se a rodada ainda está recebendo actions, o menor pode mudar.

## Percepção mental (bloqueado por falta de atributo)

O alvo só é **notificado** de que o ataque vai acertá-lo se **perceber** — teste de percepção
mental nos bastidores. Mas os **subatributos mentais ainda não existem** no sistema: hoje só
há os atributos mentais, que não abarcam percepção mental (descrita como um atributo
intermediário entre **ponderação** e **adaptabilidade**). Bloqueado; destrinchar depois.

> O diagrama tem `REL (resilience) / CRE (creativity) / ADP (adaptability) / WEG (weighting)`
> — ADP e WEG parecem ser adaptabilidade e ponderação. Confirmar quando chegarmos aqui.

## Flexibilidade exigida

Aviso explícito do dono do produto sobre as regras de reação padrão: *"essas regras não podem
estar em uma estrutura muito rígida para mudar, porque pode haver alterações após o MVP."*
Vale para toda a escada de resultados e para a cadeia esquiva→defesa.

## Regra de rolagem (fechada)

- **Teste padrão = 2 D10 somados.** Vale para perícia, acerto e actionSpeed.
- **Crítico** = ambos os dados saírem 10. **Erro crítico** = ambos saírem 1.
  (Note que isso não é o mesmo que "somou 20" ou "somou 2" — é a combinação, não o total.
  Logo o motor precisa guardar **os dados individuais**, não só a soma.)
- **D20 é opção secundária**, que entrará como regra de partida. Não é o padrão.
- **Vantagem/desvantagem**: rola o conjunto duas vezes e fica com o melhor/pior.
  `RollCondition.Bias` (−1/0/+1, acumula) já existe no código para isso.
- **`dado base 11`** (do desenho antigo) **não é a média de nada** — é a actionSpeed usada
  quando **`RoundMode == Free`**. Em `Race`, actionSpeed é rolada (2 D10).

  **Quem manda é o `RoundMode`, não a `SceneCategory`.** Decidido. Consequências:

  - **Iniciativa só é testada em `Race`, nunca em `Free`.** Pedir iniciativa **força** o
    round para `Race` se ele ainda não estiver.
  - Ligar `Race` dentro de uma cena de Roleplay é "praticamente uma porta para uma cena de
    Batalha" — mas a virada da cena é **decisão do mestre**, não automática. Se ele não
    virar, o round `Race` fica salvo numa cena de Roleplay, e tudo bem.
  - Logo `SceneCategory` e `RoundMode` seguem sendo eixos independentes, exatamente como o
    desenho pedia. `SceneCategory` é organização narrativa; `RoundMode` é regime de motor.

⚠️ `docs/game/dados.md` diz "D20 — testes gerais, testes de habilidade". **Está
desatualizado** e precisa ser corrigido junto com a implementação.

## Regras de partida (estrutura agora, conteúdo depois)

Requisito explícito: **não precisamos das regras agora, mas a estrutura precisa comportá-las
desde já.** Cada uma tem um padrão sensato embutido e vira configurável depois:

| Regra | Padrão do MVP |
|---|---|
| Conjunto de dados do teste | 2 D10 somados (D20 como alternativa) |
| **Valor médio do dado** (usado nos testes passivos) | **derivado do conjunto de dados** — 11 para 2 D10 |
| Timer de reação | desligado |
| Reação padrão se aplica na omissão | sim (esquiva por reflexo → defesa) |
| `fog_mode` | `explored` |

⚠️ **O valor médio não é digitado à mão — é derivado do conjunto de dados configurado.** Se a
partida trocar 2 D10 por D20, o valor médio tem que acompanhar automaticamente, senão os
testes passivos ficam calibrados para dados que não estão mais em uso. Uma configuração
manual por cima (ex.: a comunidade preferir 10 em vez de 11) pode existir depois, como
sobrescrita explícita.

**Motivação declarada para a configurabilidade:** *"algumas regras de batalha serão
configuráveis dentro desse contexto que a comunidade não gosta muito de alguma coisa, então
eu flexibilizo. Claro que não dá para fazer isso com tudo, mas precisamos de uma arquitetura
capaz de suportar isto. É um trade-off. Não gostaria de um código super complexo para isso."*

As duas do meio são **regras relacionadas mas independentes**: dá para ter timer sem padrão
automático, e padrão automático sem timer.

> **Convergência a explorar:** o mecanismo de configuração de partida **ainda não existe** no
> backend, e `fog_mode` já esbarra nisso (ver "Known Issues" em `AGENTS.md` — hoje está
> hardcoded em `room.go`). São o mesmo buraco. Vale resolver uma vez.

## Invariante de recálculo

**Todo momento em que o mestre edita ou altera qualquer coisa exige recálculo** — da action,
da reaction e da colisão. Sem exceção, e sem novo sorteio. Isso vale para: trocar perícia,
remover ou adicionar teste, definir ou mudar CD, conceder vantagem/desvantagem, mudar alvo.
