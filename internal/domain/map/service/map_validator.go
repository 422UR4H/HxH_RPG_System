package service

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"
	"unicode/utf8"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
)

// wallBoundsTolerance absorve ruído de ponto flutuante: o front calcula os
// vértices do hex como centro + cellSize*cos(ângulo) e a borda esquerda sai
// 1 ulp além de minX. Em unidades LOCAIS: muito acima do ruído de float64
// (~1e-14) e muito abaixo de um pixel.
const wallBoundsTolerance = 1e-6

var (
	ErrNameTooShort      = errtext.Sentence("O nome do mapa deve ter pelo menos 3 caracteres.")
	ErrNameHasEmoji      = errtext.Sentence("O nome do mapa não pode ter emoji.")
	ErrInvalidCellSize   = errtext.Sentence("O tamanho da célula deve ser maior que zero.")
	ErrInvalidCols       = errtext.Sentence("O mapa precisa de pelo menos uma coluna.")
	ErrInvalidRows       = errtext.Sentence("O mapa precisa de pelo menos uma linha.")
	ErrInvalidSkewRatio  = errtext.Sentence("A inclinação do mapa deve ficar entre 0 e 1.")
	ErrWallSameEndpoints = errtext.Sentence("Uma parede precisa de dois pontos diferentes.")
	ErrWallInvalidType   = errtext.Sentence("Tipo de parede inválido.")
	ErrWallNegativeHP    = errtext.Sentence("A vida da parede não pode ser negativa.")
	ErrWallOutOfBounds   = errtext.Sentence("Uma parede está fora da grade do mapa.")
	ErrInvalidFogMode    = errtext.Sentence("Modo de névoa inválido: escolha ao vivo ou explorado.")
)

func ValidateMap(name string, grid entity.GridShape) error {
	if utf8.RuneCountInString(name) < 3 {
		return ErrNameTooShort
	}
	if domain.ContainsEmoji(name) {
		return ErrNameHasEmoji
	}
	if grid.CellSize <= 0 {
		return ErrInvalidCellSize
	}
	if grid.Cols <= 0 {
		return ErrInvalidCols
	}
	if grid.Rows <= 0 {
		return ErrInvalidRows
	}
	if grid.SkewRatio < 0 || grid.SkewRatio > 1 {
		return ErrInvalidSkewRatio
	}
	return nil
}

// ValidateFogMode rejects a non-empty fog mode that is not a known value.
// An empty fog mode is allowed (the mapper defaults it to FogModeExplored on read).
func ValidateFogMode(m fog.FogMode) error {
	if m != "" && !m.IsValid() {
		return ErrInvalidFogMode
	}
	return nil
}

func ValidateWallSegments(walls []entity.WallSegment, grid entity.GridShape) error {
	minX, minY, maxX, maxY := GridLocalBounds(grid)
	for _, w := range walls {
		if w.P1 == w.P2 {
			return ErrWallSameEndpoints
		}
		switch w.WallType {
		case entity.WallTypeWall, entity.WallTypeDoor, entity.WallTypeWindow,
			entity.WallTypeSecretDoor, entity.WallTypeTerrain:
		default:
			return ErrWallInvalidType
		}
		if w.HP < 0 {
			return ErrWallNegativeHP
		}
		for _, p := range [][2]float64{w.P1, w.P2} {
			if p[0] < minX-wallBoundsTolerance || p[0] > maxX+wallBoundsTolerance ||
				p[1] < minY-wallBoundsTolerance || p[1] > maxY+wallBoundsTolerance {
				return ErrWallOutOfBounds
			}
		}
	}
	return nil
}
