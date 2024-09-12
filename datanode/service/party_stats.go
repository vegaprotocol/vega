// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package service

import (
	"context"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"
)

// dependencies

// EpochStore is used to get the last epoch from DB.
type EpochStore interface {
	GetCurrent(ctx context.Context) (entities.Epoch, error)
}

// ReferralSetStore gets the referral set data, consider adding a custom method without the noise.
type ReferralSetStore interface {
	GetReferralSetStats(ctx context.Context, setID *entities.ReferralSetID, atEpoch *uint64, referee *entities.PartyID, pagination entities.CursorPagination) ([]entities.FlattenReferralSetStats, entities.PageInfo, error)
}

// VDSStore is Volume Discount Stats storage, again custom methods might need to be added.
type VDSStore interface {
	Stats(ctx context.Context, atEpoch *uint64, partyID *string, pagination entities.CursorPagination) ([]entities.FlattenVolumeDiscountStats, entities.PageInfo, error)
	LatestStats(ctx context.Context, partyID string) (entities.VolumeDiscountStats, error)
}

// VRSStore is Volume Rebate Stats, may need custom methods.
type VRSStore interface {
	Stats(ctx context.Context, atEpoch *uint64, partyID *string, pagination entities.CursorPagination) ([]entities.FlattenVolumeRebateStats, entities.PageInfo, error)
}

// MktStore is a duplicate interface at this point, but again: custom method fetching list of markets would be handy.
type MktStore interface {
	GetByIDs(ctx context.Context, marketID []string) ([]entities.Market, error)
	// NB: although it returns Market entity, all it has is id and fees. Trying to access anything else on it will get NPE.
	GetAllFees(ctx context.Context) ([]entities.Market, error)
}

type VRStore interface {
	GetCurrentVolumeRebateProgram(ctx context.Context) (entities.VolumeRebateProgram, error)
}

type RPStore interface {
	GetCurrentReferralProgram(ctx context.Context) (entities.ReferralProgram, error)
}

type VDStore interface {
	GetCurrentVolumeDiscountProgram(ctx context.Context) (entities.VolumeDiscountProgram, error)
}

// PSvc the actual service combining data from all dependencies.
type PSvc struct {
	epoch EpochStore
	ref   ReferralSetStore
	vds   VDSStore
	vrs   VRSStore
	mkt   MktStore
	rp    RPStore
	vd    VDStore
	vr    VRStore
}

type partyFeeFactors struct {
	maker     num.Decimal
	infra     num.Decimal
	liquidity num.Decimal
}

func NewPartyStatsService(epoch EpochStore, ref ReferralSetStore, vds VDSStore, vrs VRSStore, mkt MktStore, rp RPStore, vd VDStore, vr VRStore) *PSvc {
	return &PSvc{
		epoch: epoch,
		ref:   ref,
		vds:   vds,
		vrs:   vrs,
		mkt:   mkt,
		rp:    rp,
		vd:    vd,
		vr:    vr,
	}
}

