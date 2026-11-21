package submission_test

import (
	"context"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/submission"
	"github.com/google/uuid"
)

func TestAcceptSheetSubmissionHandler_InvalidUUIDReadsInPortuguese(t *testing.T) {
	ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())

	_, err := submission.AcceptSheetSubmissionHandler(nil)(ctx, &submission.AcceptSheetSubmissionRequest{SheetUUID: "x"})
	if err == nil || err.Error() != "Ficha inválida." {
		t.Errorf("err = %v", err)
	}
}
