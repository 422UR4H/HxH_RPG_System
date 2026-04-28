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

### Proficiência Conjunta (Joint Proficiency)

Uma proficiência conjunta agrupa **múltiplas armas** sob uma única progressão compartilhada:

- Várias armas → uma proficiência com nome personalizado
- Todas as armas do grupo compartilham o mesmo nível e EXP
- Propaga EXP tanto para perícias físicas quanto para perícias de habilidade (ability skills)
- Suporta **buffs** por arma, que são somados ao nível para testes
- Requer inicialização (`Init`) com referências às perícias físicas e de habilidade

**Exemplo:** Um personagem que luta com Espada e Adaga pode ter uma proficiência conjunta chamada "Lâminas" que evolui ao usar qualquer uma das duas armas.

#### Sistema de Buffs

Proficiências conjuntas possuem um sistema de buff:

- `SetBuff(arma, valor)` — define um buff para a arma, retorna nível + buff
- `DeleteBuff(arma)` — remove o buff
- `GetBuff()` — retorna o valor atual do buff

## Fluxo de Cascata

Quando EXP é inserida em uma proficiência, o fluxo segue esta ordem:

```
Proficiência (recebe EXP)
    └─→ Perícias Físicas (CascadeUpgrade)
         └─→ Habilidades (CascadeUpgrade)
              └─→ Atributos (CascadeUpgrade)
                   └─→ EXP do Personagem
```

Para proficiências conjuntas, o fluxo inclui também as perícias de habilidade:

```
Proficiência Conjunta (recebe EXP)
    ├─→ Perícias Físicas (CascadeUpgrade)
    └─→ Perícias de Habilidade (CascadeUpgrade)
         └─→ ... (continuação da cascata)
```

## Gerenciador de Proficiências (Manager)

O `Manager` organiza todas as proficiências de um personagem:

- **Busca por arma (`Get`)**: procura primeiro nas proficiências conjuntas, depois nas comuns
- **Adição**: `AddCommon` para comuns, `AddJoint` para conjuntas (com inicialização)
- **Aumento de EXP (`IncreaseExp`)**: localiza a proficiência e dispara a cascata
- **Buffs do Manager**: buffs independentes armazenados no manager, somados ao nível no cálculo de teste

### Valor para Teste

O método `GetValueForTestOf(arma)` calcula:

```
valor = nível da proficiência + buff do manager
```

> **Nota:** Atualmente, o valor para teste utiliza apenas o nível da proficiência. A integração com o poder do atributo associado está pendente de implementação futura.

## Lista de Armas

### Corpo a Corpo — Curtas

| Enum           | Arma              |
|----------------|-------------------|
| Dagger         | Adaga             |
| Scimitar       | Cimitarra         |
| Rapier         | Rapieira          |
| Whip           | Chicote           |
| Club           | Clava             |
| Sword          | Espada            |
| Scythe         | Foice             |
| Katana         | Katana            |
| Katar          | Katar             |
| Spear          | Lança             |
| Axe            | Machado           |
| Hammer         | Martelo           |
| Massa          | Maça              |
| Mangual        | Mangual           |
| Pickaxe        | Picareta          |
| Fist           | Punho             |
| Trident        | Tridente          |
| Tchaco         | Tchaco            |
| Staff          | Bastão            |

### Corpo a Corpo — Longas

| Enum           | Arma              |
|----------------|-------------------|
| Halberd        | Alabarda          |
| Longbow        | Arco Longo        |
| Longclub       | Clava Longa       |
| Longsword      | Espada Longa      |
| Longscythe     | Foice Longa       |
| Longspear      | Lança Longa       |
| Longaxe        | Machado Longo     |
| Warhammer      | Martelo de Guerra |
| Longmass       | Maça Longa        |

### Arremesso

| Enum           | Arma              |
|----------------|-------------------|
| ThrowingDagger | Adaga de Arremesso|
| ThrowingAxe    | Machado de Arremesso |
| ThrowingHammer | Martelo de Arremesso |

### Armas de Projétil e Armas de Fogo

| Enum           | Arma              |
|----------------|-------------------|
| Bow            | Arco              |
| Crossbow       | Besta             |
| Ak47           | AK-47             |
| Ar15           | AR-15             |
| MachineGun     | Metralhadora      |
| Pistol38       | Pistola .38       |
| Rifle          | Rifle             |
| Uzi            | Uzi               |
| Bomb           | Bomba             |

## Referência Técnica

- **Pacote:** `internal/domain/entity/character_sheet/proficiency`
- **Interface:** `IProficiency` — define os métodos comuns (`CascadeUpgradeTrigger`, `GetLevel`, `GetExpPoints`, etc.)
- **Structs:** `Proficiency`, `JointProficiency`, `Manager`
- **Tabela de EXP:** Utiliza a mesma `ExpTable` do sistema de experiência geral
