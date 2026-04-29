package submission_test

import (
	"context"

	"github.com/google/uuid"
)

type mockSubmitCharacterSheet struct {
	fn func(ctx context.Context, userUUID, sheetUUID, campaignUUID uuid.UUID) error
}

func (m *mockSubmitCharacterSheet) Submit(ctx context.Context, userUUID, sheetUUID, campaignUUID uuid.UUID) error {
	return m.fn(ctx, userUUID, sheetUUID, campaignUUID)
}

type mockAcceptCharacterSheetSubmission struct {
	fn func(ctx context.Context, sheetUUID, masterUUID uuid.UUID) error
}

func (m *mockAcceptCharacterSheetSubmission) Accept(ctx context.Context, sheetUUID, masterUUID uuid.UUID) error {
	return m.fn(ctx, sheetUUID, masterUUID)
}

type mockRejectCharacterSheetSubmission struct {
	fn func(ctx context.Context, sheetUUID, masterUUID uuid.UUID) error
}

func (m *mockRejectCharacterSheetSubmission) Reject(ctx context.Context, sheetUUID, masterUUID uuid.UUID) error {
	return m.fn(ctx, sheetUUID, masterUUID)
}
