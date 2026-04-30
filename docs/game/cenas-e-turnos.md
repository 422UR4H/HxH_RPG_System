# Cenas e Turnos

> Estrutura de execução em tempo real de uma partida no HxH RPG.

## Visão Geral

Durante uma partida em execução, o fluxo de jogo se organiza em uma
hierarquia: **Partida → Cenas → Turnos → Rounds**.

```
Partida
├── Cena — roleplay ou battle
│   ├── Turno — modo free ou race
│   │   ├── Round — ação de um personagem
│   │   │   ├── Ação
│   │   │   └── Reações — de outros jogadores
│   │   └── Round ...
│   └── Turno ...
└── Cena ...
```

## Cenas

Uma cena é um segmento contínuo da partida com um propósito narrativo.
Existem duas categorias:

| Categoria | Descrição | Exemplo |
|-----------|-----------|---------|
| **Roleplay** | Interação, investigação, diálogo | Jogadores exploram uma cidade |
| **Battle** | Combate entre personagens/NPCs | Luta contra um inimigo |

### Importante: Categoria ≠ Modo do Turno

A categoria da cena serve para **classificação e histórico** — facilitar a
leitura sequencial dos eventos de uma partida pelos jogadores e mestre após
a sessão.

A categoria **NÃO determina** o modo do turno. Um turno pode ser livre (free)
ou disputado (race) independente de a cena ser roleplay ou battle. Embora o
esperado seja:

- Roleplay → turnos free (sem disputa de tempo)
- Battle → turnos race (ordem por velocidade)

...isso não é mandatório. O mestre pode configurar como desejar.

### Ciclo de Vida

1. Mestre cria uma cena com categoria e descrição breve inicial
2. Turnos são executados dentro da cena
3. Mestre finaliza a cena com descrição breve final
4. Cena finalizada não aceita mais turnos

## Turnos

Um turno é um ciclo de ações dentro de uma cena. Possui dois modos:

### Modo Free (Turno Livre)

- Sem pressão de tempo
- Jogadores agem em ordem natural (definida pelo mestre ou por consenso)
- Típico de cenas de roleplay, exploração, diálogo
- Não há fila de prioridade

### Modo Race (Turno Disputado)

- Cada milissegundo conta
- Ações são resolvidas por velocidade (fila de prioridade — as ações mais rápidas são resolvidas primeiro)
- Típico de combates
- A ação mais rápida é processada primeiro
- Reações podem ser engatilhadas ou desengatilhadas ao longo do fluxo

### Sistema de Turnos

O sistema de turnos gerencia a execução dos turnos **sem considerar se a cena é
roleplay ou battle**. Ele apenas conhece o modo do turno (free ou race) e
executa de acordo.

Essa separação é intencional:
- A **cena** classifica o contexto narrativo
- O **turno** executa a mecânica de jogo
- O **sistema** orquestra o fluxo

## Rounds

Um round é a **ação de um personagem** dentro de um turno. Composto por:

- **Ação principal** — o que o personagem faz
- **Reações** — respostas de outros personagens à ação
  - Reações podem ser **engatilhadas** automaticamente
  - Reações podem ser **desengatilhadas** ao longo do fluxo

Tudo dentro de um round parte da ação de um personagem específico.

### Em Modo Race

No modo race, rounds são ordenados pela velocidade da ação:

1. Todos declaram ações
2. Ações entram na fila de prioridade (por velocidade)
3. A ação mais rápida é extraída e resolvida
4. Reações geradas entram na fila com sua própria velocidade
5. Repete até a fila esvaziar

Ver `docs/game/combate/acoes.md` para detalhes sobre ações, ataques e fila
de prioridade.

## Eventos de Jogo

Além da hierarquia Cena → Turno → Round, uma partida registra **eventos de jogo**
genéricos que podem ocorrer a qualquer momento: mudanças de data narrativa,
mortes, notícias, ações desfeitas, etc.

Esses eventos complementam o histórico de ações para fornecer uma leitura
completa do que aconteceu em cada partida.

## Comunicação em Tempo Real

Quando uma partida está em execução, a comunicação acontece em tempo real.
Mestre e jogadores permanecem conectados ao servidor de jogo durante toda a
sessão.

O fluxo de cenas, turnos e rounds é transmitido em tempo real por essa
conexão, garantindo que todos os participantes acompanhem os eventos
simultaneamente.

---

> **🔧 Para Desenvolvedores**
>
> Implementação técnica: [`docs/dev/match/scenes.md`](../dev/match/scenes.md) · [`docs/dev/match/turns-rounds.md`](../dev/match/turns-rounds.md)
> Código-fonte: `internal/domain/entity/match/`
