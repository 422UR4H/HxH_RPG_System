package match_test

import (
	"context"
	"errors"
	"testing"
	"time"

	matchApp "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

func TestRemoveMatchNPC(t *testing.T) {
	ctx := context.Background()

	masterUUID := uuid.New()
	matchUUID := uuid.New()
	campaignUUID := uuid.New()
	sheetUUID := uuid.New()

	t.Run("match not found", func(t *testing.T) {
		matchReader := &mockMatchReader{
			GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
				return nil, matchPg.ErrMatchNotFound
			},
		}
		roster := &mockNPCRoster{}

		uc := matchApp.NewRemoveMatchNPCUC(matchReader, roster)
		err := uc.Remove(ctx, &matchApp.RemoveMatchNPCInput{
			RequesterUUID: masterUUID, MatchUUID: matchUUID, SheetUUID: sheetUUID,
		})
		if !errors.Is(err, matchApp.ErrMatchNotFound) {
			t.Fatalf("expected ErrMatchNotFound, got: %v", err)
		}
	})

	t.Run("requester is not the match master", func(t *testing.T) {
		otherUUID := uuid.New()
		matchReader := &mockMatchReader{
			GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
				return &matchEntity.Match{UUID: matchUUID, MasterUUID: masterUUID, CampaignUUID: campaignUUID}, nil
			},
		}
		rosterCalled := false
		roster := &mockNPCRoster{
			RemoveNPCParticipantFn: func(ctx context.Context, mUUID, sUUID uuid.UUID) error {
				rosterCalled = true
				return nil
			},
		}

		uc := matchApp.NewRemoveMatchNPCUC(matchReader, roster)
		err := uc.Remove(ctx, &matchApp.RemoveMatchNPCInput{
			RequesterUUID: otherUUID, MatchUUID: matchUUID, SheetUUID: sheetUUID,
		})
		if !errors.Is(err, matchApp.ErrNotMatchMaster) {
			t.Fatalf("expected ErrNotMatchMaster, got: %v", err)
		}
		if rosterCalled {
			t.Fatal("npc roster must not be called when the requester isn't the master")
		}
	})

	t.Run("npc not in the match", func(t *testing.T) {
		matchReader := &mockMatchReader{
			GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
				return &matchEntity.Match{UUID: matchUUID, MasterUUID: masterUUID, CampaignUUID: campaignUUID}, nil
			},
		}
		roster := &mockNPCRoster{
			RemoveNPCParticipantFn: func(ctx context.Context, mUUID, sUUID uuid.UUID) error {
				return matchPg.ErrNPCNotInMatch
			},
		}

		uc := matchApp.NewRemoveMatchNPCUC(matchReader, roster)
		err := uc.Remove(ctx, &matchApp.RemoveMatchNPCInput{
			RequesterUUID: masterUUID, MatchUUID: matchUUID, SheetUUID: sheetUUID,
		})
		if !errors.Is(err, matchApp.ErrNPCNotInMatch) {
			t.Fatalf("expected ErrNPCNotInMatch, got: %v", err)
		}
	})

	t.Run("removes the npc on a finished match (no StoryEndAt guard, D5)", func(t *testing.T) {
		finishedAt := time.Now().Add(-time.Hour)
		matchReader := &mockMatchReader{
			GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
				return &matchEntity.Match{
					UUID: matchUUID, MasterUUID: masterUUID, CampaignUUID: campaignUUID,
					StoryEndAt: &finishedAt,
				}, nil
			},
		}
		var gotMatchUUID, gotSheetUUID uuid.UUID
		roster := &mockNPCRoster{
			RemoveNPCParticipantFn: func(ctx context.Context, mUUID, sUUID uuid.UUID) error {
				gotMatchUUID, gotSheetUUID = mUUID, sUUID
				return nil
			},
		}

		uc := matchApp.NewRemoveMatchNPCUC(matchReader, roster)
		err := uc.Remove(ctx, &matchApp.RemoveMatchNPCInput{
			RequesterUUID: masterUUID, MatchUUID: matchUUID, SheetUUID: sheetUUID,
		})
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if gotMatchUUID != matchUUID || gotSheetUUID != sheetUUID {
			t.Fatalf("expected roster called with (%v, %v), got (%v, %v)",
				matchUUID, sheetUUID, gotMatchUUID, gotSheetUUID)
		}
	})

	t.Run("removes the npc successfully", func(t *testing.T) {
		matchReader := &mockMatchReader{
			GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
				return &matchEntity.Match{UUID: matchUUID, MasterUUID: masterUUID, CampaignUUID: campaignUUID}, nil
			},
		}
		roster := &mockNPCRoster{
			RemoveNPCParticipantFn: func(ctx context.Context, mUUID, sUUID uuid.UUID) error {
				return nil
			},
		}

		uc := matchApp.NewRemoveMatchNPCUC(matchReader, roster)
		err := uc.Remove(ctx, &matchApp.RemoveMatchNPCInput{
			RequesterUUID: masterUUID, MatchUUID: matchUUID, SheetUUID: sheetUUID,
		})
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
	})
}
