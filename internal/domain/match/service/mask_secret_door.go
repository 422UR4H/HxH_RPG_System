package service

import mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"

// MaskSecretDoorForPlayer returns a copy of a secret door that looks like a plain wall,
// preserving all combat-relevant fields (id, material, hp, resistance, move, sense, direction).
func MaskSecretDoorForPlayer(w mapentity.WallSegment) mapentity.WallSegment {
	masked := w
	masked.WallType = mapentity.WallTypeWall
	masked.DoorSubtype = nil
	masked.WindowSubtype = nil
	masked.Open = false
	masked.Locked = false
	return masked
}
