# Aptidões

## Visão Geral

As aptidões são as quatro grandes áreas em que um personagem evolui. Cada uma reúne uma parte da ficha e cresce junto com ela:

| Aptidão | Na ficha | Reúne |
|---|---|---|
| Aptidão Física | Physicals | Atributos e perícias físicas, Vida (HP) e Estamina (SP) |
| Aptidão Mental | Mentals | Atributos mentais |
| Aptidão Espiritual | Spirituals | Atributos espirituais, princípios Nen e Aura (AP) |
| Aptidão de Perícias | Skills | O treino de todas as perícias juntas |

**Vida (HP)**, **Estamina (SP)** e **Aura (AP)** são as três barras do personagem. Veja [Vida e estamina](./status.md).

Cada aptidão tem um nível, que sobe com a experiência que chega até ela.

## Pontos de Personagem

Os pontos de personagem medem a **força geral** do personagem. É a mesma ideia dos "pontos do Hisoka": um número que diz o quanto alguém é forte.

**Eles não são pontos para distribuir.** Você não gasta pontos de personagem em lugar nenhum; eles só crescem.

Cada nível que **qualquer** aptidão ganha soma 1 ponto de personagem. Ou seja:

> *pontos de personagem = soma dos níveis das quatro aptidões*

### Por que não usar o nível do personagem?

O **nível do personagem** vem de toda a experiência que ele já ganhou (veja [Experiência](./experiencia.md)). Já os pontos de personagem contam níveis de aptidão, e a Aptidão Espiritual sobe de nível bem mais rápido que as outras. Por isso, de propósito, despertar e treinar o Nen pesa mais na força geral do personagem do que treinar só o corpo.

### Pontos para distribuir

Os **pontos para distribuir** são outra coisa. Eles vêm de duas aptidões só:

- Cada nível da **Aptidão Física** dá **1 ponto** para distribuir nos atributos físicos primários.
- Cada nível da **Aptidão Mental** dá **1 ponto** para distribuir nos atributos mentais primários.

A Aptidão Espiritual e a Aptidão de Perícias não dão pontos para distribuir. Veja como distribuir em [Atributos](./atributos.md#distribuicao-de-pontos).

## Bônus de Aptidão

Cada aptidão tem um bônus, que mistura a força geral do personagem com o nível daquela aptidão:

> *bônus de aptidão = (pontos de personagem + nível da aptidão) / 2*

Esse bônus entra no **poder dos atributos** (veja [Atributos](./atributos.md#formulas)) e no máximo de **Vida, Estamina e Aura** (veja [Vida e estamina](./status.md)).

## Talento

O talento é o **potencial** do personagem: ele dita a velocidade com que o personagem evolui, porque multiplica a experiência que ele recebe. Quanto maior o talento, mais cada treino rende.

O talento tem nível, e esse nível pode **subir ou cair** ao longo do jogo, conforme a interpretação do personagem, as boas jogadas e as ações estratégicas, heroicas ou importantes para a história e para o cenário.

Os personagens começam com talento alto justamente porque têm potencial para se tornar hunters. É o mesmo talento que a Associação Hunter observa, como o presidente Netero deixa claro ao final do primeiro arco.

Os atributos físicos e mentais primários também são canônicos nesse sentido, citados pelo próprio Netero. Só os secundários foram criados para o jogo. Veja [Atributos](./atributos.md).

## Progressão em Cascata

A experiência que o personagem ganha treinando sobe em cascata pela ficha até chegar às aptidões e ao nível do personagem. A explicação completa está no fim de [Atributos](./atributos.md#progressao-em-cascata).

---

> **🔧 Para Desenvolvedores**
>
> Implementação técnica: [`docs/dev/character-sheet/abilities-attributes.md`](../../dev/character-sheet/abilities-attributes.md)
> Código-fonte: `internal/domain/entity/character_sheet/ability/`
>
> - Pontos de personagem: `Ability.CascadeUpgrade` soma em `CharacterExp.IncreaseCharacterPoints`
>   os níveis que a aptidão ganhou. Coeficientes: Physicals/Mentals/Skills 20, Spirituals 5.
> - Pontos para distribuir: `CharacterSheet.DistributePrimaryPhysicalPoints` exige soma dos
>   primários ≤ nível de Physicals (idem Mentals); `ApplyInitialAttributePoints` exige igualdade.
> - Talento: o multiplicador de XP ainda **não** é aplicado no código, e o nível inicial
>   (`BASE_TALENT_LVL` = 20 + bônus por categorias, `TalentByCategorySet`) está comentado em
>   `create_character_sheet.go` (TODO: mover para o fluxo de aceite). A regra acima é a de jogo.
