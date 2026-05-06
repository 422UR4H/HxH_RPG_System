package match

import (
	"context"
	"fmt"

	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	"github.com/google/uuid"
)

func (r *Repository) ListParticipantsByMatchUUID(
	ctx context.Context, matchUUID uuid.UUID,
) ([]*matchEntity.Participant, error) {
	const query = `
		SELECT
			mp.uuid, mp.match_uuid,
			mp.joined_at, mp.left_at,
			mp.created_at, mp.updated_at,
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
			cs.story_start_at, cs.story_current_at, cs.dead_at
		FROM match_participants mp
		JOIN character_sheets cs   ON cs.uuid = mp.character_sheet_uuid
		JOIN character_profiles cp ON cp.character_sheet_uuid = cs.uuid
		LEFT JOIN users u          ON u.uuid = cs.player_uuid
		WHERE mp.match_uuid = $1
		ORDER BY mp.joined_at ASC
	`

	rows, err := r.q.Query(ctx, query, matchUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to query participants: %w", err)
	}
	defer rows.Close()

	out := make([]*matchEntity.Participant, 0)
	for rows.Next() {
		var p matchEntity.Participant
		s := &p.Sheet
		err := rows.Scan(
			&p.UUID, &p.MatchUUID,
			&p.JoinedAt, &p.LeftAt,
			&p.CreatedAt, &p.UpdatedAt,
			&s.ID, &s.UUID, &s.PlayerUUID, &s.MasterUUID, &s.CampaignUUID,
			&s.CategoryName, &s.CurrHexValue,
			&s.Level, &s.Points, &s.TalentLvl, &s.SkillsLvl,
			&s.Health.Min, &s.Health.Curr, &s.Health.Max,
			&s.Stamina.Min, &s.Stamina.Curr, &s.Stamina.Max,
			&s.PhysicalsLvl, &s.MentalsLvl, &s.SpiritualsLvl,
			&s.Aura.Min, &s.Aura.Curr, &s.Aura.Max,
			&s.CreatedAt, &s.UpdatedAt,
			&s.NickName, &s.FullName, &s.Alignment, &s.CharacterClass, &s.Birthday,
			&s.StoryStartAt, &s.StoryCurrentAt, &s.DeadAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan participant row: %w", err)
		}
		out = append(out, &p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	return out, nil
}
