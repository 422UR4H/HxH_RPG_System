# `fog_mode` — documentar como pendência de configuração de partida (backend)

> **Para quem implementa:** esta é uma tarefa **exclusivamente de documentação**.
> **Nenhuma linha de código executável muda.** Nenhum teste novo, nenhum teste
> alterado, nenhum comportamento diferente. Se você se pegar editando lógica, parou de
> seguir o plano.

**Spec de referência (leia a §3 antes de começar):**
`System_X_System_React/docs/superpowers/specs/2026-08-06-tactical-map-refactor-design.md`
(repo do frontend)

**Branch:** `docs/fog-mode-pendencia-config-partida`

**Repo:** `System_X_System` apenas. Independente da Fase 1 do frontend — pode ir antes,
depois ou em paralelo.

**Arquivos:**
- Modificar: `internal/app/game/room.go` (só comentários)
- Modificar: `internal/domain/map/entity/map.go` (só comentário)
- Modificar: `AGENTS.md`

---

## Por que esta tarefa existe

Numa revisão de código, `fog_mode` foi diagnosticado à primeira vista como fiação
morta: existe coluna no Postgres, existe `fog.FogMode`, existe `ValidateFogMode`,
existe mapeamento no `pgmap.toEntity` — mas **nenhum request ou response REST carrega
o campo**, e `room.go` passa `FogModeExplored` fixo para a sessão. A conclusão errada e
tentadora é "código morto, apagar".

**Apagar seria um erro.** `internal/domain/match/service/filter_map_state.go:115` é o
único ponto que separa os dois modos de fog:

```go
if !seen && fogMode == fog.FogModeExplored {
    seen = memory.Has(fog.FeatureWall, w.ID)
}
```

Sem `FogMode`, o modo `live` (paredes somem ao sair da linha de visão) deixa de existir
no produto. E o spec de 2026-08-05
(`docs/superpowers/specs/2026-08-05-tactical-map-wall-memory-design.md:137`) diz
explicitamente que `fog_mode` **permanece** e continua significativo.

**Decisão do dono do produto (2026-08-06):** `fog_mode` será uma configuração de
**partida (match)**, escolhida pelo mestre ao criar ou editar a partida. O mecanismo de
configurações de campanha/partida ainda não existe no backend — há apenas espaço e um
template inicial no frontend. Até ele existir, o hardcode fica.

O objetivo desta tarefa é que a próxima pessoa (ou agente) que olhar para esse código
**não repita o diagnóstico errado.**

---

## Task 1 — Comentar o hardcode em `room.go`

Há **dois** pontos onde `FogModeExplored` é passado fixo. Encontre-os com:

```
grep -n "SyncPlayerMemories(nil, fogentity.FogModeExplored)" internal/app/game/room.go
```

Devem aparecer por volta das linhas **176** e **318**. Na linha 317 já existe um `TODO`
parcial — ele será substituído pelo texto abaixo.

Em **cada** um dos dois pontos, coloque este comentário imediatamente acima da chamada:

```go
// fogMode fixo em explored: PENDENTE de configurações de partida.
// FogMode é real e usado (filter_map_state.go decide memória de parede por ele), mas
// o valor persistido em maps.fog_mode nunca chega aqui porque não existe ainda o
// mecanismo de configuração de campanha/partida no backend — fog_mode será uma opção
// que o mestre escolhe ao criar/editar a partida. Quando esse mecanismo existir, ler o
// modo da configuração da partida e passar aqui. NÃO remover FogMode achando que é
// código morto: isso eliminaria o modo `live` do produto.
// Ver: System_X_System_React/docs/superpowers/specs/2026-08-06-tactical-map-refactor-design.md §3
r.session.SyncPlayerMemories(nil, fogentity.FogModeExplored)
```

Se na linha ~317 houver um `TODO` antigo sobre isso, **substitua-o** por este texto —
não empilhe os dois.

**Cuidado:** não altere a chamada em si. O argumento continua
`fogentity.FogModeExplored`.

---

## Task 2 — Comentar o campo na entidade

**Arquivo:** `internal/domain/map/entity/map.go`, linha ~21.

Hoje:

```go
	FogMode     fog.FogMode  `json:"fog_mode"` // default FogModeExplored when zero
```

Passa a ser:

```go
	// Persisted in maps.fog_mode and honoured by FilterMapState, but not yet exposed in
	// any REST request/response: fog_mode is slated to become a per-match setting chosen
	// by the master, and the campaign/match settings mechanism does not exist yet. The
	// game server hardcodes explored until then (see room.go). Do not remove — dropping
	// FogMode would remove the `live` fog mode from the product.
	// See: System_X_System_React/docs/superpowers/specs/2026-08-06-tactical-map-refactor-design.md §3
	FogMode     fog.FogMode  `json:"fog_mode"` // default FogModeExplored when zero
```

---

## Task 3 — Registrar em `AGENTS.md`

**Arquivo:** `AGENTS.md`, seção `## Known Issues`.

Acrescente um bloco novo **depois** do bloco `**Deferred to Phase 4:**` existente. Não
mexa no que já está lá.

```markdown
**Pendente de configurações de campanha/partida:**
- `fog_mode` (`live` | `explored`) é persistido em `maps.fog_mode` e honrado por
  `FilterMapState`, mas nenhum endpoint REST o expõe e `room.go` hardcoda `explored`.
  Será uma configuração de **partida**, escolhida pelo mestre — o mecanismo de
  configurações ainda não existe no backend. **Não remover** `FogMode`: isso eliminaria
  o modo `live`. Ver spec do refactor do mapa (repo do front), §3.
```

Mantenha curto — `AGENTS.md` é carregado em toda sessão e cada linha custa contexto.

---

## Verificação

1. `go build ./...` — compila.
2. `go vet ./...` — limpo.
3. `go test ./...` — verde. **Baseline: 1228 testes passando em 69 pacotes.** Como esta
   tarefa só mexe em comentários, o número tem que bater **exatamente**. Se mudar,
   você editou código sem querer — desfaça e refaça.
4. `git diff` — revise e confirme que **toda** linha alterada é comentário, string de
   markdown, ou linha em branco. Nenhuma expressão, nenhuma chamada, nenhuma
   declaração.

---

## Entrega

1. `./dev-checkout.sh docs/fog-mode-pendencia-config-partida` a partir de
   `System_X_System_Project/`.
2. **Smoke test não se aplica** — não há mudança de comportamento a verificar. Registre
   isso no corpo do PR, junto do resultado de `go test ./...`, como a evidência
   equivalente prevista no `CLAUDE.md` da raiz.
3. Abrir o PR.

**Título do PR:** `docs(fog): registrar fog_mode como pendência de configuração de partida`

No corpo: explique em duas frases por que o campo parece morto e não é, e cole a saída
de `go test ./...`.

---

## O que NÃO fazer

- **Não** adicionar `fog_mode` ao `MapResponse` nem aos requests de create/update.
  Isso é a feature de configurações de partida, não esta tarefa.
- **Não** fazer `room.go` ler o `fog_mode` do mapa. Pelo mesmo motivo.
- **Não** remover nada: nem `FogMode`, nem `ValidateFogMode`, nem a coluna, nem o
  mapeamento em `pgmap`.
- **Não** aproveitar para tipar `MapResponse.Bg/Pieces/Decorations/Items` (hoje `any`).
  Está registrado no spec como C2, para a próxima mudança de contrato de mapa.
