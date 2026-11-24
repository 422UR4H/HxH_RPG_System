package service

import (
	"sort"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/google/uuid"
)

// RollInput is everything Derive needs besides the dice themselves.
//
// Condition is the master's struct for this one roll: their dice bias and their manual
// adjustment. Ledger is the character's standing accumulated difference. They are summed
// here, never merged upstream, so the master can cancel the system's disadvantage without
// either one overwriting the other.
type RollInput struct {
	SkillName  string
	SkillValue int                   // already resolved from the sheet by the caller
	Passive    bool                  // take the set's base value (its average) instead of rolling
	Condition  *action.RollCondition // master-owned; nil = neutral
	// Ledger is character-owned; nil = empty. WHAT each entry modifies is now the entry's own
	// business (match.Modifier.Applies), read against Dimension below — the caller no longer
	// decides by passing nil. An earlier comment here claimed the accumulated difference was
	// "always an actionSpeed adjustment, never a hit adjustment": that was true of the duel
	// reserve and false as a general law. The closed dodge's reserve modifies the dodge.
	Ledger *match.ModifierLedger
	// Dimension is which reserve this roll reads out of the ledger. The zero value reads
	// nothing, which is the right default for a test no reserve applies to.
	Dimension match.Dimension
	// SystemBias is advantage/disadvantage the ENGINE imposes on this one roll — the
	// action→reaction conversion is the first one. It is not the ledger (that is the
	// character's, and lasts until it expires) and not RollCondition (that is the master's).
	// Three origins, three homes, none overwriting another.
	SystemBias int
	AgainstID  *uuid.UUID // whom the roll is against; nil = nobody in particular
	// Status é o que os efeitos de status do personagem fazem com este teste — a quarta
	// origem de viés, ao lado do sistema, do mestre e do ledger. O zero não muda nada.
	Status StatusRoll
}

// StatusRoll é a leitura, para um teste, dos efeitos de status de quem testa
// (statuseffect.RollEffect). Bias soma níveis como os outros vieses; Penalty já vem com
// sinal (negativo = penalidade: Lento 2 → −2) e soma ao modificador. AutoFail é a falha
// automática: o teste não é rolado, total 0, sem crítico. RolledOnly faz um teste passivo
// ser rolado.
type StatusRoll struct {
	Bias       int
	Penalty    int
	AutoFail   bool
	RolledOnly bool
}

// RollOutcome is the derived result of one test. The individual dice survive because a
// critical is the combination, not the sum.
type RollOutcome struct {
	SkillName         string
	SkillValue        int
	Dice              []int // os dados MANTIDOS; vazio num teste passivo sem viés
	DiceTotal         int
	Bias              int // viés líquido acumulado (sistema + mestre + ledger + status), limitado a ±MaxBiasLevels
	Modifier          int // ajuste fixo líquido acumulado (mestre + ledger + status)
	Passive           bool
	Total             int
	IsCritical        bool
	IsCriticalFailure bool
	// Rolled é o monte inteiro lido (conjunto + extras do viés), na ordem em que caiu; Dice é
	// o que dele foi mantido.
	Rolled []int
	// AutoFailed: um efeito de status fez o teste falhar sem rolar (Total 0, sem crítico).
	AutoFailed bool
	// Short é quantos dados faltaram para ler este teste como o viés pede: o conjunto inteiro
	// quando ainda não caiu nenhum, mais os extras que faltam. Derive nunca rola — deriva com
	// o que existe e avisa; quem pode rolar (a sessão) tira esses dados uma vez (TopUp) e
	// deriva de novo. Não é gravado.
	Short int
}

// Margin is the reading of this outcome against a difficulty class. The result of a test
// is the margin, not a boolean — success and failure are readings of the margin against
// thresholds.
//
// It is a method rather than a field because the CD comes from the opposed roll, which
// does not exist until the collision is implemented.
func (o RollOutcome) Margin(cd int) int { return o.Total - cd }

