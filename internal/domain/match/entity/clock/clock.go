// Package clock modela o relógio da história de uma partida: um tempo
// ingênuo (UTC usado como relógio de parede) que avança em tempo real
// enquanto roda e nunca desce abaixo de um limite.
package clock

import "time"

type LimitReason string

const (
	LimitStart     LimitReason = "start"
	LimitTurn      LimitReason = "turn"
	LimitScene     LimitReason = "scene"
	LimitRoundMode LimitReason = "roundMode"
	LimitRound     LimitReason = "round"
)

type PauseReason string

const (
	PauseManual     PauseReason = "manual"
	PauseRoundMode  PauseReason = "roundMode"
	PauseScene      PauseReason = "scene"
	PauseMasterLeft PauseReason = "masterLeft"
	PauseEnd        PauseReason = "end"
	PauseRestart    PauseReason = "restart"
)

// LongSkip é o maior avanço sem confirmação.
const LongSkip = 48 * time.Hour

// Snapshot é um checkpoint do relógio: StoryAt vale no instante real
// RunningSince (quando rodando).
type Snapshot struct {
	StoryAt      *time.Time
	Running      bool
	RunningSince *time.Time
	LimitAt      *time.Time
	LimitReason  LimitReason
}

type Pause struct {
	From, To time.Time
	Elapsed  time.Duration
	Reason   PauseReason
}

type Clock struct {
	set          bool
	storyAt      time.Time
	running      bool
	runningSince time.Time
	// baseline do Play atual: só Play (e Reconstruct) escrevem; ajustes e
	// checkpoints não mexem, para Pause reportar a corrida inteira.
	playStoryAt time.Time
	playRealAt  time.Time
	limitAt     time.Time
	limitReason LimitReason
}

func norm(t time.Time) time.Time { return t.UTC().Truncate(time.Second) }

func NewUnset() *Clock { return &Clock{} }

func NewAt(storyAt time.Time) *Clock {
	c := &Clock{}
	_ = c.Start(storyAt)
	return c
}

func Reconstruct(s Snapshot) *Clock {
	if s.StoryAt == nil {
		return NewUnset()
	}
	c := &Clock{set: true, storyAt: norm(*s.StoryAt)}
	if s.LimitAt != nil {
		c.limitAt = norm(*s.LimitAt)
		c.limitReason = s.LimitReason
	} else {
		c.limitAt = c.storyAt
		c.limitReason = LimitStart
	}
	// rodando sem referência real não é reconstruível: fica pausado
	if s.Running && s.RunningSince != nil {
		c.running = true
		c.runningSince = s.RunningSince.UTC()
		// o baseline do Play de um relógio reconstruído é o checkpoint
		// (StoryAt, RunningSince): um restart perde no máximo o intervalo
		// desde o último checkpoint, o que o spec aceita.
		c.playStoryAt = c.storyAt
		c.playRealAt = c.runningSince
	}
	return c
}

func (c *Clock) IsSet() bool   { return c.set }
func (c *Clock) Running() bool { return c.running }

// Now devolve o tempo da história em real; relógio não definido devolve
// o time.Time zero.
func (c *Clock) Now(real time.Time) time.Time {
	if !c.set {
		return time.Time{}
	}
	if !c.running {
		return c.storyAt
	}
	elapsed := real.Sub(c.runningSince)
	if elapsed < 0 {
		elapsed = 0
	}
	return norm(c.storyAt.Add(elapsed))
}

func (c *Clock) Limit() (time.Time, LimitReason) { return c.limitAt, c.limitReason }

func (c *Clock) Start(at time.Time) error {
	if c.set {
		return ErrAlreadySet
	}
	c.set = true
	c.storyAt = norm(at)
	c.limitAt = c.storyAt
	c.limitReason = LimitStart
	return nil
}

// CheckMove valida um movimento para `to` sem mutar o relógio.
func (c *Clock) CheckMove(to, minimum time.Time, confirmLong bool, real time.Time) error {
	if !c.set {
		return ErrNotSet
	}
	to = norm(to)
	if to.Before(minimum) {
		return ErrBelowLimit
	}
	if to.Sub(c.Now(real)) > LongSkip && !confirmLong {
		return ErrLongSkipUnconfirmed
	}
	return nil
}

func (c *Clock) move(to time.Time, confirmLong bool, real time.Time) (time.Time, error) {
	if !c.set {
		return time.Time{}, ErrNotSet
	}
	from := c.Now(real)
	to = norm(to)
	if to.Equal(from) {
		return from, ErrZeroDelta
	}
	if err := c.CheckMove(to, c.limitAt, confirmLong, real); err != nil {
		return from, err
	}
	c.storyAt = to
	if c.running {
		c.runningSince = real
	}
	return from, nil
}

func (c *Clock) Adjust(delta time.Duration, confirmLong bool, real time.Time) (from, to time.Time, err error) {
	if !c.set {
		return time.Time{}, time.Time{}, ErrNotSet
	}
	to = c.Now(real).Add(delta)
	from, err = c.move(to, confirmLong, real)
	if err != nil {
		return from, time.Time{}, err
	}
	return from, norm(to), nil
}

func (c *Clock) Set(to time.Time, confirmLong bool, real time.Time) (from time.Time, err error) {
	return c.move(to, confirmLong, real)
}

func (c *Clock) Play(real time.Time) error {
	if !c.set {
		return ErrNotSet
	}
	if c.running {
		return ErrAlreadyRunning
	}
	c.running = true
	c.runningSince = real
	c.playStoryAt = c.storyAt
	c.playRealAt = real
	return nil
}

// Pause devolve nil (sem mudar nada) quando o relógio não estava rodando.
func (c *Clock) Pause(reason PauseReason, real time.Time) *Pause {
	if !c.set || !c.running {
		return nil
	}
	from := c.playStoryAt
	to := c.Now(real)
	elapsed := real.Sub(c.playRealAt)
	if elapsed < 0 {
		elapsed = 0
	}
	c.storyAt = to
	c.running = false
	return &Pause{From: from, To: to, Elapsed: elapsed.Truncate(time.Second), Reason: reason}
}

// RaiseLimit fixa o limite no tempo atual da história.
func (c *Clock) RaiseLimit(reason LimitReason, real time.Time) {
	if !c.set {
		return
	}
	c.limitAt = c.Now(real)
	c.limitReason = reason
}

// AddRoundSecond avança 1 s (1 round = 1 s) e sobe o limite.
func (c *Clock) AddRoundSecond(real time.Time) {
	if !c.set {
		return
	}
	c.storyAt = c.Now(real).Add(time.Second)
	if c.running {
		c.runningSince = real
	}
	c.RaiseLimit(LimitRound, real)
}

// Minimum é o menor tempo permitido: max(limite, agora+extra).
func (c *Clock) Minimum(extra time.Duration, real time.Time) time.Time {
	m := c.Now(real).Add(extra)
	if c.limitAt.After(m) {
		return c.limitAt
	}
	return m
}

func (c *Clock) Snapshot(real time.Time) Snapshot {
	if !c.set {
		return Snapshot{}
	}
	now := c.Now(real)
	limit := c.limitAt
	s := Snapshot{StoryAt: &now, Running: c.running, LimitAt: &limit, LimitReason: c.limitReason}
	if c.running {
		r := real
		s.RunningSince = &r
	}
	return s
}

// RoundEquivalents converte um intervalo em rounds (1 round = 1 s).
func RoundEquivalents(from, to time.Time) int {
	return int(to.Sub(from) / time.Second)
}
