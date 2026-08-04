package service

import (
	"testing"

	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
)

func TestMaskSecretDoor_LooksLikeWallKeepsCombatFields(t *testing.T) {
	sub := mapentity.DoorSubtypeBasic
	sd := mapentity.WallSegment{
		ID: "w1", WallType: mapentity.WallTypeSecretDoor, Material: mapentity.WallMaterialStone,
		DoorSubtype: &sub, Open: true, Locked: true, HP: 80, MaxHP: 100, Resistance: 5,
	}
	m := MaskSecretDoorForPlayer(sd)
	if m.WallType != mapentity.WallTypeWall {
		t.Fatal("masked type must be wall")
	}
	if m.DoorSubtype != nil || m.Open || m.Locked {
		t.Fatal("masked door must not leak subtype/open/locked")
	}
	if m.ID != "w1" || m.HP != 80 || m.MaxHP != 100 || m.Resistance != 5 || m.Material != mapentity.WallMaterialStone {
		t.Fatal("masked wall must keep id/material/hp for combat parity")
	}
}
