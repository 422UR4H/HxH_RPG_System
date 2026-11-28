-- +goose Up
-- +goose StatementBegin

-- As habilidades de Força Push e Carry viraram Force e Charge (Forçar e Carga). O nome da habilidade
-- vive em: character_sheets.push_exp / carry_exp (colunas renomeadas, o XP fica onde está) e dentro
-- dos JSONB do histórico, sob as chaves SkillName / skillName / DamageSkill / damageSkill:
-- actions (speed, skills, move, attack, defense, dodge, feint, trigger, interact, default_defense,
-- repel), cancelled_actions.action, turns.resolution, master_actions.content,
-- overridden_action_values.original_value e match_events.payload. Em overridden_action_values
-- o campo "skill:<Nome>.condition" também muda, e o original de "damageSkill" é uma string JSON solta.
-- Reescreve só esses valores sob essas chaves, nunca texto solto (uma arma ou nota "Push" fica como está).
-- Idempotente: reexecutar não encontra mais o nome antigo.
ALTER TABLE character_sheets RENAME COLUMN push_exp TO force_exp;
ALTER TABLE character_sheets RENAME COLUMN carry_exp TO charge_exp;

CREATE OR REPLACE FUNCTION pg_temp.map_skill_names(doc JSONB, renames JSONB) RETURNS JSONB AS $f$
DECLARE
  result JSONB;
