package matchsession

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"
	"log"
	"maps"
	"slices"
	"strings"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

// Efeitos de status na memória da sessão, por ficha (a mesma chave de charSheets). Quem chama
// segura o r.mu; a sessão não tem lock próprio. Toda leitura devolve cópias.

func cloneEffects(es []statuseffect.Effect) []statuseffect.Effect {
	out := make([]statuseffect.Effect, len(es))
	for i, e := range es {
		out[i] = e.Clone()
	}
	return out
}

// SetStatusEffects troca todos os efeitos de uma vez (nascimento da sessão). Fichas sem efeito
// não ficam no mapa.
func (s *MatchSession) SetStatusEffects(byChar map[uuid.UUID][]statuseffect.Effect) {
	s.statusEffects = make(map[uuid.UUID][]statuseffect.Effect, len(byChar))
	for id, es := range byChar {
		if len(es) > 0 {
			s.statusEffects[id] = cloneEffects(es)
		}
	}
}

// StatusEffectsOf devolve cópias dos efeitos da ficha, em ordem; nunca nil.
func (s *MatchSession) StatusEffectsOf(charID uuid.UUID) []statuseffect.Effect {
	return cloneEffects(s.statusEffects[charID])
}

// ReplaceStatusEffects troca o conjunto da ficha; vazio apaga a entrada.
func (s *MatchSession) ReplaceStatusEffects(charID uuid.UUID, effects []statuseffect.Effect) {
	if len(effects) == 0 {
		delete(s.statusEffects, charID)
		return
	}
	s.statusEffects[charID] = cloneEffects(effects)
}

// FindStatusEffect acha um efeito pelo id em qualquer ficha.
func (s *MatchSession) FindStatusEffect(id uuid.UUID) (statuseffect.Effect, bool) {
	for _, es := range s.statusEffects {
		for _, e := range es {
			if e.ID == id {
				return e.Clone(), true
			}
		}
	}
	return statuseffect.Effect{}, false
}

// CharactersWithStatusEffects lista as fichas com ao menos um efeito, em ordem estável.
func (s *MatchSession) CharactersWithStatusEffects() []uuid.UUID {
	out := make([]uuid.UUID, 0, len(s.statusEffects))
	for id := range s.statusEffects {
		out = append(out, id)
	}
	slices.SortFunc(out, func(a, b uuid.UUID) int { return bytes.Compare(a[:], b[:]) })
	return out
}

// StatusSummaryOf resume as restrições que os efeitos da ficha impõem.
func (s *MatchSession) StatusSummaryOf(charID uuid.UUID) statuseffect.Summary {
	return statuseffect.Summarize(s.statusEffects[charID])
}

// IsPlayerParticipant diz se o usuário é jogador da partida (dono de uma ficha de jogador).
// O mestre não conta por causa dos NPCs: a entrada de um NPC não vai para o mapa de jogadores.
func (s *MatchSession) IsPlayerParticipant(userID uuid.UUID) bool {
	_, ok := s.participants[userID]
	return ok
}

// ─── peças que bloqueiam (decisões 8 e 9 do plano de efeitos de status) ───────

// domainLabel é o verbo do domínio na mensagem de recusa ("não pode <verbo> porque está …").
var domainLabel = map[statuseffect.Domain]string{
	statuseffect.DomainAttack:   "atacar",
	statuseffect.DomainDefense:  "defender",
	statuseffect.DomainMovement: "se mover",
	statuseffect.DomainDodge:    "esquivar",
}

// reactionLabel é o que a peça de reação faz, como verbo, na mensagem de recusa.
var reactionLabel = map[statuseffect.Reaction]string{
	statuseffect.ReactionReflexDodge: "esquivar por reflexo",
	statuseffect.ReactionEvasion:     "usar a evasão",
	statuseffect.ReactionDefense:     "usar a defesa padrão",
	statuseffect.ReactionRepel:       "repelir",
	statuseffect.ReactionEscape:      "escapar",
}

// block é um motivo de recusa: o que fica proibido e quem proíbe.
type block struct {
	what  string
	kinds []statuseffect.Kind
}

// reactionNeed é uma peça de reação que um tipo de reação usa, com o domínio que ela exige
// livre (decisão 9: esquiva → reflexo e evasão; defesa → defesa padrão e repelir; escapar →
// movimento E esquiva, §5.4).
type reactionNeed struct {
	piece   statuseffect.Reaction
	domains []statuseffect.Domain
}

var (
	needReflex  = reactionNeed{statuseffect.ReactionReflexDodge, []statuseffect.Domain{statuseffect.DomainDodge}}
	needEvasion = reactionNeed{statuseffect.ReactionEvasion, []statuseffect.Domain{statuseffect.DomainDodge}}
	needDefense = reactionNeed{statuseffect.ReactionDefense, []statuseffect.Domain{statuseffect.DomainDefense}}
	needRepel   = reactionNeed{statuseffect.ReactionRepel, []statuseffect.Domain{statuseffect.DomainDefense}}
	needEscape  = reactionNeed{statuseffect.ReactionEscape, []statuseffect.Domain{statuseffect.DomainMovement, statuseffect.DomainDodge}}
)

