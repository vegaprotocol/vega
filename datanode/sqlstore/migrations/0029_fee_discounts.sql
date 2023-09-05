-- +goose Up
ALTER TABLE trades ADD COLUMN buyer_maker_fee_referral_discount HUGEINT;
Update trades set buyer_maker_fee_referral_discount = 0;
ALTER TABLE trades ADD COLUMN buyer_infrastructure_fee_referral_discount HUGEINT;
Update trades set buyer_infrastructure_fee_referral_discount = 0;
ALTER TABLE trades ADD COLUMN buyer_liquidity_fee_referral_discount HUGEINT;
Update trades set buyer_liquidity_fee_referral_discount = 0;
ALTER TABLE trades ADD COLUMN buyer_maker_fee_volume_discount HUGEINT;
Update trades set buyer_maker_fee_volume_discount = 0;
ALTER TABLE trades ADD COLUMN buyer_infrastructure_fee_volume_discount HUGEINT;
Update trades set buyer_infrastructure_fee_volume_discount = 0;
ALTER TABLE trades ADD COLUMN buyer_liquidity_fee_volume_discount HUGEINT;
Update trades set buyer_liquidity_fee_volume_discount = 0;
ALTER TABLE trades ADD COLUMN seller_maker_fee_referral_discount HUGEINT;
Update trades set seller_maker_fee_referral_discount = 0;
ALTER TABLE trades ADD COLUMN seller_infrastructure_fee_referral_discount HUGEINT;
Update trades set seller_infrastructure_fee_referral_discount = 0;
ALTER TABLE trades ADD COLUMN seller_liquidity_fee_referral_discount HUGEINT;
Update trades set seller_liquidity_fee_referral_discount = 0;
ALTER TABLE trades ADD COLUMN seller_maker_fee_volume_discount HUGEINT;
Update trades set seller_maker_fee_volume_discount = 0;
ALTER TABLE trades ADD COLUMN seller_infrastructure_fee_volume_discount HUGEINT;
Update trades set seller_infrastructure_fee_volume_discount = 0;
ALTER TABLE trades ADD COLUMN seller_liquidity_fee_volume_discount HUGEINT;
Update trades set seller_liquidity_fee_volume_discount = 0;

-- +goose Down
ALTER TABLE trades DROP COLUMN buyer_maker_fee_referral_discount;
ALTER TABLE trades DROP COLUMN buyer_infrastructure_fee_referral_discount;
ALTER TABLE trades DROP COLUMN buyer_liquidity_fee_referral_discount;
ALTER TABLE trades DROP COLUMN buyer_maker_fee_volume_discount;
ALTER TABLE trades DROP COLUMN buyer_infrastructure_fee_volume_discount;
ALTER TABLE trades DROP COLUMN buyer_liquidity_fee_volume_discount;
ALTER TABLE trades DROP COLUMN seller_maker_fee_referral_discount;
ALTER TABLE trades DROP COLUMN seller_infrastructure_fee_referral_discount;
ALTER TABLE trades DROP COLUMN seller_liquidity_fee_referral_discount;
ALTER TABLE trades DROP COLUMN seller_maker_fee_volume_discount;
ALTER TABLE trades DROP COLUMN seller_infrastructure_fee_volume_discount;
ALTER TABLE trades DROP COLUMN seller_liquidity_fee_volume_discount;