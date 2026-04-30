# Proficiências

Proficiências representam a habilidade do personagem no uso de armas específicas. Diferentemente das [Perícias](pericias.md), que são baseadas em atributos, as proficiências são vinculadas diretamente a armas e evoluem conforme o personagem pratica com elas.

## Conceito

Cada proficiência possui seu próprio sistema de experiência (EXP) e nível. Ao usar uma arma em combate, a experiência é distribuída em cascata: a proficiência recebe EXP, que se propaga para as perícias físicas, habilidades, atributos e, finalmente, a experiência do personagem.

## Tipos de Proficiência

### Proficiência Comum

Uma proficiência comum é vinculada a **uma única arma**. É o tipo mais simples:

- Uma arma → uma proficiência
- Evolui independentemente
- Propaga EXP para perícias físicas via cascata

**Exemplo:** Um personagem que treina apenas com Espada terá uma proficiência comum de Espada.

### Proficiência Conjunta

Uma proficiência conjunta agrupa **múltiplas armas** sob uma única progressão compartilhada:

- Várias armas → uma proficiência com nome personalizado
- Todas as armas do grupo compartilham o mesmo nível e EXP
- Propaga EXP tanto para perícias físicas quanto para perícias de habilidade
- Suporta **buffs** por arma, que são somados ao nível para testes

**Exemplo:** Um personagem que luta com Espada e Adaga pode ter uma proficiência conjunta chamada "Lâminas" que evolui ao usar qualquer uma das duas armas.

#### Sistema de Buffs

Proficiências conjuntas possuem um sistema de buff por arma. O jogador pode receber um bônus temporário para uma arma específica dentro da proficiência conjunta. Esse buff é somado ao nível da proficiência no momento do teste. Buffs podem ser aplicados, removidos ou consultados durante o jogo.

## Fluxo de Cascata

Quando EXP é inserida em uma proficiência, o fluxo segue esta ordem:

1. A **proficiência** recebe a experiência e avança sua progressão
2. As **perícias físicas** associadas recebem a experiência em cascata
3. As **habilidades** correspondentes avançam com a experiência propagada
4. Os **atributos** recebem a cascata
5. A **experiência geral do personagem** é atualizada

Para proficiências conjuntas, o fluxo inclui também as perícias de habilidade:

1. A **proficiência conjunta** recebe a experiência
2. As **perícias físicas** e as **perícias de habilidade** recebem a experiência em cascata simultaneamente
3. A cascata continua normalmente para habilidades, atributos e experiência do personagem

## Como Proficiências Funcionam

Todas as proficiências do personagem são organizadas pelo sistema. Ao buscar uma proficiência por nome de arma, o sistema sempre verifica primeiro as proficiências conjuntas e depois as comuns — assim, se uma arma pertence a um grupo conjunto, a proficiência compartilhada é usada automaticamente.

O sistema também mantém buffs independentes por proficiência, que são somados ao nível no cálculo do valor de teste.

### Valor para Teste

O valor para teste de uma proficiência é calculado como:

> **Valor de Teste** = nível da proficiência + buff temporário

> **Nota:** Atualmente, o valor para teste utiliza apenas o nível da proficiência. A integração com o poder do atributo associado está pendente de implementação futura.

## Lista de Armas

### Corpo a Corpo — Curtas

| Arma              |
|-------------------|
| Adaga             |
| Cimitarra         |
| Rapieira          |
| Chicote           |
| Clava             |
| Espada            |
| Foice             |
| Katana            |
| Katar             |
| Lança             |
| Machado           |
| Martelo           |
| Maça              |
| Mangual           |
| Picareta          |
| Punho             |
| Tridente          |
| Tchaco            |
| Bastão            |

### Corpo a Corpo — Longas

| Arma              |
|-------------------|
| Alabarda          |
| Arco Longo        |
| Clava Longa       |
| Espada Longa      |
| Foice Longa       |
| Lança Longa       |
| Machado Longo     |
| Martelo de Guerra |
| Maça Longa        |

### Arremesso

| Arma              |
|-------------------|
| Adaga de Arremesso|
| Machado de Arremesso |
| Martelo de Arremesso |

### Armas de Projétil e Armas de Fogo

| Arma              |
|-------------------|
| Arco              |
| Besta             |
| AK-47             |
| AR-15             |
| Metralhadora      |
| Pistola .38       |
| Rifle             |
| Uzi               |
| Bomba             |

---

> **🔧 Para Desenvolvedores**
>
> Implementação técnica: [`docs/dev/character-sheet/skills-proficiencies.md`](../../dev/character-sheet/skills-proficiencies.md)
> Código-fonte: `internal/domain/entity/character_sheet/proficiency/`