// reactionNeeds é a decisão 9: as peças de cada tipo de reação. nothing não usa nenhuma; default
// tem as duas passivas, tratadas à parte (só cai sem as duas).
func reactionNeeds(k action.ReactionKind) []reactionNeed {
	switch k {
	case action.ReactDodge:
		return []reactionNeed{needReflex}
	case action.ReactClosedDodge:
		return []reactionNeed{needReflex, needEvasion}
	case action.ReactEscape, action.ReactEscapeGuard:
		return []reactionNeed{needEscape}
	case action.ReactClosedEscape:
		return []reactionNeed{needEscape, needEvasion}
	case action.ReactRepel:
		return []reactionNeed{needRepel}
	}
	return nil
}

// needBlock diz se a peça está proibida ou se um domínio que ela exige está travado.
func needBlock(sum statuseffect.Summary, n reactionNeed) *block {
	if kinds := sum.ForbiddenReactions[n.piece]; len(kinds) > 0 {
		return &block{reactionLabel[n.piece], kinds}
	}
	for _, d := range n.domains {
		if kinds := sum.LockedDomains[d]; len(kinds) > 0 {
			return &block{domainLabel[d], kinds}
		}
	}
	return nil
}

// passivesGone diz quais passivas da reação padrão os status tiraram.
func passivesGone(sum statuseffect.Summary) (noReflex, noDefense bool) {
	return needBlock(sum, needReflex) != nil, needBlock(sum, needDefense) != nil
}

// ActionDomains são os domínios que os componentes de uma action usam (decisão 9): ataque e
// finta → ataque; defesa → defesa; esquiva → esquiva; movimento → movimento. Interagir e só
// perícias não usam domínio.
func ActionDomains(a *action.Action) []statuseffect.Domain {
	var out []statuseffect.Domain
	if a.Attack != nil || a.Feint != nil {
		out = append(out, statuseffect.DomainAttack)
	}
	if a.Defense != nil {
		out = append(out, statuseffect.DomainDefense)
	}
	if a.Dodge != nil {
		out = append(out, statuseffect.DomainDodge)
	}
	if a.Move != nil {
		out = append(out, statuseffect.DomainMovement)
	}
	return out
}

func isReaction(a *action.Action) bool {
	return a.ReactToID != uuid.Nil || a.ReactionKind != ""
}

// statusBlock é o primeiro motivo pelo qual os status do ator recusam a; nil = permitido.
func statusBlock(sum statuseffect.Summary, a *action.Action) *block {
	if !isReaction(a) {
		if sum.CannotAct {
			return &block{"agir", sum.CannotActBy}
		}
		for _, d := range ActionDomains(a) {
			if kinds := sum.LockedDomains[d]; len(kinds) > 0 {
				return &block{domainLabel[d], kinds}
			}
		}
		if a.Move != nil && sum.NoDisplacement {
			return &block{"se deslocar", sum.NoDisplacementBy}
		}
		return nil
	}
	k := a.ReactionKind
	if k == action.ReactNothing {
		return nil
	}
	if sum.CannotReact {
		return &block{"reagir", sum.CannotReactBy}
	}
	if k == action.ReactDefault {
		// A reação padrão só cai sem as duas passivas; com uma, ela vale pela que sobrou.
		r, d := needBlock(sum, needReflex), needBlock(sum, needDefense)
		if r == nil || d == nil {
			return nil
		}
		return &block{"reagir com a reação padrão", mergeKinds(r.kinds, d.kinds)}
	}
	for _, n := range reactionNeeds(k) {
		if b := needBlock(sum, n); b != nil {
			return b
		}
	}
	if k.Displaces() && sum.NoDisplacement {
		return &block{"se deslocar", sum.NoDisplacementBy}
	}
	return nil
}

func mergeKinds(a, b []statuseffect.Kind) []statuseffect.Kind {
	out := append([]statuseffect.Kind(nil), a...)
	for _, k := range b {
		if !slices.Contains(out, k) {
			out = append(out, k)
		}
	}
	slices.Sort(out)
	return out
}

func (s *MatchSession) characterName(charID uuid.UUID) string {
	if cs, ok := s.charSheets[charID]; ok && cs != nil {
		if n := strings.TrimSpace(cs.GetProfile().NickName); n != "" {
			return n
		}
	}
	return "O personagem"
}

// StatusBlockError é a causa estruturada de uma recusa por status: quem, o que fica proibido
// e por quais tipos. Error() nomeia todos os tipos — é o texto do mestre; a Room monta o texto
// de cada destinatário com Message, nomeando só os tipos que ele pode ver (§9: um jogador não
// descobre por uma recusa um status que não podia ver). errors.Is(err, ErrBlockedByStatus) vale.
type StatusBlockError struct {
	CharacterID uuid.UUID
	Character   string // apelido da ficha
	What        string // o verbo da recusa: "agir", "atacar", ...
	Kinds       []statuseffect.Kind
}

