package match_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/application/testutil"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

type fakeRevealStore struct {
	hasEvent  bool
	hasErr    error
	insertErr error
	inserted  []struct {
		match, effect uuid.UUID
		at            time.Time
	}
	hasCalls int
}

func (f *fakeRevealStore) HasEventInMatch(_ context.Context, _, _ uuid.UUID) (bool, error) {
	f.hasCalls++
	return f.hasEvent, f.hasErr
}

func (f *fakeRevealStore) InsertReveal(_ context.Context, m, e uuid.UUID, at time.Time) error {
	if f.insertErr != nil {
		return f.insertErr
	}
	f.inserted = append(f.inserted, struct {
		match, effect uuid.UUID
		at            time.Time
	}{m, e, at})
	return nil
}

func TestRevealStatusEffect(t *testing.T) {
	ctx := context.Background()
	master, matchUUID, effect := uuid.New(), uuid.New(), uuid.New()
	reader := func(err error) match.IMatchReader {
		return &testutil.MockMatchRepo{GetMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
			if err != nil {
				return nil, err
			}
			return &matchEntity.Match{UUID: matchUUID, MasterUUID: master}, nil
		}}
	}

	t.Run("the master reveals and the instant is truncated to the microsecond", func(t *testing.T) {
		store := &fakeRevealStore{hasEvent: true}
		before := time.Now()
		if err := match.NewRevealStatusEffectUC(reader(nil), store, store).Reveal(ctx, matchUUID, effect, master); err != nil {
			t.Fatalf("Reveal: %v", err)
		}
		if len(store.inserted) != 1 || store.inserted[0].match != matchUUID || store.inserted[0].effect != effect {
			t.Fatalf("inserted = %+v", store.inserted)
		}
		at := store.inserted[0].at
		if at.Before(before.Truncate(time.Microsecond)) || at != at.Truncate(time.Microsecond) {
			t.Fatalf("at = %v, want now truncated to µs", at)
		}
	})

	t.Run("every call inserts a new row", func(t *testing.T) {
		store := &fakeRevealStore{hasEvent: true}
		uc := match.NewRevealStatusEffectUC(reader(nil), store, store)
		for i := 0; i < 2; i++ {
			if err := uc.Reveal(ctx, matchUUID, effect, master); err != nil {
				t.Fatalf("Reveal: %v", err)
			}
		}
		if len(store.inserted) != 2 {
			t.Fatalf("inserted = %d, want 2", len(store.inserted))
		}
	})

	t.Run("works after the match ended", func(t *testing.T) {
		ended := time.Now()
		r := &testutil.MockMatchRepo{GetMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
			return &matchEntity.Match{UUID: matchUUID, MasterUUID: master, GameEndAt: &ended}, nil
		}}
		store := &fakeRevealStore{hasEvent: true}
		if err := match.NewRevealStatusEffectUC(r, store, store).Reveal(ctx, matchUUID, effect, master); err != nil {
			t.Fatalf("Reveal: %v", err)
		}
	})

	t.Run("match not found", func(t *testing.T) {
		store := &fakeRevealStore{hasEvent: true}
		err := match.NewRevealStatusEffectUC(reader(matchPg.ErrMatchNotFound), store, store).Reveal(ctx, matchUUID, effect, master)
		if !errors.Is(err, match.ErrMatchNotFound) {
			t.Fatalf("err = %v, want ErrMatchNotFound", err)
		}
	})

	t.Run("a caller who is not the master is refused before anything is read", func(t *testing.T) {
		store := &fakeRevealStore{hasEvent: true}
		err := match.NewRevealStatusEffectUC(reader(nil), store, store).Reveal(ctx, matchUUID, effect, uuid.New())
		if !errors.Is(err, match.ErrNotMatchMaster) {
			t.Fatalf("err = %v, want ErrNotMatchMaster", err)
		}
		if store.hasCalls != 0 || len(store.inserted) != 0 {
			t.Fatal("the store was touched")
		}
	})

	t.Run("an effect with no event in the match is not found", func(t *testing.T) {
		store := &fakeRevealStore{hasEvent: false}
		err := match.NewRevealStatusEffectUC(reader(nil), store, store).Reveal(ctx, matchUUID, effect, master)
		if !errors.Is(err, statuseffect.ErrNotFound) || len(store.inserted) != 0 {
			t.Fatalf("err = %v, inserted = %d", err, len(store.inserted))
		}
	})

	t.Run("the nil effect is never revealable, even when the store says it has an event", func(t *testing.T) {
		store := &fakeRevealStore{hasEvent: true}
		err := match.NewRevealStatusEffectUC(reader(nil), store, store).Reveal(ctx, matchUUID, uuid.Nil, master)
		if !errors.Is(err, statuseffect.ErrNotFound) || len(store.inserted) != 0 {
			t.Fatalf("err = %v, inserted = %d", err, len(store.inserted))
		}
	})

	t.Run("store errors surface", func(t *testing.T) {
		boom := errors.New("boom")
		store := &fakeRevealStore{hasErr: boom}
		if err := match.NewRevealStatusEffectUC(reader(nil), store, store).Reveal(ctx, matchUUID, effect, master); !errors.Is(err, boom) {
			t.Fatalf("has: %v", err)
		}
		store = &fakeRevealStore{hasEvent: true, insertErr: boom}
		if err := match.NewRevealStatusEffectUC(reader(nil), store, store).Reveal(ctx, matchUUID, effect, master); !errors.Is(err, boom) {
			t.Fatalf("insert: %v", err)
		}
	})
}
