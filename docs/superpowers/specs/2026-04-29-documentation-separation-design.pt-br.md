# Spec de Design: Separação de Documentação

> Separar documentação de regras do jogo (para jogadores) da documentação técnica (para desenvolvedores).

## Problema

O diretório `docs/game/` contém conteúdo misto: regras de jogo para jogadores ao lado de detalhes de implementação para desenvolvedores (nomes de métodos, referências a pacotes, tipos de erro, padrões internos). Isso torna a documentação inadequada para dois públicos-chave:

1. **Jogadores/Mestres** — que precisam de regras claras para consultar durante o jogo (como um Livro do Jogador de D&D)
2. **Desenvolvedores** — que precisam de documentação técnica explicando fluxos de código, entidades e como contribuir

Além disso, a documentação de jogo será exportada no futuro para:
- Um livro de regras de RPG (físico/digital)
- Exibição no frontend, alinhada com a UX definida pelo designer do projeto

## Objetivos

1. **Criar** `docs/architecture/` com docs técnicos por domínio — detalhes de implementação, fluxos de código, relações entre entidades, guias de contribuição
2. **Refatorar** `docs/game/` para ler como um livro de regras puro — acessível a não-desenvolvedores, modular, exportável
3. **Conectar** as duas via footers de desenvolvedor nos game docs (onde relevante), linkando para os correspondentes técnicos

## Não-Objetivos

- Reescrever regras de jogo ou alterar mecânicas
- Documentar a integração com o frontend
- Criar layout formatado de livro (isso é um passo futuro)

## Design

### Fase 1: Criar Documentação Técnica (`docs/dev/`)

Criar docs técnicos por domínio usando conteúdo que já existe nos game docs MAIS leitura do código-fonte para detalhes mais profundos de implementação. Nem todo domínio precisa de um doc dedicado — ser crítico sobre o que genuinamente agrega valor além do que o código já comunica.

#### Estrutura

```
docs/dev/
├── overview.md                  (mover de docs/architecture/ — enriquecer)
├── character-sheet/
│   ├── experience.md            ← ExpTable, mecanismo de cascade, interfaces, coefs
│   ├── abilities-attributes.md  ← hierarquia de entidades, fórmulas, padrão Manager
│   ├── skills-proficiencies.md  ← tipos, distribuição, diferenças de cascade
│   ├── spiritual.md             ← princípios Nen, algoritmo do hexágono, Hatsu
│   ├── status.md                ← cálculos HP/SP/AP, triggers de upgrade
│   └── factory.md               ← CharacterSheetFactory, grafo de entidades, class wrap
├── auth.md                      ← JWT, sessão (sync.Map + DB), bcrypt, fluxo de login
├── campaigns-scenarios.md       ← fluxos CRUD, regras de validação, repos
├── enrollment.md                ← fluxo de enrollment, JOIN path, relações match↔sheet
├── weapons-dice.md              ← entidade Weapon, cálculos de penalidade/stamina, RNG
├── match/
│   ├── scenes.md                ← ciclo de vida, separação categoria vs modo
│   ├── turns-rounds.md          ← Turn Engine, free/race, implementação priority queue
│   └── actions.md               ← struct Action, velocidade, resolução de combate
└── websocket.md                 ← Hub/Room/Client, máquina de estados, protocolo
```

Esta é uma estrutura inicial — alguns docs podem ser mesclados ou omitidos se o código for auto-explicativo para aquele domínio.

#### Diretrizes de Conteúdo (Docs Técnicos)

O código deve ser auto-explicativo para entidades individuais. Docs de dev existem para fornecer contexto que NÃO é óbvio lendo um único arquivo:

- **Fluxos entre pacotes** — explicar jornadas que atravessam múltiplos arquivos/pacotes (ex: cascata de XP cruzando 4 pacotes)
- **Razão do design** — POR QUE algo foi projetado assim, não O QUE a struct contém
- **Relações não-óbvias** — como entidades se conectam de formas que não ficam claras apenas pelos imports
- **Referências a arquivos-fonte** — linkar para arquivos ao invés de replicar definições de structs
- **Guia de extensão** — como adicionar um novo componente (nova perícia, nova arma) quando o padrão não é imediatamente óbvio