func (e *StatusBlockError) Error() string { return e.Message(e.Kinds) }

func (e *StatusBlockError) Unwrap() error { return ErrBlockedByStatus }

// Message é o texto da recusa nomeando só os tipos dados, na forma "Fulano não pode atacar
// porque está paralisado e caído." (Kind.StatePhrase); sem nenhum, "por causa de um efeito".
func (e *StatusBlockError) Message(visible []statuseffect.Kind) string {
	return e.Sentence(visible) + "."
}

// Sentence é a mesma recusa como oração, sem o ponto final, para ser encaixada noutra frase
// (o motivo de um restore_refused).
func (e *StatusBlockError) Sentence(visible []statuseffect.Kind) string {
	if len(visible) == 0 {
		return fmt.Sprintf("%s não pode %s por causa de um efeito", e.Character, e.What)
	}
	states := make([]string, len(visible))
	for i, k := range visible {
		states[i] = k.StatePhrase()
	}
	return fmt.Sprintf("%s não pode %s porque está %s", e.Character, e.What, joinPT(states))
}

// joinPT junta com vírgula e "e" no último: "a", "a e b", "a, b e c".
func joinPT(items []string) string {
	if len(items) <= 1 {
		return strings.Join(items, "")
	}
	return strings.Join(items[:len(items)-1], ", ") + " e " + items[len(items)-1]
}

// StatusRefusal diz se os status do ator recusam a action (ou a reação): nil = permitida; senão
// um *StatusBlockError (ErrBlockedByStatus) com o que está proibido e por quais tipos. Lê só as
// peças que bloqueiam (statuseffect.Summarize); rolagens e penalidades ficam para a fase das
// rolagens.
func (s *MatchSession) StatusRefusal(a *action.Action) error {
	b := statusBlock(s.StatusSummaryOf(a.GetActorID()), a)
	if b == nil {
		return nil
	}
	return &StatusBlockError{
		CharacterID: a.GetActorID(), Character: s.characterName(a.GetActorID()),
		What: b.what, Kinds: append([]statuseffect.Kind(nil), b.kinds...),
	}
}

// applyStatusPassives marca na reação as passivas que os status do reator tiraram: o reflexo de
// uma reação padrão, a defesa padrão de todo tipo que a mantém (Paralisado em defesa: §5.4,
// "Defesa — conforme (defesa)"). Recalculado do zero: um status removido devolve a passiva.
func (s *MatchSession) applyStatusPassives(r *action.Action) {
	noReflex, noDefense := passivesGone(s.StatusSummaryOf(r.GetActorID()))
	r.WithoutReflexDodge = r.ReactionKind.IsPassive() && noReflex
	r.WithoutDefaultDefense = r.ReactionKind.KeepsDefault() && noDefense
}

// seatedReactionFor é a reação que a sessão senta para um alvo, conforme os status dele: a
// padrão inteira, a padrão sem uma das passivas, ou um nothing quando ele não pode reagir ou
// perdeu as duas.
func (s *MatchSession) seatedReactionFor(actorID, reactToID uuid.UUID) *action.Action {
	sum := s.StatusSummaryOf(actorID)
	noReflex, noDefense := passivesGone(sum)
	if sum.CannotReact || (noReflex && noDefense) {
		r := action.NewAction(actorID, nil, reactToID, nil, action.ActionSpeed{},
			nil, nil, nil, nil, nil, nil, nil)
		r.ReactionKind = action.ReactNothing
		return r
	}
	r := newDefaultReaction(actorID, reactToID)
	r.WithoutReflexDodge, r.WithoutDefaultDefense = noReflex, noDefense
	return r
}

// DropRefusedByStatus aplica o conjunto de status atual do personagem ao que ele tem pendente
// (decisão 8): tira da fila toda action que o conjunto recusa — com as empilhadas nela, como o
// cancel_action — e devolve essas, em ordem, para quem chama gravá-las como canceladas pelo
// mestre. No turno aberto, cada reação dele ainda não aberta é refeita: uma que o conjunto
// recusa sai e dá lugar à sentada (padrão ou nothing); a sentada é trocada pela que o conjunto
// pede; a enviada que continua permitida só tem as passivas recalculadas. O que já foi aberto
// fica. Uma reação trocada não é uma action cancelada: não vai para a gravação de canceladas.
//
// Nada é devolvido nem marcado como troca: a barra que uma reação recusada cobrou fica cobrada,
// e a decisão é do mestre, não do jogador (sem swapMarks). Só o desfazer no turno, com o pedido
// do mestre, devolve alguma coisa (RestoreCancelledByStatus, RestoreSwappedReactions).
//
// Cada StatusDrop leva, do instante da saída, o que o desfazer precisa para devolver a ação
// como estava (RestoreCancelledByStatus): o lugar dela na fila, as edições do mestre que a saída
// descarta e se o movimento dela se apoiava numa base ainda na fila. As reações trocadas saem
// em DropRefusedByStatusWithReactions.
func (s *MatchSession) DropRefusedByStatus(charID uuid.UUID) []StatusDrop {
	drops, _ := s.DropRefusedByStatusWithReactions(charID)
	return drops
}

