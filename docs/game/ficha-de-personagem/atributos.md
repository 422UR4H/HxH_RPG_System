# Atributos (Attributes)

## Visão Geral

Atributos representam as características fundamentais de um personagem no HxH RPG System. Eles são divididos em três categorias — **Físicos**, **Mentais** e **Espirituais** — e determinam diretamente o poder, valor e capacidades do personagem em diversas situações de jogo.

## Tipos de Atributo

### Atributos Primários (PrimaryAttribute)

Atributos primários são a base da ficha. Cada um possui:
- **Pontos distribuídos** — adicionados manualmente pelo jogador
- **Experiência própria** — recebida via cascade upgrade
- **Nível** — determinado pela tabela de experiência
- **Bônus de habilidade** — influência da habilidade governante

**Fórmulas:**
```
valor     = pontos + nível
poder     = valor + int(bônusDeHabilidade) + buff
```

#### Atributos Físicos (Coef. XP: 5.0)

| Atributo | EN | Descrição |
|---|---|---|
| Resistência | Resistance | Defesa e resistência a dano |
| Força | Strength | Potência física e dano corpo a corpo |
| Agilidade | Agility | Velocidade de reação e esquiva |
| Celeridade | Celerity | Velocidade de movimento |
| Flexibilidade | Flexibility | Amplitude de movimentos |
| Destreza | Dexterity | Precisão e controle motor fino |
| Sentido | Sense | Percepção sensorial |
| Constituição | Constitution | Atributo médio de Resistência e Força |

#### Atributos Mentais (Coef. XP: 1.0)

| Atributo | EN | Descrição |
|---|---|---|
| Resiliência | Resilience | Resistência mental e foco |
| Adaptabilidade | Adaptability | Capacidade de se ajustar a situações |
| Ponderação | Weighting | Análise e tomada de decisão |
| Criatividade | Creativity | Inovação e pensamento lateral |

### Atributos Médios (MiddleAttribute)

Atributos médios são calculados automaticamente como a **média arredondada** dos seus atributos primários componentes. Não possuem pontos distribuíveis próprios.

**Fórmulas:**
```
pontos    = round(soma_pontos_primários / quantidade_primários)
valor     = pontos + nível
poder     = valor + int(bônusDeHabilidade) + buff
```

O arredondamento segue `math.Round` (arredondamento bancário): valores `.5` arredondam para o inteiro par mais próximo. Exemplo: `avg(3, 4) = 3.5 → 4`.

O bônus de habilidade de um atributo médio é a média dos bônus dos seus atributos primários componentes.

### Atributos Espirituais (SpiritualAttribute)

Atributos espirituais representam o poder Nen do personagem. Diferem dos primários por **não possuírem pontos distribuíveis** — seu poder vem exclusivamente do nível, bônus de habilidade e buffs.

**Fórmula:**
```
poder     = nível + int(bônusDeHabilidade) + buff
```

| Atributo | EN | Coef. XP | Descrição |
|---|---|---|---|
| Chama | Flame | 1.0 | Intensidade e força da aura |
| Consciência | Conscience | 1.0 | Controle e percepção da aura |

> **Nota:** Atributos espirituais só existem para personagens que despertaram o Nen.

## Upgrade em Cascata

Quando experiência é inserida em uma perícia, ela se propaga pela cadeia:

1. **Perícia** recebe o XP
2. **Atributo** recebe o XP (para atributos médios, o XP é dividido entre os primários)
3. **Habilidade** recebe o XP
4. **Experiência do personagem** recebe o XP

Para atributos médios, a divisão de XP considera o **resto da divisão anterior**:
```
exp_para_primário = (resto_anterior + xp_recebido) / quantidade_primários
```
Isso garante que nenhum ponto de experiência é perdido ao longo do tempo.

## Sistema de Buffs

Cada atributo possui um ponteiro para um valor de buff que é somado ao cálculo de poder. Buffs podem ser:
- **Definidos** via `SetBuff(nome, valor)` — altera o valor do buff
- **Removidos** via `RemoveBuff(nome)` — reseta o buff para 0

Os buffs são compartilhados por referência (ponteiro), permitindo que alterações no gerenciador reflitam imediatamente no cálculo de poder do atributo.

## Gerenciadores de Atributos

### Manager (Físicos e Mentais)

O `Manager` gerencia atributos primários e médios. Oferece:
- **Get(nome)** — busca qualquer atributo (primário ou médio)
- **GetPrimary(nome)** — busca apenas atributos primários (retorna cópia por valor)
- **IncreasePointsForPrimary(nome, valor)** — distribui pontos a um atributo primário
- **SetBuff / RemoveBuff** — gerencia buffs
- **GetAllAttributes** — retorna mapa completo de atributos
- **GetAttributesLevel / GetAttributesPoints** — consulta agregada

### SpiritualManager

O `SpiritualManager` gerencia exclusivamente atributos espirituais. Não possui distribuição de pontos (espirituais não têm pontos distribuíveis). Oferece:
- **Get(nome)** — busca atributo espiritual
- **SetBuff / RemoveBuff** — gerencia buffs
- **GetAllAttributes** — retorna mapa de atributos espirituais
- **GetAttributesLevel** — consulta de níveis

## Distribuição de Pontos

Apenas **atributos primários físicos** podem receber pontos distribuídos pelo jogador. Atributos mentais, médios e espirituais não possuem distribuição manual de pontos — seu progresso depende exclusivamente da experiência obtida por treinamento (cascade upgrades) e buffs temporários.
