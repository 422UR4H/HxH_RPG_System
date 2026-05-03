package enrollment

import (
	"context"
	"fmt"

	enrollmentEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enrollment"
	"github.com/google/uuid"
)

func (r *Repository) ListByMatchUUID(
	ctx context.Context, matchUUID uuid.UUID,
) ([]*enrollmentEntity.Enrollment, error) {
	const query = `
		SELECT
			e.uuid, e.status, e.created_at,
			cs.id, cs.uuid, cs.player_uuid, cs.master_uuid, cs.campaign_uuid,
			cs.category_name, cs.curr_hex_value,
			COALESCE(cs.level, 0), COALESCE(cs.points, 0),
			COALESCE(cs.talent_lvl, 0), COALESCE(cs.skills_lvl, 0),
			COALESCE(cs.health_min_pts, 0), COALESCE(cs.health_curr_pts, 0), COALESCE(cs.health_max_pts, 0),
			COALESCE(cs.stamina_min_pts, 0), COALESCE(cs.stamina_curr_pts, 0), COALESCE(cs.stamina_max_pts, 0),
			COALESCE(cs.physicals_lvl, 0), COALESCE(cs.mentals_lvl, 0), COALESCE(cs.spirituals_lvl, 0),
			COALESCE(cs.aura_min_pts, 0), COALESCE(cs.aura_curr_pts, 0), COALESCE(cs.aura_max_pts, 0),
			cs.created_at, cs.updated_at,
			cp.nickname, cp.fullname, cp.alignment, cp.character_class, cp.birthday,
			u.uuid, u.nick
		FROM enrollments e
		JOIN character_sheets cs   ON cs.uuid = e.character_sheet_uuid
		JOIN character_profiles cp ON cp.character_sheet_uuid = cs.uuid
		JOIN users u               ON u.uuid = cs.player_uuid
		WHERE e.match_uuid = $1
		ORDER BY e.created_at ASC
	`

	rows, err := r.q.Query(ctx, query, matchUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to query enrollments by match: %w", err)
	}
	defer rows.Close()

	out := make([]*enrollmentEntity.Enrollment, 0)
	for rows.Next() {
		var e enrollmentEntity.Enrollment
		var s = &e.CharacterSheet
		err := rows.Scan(
			&e.UUID, &e.Status, &e.CreatedAt,
			&s.ID, &s.UUID, &s.PlayerUUID, &s.MasterUUID, &s.CampaignUUID,
			&s.CategoryName, &s.CurrHexValue,
			&s.Level, &s.Points, &s.TalentLvl, &s.SkillsLvl,
			&s.Health.Min, &s.Health.Curr, &s.Health.Max,
			&s.Stamina.Min, &s.Stamina.Curr, &s.Stamina.Max,
			&s.PhysicalsLvl, &s.MentalsLvl, &s.SpiritualsLvl,
			&s.Aura.Min, &s.Aura.Curr, &s.Aura.Max,
			&s.CreatedAt, &s.UpdatedAt,
			&s.NickName, &s.FullName, &s.Alignment, &s.CharacterClass, &s.Birthday,
			&e.Player.UUID, &e.Player.Nick,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan enrollment row: %w", err)
		}
		out = append(out, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	return out, nil
}
