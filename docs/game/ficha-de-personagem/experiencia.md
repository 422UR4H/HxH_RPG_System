# Sistema de Experiência (Experience System)

## Visão Geral

O sistema de experiência (XP) é a base de toda a progressão no HxH RPG System. Cada entidade na ficha de personagem — perícias, atributos, habilidades, princípios Nen — possui sua própria instância de experiência com tabela de progressão configurável.

## Tabela de Experiência (ExpTable)

Cada componente utiliza uma **ExpTable** com um coeficiente multiplicador. A tabela suporta até **100 níveis** (0–100) e é gerada por uma função sigmoidal tripla:

```
f(lvl) = 1700/(1 + e^(0.37*(12-lvl))) + 1800/(1 + e^(0.37*(38-lvl))) + 2000/(1 + e^(0.28*(74-lvl)))
```

O XP base para cada nível é `coeficiente × f(nível)`, e o XP agregado é a soma cumulativa de todos os níveis anteriores.

### Coeficientes por Componente

| Componente | Coeficiente | Descrição |
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

## Experiência do Personagem (CharacterExp)

A `CharacterExp` é o nível geral do personagem. Ela recebe XP no final de cada **upgrade em cascata** (cascade upgrade) — sempre que qualquer perícia ou princípio recebe experiência, o XP se propaga pela cadeia até chegar à experiência do personagem.

### Pontos de Personagem

Cada vez que uma **habilidade** (Physicals, Mentals, Spirituals, Skills) sobe de nível, o personagem ganha **pontos de personagem** que influenciam o bônus de habilidade.

### Fórmula do Bônus de Habilidade

```
bônus = (pontosDePersonagem + nívelDaHabilidade) / 2
```

## Upgrade em Cascata (Cascade Upgrade)

O mecanismo central de progressão. Quando XP é inserido em uma perícia:

1. A **perícia** recebe o XP
2. O **atributo** associado recebe o XP
3. A **habilidade** associada recebe o XP
4. A **experiência do personagem** recebe o XP
5. Todos os **status** são recalculados

Este processo garante que treinar qualquer aspecto do personagem contribui para sua progressão geral.
