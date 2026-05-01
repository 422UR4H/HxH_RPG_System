# Habilidades

## Visão Geral

Habilidades representam as quatro grandes áreas de competência de um personagem. Cada habilidade governa um conjunto de atributos e perícias, e seu nível influencia diretamente o bônus aplicado a todos os componentes subordinados.

## As Quatro Habilidades

| Habilidade | Velocidade de Progressão | Governa |
|---|---|---|
| Físicos | 20.0 | Atributos e perícias físicas, HP, SP |
| Mentais | 20.0 | Atributos e perícias mentais |
| Espirituais | 5.0 | Atributos espirituais, princípios Nen, AP |
| Perícias | 20.0 | Experiência geral de todas as perícias |

## Bônus de Habilidade

O bônus é calculado pela média entre os pontos de personagem e o nível da habilidade:

> *bônus = (pontos de personagem + nível da habilidade) / 2*

Este bônus é usado nas fórmulas de:
- **Poder do atributo**
- **Status máximo** (HP, SP, AP)

## Talento

O talento é um sistema de progressão especial baseado nas categorias Nen ativas do personagem. O nível do talento é determinado pela quantidade de categorias Nen que o personagem domina:

- **Base:** nível 20
- **Sem hexágono:** bônus = (categorias ativas − 1) × 2 (mínimo 1)
- **Com hexágono:** bônus = categorias ativas − 1

## Progressão em Cascata

Quando uma habilidade recebe XP via progressão em cascata:

1. O XP é adicionado à experiência da habilidade
2. A experiência do personagem recebe o mesmo XP
3. Se a habilidade sobe de nível, os pontos de personagem aumentam

## Pontos de Personagem

Cada vez que uma habilidade sobe de nível, o personagem ganha pontos que:
- Aumentam o bônus de todas as habilidades
- Influenciam indiretamente todos os status e poderes de atributo

---

> **🔧 Para Desenvolvedores**
>
> Implementação técnica: [`docs/dev/character-sheet/abilities-attributes.md`](../../dev/character-sheet/abilities-attributes.md)
> Código-fonte: `internal/domain/entity/character_sheet/ability/`
