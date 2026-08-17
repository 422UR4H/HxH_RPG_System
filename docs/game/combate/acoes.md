# Ações em Combate

> Sistema de ações e prioridade em combate do HxH RPG.

## Visão Geral

Durante o combate, cada personagem declara uma **Ação** que será resolvida de acordo com sua velocidade. Ações mais rápidas são processadas primeiro através de uma **Fila de Prioridade**.

## Estrutura de uma Ação

Uma ação é composta por:

| Componente | Descrição |
|------------|-----------|
| Ator | Personagem que realiza a ação |
| Alvo(s) | Um ou mais personagens afetados |
| Reação a | Referência à ação que provocou esta reação |
| Velocidade | Determina prioridade de resolução |
| Perícias | Perícias utilizadas na ação |
| Gatilho | Condição que pode ativar a ação automaticamente |
| Finta | Tentativa de enganar o oponente |
| Movimento | Deslocamento do personagem |
| Ataque | Componente ofensivo |
| Defesa | Componente defensivo |
| Esquiva | Componente de evasão |

## Velocidade da Ação

A velocidade determina a **ordem de resolução** das ações na rodada. Composta por:
- **Barra** — valor base de velocidade
- **Rolagem de Verificação** — resultado da rolagem de velocidade

O resultado final define a posição na fila de prioridade.

## Fila de Prioridade

O sistema usa uma fila de prioridade ordenada por velocidade para resolver as ações:

- **Ação com maior velocidade** é processada primeiro
- Ações são adicionadas à fila mantendo a ordenação
- A ação mais rápida é processada e removida da fila
- É possível consultar a próxima ação sem removê-la
- Uma ação específica pode ser cancelada e removida da fila

### Fluxo de uma Rodada

1. Os jogadores declaram suas ações — **a qualquer momento**, inclusive durante o turno dos
   outros. Só o **round 0** tem um momento reservado para isso.
2. O sistema rola a velocidade de cada ação assim que ela chega e a enfileira.
3. O mestre **abre** a ação do topo da fila — o que fecha o turno anterior.
4. Todos veem a mecânica da ação (alvos, arma, perícia); só o mestre vê o resultado. É a vez
   do dono narrar.
5. Os alvos podem reagir. Cada reação é aberta pelo mestre, e seu dono narra.
6. O mestre encerra o turno, e o resultado aparece para todos.

> A fila **nunca esvazia por si só** — ela é permanentemente viva. Quem não declarou nada
> apenas não age naquele round.

Detalhes da economia de turno em [Barra de Ação](barra-de-acao.md); catálogo completo de
respostas em [Reações](reacoes.md).

## Ataque

Um ataque contém:
- **Arma** — opcional, determina dados de dano
- **Acerto** — rolagem para determinar se acerta
- **Dano** — rolagem de dano se acertou
- **Carga** — rolagem opcional para ataques carregados
- **Velocidade Relativa** — diferença de velocidade entre ator e alvo (bônus/penalidade)

## Contexto de Rolagem

Cada componente de ação que envolve aleatoriedade possui um contexto de rolagem:
- **Dados** — lista de dados a serem rolados
- **Condição** — modificadores ou condições especiais
- **Resultado** — soma de todos os dados rolados

O resultado é calculado como a **soma dos resultados de todos os dados** no contexto.

## Partida

Uma **Partida** representa uma sessão de jogo completa:
- Pertence a uma **Campanha**
- É conduzida por um **Mestre**
- Contém **Cenas** em sequência
- Registra **Eventos de Jogo**
- Possui datas de início da narrativa e do jogo

## Eventos de Jogo

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
- **Momento** — registro de quando ocorreu

---

> **🔧 Para Desenvolvedores**
>
> Implementação técnica: [`docs/dev/match/actions.md`](../../dev/match/actions.md)
> Código-fonte: `internal/domain/entity/match/`