// DropRefusedByStatusWithReactions é DropRefusedByStatus respondendo também as reações não
// abertas que a mudança tirou do turno aberto (ReactionDrop: a enviada que o conjunto recusa e
// a sentada trocada por outra de tipo diferente), com o retrato do instante da saída — o que
// RestoreSwappedReactions devolve.
func (s *MatchSession) DropRefusedByStatusWithReactions(charID uuid.UUID) ([]StatusDrop, []ReactionDrop) {
	sum := s.StatusSummaryOf(charID)
	seen := map[uuid.UUID]bool{}
	var removed []*action.Action
	for _, a := range s.activeQueue.All() {
		if a.GetActorID() != charID || seen[a.GetID()] || statusBlock(sum, a) == nil {
			continue
		}
		for _, c := range append([]*action.Action{a}, s.stackedAfter(a)...) {
			if !seen[c.GetID()] {
				seen[c.GetID()] = true
				removed = append(removed, c)
			}
		}
	}
	// O retrato vem antes de qualquer saída: a base de um empilhado que sai junto ainda está na
	// fila, e as edições ainda estão na ação.
	drops := make([]StatusDrop, 0, len(removed))
	for _, a := range removed {
		drops = append(drops, StatusDrop{
			Action:     a,
			arrival:    s.arrivals[a.GetID()],
			baseQueued: a.Move != nil && a.Move.StackedOn != uuid.Nil && s.IsQueued(a.Move.StackedOn),
			edits:      s.snapshotEdits(a),
			derived:    snapshotDerived(a),
		})
	}
	for _, a := range removed {
		s.dropFromQueue(a)
	}
	return drops, s.reseatForStatus(charID)
}

// StatusDrop é uma ação que DropRefusedByStatus tirou da fila, com o retrato do instante da
// saída que só RestoreCancelledByStatus lê: a ordem de chegada (o lugar na fila, empates
// incluídos), se o movimento se apoiava numa base ainda na fila e as edições do mestre que a
// saída descartou, e os valores derivados que a saída recalcula (derived). O ponteiro é o que
// saiu da fila; depois da saída só o desfazer o segura.
type StatusDrop struct {
	Action     *action.Action
	arrival    uint64
	baseQueued bool
	edits      editSnapshot
	derived    derivedSnapshot
}

// derivedSnapshot são os campos que o discardEdits recalcula (pelo deriveSpeeds) numa ação que
// sai da fila, como estavam no instante ANTES da saída: a velocidade da ação (perícia, valor,
// dados e resultado), a do movimento, o FinalSpeed e o viés do sistema. O desfazer os escreve
// de volta em vez de derivar de novo: o ledger do personagem pode ter mudado desde o enqueue ou
// a última edição (um turno que fechou paga ou expira modificadores de actionSpeed), e uma
// velocidade derivada de novo poria a ação noutro lugar da ordem do Race.
type derivedSnapshot struct {
	speed      action.ActionSpeed
	moveSpeed  *action.RollCheck
	finalSpeed int
	systemBias int
}

func snapshotDerived(a *action.Action) derivedSnapshot {
	d := derivedSnapshot{speed: a.Speed, systemBias: a.SystemBias}
	if a.Move != nil {
		d.finalSpeed = a.Move.FinalSpeed
		if a.Move.Speed != nil {
			ms := *a.Move.Speed
			d.moveSpeed = &ms
		}
	}
	return d
}

// restoreDerived escreve de volta o que snapshotDerived guardou.
func restoreDerived(a *action.Action, d derivedSnapshot) {
	a.Speed, a.SystemBias = d.speed, d.systemBias
	if a.Move == nil {
		return
	}
	a.Move.FinalSpeed = d.finalSpeed
	if d.moveSpeed == nil {
		a.Move.Speed = nil
		return
	}
	ms := *d.moveSpeed
	a.Move.Speed = &ms
}

// DroppedActions são as ações de uma lista de StatusDrop, na mesma ordem.
func DroppedActions(ds []StatusDrop) []*action.Action {
	out := make([]*action.Action, 0, len(ds))
	for _, d := range ds {
		out = append(out, d.Action)
	}
	return out
}

// editSnapshot são as edições do mestre numa ação enfileirada no instante em que um status a
// tirou da fila: as capturas (overrides, com os originais) e os dados de perícias retiradas,
// mais o valor EDITADO de cada campo capturado — o discardEdits põe o original de volta, e o
// desfazer reescreve o editado (reapplyEdits). Vazio quando não havia edição.
type editSnapshot struct {
	overrides   map[string]match.OverriddenValue
	removedDice map[string]action.RollAttempts
	edited      []match.OverriddenValue
}

