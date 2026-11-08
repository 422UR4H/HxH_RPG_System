package match

import (
	"context"
	"errors"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/clock"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
)

// ErrClockPlayInRace: o tempo real só existe no regime livre (spec §3.4).
var ErrClockPlayInRace = domain.NewDomainError(errors.New("story clock cannot run in the race regime"))

// IMatchGetter é o pedaço do repositório de partidas que os comandos do relógio leem: a data
// da história, para o set_clock_start.
type IMatchGetter interface {
	GetMatch(ctx context.Context, uuid uuid.UUID) (*match.Match, error)
}

// IClockCommands são os comandos do mestre sobre o relógio da história (spec §3, §7). Todos,
// menos StoryDate, mexem na sessão e exigem o r.mu de quem chama; nenhum faz I/O — gravar e
// anunciar é da sala. real é o instante real da mudança.
type IClockCommands interface {
	// StoryDate lê a data da história da partida (I/O: fora do r.mu).
	StoryDate(ctx context.Context, matchUUID uuid.UUID) (time.Time, error)
	Start(session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID, at time.Time) error
	Adjust(session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID, delta time.Duration, confirmLong bool, real time.Time) (from, to time.Time, err error)
	Set(session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID, to time.Time, confirmLong bool, real time.Time) (from time.Time, err error)
	Play(session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID, real time.Time) error
	// Pause devolve nil, nil quando o relógio já estava parado.
	Pause(session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID, real time.Time) (*clock.Pause, error)
}

// ClockCommandsUC confere o mestre e delega ao relógio da sessão. As regras (limite, salto
// longo, delta zero) são do relógio; a única daqui é "sem tempo real no regime disputado".
type ClockCommandsUC struct {
	matches IMatchGetter
}

func NewClockCommandsUC(matches IMatchGetter) *ClockCommandsUC {
	return &ClockCommandsUC{matches: matches}
}

func (uc *ClockCommandsUC) StoryDate(ctx context.Context, matchUUID uuid.UUID) (time.Time, error) {
	mt, err := uc.matches.GetMatch(ctx, matchUUID)
	if err != nil {
		return time.Time{}, err
	}
	d := mt.StoryStartAt.UTC()
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC), nil
}

func (uc *ClockCommandsUC) Start(session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID, at time.Time) error {
	if callerUUID != masterUUID {
		return ErrNotMatchMaster
	}
	return session.Clock().Start(at)
}

func (uc *ClockCommandsUC) Adjust(
	session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID,
	delta time.Duration, confirmLong bool, real time.Time,
) (time.Time, time.Time, error) {
	if callerUUID != masterUUID {
		return time.Time{}, time.Time{}, ErrNotMatchMaster
	}
	return session.Clock().Adjust(delta, confirmLong, real)
}

func (uc *ClockCommandsUC) Set(
	session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID,
	to time.Time, confirmLong bool, real time.Time,
) (time.Time, error) {
	if callerUUID != masterUUID {
		return time.Time{}, ErrNotMatchMaster
	}
	return session.Clock().Set(to, confirmLong, real)
}

func (uc *ClockCommandsUC) Play(session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID, real time.Time) error {
	if callerUUID != masterUUID {
		return ErrNotMatchMaster
	}
	if !session.Clock().IsSet() {
		return clock.ErrNotSet
	}
	if session.GetActiveRound().GetMode() == enum.Race {
		return ErrClockPlayInRace
	}
	return session.Clock().Play(real)
}

func (uc *ClockCommandsUC) Pause(session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID, real time.Time) (*clock.Pause, error) {
	if callerUUID != masterUUID {
		return nil, ErrNotMatchMaster
	}
	if !session.Clock().IsSet() {
		return nil, clock.ErrNotSet
	}
	return session.Clock().Pause(clock.PauseManual, real), nil
}
