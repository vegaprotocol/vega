-- +goose Up
ALTER TABLE oracle_data ADD COLUMN IF NOT EXISTS meta_data JSONB;

ALTER TABLE oracle_data ADD COLUMN IF NOT EXISTS error text;
