-- +goose Up

alter table ledger add column if not exists transfer_id bytea null;

-- +goose Down

alter table ledger drop column if exists transfer_id;
