package matchsession

import (
	"bytes"
	"errors"
	"fmt"
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

// domainLabel é o nome do domínio na mensagem de recusa.
var domainLabel = map[statuseffect.Domain]string{
	statuseffect.DomainAttack:   "ataque",
	statuseffect.DomainDefense:  "defesa",
	statuseffect.DomainMovement: "movimento",
	statuseffect.DomainDodge:    "esquiva",
}

// reactionLabel é o nome da peça de reação na mensagem de recusa.
var reactionLabel = map[statuseffect.Reaction]string{
	statuseffect.ReactionReflexDodge: "a esquiva por reflexo",
	statuseffect.ReactionEvasion:     "a evasão",
	statuseffect.ReactionDefense:     "a defesa padrão",
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
		return &block{"usar " + reactionLabel[n.piece], kinds}
	}
	for _, d := range n.domains {
		if kinds := sum.LockedDomains[d]; len(kinds) > 0 {
			return &block{"usar o domínio " + domainLabel[d], kinds}
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
				return &block{"usar o domínio " + domainLabel[d], kinds}
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
	What        string // "agir", "usar o domínio ataque", ...
	Kinds       []statuseffect.Kind
}

func (e *StatusBlockError) Error() string { return e.Message(e.Kinds) }

func (e *StatusBlockError) Unwrap() error { return ErrBlockedByStatus }

// Message é o texto da recusa nomeando só os tipos dados; sem nenhum, "um efeito".
func (e *StatusBlockError) Message(visible []statuseffect.Kind) string {
	cause := "um efeito"
	if len(visible) > 0 {
		names := make([]string, len(visible))
		for i, k := range visible {
			names[i] = string(k)
		}
		cause = strings.Join(names, ", ")
	}
	return fmt.Sprintf("%s não pode %s: %s", e.Character, e.What, cause)
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
// fica. Uma reação trocada não é uma action cancelada e não é devolvida.
//
// Nada é devolvido nem marcado como troca: a barra que uma reação recusada cobrou fica cobrada,
// e a decisão é do mestre, não do jogador (sem swapMarks).
//
// Cada StatusDrop leva, do instante da saída, o que o desfazer precisa para devolver a ação
// como estava (RestoreCancelledByStatus): o lugar dela na fila, as edições do mestre que a saída
// descarta e se o movimento dela se apoiava numa base ainda na fila.
func (s *MatchSession) DropRefusedByStatus(charID uuid.UUID) []StatusDrop {
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
		})
	}
	for _, a := range removed {
		s.dropFromQueue(a)
	}
	s.reseatForStatus(charID)
	return drops
}

// StatusDrop é uma ação que DropRefusedByStatus tirou da fila, com o retrato do instante da
// saída que só RestoreCancelledByStatus lê: a ordem de chegada (o lugar na fila, empates
// incluídos), se o movimento se apoiava numa base ainda na fila e as edições do mestre que a
// saída descartou. O ponteiro é o que saiu da fila; depois da saída só o desfazer o segura.
type StatusDrop struct {
	Action     *action.Action
	arrival    uint64
	baseQueued bool
	edits      editSnapshot
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
// capturas e os dados retirados voltam, e as velocidades são derivadas de novo com o viés do
// sistema que a ação já tinha — o mesmo que a edição usou.
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
	s.deriveSpeeds(a, a.SystemBias)
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
// DropRefusedByStatus).
func (s *MatchSession) reseatForStatus(charID uuid.UUID) {
	t := s.openTurn()
	if t == nil {
		return
	}
	sum := s.StatusSummaryOf(charID)
	for _, r := range t.GetReactions() {
		if r.GetActorID() != charID || t.IsOpened(r.GetID()) {
			continue
		}
		live := t.ReactionRef(r.GetID())
		if t.IsSeated(r.GetID()) {
			want := s.seatedReactionFor(charID, r.ReactToID)
			if want.ReactionKind == live.ReactionKind {
				// Mesmo tipo: guarda os dados e as edições do mestre, só as passivas mudam.
				live.WithoutReflexDodge, live.WithoutDefaultDefense = want.WithoutReflexDodge, want.WithoutDefaultDefense
				continue
			}
			t.DropUnopened(r.GetID())
			s.dropEdits(r.GetID())
			t.Seat(want)
			s.noteAnchorMove(t.GetID(), r.GetID(), want.GetID())
			continue
		}
		if statusBlock(sum, live) == nil {
			s.applyStatusPassives(live)
			continue
		}
		t.DropUnopened(r.GetID())
		s.dropEdits(r.GetID())
		seat := s.seatedReactionFor(charID, r.ReactToID)
		t.Seat(seat)
		s.noteAnchorMove(t.GetID(), r.GetID(), seat.GetID())
	}
}

// RestoreRefusal é uma ação que RestoreCancelledByStatus não devolveu: o motivo (em português,
// mostrado ao mestre), os tipos de status que ainda a recusam quando é isso (BlockedBy; nil
// por outro motivo) e o próprio StatusDrop, para quem chama poder guardá-lo de novo.
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
// os dados e as edições do mestre que a saída tinha descartado (reapplyEdits); depois, as reações
// não abertas de cada ator são refeitas para o conjunto de agora. A recusada volta com o motivo.
//
// O inverso de aplicar um status no turno aberto é o desfazer com restoreCancelledActions
// (undoInTurnAppliesLocked + RestoreCancelledByStatus). Um futuro ctrl+Z deve chamar este
// caminho, nunca escrever a sua própria reversão: a reversão revalida contra o estado de agora,
// sob r.mu, e recusa com motivo o que não cabe mais.
func (s *MatchSession) RestoreCancelledByStatus(drops []StatusDrop) (restored []*action.Action, refused []RestoreRefusal) {
	var actors []uuid.UUID
	for _, d := range drops {
		a := d.Action
		if a == nil {
			continue
		}
		if reason, kinds := s.restoreRefusal(d); reason != "" {
			refused = append(refused, RestoreRefusal{ActionID: a.GetID(), Reason: reason, BlockedBy: kinds, Drop: d})
			continue
		}
		s.reapplyEdits(a, d.edits)
		s.insertByArrival(a, d.arrival)
		restored = append(restored, a)
		if !slices.Contains(actors, a.GetActorID()) {
			actors = append(actors, a.GetActorID())
		}
	}
	for _, id := range actors {
		s.reseatForStatus(id)
	}
	return restored, refused
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
			return err.Error(), be.Kinds
		}
		return err.Error(), nil
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

// HasReactionOnOpenTurn diz se o personagem tem uma reação no turno aberto — o que uma mudança
// de status pode ter refeito, e por isso a resolução volta ao mestre.
func (s *MatchSession) HasReactionOnOpenTurn(charID uuid.UUID) bool {
	t := s.openTurn()
	if t == nil {
		return false
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
