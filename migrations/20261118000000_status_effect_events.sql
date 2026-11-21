-- migrations/20261118000000_status_effect_events.sql
-- +goose Up
-- +goose StatementBegin
BEGIN;

-- A história de cada efeito de status: uma linha por aplicação, edição ou remoção, gravada na
-- mesma operação que grava o efeito (fora de turno, na hora; dentro, com o fechamento dele).
--
-- snapshot = o efeito como ficou (removed: como era), no formato de armazenamento do gateway.
-- listing  = quem a lista (ou o padrão do tipo) incluía NAQUELE instante, já resolvida:
--            {"everyone":true} ou {"players":[...]} — nunca NULL.
-- views    = jogadores que enxergavam o personagem logo antes ou logo depois + o dono
--            ({"<playerUUID>":"full"}); o mestre não entra.
-- hidden_from = jogadores (nunca o mestre nem o dono) para quem o personagem estava Escondido
--            no instante: {"<playerUUID>":"<hiddenEffectUUID>"}. NULL = ninguém.
-- action_uuid = a action ou reação do turno a que a mudança pertence; NULL = do turno/round.
--            Sem FK: a linha da action é gravada no mesmo fechamento.
--
-- scene/round/turn sem FK: a gravação fora de turno não garante as linhas (flushStatusEffects
-- não tem o ensure do recordMasterAction) e uma FK recusada envenenaria a fila de cura para
-- sempre. seq ordena "o último evento" sem depender de relógio (empate de happened_at).
CREATE TABLE IF NOT EXISTS status_effect_events (
  uuid                 UUID        PRIMARY KEY,
  seq                  BIGINT      GENERATED ALWAYS AS IDENTITY,
  effect_uuid          UUID        NOT NULL,
  character_sheet_uuid UUID        NOT NULL REFERENCES character_sheets(uuid) ON DELETE CASCADE,
  match_uuid           UUID        NOT NULL REFERENCES matches(uuid) ON DELETE CASCADE,
  scene_uuid           UUID        NOT NULL,
  round_uuid           UUID        NOT NULL,
  turn_uuid            UUID,
  action_uuid          UUID,
  change               VARCHAR(16) NOT NULL CHECK (change IN ('applied','edited','removed')),
  snapshot             JSONB       NOT NULL,
  listing              JSONB       NOT NULL,
  views                JSONB       NOT NULL,
  hidden_from          JSONB,
  happened_at          TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_status_effect_events_match  ON status_effect_events(match_uuid, happened_at, seq);
CREATE INDEX IF NOT EXISTS idx_status_effect_events_effect ON status_effect_events(effect_uuid, seq DESC);
CREATE INDEX IF NOT EXISTS idx_status_effect_events_sheet  ON status_effect_events(character_sheet_uuid);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;
DROP TABLE IF EXISTS status_effect_events;
COMMIT;
-- +goose StatementEnd