func (s *MatchSession) snapshotEdits(a *action.Action) editSnapshot {
	id := a.GetID()
	ovs, dice := s.overrides[id], s.removedSkillDice[id]
	if len(ovs) == 0 && len(dice) == 0 {
		return editSnapshot{}
	}
	snap := editSnapshot{overrides: maps.Clone(ovs), removedDice: maps.Clone(dice)}
	for field := range ovs {
		snap.edited = append(snap.edited, match.OverriddenValue{Field: field, Original: currentValue(a, field)})
	}
	return snap
}

// reapplyEdits desfaz o discardEdits de uma ação que volta: o valor editado de cada campo, as
// capturas e os dados retirados voltam. As velocidades não são derivadas aqui: restoreDerived
// as escreve como estavam antes da saída.
func (s *MatchSession) reapplyEdits(a *action.Action, snap editSnapshot) {
	if len(snap.overrides) == 0 && len(snap.removedDice) == 0 {
		return
	}
	id := a.GetID()
	for _, ov := range snap.edited {
		restoreOriginal(a, ov)
	}
	if len(snap.overrides) > 0 {
		if s.overrides == nil {
			s.overrides = map[uuid.UUID]map[string]match.OverriddenValue{}
		}
		s.overrides[id] = maps.Clone(snap.overrides)
	}
	if len(snap.removedDice) > 0 {
		if s.removedSkillDice == nil {
			s.removedSkillDice = map[uuid.UUID]map[string]action.RollAttempts{}
		}
		s.removedSkillDice[id] = maps.Clone(snap.removedDice)
	}
}

// currentValue é o valor de agora de um campo que captureOverride captura, na forma que
// restoreOriginal escreve de volta (as chaves de lá: "<campo>.condition",
// "skill:<nome>.condition", "damageSkill", "targetIds", "skills"). nil = sem valor (uma
// condição ausente).
func currentValue(a *action.Action, field string) any {
	switch field {
	case "damageSkill":
		if a.Attack == nil {
			return nil
		}
		return a.Attack.DamageSkill
	case "targetIds":
		return append([]uuid.UUID(nil), a.TargetID...)
	case "skills":
		return append([]action.Skill(nil), a.Skills...)
	}
	key, ok := strings.CutSuffix(field, ".condition")
	if !ok {
		return nil
	}
	edit := action.ConditionEdit{Field: action.ConditionField(key)}
	if name, isSkill := strings.CutPrefix(key, "skill:"); isSkill {
		edit = action.ConditionEdit{SkillName: name}
	}
	rc, err := resolveRollCheck(a, edit)
	if err != nil || rc.Context.Condition == nil {
		return nil
	}
	return *rc.Context.Condition
}

// reseatForStatus refaz as reações não abertas do personagem no turno aberto (ver
// DropRefusedByStatus) e devolve as que tirou do turno, com o retrato de antes da saída.
func (s *MatchSession) reseatForStatus(charID uuid.UUID) []ReactionDrop {
	t := s.openTurn()
	if t == nil {
		return nil
	}
	sum := s.StatusSummaryOf(charID)
	var swapped []ReactionDrop
	for _, r := range t.GetReactions() {
		if r.GetActorID() != charID || t.IsOpened(r.GetID()) {
			continue
		}
		live := t.ReactionRef(r.GetID())
		var seat *action.Action
		if t.IsSeated(r.GetID()) {
			seat = s.seatedReactionFor(charID, r.ReactToID)
			if seat.ReactionKind == live.ReactionKind {
				// Mesmo tipo: guarda os dados e as edições do mestre, só as passivas mudam.
				live.WithoutReflexDodge, live.WithoutDefaultDefense = seat.WithoutReflexDodge, seat.WithoutDefaultDefense
				continue
			}
		} else {
			if statusBlock(sum, live) == nil {
				s.applyStatusPassives(live)
				continue
			}
			seat = s.seatedReactionFor(charID, r.ReactToID)
		}
		// O retrato vem antes da saída: o lugar, a sentada, o pouso e as edições do mestre.
		swapped = append(swapped, s.snapshotReaction(t, live))
		t.DropUnopened(r.GetID())
		s.dropEdits(r.GetID())
		t.Seat(seat)
		s.noteAnchorMove(t.GetID(), r.GetID(), seat.GetID())
	}
	return swapped
}

// ReactionDrop é uma reação não aberta que um status tirou do turno aberto (reseatForStatus),
// com o retrato do instante da saída que só RestoreSwappedReactions lê: o turno, a reação
// inteira (uma cópia: o turno guarda reações por valor, e a que saiu não está mais em lugar
// nenhum — os dados, as velocidades e o que ela consumiu vêm nela), o lugar entre as reações do
// turno, se era a sentada, a escolha de pouso do mestre e as edições dele (as capturas, que a
// saída descarta).
type ReactionDrop struct {
	TurnID   uuid.UUID
	Reaction action.Action
	index    int
	seated   bool
	landing  *[3]int
	edits    editSnapshot
}

