package match_test

import (
	"context"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/match"
	matchUC "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/google/uuid"
)

func TestDeleteMatchHandler_ErrorsReadInPortuguese(t *testing.T) {
	ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())

	_, err := match.DeleteMatchHandler(nil)(ctx, &match.DeleteMatchRequest{UUID: "x"})
	if err == nil || err.Error() != "Partida inválida." {
		t.Errorf("invalid uuid: err = %v", err)
	}

	mock := &mockDeleteMatch{fn: func(context.Context, *matchUC.DeleteMatchInput) error { return matchUC.ErrMatchNotFound }}
	_, err = match.DeleteMatchHandler(mock)(ctx, &match.DeleteMatchRequest{UUID: uuid.NewString()})
	if err == nil || err.Error() != "Partida não encontrada." {
		t.Errorf("not found: err = %v", err)
	}
}
