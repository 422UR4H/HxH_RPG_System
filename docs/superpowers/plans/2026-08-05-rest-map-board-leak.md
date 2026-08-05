# `GET /maps/:id` vaza o tabuleiro inteiro para jogadores (backend) — Plano

> **Para quem implementa:** execute tarefa por tarefa, na ordem. Leia a seção de contexto
> inteira antes de escrever qualquer linha — ela contém o fato de arquitetura que
> determina a solução, e sem ele a correção "óbvia" é impossível de escrever.

**Objetivo:** parar de servir conteúdo de tabuleiro (peças e paredes) a quem não é mestre
no `GET /maps/:id`, e passar a elevação da peça no payload WS para que o cliente não
precise mais do REST para isso.

**Branch:** `fix/rest-map-board-leak` (já criada)

**Ordem obrigatória:** este plano vem **antes** do plano do frontend
(`System_X_System_React/docs/superpowers/plans/2026-08-05-rest-map-board-leak-frontend.md`).
O contrato é o handoff.

**Arquivos:**
- Modificar: `internal/app/api/map/get_map.go`
- Modificar: `internal/app/api/map/get_map_test.go`
- Modificar: `internal/app/game/message.go`
- Modificar: `docs/dev/api/maps.md`
- Modificar: `docs/documentation-map.yaml`

---

## 1. O problema

`GET /maps/:id` devolve **todas** as paredes e **todas** as peças visíveis do mapa para
qualquer participante autenticado. `applyRoleFilter` mascara porta secreta não revelada e
remove peça `visible=false` — mas **nunca aplica linha de visão**.

Um jogador que abrir o devtools, ou der um `curl` no endpoint com o próprio token, lê o
mapa inteiro: onde está cada personagem e onde está cada parede, inclusive o que o
personagem dele não vê.

Já existe uma mitigação **no cliente** (`visibleBoardPieces`, em
`src/features/tactical-map/utils/boardSource.ts`): o front não *usa* mais o payload REST
para montar o tabuleiro do jogador. Isso parou de exibir o vazamento, mas **a API continua
entregando o dado**. Este plano fecha a fonte.

O próprio código já sabia — o comentário em `get_map.go` diz:

```go
// LOS-at-REST is deferred to the WS layer (live match) which has exact per-player polygons.
// TODO(10-D): wire live fog state here once a /maps/:id REST endpoint is called mid-match.
```

## 2. O fato de arquitetura que decide tudo

> **REST e game server são processos separados.**

`cmd/api/` (porta 5000) e `cmd/game/` (porta 8081) são binários distintos, e
`internal/app/api/` **não importa** `internal/app/game/` — verifique você mesmo antes de
começar:

```bash
grep -rln "app/game" internal/app/api/ cmd/api/   # deve não retornar nada
```

Os polígonos de visibilidade vivem em `MatchSession`, na RAM do processo do game server, e
são recalculados a cada movimento. O processo do REST **não tem como** enxergá-los, e eles
não são persistidos em lugar nenhum.

**Consequência: não existe "filtrar por LOS no REST".** Se você se pegar tentando alcançar
a sessão a partir do handler HTTP, parou no lugar errado.

### Alternativas rejeitadas

| Opção | Por que não |
|---|---|
| Filtrar por LOS no handler REST | Impossível — o dado está em outro processo |
| Game server expõe endpoint interno que o REST consulta | Cria dependência de runtime da API no game server, exige auth entre serviços, e a resposta já nasce velha (a LOS muda no próximo movimento). Trabalho grande para entregar o que o WS já entrega |
| Persistir os polígonos no banco | Mudam a cada movimento — escrita quente para responder uma pergunta que o WS já responde melhor |
| Negar `GET /maps/:id` a jogador (403) | O jogador precisa de grid e background para desenhar o mapa |

## 3. A solução

**O REST devolve a *casca* do mapa para quem não é mestre: grid, background, `fog_mode`,
nome — com `pieces` e `walls` vazios.**

Isso não é uma mitigação, é um estreitamento: o endpoint deixa de carregar um dado que
nunca teve como filtrar corretamente. O game server já calcula exatamente isso por jogador
e empurra em `map_full_state`, **tanto no lobby quanto na partida** — ele é o único lugar
que consegue, então passa a ser o único que o faz.

### O efeito colateral que precisa entrar na mesma mudança

O cliente hoje restaura a elevação da peça (`z`) a partir do mapa REST, porque o payload
WS não a carrega (`PieceMovedPayload` não tem campo de elevação). Cortar as peças do REST
**sem mais nada achata todas as peças do jogador no chão**.

