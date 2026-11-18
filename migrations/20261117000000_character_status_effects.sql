-- migrations/20261117000000_character_status_effects.sql
-- +goose Up
-- +goose StatementBegin
BEGIN;

-- Efeitos de status de um personagem. Tabela da ficha: vale em qualquer campanha ou partida
-- (spec §10), por isso a chave é a ficha e não a partida.
--
-- variant é o sentido (kind = sense) ou a Aptidão (kind = weakened); vazio nos demais tipos.
-- Faz parte da identidade: um personagem tem no máximo um efeito por (kind, variant).
-- params é o JSON dos parâmetros normalizados (camelCase, chaves ausentes = zero).
-- audience NULL = padrão do tipo (§9); {"everyone":true} ou {"players":[...]}, com
-- {"players":[]} = só o mestre.
CREATE TABLE IF NOT EXISTS character_status_effects (
  uuid                  UUID         PRIMARY KEY,
  character_sheet_uuid  UUID         NOT NULL REFERENCES character_sheets(uuid) ON DELETE CASCADE,
  kind                  VARCHAR(32)  NOT NULL,
  variant               VARCHAR(32)  NOT NULL DEFAULT '',
  params                JSONB        NOT NULL DEFAULT '{}',
  alias                 VARCHAR(60)  NOT NULL DEFAULT '',
  audience              JSONB,
  applied_at            TIMESTAMPTZ  NOT NULL,
  updated_at            TIMESTAMPTZ  NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_character_status_effects_identity
  ON character_status_effects(character_sheet_uuid, kind, variant);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;
DROP INDEX IF EXISTS uq_character_status_effects_identity;
DROP TABLE IF EXISTS character_status_effects;
COMMIT;
-- +goose StatementEnd
