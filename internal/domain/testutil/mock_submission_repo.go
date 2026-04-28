package testutil

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type MockSubmissionRepo struct {
	SubmitCharacterSheetFn                 func(ctx context.Context, sheetUUID uuid.UUID, campaignUUID uuid.UUID, createdAt time.Time) error
	ExistsSubmittedCharacterSheetFn        func(ctx context.Context, uuid uuid.UUID) (bool, error)
	AcceptCharacterSheetSubmissionFn       func(ctx context.Context, sheetUUID uuid.UUID, campaignUUID uuid.UUID) error
	GetSubmissionCampaignUUIDBySheetUUIDFn func(ctx context.Context, sheetUUID uuid.UUID) (uuid.UUID, error)
	RejectCharacterSheetSubmissionFn       func(ctx context.Context, sheetUUID uuid.UUID) error
}

func (m *MockSubmissionRepo) SubmitCharacterSheet(ctx context.Context, sheetUUID uuid.UUID, campaignUUID uuid.UUID, createdAt time.Time) error {
	if m.SubmitCharacterSheetFn != nil {
		return m.SubmitCharacterSheetFn(ctx, sheetUUID, campaignUUID, createdAt)
	}
	return nil
}

func (m *MockSubmissionRepo) ExistsSubmittedCharacterSheet(ctx context.Context, id uuid.UUID) (bool, error) {
	if m.ExistsSubmittedCharacterSheetFn != nil {
		return m.ExistsSubmittedCharacterSheetFn(ctx, id)
	}
	return false, nil
}

func (m *MockSubmissionRepo) AcceptCharacterSheetSubmission(ctx context.Context, sheetUUID uuid.UUID, campaignUUID uuid.UUID) error {
	if m.AcceptCharacterSheetSubmissionFn != nil {
		return m.AcceptCharacterSheetSubmissionFn(ctx, sheetUUID, campaignUUID)
	}
	return nil
}

func (m *MockSubmissionRepo) GetSubmissionCampaignUUIDBySheetUUID(ctx context.Context, sheetUUID uuid.UUID) (uuid.UUID, error) {
	if m.GetSubmissionCampaignUUIDBySheetUUIDFn != nil {
		return m.GetSubmissionCampaignUUIDBySheetUUIDFn(ctx, sheetUUID)
	}
	return uuid.Nil, nil
}

func (m *MockSubmissionRepo) RejectCharacterSheetSubmission(ctx context.Context, sheetUUID uuid.UUID) error {
	if m.RejectCharacterSheetSubmissionFn != nil {
		return m.RejectCharacterSheetSubmissionFn(ctx, sheetUUID)
	}
	return nil
}
