# Barra de Ação e o Turno Dinâmico

> Como o sistema decide quem age, quando, e quantas vezes.

## O problema que isso resolve

Em RPG de mesa, boa parte do tempo é gasto **esperando**. Espera o mestre lembrar de quem é a
vez, espera o jogador decidir o que fazer, espera alguém somar dados.

Aqui não existe fase de espera. **Você declara sua ação a qualquer momento**, o sistema
calcula em paralelo, e o mestre vai abrindo as ações na ordem certa. Enquanto um jogador
narra, os outros já estão montando o que vão fazer.

## Round 0 — o único momento de calma

No começo da batalha todo mundo declara sua primeira ação. Esse é o **round 0**: o único
round em que ninguém está narrando ainda.

**Depois dele, nunca mais existe um momento reservado para declarar.** Você monta suas
próximas ações **durante os turnos dos outros**. A ideia é estar sempre com a próxima ação
já engatilhada.

Se você não declarar nada, seu personagem simplesmente não age naquele round. Não é um
castigo — às vezes é a jogada certa. Mas é uma escolha.

## Velocidade da ação

Cada ação que você envia rola a sua própria **velocidade de ação**. É ela que define sua
posição na fila.

**Seu personagem só tem uma velocidade por round.** Se ele agir mais de uma vez, a velocidade
do round é a **média** de todas as ações que ele fez.

## O preço de agir

Todo mundo paga o mesmo por ação: **a menor velocidade do round**.

Quem foi o mais lento do round age por último e começa o round seguinte do zero. Quem foi
mais rápido sobra troco — e é esse troco que decide se ele age de novo.

### Um exemplo

Três personagens rolam: **p1 = 20**, **p2 = 23**, **p3 = 11**. O preço do round é **11**, a
menor das três.

| Ordem | Quem | Tinha | Depois de pagar 11 |
|-------|------|-------|--------------------|
| 1º | **p2** | 23 | sobram 12 — e 12 ainda dá para pagar outra ação |
| 2º | **p1** | 20 | sobram 9 — não dá para outra; leva **+9** ao próximo round |
| 3º | **p3** | 11 | sobra 0 — começa o próximo round zerado |
| 4º | **p2** | | age a segunda vez |

### Agir de novo custa depois

Quando p2 percebe que pode agir de novo, ele envia a segunda ação — e ela **rola a própria
velocidade**.

Digamos que role 17. A velocidade de p2 no round vira a média: `(23 + 17) ÷ 2 = 20`. E ele
pagou duas vezes: `20 − 22 = −2`.

**A ação acontece de qualquer jeito.** O que a segunda rolagem decide não é *se* você age, e
sim quanto isso vai te custar depois: p2 agiu duas vezes e começa o próximo round devendo,
enquanto p1 agiu uma vez e começa com +9.

E tem mais: como a média puxou p2 de 12 para 9, ele passa a agir **depois de p3**, que tinha
11. A segunda ação dele atrasou dentro do próprio round.

É isso que impede o mais rápido de dominar a batalha de graça. Agir mais vezes é sempre
possível — só não é grátis.

## O que sobra atravessa o round

O que sobrar na sua barra vai para o próximo round, como crédito ou como dívida.

**Existe um teto: você nunca carrega mais que o preço daquele round.** Isso impede
acumular tempo indefinidamente — ficar parado dois rounds seguidos não rende mais do que
ficar parado um.

### Ficar parado é legítimo

Se você não agir, carrega o piso do round. Isso é de propósito: em batalha, muitas vezes o
certo é **parar e ler a luta** — analisar o contexto, esperar uma brecha, montar a estratégia.

Quem fica parado começa o round seguinte um pouco mais rápido. É uma troca justa: **você
trocou uma ação por tempo**. E é uma vantagem que dura um round só — quem age também chega ao
teto depois de alguns rounds.

## Duas barras, não uma

Seu personagem tem **duas barras separadas**:

| Barra | Cobre |
|-------|-------|
| **Ação** | atacar, usar um item, puxar algo da mochila, usar uma habilidade |
| **Movimento** | shift, dash, salto, rolamento |

Cada uma tem a sua própria velocidade e o seu próprio saldo. Mover não gasta a sua ação, e
agir não gasta o seu movimento.

## Mover e atacar

Como são duas barras, você tem duas formas de fazer as duas coisas — e a escolha entre elas
**é sua**.

### Duas ações separadas

Envie um movimento e um ataque como ações independentes. Cada um entra na sua barra e
acontece quando aquela barra chegar a vez. **A ordem sai do relógio, não da sua intenção.**

É o que você faz quando não se importa com a ordem — quer só fazer o que for mais rápido
primeiro.

Se a ordem **importa**, você controla ela não enfileirando as duas de uma vez: manda o
movimento, espera acontecer, e só então manda o ataque. *"Só quero atacar depois de me
movimentar."*

### Ação combinada

Você também pode amarrar as duas numa ação só. Aí elas acontecem **juntas**, e no tempo da
**mais lenta** das duas.

| Ação combinada | Como funciona |
|----------------|---------------|
| **Cait** | Correr se afastando do inimigo e atacá-lo. A ordem é livre: atacar antes de sair, atacar durante, ou atacar no fim do movimento — depende de qual barra estiver na frente. |
| **Arremetida** | Percorre 1 quadrado e ataca. O movimento vem **obrigatoriamente antes** do ataque. |
| **Investida** | Percorre 2 ou mais quadrados e ataca. Também com movimento antes. |

> Se você quer recuar e atacar mas **na sua ordem**, não use o cait: mande as duas ações
> separadas. O cait existe justamente para quando tanto faz — ou quando você quer as duas ao
> mesmo tempo.

## Enxergando a barra

Você vê **a sua própria barra** — quanto custou sua ação e quanto sobrou — para decidir se já
manda a próxima.

E você vê **a barra geral**, com a ordem de todo mundo, incluindo quem vai agir mais de uma
vez. É ela que te diz quanto tempo você tem para montar a próxima ação enquanto os outros
narram.

> Sem enxergar a barra geral, você só descobriria que era a sua vez quando ela já passou.

---

> **🔧 Para Desenvolvedores**
>
> Implementação técnica: [`docs/dev/match/combat-engine.md`](../../dev/match/combat-engine.md)