Não documente mecanicamente. Seja crítico — se o código de um domínio é claro o suficiente por si só, um doc não é necessário. Se um fluxo cruza 4 pacotes e levaria 30 minutos para um dev rastrear, um doc que explica em 2 minutos é valioso.

**Idioma:** Docs de dev são escritos em PT-BR (assim como os game docs), já que a comunidade de desenvolvedores do projeto é brasileira. Referências a código (nomes de tipos, métodos, caminhos de arquivos) permanecem em inglês conforme aparecem no código-fonte.

### Fase 2: Refatorar Documentação de Jogo (`docs/game/`)

Reescrever cada arquivo para ler como um capítulo de livro de regras para jogadores.

Nem todo domínio precisa de um game doc — por exemplo, detalhes de autenticação não são regras de jogo. Seja crítico sobre o que genuinamente serve aos jogadores.

#### Princípios da Refatoração

1. **Remover todas as referências a código** — sem nomes de métodos (`GetValueForTest`, `CascadeUpgrade`), sem caminhos de pacotes, sem nomes de tipos de erro, sem nomes de tecnologias (bcrypt, JWT, crypto/rand, max-heap)
2. **Manter todas as mecânicas de jogo** — fórmulas (em notação matemática simples), tabelas de valores, regras
3. **Usar linguagem acessível** — um jogador que nunca programou deve entender tudo
4. **Tom** — informativo, claro, amigável. Como um capítulo de livro de RPG bem escrito
5. **Modular** — cada arquivo é um "capítulo" auto-contido que faz sentido independentemente
6. **Preservar estrutura** — manter os mesmos arquivos e organização geral

#### Convenção do Footer de Desenvolvedor

No final dos game docs (onde relevante), incluir um footer separado por `---`:

```markdown
---

> **🔧 Para Desenvolvedores**
>
> Implementação técnica: [`docs/dev/<caminho>`](../dev/<caminho>)
> Código-fonte: `internal/domain/entity/<caminho>/`
```

Este footer:
- É visualmente separado do conteúdo de jogo
- Claramente marcado como apenas para desenvolvedores
- Linka diretamente para o correspondente técnico
- Referencia a localização do código-fonte
- NÃO está presente em todos os arquivos — apenas onde agrega valor

### Ordem de Execução

A abordagem "trocar duas variáveis com uma terceira":

1. **Primeiro**: CRIAR todos os arquivos de `docs/dev/` (a "terceira variável")
   - Mover `docs/architecture/overview.md` existente para `docs/dev/overview.md`
   - Usar game docs existentes como fonte de conteúdo técnico
   - Ler código-fonte para enriquecer além do que os game docs atualmente contêm
   - Isso garante que nenhuma informação se perca
2. **Depois**: REFATORAR arquivos de `docs/game/`
   - Remover detalhes técnicos (eles agora vivem em dev)
   - Reescrever para legibilidade de jogador
   - Adicionar footers de desenvolvedor onde relevante
   - Remover arquivos que não fazem sentido como regras de jogo (ex: `autenticacao.md`)
3. **Por fim**: Enriquecer `docs/dev/overview.md` com referências cruzadas
4. **Implementar exemplos**: Os exemplos de transformação nesta spec (seção de curiosidade dos dados, explicação de perícias, explicação da cascata) devem ser implementados tal como estão durante a execução — eles servem como barra de qualidade para a documentação.

### Ordem de Prioridade (Fluxo de Cascata)

Seguir a estrutura de cascata de experiência, depois expandir:

1. Sistema de experiência
2. Perícias & Proficiências
3. Atributos (físicos, mentais, espirituais)
4. Habilidades
5. Barras de status
6. Sistema Nen/Espiritual
7. Classes
8. Armas & Dados
9. Autenticação & Sessões
10. Campanhas & Cenários
11. Partidas & Enrollment
12. Cenas, Turnos & Rounds
13. Ações em Combate

### Exemplos de Transformação

