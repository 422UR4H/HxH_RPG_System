# Ações em Combate (Combat Actions)

> Sistema de ações e prioridade em combate do HxH RPG.

## Visão Geral

Durante o combate, cada personagem declara uma **Ação (Action)** que será resolvida de acordo com sua velocidade. Ações mais rápidas são processadas primeiro através de uma **Fila de Prioridade (Priority Queue)**.

## Estrutura de uma Ação

Uma ação é composta por:

| Componente | Descrição |
|------------|-----------|
| Ator (Actor) | Personagem que realiza a ação |
| Alvo(s) (Target) | Um ou mais personagens afetados |
| Reação a (ReactTo) | Referência à ação que provocou esta reação |
| Velocidade (Speed) | Determina prioridade de resolução |
| Perícias (Skills) | Perícias utilizadas na ação |
| Gatilho (Trigger) | Condição que pode ativar a ação automaticamente |
| Finta (Feint) | Tentativa de enganar o oponente |
| Movimento (Move) | Deslocamento do personagem |
| Ataque (Attack) | Componente ofensivo |
| Defesa (Defense) | Componente defensivo |
| Esquiva (Dodge) | Componente de evasão |

## Velocidade da Ação (Action Speed)

A velocidade determina a **ordem de resolução** das ações no round. Composta por:
- **Bar** — valor base de velocidade
- **RollCheck** — resultado da rolagem de velocidade

O resultado final (`Speed.Result`) define a posição na fila de prioridade.

## Fila de Prioridade (Priority Queue)

O sistema usa um **max-heap** para ordenar ações por velocidade:

- **Ação com maior velocidade** é processada primeiro
- **Inserção (Insert)** — adiciona ação mantendo a ordenação
- **Extração Máxima (ExtractMax)** — remove e retorna a ação mais rápida
- **Espiar (Peek)** — consulta a próxima ação sem removê-la
- **Extrair por ID (ExtractByID)** — remove uma ação específica (para cancelamentos)

### Fluxo de um Round

```
1. Todos os jogadores declaram suas ações
2. Ações são inseridas na Fila de Prioridade
3. Sistema extrai a ação mais rápida (ExtractMax)
4. Resolve a ação → pode gerar reações
5. Reações entram na fila com sua própria velocidade
6. Repete até a fila esvaziar
```

## Ataque (Attack)

Um ataque contém:
- **Arma** (Weapon) — opcional, determina dados de dano
- **Acerto (Hit)** — rolagem para determinar se acerta
- **Dano (Damage)** — rolagem de dano se acertou
- **Carga (Charge)** — rolagem opcional para ataques carregados
- **Velocidade Relativa** (RelativeVelocity) — diferença de velocidade entre ator e alvo (bônus/penalidade)

## Contexto de Rolagem (Roll Context)

Cada componente de ação que envolve aleatoriedade possui um `RollContext`:
- **Dados** (Dice) — lista de dados a serem rolados
- **Condição** (Condition) — modificadores ou condições especiais
- **Resultado** (Result) — soma de todos os dados rolados

O resultado é calculado como a **soma dos resultados de todos os dados** no contexto.

## Partida (Match)

Uma **Partida (Match)** representa uma sessão de jogo completa:
- Pertence a uma **Campanha (Campaign)**
- É conduzida por um **Mestre (Master)**
- Contém **Cenas (Scenes)** em sequência
- Registra **Eventos (Game Events)**
- Possui datas de início da narrativa (Story) e do jogo (Game)

## Eventos de Jogo (Game Events)

Eventos que ocorrem durante uma partida:

| Categoria | Descrição |
|-----------|-----------|
| Mudança de Data | Avanço do tempo narrativo |
| Morte | Falecimento de um personagem |
| Notícia | Informação que afeta o cenário |
| Ação Desfeita | Reversão de uma ação (ctrl+z) |
| Outro | Eventos diversos |

Cada evento registra:
- **Categorias** — pode ter múltiplas (padrão: "Outro")
- **Título** — descrição curta
- **Descrição** — detalhes opcionais
- **Mudança de Data** — nova data narrativa (opcional)
- **Momento** — timestamp de quando ocorreu
