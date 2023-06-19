-- +goose Up
ALTER TABLE oracle_data ADD COLUMN meta_data JSONB;
ALTER TABLE oracle_data_current ADD COLUMN meta_data JSONB;