Cada exemplo mostra o mesmo conteúdo em três formas: o estado atual misturado, a reescrita para jogadores, e o correspondente técnico para devs.

---

#### Exemplo 1: Dados — Geração de Números Aleatórios

**Atual (`docs/game/dados.md`) — Misturado:**

```markdown
### Geração de Resultado
O sistema utiliza aleatoriedade criptograficamente segura (`crypto/rand`)
para gerar resultados. Em caso de falha (extremamente raro), realiza
fallback para `math/rand/v2`.
```

**Depois — Para jogadores (`docs/game/dados.md`):**

```markdown
### Geração de Resultado

O sistema garante que todas as rolagens de dados são **verdadeiramente
aleatórias** — não são pseudo-aleatórias como na maioria dos jogos digitais.
Isso significa que cada rolagem é tão imprevisível quanto um dado físico
na mesa.

#### 🎲 Curiosidade: Por que os dados são realmente aleatórios?

Computadores normalmente geram números "pseudo-aleatórios" — sequências que
parecem aleatórias mas são geradas por uma fórmula matemática determinística.
Se você soubesse a semente (seed), poderia prever todos os resultados futuros.

Neste sistema, os dados utilizam uma fonte de entropia do sistema operacional
— ruído de hardware, timing de eventos, e outras fontes físicas impossíveis
de prever. O resultado é estatisticamente equivalente a rolar dados físicos.

Apenas em um cenário extremamente raro de falha no gerador (quase impossível
em condições normais), o sistema recorre a um gerador pseudo-aleatório como
fallback de segurança.
```

**Depois — Para devs (`docs/dev/weapons-dice.md`):**

```markdown
## Random Number Generation

The dice system uses `crypto/rand` for true randomness (hardware entropy).
Fallback to `math/rand/v2` on the extremely rare case of reader failure.

See: `internal/domain/entity/die/die.go`

The `Roll()` method generates a value in `[1, N]` via `crypto/rand.Int()`.
If `rand.Int()` returns an error, it falls back to `rand.IntN()` from
`math/rand/v2` (which is auto-seeded and not predictable, but technically
pseudo-random).
```

> **Nota:** Este é um bom exemplo de enriquecer game docs com conteúdo curado que jogadores acham interessante — aleatoriedade verdadeira é um diferencial que vale a pena destacar. Procure oportunidades similares onde um detalhe técnico se traduz em algo que jogadores apreciam saber.

---

#### Exemplo 2: Perícias — Manager e Perícias Conjuntas

**Atual (`docs/game/pericias.md`) — Misturado:**

```markdown
## Gerenciador de Perícias (Manager)
O `Manager` centraliza o acesso a todas as perícias do personagem:
- **Init** — inicializa o mapa de perícias (só pode ser chamado uma vez)
- **Get** — busca uma perícia pelo nome; prioriza perícias conjuntas
- **IncreaseExp** — insere experiência e dispara a cascata
- **AddJointSkill** — registra uma perícia conjunta (deve estar inicializada)
- **GetValueForTestOf** — retorna o valor de teste incluindo buffs
- **GetSkillsLevel** — retorna o nível de todas as perícias em um mapa
```

**Depois — Para jogadores (`docs/game/pericias.md`):**

```markdown
## Perícias Conjuntas e Prioridade

Algumas classes de personagem possuem **perícias conjuntas** — perícias
especiais que agrupam duas ou mais perícias comuns em uma única progressão.

Quando seu personagem possui uma perícia conjunta, toda vez que ele usar
qualquer uma das perícias que fazem parte desse grupo, o sistema
automaticamente utiliza a conjunta. Isso significa que treinar qualquer
uma delas beneficia todo o grupo de uma vez.

**Exemplo prático:** Se seu personagem tem a perícia conjunta "Combate
Corpo a Corpo" (que agrupa Empurrar e Agarrar), toda vez que ele empurrar
ou agarrar um oponente, a experiência vai para a conjunta — e ambas as
perícias se beneficiam igualmente.
```

**Depois — Para devs (`docs/dev/character-sheet/skills-proficiencies.md`):**

