# Classes de Personagem (Character Classes)

> Sistema de classes do HxH RPG.

## Visão Geral

A classe de personagem define a **especialização inicial** e determina:
- Experiência (XP) inicial em perícias (Skills) específicas
- Experiência inicial em proficiências (Proficiencies) de armas
- Experiência inicial em atributos (Attributes)
- Categorias Nen indicadas (Indicated Categories)
- Distribuição de pontos na criação (Distribution)
- Perícias Conjuntas (Joint Skills) exclusivas
- Proficiências Conjuntas (Joint Proficiencies) exclusivas

## Distribuição de Pontos (Distribution)

Na criação do personagem, o jogador distribui pontos conforme as regras da classe:

### Perícias (Skills Distribution)
- A classe define quantos pontos distribuir e os valores disponíveis
- O jogador escolhe **quais perícias** receberão esses pontos
- Apenas perícias permitidas pela classe podem ser escolhidas

**Exemplo:** Uma classe com `SkillPoints: [80, 40, 40]` permite que o jogador escolha 3 perícias da lista permitida, atribuindo 80 XP a uma, 40 a outra, e 40 à terceira.

### Proficiências (Proficiencies Distribution)
- Mesma mecânica: valores pré-definidos distribuídos entre armas permitidas
- Apenas armas da lista `ProficienciesAllowed` da classe são válidas

### Validação

O sistema valida rigorosamente:
1. **Quantidade correta** — número de escolhas deve bater com o número de pontos
2. **Perícias/armas permitidas** — apenas opções da lista da classe
3. **Pontos corretos** — cada valor atribuído deve corresponder a um valor disponível (sem repetição)

**Erros possíveis:**
- `NoSkillDistributionError` — classe não tem distribuição, mas jogador enviou escolhas
- `SkillsCountMismatchError` — número de escolhas diferente do esperado
- `SkillNotAllowedError` — perícia não está na lista permitida
- `SkillsPointsMismatchError` — valores não correspondem aos pontos disponíveis
- (mesmos erros para proficiências)

## Classes Disponíveis (12 ativas)

| Classe | EN | Categorias Nen Indicadas |
|--------|-----|--------------------------|
| Espadachim | Swordsman | — |
| Samurai | Samurai | — |
| Ninja | Ninja | — |
| Ladino | Rogue | — |
| Netrunner | Netrunner | — |
| Pirata | Pirate | — |
| Mercenário | Mercenary | — |
| Terrorista | Terrorist | — |
| Monge | Monk | — |
| Militar | Military | — |
| Hunter | Hunter | — |
| Mestre de Armas | WeaponsMaster | — |

### Classes Futuras (4 planejadas)

| Classe | EN | Status |
|--------|-----|--------|
| Atleta | Athlete | Planejada |
| Tribal | Tribal | Planejada |
| Experimento | Experiment | Planejada |
| Circo | Circus | Planejada |

## Perfil da Classe (Class Profile)

Cada classe possui um perfil com metadados:
- **Nome** (Name) — identificador único da classe
- Dados adicionais para exibição e lore

## Aplicação de XP

Após validação da distribuição, o sistema aplica as experiências:
- `ApplySkills` — aplica XP nas perícias escolhidas
- `ApplyProficiencies` — aplica XP nas proficiências escolhidas

Estas experiências são somadas às experiências base que a classe já fornece (`SkillsExps`, `ProficienciesExps`, `AttributesExps`).
