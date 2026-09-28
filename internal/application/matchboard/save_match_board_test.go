package matchboarduc_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	matchboarduc "github.com/422UR4H/HxH_RPG_System/internal/application/matchboard"
	fogentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchboard"
)

type fakeSaveBoardRepo struct {
	err   error
	calls int
	saved *matchboard.Board
}

func (f *fakeSaveBoardRepo) Save(_ context.Context, b *matchboard.Board) error {
	f.calls++
	f.saved = b
	return f.err
}

type fakeMemoryRepo struct {
	errFor map[uuid.UUID]error
	saved  []fogentity.PlayerMemory
}

func (f *fakeMemoryRepo) Upsert(_ context.Context, m fogentity.PlayerMemory) error {
	if err, ok := f.errFor[m.PlayerID]; ok {
		return err
	}
	f.saved = append(f.saved, m)
	return nil
}

func TestSaveMatchBoardUC_Save(t *testing.T) {
	matchUUID := uuid.New()
	mapUUID := uuid.New()

	t.Run("saves the board row and upserts every memory", func(t *testing.T) {
		boardRepo := &fakeSaveBoardRepo{}
		memoryRepo := &fakeMemoryRepo{}
		uc := matchboarduc.NewSaveMatchBoardUC(boardRepo, memoryRepo)

		p1, p2 := uuid.New(), uuid.New()
		mems := []fogentity.PlayerMemory{
			{PlayerID: p1, MatchID: matchUUID, MapID: mapUUID},
			{PlayerID: p2, MatchID: matchUUID, MapID: mapUUID},
		}
		b := &matchboard.Board{MatchUUID: matchUUID, MapUUID: mapUUID}

		if err := uc.Save(context.Background(), b, mems); err != nil {
			t.Fatalf("Save returned an error: %v", err)
		}
		if boardRepo.calls != 1 {
			t.Fatalf("board repo Save called %d time(s), want 1", boardRepo.calls)
		}
		if boardRepo.saved != b {
			t.Fatalf("board repo got %+v, want the exact board %+v", boardRepo.saved, b)
		}
		if len(memoryRepo.saved) != 2 {
			t.Fatalf("memory repo got %d upsert(s), want 2", len(memoryRepo.saved))
		}
	})

	t.Run("a board save failure aborts before any memory is attempted", func(t *testing.T) {
		boardRepo := &fakeSaveBoardRepo{err: errors.New("boom")}
		memoryRepo := &fakeMemoryRepo{}
		uc := matchboarduc.NewSaveMatchBoardUC(boardRepo, memoryRepo)

		mems := []fogentity.PlayerMemory{{PlayerID: uuid.New(), MatchID: matchUUID, MapID: mapUUID}}
		b := &matchboard.Board{MatchUUID: matchUUID, MapUUID: mapUUID}

		if err := uc.Save(context.Background(), b, mems); err == nil {
			t.Fatal("Save returned no error for a board repository failure")
		}
		if len(memoryRepo.saved) != 0 {
			t.Fatalf("memory repo was called %d time(s), want 0 — the board row failed, nothing "+
				"to be consistent with", len(memoryRepo.saved))
		}
	})

	t.Run("one memory failing does not stop the others, and every error comes back joined", func(t *testing.T) {
		boardRepo := &fakeSaveBoardRepo{}
		bad := uuid.New()
		good1, good2 := uuid.New(), uuid.New()
		memoryRepo := &fakeMemoryRepo{errFor: map[uuid.UUID]error{bad: errors.New("player memory boom")}}
		uc := matchboarduc.NewSaveMatchBoardUC(boardRepo, memoryRepo)

		mems := []fogentity.PlayerMemory{
			{PlayerID: good1, MatchID: matchUUID, MapID: mapUUID},
			{PlayerID: bad, MatchID: matchUUID, MapID: mapUUID},
			{PlayerID: good2, MatchID: matchUUID, MapID: mapUUID},
		}
		b := &matchboard.Board{MatchUUID: matchUUID, MapUUID: mapUUID}

		err := uc.Save(context.Background(), b, mems)
		if err == nil {
			t.Fatal("Save returned no error, want the one memory failure joined back")
		}
		if len(memoryRepo.saved) != 2 {
			t.Fatalf("memory repo got %d successful upsert(s), want 2 — the other two must not "+
				"be skipped because one failed", len(memoryRepo.saved))
		}
		if !errors.Is(err, memoryRepo.errFor[bad]) {
			t.Fatalf("Save error %v does not wrap the underlying memory failure %v", err, memoryRepo.errFor[bad])
		}
	})

	t.Run("no memories is not an error", func(t *testing.T) {
		boardRepo := &fakeSaveBoardRepo{}
		memoryRepo := &fakeMemoryRepo{}
		uc := matchboarduc.NewSaveMatchBoardUC(boardRepo, memoryRepo)

		b := &matchboard.Board{MatchUUID: matchUUID, MapUUID: mapUUID}
		if err := uc.Save(context.Background(), b, nil); err != nil {
			t.Fatalf("Save returned an error for a board with no memories: %v", err)
		}
	})
}
