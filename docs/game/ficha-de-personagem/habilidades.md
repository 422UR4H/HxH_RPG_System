# Habilidades (Abilities)

## Visão Geral

Habilidades representam as quatro grandes áreas de competência de um personagem. Cada habilidade governa um conjunto de atributos e perícias, e seu nível influencia diretamente o bônus aplicado a todos os componentes subordinados.

## As Quatro Habilidades

| Habilidade | EN | Coef. XP | Governa |
|---|---|---|---|
| Físicos | Physicals | 20.0 | Atributos e perícias físicas, HP, SP |
| Mentais | Mentals | 20.0 | Atributos e perícias mentais |
| Espirituais | Spirituals | 5.0 | Atributos espirituais, princípios Nen, AP |
| Perícias | Skills | 20.0 | Experiência geral de todas as perícias |

## Bônus de Habilidade (Ability Bonus)

O bônus é calculado pela média entre os pontos de personagem e o nível da habilidade:

```
bônus = (pontosDePersonagem + nívelDaHabilidade) / 2
```

Este bônus é usado nas fórmulas de:
- **Poder do atributo** (GetPower)
- **Status máximo** (HP, SP, AP)

## Talento (Talent)

O talento é um sistema de progressão especial baseado nas categorias Nen ativas do personagem. O nível do talento é determinado pelo `TalentByCategorySet`:

- **Base:** nível 20
- **Sem hexágono:** bônus = (categorias ativas - 1) × 2 (mínimo 1)
- **Com hexágono:** bônus = categorias ativas - 1

## Upgrade em Cascata

Quando uma habilidade recebe XP via cascade upgrade:
1. O XP é adicionado à experiência da habilidade
2. A experiência do personagem recebe o mesmo XP
3. Se a habilidade sobe de nível, os pontos de personagem aumentam

## Pontos de Personagem

Cada vez que uma habilidade sobe de nível, o personagem ganha pontos que:
- Aumentam o bônus de todas as habilidades
- Influenciam indiretamente todos os status e poderes de atributo
