-- +goose Up
-- +goose StatementBegin
BEGIN;

CREATE TABLE IF NOT EXISTS character_sheets (
  id SERIAL PRIMARY KEY,
  uuid UUID NOT NULL DEFAULT gen_random_uuid(),

  category_name VARCHAR(16) NOT NULL,
  curr_hex_value INT,
  talent_exp INT,

  -- Levels
  level INT,
  points INT,
  talent_lvl INT,
  physicals_lvl INT,
  mentals_lvl INT,
  spirituals_lvl INT,
  skills_lvl INT,

  -- Status
  health_min_pts INT DEFAULT 0,
  health_curr_pts INT,
  health_max_pts INT,

  stamina_min_pts INT DEFAULT 0,
  stamina_curr_pts INT,
  stamina_max_pts INT,

  aura_min_pts INT DEFAULT 0,
  aura_curr_pts INT,
  aura_max_pts INT,

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

  story_start_at DATE,
  story_current_at DATE,
  dead_at TIMESTAMP,

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
