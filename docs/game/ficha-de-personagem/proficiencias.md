# Proficiências

Proficiências medem o quanto o personagem domina cada arma. Diferente das [Perícias](./pericias.md), que pertencem a atributos, as proficiências pertencem às armas e evoluem conforme o personagem pratica com elas.

## Conceito

Cada proficiência tem a própria experiência e o próprio nível. Ao usar uma arma em combate, ou treinar com ela, a proficiência daquela arma recebe experiência. Essa experiência também sobe para a **Aptidão Física** e, dali, para o nível do personagem.

Algumas classes deixam você escolher armas que recebem experiência extra na criação. Veja [Classes](../classes.md#escolha-de-armas).

## Tipos de Proficiência

### Proficiência Comum

Uma proficiência comum pertence a **uma única arma** e evolui sozinha.

**Exemplo:** um personagem que treina só com Espada tem uma proficiência comum de Espada.

### Proficiência Conjunta

Uma proficiência conjunta reúne **várias armas** numa progressão só: usar qualquer uma delas faz a mesma proficiência evoluir, e todas compartilham o mesmo nível. Hoje ela vem de classe — o Espadachim e o Mestre de Armas começam com uma.

Além da Aptidão Física, a experiência de uma proficiência conjunta também sobe para a **Aptidão de Perícias**.

## Valor de Teste

> **Valor de Teste** = nível da proficiência

> **Nota:** Por enquanto, o valor de teste usa só o nível da proficiência. O poder do atributo ligado à arma ainda vai entrar nessa conta.

## Lista de Armas

O dano de cada arma é a soma dos dados com o bônus fixo. **2 D12 + 3**, por exemplo, quer dizer: role dois dados de 12 lados e some 3. Veja mais em [Armas](../armas.md).

### Corpo a Corpo

| Arma | Na ficha | Dano |
|---|---|---|
| Punho | Fist | 2 D6 + D4 |
| Adaga | Dagger | D8 + 5 |
| Katar | Katar | D6 + 6 |
| Espada | Sword | D10 + D4 + 2 |
| Katana | Katana | D12 + D4 + 7 |
| Cimitarra | Scimitar | D6 + D4 + 4 |
| Rapieira | Rapier | 2 D4 + 5 |
| Lança | Spear | D8 + D4 + 3 |
| Tridente | Trident | 3 D8 + 3 |
| Machado | Axe | D10 + D6 + 1 |
| Picareta | Pickaxe | D8 + D6 + 2 |
| Martelo | Hammer | D12 + D6 |
| Maça | Mace | D12 + D4 + 1 |
| Mangual | Flail | D12 + D4 + 1 |
| Clava | Club | 2 D8 + 1 |
| Cajado | Staff | D10 + D8 |
| Chicote | Whip | D8 + 2 D4 |
| Foice | Scythe | D6 + 2 D4 + 2 |
| Tchaco | Tchaco | D10 + 4 |

### Corpo a Corpo — Longas

| Arma | Na ficha | Dano |
|---|---|---|
| Espada Longa | Longsword | D12 + D10 + D4 + 2 |
| Lança Longa | Longspear | D12 + D8 + D4 + 3 |
| Alabarda | Halberd | D12 + D10 + D6 + 1 |
| Machado Longo | Longaxe | D12 + D10 + D6 + 1 |
| Martelo de Guerra | Warhammer | 2 D12 + D6 |
| Maça Longa | Longmace | 2 D12 + D4 + 1 |
| Clava Longa | Longclub | D12 + 2 D8 + 1 |
| Foice Longa | Longscythe | D12 + D6 + 2 D4 + 2 |

### Arremesso

| Arma | Na ficha | Dano |
|---|---|---|
| Adaga de Arremesso | ThrowingDagger | D8 + 2 |
| Machado de Arremesso | ThrowingAxe | D10 + 1 |
| Martelo de Arremesso | ThrowingHammer | D12 |

### Projéteis, Armas de Fogo e Explosivos

| Arma | Na ficha | Dano |
|---|---|---|
| Arco | Bow | D12 + 3 |
| Arco Longo | Longbow | 2 D12 + 3 |
| Besta | Crossbow | 3 D12 + 2 |
| Pistola .38 | Pistol38 | D12 + 4 |
| Uzi | Uzi | D12 + D8 + 1 |
| AK-47 | Ak47 | 3 D10 + 1 |
| AR-15 | Ar15 | 2 D10 + 6 |
| Rifle | Rifle | D12 + D10 + 8 |
| Metralhadora | MachineGun | D12 + D10 + 3 |
| Bomba | Bomb | 3 D12 |

---

> **🔧 Para Desenvolvedores**
>
> Implementação técnica: [`docs/dev/character-sheet/skills-proficiencies.md`](../../dev/character-sheet/skills-proficiencies.md)
> Código-fonte: `internal/domain/entity/character_sheet/proficiency/`
>
> - Cascata: `Proficiency.CascadeUpgradeTrigger` → `Physicals` (referência de
>   `GetExpReferenceOf(enum.Physicals)`); `JointProficiency` também → `Skills`.
> - Busca: o `proficiency.Manager` procura primeiro nas conjuntas e depois nas comuns — se uma
>   arma pertence a uma conjunta, a compartilhada é usada.
> - Buffs: o `Manager` guarda buff por proficiência, e a conjunta guarda buff por arma; os dois
>   são somados ao nível no valor de teste (`nível + buff`). Ficaram fora do texto do jogador.
> - `GetValueForTest` = só o nível (`//+ p.attribute.GetPower()` pendente).
> - Dano da tabela gerado de `internal/domain/entity/item/weapons_factory.go` (dados em ordem
>   decrescente, iguais agrupados, `+ N` = dano base).