// RollCalculator is a stateless domain service that turns dice plus a character's numbers
// into a result. It rolls once (Roll) and derives as many times as the master edits
// (Derive).
type RollCalculator struct{}

// Roll rola o conjunto da partida (Primary), uma vez, quando a ação ou reação chega. Os
// extras de vantagem/desvantagem não caem aqui: caem só quando um viés os pede (TopUp).
// Um src nil é o rolador de produção.
func (rc RollCalculator) Roll(rules match.MatchRules, src RollSource) action.RollAttempts {
	return action.RollAttempts{Primary: rc.RollDice(rules.DiceSet.Dice(), src)}
}

// TopUp tira os need dados que faltam a um teste, uma vez: primeiro o conjunto, se ainda não
// caiu, e o resto vai para os extras. Nada que já caiu é rolado de novo. need vem do Short de
// um Derive.
func (rc RollCalculator) TopUp(rules match.MatchRules, a *action.RollAttempts, need int, src RollSource) {
	if a == nil || need <= 0 {
		return
	}
	if len(a.Primary) == 0 {
		a.Primary = rc.RollDice(rules.DiceSet.Dice(), src)
		need -= len(a.Primary)
	}
	if need <= 0 {
		return
	}
	extra := make([]enum.DieSides, need)
	for i := range extra {
		extra[i] = rules.DiceSet.Dice()[0]
	}
	a.Extra = append(a.Extra, rc.RollDice(extra, src)...)
}

// RollDice rolls an arbitrary set of dice, in order. Damage needs it: a weapon carries its
// own dice (a Sword is D10 + D4), which are not the match's test set.
func (rc RollCalculator) RollDice(sides []enum.DieSides, src RollSource) []int {
	s := sourceOrDefault(src)
	out := make([]int, 0, len(sides))
	for _, face := range sides {
		out = append(out, s.RollDie(face))
	}
	return out
}

// Derive computes the outcome from dice already rolled. Pure: same inputs, same output,
// no new dice. Every master edit goes through here.
//
// O viés soma as quatro origens (sistema, mestre, ledger, status), cada nível cancelando um
// do sinal oposto, e é limitado a ±MaxBiasLevels. Cada nível lê um dado extra; a vantagem
// mantém os SetSize maiores do monte, a desvantagem os menores. Faltando dados, deriva com
// o que existe e informa em Short. O crítico é lido nos dados mantidos, só quando eles contam,
// e aplicado ao total inteiro por MatchRules.CriticalTotal.
func (rc RollCalculator) Derive(
	rules match.MatchRules, attempts action.RollAttempts, in RollInput,
) RollOutcome {
	bias, modifier := in.SystemBias+in.Status.Bias, in.Status.Penalty
	if in.Condition != nil {
		bias += in.Condition.Bias
		modifier += in.Condition.Modifier
	}
	if in.Ledger != nil {
		bias += in.Ledger.TotalBias(in.Dimension, in.AgainstID)
		modifier += in.Ledger.TotalAmount(in.Dimension, in.AgainstID)
	}
	bias = max(-match.MaxBiasLevels, min(match.MaxBiasLevels, bias))
	passive := in.Passive && !in.Status.RolledOnly

	out := RollOutcome{
		SkillName:  in.SkillName,
		SkillValue: in.SkillValue,
		Bias:       bias,
		Modifier:   modifier,
		Passive:    passive,
	}

	// Falha automática: nenhum dado é lido nem pedido, e não há crítico.
	if in.Status.AutoFail {
		out.AutoFailed = true
		return out
	}

	diceCount := !passive
	if passive {
		// A passive test takes the average of the set instead of rolling, so rolling has
		// exactly zero expected gain. No dice means no critical either way.
		out.DiceTotal = rules.PassiveValue()
	}
	if !passive || bias != 0 {
		pool, short := dicePool(rules, attempts, bias)
		kept := keepDice(pool, bias, rules.DiceSet.SetSize())
		out.Short = short
		if len(pool) > 0 {
			out.Rolled = pool
			out.Dice = kept
		}
		if passive {
			diceCount = readPassiveBias(&out, kept, bias)
		} else {
			out.DiceTotal = sumDice(kept)
		}
	}
	critical, failure := false, false
	if diceCount {
		critical = allDiceShow(out.Dice, rules.DiceSet.MaxFace())
		failure = allDiceShow(out.Dice, 1)
	}
	out.IsCritical, out.IsCriticalFailure = critical, failure
	out.Total = rules.CriticalTotal(out.DiceTotal+in.SkillValue+modifier, critical, failure)
	return out
}

