-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION check_campaign_nick_unique()
RETURNS TRIGGER AS $$
DECLARE
    sheet_nick VARCHAR(16);
BEGIN
    IF NEW.campaign_uuid IS NULL THEN
        RETURN NEW;
    END IF;

    IF TG_OP = 'UPDATE' AND OLD.campaign_uuid IS NOT DISTINCT FROM NEW.campaign_uuid THEN
        RETURN NEW;
    END IF;

    SELECT nickname INTO sheet_nick
    FROM character_profiles
    WHERE character_sheet_uuid = NEW.uuid;

    IF NOT FOUND THEN
        RETURN NEW;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM character_sheets cs
        JOIN character_profiles cp ON cp.character_sheet_uuid = cs.uuid
        WHERE cs.campaign_uuid = NEW.campaign_uuid
          AND cp.nickname = sheet_nick
          AND cs.uuid != NEW.uuid
    ) THEN
        RAISE EXCEPTION 'a character with this nickname already exists in the campaign'
            USING ERRCODE = '23505';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER enforce_campaign_nick_unique
    BEFORE INSERT OR UPDATE OF campaign_uuid ON character_sheets
    FOR EACH ROW
    EXECUTE FUNCTION check_campaign_nick_unique();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS enforce_campaign_nick_unique ON character_sheets;
-- +goose StatementEnd

-- +goose StatementBegin
DROP FUNCTION IF EXISTS check_campaign_nick_unique();
-- +goose StatementEnd
