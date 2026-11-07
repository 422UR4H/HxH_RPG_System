package matchsession

import (
	"slices"
	"sort"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

// BarView is one character's one bar as a given recipient may see it.
type BarView struct {
	Carry float64
	// Balance is live, with no ceiling: BarEconomy.Balance(carry, acted ++ pending[:1], price).
	// Only the NEXT pending action counts (the first in projected order): what the bar will
	// read once it opens. The ones behind it weigh nothing until they are the next. A player
	// has no pending, so theirs is acted only. It is just Carry when there is no price.
	Balance float64
	// Acted is every speed for the master, only the public ones for a player.
	Acted []int
	// Pending (master only) are the speeds that open this round, in projected order.
	Pending []int
	// Deferred (master only) are the queued speeds that do not fit this round.
	Deferred []int
}

// CharacterBarsView holds both bars of one character.
type CharacterBarsView struct {
	CharacterID  uuid.UUID
	Action, Move BarView
}

// BarsView is the domain view of both bars for one kind of recipient.
type BarsView struct {
	Prices      map[action.Bar]int // frozen; public
	Provisional map[action.Bar]int // master only (nil for players)
	Characters  []CharacterBarsView
	Order       []service.OrderSlot // master only (nil for players)
	RoundTurns  int
}

var viewBars = []action.Bar{action.BarAction, action.BarMove}

// BarsView builds the view for the master (everything, live) or for a player (only what the
// master has already opened: frozen prices and public speeds).
func (s *MatchSession) BarsView(isMaster bool) BarsView {
	in := s.scheduleInput()
	view := BarsView{Prices: s.RoundPrices(), RoundTurns: s.RoundTurnCount()}

	var provisional map[action.Bar]int
	var proj service.RoundProjection
	if isMaster {
		provisional = s.scheduler.ProvisionalPrices(in)
		proj = s.scheduler.ProjectRound(in)
		view.Provisional = provisional
		view.Order = proj.Order
	}

	ids := s.CharacterIDs()
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	view.Characters = make([]CharacterBarsView, 0, len(ids))
	for _, id := range ids {
		c := CharacterBarsView{CharacterID: id}
		for _, bar := range viewBars {
			bv := &c.Action
			if bar == action.BarMove {
				bv = &c.Move
			}
			var carry float64
			if isMaster {
				carry, bv.Acted = s.BarState(id, bar)
				bv.Pending = speedsOn(proj.Order, id, bar)
				bv.Deferred = speedsOn(proj.Deferred, id, bar)
			} else {
				carry, bv.Acted = s.PublicBarState(id, bar)
			}
			bv.Carry = carry
			bv.Balance = carry
			price, hasPrice := view.Prices[bar]
			if !hasPrice && isMaster {
				price, hasPrice = provisional[bar]
			}
			if hasPrice {
				counted := slices.Clone(bv.Acted)
				if len(bv.Pending) > 0 {
					counted = append(counted, bv.Pending[0])
				}
				bv.Balance = service.BarEconomy{}.Balance(carry, counted, price)
			}
		}
		view.Characters = append(view.Characters, c)
	}
	return view
}

// speedsOn lists the speeds one character has on a bar across the slots, in slot order.
func speedsOn(slots []service.OrderSlot, id uuid.UUID, bar action.Bar) []int {
	var out []int
	for _, sl := range slots {
		if sl.ActorID != id {
			continue
		}
		if speed, ok := sl.Speeds[bar]; ok {
			out = append(out, speed)
		}
	}
	return out
}
