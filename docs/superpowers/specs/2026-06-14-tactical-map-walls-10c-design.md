# Tactical Map — Paredes: Fase 10-C — Interações como Player Action

**Data:** 2026-06-14
**Status:** Aprovado — pronto para writing-plans
**Escopo:** Cross-repo (`System_X_System` + `System_X_System_React`)
**Spec master:** `2026-06-10-tactical-map-walls-design.md` (seção Fase 10-C)
**Fase anterior:** Fase 10-B (Browse/Draw mode + seleção de paredes — merged)

---

## 1. Visão geral

Fase 10-C transforma paredes em alvos interativos para jogadores. Atacar uma parede,
arrombar uma porta trancada, ou examinar uma porta secreta são `Action`s que entram
na fila de prioridade e são abertas pelo mestre — respeitando iniciativa.

Além da mecânica, esta fase inclui três limpezas arquiteturais acordadas no brainstorm:
renomear `CombatResolver` → `TurnResolver` + migrar estado de mapa do `room.go` para
o `MatchSession` + remover prefixo `lobby_` das MessageTypes.

---

## 2. Decisões de design

| Decisão | Escolha | Motivo |
|---|---|---|
| Ação de parede entra na fila? | Sim — fila de prioridade | Respeita iniciativa; Attack/Interact em parede consome o turno do jogador |
| Quem roteia por tipo de alvo? | `TurnResolver` (domain) via interface `TargetReader` | Regra de negócio pertence ao domínio; delivery só faz broadcast dos resultados |
| `CategorizeTarget` onde? | Método em `MatchSession`; chamado via interface pelo `TurnResolver` | Session tem walls + participants; TurnResolver acessa por interface (sem import circular) |
| 1 ou 2 eventos WS para dano? | 1 — `wall_hp_changed` com payload completo | Cliente deriva "damaged"/"destroyed" dos campos; ver `game-server.instructions.md` |
| Dano rebote | `StructuralDamageResult` retorna `ReboundDamage`; aplicação ao actor é TODO | Requer Attack.Category (melee/range) + Defense do actor — ainda sem contrato finalizado |
| Resistência + guard indestrutível | `effective = max(0, raw−resistance)`; se `MaxHP==0`, skip (indestrutível) | Escalável para paredes mágicas permanentes futuras |
| `Destroyed` quando? | `HP == 0 && MaxHP > 0` | `MaxHP==0` sinaliza indestrutível; `destroyed` é fonte canônica de estado no frontend |
| Persistência de HP | In-memory no `MatchSession`; TODO flush no fechamento do turno | Fluxo de persistência de turno ainda em definição |

---

## 3. Renomeação: `CombatResolver` → `TurnResolver`

`CombatResolver` resolve apenas combate entre personagens — nome errado para um sistema
que calcula dano em estruturas, lockpick, etc.

`TurnResolver` representa o papel real: dado um `Turn` aberto, calcular o resultado
previsto de qualquer `Action` e apresentar ao mestre antes que execute in-game.

`TurnResolver.Resolve` ganha um terceiro parâmetro `TargetReader` e `TurnResolution`
ganha `WallResults`:

```go
// service/turn_resolver.go (renomeado de combat_resolver.go)

type TargetReader interface {
    CategorizeTarget(id uuid.UUID) TargetKind
    GetWall(id string) (mapentity.WallSegment, bool)
}

type WallResult struct {
    UpdatedWall     mapentity.WallSegment
    EffectiveDamage int
    ReboundDamage   int // candidato a rebote no actor — TODO: aplicar se melee
}

type TurnResolution struct {
    ActionResult    RollResult
    ReactionResults []ReactionResult
    Blows           []*battle.Blow
    WallResults     []WallResult // novo
    IsSettled       bool
}

type TurnResolver struct{}

func (tr TurnResolver) Resolve(
    t *turn.Turn,
    sheets map[uuid.UUID]*csSheet.CharacterSheet,
    targets TargetReader,
) *TurnResolution
```

**Arquivos afetados:**
- `domain/match/service/combat_resolver.go` → renomear para `turn_resolver.go`; struct e testes
- `domain/match/service/combat_resolver_test.go` → renomear para `turn_resolver_test.go`
- `domain/match/matchsession/match_session.go` → campo `combatRes` → `turnResolver`; atualizar chamada em `AttachReaction` e `OpenNextAction`
- `application/match/open_next_action.go` → passar `session` como `TargetReader` ao `Resolve`

---

## 4. Migração: estado de mapa do `room.go` → `MatchSession`

### 4.1 Motivação

`r.lobbyWalls`, `r.lobbyPieces`, `r.lobbyGridSize` vivem no delivery layer (`room.go`).
Estado de domínio pertence ao `MatchSession`. Room deve ser roteador WS apenas.

### 4.2 Campos novos em `MatchSession`

```go
walls    map[string]mapentity.WallSegment // keyed by wall UUID string
pieces   map[string]PieceMovedPayload     // keyed by piece_id; tipo renomeado de LobbyPieceMovedPayload
gridSize float64                          // cell size em world coords
```

