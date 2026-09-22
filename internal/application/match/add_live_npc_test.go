package match_test

import (
	"context"
	"errors"
	"testing"

	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/application/character_sheet"
	matchApp "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/google/uuid"
)

// mockAddMatchNPC is local to this test package on purpose — same rationale as
// mockMatchReader in add_match_npc_test.go: AddLiveNPCUC only needs the Add method,
// so it depends on the narrow IAddMatchNPC seam instead of the concrete AddMatchNPCUC.
type mockAddMatchNPC struct {
	AddFn func(ctx context.Context, input *matchApp.AddMatchNPCInput) (*matchEntity.Participant, error)
}

func (m *mockAddMatchNPC) Add(
	ctx context.Context, input *matchApp.AddMatchNPCInput,
) (*matchEntity.Participant, error) {
	return m.AddFn(ctx, input)
}

// mockCharSheetLoader tracks whether it was called and with which UUID, so tests can
// assert the loader is skipped when the roster rejects the request outright.
type mockCharSheetLoader struct {
	GetCharacterSheetByUUIDFn func(ctx context.Context, uuid string) (*csSheet.CharacterSheet, bool, error)
	called                    bool
	gotUUID                   string
}

func (m *mockCharSheetLoader) GetCharacterSheetByUUID(
	ctx context.Context, id string,
) (*csSheet.CharacterSheet, bool, error) {
	m.called = true
	m.gotUUID = id
	return m.GetCharacterSheetByUUIDFn(ctx, id)
}

func TestAddLiveNPC(t *testing.T) {
	ctx := context.Background()

	masterUUID := uuid.New()
	matchUUID := uuid.New()
	sheetUUID := uuid.New()

	t.Run("happy path: added then loaded", func(t *testing.T) {
		wantSheet := &csSheet.CharacterSheet{UUID: sheetUUID}
		roster := &mockAddMatchNPC{
			AddFn: func(ctx context.Context, input *matchApp.AddMatchNPCInput) (*matchEntity.Participant, error) {
				return &matchEntity.Participant{UUID: uuid.New(), MatchUUID: matchUUID}, nil
			},
		}
		loader := &mockCharSheetLoader{
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*csSheet.CharacterSheet, bool, error) {
				return wantSheet, false, nil
			},
		}

		uc := matchApp.NewAddLiveNPCUC(roster, loader)
		got, err := uc.Execute(ctx, &matchApp.AddMatchNPCInput{
			RequesterUUID: masterUUID, MatchUUID: matchUUID, SheetUUID: sheetUUID,
		})
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if got != wantSheet {
			t.Fatalf("expected returned sheet %+v, got %+v", wantSheet, got)
		}
		if !loader.called {
			t.Fatal("expected sheet loader to be called")
		}
		if loader.gotUUID != sheetUUID.String() {
			t.Fatalf("expected loader called with %s, got %s", sheetUUID, loader.gotUUID)
		}
	})

	t.Run("npc already in match still loads and returns the sheet (D2)", func(t *testing.T) {
		wantSheet := &csSheet.CharacterSheet{UUID: sheetUUID}
		roster := &mockAddMatchNPC{
			AddFn: func(ctx context.Context, input *matchApp.AddMatchNPCInput) (*matchEntity.Participant, error) {
				return nil, matchApp.ErrNPCAlreadyInMatch
			},
		}
		loader := &mockCharSheetLoader{
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*csSheet.CharacterSheet, bool, error) {
				return wantSheet, false, nil
			},
		}

		uc := matchApp.NewAddLiveNPCUC(roster, loader)
		got, err := uc.Execute(ctx, &matchApp.AddMatchNPCInput{
			RequesterUUID: masterUUID, MatchUUID: matchUUID, SheetUUID: sheetUUID,
		})
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if got != wantSheet {
			t.Fatalf("expected returned sheet %+v, got %+v", wantSheet, got)
		}
		if !loader.called {
			t.Fatal("expected sheet loader to still be called when npc is already in match")
		}
	})

	t.Run("roster errors are returned and the loader is skipped", func(t *testing.T) {
		rosterErrs := []error{
			matchApp.ErrNotMatchMaster,
			matchApp.ErrSheetNotNPC,
			matchApp.ErrMatchAlreadyFinished,
		}
		for _, wantErr := range rosterErrs {
			t.Run(wantErr.Error(), func(t *testing.T) {
				roster := &mockAddMatchNPC{
					AddFn: func(ctx context.Context, input *matchApp.AddMatchNPCInput) (*matchEntity.Participant, error) {
						return nil, wantErr
					},
				}
				loader := &mockCharSheetLoader{}

				uc := matchApp.NewAddLiveNPCUC(roster, loader)
				_, err := uc.Execute(ctx, &matchApp.AddMatchNPCInput{
					RequesterUUID: masterUUID, MatchUUID: matchUUID, SheetUUID: sheetUUID,
				})
				if !errors.Is(err, wantErr) {
					t.Fatalf("expected %v, got: %v", wantErr, err)
				}
				if loader.called {
					t.Fatal("expected sheet loader not to be called")
				}
			})
		}
	})

	t.Run("loader character sheet not found maps to match package error", func(t *testing.T) {
		roster := &mockAddMatchNPC{
			AddFn: func(ctx context.Context, input *matchApp.AddMatchNPCInput) (*matchEntity.Participant, error) {
				return &matchEntity.Participant{UUID: uuid.New(), MatchUUID: matchUUID}, nil
			},
		}
		loader := &mockCharSheetLoader{
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*csSheet.CharacterSheet, bool, error) {
				return nil, false, charactersheet.ErrCharacterSheetNotFound
			},
		}

		uc := matchApp.NewAddLiveNPCUC(roster, loader)
		_, err := uc.Execute(ctx, &matchApp.AddMatchNPCInput{
			RequesterUUID: masterUUID, MatchUUID: matchUUID, SheetUUID: sheetUUID,
		})
		if !errors.Is(err, matchApp.ErrCharacterSheetNotFound) {
			t.Fatalf("expected ErrCharacterSheetNotFound, got: %v", err)
		}
	})

	t.Run("loader generic error is propagated", func(t *testing.T) {
		genericErr := errors.New("db exploded")
		roster := &mockAddMatchNPC{
			AddFn: func(ctx context.Context, input *matchApp.AddMatchNPCInput) (*matchEntity.Participant, error) {
				return &matchEntity.Participant{UUID: uuid.New(), MatchUUID: matchUUID}, nil
			},
		}
		loader := &mockCharSheetLoader{
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*csSheet.CharacterSheet, bool, error) {
				return nil, false, genericErr
			},
		}

		uc := matchApp.NewAddLiveNPCUC(roster, loader)
		_, err := uc.Execute(ctx, &matchApp.AddMatchNPCInput{
			RequesterUUID: masterUUID, MatchUUID: matchUUID, SheetUUID: sheetUUID,
		})
		if !errors.Is(err, genericErr) {
			t.Fatalf("expected genericErr, got: %v", err)
		}
	})
}
