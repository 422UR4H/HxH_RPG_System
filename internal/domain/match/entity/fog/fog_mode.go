package fog

type FogMode string

const (
	FogModeLive     FogMode = "live"
	FogModeExplored FogMode = "explored"
)

// IsValid reports whether m is a known fog mode.
func (m FogMode) IsValid() bool {
	return m == FogModeLive || m == FogModeExplored
}