func (s *MatchSession) snapshotReaction(t *turn.Turn, live *action.Action) ReactionDrop {
	d := ReactionDrop{
		TurnID: t.GetID(), Reaction: *live, index: t.ReactionIndex(live.GetID()),
		seated: t.IsSeated(live.GetID()), edits: s.snapshotEdits(live),
	}
	if pos, ok := t.EscapeLandings()[live.GetID()]; ok {
		d.landing = &pos
	}
	return d
}

// RestoredReaction é uma reação que RestoreSwappedReactions pôs de volta no turno: uma cópia de
// como ela ficou, se ela é a sentada da sessão (Seated: não é resposta de ninguém) e a reação que
// estava no lugar dela e saiu (Replaced; a sentada que o status ou a remoção dele puseram).
type RestoredReaction struct {
	TurnID   uuid.UUID
	Reaction action.Action
	Seated   bool
	Replaced uuid.UUID
}

// ReactionRefusal é uma reação que RestoreSwappedReactions não devolveu: o motivo (em
// português, mostrado ao mestre), os tipos de status que ainda a recusam (BlockedBy; nil por
// outro motivo) e o próprio ReactionDrop, para quem chama poder guardá-lo de novo.
type ReactionRefusal struct {
	ReactionID uuid.UUID
	Reason     string
	BlockedBy  []statuseffect.Kind
	Drop       ReactionDrop
}

// RestoreSwappedReactions devolve ao turno aberto as reações que um status tirou dele
// (DropRefusedByStatusWithReactions) — o par de RestoreCancelledByStatus para as reações, no
// mesmo desfazer. Revalida, nunca repete: cada uma é conferida contra o turno de AGORA, em
// ordem —
//
//   - o turno dela ainda é o aberto;
//   - o reator ainda está na partida e ainda é alvo da ação do turno;
//   - nenhuma reação com esse id está no turno (idempotente: uma segunda chamada só recusa);
//   - os status de agora a permitem (outro status pode continuar recusando: BlockedBy);
//   - o reator não mandou outra reação depois (a que ele mandou fica: é a última palavra dele),
//     e a sentada que ficou no lugar não foi aberta pelo mestre (o que foi narrado fica).
//
// A aceita volta como estava — o mesmo id, os dados, as velocidades, o que ela consumiu, o lugar
// entre as reações, sentada ou não, o pouso e as edições do mestre — no lugar da sentada que
// está lá (que sai, com as edições dela, e cujas âncoras passam para a devolvida: noteAnchorMove);
// só as passivas são recalculadas para o conjunto de agora. Nada é cobrado de novo: a barra que
// ela cobrou na chegada (RecordSecretSpeed, pelo id dela) ficou cobrada e volta a corresponder a
// uma reação do turno. A recusada volta com o motivo e continua fora do turno.
func (s *MatchSession) RestoreSwappedReactions(drops []ReactionDrop) (restored []RestoredReaction, refused []ReactionRefusal) {
	for _, d := range drops {
		reason, kinds := s.reactionRestoreRefusal(d)
		if reason != "" {
			refused = append(refused, ReactionRefusal{ReactionID: d.Reaction.GetID(), Reason: reason, BlockedBy: kinds, Drop: d})
			continue
		}
		t := s.openTurn()
		var replaced uuid.UUID
		if cur := currentReactionOf(t, d.Reaction.GetActorID()); cur != nil {
			replaced = cur.GetID()
			t.DropUnopened(replaced)
			s.dropEdits(replaced)
		}
		t.RestoreUnopened(d.Reaction, d.index, d.seated, d.landing)
		live := t.ReactionRef(d.Reaction.GetID())
		s.reapplyEdits(live, d.edits)
		s.applyStatusPassives(live)
		if replaced != uuid.Nil {
			s.noteAnchorMove(t.GetID(), replaced, live.GetID())
		}
		restored = append(restored, RestoredReaction{TurnID: t.GetID(), Reaction: *live, Seated: d.seated, Replaced: replaced})
	}
	return restored, refused
}

// currentReactionOf é a reação de actorID no turno agora; nil sem nenhuma.
func currentReactionOf(t *turn.Turn, actorID uuid.UUID) *action.Action {
	for _, r := range t.GetReactions() {
		if r.GetActorID() == actorID {
			return t.ReactionRef(r.GetID())
		}
	}
	return nil
}

