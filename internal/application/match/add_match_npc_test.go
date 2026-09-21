package match_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/application/character_sheet"
	matchApp "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

// mockMatchReader is local to this test package on purpose — see
// AddMatchNPCUC's IMatchReader doc for why the interface isn't widened onto
// the shared testutil mock.
type mockMatchReader struct {
	GetMatchFn func(ctx context.Context, uuid uuid.UUID) (*matchEntity.Match, error)
}

func (m *mockMatchReader) GetMatch(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
	return m.GetMatchFn(ctx, id)
}

// mockSheetOwnershipReader tracks whether it was called so tests can assert
// the requester's authorization is checked before any sheet is read.
type mockSheetOwnershipReader struct {
	GetCharacterSheetRelationshipUUIDsFn func(ctx context.Context, uuid uuid.UUID) (csEntity.RelationshipUUIDs, error)
	called                               bool
}

func (m *mockSheetOwnershipReader) GetCharacterSheetRelationshipUUIDs(
	ctx context.Context, id uuid.UUID,
) (csEntity.RelationshipUUIDs, error) {
	m.called = true
	return m.GetCharacterSheetRelationshipUUIDsFn(ctx, id)
}

type mockNPCRoster struct {
	AddNPCParticipantFn func(
		ctx context.Context, matchUUID, sheetUUID uuid.UUID, joinedAt time.Time,
	) (*matchEntity.Participant, error)
	RemoveNPCParticipantFn func(ctx context.Context, matchUUID, sheetUUID uuid.UUID) error
}

func (m *mockNPCRoster) AddNPCParticipant(
	ctx context.Context, matchUUID, sheetUUID uuid.UUID, joinedAt time.Time,
) (*matchEntity.Participant, error) {
	return m.AddNPCParticipantFn(ctx, matchUUID, sheetUUID, joinedAt)
}

func (m *mockNPCRoster) RemoveNPCParticipant(ctx context.Context, matchUUID, sheetUUID uuid.UUID) error {
	return m.RemoveNPCParticipantFn(ctx, matchUUID, sheetUUID)
}

