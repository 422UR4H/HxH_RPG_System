# O Mestre na Batalha

> O que o mestre faz enquanto o sistema calcula.

## O princípio: o sistema propõe, o mestre dispõe

Tudo o que o sistema calcula é um **padrão**, não um veredito. A dificuldade que ele sugeriu,
o dano que ele somou, o desfecho que ele escolheu na escada — cada um desses números pode ser
substituído pelo mestre.

Não é uma exceção nem um modo de administrador. É a forma como o sistema foi pensado: ele
existe para tirar a aritmética do caminho, **não** para tirar a mesa das mãos de quem a
conduz.

> Quando o mestre substitui um valor, o valor substituído **não é jogado fora** — fica
> guardado. Nada que o sistema calculou ou que um jogador enviou se perde por causa de uma
> sobrescrita.

## A ação do mestre

O mestre também age, e a ação dele não é uma ação de jogador com outro nome. Ela opera em
dois níveis:

| Nível | Exemplos |
|---|---|
| **Dentro de uma ação de jogador** | trocar os alvos, acrescentar ou tirar perícias da corrente de testes, mexer na velocidade, conceder ou anular vantagem |
| **Acima da batalha** | o que o mestre conduz e nenhum jogador conduz |

O segundo nível ainda está por escrever. Ele é o motivo de a ação do mestre ser uma coisa
própria, e não um caso particular da ação de jogador.

## Acrescentar e tirar perícias

O mestre mexe na **corrente de testes** de uma ação: acrescenta uma perícia — e o personagem
passa a depender de mais um teste — ou tira uma, e a ação fica mais barata de executar.

**Acrescentar rola dados novos.** Isso não fere a regra de que o mestre nunca re-rola o dado
de um jogador: não é uma re-rolagem, é a primeira rolagem de um teste que não existia.

**Tirar não é jogar fora.** Enquanto o turno está aberto, os dados da perícia removida
continuam guardados. Se o mestre mudar de ideia e recolocá-la, ela volta com os dados que já
tinha — senão, tirar e pôr de volta seria uma re-rolagem de graça.

Quando o turno fecha, esses dados somem: um teste que não aconteceu não deixa rastro no
histórico da partida.

## Vantagem num teste passivo

Um teste passivo não rola: usa o valor médio dos dados (11 nos 2 D10). Quando o mestre dá
**vantagem** ou **desvantagem** a um teste passivo, ele passa a ser **rolado** — e o valor médio
vira o outro resultado da comparação:

- **com vantagem**, vale o **melhor** entre o dado e o valor médio;
- **com desvantagem**, vale o **pior** entre os dois.

O passivo continua sendo o piso (ou o teto): a vantagem traz a chance de ir além, a desvantagem
traz o risco de ficar abaixo. O dado é rolado **uma vez só**, na primeira vez que alguém pede —
tirar e pôr a vantagem de volta lê o mesmo dado.

A **desvantagem do sistema** (a de quem troca uma ação declarada) vale do mesmo jeito: um teste
passivo de quem trocou rola contra o valor médio e fica com o pior. Se o mestre der vantagem a esse
teste, as duas se anulam e o valor médio volta a valer.

## Ajustar uma ação que ainda está na fila

O mestre pode ajustar uma ação **antes de ela agir**, enquanto ainda espera na fila: dar ou tirar
vantagem, somar um ajuste, trocar a perícia que mede o dano. Se ele mexe na **velocidade**, a
ação muda de lugar na fila. O jogador que declarou a ação não vê o ajuste. Se a ação for
cancelada, ou trocada por uma reação, o ajuste vai embora com ela.

## O que os jogadores enxergam

**A ação editada é a ação.** Quando o mestre muda alguma coisa, o que a mesa vê é o resultado
já com a mudança — não a versão do jogador com uma correção pendurada.

---

> **🔧 Para Desenvolvedores**
>
> Implementação técnica: [`docs/dev/match/combat-engine.md`](../../dev/match/combat-engine.md)
