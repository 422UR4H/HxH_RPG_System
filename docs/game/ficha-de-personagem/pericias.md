# Perícias (Skills)

## Visão Geral

Perícias representam as habilidades específicas de um personagem, derivadas dos seus atributos. Cada perícia está vinculada a um atributo pai e determina a competência do personagem em ações concretas — desde combate até percepção sensorial e controle de Nen.

## Fórmula: Valor de Teste

Quando um personagem realiza um teste de perícia, o **valor de teste** é calculado como:

```
valor_de_teste = nível_perícia + poder_atributo (+ buff)
```

Onde:
- **nível_perícia** — determinado pela experiência acumulada na perícia
- **poder_atributo** — o poder do atributo pai (valor + bônus de habilidade + buff do atributo)
- **buff** — bônus temporário aplicado diretamente na perícia pelo Manager (opcional)

## Tipos de Perícia

### Perícias Comuns (CommonSkill)

São as perícias padrão, cada uma vinculada a um único atributo. Ao receber experiência, a perícia propaga o upgrade em cascata para o atributo pai e, em seguida, para a experiência de habilidade.

**Fluxo de cascade:**
```
Perícia → Atributo → Habilidade → Personagem (exp)
```

### Perícias Conjuntas (JointSkill)

Perícias conjuntas são compostas por duas ou mais perícias comuns. Quando o jogador usa uma perícia que pertence a uma conjunta, o sistema redireciona para a perícia conjunta automaticamente.

**Características:**
- Possuem **buff próprio** que é somado ao valor de teste
- Precisam ser **inicializadas** (`Init`) antes de serem adicionadas ao Manager
- Na cascata, a experiência é **multiplicada** pelo número de perícias comuns que compõem a conjunta antes de propagar para a habilidade

**Fórmula do valor de teste (JointSkill):**
```
valor_de_teste = nível_conjunta + poder_atributo + buff_conjunta
```

**Multiplicação de exp na cascata:**
```
exp_propagada = exp_recebida × quantidade_de_perícias_comuns
```

## Lista Completa de Perícias (31)

### Atributos Físicos

#### Resistência (Resistance) — Coef. XP: 5.0

| Perícia | EN | Descrição |
|---|---|---|
| Vitalidade | Vitality | Capacidade de resistir a dano e manter-se consciente |
| Energia | Energy | Reserva energética para ações prolongadas |
| Defesa | Defense | Capacidade de bloquear ou absorver impactos |

#### Força (Strength) — Coef. XP: 5.0

| Perícia | EN | Descrição |
|---|---|---|
| Empurrar | Push | Força para empurrar objetos ou oponentes |
| Agarrar | Grab | Capacidade de segurar e imobilizar |
| Carregar | Carry | Peso máximo que pode ser transportado |

#### Agilidade (Agility) — Coef. XP: 5.0

| Perícia | EN | Descrição |
|---|---|---|
| Velocidade | Velocity | Velocidade máxima de movimento |
| Aceleração | Accelerate | Rapidez para atingir velocidade máxima |
| Frenagem | Brake | Capacidade de parar ou reduzir velocidade rapidamente |

#### Celeridade (Celerity) — Coef. XP: 5.0

| Perícia | EN | Descrição |
|---|---|---|
| Destreza | Legerity | Agilidade fina e controle de movimentos rápidos |
| Repelir | Repel | Capacidade de desviar ataques ou objetos com velocidade |
| Finta | Feint | Habilidade de enganar o oponente com movimentos falsos |

#### Flexibilidade (Flexibility) — Coef. XP: 5.0

| Perícia | EN | Descrição |
|---|---|---|
| Acrobacia | Acrobatics | Movimentos acrobáticos, saltos e piruetas |
| Evasão | Evasion | Esquiva sem necessidade de grande deslocamento |
| Furtividade | Sneak | Movimentação silenciosa e discreta |

#### Destreza (Dexterity) — Coef. XP: 5.0

| Perícia | EN | Descrição |
|---|---|---|
| Reflexo | Reflex | Tempo de reação a estímulos |
| Precisão | Accuracy | Pontaria e precisão em ataques (crítico) |
| Ocultação | Stealth | Capacidade de se esconder e passar despercebido |

#### Sentido (Sense) — Coef. XP: 5.0

| Perícia | EN | Descrição |
|---|---|---|
| Visão | Vision | Acuidade visual e capacidade de detectar detalhes |
| Audição | Hearing | Percepção sonora e localização por som |
| Olfato | Smell | Detecção e identificação de odores |
| Tato | Tact | Sensibilidade tátil e percepção por toque |
| Paladar | Taste | Identificação de substâncias pelo sabor |

#### Constituição (Constitution) — Coef. XP: 5.0

| Perícia | EN | Descrição |
|---|---|---|
| Cura | Heal | Recuperação natural de ferimentos |
| Fôlego | Breath | Resistência respiratória e capacidade pulmonar |
| Tenacidade | Tenacity | Resistência à fadiga e perseverança física |

### Atributos Espirituais

#### Chama Nen (Flame)

| Perícia | EN | Descrição |
|---|---|---|
| Foco | Focus | Concentração para canalizar Nen |
| Força de Vontade | WillPower | Determinação e resistência mental/espiritual |
| Autoconhecimento | SelfKnowledge | Compreensão de si mesmo e do próprio Nen |

#### Consciência Nen (Conscience)

| Perícia | EN | Descrição |
|---|---|---|
| Coa | Coa | Controle e ocultação da aura (Cocooning Aura) |
| Mop | Mop | Manutenção e fluxo da aura pelo corpo (Mora of Pores) |
| Aop | Aop | Projeção e liberação da aura (Aura Output Power) |

## Sistema de Buffs

O **Manager** de perícias mantém um mapa de buffs temporários por perícia. Buffs são somados ao valor de teste no momento da consulta, sem alterar a experiência ou nível da perícia.

```
valor_de_teste_final = perícia.GetValueForTest() + buff_do_manager
```

**Operações:**
- `SetBuff(nome, valor)` — define um buff para a perícia
- `DeleteBuff(nome)` — remove o buff
- `GetBuffs()` — retorna todos os buffs ativos

## Fluxo de Experiência (Cascade)

Quando experiência é inserida em uma perícia, o sistema propaga automaticamente:

```
1. Perícia recebe exp → incrementa pontos e possivelmente sobe de nível
2. Atributo pai recebe cascade → incrementa sua própria exp
3. Habilidade governante recebe cascade → incrementa sua exp
4. Experiência do personagem é atualizada
```

Para **perícias conjuntas**, o passo adicional é:
```
1. Perícia conjunta recebe exp → incrementa pontos
2. Atributo pai recebe cascade
3. Exp é multiplicada pelo número de perícias comuns componentes
4. Habilidade governante recebe a exp multiplicada
```

## Gerenciador de Perícias (Manager)

O `Manager` centraliza o acesso a todas as perícias do personagem:

- **Init** — inicializa o mapa de perícias (só pode ser chamado uma vez)
- **Get** — busca uma perícia pelo nome; prioriza perícias conjuntas
- **IncreaseExp** — insere experiência e dispara a cascata
- **AddJointSkill** — registra uma perícia conjunta (deve estar inicializada)
- **GetValueForTestOf** — retorna o valor de teste incluindo buffs
- **GetSkillsLevel** — retorna o nível de todas as perícias em um mapa
