-- +goose Up
-- +goose StatementBegin
BEGIN;

CREATE TABLE IF NOT EXISTS character_sheets (
  id SERIAL PRIMARY KEY,
  uuid UUID NOT NULL DEFAULT gen_random_uuid(),

  curr_hex_value INT,
  talent_exp INT,

  -- Physical Attributes
  resistance_pts INT,
  strength_pts INT,
  agility_pts INT,
  action_speed_pts INT,
  flexibility_pts INT,
  dexterity_pts INT,
  sense_pts INT,
  constitution_pts INT,

  -- Mental Attributes
  resilience_pts INT,
  adaptability_pts INT,
  weighting_pts INT,
  creativity_pts INT,
  resilience_exp INT,
  adaptability_exp INT,
  weighting_exp INT,
  creativity_exp INT,

  -- Physical Skills
  -- resistance
  vitality_exp INT,
  energy_exp INT,
  defense_exp INT,
  -- strength
  push_exp INT,
  grab_exp INT,
  carry_capacity_exp INT,
  -- agility
  velocity_exp INT,
  accelerate_exp INT,
  brake_exp INT,
  -- action speed
  attack_speed_exp INT,
  repel_exp INT,
  feint_exp INT,
  -- flexibility
  acrobatics_exp INT,
  evasion_exp INT,
  sneak_exp INT,
  -- dexterity
  reflex_exp INT,
  accuracy_exp INT,
  stealth_exp INT,
  -- sense
  vision_exp INT,
  hearing_exp INT,
  smell_exp INT,
  tact_exp INT,
  taste_exp INT,
  -- constitution
  heal_exp INT,
  breath_exp INT,
  tenacity_exp INT,

  -- Spirituals
  nen_exp INT,
  focus_exp INT,
  will_power_exp INT,

  -- Nen Principles
  ten_exp INT,
  zetsu_exp INT,
  ren_exp INT,
  gyo_exp INT,
  kou_exp INT,
  ken_exp INT,
  ryu_exp INT,
  in_exp INT,
  en_exp INT,
  aura_control_exp INT,
  aop_exp INT,

  -- Nen Categories
  reinforcement_exp INT,
  transmutation_exp INT,
  materialization_exp INT,
  specialization_exp INT,
  manipulation_exp INT,
  emission_exp INT,

  stamina_curr_pts INT,
  health_curr_pts INT,

  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (uuid)
);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

DROP TABLE IF EXISTS character_sheets;

COMMIT;
-- +goose StatementEnd