// dicePool é o monte que um viés lê: o conjunto mais um extra por nível, na ordem em que
// caíram (uma cópia, nunca um alias de attempts). short é o que falta para lê-lo inteiro: os
// extras que ainda não caíram, mais o conjunto quando nem ele caiu.
func dicePool(rules match.MatchRules, a action.RollAttempts, bias int) (pool []int, short int) {
	levels := bias
	if levels < 0 {
		levels = -levels
	}
	n := min(levels, len(a.Extra))
	pool = make([]int, 0, len(a.Primary)+n)
	pool = append(pool, a.Primary...)
	pool = append(pool, a.Extra[:n]...)
	short = levels - len(a.Extra)
	if short < 0 {
		short = 0
	}
	if len(a.Primary) == 0 {
		short += rules.DiceSet.SetSize()
	}
	return pool, short
}

// keepDice mantém size dados do monte: os maiores numa vantagem, os menores numa
// desvantagem, o conjunto (os primeiros) sem viés. Entre dados iguais, fica o que caiu
// primeiro. Os mantidos saem na ordem do monte.
func keepDice(pool []int, bias, size int) []int {
	if len(pool) <= size {
		return append([]int(nil), pool...)
	}
	if bias == 0 {
		return append([]int(nil), pool[:size]...)
	}
	idx := make([]int, len(pool))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(i, j int) bool {
		if bias > 0 {
			return pool[idx[i]] > pool[idx[j]]
		}
		return pool[idx[i]] < pool[idx[j]]
	})
	keep := make([]bool, len(pool))
	for _, i := range idx[:size] {
		keep[i] = true
	}
	kept := make([]int, 0, size)
	for i, d := range pool {
		if keep[i] {
			kept = append(kept, d)
		}
	}
	return kept
}

// readPassiveBias é um viés num teste passivo (decisões do dono, 2026-10-12, 2026-10-13 e
// spec §16.4 item 28): o teste passa a rolar o conjunto mais um dado por nível, e o par
// mantido é comparado com o valor base — a vantagem fica com o melhor dos dois, a
// desvantagem com o pior; empate fica com a base. A passiva continua sendo o piso (ou o
// teto): o viés acrescenta risco ou chance, nunca uma rolagem de graça.
//
// O viés é o LÍQUIDO, o mesmo que um teste rolado lê: o do mestre, o do sistema (a
// desvantagem de uma troca), o do ledger e o dos status contam igual e se cancelam igual.
//
// Os dados caem na primeira vez que um viés os pede, pela sessão (Short + TopUp), e nunca
// de novo. Sem dados não há o que ler e a base fica. Devolve se os dados é que contam —
// só então o crítico é lido.
func readPassiveBias(out *RollOutcome, kept []int, bias int) bool {
	if len(kept) == 0 {
		return false
	}
	sum := sumDice(kept)
	if (bias > 0 && sum > out.DiceTotal) || (bias < 0 && sum < out.DiceTotal) {
		out.DiceTotal = sum
		return true
	}
	return false
}

func sumDice(dice []int) int {
	total := 0
	for _, d := range dice {
		total += d
	}
	return total
}

// allDiceShow reports whether every die of the set landed on face. A critical is read on
// the individual dice, never on the sum.
func allDiceShow(dice []int, face int) bool {
	if len(dice) == 0 {
		return false
	}
	for _, d := range dice {
		if d != face {
			return false
		}
	}
	return true
}
