-- migrations/20261113000000_rename_weapon_enum_values.sql
-- +goose Up
-- +goose StatementBegin

-- Os valores do enum de armas Massa/Longmass/Mangual viraram Mace/Longmace/Flail (typo e mistura de
-- idiomas). O nome da arma é gravado como texto em: proficiencies.weapon, joint_proficiencies.weapons
-- (TEXT[]) e na chave "Weapon" dos JSONB attack/defense/repel de actions e de cancelled_actions.action
-- (documento da ação inteira; mesmas três chaves). Reescreve só esses campos, nunca texto solto.
-- Idempotente: reexecutar não encontra mais o valor antigo.
CREATE OR REPLACE FUNCTION pg_temp.map_weapon_json(doc JSONB, from_a TEXT, to_a TEXT) RETURNS JSONB AS $f$
  SELECT CASE
    WHEN doc IS NOT NULL AND jsonb_typeof(doc) = 'object' AND doc->>'Weapon' = from_a
      THEN jsonb_set(doc, '{Weapon}', to_jsonb(to_a))
    ELSE doc
  END
$f$ LANGUAGE sql IMMUTABLE;

CREATE OR REPLACE FUNCTION pg_temp.map_weapon_action(doc JSONB, from_a TEXT, to_a TEXT) RETURNS JSONB AS $f$
  SELECT CASE WHEN jsonb_typeof(doc) = 'object' THEN
    (SELECT COALESCE(jsonb_object_agg(k,
        CASE WHEN k IN ('attack', 'defense', 'repel') THEN pg_temp.map_weapon_json(v, from_a, to_a) ELSE v END), '{}'::jsonb)
     FROM jsonb_each(doc) AS e(k, v))
  ELSE doc END
$f$ LANGUAGE sql IMMUTABLE;

UPDATE proficiencies SET weapon = 'Mace' WHERE weapon = 'Massa';
UPDATE proficiencies SET weapon = 'Longmace' WHERE weapon = 'Longmass';
UPDATE proficiencies SET weapon = 'Flail' WHERE weapon = 'Mangual';

UPDATE joint_proficiencies
SET weapons = array_replace(array_replace(array_replace(weapons, 'Massa', 'Mace'), 'Longmass', 'Longmace'), 'Mangual', 'Flail')
WHERE weapons && ARRAY['Massa', 'Longmass', 'Mangual'];

UPDATE actions SET
  attack  = pg_temp.map_weapon_json(pg_temp.map_weapon_json(pg_temp.map_weapon_json(attack,  'Massa', 'Mace'), 'Longmass', 'Longmace'), 'Mangual', 'Flail'),
  defense = pg_temp.map_weapon_json(pg_temp.map_weapon_json(pg_temp.map_weapon_json(defense, 'Massa', 'Mace'), 'Longmass', 'Longmace'), 'Mangual', 'Flail'),
  repel   = pg_temp.map_weapon_json(pg_temp.map_weapon_json(pg_temp.map_weapon_json(repel,   'Massa', 'Mace'), 'Longmass', 'Longmace'), 'Mangual', 'Flail')
WHERE attack->>'Weapon' IN ('Massa', 'Longmass', 'Mangual')
   OR defense->>'Weapon' IN ('Massa', 'Longmass', 'Mangual')
   OR repel->>'Weapon' IN ('Massa', 'Longmass', 'Mangual');

UPDATE cancelled_actions SET
  action = pg_temp.map_weapon_action(pg_temp.map_weapon_action(pg_temp.map_weapon_action(action, 'Massa', 'Mace'), 'Longmass', 'Longmace'), 'Mangual', 'Flail')
WHERE action->'attack'->>'Weapon' IN ('Massa', 'Longmass', 'Mangual')
   OR action->'defense'->>'Weapon' IN ('Massa', 'Longmass', 'Mangual')
   OR action->'repel'->>'Weapon' IN ('Massa', 'Longmass', 'Mangual');

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Desfaz o rename (Mace -> Massa, Longmace -> Longmass, Flail -> Mangual), nos mesmos campos.
CREATE OR REPLACE FUNCTION pg_temp.map_weapon_json(doc JSONB, from_a TEXT, to_a TEXT) RETURNS JSONB AS $f$
  SELECT CASE
    WHEN doc IS NOT NULL AND jsonb_typeof(doc) = 'object' AND doc->>'Weapon' = from_a
      THEN jsonb_set(doc, '{Weapon}', to_jsonb(to_a))
    ELSE doc
  END
$f$ LANGUAGE sql IMMUTABLE;

CREATE OR REPLACE FUNCTION pg_temp.map_weapon_action(doc JSONB, from_a TEXT, to_a TEXT) RETURNS JSONB AS $f$
  SELECT CASE WHEN jsonb_typeof(doc) = 'object' THEN
    (SELECT COALESCE(jsonb_object_agg(k,
        CASE WHEN k IN ('attack', 'defense', 'repel') THEN pg_temp.map_weapon_json(v, from_a, to_a) ELSE v END), '{}'::jsonb)
     FROM jsonb_each(doc) AS e(k, v))
  ELSE doc END
$f$ LANGUAGE sql IMMUTABLE;

UPDATE proficiencies SET weapon = 'Massa' WHERE weapon = 'Mace';
UPDATE proficiencies SET weapon = 'Longmass' WHERE weapon = 'Longmace';
UPDATE proficiencies SET weapon = 'Mangual' WHERE weapon = 'Flail';

UPDATE joint_proficiencies
SET weapons = array_replace(array_replace(array_replace(weapons, 'Mace', 'Massa'), 'Longmace', 'Longmass'), 'Flail', 'Mangual')
WHERE weapons && ARRAY['Mace', 'Longmace', 'Flail'];

UPDATE actions SET
  attack  = pg_temp.map_weapon_json(pg_temp.map_weapon_json(pg_temp.map_weapon_json(attack,  'Mace', 'Massa'), 'Longmace', 'Longmass'), 'Flail', 'Mangual'),
  defense = pg_temp.map_weapon_json(pg_temp.map_weapon_json(pg_temp.map_weapon_json(defense, 'Mace', 'Massa'), 'Longmace', 'Longmass'), 'Flail', 'Mangual'),
  repel   = pg_temp.map_weapon_json(pg_temp.map_weapon_json(pg_temp.map_weapon_json(repel,   'Mace', 'Massa'), 'Longmace', 'Longmass'), 'Flail', 'Mangual')
WHERE attack->>'Weapon' IN ('Mace', 'Longmace', 'Flail')
   OR defense->>'Weapon' IN ('Mace', 'Longmace', 'Flail')
   OR repel->>'Weapon' IN ('Mace', 'Longmace', 'Flail');

UPDATE cancelled_actions SET
  action = pg_temp.map_weapon_action(pg_temp.map_weapon_action(pg_temp.map_weapon_action(action, 'Mace', 'Massa'), 'Longmace', 'Longmass'), 'Flail', 'Mangual')
WHERE action->'attack'->>'Weapon' IN ('Mace', 'Longmace', 'Flail')
   OR action->'defense'->>'Weapon' IN ('Mace', 'Longmace', 'Flail')
   OR action->'repel'->>'Weapon' IN ('Mace', 'Longmace', 'Flail');

-- +goose StatementEnd