// reactionRestoreRefusal é o motivo pelo qual d não pode voltar ao turno agora ("" = pode) e,
// quando é um status, os tipos que a recusam.
func (s *MatchSession) reactionRestoreRefusal(d ReactionDrop) (string, []statuseffect.Kind) {
	r := d.Reaction
	actor := r.GetActorID()
	t := s.openTurn()
	if t == nil || t.GetID() != d.TurnID {
		return "o turno da reação não é mais o turno aberto", nil
	}
	if _, ok := s.charToPlayer[actor.String()]; !ok {
		return "o personagem não está mais na partida", nil
	}
	name := s.characterName(actor)
	if t.ReactionRef(r.GetID()) != nil {
		return fmt.Sprintf("a reação de %s já está no turno", name), nil
	}
	if act := t.GetAction(); act.GetID() != r.ReactToID || !slices.Contains(act.TargetID, actor) {
		return fmt.Sprintf("%s não é mais alvo da ação do turno", name), nil
	}
	if err := s.StatusRefusal(&r); err != nil {
		var be *StatusBlockError
		if errors.As(err, &be) {
			return be.Sentence(be.Kinds), be.Kinds
		}
		return refusalClause(err), nil
	}
	if cur := currentReactionOf(t, actor); cur != nil {
		if !t.IsSeated(cur.GetID()) {
			return fmt.Sprintf("%s mandou outra reação depois", name), nil
		}
		if t.IsOpened(cur.GetID()) {
			return fmt.Sprintf("o mestre já abriu a reação que ficou no lugar da de %s", name), nil
		}
	}
	return "", nil
}

// RestoreRefusal é uma ação que RestoreCancelledByStatus não devolveu: o motivo (em português,
// mostrado ao mestre), os tipos de status que ainda a recusam (BlockedBy; nil por outro motivo)
// — ou que recusam a base dela, quando ela é um movimento empilhado numa base recusada nesta
// mesma chamada: ela segue a base — e o próprio StatusDrop, para quem chama poder guardá-lo de
// novo.
type RestoreRefusal struct {
	ActionID  uuid.UUID
	Reason    string
	BlockedBy []statuseffect.Kind
	Drop      StatusDrop
}

// RestoreCancelledByStatus devolve à fila ações que DropRefusedByStatus tirou (D3: o mestre
// desfez, no mesmo turno aberto, o status que as cancelou). Revalida, nunca repete: cada ação é
// conferida contra a sessão de AGORA, em ordem —
//
//   - o ator ainda está na partida;
//   - nenhuma ação com esse id está na fila (idempotente: uma segunda chamada só recusa);
//   - os status de agora a permitem (outro status pode continuar recusando: BlockedBy);
//   - um movimento volta só se a cadeia dele está como estava na saída: o que se apoiava numa
//     base ainda na fila naquele instante volta só se essa base continua na fila (ou voltou
//     nesta mesma chamada) e é o fim da cadeia; o que era cabeça — de fato, ou porque a base
//     já tinha saído da fila (aberta, consumida) — volta só se o personagem não tem nenhum
//     movimento pendente. Nunca meia cadeia, nunca um garfo.
//
// A aceita volta ao MESMO lugar da fila (a ordem de chegada dela, empates incluídos), com o id,
// os dados, as edições do mestre que a saída tinha descartado (reapplyEdits) e os valores
// derivados — velocidades, viés do sistema — exatamente como estavam antes da saída
// (restoreDerived, sem derivar de novo); depois, as reações não abertas de cada ator são refeitas
// para o conjunto de agora. A recusada volta com o motivo.
//
// O inverso de aplicar um status no turno aberto é o desfazer com restoreCancelledActions
// (undoInTurnAppliesLocked + giveBackLocked, que chama este método para as ações da fila e
// RestoreSwappedReactions para as reações que o status trocou no turno). Um futuro ctrl+Z deve
// chamar este caminho, nunca escrever a sua própria reversão: a reversão revalida contra o estado
// de agora, sob r.mu, e recusa com motivo o que não cabe mais. Não volta: a janela de fusão do
// relógio que o cancelamento fechou (inofensivo) e o que a revalidação recusa.
func (s *MatchSession) RestoreCancelledByStatus(drops []StatusDrop) (restored []*action.Action, refused []RestoreRefusal) {
	var actors []uuid.UUID
	// blockedHere: as recusadas nesta chamada por um status, com os tipos — um empilhado nelas
	// segue a base.
	blockedHere := map[uuid.UUID][]statuseffect.Kind{}
	for _, d := range drops {
		a := d.Action
		if a == nil {
			continue
		}
		if kinds, ok := blockedHere[stackedOn(a)]; ok && d.baseQueued {
			reason := fmt.Sprintf("o movimento de %s em que ela se apoia também não pôde voltar", s.characterName(a.GetActorID()))
			refused = append(refused, RestoreRefusal{ActionID: a.GetID(), Reason: reason, BlockedBy: kinds, Drop: d})
			blockedHere[a.GetID()] = kinds
			continue
		}
		if reason, kinds := s.restoreRefusal(d); reason != "" {
			refused = append(refused, RestoreRefusal{ActionID: a.GetID(), Reason: reason, BlockedBy: kinds, Drop: d})
			if len(kinds) > 0 {
				blockedHere[a.GetID()] = kinds
			}
			continue
		}
		s.reapplyEdits(a, d.edits)
		restoreDerived(a, d.derived)
		s.insertByArrival(a, d.arrival)
		restored = append(restored, a)
		if !slices.Contains(actors, a.GetActorID()) {
			actors = append(actors, a.GetActorID())
		}
	}
	for _, id := range actors {
		// O conjunto de agora é o mesmo da remoção que veio antes (DropRefusedByStatus): nada
		// novo sai do turno aqui, e o que sairia não é deste desfazer.
		_ = s.reseatForStatus(id)
	}
	return restored, refused
}

