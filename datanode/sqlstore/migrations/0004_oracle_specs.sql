-- +goose Up
ALTER TABLE oracle_specs DROP COLUMN IF EXISTS signers;
ALTER TABLE oracle_specs DROP COLUMN IF EXISTS filters;
ALTER TABLE oracle_specs ADD COLUMN data JSONB;

-- +goose Down
ALTER TABLE oracle_specs DROP COLUMN IF EXISTS data;
ALTER TABLE oracle_specs ADD COLUMN filters jsonb;
ALTER TABLE oracle_specs ADD COLUMN signers bytea[];
