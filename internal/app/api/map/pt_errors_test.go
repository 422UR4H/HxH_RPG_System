package mapapi_test

import (
	"context"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	mapapi "github.com/422UR4H/HxH_RPG_System/internal/app/api/map"
	mapuc "github.com/422UR4H/HxH_RPG_System/internal/application/map"
	"github.com/google/uuid"
)

type deleteMapStub struct{ err error }

func (s deleteMapStub) DeleteMap(context.Context, uuid.UUID, uuid.UUID) error { return s.err }

func TestDeleteMapHandler_NotFoundReadsInPortuguese(t *testing.T) {
	ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())

	_, err := mapapi.DeleteMapHandler(deleteMapStub{err: mapuc.ErrMapNotFound})(ctx, &mapapi.DeleteMapRequest{MapID: uuid.New()})
	if err == nil || err.Error() != "Mapa não encontrado." {
		t.Errorf("err = %v", err)
	}
}
