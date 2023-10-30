-- +goose Up
ALTER TABLE trades ADD COLUMN buyer_maker_fee_referral_discount HUGEINT NOT NULL DEFAULT(0),
                   ADD COLUMN buyer_infrastructure_fee_referral_discount HUGEINT NOT NULL DEFAULT(0),
                   ADD COLUMN buyer_liquidity_fee_referral_discount HUGEINT NOT NULL DEFAULT(0),
                   ADD COLUMN buyer_maker_fee_volume_discount HUGEINT NOT NULL DEFAULT(0),
                   ADD COLUMN buyer_infrastructure_fee_volume_discount HUGEINT NOT NULL DEFAULT(0),
                   ADD COLUMN buyer_liquidity_fee_volume_discount HUGEINT NOT NULL DEFAULT(0),
                   ADD COLUMN seller_maker_fee_referral_discount HUGEINT NOT NULL DEFAULT(0),
                   ADD COLUMN seller_infrastructure_fee_referral_discount HUGEINT NOT NULL DEFAULT(0),
                   ADD COLUMN seller_liquidity_fee_referral_discount HUGEINT NOT NULL DEFAULT(0),
                   ADD COLUMN seller_maker_fee_volume_discount HUGEINT NOT NULL DEFAULT(0),
                   ADD COLUMN seller_infrastructure_fee_volume_discount HUGEINT NOT NULL DEFAULT(0),
                   ADD COLUMN seller_liquidity_fee_volume_discount HUGEINT NOT NULL DEFAULT(0);

-- +goose Down
ALTER TABLE trades DROP COLUMN buyer_maker_fee_referral_discount,
                   DROP COLUMN buyer_infrastructure_fee_referral_discount,
                   DROP COLUMN buyer_liquidity_fee_referral_discount,
                   DROP COLUMN buyer_maker_fee_volume_discount,
                   DROP COLUMN buyer_infrastructure_fee_volume_discount,
                   DROP COLUMN buyer_liquidity_fee_volume_discount,
                   DROP COLUMN seller_maker_fee_referral_discount,
                   DROP COLUMN seller_infrastructure_fee_referral_discount,
                   DROP COLUMN seller_liquidity_fee_referral_discount,
                   DROP COLUMN seller_maker_fee_volume_discount,
                   DROP COLUMN seller_infrastructure_fee_volume_discount,
                   DROP COLUMN seller_liquidity_fee_volume_discount;
