-- +goose Up

ALTER TYPE transfer_type ADD VALUE 'GovernanceOneOff';
ALTER TYPE transfer_type ADD VALUE 'GovernanceRecurring';