Métodos novos:
```go
func (s *MatchSession) SyncMapState(pieces []PieceMovedPayload, walls []mapentity.WallSegment, gridSize float64)
func (s *MatchSession) GetWall(id string) (mapentity.WallSegment, bool)
func (s *MatchSession) UpdateWall(w mapentity.WallSegment)
func (s *MatchSession) GetPieces() []PieceMovedPayload
func (s *MatchSession) MovePiece(p PieceMovedPayload)
func (s *MatchSession) RemovePiece(id string)
func (s *MatchSession) GetGridSize() float64
```

`room.go` continua detendo `r.mu`; acessa session sempre sob lock.

### 4.3 Remoção de `r.lobbyWalls`, `r.lobbyPieces`, `r.lobbyGridSize`

Após migração, campos removidos do `Room` struct. Referências em `room.go` delegam para
`r.session.*`. Movimento de bloqueio de caminho (`IsPathBlocked`) também passa a ler
via session.

### 4.4 Remoção do prefixo `lobby_` nas MessageTypes e tipos

Prefixo "lobby" descreve uma fase histórica de desenvolvimento, não o domínio.

| Antes (MessageType) | Depois |
|---|---|
| `lobby_piece_moved` | `piece_moved` |
| `lobby_piece_removed` | `piece_removed` |
| `lobby_state_sync` | `map_state_sync` |
| `lobby_full_state` | `map_full_state` |

| Antes (tipo Go) | Depois |
|---|---|
| `LobbyPieceMovedPayload` | `PieceMovedPayload` |
| `LobbyPieceRemovedPayload` | `PieceRemovedPayload` |
| `LobbyPiecesPayload` | `MapPiecesPayload` |
| `LobbyStateSyncPayload` | `MapStateSyncPayload` |

**Impacto cross-repo:** contrato WS muda → PR paired com frontend. Atualizar strings de
MessageType em `useMatchMapWs.ts` (e correlatos) na mesma sessão de implementação.

---

## 5. `CategorizeTarget`

Vive em `MatchSession` (único lugar com walls + participants). Exposto via interface
`TargetReader` definida em `service/` — sem import circular.

```go
// matchsession/match_session.go

type TargetKind string

const (
    TargetKindCharacter   TargetKind = "character"
    TargetKindWallSegment TargetKind = "wall_segment"
    TargetKindUnknown     TargetKind = "unknown"
    // TODO: TargetKindFloorTile, TargetKindItem — fases futuras (Decorations, Items)
)

func (s *MatchSession) CategorizeTarget(id uuid.UUID) TargetKind {
    if _, ok := s.participants[id]; ok {
        return TargetKindCharacter
    }
    if _, ok := s.walls[id.String()]; ok {
        return TargetKindWallSegment
    }
    return TargetKindUnknown
}
```

`MatchSession` implementa `service.TargetReader` implicitamente (Go interfaces).

---

## 6. `ApplyStructuralDamage`

Função stateless em `domain/match/service/structural_damage.go`:

```go
type StructuralDamageResult struct {
    UpdatedWall     mapentity.WallSegment
    EffectiveDamage int // dano aplicado na parede (≥ 0)
    ReboundDamage   int // = min(rawDamage, Resistance) — candidato a rebote no actor
    // TODO: aplicar ReboundDamage no actor somente se ataque melee (verificar Attack.Category)
    // TODO: subtrair Defense do actor do ReboundDamage antes de aplicar
    // TODO: incluir resultado de rebote no broadcast (evento separado ou enriquecer wall_hp_changed)
}

// ApplyStructuralDamage applies raw attack damage to a WallSegment, respecting
// material resistance. MaxHP==0 signals an indestructible wall (no HP system).
func ApplyStructuralDamage(w mapentity.WallSegment, rawDamage int) StructuralDamageResult {
    if w.MaxHP == 0 {
        // indestructible — no HP system; full rebound
        // TODO: se range attack, ReboundDamage = 0
        return StructuralDamageResult{UpdatedWall: w, EffectiveDamage: 0, ReboundDamage: rawDamage}
    }
    effective := rawDamage - w.Resistance
    if effective < 0 {
        effective = 0
    }
    rebound := rawDamage - effective // = min(rawDamage, Resistance)
    w.HP -= effective
    if w.HP < 0 {
        w.HP = 0
    }
    if w.HP == 0 && w.MaxHP > 0 {
        w.Destroyed = true
    }
    // TODO: persistir novo estado de HP no snapshot de maps ao fechar o turno.
    // Ver PersistTurnClose — aguardar finalização do fluxo de persistência de turno.
    return StructuralDamageResult{UpdatedWall: w, EffectiveDamage: effective, ReboundDamage: rebound}
}
```

---

## 7. Fluxo de Action em parede — backend

O domínio roteia; o delivery layer só broadcast.

