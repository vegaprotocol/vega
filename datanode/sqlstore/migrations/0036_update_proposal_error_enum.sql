-- +goose Up

ALTER TYPE proposal_error ADD VALUE IF NOT EXISTS 'PROPOSAL_ERROR_LP_PRICE_RANGE_NONPOSITIVE';
ALTER TYPE proposal_error ADD VALUE IF NOT EXISTS 'PROPOSAL_ERROR_LP_PRICE_RANGE_TOO_LARGE';
ALTER TYPE proposal_error ADD VALUE IF NOT EXISTS 'PROPOSAL_ERROR_LINEAR_SLIPPAGE_FACTOR_OUT_OF_RANGE';
ALTER TYPE proposal_error ADD VALUE IF NOT EXISTS 'PROPOSAL_ERROR_QUADRATIC_SLIPPAGE_FACTOR_OUT_OF_RANGE';
ALTER TYPE proposal_error ADD VALUE IF NOT EXISTS 'PROPOSAL_ERROR_INVALID_SPOT';
ALTER TYPE proposal_error ADD VALUE IF NOT EXISTS 'PROPOSAL_ERROR_SPOT_PRODUCT_DISABLED';
ALTER TYPE proposal_error ADD VALUE IF NOT EXISTS 'PROPOSAL_ERROR_INVALID_REFERRAL_PROGRAM';
ALTER TYPE proposal_error ADD VALUE IF NOT EXISTS 'PROPOSAL_ERROR_INVALID_VOLUME_DISCOUNT_PROGRAM';

-- +goose Down

-- do nothing