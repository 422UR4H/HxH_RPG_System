package testutil

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type MockSheetBirthdayReader struct {
	GetCharacterSheetBirthInfoFn func(ctx context.Context, sheetUUID uuid.UUID) (time.Time, int, error)
	GetCharacterSheetNickFn      func(ctx context.Context, sheetUUID uuid.UUID) (string, error)
}

func (m *MockSheetBirthdayReader) GetCharacterSheetBirthInfo(
	ctx context.Context, sheetUUID uuid.UUID,
) (time.Time, int, error) {
	if m.GetCharacterSheetBirthInfoFn != nil {
		return m.GetCharacterSheetBirthInfoFn(ctx, sheetUUID)
	}
	return time.Time{}, 0, nil
}

func (m *MockSheetBirthdayReader) GetCharacterSheetNick(
	ctx context.Context, sheetUUID uuid.UUID,
) (string, error) {
	if m.GetCharacterSheetNickFn != nil {
		return m.GetCharacterSheetNickFn(ctx, sheetUUID)
	}
	return "", nil
}
