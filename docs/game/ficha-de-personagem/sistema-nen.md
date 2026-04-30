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

> **Valor de Teste** = nível do princípio + poder da Consciência + nível da Chama

Esse valor representa a efetividade total do personagem ao utilizar o princípio em testes durante o jogo.

## Categorias Nen

As categorias definem a afinidade do personagem com um tipo específico de Nen. Existem **6 categorias**:

| Categoria | Centro no Hexágono |
|-----------|-------------------|
| **Reforço** | 0 |
| **Transmutação** | 100 |
| **Materialização** | 200 |
| **Especialização** | 300 |
| **Manipulação** | 400 |
| **Emissão** | 500 |

### Valor de Teste das Categorias

O valor de teste de cada categoria é calculado pela fórmula:

> **Valor de Teste da Categoria** = (nível da categoria + valor de teste do Hatsu) × porcentagem ÷ 100 (arredondado para baixo)

Onde a **porcentagem** é determinada pelo Hexágono Nen (veja abaixo).

## Hexágono Nen

O Hexágono Nen é o sistema que distribui as porcentagens de afinidade entre as categorias com base na posição do personagem no hexágono.

### Escala

- O hexágono opera numa escala de **0 a 599** (total de 600 valores)
- Cada categoria ocupa **100 unidades** consecutivas
- O valor é circular: após 599 volta para 0

### Cálculo de Porcentagem

A porcentagem de afinidade com cada categoria é calculada com base na **distância hexagonal**:

> **Diferença** = |centro da categoria − valor hexagonal atual|
> Se a diferença for maior que 300, ajusta-se: **diferença** = 600 − diferença
> **Porcentagem** = 100 − diferença ÷ 5

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

> **Valor de Teste do Hatsu** = nível do Hatsu + poder da Consciência + nível da Chama

### Progressão em Cascata

Quando experiência é adicionada a uma categoria ou princípio, ela se propaga em cascata:

1. A **categoria** (ou princípio) recebe a experiência
2. O **Hatsu** recebe a experiência em cascata
3. A **Consciência** avança com a experiência propagada
4. A **habilidade Espiritual** recebe a cascata
5. A **experiência geral do personagem** é atualizada

Cada nível dessa cascata registra os dados de progressão (nível, experiência e valor de teste), permitindo que o sistema acompanhe a evolução completa do personagem.

## Controle do Sistema Nen

O sistema Nen coordena todos os princípios, o Hexágono Nen e o Hatsu de forma integrada. O jogador pode interagir com o sistema das seguintes formas:

### Gerenciamento dos Princípios

O sistema armazena e fornece acesso aos 10 princípios (o Hatsu é tratado separadamente por possuir mecânicas próprias). O jogador pode consultar níveis, experiência e valores de teste de todos os princípios e categorias.

### Operações do Hexágono

- **Aumentar o valor hexagonal** — avança a posição no hexágono e atualiza as porcentagens de afinidade
- **Diminuir o valor hexagonal** — recua a posição no hexágono e atualiza as porcentagens de afinidade
- **Resetar a categoria** — retorna a posição ao centro da categoria Nen atual
- **Consultar a categoria atual** — verifica qual é a categoria Nen do personagem
- **Consultar o valor hexagonal** — verifica a posição atual no hexágono

---

> **🔧 Para Desenvolvedores**
>
> Implementação técnica: [`docs/dev/character-sheet/spiritual.md`](../../dev/character-sheet/spiritual.md)
> Código-fonte: `internal/domain/entity/character_sheet/spiritual/`
