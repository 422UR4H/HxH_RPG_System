# Atributos

## Visão Geral

Atributos representam as características fundamentais de um personagem no HxH RPG System. Eles são divididos em três categorias — **Físicos**, **Mentais** e **Espirituais** — e determinam diretamente o poder, valor e capacidades do personagem em diversas situações de jogo.

## Tipos de Atributo

### Atributos Primários

Atributos primários são a base da ficha. Cada um possui:
- **Pontos distribuídos** — adicionados manualmente pelo jogador
- **Experiência própria** — recebida via progressão em cascata
- **Nível** — determinado pela tabela de progressão
- **Bônus de habilidade** — influência da habilidade governante

**Fórmulas:**

> *valor = pontos + nível*
>
> *poder = valor + bônus de habilidade (arredondado) + buff*

#### Atributos Físicos (Velocidade de Progressão: 5.0)

| Atributo | Descrição |
|---|---|
| Resistência | Defesa e resistência a dano |
| Força | Potência física e dano corpo a corpo |
| Agilidade | Velocidade de reação e esquiva |
| Celeridade | Velocidade de movimento |
| Flexibilidade | Amplitude de movimentos |
| Destreza | Precisão e controle motor fino |
| Sentido | Percepção sensorial |
| Constituição | Atributo médio de Resistência e Força |

#### Atributos Mentais (Velocidade de Progressão: 1.0)

| Atributo | Descrição |
|---|---|
| Resiliência | Resistência mental e foco |
| Adaptabilidade | Capacidade de se ajustar a situações |
| Ponderação | Análise e tomada de decisão |
| Criatividade | Inovação e pensamento lateral |

### Atributos Médios

Atributos médios são calculados automaticamente como a **média arredondada** dos seus atributos primários componentes. Não possuem pontos distribuíveis próprios.

**Fórmulas:**

> *pontos = média arredondada dos pontos dos atributos primários componentes*
>
> *valor = pontos + nível*
>
> *poder = valor + bônus de habilidade (arredondado) + buff*

O arredondamento utilizado é o padrão: metade arredonda para longe do zero. Exemplo: a média de 3 e 4 é 3,5, que arredonda para 4.

O bônus de habilidade de um atributo médio é a média dos bônus dos seus atributos primários componentes.

### Atributos Espirituais

Atributos espirituais representam o poder Nen do personagem. Diferem dos primários por **não possuírem pontos distribuíveis** — seu poder vem exclusivamente do nível, bônus de habilidade e buffs.

**Fórmula:**

> *poder = nível + bônus de habilidade (arredondado) + buff*

| Atributo | Velocidade de Progressão | Descrição |
|---|---|---|
| Chama | 1.0 | Intensidade e força da aura |
| Consciência | 1.0 | Controle e percepção da aura |

> **Nota:** Atributos espirituais só existem para personagens que despertaram o Nen.

## Progressão em Cascata

Quando experiência é inserida em uma perícia, ela se propaga pela cadeia:

1. A **perícia** recebe o XP
2. O **atributo** recebe o XP (para atributos médios, o XP é dividido entre os primários)
3. A **habilidade** recebe o XP
4. A **experiência do personagem** recebe o XP

Para atributos médios, a divisão de XP entre os primários é feita de forma precisa, levando em conta o resto de divisões anteriores. Isso garante que nenhum ponto de experiência é perdido ao longo do tempo.

## Buffs Temporários

Buffs são bônus (ou penalidades) temporários que alteram o poder de um atributo. Eles representam efeitos passageiros no jogo, como:
- **Itens equipados** — uma armadura pode aumentar a Resistência
- **Habilidades Nen** — técnicas que ampliam temporariamente um atributo
- **Condições de combate** — efeitos de terreno, poções, estados alterados

Os buffs são aplicados e removidos automaticamente pelo sistema conforme as circunstâncias do jogo. Quando um buff está ativo, seu valor é somado diretamente ao poder do atributo afetado, refletindo imediatamente em todas as mecânicas que dependem daquele atributo.

## Distribuição de Pontos

Apenas **atributos primários físicos** podem receber pontos distribuídos pelo jogador. Atributos mentais, médios e espirituais não possuem distribuição manual de pontos — seu progresso depende exclusivamente da experiência obtida por treinamento (progressão em cascata) e buffs temporários.

---

> **🔧 Para Desenvolvedores**
>
> Implementação técnica: [`docs/dev/character-sheet/abilities-attributes.md`](../../dev/character-sheet/abilities-attributes.md)
> Código-fonte: `internal/domain/entity/character_sheet/attribute/`
