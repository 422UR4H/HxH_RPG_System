package game

import (
	"context"
	"sync"
	"testing"
	"time"

	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/clock"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
)

type countingClockRepo struct {
	mu    sync.Mutex
	saves []clock.Snapshot
}

func (r *countingClockRepo) Load(context.Context, uuid.UUID) (*clock.Snapshot, error) {
	return nil, nil
}

func (r *countingClockRepo) Save(_ context.Context, _ uuid.UUID, snap clock.Snapshot, _ *appmatch.ClockEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.saves = append(r.saves, snap)
	return nil
}

func (r *countingClockRepo) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.saves)
}

// Um checkpoint cujo retrato é mais velho que outra gravação do relógio (um fechamento de turno,
// uma troca de cena) não grava: sobrescreveria o relógio mais novo com o dele.
func TestClockCheckpointSkipsWhenANewerWriteHappened(t *testing.T) {
	repo := &countingClockRepo{}
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	r := NewRoom(uuid.New(), uuid.New(), RoomDeps{ClockRepo: repo, Now: func() time.Time { return now }})
	sess := matchsession.NewMatchSession(r.matchUUID, nil, nil)
	if err := sess.Clock().Play(now); err != nil {
		t.Fatal(err)
	}

	r.mu.RLock()
	stale := sess.ClockSnapshot(r.now())
	gen := r.clockWriteGen.Load()
	// outra gravação leva um retrato depois do do checkpoint (ex.: PersistTurnClose)
	_ = r.clockSnapshotForWriteLocked(sess)
	r.mu.RUnlock()

	if r.writeClockCheckpoint(stale, gen) || repo.count() != 0 {
		t.Fatalf("a checkpoint older than another clock write was written (%d saves)", repo.count())
	}

	r.mu.RLock()
	fresh := sess.ClockSnapshot(r.now())
	gen = r.clockWriteGen.Load()
	r.mu.RUnlock()
	if !r.writeClockCheckpoint(fresh, gen) || repo.count() != 1 {
		t.Fatalf("a current checkpoint was not written (%d saves)", repo.count())
	}
}