func (s *PSvc) GetPartyStats(ctx context.Context, partyID string, markets []string) (*v2.GetPartyDiscountStatsResponse, error) {
	// first up, last epoch to get the stats:
	epoch, err := s.epoch.GetCurrent(ctx)
	if err != nil {
		return nil, err
	}
	// then get the markets:
	var mkts []entities.Market
	if len(markets) > 0 {
		mkts, err = s.mkt.GetByIDs(ctx, markets)
	} else {
		mkts, err = s.mkt.GetAllFees(ctx)
	}
	if err != nil {
		return nil, err
	}
	lastE := uint64(epoch.ID - 1)

	data := &v2.GetPartyDiscountStatsResponse{}
	rfDiscountFactors := partyFeeFactors{}
	rfRewardFactors := partyFeeFactors{}
	vdDiscountFactors := partyFeeFactors{}

	// now that we've gotten the epoch and all markets, get the party stats.
	// 1. referral set stats.
	refStats, _, err := s.ref.GetReferralSetStats(ctx, nil, &lastE, ptr.From(entities.PartyID(partyID)), entities.DefaultCursorPagination(true))
	if err != nil {
		return nil, err
	}
	if len(refStats) > 0 {
		tier, err := s.getReferralTier(ctx, refStats[0])
		if err != nil {
			return nil, err
		}
		if err := setRefFeeFactors(&rfRewardFactors, &rfDiscountFactors, refStats[0]); err != nil {
			return nil, err
		}
		if tier != nil {
			data.ReferralDiscountTier = *tier.TierNumber
		}
	}
	// 2. volume discount stats.
	vdStats, _, err := s.vds.Stats(ctx, &lastE, &partyID, entities.DefaultCursorPagination(true))
	if err != nil {
		return nil, err
	}
	if len(vdStats) > 0 {
		tier, err := s.getVolumeDiscountTier(ctx, vdStats[0])
		if err != nil {
			return nil, err
		}
		if err := setVolFeeFactors(&vdDiscountFactors, vdStats[0]); err != nil {
			return nil, err
		}
		if tier != nil {
			data.VolumeDiscountTier = *tier.TierNumber
		}
	}
	// 3. Volume Rebate stats.
	vrStats, _, err := s.vrs.Stats(ctx, &lastE, &partyID, entities.DefaultCursorPagination(true))
	if err != nil {
		return nil, err
	}
	var rebate num.Decimal
	if len(vrStats) > 0 {
		tier, err := s.getVolumeRebateTier(ctx, vrStats[0])
		if err != nil {
			return nil, err
		}
		rebate, err = num.DecimalFromString(vrStats[0].AdditionalRebate)
		if err != nil {
			return nil, err
		}
		if tier != nil {
			data.VolumeRebateTier = *tier.TierNumber
		}
	}
	for _, mkt := range mkts {
		// @TODO ensure non-nil slice!
		if err := setMarketFees(data, mkt, rfDiscountFactors, rfRewardFactors, vdDiscountFactors, rebate); err != nil {
			return nil, err
		}
	}
	return data, nil
}

func setMarketFees(data *v2.GetPartyDiscountStatsResponse, mkt entities.Market, rfDiscount, rfRewards, vdFactors partyFeeFactors, rebate num.Decimal) error {
	maker, infra, liquidity, bb, treasury, err := feeFactors(mkt)
	if err != nil {
		return err
	}
	// undiscounted
	base := num.DecimalZero().Add(maker).Add(infra).Add(liquidity).Add(bb).Add(treasury)
	// discounted
	discountedMaker := maker.Mul(num.DecimalOne().Sub(rfDiscount.maker)).Mul(num.DecimalOne().Sub(vdFactors.maker)).Mul(num.DecimalOne().Sub(rfRewards.maker))
	discountedInfra := infra.Mul(num.DecimalOne().Sub(rfDiscount.infra)).Mul(num.DecimalOne().Sub(vdFactors.infra)).Mul(num.DecimalOne().Sub(rfRewards.infra))
	discountedLiquidity := liquidity.Mul(num.DecimalOne().Sub(rfDiscount.liquidity)).Mul(num.DecimalOne().Sub(vdFactors.liquidity)).Mul(num.DecimalOne().Sub(rfRewards.liquidity))
	discounted := discountedMaker.Add(discountedInfra).Add(discountedLiquidity).Add(bb).Add(treasury)

	data.PartyMarketFees = append(data.PartyMarketFees, &v2.MarketFees{
		MarketId:             mkt.ID.String(),
		UndiscountedTakerFee: base.String(),
		DiscountedTakerFee:   discounted.String(),
		BaseMakerRebate:      maker.String(),
		UserMakerRebate:      maker.Add(rebate).String(),
	})
	return nil
}

func feeFactors(mkt entities.Market) (maker, infra, liquidity, bb, treasury num.Decimal, err error) {
	if maker, err = num.DecimalFromString(mkt.Fees.Factors.MakerFee); err != nil {
		return
	}
	if infra, err = num.DecimalFromString(mkt.Fees.Factors.InfrastructureFee); err != nil {
		return
	}
	if liquidity, err = num.DecimalFromString(mkt.Fees.Factors.LiquidityFee); err != nil {
		return
	}
	if bb, err = num.DecimalFromString(mkt.Fees.Factors.BuyBackFee); err != nil {
		return
	}
	if treasury, err = num.DecimalFromString(mkt.Fees.Factors.TreasuryFee); err != nil {
		return
	}

	return
}