func TestAddMatchNPC(t *testing.T) {
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
		sheets := &mockSheetOwnershipReader{}
		roster := &mockNPCRoster{}

		uc := matchApp.NewAddMatchNPCUC(matchReader, sheets, roster)
		_, err := uc.Add(ctx, &matchApp.AddMatchNPCInput{
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
		sheets := &mockSheetOwnershipReader{}
		roster := &mockNPCRoster{}

		uc := matchApp.NewAddMatchNPCUC(matchReader, sheets, roster)
		_, err := uc.Add(ctx, &matchApp.AddMatchNPCInput{
			RequesterUUID: otherUUID, MatchUUID: matchUUID, SheetUUID: sheetUUID,
		})
		if !errors.Is(err, matchApp.ErrNotMatchMaster) {
			t.Fatalf("expected ErrNotMatchMaster, got: %v", err)
		}
		if sheets.called {
			t.Fatal("sheet ownership reader must not be called when the requester isn't the master")
		}
	})

	t.Run("match already finished", func(t *testing.T) {
		finishedAt := time.Now().Add(-time.Hour)
		matchReader := &mockMatchReader{
			GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
				return &matchEntity.Match{
					UUID: matchUUID, MasterUUID: masterUUID, CampaignUUID: campaignUUID,
					StoryEndAt: &finishedAt,
				}, nil
			},
		}
		sheets := &mockSheetOwnershipReader{}
		roster := &mockNPCRoster{}

		uc := matchApp.NewAddMatchNPCUC(matchReader, sheets, roster)
		_, err := uc.Add(ctx, &matchApp.AddMatchNPCInput{
			RequesterUUID: masterUUID, MatchUUID: matchUUID, SheetUUID: sheetUUID,
		})
		if !errors.Is(err, matchApp.ErrMatchAlreadyFinished) {
			t.Fatalf("expected ErrMatchAlreadyFinished, got: %v", err)
		}
	})

	t.Run("character sheet not found", func(t *testing.T) {
		matchReader := &mockMatchReader{
			GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
				return &matchEntity.Match{UUID: matchUUID, MasterUUID: masterUUID, CampaignUUID: campaignUUID}, nil
			},
		}
		sheets := &mockSheetOwnershipReader{
			GetCharacterSheetRelationshipUUIDsFn: func(ctx context.Context, id uuid.UUID) (csEntity.RelationshipUUIDs, error) {
				return csEntity.RelationshipUUIDs{}, charactersheet.ErrCharacterSheetNotFound
			},
		}
		roster := &mockNPCRoster{}

		uc := matchApp.NewAddMatchNPCUC(matchReader, sheets, roster)
		_, err := uc.Add(ctx, &matchApp.AddMatchNPCInput{
			RequesterUUID: masterUUID, MatchUUID: matchUUID, SheetUUID: sheetUUID,
		})
		if !errors.Is(err, matchApp.ErrCharacterSheetNotFound) {
			t.Fatalf("expected ErrCharacterSheetNotFound, got: %v", err)
		}
	})

	t.Run("sheet belongs to a player", func(t *testing.T) {
		playerUUID := uuid.New()
		matchReader := &mockMatchReader{
			GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
				return &matchEntity.Match{UUID: matchUUID, MasterUUID: masterUUID, CampaignUUID: campaignUUID}, nil
			},
		}
		sheets := &mockSheetOwnershipReader{
			GetCharacterSheetRelationshipUUIDsFn: func(ctx context.Context, id uuid.UUID) (csEntity.RelationshipUUIDs, error) {
				return csEntity.RelationshipUUIDs{PlayerUUID: &playerUUID}, nil
			},
		}
		roster := &mockNPCRoster{}

		uc := matchApp.NewAddMatchNPCUC(matchReader, sheets, roster)
		_, err := uc.Add(ctx, &matchApp.AddMatchNPCInput{
			RequesterUUID: masterUUID, MatchUUID: matchUUID, SheetUUID: sheetUUID,
		})
		if !errors.Is(err, matchApp.ErrSheetNotNPC) {
			t.Fatalf("expected ErrSheetNotNPC, got: %v", err)
		}
	})

	t.Run("npc of another master and another campaign", func(t *testing.T) {
		otherMasterUUID := uuid.New()
		otherCampaignUUID := uuid.New()
		matchReader := &mockMatchReader{
			GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
				return &matchEntity.Match{UUID: matchUUID, MasterUUID: masterUUID, CampaignUUID: campaignUUID}, nil
			},
		}
		sheets := &mockSheetOwnershipReader{
			GetCharacterSheetRelationshipUUIDsFn: func(ctx context.Context, id uuid.UUID) (csEntity.RelationshipUUIDs, error) {
				return csEntity.RelationshipUUIDs{MasterUUID: &otherMasterUUID, CampaignUUID: &otherCampaignUUID}, nil
			},
		}
		roster := &mockNPCRoster{}

		uc := matchApp.NewAddMatchNPCUC(matchReader, sheets, roster)
		_, err := uc.Add(ctx, &matchApp.AddMatchNPCInput{
			RequesterUUID: masterUUID, MatchUUID: matchUUID, SheetUUID: sheetUUID,
		})
		if !errors.Is(err, matchApp.ErrSheetNotOwnedByMaster) {
			t.Fatalf("expected ErrSheetNotOwnedByMaster, got: %v", err)
		}
	})

	t.Run("npc of another master but the match's campaign succeeds (D2)", func(t *testing.T) {
		otherMasterUUID := uuid.New()
		now := time.Now()
		wantParticipant := &matchEntity.Participant{UUID: uuid.New(), MatchUUID: matchUUID, JoinedAt: now}

		matchReader := &mockMatchReader{
			GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
				return &matchEntity.Match{UUID: matchUUID, MasterUUID: masterUUID, CampaignUUID: campaignUUID}, nil
			},
		}
		sheets := &mockSheetOwnershipReader{
			GetCharacterSheetRelationshipUUIDsFn: func(ctx context.Context, id uuid.UUID) (csEntity.RelationshipUUIDs, error) {
				return csEntity.RelationshipUUIDs{MasterUUID: &otherMasterUUID, CampaignUUID: &campaignUUID}, nil
			},
		}
		roster := &mockNPCRoster{
			AddNPCParticipantFn: func(
				ctx context.Context, mUUID, sUUID uuid.UUID, joinedAt time.Time,
			) (*matchEntity.Participant, error) {
				return wantParticipant, nil
			},
		}

		uc := matchApp.NewAddMatchNPCUC(matchReader, sheets, roster)
		got, err := uc.Add(ctx, &matchApp.AddMatchNPCInput{
			RequesterUUID: masterUUID, MatchUUID: matchUUID, SheetUUID: sheetUUID,
		})
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if got != wantParticipant {
			t.Fatalf("expected returned participant %+v, got %+v", wantParticipant, got)
		}
	})

	t.Run("npc owned directly by the requesting master succeeds", func(t *testing.T) {
		now := time.Now()
		wantParticipant := &matchEntity.Participant{UUID: uuid.New(), MatchUUID: matchUUID, JoinedAt: now}

		matchReader := &mockMatchReader{
			GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
				return &matchEntity.Match{UUID: matchUUID, MasterUUID: masterUUID, CampaignUUID: campaignUUID}, nil
			},
		}
		sheets := &mockSheetOwnershipReader{
			GetCharacterSheetRelationshipUUIDsFn: func(ctx context.Context, id uuid.UUID) (csEntity.RelationshipUUIDs, error) {
				return csEntity.RelationshipUUIDs{MasterUUID: &masterUUID}, nil
			},
		}
		roster := &mockNPCRoster{
			AddNPCParticipantFn: func(
				ctx context.Context, mUUID, sUUID uuid.UUID, joinedAt time.Time,
			) (*matchEntity.Participant, error) {
				return wantParticipant, nil
			},
		}

		uc := matchApp.NewAddMatchNPCUC(matchReader, sheets, roster)
		got, err := uc.Add(ctx, &matchApp.AddMatchNPCInput{
			RequesterUUID: masterUUID, MatchUUID: matchUUID, SheetUUID: sheetUUID,
		})
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if got != wantParticipant {
			t.Fatalf("expected returned participant %+v, got %+v", wantParticipant, got)
		}
	})

	t.Run("npc already in the match", func(t *testing.T) {
		matchReader := &mockMatchReader{
			GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
				return &matchEntity.Match{UUID: matchUUID, MasterUUID: masterUUID, CampaignUUID: campaignUUID}, nil
			},
		}
		sheets := &mockSheetOwnershipReader{
			GetCharacterSheetRelationshipUUIDsFn: func(ctx context.Context, id uuid.UUID) (csEntity.RelationshipUUIDs, error) {
				return csEntity.RelationshipUUIDs{MasterUUID: &masterUUID}, nil
			},
		}
		roster := &mockNPCRoster{
			AddNPCParticipantFn: func(
				ctx context.Context, mUUID, sUUID uuid.UUID, joinedAt time.Time,
			) (*matchEntity.Participant, error) {
				return nil, matchPg.ErrNPCAlreadyInMatch
			},
		}

		uc := matchApp.NewAddMatchNPCUC(matchReader, sheets, roster)
		_, err := uc.Add(ctx, &matchApp.AddMatchNPCInput{
			RequesterUUID: masterUUID, MatchUUID: matchUUID, SheetUUID: sheetUUID,
		})
		if !errors.Is(err, matchApp.ErrNPCAlreadyInMatch) {
			t.Fatalf("expected ErrNPCAlreadyInMatch, got: %v", err)
		}
	})

	t.Run("roster rejects the sheet as not npc-eligible", func(t *testing.T) {
		matchReader := &mockMatchReader{
			GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
				return &matchEntity.Match{UUID: matchUUID, MasterUUID: masterUUID, CampaignUUID: campaignUUID}, nil
			},
		}
		sheets := &mockSheetOwnershipReader{
			GetCharacterSheetRelationshipUUIDsFn: func(ctx context.Context, id uuid.UUID) (csEntity.RelationshipUUIDs, error) {
				return csEntity.RelationshipUUIDs{MasterUUID: &masterUUID}, nil
			},
		}
		roster := &mockNPCRoster{
			AddNPCParticipantFn: func(
				ctx context.Context, mUUID, sUUID uuid.UUID, joinedAt time.Time,
			) (*matchEntity.Participant, error) {
				// Double-wrapped, mirroring the real gateway
				// (internal/gateway/pg/match/add_npc_participant.go:66:
				// fmt.Errorf("%w: %w", ErrSheetNotEligibleNPC, pgx.ErrNoRows)).
				// A bare sentinel here would let a regression from errors.Is
				// to == in production pass this test unnoticed.
				return nil, fmt.Errorf("%w: %w", matchPg.ErrSheetNotEligibleNPC, errors.New("no rows"))
			},
		}

		uc := matchApp.NewAddMatchNPCUC(matchReader, sheets, roster)
		_, err := uc.Add(ctx, &matchApp.AddMatchNPCInput{
			RequesterUUID: masterUUID, MatchUUID: matchUUID, SheetUUID: sheetUUID,
		})
		if !errors.Is(err, matchApp.ErrSheetNotNPC) {
			t.Fatalf("expected ErrSheetNotNPC, got: %v", err)
		}
	})
}
