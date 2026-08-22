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

## A corrente de testes

Uma ação pode juntar várias perícias. *"Dou um mortal por cima dele e corto com a espada
enquanto caio"* são duas perícias além do ataque: Acrobacia e o golpe. **Cada perícia é um
teste**, e eles acontecem em corrente, um alimentando o próximo.

### De onde vem a dificuldade

| Tipo de teste | Quem define a CD |
|---|---|
| Direto contra outro personagem — acerto × esquiva, dano × defesa | **o adversário**: a CD é o resultado dele, e a conta é subtração direta |
| Tudo o mais — se você consegue mesmo dar o mortal que descreveu | **o mestre**, na hora, a olho |

Isso é de propósito. O mestre é quem está lendo a cena; ele decide o que aquele mortal exige
naquele chão, naquele momento, com aquele inimigo em cima.

### O resultado atravessa

**O que sobra de um teste entra no próximo.** Passou com folga, a folga ajuda o golpe
seguinte. Ficou negativo, o negativo pode ser descontado do próximo. Sua ação não é uma lista
de testes independentes — é uma sequência em que você vai ganhando ou perdendo terreno.

### Quando a corrente morre

**Errar por 10 ou mais faz você falhar na própria ação**, e os testes seguintes não acontecem.
Se o mortal deu muito errado, não há golpe de espada para testar — você está no chão.

> É o mesmo degrau de 10 que aparece no repelir e no resto do sistema. Não é coincidência: é
> a régua com que este sistema mede "errou" contra "errou feio".

**E o mestre pode mexer nisso.** Ele pode mudar a margem que mata a corrente, ou simplesmente
deixar a corrente seguir mesmo depois do erro grande, se a cena pedir.

### O que acontece com quem falha

Ainda não está fechado. As possibilidades desenhadas são ficar de guarda aberta, cair no chão,
receber dano igual à diferença — e **quem decide é o mestre**. O que o sistema faz é propor um
padrão razoável para ele aceitar ou substituir.

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
