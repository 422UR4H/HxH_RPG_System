-- migrations/20261028000000_spiritual_skills_exp.sql
-- +goose Up
-- +goose StatementBegin
BEGIN;

-- O commit 1bf85a9 reestruturou o domínio espiritual: o atributo Spirit virou Flame + Conscience
-- e as perícias espirituais passaram a ser Focus, WillPower, SelfKnowledge (Flame) e
-- Coa, Mop, Aop (Conscience). O banco nunca acompanhou: faltavam colunas para SelfKnowledge e
-- Mop, e a EXP do CoA vivia em aura_control_exp (o antigo princípio AuraControl).
-- Nenhum código jamais gravou essas colunas (todas as linhas têm 0), então não se perde nada.
ALTER TABLE character_sheets RENAME COLUMN aura_control_exp TO coa_exp;
ALTER TABLE character_sheets ADD COLUMN self_knowledge_exp INT DEFAULT 0;
ALTER TABLE character_sheets ADD COLUMN mop_exp INT DEFAULT 0;

-- A perícia Nen foi removida no commit 1bf85a9 e nenhum código grava nen_exp.
ALTER TABLE character_sheets DROP COLUMN nen_exp;

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

ALTER TABLE character_sheets ADD COLUMN nen_exp INT;
ALTER TABLE character_sheets DROP COLUMN mop_exp;
ALTER TABLE character_sheets DROP COLUMN self_knowledge_exp;
ALTER TABLE character_sheets RENAME COLUMN coa_exp TO aura_control_exp;

COMMIT;
-- +goose StatementEnd
