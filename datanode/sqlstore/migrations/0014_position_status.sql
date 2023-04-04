-- +goose Up

ALTER TYPE position_status_type ADD VALUE IF NOT EXISTS 'POSITION_STATUS_DISTRESSED';

-- +goose Down

-- noop