Por isso `PieceMovedPayload` ganha `z` nesta mesma tarefa. O game server trata a elevação
como **passagem opaca**, exatamente como já faz com `slot`: elevação não participa de LOS
(a varredura é 2D), então nada na lógica de visibilidade muda.

---

## Task 1: `applyRoleFilter` devolve só a casca

**Arquivos:** `internal/app/api/map/get_map.go`

- [ ] **Passo 1: substituir a função inteira**

Localize `func applyRoleFilter` (junto com todo o bloco de comentário acima dela) e
substitua por:

```go
// applyRoleFilter mutates the map in-place to remove information the viewer must not see.
//
// Master: no filtering — receives the full unmasked board.
//
// Non-master: receives the map SHELL only — grid, background, fog mode — with no pieces
// and no walls at all.
//
// That is deliberate, and it is not over-caution. GET /maps/:id is served by the REST
// process (cmd/api). Everything needed to decide what a player may see — the per-player
// visibility polygons — lives in MatchSession, in the RAM of a DIFFERENT process
// (cmd/game), and is recomputed on every move. REST cannot filter by line of sight and
// never could, so it used to answer with the whole board: any player could read every
// wall and every character position straight off the endpoint with their own token.
//
// The game server already computes exactly this, per player, and pushes it over
// map_full_state — in the lobby as well as in a live match. It is the only place that
// can, so it is the only place that does.
func applyRoleFilter(m *mapentity.TacticalMap, isMaster bool) {
	if isMaster {
		return
	}
	// Empty slices, not nil: the JSON stays `[]` instead of flipping to `null`, so no
	// client has to learn a second shape for "nothing here".
	m.Pieces = []mapentity.Piece{}
	m.Walls = []mapentity.WallSegment{}
}
```

- [ ] **Passo 2: remover o import que ficou órfão**

`matchservice` era usado só por `MaskSecretDoorForPlayer`, que saiu junto. Remova a linha:

```go
	matchservice "github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
```

> **Não apague `MaskSecretDoorForPlayer` do domínio.** Ela continua sendo usada por
> `FilterMapState`, no caminho do WS, que é agora o único filtro real. Só o *uso* no REST
> desapareceu.

- [ ] **Passo 3: compilar**

```bash
go build ./... && go vet ./internal/app/api/...
```

---

## Task 2: elevação no payload WS

**Arquivos:** `internal/app/game/message.go`

- [ ] **Passo 1: acrescentar o campo**

```go
type PieceMovedPayload struct {
	PieceID     string      `json:"piece_id"`
	Slot        SlotPayload `json:"slot"`
	CharacterID string      `json:"character_id,omitempty"`
	Visible     *bool       `json:"visible,omitempty"`
	// Z is the piece's virtual height in metres (0 = ground). The game server never
	// reads it: line of sight is computed in 2D, so elevation rides through as opaque
	// passthrough exactly like Slot does. It is on the wire because the client used to
	// recover it from GET /maps/:id, and that endpoint no longer hands a player any
	// pieces — without this field every piece would flatten to the ground for players.
	Z float64 `json:"z,omitempty"`
}
```

> `omitempty` suprime `z: 0`, e o cliente já trata ausência como 0 (chão). As duas pontas
> concordam, e o frame não carrega o caso mais comum.

- [ ] **Passo 2: confirmar que não é preciso mexer em mais nada**

`room.go` guarda `r.pieces` como `PieceMovedPayload` e o repassa inteiro em
`buildMapFullState` e em `handlePieceMoved`. Acrescentar o campo à struct já faz a
elevação atravessar `map_state_sync` → estado do room → `map_full_state`. **Não** adicione
`z` a `PieceVisibility` nem a nada em `internal/domain/match/service/` — elevação não entra
em LOS.

```bash
grep -rn "PieceMovedPayload{" internal/app/game/ | grep -v _test
```

Confirme que os pontos que constroem o payload copiam a struct inteira (ou usam o valor
guardado em `r.pieces`), e não montam campo a campo. Se algum montar campo a campo,
acrescente `Z` lá.

- [ ] **Passo 3: build + testes**

```bash
go build ./... && go test ./internal/app/game/... 2>&1 | tail -5
```

---

## Task 3: testes

**Arquivos:** `internal/app/api/map/get_map_test.go`