// stackedOn é a base do movimento empilhado de a; uuid.Nil sem movimento ou sem base.
func stackedOn(a *action.Action) uuid.UUID {
	if a.Move == nil {
		return uuid.Nil
	}
	return a.Move.StackedOn
}

// restoreRefusal é o motivo pelo qual d não pode voltar à fila agora ("" = pode) e, quando é um
// status, os tipos que a recusam.
func (s *MatchSession) restoreRefusal(d StatusDrop) (string, []statuseffect.Kind) {
	a := d.Action
	actor := a.GetActorID()
	if _, ok := s.charToPlayer[actor.String()]; !ok {
		return "o personagem não está mais na partida", nil
	}
	name := s.characterName(actor)
	if s.IsQueued(a.GetID()) {
		return fmt.Sprintf("a ação de %s já está na fila", name), nil
	}
	if err := s.StatusRefusal(a); err != nil {
		var be *StatusBlockError
		if errors.As(err, &be) {
			return be.Sentence(be.Kinds), be.Kinds
		}
		return refusalClause(err), nil
	}
	if a.Move == nil {
		return "", nil
	}
	tail := s.MoveChainTail(actor)
	if !d.baseQueued {
		if tail != nil {
			return fmt.Sprintf("a cadeia de movimentos de %s mudou desde o cancelamento", name), nil
		}
		return "", nil
	}
	base := a.Move.StackedOn
	if !s.IsQueued(base) {
		return fmt.Sprintf("o movimento de %s em que ela se apoia não está mais na fila", name), nil
	}
	if tail == nil || tail.GetID() != base {
		return fmt.Sprintf("a cadeia de movimentos de %s mudou desde o cancelamento", name), nil
	}
	return "", nil
}

// TakesPartInOpenTurn diz se o personagem está no turno aberto — o ator da ação, um alvo dela
// ou o autor de uma reação. Uma mudança de status dele pode ter refeito uma reação ou mudado o
// viés de um teste que a resolução lê, e por isso a resolução é refeita sob a trava de escrita
// (onde os dados que faltam caem) e volta ao mestre. Antes perguntava só pela reação: um status
// no ator deixava a primeira leitura para quem só lê.
func (s *MatchSession) TakesPartInOpenTurn(charID uuid.UUID) bool {
	t := s.openTurn()
	if t == nil {
		return false
	}
	a := t.ActionRef()
	if a.GetActorID() == charID || slices.Contains(a.TargetID, charID) {
		return true
	}
	for _, r := range t.GetReactions() {
		if r.GetActorID() == charID {
			return true
		}
	}
	return false
}

// IsOpenTurnAction diz se id é a action do turno aberto ou uma das reações dele (anexada,
// aberta ou sentada) — o que um evento de status pode nomear como âncora. false sem turno
// aberto.
func (s *MatchSession) IsOpenTurnAction(id uuid.UUID) bool {
	t := s.openTurn()
	if t == nil || id == uuid.Nil {
		return false
	}
	if t.ActionRef().GetID() == id {
		return true
	}
	for _, r := range t.GetReactions() {
		if r.GetID() == id {
			return true
		}
	}
	return false
}

// openTurn é o turno aberto agora; nil sem round, sem turno ou com o último já fechado.
func (s *MatchSession) openTurn() *turn.Turn {
	if s.activeRound == nil {
		return nil
	}
	t := s.activeRound.CurrentTurn()
	if t == nil || t.GetFinishedAt() != nil {
		return nil
	}
	return t
}

// ResolveOpenTurn é a resolução do turno aberto; nil quando não há turno aberto.
func (s *MatchSession) ResolveOpenTurn() *service.TurnResolution {
	t := s.openTurn()
	if t == nil {
		return nil
	}
	return s.ResolveTurn(t)
}

// refusalClause é o motivo de uma recusa que não é de status, encaixado depois de "A ação não
// voltou: ". Frase de mesa (errtext) entra sem o ponto final; qualquer outro erro vira uma
// cláusula fixa e o original vai para o log.
func refusalClause(err error) string {
	if text, ok := errtext.Of(err); ok {
		return strings.TrimSuffix(text, ".")
	}
	log.Printf("restore refusal with no sentence: %v", err)
	return "algo impediu a volta"
}