BEGIN
  IF doc IS NULL THEN
    RETURN NULL;
  END IF;
  IF jsonb_typeof(doc) = 'object' THEN
    SELECT COALESCE(jsonb_object_agg(e.k,
      CASE
        WHEN e.k IN ('SkillName', 'skillName', 'DamageSkill', 'damageSkill')
             AND jsonb_typeof(e.v) = 'string' AND renames ? (e.v #>> '{}')
          THEN renames -> (e.v #>> '{}')
        ELSE pg_temp.map_skill_names(e.v, renames)
      END), '{}'::jsonb)
    INTO result
    FROM jsonb_each(doc) AS e(k, v);
    RETURN result;
  ELSIF jsonb_typeof(doc) = 'array' THEN
    SELECT COALESCE(jsonb_agg(pg_temp.map_skill_names(a.v, renames) ORDER BY a.ord), '[]'::jsonb)
    INTO result
    FROM jsonb_array_elements(doc) WITH ORDINALITY AS a(v, ord);
    RETURN result;
  END IF;
  RETURN doc;
END
$f$ LANGUAGE plpgsql IMMUTABLE;

UPDATE actions SET speed = pg_temp.map_skill_names(speed, '{"Push":"Force","Carry":"Charge"}'::jsonb)
WHERE speed::text ~ '"(Push|Carry)"';

UPDATE actions SET skills = pg_temp.map_skill_names(skills, '{"Push":"Force","Carry":"Charge"}'::jsonb)
WHERE skills::text ~ '"(Push|Carry)"';

UPDATE actions SET move = pg_temp.map_skill_names(move, '{"Push":"Force","Carry":"Charge"}'::jsonb)
WHERE move::text ~ '"(Push|Carry)"';

UPDATE actions SET attack = pg_temp.map_skill_names(attack, '{"Push":"Force","Carry":"Charge"}'::jsonb)
WHERE attack::text ~ '"(Push|Carry)"';

UPDATE actions SET defense = pg_temp.map_skill_names(defense, '{"Push":"Force","Carry":"Charge"}'::jsonb)
WHERE defense::text ~ '"(Push|Carry)"';

UPDATE actions SET dodge = pg_temp.map_skill_names(dodge, '{"Push":"Force","Carry":"Charge"}'::jsonb)
WHERE dodge::text ~ '"(Push|Carry)"';

UPDATE actions SET feint = pg_temp.map_skill_names(feint, '{"Push":"Force","Carry":"Charge"}'::jsonb)
WHERE feint::text ~ '"(Push|Carry)"';

UPDATE actions SET trigger = pg_temp.map_skill_names(trigger, '{"Push":"Force","Carry":"Charge"}'::jsonb)
WHERE trigger::text ~ '"(Push|Carry)"';

UPDATE actions SET interact = pg_temp.map_skill_names(interact, '{"Push":"Force","Carry":"Charge"}'::jsonb)
WHERE interact::text ~ '"(Push|Carry)"';

UPDATE actions SET default_defense = pg_temp.map_skill_names(default_defense, '{"Push":"Force","Carry":"Charge"}'::jsonb)
WHERE default_defense::text ~ '"(Push|Carry)"';

UPDATE actions SET repel = pg_temp.map_skill_names(repel, '{"Push":"Force","Carry":"Charge"}'::jsonb)
WHERE repel::text ~ '"(Push|Carry)"';

UPDATE cancelled_actions SET action = pg_temp.map_skill_names(action, '{"Push":"Force","Carry":"Charge"}'::jsonb)
WHERE action::text ~ '"(Push|Carry)"';

UPDATE turns SET resolution = pg_temp.map_skill_names(resolution, '{"Push":"Force","Carry":"Charge"}'::jsonb)
WHERE resolution::text ~ '"(Push|Carry)"';

UPDATE master_actions SET content = pg_temp.map_skill_names(content, '{"Push":"Force","Carry":"Charge"}'::jsonb)
WHERE content::text ~ '"(Push|Carry)"';

UPDATE overridden_action_values SET original_value = pg_temp.map_skill_names(original_value, '{"Push":"Force","Carry":"Charge"}'::jsonb)
WHERE original_value::text ~ '"(Push|Carry)"';

UPDATE match_events SET payload = pg_temp.map_skill_names(payload, '{"Push":"Force","Carry":"Charge"}'::jsonb)
WHERE payload::text ~ '"(Push|Carry)"';

UPDATE overridden_action_values SET field = 'skill:Force.condition'
WHERE field = 'skill:Push.condition';
UPDATE overridden_action_values SET field = 'skill:Charge.condition'
WHERE field = 'skill:Carry.condition';

UPDATE overridden_action_values SET original_value = to_jsonb('Force'::text)
WHERE field = 'damageSkill' AND original_value = to_jsonb('Push'::text);
UPDATE overridden_action_values SET original_value = to_jsonb('Charge'::text)
WHERE field = 'damageSkill' AND original_value = to_jsonb('Carry'::text);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Desfaz o rename (Force -> Push, Charge -> Carry), nos mesmos campos.
ALTER TABLE character_sheets RENAME COLUMN force_exp TO push_exp;
ALTER TABLE character_sheets RENAME COLUMN charge_exp TO carry_exp;

CREATE OR REPLACE FUNCTION pg_temp.map_skill_names(doc JSONB, renames JSONB) RETURNS JSONB AS $f$
DECLARE
  result JSONB;
BEGIN
  IF doc IS NULL THEN
    RETURN NULL;
  END IF;
  IF jsonb_typeof(doc) = 'object' THEN
    SELECT COALESCE(jsonb_object_agg(e.k,
      CASE
        WHEN e.k IN ('SkillName', 'skillName', 'DamageSkill', 'damageSkill')
             AND jsonb_typeof(e.v) = 'string' AND renames ? (e.v #>> '{}')
          THEN renames -> (e.v #>> '{}')
        ELSE pg_temp.map_skill_names(e.v, renames)
      END), '{}'::jsonb)
    INTO result
    FROM jsonb_each(doc) AS e(k, v);
    RETURN result;
  ELSIF jsonb_typeof(doc) = 'array' THEN
    SELECT COALESCE(jsonb_agg(pg_temp.map_skill_names(a.v, renames) ORDER BY a.ord), '[]'::jsonb)
    INTO result
    FROM jsonb_array_elements(doc) WITH ORDINALITY AS a(v, ord);
    RETURN result;
  END IF;
  RETURN doc;
END
$f$ LANGUAGE plpgsql IMMUTABLE;

UPDATE actions SET speed = pg_temp.map_skill_names(speed, '{"Force":"Push","Charge":"Carry"}'::jsonb)
WHERE speed::text ~ '"(Force|Charge)"';

UPDATE actions SET skills = pg_temp.map_skill_names(skills, '{"Force":"Push","Charge":"Carry"}'::jsonb)
WHERE skills::text ~ '"(Force|Charge)"';

UPDATE actions SET move = pg_temp.map_skill_names(move, '{"Force":"Push","Charge":"Carry"}'::jsonb)
WHERE move::text ~ '"(Force|Charge)"';

UPDATE actions SET attack = pg_temp.map_skill_names(attack, '{"Force":"Push","Charge":"Carry"}'::jsonb)
WHERE attack::text ~ '"(Force|Charge)"';

UPDATE actions SET defense = pg_temp.map_skill_names(defense, '{"Force":"Push","Charge":"Carry"}'::jsonb)
WHERE defense::text ~ '"(Force|Charge)"';

UPDATE actions SET dodge = pg_temp.map_skill_names(dodge, '{"Force":"Push","Charge":"Carry"}'::jsonb)
WHERE dodge::text ~ '"(Force|Charge)"';

UPDATE actions SET feint = pg_temp.map_skill_names(feint, '{"Force":"Push","Charge":"Carry"}'::jsonb)
WHERE feint::text ~ '"(Force|Charge)"';

UPDATE actions SET trigger = pg_temp.map_skill_names(trigger, '{"Force":"Push","Charge":"Carry"}'::jsonb)
WHERE trigger::text ~ '"(Force|Charge)"';

UPDATE actions SET interact = pg_temp.map_skill_names(interact, '{"Force":"Push","Charge":"Carry"}'::jsonb)
WHERE interact::text ~ '"(Force|Charge)"';

UPDATE actions SET default_defense = pg_temp.map_skill_names(default_defense, '{"Force":"Push","Charge":"Carry"}'::jsonb)
WHERE default_defense::text ~ '"(Force|Charge)"';

UPDATE actions SET repel = pg_temp.map_skill_names(repel, '{"Force":"Push","Charge":"Carry"}'::jsonb)
WHERE repel::text ~ '"(Force|Charge)"';

UPDATE cancelled_actions SET action = pg_temp.map_skill_names(action, '{"Force":"Push","Charge":"Carry"}'::jsonb)
WHERE action::text ~ '"(Force|Charge)"';

UPDATE turns SET resolution = pg_temp.map_skill_names(resolution, '{"Force":"Push","Charge":"Carry"}'::jsonb)
WHERE resolution::text ~ '"(Force|Charge)"';

UPDATE master_actions SET content = pg_temp.map_skill_names(content, '{"Force":"Push","Charge":"Carry"}'::jsonb)
WHERE content::text ~ '"(Force|Charge)"';

UPDATE overridden_action_values SET original_value = pg_temp.map_skill_names(original_value, '{"Force":"Push","Charge":"Carry"}'::jsonb)
WHERE original_value::text ~ '"(Force|Charge)"';

UPDATE match_events SET payload = pg_temp.map_skill_names(payload, '{"Force":"Push","Charge":"Carry"}'::jsonb)
WHERE payload::text ~ '"(Force|Charge)"';

UPDATE overridden_action_values SET field = 'skill:Push.condition'
WHERE field = 'skill:Force.condition';
UPDATE overridden_action_values SET field = 'skill:Carry.condition'
WHERE field = 'skill:Charge.condition';

UPDATE overridden_action_values SET original_value = to_jsonb('Push'::text)
WHERE field = 'damageSkill' AND original_value = to_jsonb('Force'::text);
UPDATE overridden_action_values SET original_value = to_jsonb('Carry'::text)
WHERE field = 'damageSkill' AND original_value = to_jsonb('Charge'::text);

-- +goose StatementEnd