O arquivo hoje tem testes que afirmam coisas que **mudaram de significado**: que o
não-mestre recebe porta secreta mascarada e não recebe peça invisível. Agora ele não
recebe parede nem peça alguma — garantia estritamente mais forte.

**Não apague o arquivo.** Reescreva as afirmações do caminho não-mestre e preserve
intactas as do mestre.

- [ ] **Passo 1: ler o arquivo inteiro antes de editar**

```bash
sed -n '1,120p' internal/app/api/map/get_map_test.go
```

Ele usa `humatest`. Siga a montagem que já existe lá — **não invente helper novo**.

- [ ] **Passo 2: substituir as asserções do não-mestre**

O teste do jogador passa a afirmar:

```go
	// A player gets the map shell and nothing else. The REST process has no access to
	// the visibility polygons (they live in the game server's memory), so it cannot
	// decide which wall or which piece this player may see — and therefore sends none.
	// map_full_state over the WS is what fills the board.
	pieces, ok := mapObj["pieces"].([]any)
	if !ok {
		t.Fatalf("pieces missing from response: %v", mapObj)
	}
	if len(pieces) != 0 {
		t.Fatalf("player must receive no pieces from REST, got %d", len(pieces))
	}

	walls, ok := mapObj["walls"].([]any)
	if !ok {
		t.Fatalf("walls missing from response: %v", mapObj)
	}
	if len(walls) != 0 {
		t.Fatalf("player must receive no walls from REST, got %d", len(walls))
	}

	// The shell is still there — without it the client cannot draw the map at all.
	if mapObj["grid"] == nil {
		t.Fatal("player must still receive the grid")
	}
```

- [ ] **Passo 3: o teste do mestre fica como está**

Ele afirma que o mestre recebe tudo, inclusive peça invisível e porta secreta com o tipo
real. Continua valendo palavra por palavra. **Não relaxe nada nele.**

- [ ] **Passo 4: rodar**

```bash
go test ./internal/app/api/map/... -v 2>&1 | tail -20
```

---

## Task 4: contrato e mapa de documentação

**Arquivos:** `docs/dev/api/maps.md`, `docs/documentation-map.yaml`

- [ ] **Passo 1: `docs/dev/api/maps.md`**

Na seção de `GET /maps/:id`, deixe explícito o que cada papel recebe. Acrescente:

```markdown
**O que cada papel recebe.** O mestre recebe o mapa completo, sem máscara. Quem não é
mestre recebe apenas a **casca**: grid, background, `fog_mode`, nome — com `pieces` e
`walls` **vazios**.

Isso não é uma limitação temporária. `GET /maps/:id` é servido pelo processo REST
(`cmd/api`), e o que decide o que um jogador pode ver — os polígonos de visibilidade —
vive na memória do processo do game server (`cmd/game`), recalculado a cada movimento. O
REST não tem como filtrar por linha de visão, então não envia tabuleiro nenhum.

O tabuleiro do jogador chega **exclusivamente** por `map_full_state` no WebSocket, tanto
no lobby quanto em partida. Ver a seção de eventos WS abaixo.
```

Na tabela de campos do `map_full_state`, acrescente `z` à descrição de `pieces` (elevação
em metros, 0 = chão, omitido quando 0).

- [ ] **Passo 2: `docs/documentation-map.yaml`**

Acrescente uma entrada para `internal/app/api/map/get_map.go` apontando para
`docs/dev/api/maps.md` com `confidence: directly_affected`, e uma nota registrando que o
filtro de papel do REST é casca-apenas por causa da separação de processos. Leia o arquivo
antes e **imite a estrutura existente — não invente campos**.

- [ ] **Passo 3: commit**

```bash
git add internal/ docs/
git commit -m "fix(api): GET /maps/:id devolve só a casca do mapa para quem não é mestre"
```

---

## Verificação final do backend

```bash
go build ./... && go test ./... 2>&1 | tail -10
golangci-lint run
```

Depois, smoke test manual — é o que prova que o vazamento fechou de verdade:

```bash
# com o token de um JOGADOR (não do mestre) numa partida com mapa anexado
curl -s -H "Authorization: Bearer $PLAYER_TOKEN" http://localhost:5000/maps/$MAP_ID \
  | python3 -c "import json,sys; m=json.load(sys.stdin)['map']; print('pieces:', len(m['pieces']), 'walls:', len(m['walls']), 'grid:', m['grid'] is not None)"
```

Esperado: `pieces: 0 walls: 0 grid: True`.

Com o token do **mestre**, os dois primeiros devem ser diferentes de zero.

Só depois disso siga para o plano do frontend.
