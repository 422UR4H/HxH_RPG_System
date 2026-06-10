package entity

type WallType string

const (
	WallTypeWall       WallType = "wall"
	WallTypeDoor       WallType = "door"
	WallTypeWindow     WallType = "window"
	WallTypeSecretDoor WallType = "secret_door"
	WallTypeTerrain    WallType = "terrain"
)

type WallMaterial string

const (
	WallMaterialStone   WallMaterial = "stone"
	WallMaterialWood    WallMaterial = "wood"
	WallMaterialIron    WallMaterial = "iron"
	WallMaterialMagical WallMaterial = "magical"
)

type DoorSubtype string

const (
	DoorSubtypeBasic      DoorSubtype = "basic"
	DoorSubtypeDouble     DoorSubtype = "double"
	DoorSubtypePortcullis DoorSubtype = "portcullis"
	DoorSubtypeDrawbridge DoorSubtype = "drawbridge"
)

type WindowSubtype string

const (
	WindowSubtypeBasic     WindowSubtype = "basic"
	WindowSubtypeBarred    WindowSubtype = "barred"
	WindowSubtypeShuttered WindowSubtype = "shuttered"
)

type SenseKind string

const (
	SenseFull  SenseKind = "full"
	SenseSight SenseKind = "sight"
	SenseNone  SenseKind = "none"
)

type WallDirection string

const (
	WallDirectionBoth  WallDirection = "both"
	WallDirectionLeft  WallDirection = "left"
	WallDirectionRight WallDirection = "right"
)

type WallSegment struct {
	ID            string         `json:"id"`
	P1            [2]float64     `json:"p1"`
	P2            [2]float64     `json:"p2"`
	WallType      WallType       `json:"wall_type"`
	Material      WallMaterial   `json:"material"`
	DoorSubtype   *DoorSubtype   `json:"door_subtype,omitempty"`
	WindowSubtype *WindowSubtype `json:"window_subtype,omitempty"`
	Move          bool           `json:"move"`
	Sense         SenseKind      `json:"sense"`
	Direction     WallDirection  `json:"direction"`
	Open          bool           `json:"open"`
	Locked        bool           `json:"locked"`
	HP            int            `json:"hp"`
	MaxHP         int            `json:"max_hp"`
	Resistance    int            `json:"resistance"`
	Destroyed     bool           `json:"destroyed"`
}
