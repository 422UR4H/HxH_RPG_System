package match

import (
	"fmt"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

// IStatusEffects é o que a sala chama quando o mestre mexe nos efeitos de status. Sem I/O:
// muda a sessão (sob o r.mu de quem chama) e devolve o que a sala deve gravar e difundir.
type IStatusEffects interface {
	Apply(s *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID, in ApplyStatusEffectInput, now time.Time) (StatusEffectChange, error)
	Edit(s *matchsession.MatchSession, masterUUID, callerUUID, id uuid.UUID, p statuseffect.Patch, now time.Time) (StatusEffectChange, error)
	Remove(s *matchsession.MatchSession, masterUUID, callerUUID, id uuid.UUID) (StatusEffectChange, error)
}

type ApplyStatusEffectInput struct {
	CharacterIDs []uuid.UUID
	Draft        statuseffect.Draft
}

// StatusEffectChange é o resultado de uma mudança já aplicada na sessão.
type StatusEffectChange struct {
	// Characters são as fichas cujo conjunto mudou.
	Characters []uuid.UUID
	// Writes são as gravações a fazer, em ordem.
	Writes []statuseffect.Write
	// HiddenChanged: um efeito escondido entrou, saiu ou teve a audiência trocada (quem o vê
	// mudou, então a peça e as barras precisam ser reavaliadas).
	HiddenChanged bool
}

type StatusEffectsUC struct {
	newID func() uuid.UUID
}

func NewStatusEffectsUC() *StatusEffectsUC {
	return &StatusEffectsUC{newID: uuid.New}
}

func invalidParams(msg string) error {
	return fmt.Errorf("%w: %s", statuseffect.ErrInvalidParams, msg)
}

// guard cobre o que vale para as três operações: só o mestre, e só com a partida aberta.
func (uc *StatusEffectsUC) guard(s *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID) error {
	if callerUUID != masterUUID {
		return ErrNotMatchMaster
	}
	if s.IsEnded() {
		return matchsession.ErrMatchEnded
	}
	return nil
}

// checkAudience exige que cada jogador listado seja jogador da partida e não o mestre (o mestre
// aparece nos participantes por causa dos NPCs, mas já vê tudo).
func checkAudience(s *matchsession.MatchSession, masterUUID uuid.UUID, a *statuseffect.Audience) error {
	if a == nil {
		return nil
	}
	if err := a.Validate(); err != nil {
		return err
	}
	for _, p := range a.Players {
		if p == masterUUID || !s.IsPlayerParticipant(p) {
			return fmt.Errorf("%w: jogador fora da partida", statuseffect.ErrInvalidAudience)
		}
	}
	return nil
}

// checkController exige que quem controla seja um personagem da sessão e outro que o alvo.
func checkController(s *matchsession.MatchSession, target uuid.UUID, controller *uuid.UUID) error {
	if controller == nil {
		return nil
	}
	if *controller == target {
		return invalidParams("controllerId igual ao alvo")
	}
	if _, err := s.GetCharSheet(*controller); err != nil {
		return invalidParams("controllerId fora da partida")
	}
	return nil
}

func (uc *StatusEffectsUC) Apply(
	s *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID, in ApplyStatusEffectInput, now time.Time,
) (StatusEffectChange, error) {
	if err := uc.guard(s, masterUUID, callerUUID); err != nil {
		return StatusEffectChange{}, err
	}
	if len(in.CharacterIDs) == 0 {
		return StatusEffectChange{}, invalidParams("characterIds vazio")
	}
	seen := make(map[uuid.UUID]struct{}, len(in.CharacterIDs))
	for _, id := range in.CharacterIDs {
		if _, dup := seen[id]; dup {
			return StatusEffectChange{}, invalidParams("characterIds repetido")
		}
		seen[id] = struct{}{}
		if _, err := s.GetCharSheet(id); err != nil {
			return StatusEffectChange{}, err
		}
	}
	if err := checkAudience(s, masterUUID, in.Draft.Audience); err != nil {
		return StatusEffectChange{}, err
	}

	// Tudo ou nada: calcula o conjunto seguinte de cada ficha; só depois troca todos.
	nexts := make([][]statuseffect.Effect, len(in.CharacterIDs))
	var change StatusEffectChange
	for i, id := range in.CharacterIDs {
		if err := checkController(s, id, in.Draft.Params.ControllerID); err != nil {
			return StatusEffectChange{}, err
		}
		next, added, err := statuseffect.Apply(s.StatusEffectsOf(id), id, in.Draft, uc.newID, now)
		if err != nil {
			return StatusEffectChange{}, err
		}
		nexts[i] = next
		change.Characters = append(change.Characters, id)
		for _, e := range added {
			change.Writes = append(change.Writes, statuseffect.Write{Effect: e})
			if e.Kind == statuseffect.KindHidden {
				change.HiddenChanged = true
			}
		}
	}
	for i, id := range in.CharacterIDs {
		s.ReplaceStatusEffects(id, nexts[i])
	}
	return change, nil
}

func (uc *StatusEffectsUC) Edit(
	s *matchsession.MatchSession, masterUUID, callerUUID, id uuid.UUID, p statuseffect.Patch, now time.Time,
) (StatusEffectChange, error) {
	if err := uc.guard(s, masterUUID, callerUUID); err != nil {
		return StatusEffectChange{}, err
	}
	cur, ok := s.FindStatusEffect(id)
	if !ok {
		// Pede ao domínio o erro de não encontrado, para ter uma só redação.
		_, _, err := statuseffect.Edit(nil, id, p, now)
		return StatusEffectChange{}, err
	}
	if p.SetAudience {
		if err := checkAudience(s, masterUUID, p.Audience); err != nil {
			return StatusEffectChange{}, err
		}
	}
	if p.Params != nil {
		if err := checkController(s, cur.CharacterID, p.Params.ControllerID); err != nil {
			return StatusEffectChange{}, err
		}
	}
	next, edited, err := statuseffect.Edit(s.StatusEffectsOf(cur.CharacterID), id, p, now)
	if err != nil {
		return StatusEffectChange{}, err
	}
	s.ReplaceStatusEffects(cur.CharacterID, next)
	return StatusEffectChange{
		Characters:    []uuid.UUID{cur.CharacterID},
		Writes:        []statuseffect.Write{{Effect: edited}},
		HiddenChanged: cur.Kind == statuseffect.KindHidden && p.SetAudience,
	}, nil
}

func (uc *StatusEffectsUC) Remove(
	s *matchsession.MatchSession, masterUUID, callerUUID, id uuid.UUID,
) (StatusEffectChange, error) {
	if err := uc.guard(s, masterUUID, callerUUID); err != nil {
		return StatusEffectChange{}, err
	}
	cur, ok := s.FindStatusEffect(id)
	if !ok {
		_, _, err := statuseffect.Remove(nil, id)
		return StatusEffectChange{}, err
	}
	next, removed, err := statuseffect.Remove(s.StatusEffectsOf(cur.CharacterID), id)
	if err != nil {
		return StatusEffectChange{}, err
	}
	s.ReplaceStatusEffects(cur.CharacterID, next)
	return StatusEffectChange{
		Characters:    []uuid.UUID{cur.CharacterID},
		Writes:        []statuseffect.Write{{Delete: true, Effect: removed}},
		HiddenChanged: removed.Kind == statuseffect.KindHidden,
	}, nil
}

var _ IStatusEffects = (*StatusEffectsUC)(nil)