```
player envia enqueue_action { Attack | Interact, TargetID: [wallUUID] }
  ↓
room.go → buildAction() → EnqueueActionUC (fila de prioridade — sem mudança)

  [mestre abre o turno]

mestre envia open_next_action
  ↓
OpenNextActionUC:
  session.OpenNextAction() → Turn aberto
  turnResolver.Resolve(turn, charSheets, session)  ← session como TargetReader
    internamente, para cada TargetID:
      switch session.CategorizeTarget(targetID):
        case TargetKindCharacter:
          // rolls, blows (fluxo existente — sem mudança)
        case TargetKindWallSegment:
          wall, _ = session.GetWall(targetID)
          if action.Attack != nil:
            // rawDamage vem de Attack payload
            // TODO: mapear Attack.Damage quando contrato de Attack finalizar
            result = ApplyStructuralDamage(wall, rawDamage)
            session.UpdateWall(result.UpdatedWall)
            res.WallResults = append(res.WallResults, WallResult{...})
          if action.Interact != nil:
            // open/close/lockpick — reutiliza lógica de applyWallInteract
        case TargetKindUnknown:
          registrar erro na resolution
  retorna OpenNextActionResult { ClosedTurn, OpenedTurn, Resolution }
  ↓
room.go recebe resolution:
  broadcast turn_opened
  para cada WallResult em resolution.WallResults:
    broadcast wall_hp_changed { wallID, hp, maxHp, destroyed }
  para cada wall interact em resolution:
    broadcast wall_state_changed { wallID, open, locked }
  se TargetKindUnknown → SendMessage(error "invalid_target") ao client
```

---

## 8. Evento WS: `wall_hp_changed`

```go
// message.go — adicionar

MsgTypeWallHpChanged MessageType = "wall_hp_changed"

type WallHpChangedPayload struct {
    WallID    string `json:"wall_id"`
    HP        int    `json:"hp"`
    MaxHP     int    `json:"max_hp"`
    Destroyed bool   `json:"destroyed"`
}
```

O cliente deriva "danificada" (hp < maxHp/2) e "destruída" (destroyed == true) dos
campos — sem necessidade de eventos separados. Ver princípio em `game-server.instructions.md`.

---

## 9. Frontend

### 9.1 `TacticalMapViewer.tsx` — action picker

Tap em `WallSegment` abre menu contextual com opções filtradas por `wallType`:

| wallType | Opções visíveis |
|---|---|
| wall / terrain | "Atacar" (se `maxHp > 0 && !destroyed`) |
| door (fechada, não trancada) | "Abrir", "Atacar" |
| door (trancada) | "Arrombar fechadura", "Atacar" |
| window | "Abrir", "Atacar" |
| secret_door | não renderizada para jogadores |

Ao confirmar, envia `enqueue_action { Attack | Interact, TargetID: [wallID] }` via WS.

### 9.2 `useMatchMapWs.ts` — novos eventos

Processar:
- `wall_hp_changed` → atualiza `WallSegment.hp` + `WallSegment.destroyed` no store local
- `wall_state_changed` → atualiza `open`/`locked` (já existia; verificar se precisa ajuste)
- MessageTypes renomeados (`map_state_sync`, `map_full_state`, `piece_moved`, `piece_removed`)

### 9.3 Transições visuais em `WallsLayer.tsx`

| Estado | Condição | Visual |
|---|---|---|
| intacta | `hp == maxHp` | cor cheia, opacidade 1.0 |
| danificada | `hp > 0 && hp < maxHp` | tracejada, opacidade 0.8 |
| destruída | `destroyed == true` | pontilhada fina, opacidade 0.4, marcas × nos endpoints |
| indestrutível | `maxHp == 0` | cor cheia, opacidade 1.0 (sem barra de HP) |

Transição animada ~300ms via tween de opacidade. `destroyed` é a fonte canônica — não derivar de `hp == 0`.

---

## 10. Documentação

- `game-server.instructions.md`: WS Event Design rule + MatchSession description ✅
- `docs/player/walls.md` (criar): tipos de parede, interações disponíveis, custo de ação
- Docs que mencionem "turn engine" ou `CombatResolver`: atualizar para `RoundOrchestrator` + `TurnResolver`
- `docs/documentation-map.yaml`: registrar `structural_damage.go`, `turn_resolver.go`, evento `wall_hp_changed`

---

## 11. Fora de escopo (10-C)

- Persistência de HP no banco ao fechar turno — TODO em `ApplyStructuralDamage` e `PersistTurnClose`
- Aplicação de `ReboundDamage` no actor — TODO em `StructuralDamageResult`
- `examine` em `secret_door` — depende de roll de Skill; aguarda fluxo de Skills
- `lockpick` completo — idem
- Fog of War / LOS — Fase 10-D

---

## 12. Critério de pronto

- Jogador toca parede de madeira no viewer → "Atacar" → intent enviado com `Attack + TargetID=[wallUUID]` → mestre abre o turno → dano aplicado com resistência → todos os clientes veem estado "danificada"; ao `destroyed=true` → visual pontilhado com ×.
- `map_state_sync` (renomeado) funciona corretamente no reload/reconexão.
- `MatchSession` detém walls/pieces/gridSize; `r.lobbyWalls` etc. removidos do `Room`.
- `TurnResolver` substitui `CombatResolver` sem regressão nos testes existentes.
- `TargetKindCharacter` é o primeiro case nos switches de `CategorizeTarget` / `TurnResolver`.
