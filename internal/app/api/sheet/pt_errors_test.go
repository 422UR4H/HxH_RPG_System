package sheet_test

import (
	"context"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/sheet"
	"github.com/google/uuid"
)

func TestDeleteCharacterSheetHandler_InvalidUUIDReadsInPortuguese(t *testing.T) {
	ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())

	_, err := sheet.DeleteCharacterSheetHandler(nil)(ctx, &sheet.DeleteCharacterSheetRequest{UUID: "x"})
	if err == nil || err.Error() != "Ficha inválida." {
		t.Errorf("err = %v", err)
	}
}
