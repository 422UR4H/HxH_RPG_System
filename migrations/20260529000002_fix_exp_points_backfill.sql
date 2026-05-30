-- +goose Up
-- +goose StatementBegin
-- Corrective backfill: each exp column contributes to CharacterExp.exp.points
-- according to the cascade multiplier defined by the domain's wrap() function.
--
-- Mental attrs (PrimaryAttribute → MentalsAbility → CharacterExp): ×1
-- Physical skills via PrimaryAttribute (attr→physAbility + physSkills→skillsAbility): ×2
--   Resistance-based: vitality, energy, defense
--   Agility-based:    velocity, accelerate, brake
--   Flexibility-based: acrobatics, evasion, sneak
--   Sense-based:      vision, hearing, smell, tact, taste
-- Physical skills via MiddleAttribute (lenAttrs=2):
--   MiddleAttribute divides exp by 2 before propagating to each of its 2 primary attrs.
--   Per group: (sum/2)*2 via attrs + sum/2 via physSkills = floor(groupSum/2)*3
--   Strength-based (Resistance+Agility):   push, grab, carry
--   Celerity-based (Agility+Flexibility):  legerity, repel, feint
--   Dexterity-based (Flexibility+Sense):   reflex, accuracy, stealth
--   Constitution-based (Sense+Resistance): heal, breath, tenacity
-- Spiritual principles (conscienceNen→spiritualsAbility→CharacterExp): ×1
-- Spiritual categories (hatsu→conscienceNen→spiritualsAbility→CharacterExp): ×1
-- Proficiencies (physAbility→CharacterExp): ×1
-- Excluded: talent_exp (separate system), nen/focus/will_power/aura_control/aop (not applied in wrap).
UPDATE character_sheets SET exp_points = (
    -- Mental attributes (×1)
    COALESCE(resilience_exp,     0) + COALESCE(adaptability_exp,    0) +
    COALESCE(weighting_exp,      0) + COALESCE(creativity_exp,      0) +

    -- Physical skills via PrimaryAttribute (×2)
    (COALESCE(vitality_exp,  0) + COALESCE(energy_exp,      0) + COALESCE(defense_exp,  0)) * 2 +
    (COALESCE(velocity_exp,  0) + COALESCE(accelerate_exp,  0) + COALESCE(brake_exp,    0)) * 2 +
    (COALESCE(acrobatics_exp,0) + COALESCE(evasion_exp,     0) + COALESCE(sneak_exp,    0)) * 2 +
    (COALESCE(vision_exp,    0) + COALESCE(hearing_exp,     0) + COALESCE(smell_exp,    0) +
     COALESCE(tact_exp,      0) + COALESCE(taste_exp,       0)) * 2 +

    -- Physical skills via MiddleAttribute (floor(groupSum/2)*3)
    ((COALESCE(push_exp,     0) + COALESCE(grab_exp,        0) + COALESCE(carry_exp,    0)) / 2) * 3 +
    ((COALESCE(legerity_exp, 0) + COALESCE(repel_exp,       0) + COALESCE(feint_exp,    0)) / 2) * 3 +
    ((COALESCE(reflex_exp,   0) + COALESCE(accuracy_exp,    0) + COALESCE(stealth_exp,  0)) / 2) * 3 +
    ((COALESCE(heal_exp,     0) + COALESCE(breath_exp,      0) + COALESCE(tenacity_exp, 0)) / 2) * 3 +

    -- Spiritual principles (×1)
    COALESCE(ten_exp,            0) + COALESCE(zetsu_exp,           0) +
    COALESCE(ren_exp,            0) + COALESCE(gyo_exp,             0) +
    COALESCE(shu_exp,            0) + COALESCE(kou_exp,             0) +
    COALESCE(ken_exp,            0) + COALESCE(ryu_exp,             0) +
    COALESCE(in_exp,             0) + COALESCE(en_exp,              0) +

    -- Spiritual categories (×1)
    COALESCE(reinforcement_exp,  0) + COALESCE(transmutation_exp,   0) +
    COALESCE(materialization_exp,0) + COALESCE(specialization_exp,  0) +
    COALESCE(manipulation_exp,   0) + COALESCE(emission_exp,        0) +

    -- Proficiencies (×1)
    COALESCE(
        (SELECT SUM(p.exp) FROM proficiencies p
         WHERE p.character_sheet_uuid = character_sheets.uuid),
        0
    )
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE character_sheets SET exp_points = 0;
-- +goose StatementEnd
