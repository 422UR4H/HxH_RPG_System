# Fluxos do Bounded Context `match`

Diagramas (Mermaid) do **estado atual do código** — não é design aspiracional.
Atualizados na **Fase 5** (`03` e `05` reescritos; `01` e `04` corrigidos).
Complementam os docs textuais em `docs/dev/match/` (`actions.md`, `turns-rounds.md`,
`scenes.md`, `roster.md`), que descrevem *o que cada tipo é*. Aqui descrevemos
*quem chama quem, com qual estado, e onde o fluxo ainda não existe*.

| Arquivo | Pergunta que responde |
|---------|-----------------------|
| [`01-mapa-de-modulos.md`](01-mapa-de-modulos.md) | Quais são os módulos de `domain/match` e `application/match` e como eles dependem uns dos outros? |
| [`02-ciclo-de-vida-da-partida.md`](02-ciclo-de-vida-da-partida.md) | Do `start_match` ao `MatchSession` vivo: quem constrói o quê, e quando |
| [`03-fluxo-de-acao.md`](03-fluxo-de-acao.md) | O caminho completo de uma ação: enfileirar → abrir turno → reagir → resolver → persistir |
| [`04-estado-e-persistencia.md`](04-estado-e-persistencia.md) | Quem é dono de qual estado (Room × MatchSession × Postgres) e o que sobrevive a um restart |
| [`05-lacunas.md`](05-lacunas.md) | Onde o motor está oco hoje — o mapa do que falta para "dar vida à partida" |

## Convenções dos diagramas

- **Retângulo** = tipo com estado (entidade / `MatchSession`).
- **Retângulo tracejado** = tipo **stateless** (domain service, use case).
- **Cilindro** = Postgres.
- **Seta cheia** = chamada direta (dependência de compilação).
- **Seta tracejada** = dependência invertida (interface declarada pelo consumidor).
- ⚠️ = ponto onde o código atual é *stub* ou está inconsistente. Detalhado em `05-lacunas.md`.

## E as regras?

Estes arquivos respondem **quem chama quem, com qual estado**. O *porquê* de cada regra —
os dois porteiros, a chave de prioridade, o rótulo ser o vazamento, a escada do repelir —
mora em [`../combat-engine.md`](../combat-engine.md).

## Leitura mínima para entrar no assunto

`01` → `03` → `05`. Os outros dois são referência.