func setRefFeeFactors(rewards, discounts *partyFeeFactors, stats entities.FlattenReferralSetStats) error {
	maker, err := num.DecimalFromString(stats.RewardFactors.MakerRewardFactor)
	if err != nil {
		return err
	}
	rewards.maker = maker
	infra, err := num.DecimalFromString(stats.RewardFactors.InfrastructureRewardFactor)
	if err != nil {
		return err
	}
	rewards.infra = infra
	liquidity, err := num.DecimalFromString(stats.RewardFactors.LiquidityRewardFactor)
	if err != nil {
		return err
	}
	rewards.liquidity = liquidity

	dmaker, err := num.DecimalFromString(stats.DiscountFactors.MakerDiscountFactor)
	if err != nil {
		return err
	}
	discounts.maker = dmaker
	dinfra, err := num.DecimalFromString(stats.DiscountFactors.InfrastructureDiscountFactor)
	if err != nil {
		return err
	}
	discounts.infra = dinfra
	dliquidity, err := num.DecimalFromString(stats.DiscountFactors.LiquidityDiscountFactor)
	if err != nil {
		return err
	}
	discounts.liquidity = dliquidity

	return nil
}

func setVolFeeFactors(ff *partyFeeFactors, stats entities.FlattenVolumeDiscountStats) error {
	maker, err := num.DecimalFromString(stats.DiscountFactors.MakerDiscountFactor)
	if err != nil {
		return err
	}
	ff.maker = maker
	infra, err := num.DecimalFromString(stats.DiscountFactors.InfrastructureDiscountFactor)
	if err != nil {
		return err
	}
	ff.infra = infra
	liquidity, err := num.DecimalFromString(stats.DiscountFactors.LiquidityDiscountFactor)
	if err != nil {
		return err
	}
	ff.liquidity = liquidity
	return nil
}

func (s *PSvc) getReferralTier(ctx context.Context, stats entities.FlattenReferralSetStats) (*vega.BenefitTier, error) {
	if stats.RewardFactors == nil {
		return nil, nil
	}
	current, err := s.rp.GetCurrentReferralProgram(ctx)
	if err != nil {
		return nil, err
	}
	for i, bt := range current.BenefitTiers {
		if bt.ReferralRewardFactors.InfrastructureRewardFactor == stats.RewardFactors.InfrastructureRewardFactor &&
			bt.ReferralRewardFactors.LiquidityRewardFactor == stats.RewardFactors.LiquidityRewardFactor &&
			bt.ReferralRewardFactors.MakerRewardFactor == stats.RewardFactors.MakerRewardFactor {
			tierNumber := uint64(i)
			bt.TierNumber = &tierNumber
			return bt, nil
		}
	}
	return nil, nil
}

func (s *PSvc) getVolumeDiscountTier(ctx context.Context, stats entities.FlattenVolumeDiscountStats) (*vega.VolumeBenefitTier, error) {
	if stats.DiscountFactors == nil {
		return nil, nil
	}
	vol, err := num.DecimalFromString(stats.RunningVolume)
	if err != nil {
		return nil, err
	}
	current, err := s.vd.GetCurrentVolumeDiscountProgram(ctx)
	if err != nil {
		return nil, err
	}
	if len(current.BenefitTiers) == 0 {
		return nil, nil
	}
	for i := len(current.BenefitTiers) - 1; i >= 0; i-- {
		dt := current.BenefitTiers[i]
		minV, _ := num.DecimalFromString(dt.MinimumRunningNotionalTakerVolume)
		if vol.GreaterThanOrEqual(minV) {
			dt.TierNumber = ptr.From(uint64(i))
			return dt, nil
		}
	}
	return nil, nil
}

func (s *PSvc) getVolumeRebateTier(ctx context.Context, stats entities.FlattenVolumeRebateStats) (*vega.VolumeRebateBenefitTier, error) {
	current, err := s.vr.GetCurrentVolumeRebateProgram(ctx)
	if err != nil {
		return nil, err
	}
	vf, err := num.DecimalFromString(stats.MakerVolumeFraction)
	if err != nil {
		return nil, err
	}
	if len(current.BenefitTiers) == 0 {
		return nil, nil
	}
	for i := len(current.BenefitTiers) - 1; i >= 0; i-- {
		bt := current.BenefitTiers[i]
		minF, _ := num.DecimalFromString(bt.MinimumPartyMakerVolumeFraction)
		if vf.GreaterThanOrEqual(minF) {
			bt.TierNumber = ptr.From(uint64(i))
			return bt, nil
		}
	}
	return nil, nil
}
