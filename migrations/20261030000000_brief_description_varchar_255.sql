-- migrations/20261030000000_brief_description_varchar_255.sql
-- +goose Up
-- +goose StatementBegin

-- O commit a550045 (2025-09-03) editou a migration 20250325000035, já aplicada, trocando
-- brief_description de VARCHAR(32) para VARCHAR(255). O goose nunca reexecuta migration aplicada,
-- então os bancos criados antes desse commit (dev local e o hospedado) ficaram com 32, enquanto o
-- domínio e o front aceitam 255: salvar uma descrição com mais de 32 caracteres dava 500.
-- Em bancos novos a coluna já nasce com 255 e este ALTER não muda nada. Ampliar VARCHAR não perde
-- dados e, no Postgres, não reescreve a tabela.
ALTER TABLE character_profiles ALTER COLUMN brief_description TYPE VARCHAR(255);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Sem desfazer de propósito: voltar para VARCHAR(32) falharia (ou exigiria truncar) quando já
-- existirem descrições maiores que 32 caracteres, e bancos novos nunca tiveram 32.
SELECT 1;

-- +goose StatementEnd
