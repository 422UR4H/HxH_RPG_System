-- migrations/20261107000000_rename_weighting_to_judgment.sql
-- +goose Up
-- +goose StatementBegin

-- O atributo mental "Weighting" (Ponderação) virou "Judgment" (Julgamento), para o nome em inglês
-- corresponder naturalmente ao português. As colunas de pontos e de exp acompanham o rename.
-- Idempotente: só renomeia se a coluna antiga existir, pois o dev local e o hospedado podem ter
-- esquemas divergentes entre si e de um banco novo (migrations antigas já foram editadas antes).
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = current_schema() AND table_name = 'character_sheets' AND column_name = 'weighting_pts'
  ) THEN
    ALTER TABLE character_sheets RENAME COLUMN weighting_pts TO judgment_pts;
  END IF;
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = current_schema() AND table_name = 'character_sheets' AND column_name = 'weighting_exp'
  ) THEN
    ALTER TABLE character_sheets RENAME COLUMN weighting_exp TO judgment_exp;
  END IF;
END $$;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Desfaz o rename, também idempotente (só renomeia se a coluna nova existir).
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = current_schema() AND table_name = 'character_sheets' AND column_name = 'judgment_pts'
  ) THEN
    ALTER TABLE character_sheets RENAME COLUMN judgment_pts TO weighting_pts;
  END IF;
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = current_schema() AND table_name = 'character_sheets' AND column_name = 'judgment_exp'
  ) THEN
    ALTER TABLE character_sheets RENAME COLUMN judgment_exp TO weighting_exp;
  END IF;
END $$;

-- +goose StatementEnd
