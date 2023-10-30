-- +goose Up

ALTER TYPE transfer_type ADD VALUE IF NOT EXISTS 'GovernanceOneOff';
ALTER TYPE transfer_type ADD VALUE IF NOT EXISTS 'GovernanceRecurring';