```markdown
## Skill Manager Lookup Priority

When resolving a skill by name, the Manager checks joint skills first,
then falls back to common skills. This ensures joint skill XP routing
is transparent to callers.

Flow: `Manager.Get(name)` → check `jointSkills[name]` → check `skills[name]`

Joint skills must be initialized (`Init()`) before being added to the
Manager. They hold references to their component common skills and
multiply XP propagation by the component count during cascade.

See: `internal/domain/entity/character_sheet/skill/manager.go`
```

> **Nota:** A reescrita para jogadores deste exemplo é aceitável, mas não ideal — o conceito de perícias conjuntas é inerentemente complexo de explicar em termos simples. Durante a implementação, investir mais esforço em encontrar a linguagem certa para esta seção. Os exemplos 1 (dados) e 3 (cascata) são excelentes e devem ser usados como barra de qualidade.

---

#### Exemplo 3: Fluxo de Cascade Upgrade

**Atual (`docs/game/ficha-de-personagem/experiencia.md`) — Misturado:**

```markdown
## Upgrade em Cascata (Cascade Upgrade)
O mecanismo central de progressão. Quando XP é inserido em uma perícia:
1. A **perícia** recebe o XP
2. O **atributo** associado recebe o XP
3. A **habilidade** associada recebe o XP
4. A **experiência do personagem** recebe o XP
5. Todos os **status** são recalculados
```

**Depois — Para jogadores (`docs/game/ficha-de-personagem/experiencia.md`):**

```markdown
## Progressão em Cascata

O sistema de HxH RPG utiliza uma mecânica chamada **cascata**: toda
experiência que você ganha em uma ação específica se propaga para cima,
fortalecendo o personagem como um todo.

**Como funciona na prática:**

Imagine que seu personagem pratica escalada (perícia Acrobacia). Essa
experiência não fica isolada — ela também fortalece:

1. O **atributo Flexibilidade** (ao qual Acrobacia está vinculada)
2. A **habilidade Físicos** (que governa todos os atributos físicos)
3. O **nível geral do personagem**
4. As **barras de status** são recalculadas (HP, SP podem aumentar)

Isso significa que treinar qualquer aspecto do seu personagem contribui
para sua evolução geral. Não existe treinamento "perdido".
```

**Depois — Para devs (`docs/dev/character-sheet/experience.md`):**

```markdown
## Cascade Upgrade Flow

The cascade is the core XP propagation mechanism. It crosses 4 packages
in a single call chain:

```
skill.CascadeUpgradeTrigger(values)
  → attribute.CascadeUpgrade(values)
    → ability.CascadeUpgrade(values)
      → characterExp.CascadeUpgrade(values)
```

Each layer:
1. Calls `exp.IncreasePoints(values.GetExp())`
2. Records its state in the `UpgradeCascade` collector struct
3. Invokes the next layer's `CascadeUpgrade`

After cascade completes, `status.Manager.Upgrade()` recalculates HP/SP/AP.

Key interfaces: `ICascadeUpgrade`, `ITriggerCascadeExp`
See: `internal/domain/entity/character_sheet/skill/skill.go` (entry point)
See: `internal/domain/entity/character_sheet/experience/experience.go`
```

## Critérios de Sucesso

- [ ] Arquivos `docs/dev/` criados com detalhes técnicos focados (fluxos, razões de design, relações não-óbvias)
- [ ] Arquivos `docs/game/` refatorados sem referências de desenvolvimento e legíveis como livro de regras
- [ ] Footers de desenvolvedor presentes nos game docs que possuem correspondentes técnicos
- [ ] `docs/dev/overview.md` enriquecido com referências cruzadas para docs por domínio
- [ ] Um não-desenvolvedor consegue ler qualquer game doc e entender as regras completamente
- [ ] Um desenvolvedor consegue rastrear qualquer fluxo entre pacotes via dev docs sem adivinhar
- [ ] Game docs incluem conteúdo curado de "curiosidades" onde detalhes técnicos se traduzem em algo que jogadores apreciam (ex: aleatoriedade verdadeira nos dados)
- [ ] Sem repetição mecânica do que o código já comunica claramente
