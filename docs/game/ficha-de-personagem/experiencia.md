# Sistema de Experiência

## Visão Geral

O sistema de experiência (XP) é a base de toda a progressão no HxH RPG System. Cada elemento na ficha de personagem — perícias, atributos, habilidades, princípios Nen — possui seu próprio sistema de experiência com uma tabela de progressão específica.

## Tabela de Progressão

Cada componente da ficha possui uma **tabela de progressão** com uma velocidade de progressão própria. A tabela suporta até **100 níveis** (0–100), e a quantidade de XP necessária para cada nível cresce de forma não-linear: os primeiros níveis são rápidos, os intermediários exigem mais dedicação, e os últimos níveis são os mais desafiadores.

O XP base para cada nível é calculado como **velocidade de progressão × custo do nível**, e o XP total acumulado é a soma de todos os níveis anteriores.

### Velocidade de Progressão por Componente

| Componente | Velocidade de Progressão | Descrição |
|---|---|---|
| Personagem | 10.0 | Experiência geral do personagem |
| Talento | 2.0 | Progressão do talento por categoria |
| Físicos | 20.0 | Habilidade física |
| Mentais | 20.0 | Habilidade mental |
| Espirituais | 5.0 | Habilidade espiritual (Nen) |
| Perícias (habilidade) | 20.0 | Habilidade geral de perícias |
| Atributos Físicos | 5.0 | Cada atributo físico |
| Atributos Mentais | 1.0 | Cada atributo mental |
| Atributos Espirituais | 1.0 | Cada atributo espiritual |
| Perícias Físicas | 1.0 | Cada perícia física |
| Perícias Mentais | 2.0 | Cada perícia mental |
| Perícias Espirituais | 3.0 | Cada perícia espiritual |
| Princípios Nen | 1.0 | Cada princípio e categoria |

> **🔍 Curiosidade — A Fórmula por trás da Progressão**
>
> A curva de progressão é gerada por uma função sigmoidal tripla:
>
> *f(nível) = 1700 / (1 + e^(0,37 × (12 − nível))) + 1800 / (1 + e^(0,37 × (38 − nível))) + 2000 / (1 + e^(0,28 × (74 − nível)))*
>
> Ela determina a velocidade com que o custo de XP cresce entre os níveis. Na prática, isso significa que a progressão começa suave, acelera nos níveis intermediários e desacelera nos níveis mais altos — criando uma experiência de jogo equilibrada. Você não precisa calcular isso manualmente; o sistema cuida de tudo!

## Experiência do Personagem

A experiência do personagem representa o **nível geral** do personagem. Ela recebe XP ao final de cada **progressão em cascata** — sempre que qualquer perícia ou princípio recebe experiência, o XP se propaga pela cadeia até chegar à experiência do personagem.

### Pontos de Personagem

Cada vez que uma **habilidade** (Físicos, Mentais, Espirituais ou Perícias) sobe de nível, o personagem ganha **pontos de personagem** que influenciam o bônus de habilidade.

### Fórmula do Bônus de Habilidade

> *bônus = (pontos de personagem + nível da habilidade) / 2*

## Progressão em Cascata

O mecanismo central de progressão. Quando XP é adicionado a uma perícia, ele se propaga automaticamente por toda a cadeia:

1. A **perícia** recebe o XP
2. O **atributo** associado recebe o XP
3. A **habilidade** associada recebe o XP
4. A **experiência do personagem** recebe o XP
5. Todos os **status** são recalculados

**Exemplo prático:** imagine que seu personagem treina *Acrobacia* (uma perícia física). Esse treino não melhora apenas a Acrobacia — ele também fortalece a *Flexibilidade* (o atributo associado), que por sua vez melhora toda a habilidade *Físicos*, e por fim incrementa a experiência geral do personagem. É como treinar um movimento específico que acaba fortalecendo o corpo inteiro.

Este processo garante que treinar qualquer aspecto do personagem contribui para sua progressão geral.

---

> **🔧 Para Desenvolvedores**
>
> Implementação técnica: [`docs/dev/character-sheet/experience.md`](../../dev/character-sheet/experience.md)
> Código-fonte: `internal/domain/entity/character_sheet/experience/`
