-- +goose Up
-- +goose StatementBegin
-- exp_points = soma de tudo que cascateia até CharacterExp no wrap():
--   mental attrs, physical skills, spiritual principles, categories, proficiencies.
--   Excluídos: talent_exp (sistema próprio), nen/focus/will_power/aura_control/aop (não aplicados no wrap).
UPDATE character_sheets SET exp_points = (
    COALESCE(resilience_exp,     0) + COALESCE(adaptability_exp,    0) +
    COALESCE(weighting_exp,      0) + COALESCE(creativity_exp,      0) +
    COALESCE(vitality_exp,       0) + COALESCE(energy_exp,          0) +
    COALESCE(defense_exp,        0) + COALESCE(push_exp,            0) +
    COALESCE(grab_exp,           0) + COALESCE(carry_exp,           0) +
    COALESCE(velocity_exp,       0) + COALESCE(accelerate_exp,      0) +
    COALESCE(brake_exp,          0) + COALESCE(legerity_exp,        0) +
    COALESCE(repel_exp,          0) + COALESCE(feint_exp,           0) +
    COALESCE(acrobatics_exp,     0) + COALESCE(evasion_exp,         0) +
    COALESCE(sneak_exp,          0) + COALESCE(reflex_exp,          0) +
    COALESCE(accuracy_exp,       0) + COALESCE(stealth_exp,         0) +
    COALESCE(vision_exp,         0) + COALESCE(hearing_exp,         0) +
    COALESCE(smell_exp,          0) + COALESCE(tact_exp,            0) +
    COALESCE(taste_exp,          0) + COALESCE(heal_exp,            0) +
    COALESCE(breath_exp,         0) + COALESCE(tenacity_exp,        0) +
    COALESCE(ten_exp,            0) + COALESCE(zetsu_exp,           0) +
    COALESCE(ren_exp,            0) + COALESCE(gyo_exp,             0) +
    COALESCE(shu_exp,            0) + COALESCE(kou_exp,             0) +
    COALESCE(ken_exp,            0) + COALESCE(ryu_exp,             0) +
    COALESCE(in_exp,             0) + COALESCE(en_exp,              0) +
    COALESCE(reinforcement_exp,  0) + COALESCE(transmutation_exp,   0) +
    COALESCE(materialization_exp,0) + COALESCE(specialization_exp,  0) +
    COALESCE(manipulation_exp,   0) + COALESCE(emission_exp,        0) +
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
