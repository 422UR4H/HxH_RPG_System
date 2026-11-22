package enrollment_test

import (
	"context"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/enrollment"
	"github.com/google/uuid"
)

func TestAcceptEnrollmentHandler_InvalidUUIDReadsInPortuguese(t *testing.T) {
	ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())

	_, err := enrollment.AcceptEnrollmentHandler(nil)(ctx, &enrollment.AcceptEnrollmentRequest{EnrollmentUUID: "x"})
	if err == nil || err.Error() != "Inscrição inválida." {
		t.Errorf("err = %v", err)
	}
}
