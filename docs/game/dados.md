# Dados

> Sistema de rolagem de dados do HxH RPG.

## Tipos de Dados

O sistema utiliza 7 tipos de dados poliédricos:

| Dado | Lados | Uso Comum |
|------|-------|-----------|
| D4   | 4     | Dano menor, efeitos leves |
| D6   | 6     | Dano básico, ataques desarmados |
| D8   | 8     | Dano médio, armas leves |
| D10  | 10    | Dano forte, armas intermediárias |
| D12  | 12    | Dano pesado, armas de grande impacto |
| D20  | 20    | Testes gerais (regra alternativa de partida) |
| D100 | 100   | Probabilidades especiais, tabelas de evento |

## Testes

Sempre que seu personagem tenta algo cujo sucesso não é garantido — acertar um golpe, executar uma acrobacia, agir mais rápido que o adversário — o sistema faz um **teste**.

### A rolagem padrão: 2 D10

Um teste rola **dois D10 e soma os resultados**. O total vai de 2 a 20, mas os valores do meio são muito mais prováveis que os extremos: você tende a rolar perto de 11, e resultados muito altos ou muito baixos são raros. Isso torna a perícia do seu personagem mais decisiva que a sorte.

Ao resultado dos dados soma-se o valor do personagem no que está sendo testado.

### Crítico e erro crítico

O que define um crítico **não é o total**, e sim os dois dados:

| Situação | Como acontece |
|----------|---------------|
| **Crítico** | Ambos os dados saem **10** |
| **Erro crítico** | Ambos os dados saem **1** |

Ou seja: somar 20 só é possível com dois dez, e somar 2 só com dois uns — mas a leitura é sempre feita nos dados individuais, não na soma.

### Classe de Dificuldade (CD)

Alguns testes têm uma **CD** definida pelo mestre: o número que o resultado precisa alcançar para o teste ter sucesso. Quanto maior a CD, mais difícil a tarefa.

### Vantagem e desvantagem

Sob certas condições, um teste é feito **com vantagem** ou **com desvantagem**:

- **Vantagem** — a rolagem é feita duas vezes e vale o **melhor** resultado.
- **Desvantagem** — a rolagem é feita duas vezes e vale o **pior** resultado.

Vantagens e desvantagens se acumulam e podem se anular entre si.

Um exemplo de desvantagem em combate: **trocar uma ação que você já havia declarado**. Mudar de ideia no meio da batalha custa caro — não compensa abandonar uma ação para tentar outra que exija muitos testes.

### Regra alternativa: D20

Uma partida pode adotar o **D20** no lugar dos 2 D10 para os testes. A diferença é a distribuição: com D20 todos os resultados são igualmente prováveis, o que torna a partida mais imprevisível e a sorte mais decisiva. O padrão do sistema continua sendo 2 D10.

## Mecânica de Rolagem

### Geração de Resultado

Ao rolar um dado, o sistema gera um número aleatório entre 1 e o número de lados do dado. O resultado fica registrado até a próxima rolagem. Um resultado zero significa que o dado ainda não foi rolado.

### Combinação de Dados

Armas e ações utilizam múltiplos dados combinados. O resultado final é a **soma de todos os dados** na rolagem.

**Exemplo:** Uma Espada usa D10 + D4. Ao atacar, rola-se ambos e soma-se os resultados.

## Dados por Arma

Cada arma possui um conjunto específico de dados que representa seu potencial de dano variável. Armas mais pesadas e complexas tendem a usar mais dados e/ou dados maiores.

Exemplos:
- **Adaga:** 1×D8 — ataque rápido, dano consistente
- **Espada Longa:** D12 + D10 + D4 — alto potencial, variável
- **Martelo de Guerra:** D12 + D12 + D6 — devastador

Consulte o documento de [Armas](armas.md) para a lista completa.

## 🎲 Curiosidade: Aleatoriedade Verdadeira

O sistema de dados do HxH RPG utiliza um gerador de números aleatórios **criptograficamente seguro** — o mesmo tipo de tecnologia usada em sistemas bancários e de segurança digital. Isso significa que cada rolagem produz resultados **verdadeiramente aleatórios**: sem padrões, sem sequências previsíveis, sem possibilidade de manipulação.

Em cenários extremamente raros (praticamente impossíveis em condições normais), o sistema possui uma camada de segurança que recorre a um gerador pseudo-aleatório como alternativa. Na prática, isso quase nunca acontece.

**Por que isso importa para você, jogador?** Fairness total. Nenhum jogador é favorecido ou prejudicado por padrões ocultos nos dados. Cada rolagem é independente e genuinamente imprevisível — exatamente como dados reais bem balanceados, mas com a garantia matemática de um sistema digital.

---

> **🔧 Para Desenvolvedores**
>
> Implementação técnica: [`docs/dev/weapons-dice.md`](../dev/weapons-dice.md)
> Código-fonte: `internal/domain/entity/die/`
