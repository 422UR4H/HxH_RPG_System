// internal/app/api/matchmap/response.go
package matchmapapi

import (
	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/matchmap/entity"
)

type MatchMapResponse struct {
	MatchUUID  string `json:"match_uuid"`
	MapUUID    string `json:"map_uuid"`
	AttachedAt string `json:"attached_at"`
}

func toMatchMapResponse(mm *entity.MatchMap) MatchMapResponse {
	return MatchMapResponse{
		MatchUUID:  mm.MatchUUID,
		MapUUID:    mm.MapUUID,
		AttachedAt: mm.AttachedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
