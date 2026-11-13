# Sistema Nen

O Nen é o pilar espiritual da ficha no System X System. É a capacidade do personagem de manipular a energia vital (aura), e é dele que vêm as técnicas e o poder espiritual do personagem.

## Princípios Nen

Os princípios são as técnicas fundamentais de Nen. São **11 princípios**:

| Princípio | Descrição |
|-----------|-----------|
| **Ten** | Manter a aura ao redor do corpo |
| **Zetsu** | Suprimir completamente a aura |
| **Ren** | Aumentar a intensidade da aura |
| **Gyo** | Concentrar aura em uma parte do corpo |
| **Hatsu** | Manifestar a categoria Nen pessoal |
| **Shu** | Envolver um objeto com aura |
| **Kou** | Concentrar toda a aura em um ponto |
| **Ken** | Manter Ren distribuído pelo corpo |
| **Ryu** | Redistribuir aura em tempo real |
| **In** | Esconder a aura de detecção |
| **En** | Expandir a aura para detectar presença |

```diagram nen-principles paint=all
Ten · Zetsu · Ren · Gyo · Hatsu · Shu · Kou · Ken · Ryu · In · En
```

O Hatsu é um dos 11, mas tem regras próprias, explicadas em [Hatsu](./sistema-nen.md#hatsu).

### Valor de Teste dos Princípios

> **Valor de Teste** = nível do princípio + poder do Nen da Consciência + nível do Nen da Chama

## Hexágono Nen

Todo usuário de Nen tem afinidade com as seis categorias Nen, umas mais e outras menos. O **Hexágono Nen** mostra essa afinidade: as categorias ficam em volta de um hexágono, e o personagem ocupa uma posição nele.

### Escala

- O hexágono tem **600 posições**, de 0 a 599, e é circular: depois da 599 vem a 0 de novo.
- Cada categoria tem um **centro** no hexágono e ocupa as **100 posições** em volta dele.
- A categoria Nen do personagem é aquela em cujo trecho ele está.

### Afinidade com cada categoria

Quanto mais perto do centro de uma categoria o personagem está, maior a afinidade com ela:

> **Porcentagem** = 100 − distância até o centro da categoria ÷ 5

A distância é contada pelo caminho mais curto em volta do hexágono. Na prática, cada categoria de distância tira **20%**:

| Distância | Porcentagem |
|-----------|-------------|
| Própria categoria | 100% |
| Vizinha (1 passo) | 80% |
| 2 passos | 60% |
| Oposta (3 passos) | 40% |

### Especialização

A **Especialização** é exceção: quem não é especialista tem **0%** nela. Só personagens cuja categoria Nen é Especialização podem usar habilidades dessa categoria.

### Voltar ao centro da categoria

A posição do personagem no hexágono pode voltar ao centro da própria categoria. A inspiração vem do **arco das Formigas Quimera**, em que Gon perdeu e precisou recuperar suas habilidades Nen, voltando ao ponto de partida da sua categoria.

## Categorias Nen

São **6 categorias**, cada uma com o seu centro no hexágono:

| Categoria | Centro no Hexágono |
|-----------|-------------------|
| **Reforço** | 0 |
| **Transmutação** | 100 |
| **Materialização** | 200 |
| **Especialização** | 300 |
| **Manipulação** | 400 |
| **Emissão** | 500 |

### Valor de Teste das Categorias

> **Valor de Teste da Categoria** = (nível da categoria + valor de teste do Hatsu) × porcentagem ÷ 100

A porcentagem entra na conta com vírgula, sem arredondar; só o valor final da categoria é arredondado.

A **porcentagem** é a afinidade com a categoria, vinda do [Hexágono Nen](./sistema-nen.md#hexagono-nen).

## Hatsu

O Hatsu é a manifestação da categoria Nen pessoal do personagem e liga os princípios às categorias.

### Valor de Teste do Hatsu

> **Valor de Teste do Hatsu** = nível do Hatsu + poder do Nen da Consciência + nível do Nen da Chama

### Progressão em Cascata

O Nen tem uma cascata própria. Quando uma categoria recebe experiência, a experiência sobe assim:

1. A **categoria** recebe a experiência.
2. O **Hatsu** recebe a experiência.
3. O **Nen da Consciência** recebe a experiência.
4. A **Aptidão Espiritual** recebe a experiência.
5. Por fim, ela chega ao **nível do personagem**.

Com um princípio é igual, mas sem passar pelo Hatsu: do princípio, a experiência vai direto para o Nen da Consciência.

A ideia geral da cascata está no fim de [Atributos](./atributos.md#progressao-em-cascata).

---

> **🔧 Para Desenvolvedores**
>
> Implementação técnica: [`docs/dev/character-sheet/spiritual.md`](../../dev/character-sheet/spiritual.md)
> Código-fonte: `internal/domain/entity/character_sheet/spiritual/`
>
> - `PrinciplesManager` guarda os 10 princípios comuns; o Hatsu (11º) fica à parte, com mecânica
>   própria. Expõe nível, experiência e valor de teste de princípios e categorias.
> - `NenHexagon`: `IncreaseCurrHexValue`/`DecreaseCurrHexValue` (atualizam as porcentagens),
>   `ResetCategory` (volta ao centro da categoria atual), `GetCategoryName`, `GetCurrHexValue`.
> - Cada passo da cascata registra nível, experiência e valor de teste no `UpgradeCascade`.
