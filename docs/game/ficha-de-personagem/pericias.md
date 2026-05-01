# Perícias

## Visão Geral

Perícias representam as habilidades específicas de um personagem, derivadas dos seus atributos. Cada perícia está vinculada a um atributo pai e determina a competência do personagem em ações concretas — desde combate até percepção sensorial e controle de Nen.

## Fórmula: Valor de Teste

Quando um personagem realiza um teste de perícia, o **valor de teste** é calculado como:

> **Valor de Teste** = nível da perícia + poder do atributo (+ buff)

Onde:
- **nível da perícia** — determinado pela experiência acumulada na perícia
- **poder do atributo** — o poder do atributo pai (valor + bônus de habilidade + buff do atributo)
- **buff** — bônus temporário aplicado diretamente na perícia pelo sistema (opcional)

## Tipos de Perícia

### Perícias Comuns

São as perícias padrão, cada uma vinculada a um único atributo. Ao receber experiência, a perícia propaga a progressão em cascata para o atributo pai e, em seguida, para a experiência de habilidade.

> **Progressão em cascata:** Perícia → Atributo → Habilidade → Personagem (experiência)

### Perícias Conjuntas

Perícias conjuntas são compostas por duas ou mais perícias comuns. Quando o jogador usa uma perícia que pertence a uma conjunta, o sistema redireciona para a perícia conjunta automaticamente. Ao buscar uma perícia pelo nome, o sistema sempre prioriza as perícias conjuntas antes de verificar as comuns.

**Características:**
- Possuem **buff próprio** que é somado ao valor de teste
- Na cascata, a experiência é **multiplicada** pelo número de perícias comuns que compõem a conjunta antes de propagar para a habilidade

**Fórmula do valor de teste da perícia conjunta:**

> **Valor de Teste** = nível da conjunta + poder do atributo + buff da conjunta

**Multiplicação de experiência na cascata:**

> **Experiência propagada** = experiência recebida × quantidade de perícias comuns

## Lista Completa de Perícias (31)

### Atributos Físicos

#### Resistência — Coef. XP: 5.0

| Perícia | Descrição |
|---|---|
| Vitalidade | Capacidade de resistir a dano e manter-se consciente |
| Energia | Reserva energética para ações prolongadas |
| Defesa | Capacidade de bloquear ou absorver impactos |

#### Força — Coef. XP: 5.0

| Perícia | Descrição |
|---|---|
| Empurrar | Força para empurrar objetos ou oponentes |
| Agarrar | Capacidade de segurar e imobilizar |
| Carregar | Peso máximo que pode ser transportado |

#### Agilidade — Coef. XP: 5.0

| Perícia | Descrição |
|---|---|
| Velocidade | Velocidade máxima de movimento |
| Aceleração | Rapidez para atingir velocidade máxima |
| Frenagem | Capacidade de parar ou reduzir velocidade rapidamente |

#### Celeridade — Coef. XP: 5.0

| Perícia | Descrição |
|---|---|
| Destreza | Agilidade fina e controle de movimentos rápidos |
| Repelir | Capacidade de desviar ataques ou objetos com velocidade |
| Finta | Habilidade de enganar o oponente com movimentos falsos |

#### Flexibilidade — Coef. XP: 5.0

| Perícia | Descrição |
|---|---|
| Acrobacia | Movimentos acrobáticos, saltos e piruetas |
| Evasão | Esquiva sem necessidade de grande deslocamento |
| Furtividade | Movimentação silenciosa e discreta |

#### Destreza — Coef. XP: 5.0

| Perícia | Descrição |
|---|---|
| Reflexo | Tempo de reação a estímulos |
| Precisão | Pontaria e precisão em ataques (crítico) |
| Ocultação | Capacidade de se esconder e passar despercebido |

#### Sentido — Coef. XP: 5.0

| Perícia | Descrição |
|---|---|
| Visão | Acuidade visual e capacidade de detectar detalhes |
| Audição | Percepção sonora e localização por som |
| Olfato | Detecção e identificação de odores |
| Tato | Sensibilidade tátil e percepção por toque |
| Paladar | Identificação de substâncias pelo sabor |

#### Constituição — Coef. XP: 5.0

| Perícia | Descrição |
|---|---|
| Cura | Recuperação natural de ferimentos |
| Fôlego | Resistência respiratória e capacidade pulmonar |
| Tenacidade | Resistência à fadiga e perseverança física |

### Atributos Espirituais

#### Chama Nen

| Perícia | Descrição |
|---|---|
| Foco | Concentração para canalizar Nen |
| Força de Vontade | Determinação e resistência mental/espiritual |
| Autoconhecimento | Compreensão de si mesmo e do próprio Nen |

#### Consciência Nen

| Perícia | Descrição |
|---|---|
| Coa | Controle e ocultação da aura (Cocooning Aura) |
| Mop | Manutenção e fluxo da aura pelo corpo (Mora of Pores) |
| Aop | Projeção e liberação da aura (Aura Output Power) |

## Sistema de Buffs

O sistema mantém buffs temporários por perícia. Buffs são bônus que se somam ao valor de teste no momento da consulta, sem alterar a experiência ou nível da perícia.

> **Valor de teste final** = valor de teste da perícia + buff temporário

Um buff pode ser aplicado, removido ou consultado a qualquer momento durante o jogo. Eles representam efeitos temporários como magias, itens ou condições especiais.

## Progressão em Cascata

Quando experiência é inserida em uma perícia, o sistema propaga automaticamente a progressão por toda a ficha:

1. A **perícia** recebe experiência, incrementa seus pontos e possivelmente sobe de nível
2. O **atributo pai** recebe a experiência em cascata e incrementa sua própria progressão
3. A **habilidade governante** recebe a experiência e avança sua progressão
4. A **experiência geral do personagem** é atualizada

Para **perícias conjuntas**, há um passo adicional:

1. A **perícia conjunta** recebe experiência e incrementa seus pontos
2. O **atributo pai** recebe a experiência em cascata
3. A experiência é **multiplicada** pelo número de perícias comuns que compõem a conjunta
4. A **habilidade governante** recebe a experiência multiplicada

---

> **🔧 Para Desenvolvedores**
>
> Implementação técnica: [`docs/dev/character-sheet/skills-proficiencies.md`](../../dev/character-sheet/skills-proficiencies.md)
> Código-fonte: `internal/domain/entity/character_sheet/skill/`
