# Sistema Nen

O sistema Nen é o pilar espiritual da ficha de personagem no RPG de Hunter × Hunter. Ele representa a capacidade do personagem de manipular a energia vital (aura) e define suas habilidades de combate espiritual.

## Princípios Nen

Os princípios são as técnicas fundamentais de Nen. Existem **11 princípios** no sistema:

| Princípio | Descrição |
|-----------|-----------|
| **Ten** | Manter a aura ao redor do corpo |
| **Zetsu** | Suprimir completamente a aura |
| **Ren** | Aumentar a intensidade da aura |
| **Gyo** | Concentrar aura em uma parte do corpo |
| **Hatsu** | Manifestar a habilidade Nen pessoal |
| **Shu** | Envolver um objeto com aura |
| **Kou** | Concentrar toda a aura em um ponto |
| **Ken** | Manter Ren distribuído pelo corpo |
| **Ryu** | Redistribuir aura em tempo real |
| **In** | Esconder a aura de detecção |
| **En** | Expandir a aura para detectar presença |

### Valor de Teste dos Princípios

Cada princípio possui um **valor de teste** calculado pela fórmula:

```
ValorDeTeste = nível do princípio + poder da Consciência + nível da Chama
```

Esse valor representa a efetividade total do personagem ao utilizar o princípio em testes durante o jogo.

## Categorias Nen

As categorias definem a afinidade do personagem com um tipo específico de Nen. Existem **6 categorias**:

| Categoria | Centro no Hexágono |
|-----------|-------------------|
| **Reforço** (Reinforcement) | 0 |
| **Transmutação** (Transmutation) | 100 |
| **Materialização** (Materialization) | 200 |
| **Especialização** (Specialization) | 300 |
| **Manipulação** (Manipulation) | 400 |
| **Emissão** (Emission) | 500 |

### Valor de Teste das Categorias

O valor de teste de cada categoria é calculado pela fórmula:

```
ValorDeTesteCategoria = int((nívelCategoria + valorTesteHatsu) × porcentagem / 100)
```

Onde `porcentagem` é determinada pelo Hexágono Nen (veja abaixo).

## Hexágono Nen

O Hexágono Nen é o sistema que distribui as porcentagens de afinidade entre as categorias com base na posição do personagem no hexágono.

### Escala

- O hexágono opera numa escala de **0 a 599** (total de 600 valores)
- Cada categoria ocupa **100 unidades** consecutivas
- O valor é circular: após 599 volta para 0

### Cálculo de Porcentagem

A porcentagem de afinidade com cada categoria é calculada com base na **distância hexagonal**:

```
diferencaAbsoluta = |centroCategoria - valorHexAtual|
se diferencaAbsoluta > 300:
    diferencaAbsoluta = 600 - diferencaAbsoluta

divisor = 100 / 20 = 5
porcentagem = 100 - diferencaAbsoluta / divisor
```

Isso resulta em saltos de **20%** por categoria de distância:

| Distância | Porcentagem |
|-----------|-------------|
| Própria categoria | 100% |
| Adjacente (1 passo) | 80% |
| 2 passos | 60% |
| Oposta (3 passos) | 40% |

### Especialização

A categoria **Especialização** possui uma regra especial: ela retorna **0%** se não for a categoria atual do personagem. Apenas personagens cuja categoria Nen é Especialização podem utilizar habilidades dessa categoria.

### Reset de Categoria

O sistema permite resetar a posição do hexágono para o centro da categoria atual. Isso é inspirado nos eventos do **arco de Formigas Quimera**, onde Gon perdeu e precisou recuperar suas habilidades Nen, retornando ao ponto base de sua categoria.

## Hatsu

O Hatsu representa as habilidades Nen individuais do personagem e funciona como o elo entre os princípios e as categorias.

### Valor de Teste do Hatsu

```
ValorDeTesteHatsu = nível do Hatsu + poder da Consciência + nível da Chama
```

### Cascade de Experiência

Quando experiência é adicionada a uma categoria ou princípio, ela se propaga em cascata:

```
Categoria → Hatsu → Consciência → Habilidade Espiritual → Exp do Personagem
```

Cada nível dessa cascata registra os dados de progressão (nível, experiência e valor de teste) no objeto `UpgradeCascade`, permitindo que o sistema acompanhe a evolução completa do personagem.

### Inicialização

O Hatsu precisa ser inicializado com o mapa de categorias antes de poder ser utilizado. Uma tentativa de inicialização dupla resulta em erro.

## PrinciplesManager

O `PrinciplesManager` é o gerenciador central que coordena todos os princípios Nen, o Hexágono Nen e o Hatsu.

### Responsabilidades

- **Gerenciar princípios**: armazena e fornece acesso aos 10 princípios (excluindo Hatsu, que é tratado separadamente)
- **Gerenciar hexágono**: controla o aumento/diminuição do valor hexagonal e a atualização das porcentagens das categorias
- **Gerenciar Hatsu**: delega operações de experiência e consultas de categorias ao Hatsu
- **Fornecer dados em lote**: retorna níveis, experiências e valores de teste de todos os princípios e categorias de uma vez

### Operações do Hexágono via Manager

- `IncreaseCurrHexValue()` — incrementa o valor hexagonal e atualiza porcentagens
- `DecreaseCurrHexValue()` — decrementa o valor hexagonal e atualiza porcentagens
- `ResetNenCategory()` — reseta para o centro da categoria atual
- `GetNenCategoryName()` — retorna a categoria Nen atual
- `GetCurrHexValue()` — retorna o valor hexagonal atual

> **Nota:** Todas as operações do hexágono verificam se ele foi inicializado antes de executar, retornando erro caso contrário.
