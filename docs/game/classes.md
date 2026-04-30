# Classes de Personagem

> Sistema de classes do HxH RPG.

## Visão Geral

A classe de personagem define a **especialização inicial** e determina:
- Experiência (XP) inicial em perícias específicas
- Experiência inicial em proficiências de armas
- Experiência inicial em atributos
- Categorias Nen indicadas
- Distribuição de pontos na criação
- Perícias Conjuntas exclusivas
- Proficiências Conjuntas exclusivas

## Distribuição de Pontos

Na criação do personagem, o jogador distribui pontos conforme as regras da classe:

### Perícias
- A classe define quantos pontos distribuir e os valores disponíveis
- O jogador escolhe **quais perícias** receberão esses pontos
- Apenas perícias permitidas pela classe podem ser escolhidas

**Exemplo:** Uma classe que oferece três lotes de experiência — 80, 40 e 40 pontos — permite que o jogador escolha 3 perícias da lista permitida, atribuindo 80 XP a uma, 40 a outra, e 40 à terceira.

### Proficiências
- Mesma mecânica: valores pré-definidos distribuídos entre armas permitidas
- Apenas armas da lista de proficiências permitidas pela classe são válidas

### Validação

O sistema valida rigorosamente:
1. **Quantidade correta** — número de escolhas deve bater com o número de pontos
2. **Perícias/armas permitidas** — apenas opções da lista da classe
3. **Pontos corretos** — cada valor atribuído deve corresponder a um valor disponível (sem repetição)

**O que pode dar errado:**
- A classe não possui distribuição de pontos, mas o jogador tentou enviar escolhas
- O número de escolhas é diferente do esperado pela classe
- Uma perícia ou proficiência escolhida não está na lista permitida da classe
- Os valores de experiência atribuídos não correspondem aos pontos disponíveis
- As mesmas regras valem tanto para perícias quanto para proficiências

## Classes Disponíveis (12 ativas)

| Classe | Categorias Nen Indicadas |
|--------|--------------------------|
| Espadachim | — |
| Samurai | — |
| Ninja | — |
| Ladino | — |
| Netrunner | — |
| Pirata | — |
| Mercenário | — |
| Terrorista | — |
| Monge | — |
| Militar | — |
| Hunter | — |
| Mestre de Armas | — |

### Classes Futuras (4 planejadas)

| Classe | Status |
|--------|--------|
| Atleta | Planejada |
| Tribal | Planejada |
| Experimento | Planejada |
| Circo | Planejada |

## Perfil da Classe

Cada classe possui um perfil com informações próprias:
- **Nome** — identificador único da classe
- Dados adicionais para exibição e história

## Como a Classe Aplica Experiência Inicial

Após a validação da distribuição de pontos, o sistema aplica automaticamente toda a experiência inicial do personagem. Isso inclui:

1. **Experiência distribuída** — os pontos de XP que o jogador escolheu atribuir a perícias e proficiências específicas durante a criação.
2. **Experiência base da classe** — cada classe já fornece uma quantidade fixa de XP em determinadas perícias, proficiências e atributos, independente das escolhas do jogador.

A experiência final de cada perícia, proficiência e atributo é a soma desses dois componentes: a base da classe mais a distribuição feita pelo jogador.

---

> **🔧 Para Desenvolvedores**
>
> Implementação técnica: [`docs/dev/character-sheet/factory.md`](../dev/character-sheet/factory.md)
> Código-fonte: `internal/domain/entity/character_class/`
