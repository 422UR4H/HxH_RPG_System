package matchmapapi_test

import (
	"context"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	matchmapapi "github.com/422UR4H/HxH_RPG_System/internal/app/api/matchmap"
	matchmapuc "github.com/422UR4H/HxH_RPG_System/internal/application/matchmap"
	"github.com/google/uuid"
)

func TestDetachMatchMapHandler_ForbiddenReadsInPortuguese(t *testing.T) {
	ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())
	mock := &mockDetachMatchMap{fn: func(context.Context, *matchmapuc.DetachMatchMapInput) error {
		return matchmapuc.ErrNotMatchMaster
	}}

	_, err := matchmapapi.DetachMatchMapHandler(mock)(ctx, &matchmapapi.DetachMatchMapRequest{MatchUUID: uuid.New()})
	if err == nil || err.Error() != "Só o mestre da partida pode fazer isso." {
		t.Errorf("err = %v", err)
	}
}
