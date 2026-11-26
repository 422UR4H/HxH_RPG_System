package match

import (
	"fmt"
	"time"

	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

// IStatusEffects é o que a sala chama quando o mestre mexe nos efeitos de status. Sem I/O:
// muda a sessão (sob o r.mu de quem chama) e devolve o que a sala deve gravar e difundir.
type IStatusEffects interface {
	Apply(s *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID, in ApplyStatusEffectInput, now time.Time) (StatusEffectChange, error)
	Edit(s *matchsession.MatchSession, masterUUID, callerUUID, id uuid.UUID, p statuseffect.Patch, now time.Time) (StatusEffectChange, error)
	// hp é o HP com que o personagem volta ao tirar o Morto (nil = o mínimo da barra, só se
	// estiver abaixo dele); recusado em qualquer outro tipo.
	Remove(s *matchsession.MatchSession, masterUUID, callerUUID, id uuid.UUID, hp *int) (StatusEffectChange, error)
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
	// HP são as fichas cujo HP a mudança já moveu na sessão (a remoção do Morto). A sala as
	// grava com as gravações de status e avisa o mestre e o dono (character_hp_changed).
	HP []HPChange
}

// HPChange é o HP que uma mudança de status deixou numa ficha. Delta é a variação
// com o sinal natural (NewHP − HP de antes): positivo quando o HP subiu, como na volta do Morto.
type HPChange struct {
	CharacterID uuid.UUID
	Sheet       *csSheet.CharacterSheet
	NewHP       int
	Delta       int
}

type StatusEffectsUC struct {
	newID func() uuid.UUID
}

func NewStatusEffectsUC() *StatusEffectsUC {
	return &StatusEffectsUC{newID: uuid.New}
}

func invalidParams(msg string) error {
	return errtext.Wrap(statuseffect.ErrInvalidParams, msg)
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
			return errtext.Wrap(statuseffect.ErrInvalidAudience, "Um dos jogadores escolhidos não está na partida.")
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
		return invalidParams("O controlador não pode ser o próprio personagem.")
	}
	if _, err := s.GetCharSheet(*controller); err != nil {
		return invalidParams("O controlador precisa ser um personagem da partida.")
	}
	return nil
}

func (uc *StatusEffectsUC) Apply(
	s *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID, in ApplyStatusEffectInput, now time.Time,
) (StatusEffectChange, error) {
	if err := uc.guard(s, masterUUID, callerUUID); err != nil {
		return StatusEffectChange{}, err
	}
	// O Postgres guarda microssegundos: truncar aqui faz memória e disco concordarem e
	// mantém estável, depois de um reinício, a ordem por applied_at (ex.: Inconsciente/Caído).
	now = now.Truncate(time.Microsecond)
	// O Morto entra só pela regra, no fechamento do turno (decisão 9 do plano).
	if in.Draft.Kind == statuseffect.KindDead {
		return StatusEffectChange{}, invalidParams("O Morto entra só pela regra.")
	}
	if len(in.CharacterIDs) == 0 {
		return StatusEffectChange{}, invalidParams("Escolha ao menos um personagem.")
	}
	seen := make(map[uuid.UUID]struct{}, len(in.CharacterIDs))
	for _, id := range in.CharacterIDs {
		if _, dup := seen[id]; dup {
			return StatusEffectChange{}, invalidParams("Um personagem foi escolhido duas vezes.")
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
			change.Writes = append(change.Writes, statuseffect.Write{Effect: e, Event: &statuseffect.Event{Change: statuseffect.ChangeApplied}})
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
	now = now.Truncate(time.Microsecond) // mesmo motivo do Apply
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
		Writes:        []statuseffect.Write{{Effect: edited, Event: &statuseffect.Event{Change: statuseffect.ChangeEdited}}},
		HiddenChanged: cur.Kind == statuseffect.KindHidden && p.SetAudience,
	}, nil
}

func (uc *StatusEffectsUC) Remove(
	s *matchsession.MatchSession, masterUUID, callerUUID, id uuid.UUID, hp *int,
) (StatusEffectChange, error) {
	if err := uc.guard(s, masterUUID, callerUUID); err != nil {
		return StatusEffectChange{}, err
	}
	cur, ok := s.FindStatusEffect(id)
	if !ok {
		_, _, err := statuseffect.Remove(nil, id)
		return StatusEffectChange{}, err
	}
	if hp != nil && cur.Kind != statuseffect.KindDead {
		return StatusEffectChange{}, invalidParams("Só a remoção do Morto aceita HP.")
	}
	// Tudo validado antes de mexer na sessão: o HP com que o Morto volta.
	var revive *hpTarget
	if cur.Kind == statuseffect.KindDead {
		t, err := reviveHP(s, cur.CharacterID, hp)
		if err != nil {
			return StatusEffectChange{}, err
		}
		revive = t
	}
	next, removed, err := statuseffect.Remove(s.StatusEffectsOf(cur.CharacterID), id)
	if err != nil {
		return StatusEffectChange{}, err
	}
	s.ReplaceStatusEffects(cur.CharacterID, next)
	change := StatusEffectChange{
		Characters:    []uuid.UUID{cur.CharacterID},
		Writes:        []statuseffect.Write{{Delete: true, Effect: removed, Event: &statuseffect.Event{Change: statuseffect.ChangeRemoved}}},
		HiddenChanged: removed.Kind == statuseffect.KindHidden,
	}
	if revive != nil {
		old := revive.bar.GetCurrent()
		if err := revive.bar.SetCurrent(revive.hp); err != nil {
			// Já conferido contra o máximo em reviveHP: falhar aqui é um defeito.
			return StatusEffectChange{}, fmt.Errorf("set revive hp of %s: %w", cur.CharacterID, err)
		}
		change.HP = []HPChange{{
			CharacterID: cur.CharacterID, Sheet: revive.sheet, NewHP: revive.hp, Delta: revive.hp - old,
		}}
	}
	return change, nil
}

// hpTarget é o HP que a remoção do Morto vai pôr na barra de vida da ficha.
type hpTarget struct {
	sheet *csSheet.CharacterSheet
	bar   interface {
		GetCurrent() int
		SetCurrent(int) error
	}
	hp int
}

// reviveHP decide o HP com que o personagem volta do Morto (decisão do dono 4): o dado pelo
// mestre, dentro de [mínimo, máximo] da barra; sem ele, o mínimo — só se o HP estiver abaixo
// dele (nunca rebaixa). nil quando o HP fica como está.
func reviveHP(s *matchsession.MatchSession, charID uuid.UUID, hp *int) (*hpTarget, error) {
	sheet, err := s.GetCharSheet(charID)
	if err != nil {
		return nil, err
	}
	bar, ok := sheet.GetAllStatusBar()[enum.Health]
	if !ok || bar == nil {
		if hp != nil {
			return nil, invalidParams("Esta ficha não tem barra de vida.")
		}
		return nil, nil
	}
	lo, hi := bar.GetMin(), bar.GetMax()
	if hp != nil {
		if *hp < lo || *hp > hi {
			return nil, invalidParams(fmt.Sprintf("O HP precisa ficar entre %d e %d.", lo, hi))
		}
		// O mesmo HP de agora não é mudança: nada a gravar nem a avisar.
		if *hp == bar.GetCurrent() {
			return nil, nil
		}
		return &hpTarget{sheet: sheet, bar: bar, hp: *hp}, nil
	}
	if bar.GetCurrent() >= lo {
		return nil, nil
	}
	return &hpTarget{sheet: sheet, bar: bar, hp: lo}, nil
}

var _ IStatusEffects = (*StatusEffectsUC)(nil)
