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

package fee

import (
	"errors"
	"sort"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

var (
	ErrEmptyTrades      = errors.New("empty trades slice sent to fees")
	ErrInvalidFeeFactor = errors.New("fee factors must be positive")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/core/fee ReferralDiscountRewardService,VolumeDiscountService,VolumeRebateService
type ReferralDiscountRewardService interface {
	ReferralDiscountFactorsForParty(party types.PartyID) types.Factors
	RewardsFactorsMultiplierAppliedForParty(party types.PartyID) types.Factors
	GetReferrer(referee types.PartyID) (types.PartyID, error)
}

type VolumeDiscountService interface {
	VolumeDiscountFactorForParty(party types.PartyID) types.Factors
}

type VolumeRebateService interface {
	VolumeRebateFactorForParty(party types.PartyID) num.Decimal
}

type Engine struct {
	log *logging.Logger
	cfg Config

	asset          string
	feeCfg         types.Fees
	f              factors
	positionFactor num.Decimal

	feesStats *FeesStats
}

type factors struct {
	makerFee          num.Decimal
	infrastructureFee num.Decimal
	liquidityFee      num.Decimal
	treasuryFee       num.Decimal
	buyBackFee        num.Decimal
}

func New(
	log *logging.Logger,
	cfg Config,
	feeCfg types.Fees,
	asset string,
	positionFactor num.Decimal,
) (*Engine, error) {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	e := &Engine{
		log:            log,
		feeCfg:         feeCfg,
		cfg:            cfg,
		asset:          asset,
		positionFactor: positionFactor,
		feesStats:      NewFeesStats(),
	}
	return e, e.UpdateFeeFactors(e.feeCfg)
}

func NewFromState(
	log *logging.Logger,
	cfg Config,
	feeCfg types.Fees,
	asset string,
	positionFactor num.Decimal,
	FeesStats *eventspb.FeesStats,
) (*Engine, error) {
	e, err := New(log, cfg, feeCfg, asset, positionFactor)
	if err != nil {
		return nil, err
	}

	e.feesStats = NewFeesStatsFromProto(FeesStats)

	return e, nil
}

func (e *Engine) GetState(assetQuantum num.Decimal) *eventspb.FeesStats {
	return e.feesStats.ToProto(e.asset, assetQuantum)
}

func (e *Engine) TotalTradingFeesPerParty() map[string]*num.Uint {
	return e.feesStats.TotalTradingFeesPerParty()
}

func (e *Engine) GetFeesStatsOnEpochEnd(assetQuantum num.Decimal) (FeesStats *eventspb.FeesStats) {
	FeesStats, e.feesStats = e.feesStats.ToProto(e.asset, assetQuantum), NewFeesStats()
	return
}

// ReloadConf is used in order to reload the internal configuration of
// the fee engine.
func (e *Engine) ReloadConf(cfg Config) {
	e.log.Info("reloading configuration")
	if e.log.GetLevel() != cfg.Level.Get() {
		e.log.Info("updating log level",
			logging.String("old", e.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		e.log.SetLevel(cfg.Level.Get())
	}

	e.cfg = cfg
}

func (e *Engine) UpdateFeeFactors(fees types.Fees) error {
	if fees.Factors.MakerFee.IsNegative() || fees.Factors.InfrastructureFee.IsNegative() || fees.Factors.LiquidityFee.IsNegative() || fees.Factors.BuyBackFee.IsNegative() || fees.Factors.TreasuryFee.IsNegative() {
		return ErrInvalidFeeFactor
	}
	e.f.makerFee = fees.Factors.MakerFee
	e.f.infrastructureFee = fees.Factors.InfrastructureFee
	// not sure we need the IsPositive check here, that ought to be validation
	if !fees.Factors.LiquidityFee.IsZero() && fees.Factors.LiquidityFee.IsPositive() {
		e.f.liquidityFee = fees.Factors.LiquidityFee
	}
	e.f.treasuryFee = fees.Factors.TreasuryFee
	e.f.buyBackFee = fees.Factors.BuyBackFee

	e.feeCfg = fees
	return nil
}

func (e *Engine) SetLiquidityFee(v num.Decimal) {
	e.f.liquidityFee = v
}

// CalculateForContinuousMode calculate the fee for
// trades which were produced from a market running
// in continuous trading mode.
// A single FeesTransfer is produced here as all fees
// are paid by the aggressive order.
func (e *Engine) CalculateForContinuousMode(
	trades []*types.Trade,
	referral ReferralDiscountRewardService,
	volumeDiscountService VolumeDiscountService,
	volumeRebateService VolumeRebateService,
) (events.FeesTransfer, error) {
	if len(trades) <= 0 {
		return nil, ErrEmptyTrades
	}

	var (
		taker, maker                 string
		totalFeeAmount               = num.UintZero()
		totalInfrastructureFeeAmount = num.UintZero()
		totalLiquidityFeeAmount      = num.UintZero()
		totalRewardAmount            = num.UintZero()
		// we allocate the len of the trades + 2
		// len(trade) = number of makerFee + 1 infra fee + 1 liquidity fee
		transfers     = make([]*types.Transfer, 0, (len(trades)*2)+2)
		transfersRecv = make([]*types.Transfer, 0, len(trades)+2)
	)

	for _, trade := range trades {
		taker = trade.Buyer
		maker = trade.Seller
		if trade.Aggressor == types.SideSell {
			taker = trade.Seller
			maker = trade.Buyer
		}
		size := num.NewUint(trade.Size)
		// multiply by size
		tradeValueForFee := size.Mul(trade.Price, size).ToDecimal().Div(e.positionFactor)
		fee, reward := e.applyDiscountsAndRewards(taker, maker, tradeValueForFee, e.calculateContinuousModeFees(trade), referral, volumeDiscountService, volumeRebateService)

		e.feesStats.RegisterMakerFee(maker, taker, fee.MakerFee)

		totalTradingFees := num.UintZero().AddSum(fee.MakerFee, fee.InfrastructureFee, fee.LiquidityFee, fee.BuyBackFee, fee.TreasuryFee, fee.HighVolumeMakerFee)

		switch trade.Aggressor {
		case types.SideBuy:
			trade.BuyerFee = fee
			trade.SellerFee = types.NewFee()
			maker = trade.Seller
		case types.SideSell:
			trade.SellerFee = fee
			trade.BuyerFee = types.NewFee()
			maker = trade.Buyer
		}

		e.feesStats.RegisterTradingFees(taker, totalTradingFees)
		e.feesStats.RegisterTradingFees(maker, fee.MakerFee)

		totalFeeAmount.AddSum(totalTradingFees)
		totalInfrastructureFeeAmount.AddSum(fee.InfrastructureFee)
		totalLiquidityFeeAmount.AddSum(fee.LiquidityFee)
		// create a transfer for the aggressor
		transfers = append(transfers, &types.Transfer{
			Owner: taker,
			Amount: &types.FinancialAmount{
				Asset:  e.asset,
				Amount: fee.MakerFee.Clone(),
			},
			Type: types.TransferTypeMakerFeePay,
		})
		// create a transfer for the maker
		transfersRecv = append(transfersRecv, &types.Transfer{
			Owner: maker,
			Amount: &types.FinancialAmount{
				Asset:  e.asset,
				Amount: fee.MakerFee.Clone(),
			},
			Type: types.TransferTypeMakerFeeReceive,
		})

		if !fee.HighVolumeMakerFee.IsZero() {
			// create a transfer for the aggressor
			transfers = append(transfers, &types.Transfer{
				Owner: taker,
				Amount: &types.FinancialAmount{
					Asset:  e.asset,
					Amount: fee.HighVolumeMakerFee.Clone(),
				},
				Type: types.TransferTypeHighMakerRebatePay,
			})
			// create a transfer for the maker
			transfersRecv = append(transfersRecv, &types.Transfer{
				Owner: maker,
				Amount: &types.FinancialAmount{
					Asset:  e.asset,
					Amount: fee.HighVolumeMakerFee.Clone(),
				},
				Type: types.TransferTypeHighMakerRebateReceive,
			})
		}

		// create a transfer for the aggressor
		transfers = append(transfers, &types.Transfer{
			Owner: taker,
			Amount: &types.FinancialAmount{
				Asset:  e.asset,
				Amount: fee.BuyBackFee.Clone(),
			},
			Type: types.TransferTypeBuyBackFeePay,
		})

		// create a transfer for the aggressor
		transfers = append(transfers, &types.Transfer{
			Owner: taker,
			Amount: &types.FinancialAmount{
				Asset:  e.asset,
				Amount: fee.TreasuryFee.Clone(),
			},
			Type: types.TransferTypeTreasuryPay,
		})

		if reward == nil {
			continue
		}
		totalRewardAmount.AddSum(reward.InfrastructureFeeReferrerReward)
		totalRewardAmount.AddSum(reward.LiquidityFeeReferrerReward)
		totalRewardAmount.AddSum(reward.MakerFeeReferrerReward)
	}

	// now create transfer for the infrastructure
	transfers = append(transfers, &types.Transfer{
		Owner: taker,
		Amount: &types.FinancialAmount{
			Asset:  e.asset,
			Amount: totalInfrastructureFeeAmount,
		},
		Type: types.TransferTypeInfrastructureFeePay,
	})
	// now create transfer for the liquidity
	transfers = append(transfers, &types.Transfer{
		Owner: taker,
		Amount: &types.FinancialAmount{
			Asset:  e.asset,
			Amount: totalLiquidityFeeAmount,
		},
		Type: types.TransferTypeLiquidityFeePay,
	})

	// if there's a referral reward - add transfers for it
	if !totalRewardAmount.IsZero() {
		referrer, _ := referral.GetReferrer(types.PartyID(taker))
		transfers = append(transfers, &types.Transfer{
			Owner: taker,
			Amount: &types.FinancialAmount{
				Asset:  e.asset,
				Amount: totalRewardAmount.Clone(),
			},
			Type: types.TransferTypeFeeReferrerRewardPay,
		})
		transfersRecv = append(transfersRecv, &types.Transfer{
			Owner: string(referrer),
			Amount: &types.FinancialAmount{
				Asset:  e.asset,
				Amount: totalRewardAmount.Clone(),
			},
			Type: types.TransferTypeFeeReferrerRewardDistribute,
		})
	}

	return &feesTransfer{
		totalFeesAmountsPerParty: map[string]*num.Uint{taker: totalFeeAmount, maker: num.UintZero()},
		transfers:                append(transfers, transfersRecv...),
	}, nil
}

// CalculateForAuctionMode calculate the fee for
// trades which were produced from a market running in
// in auction trading mode.
// A list FeesTransfer is produced each containing fees transfer from a
// single party.
func (e *Engine) CalculateForAuctionMode(
	trades []*types.Trade,
	referral ReferralDiscountRewardService,
	volumeDiscount VolumeDiscountService,
	volumeRebate VolumeRebateService,
) (events.FeesTransfer, error) {
	if len(trades) <= 0 {
		return nil, ErrEmptyTrades
	}
	var (
		totalFeesAmounts = map[string]*num.Uint{}
		// we allocate for len of trades *4 as all trades generate
		// 2 fees per party
		transfers = make([]*types.Transfer, 0, len(trades)*4)
	)

	// we iterate over all trades
	// for each trades both party needs to pay half of the fees
	// no maker fees are to be paid here.
	for _, v := range trades {
		buyerFess, sellerFees, newTransfers := e.getAuctionModeFeesAndTransfers(v, referral, volumeDiscount, volumeRebate)
		transfers = append(transfers, newTransfers...)

		// increase the total fee for the parties
		if sellerTotalFee, ok := totalFeesAmounts[v.Seller]; !ok {
			totalFeesAmounts[v.Seller] = num.Sum(sellerFees.InfrastructureFee, sellerFees.LiquidityFee)
		} else {
			sellerTotalFee.AddSum(num.Sum(sellerFees.InfrastructureFee, sellerFees.LiquidityFee))
		}
		if buyerTotalFee, ok := totalFeesAmounts[v.Buyer]; !ok {
			totalFeesAmounts[v.Buyer] = num.Sum(buyerFess.InfrastructureFee, buyerFess.LiquidityFee).Clone()
		} else {
			buyerTotalFee.AddSum(num.Sum(buyerFess.InfrastructureFee, buyerFess.LiquidityFee).Clone())
		}

		v.BuyerFee = buyerFess
		v.SellerFee = sellerFees
	}

	return &feesTransfer{
		totalFeesAmountsPerParty: totalFeesAmounts,
		transfers:                transfers,
	}, nil
}

// CalculateForFrequentBatchesAuctionMode calculate the fee for
// trades which were produced from a market running
// in auction trading mode.
// A list FeesTransfer is produced each containing fees transfer from a
// single party.
func (e *Engine) CalculateForFrequentBatchesAuctionMode(
	trades []*types.Trade,
	referral ReferralDiscountRewardService,
	volumeDiscount VolumeDiscountService,
	volumeRebate VolumeRebateService,
) (events.FeesTransfer, error) {
	if len(trades) <= 0 {
		return nil, ErrEmptyTrades
	}

	var (
		totalFeesAmounts = map[string]*num.Uint{}
		// we allocate for len of trades *4 as all trades generate
		// at lest2 fees per party
		transfers = make([]*types.Transfer, 0, len(trades)*4)
	)

	// we iterate over all trades
	// if the parties submitted the order in the same batches,
	// auction mode fees apply.
	// if not then the aggressor is the party which submitted
	// the order last, and continuous trading fees apply
	for _, v := range trades {
		var (
			sellerTotalFee, buyerTotalFee *num.Uint
			newTransfers                  []*types.Transfer
		)
		// we are in the same auction, normal auction fees applies
		if v.BuyerAuctionBatch == v.SellerAuctionBatch {
			v.BuyerFee, v.SellerFee, newTransfers = e.getAuctionModeFeesAndTransfers(v, referral, volumeDiscount, volumeRebate)
			sellerTotalFee = num.Sum(v.BuyerFee.InfrastructureFee, v.BuyerFee.LiquidityFee)
			buyerTotalFee = num.Sum(v.SellerFee.InfrastructureFee, v.SellerFee.LiquidityFee)
		} else {
			// set the aggressor to be the side of the party
			// entering the later auction
			v.Aggressor = types.SideSell
			if v.BuyerAuctionBatch > v.SellerAuctionBatch {
				v.Aggressor = types.SideBuy
			}
			// fees are being assign to the trade directly
			// no need to do add them there as well
			ftrnsfr, _ := e.CalculateForContinuousMode([]*types.Trade{v}, referral, volumeDiscount, volumeRebate)
			newTransfers = ftrnsfr.Transfers()
			buyerTotalFee = ftrnsfr.TotalFeesAmountPerParty()[v.Buyer]
			sellerTotalFee = ftrnsfr.TotalFeesAmountPerParty()[v.Seller]
		}

		transfers = append(transfers, newTransfers...)

		// increase the total fee for the parties
		if prevTotalFee, ok := totalFeesAmounts[v.Seller]; !ok {
			totalFeesAmounts[v.Seller] = sellerTotalFee.Clone()
		} else {
			prevTotalFee.AddSum(sellerTotalFee)
		}
		if prevTotalFee, ok := totalFeesAmounts[v.Buyer]; !ok {
			totalFeesAmounts[v.Buyer] = buyerTotalFee.Clone()
		} else {
			prevTotalFee.AddSum(buyerTotalFee)
		}
	}

	return &feesTransfer{
		totalFeesAmountsPerParty: totalFeesAmounts,
		transfers:                transfers,
	}, nil
}

func (e *Engine) GetFeeForPositionResolution(trades []*types.Trade,
	referral ReferralDiscountRewardService,
	volumeDiscount VolumeDiscountService,
	volumeRebate VolumeRebateService,
) (events.FeesTransfer, *types.Fee) {
	if len(trades) == 0 {
		return nil, nil
	}
	var (
		netFee *types.Fee
		gt     []*types.Transfer
	)
	transfers := make([]*types.Transfer, 0, len(trades))
	for _, t := range trades {
		fees := e.calculateContinuousModeFees(t)

		maker := t.Buyer
		if t.Buyer == types.NetworkParty {
			maker = t.Seller
		}
		size := num.NewUint(t.Size)
		// multiply by size
		tradeValueForFee := size.Mul(t.Price, size).ToDecimal().Div(e.positionFactor)
		postRewardDiscountFees, _ := e.applyDiscountsAndRewards(types.NetworkParty, maker, tradeValueForFee, fees, referral, volumeDiscount, volumeRebate)
		e.feesStats.RegisterMakerFee(maker, types.NetworkParty, postRewardDiscountFees.MakerFee)

		goodParty := t.Buyer
		t.SellerFee = postRewardDiscountFees
		if t.Buyer == types.NetworkParty {
			goodParty = t.Seller
			t.SellerFee = types.NewFee()
			t.BuyerFee = postRewardDiscountFees
		}
		netFee, gt = e.getNetworkFeeWithMakerTransfer(postRewardDiscountFees, netFee, goodParty)
		transfers = append(transfers, gt...)
	}
	netTf, total := e.getNetworkFeeTransfers(netFee)
	// calculate the
	return &feesTransfer{
		totalFeesAmountsPerParty: map[string]*num.Uint{
			types.NetworkParty: total,
		},
		transfers: append(netTf, transfers...),
	}, netFee
}

// BuildLiquidityFeeDistributionTransfer returns the set of transfers that will
// be used by the collateral engine to distribute the fees.  As shares are
// represented in float64 and fees are uint64, shares are floored and the
// remainder is assigned to the last party on the share map. Note that the map
// is sorted lexicographically to keep determinism.
func (e *Engine) BuildLiquidityFeeDistributionTransfer(shares map[string]num.Decimal, acc *types.Account) events.FeesTransfer {
	return e.buildLiquidityFeesTransfer(shares, acc, types.TransferTypeLiquidityFeeDistribute)
}

// BuildLiquidityFeeAllocationTransfer returns the set of transfers that will
// be used by the collateral engine to allocate the fees to liquidity providers per market fee accounts.
// As shares are represented in float64 and fees are uint64, shares are floored and the
// remainder is assigned to the last party on the share map. Note that the map
// is sorted lexicographically to keep determinism.
func (e *Engine) BuildLiquidityFeeAllocationTransfer(shares map[string]num.Decimal, acc *types.Account) events.FeesTransfer {
	return e.buildLiquidityFeesTransfer(shares, acc, types.TransferTypeLiquidityFeeAllocate)
}

func (e *Engine) buildLiquidityFeesTransfer(
	shares map[string]num.Decimal,
	acc *types.Account,
	transferType types.TransferType,
) events.FeesTransfer {
	if len(shares) == 0 {
		return nil
	}

	ft := &feesTransfer{
		totalFeesAmountsPerParty: map[string]*num.Uint{},
		transfers:                make([]*types.Transfer, 0, len(shares)),
	}

	// Get all the map keys
	keys := make([]string, 0, len(shares))

	for key := range shares {
		keys = append(keys, key)
		ft.totalFeesAmountsPerParty[key] = num.UintZero()
	}
	sort.Strings(keys)

	feeBal := acc.Balance.ToDecimal()
	var floored num.Decimal
	for _, key := range keys {
		share := shares[key]
		cs := feeBal.Mul(share).Floor()
		floored = floored.Add(cs)

		amount, _ := num.UintFromDecimal(cs)
		// populate the return value
		ft.totalFeesAmountsPerParty[key].AddSum(amount)
		ft.transfers = append(ft.transfers, &types.Transfer{
			Owner: key,
			Amount: &types.FinancialAmount{
				Amount: amount,
				Asset:  acc.Asset,
			},
			MinAmount: amount.Clone(),
			Type:      transferType,
		})
	}

	// if there is a remainder, just keep it in the fee account, will be used next time we pay out fees
	// last is the party who will get the remaining from ceil
	return ft
}

func (e *Engine) getNetworkFeeWithMakerTransfer(fees *types.Fee, current *types.Fee, goodParty string) (*types.Fee, []*types.Transfer) {
	transfers := []*types.Transfer{}
	transfers = append(transfers, &types.Transfer{
		Owner: goodParty,
		Amount: &types.FinancialAmount{
			Asset:  e.asset,
			Amount: fees.MakerFee.Clone(),
		},
		MinAmount: num.UintZero(),
		Type:      types.TransferTypeMakerFeeReceive,
	})
	if !fees.HighVolumeMakerFee.IsZero() {
		transfers = append(transfers, &types.Transfer{
			Owner: goodParty,
			Amount: &types.FinancialAmount{
				Asset:  e.asset,
				Amount: fees.HighVolumeMakerFee,
			},
			MinAmount: num.UintZero(),
			Type:      types.TransferTypeHighMakerRebateReceive,
		})
	}

	if current == nil {
		return fees.Clone(), transfers
	}
	current.MakerFee.AddSum(fees.MakerFee)
	current.LiquidityFee.AddSum(fees.LiquidityFee)
	current.InfrastructureFee.AddSum(fees.InfrastructureFee)
	current.BuyBackFee.AddSum(fees.BuyBackFee)
	current.TreasuryFee.AddSum(fees.TreasuryFee)
	current.HighVolumeMakerFee.AddSum(fees.HighVolumeMakerFee)

	return current, transfers
}

func (e *Engine) getNetworkFeeTransfers(fees *types.Fee) ([]*types.Transfer, *num.Uint) {
	transfers := []*types.Transfer{
		{
			Owner: types.NetworkParty,
			Amount: &types.FinancialAmount{
				Asset:  e.asset,
				Amount: fees.MakerFee.Clone(),
			},
			MinAmount: num.UintZero(),
			Type:      types.TransferTypeMakerFeePay,
		},
		{
			Owner: types.NetworkParty,
			Amount: &types.FinancialAmount{
				Asset:  e.asset,
				Amount: fees.InfrastructureFee.Clone(),
			},
			MinAmount: num.UintZero(),
			Type:      types.TransferTypeInfrastructureFeePay,
		},
		{
			Owner: types.NetworkParty,
			Amount: &types.FinancialAmount{
				Asset:  e.asset,
				Amount: fees.LiquidityFee.Clone(),
			},
			MinAmount: num.UintZero(),
			Type:      types.TransferTypeLiquidityFeePay,
		},
		{
			Owner: types.NetworkParty,
			Amount: &types.FinancialAmount{
				Asset:  e.asset,
				Amount: fees.BuyBackFee.Clone(),
			},
			MinAmount: num.UintZero(),
			Type:      types.TransferTypeBuyBackFeePay,
		},
		{
			Owner: types.NetworkParty,
			Amount: &types.FinancialAmount{
				Asset:  e.asset,
				Amount: fees.TreasuryFee.Clone(),
			},
			MinAmount: num.UintZero(),
			Type:      types.TransferTypeTreasuryPay,
		},
	}
	if !fees.HighVolumeMakerFee.IsZero() {
		transfers = append(transfers, &types.Transfer{
			Owner: types.NetworkParty,
			Amount: &types.FinancialAmount{
				Asset:  e.asset,
				Amount: fees.HighVolumeMakerFee.Clone(),
			},
			MinAmount: num.UintZero(),
			Type:      types.TransferTypeHighMakerRebatePay,
		})
	}

	return transfers, num.Sum(fees.MakerFee, fees.InfrastructureFee, fees.LiquidityFee, fees.BuyBackFee, fees.HighVolumeMakerFee)
}

func (e *Engine) applyDiscountsAndRewards(taker string, maker string, tradeValueForFeePurposes num.Decimal, fees *types.Fee, referral ReferralDiscountRewardService, volumeDiscount VolumeDiscountService, volumeRebate VolumeRebateService) (*types.Fee, *types.ReferrerReward) {
	referralDiscountFactors := referral.ReferralDiscountFactorsForParty(types.PartyID(taker))
	volumeDiscountFactors := volumeDiscount.VolumeDiscountFactorForParty(types.PartyID(taker))
	highVolumeMakerFee := volumeRebate.VolumeRebateFactorForParty(types.PartyID(maker)).Mul(tradeValueForFeePurposes)
	highVolumeMakerFeeI, _ := num.UintFromDecimal(highVolumeMakerFee)

	mf := fees.MakerFee.Clone()
	inf := fees.InfrastructureFee.Clone()
	lf := fees.LiquidityFee.Clone()

	// calculate referral discounts
	referralMakerDiscount, _ := num.UintFromDecimal(mf.ToDecimal().Mul(referralDiscountFactors.Maker).Floor())
	referralInfDiscount, _ := num.UintFromDecimal(inf.ToDecimal().Mul(referralDiscountFactors.Infra).Floor())
	referralLfDiscount, _ := num.UintFromDecimal(lf.ToDecimal().Mul(referralDiscountFactors.Liquidity).Floor())

	// apply referral discounts
	mf = mf.Sub(mf, referralMakerDiscount)
	inf = inf.Sub(inf, referralInfDiscount)
	lf = lf.Sub(lf, referralLfDiscount)

	// calculate volume discounts
	volumeMakerDiscount, _ := num.UintFromDecimal(mf.ToDecimal().Mul(volumeDiscountFactors.Maker).Floor())
	volumeInfDiscount, _ := num.UintFromDecimal(inf.ToDecimal().Mul(volumeDiscountFactors.Infra).Floor())
	volumeLfDiscount, _ := num.UintFromDecimal(lf.ToDecimal().Mul(volumeDiscountFactors.Liquidity).Floor())

	var rebateDiscountFactor num.Decimal
	bbAndTreasury := num.Sum(fees.BuyBackFee, fees.TreasuryFee).ToDecimal()
	if !bbAndTreasury.IsZero() {
		rebateDiscountFactor = num.DecimalOne().Sub(highVolumeMakerFee.Div(bbAndTreasury))
	}

	treasuryFee, _ := num.UintFromDecimal(fees.TreasuryFee.ToDecimal().Mul(rebateDiscountFactor))
	buyBackFee, _ := num.UintFromDecimal(fees.BuyBackFee.ToDecimal().Mul(rebateDiscountFactor))

	// apply volume discounts
	mf = mf.Sub(mf, volumeMakerDiscount)
	inf = inf.Sub(inf, volumeInfDiscount)
	lf = lf.Sub(lf, volumeLfDiscount)

	f := &types.Fee{
		HighVolumeMakerFee:                highVolumeMakerFeeI,
		MakerFee:                          mf,
		LiquidityFee:                      lf,
		InfrastructureFee:                 inf,
		BuyBackFee:                        buyBackFee.Clone(),
		TreasuryFee:                       treasuryFee.Clone(),
		MakerFeeVolumeDiscount:            volumeMakerDiscount,
		InfrastructureFeeVolumeDiscount:   volumeInfDiscount,
		LiquidityFeeVolumeDiscount:        volumeLfDiscount,
		MakerFeeReferrerDiscount:          referralMakerDiscount,
		InfrastructureFeeReferrerDiscount: referralInfDiscount,
		LiquidityFeeReferrerDiscount:      referralLfDiscount,
	}

	e.feesStats.RegisterRefereeDiscount(
		taker,
		num.Sum(
			referralMakerDiscount,
			referralInfDiscount,
			referralLfDiscount,
		),
	)

	e.feesStats.RegisterVolumeDiscount(
		taker,
		num.Sum(
			volumeMakerDiscount,
			volumeInfDiscount,
			volumeLfDiscount,
		),
	)

	// calculate rewards
	factors := referral.RewardsFactorsMultiplierAppliedForParty(types.PartyID(taker))
	if factors.IsEmpty() {
		return f, nil
	}

	referrerReward := types.NewReferrerReward()

	referrerReward.MakerFeeReferrerReward, _ = num.UintFromDecimal(factors.Maker.Mul(mf.ToDecimal()).Floor())
	referrerReward.InfrastructureFeeReferrerReward, _ = num.UintFromDecimal(factors.Infra.Mul(inf.ToDecimal()).Floor())
	referrerReward.LiquidityFeeReferrerReward, _ = num.UintFromDecimal(factors.Liquidity.Mul(lf.ToDecimal()).Floor())

	mf = mf.Sub(mf, referrerReward.MakerFeeReferrerReward)
	inf = inf.Sub(inf, referrerReward.InfrastructureFeeReferrerReward)
	lf = lf.Sub(lf, referrerReward.LiquidityFeeReferrerReward)

	referrer, err := referral.GetReferrer(types.PartyID(taker))
	if err != nil {
		e.log.Error("could not load referrer from taker of trade", logging.PartyID(taker))
	} else {
		e.feesStats.RegisterReferrerReward(
			string(referrer),
			taker,
			num.Sum(
				referrerReward.MakerFeeReferrerReward,
				referrerReward.InfrastructureFeeReferrerReward,
				referrerReward.LiquidityFeeReferrerReward,
			),
		)
	}

	f.MakerFee = mf
	f.InfrastructureFee = inf
	f.LiquidityFee = lf
	return f, referrerReward
}

func (e *Engine) getAuctionModeFeesAndTransfers(t *types.Trade, referral ReferralDiscountRewardService, volumeDiscount VolumeDiscountService, volumeRebate VolumeRebateService) (*types.Fee, *types.Fee, []*types.Transfer) {
	fee := e.calculateAuctionModeFees(t)
	// in auction there is no maker so there is no rebate, so passing 0 as the trade value
	buyerFees, buyerReferrerRewards := e.applyDiscountsAndRewards(t.Buyer, t.Buyer, num.DecimalZero(), fee, referral, volumeDiscount, volumeRebate)
	sellerFees, sellerReferrerRewards := e.applyDiscountsAndRewards(t.Seller, t.Seller, num.DecimalZero(), fee, referral, volumeDiscount, volumeRebate)

	transfers := make([]*types.Transfer, 0, 12)
	transfers = append(transfers,
		e.getAuctionModeFeeTransfers(
			sellerFees.InfrastructureFee, sellerFees.LiquidityFee, sellerFees.BuyBackFee, sellerFees.TreasuryFee, t.Seller)...)
	transfers = append(transfers,
		e.getAuctionModeFeeTransfers(
			buyerFees.InfrastructureFee, buyerFees.LiquidityFee, buyerFees.BuyBackFee, buyerFees.TreasuryFee, t.Buyer)...)

	if buyerReferrerRewards != nil {
		referrerParty, _ := referral.GetReferrer(types.PartyID(t.Buyer))
		transfers = append(transfers,
			e.getAuctionModeFeeReferrerRewardTransfers(
				num.Sum(buyerReferrerRewards.InfrastructureFeeReferrerReward, buyerReferrerRewards.LiquidityFeeReferrerReward), t.Buyer, string(referrerParty))...)
	}

	if sellerReferrerRewards != nil {
		referrerParty, _ := referral.GetReferrer(types.PartyID(t.Seller))
		transfers = append(transfers,
			e.getAuctionModeFeeReferrerRewardTransfers(
				num.Sum(sellerReferrerRewards.InfrastructureFeeReferrerReward, sellerReferrerRewards.LiquidityFeeReferrerReward), t.Seller, string(referrerParty))...)
	}

	return buyerFees, sellerFees, transfers
}

func (e *Engine) calculateContinuousModeFees(trade *types.Trade) *types.Fee {
	size := num.NewUint(trade.Size)
	// multiply by size
	total := size.Mul(trade.Price, size).ToDecimal().Div(e.positionFactor)
	mf, _ := num.UintFromDecimal(total.Mul(e.f.makerFee).Ceil())
	inf, _ := num.UintFromDecimal(total.Mul(e.f.infrastructureFee).Ceil())
	lf, _ := num.UintFromDecimal(total.Mul(e.f.liquidityFee).Ceil())
	bbf, _ := num.UintFromDecimal(total.Mul(e.f.buyBackFee).Ceil())
	tf, _ := num.UintFromDecimal(total.Mul(e.f.treasuryFee).Ceil())
	return &types.Fee{
		MakerFee:           mf,
		InfrastructureFee:  inf,
		LiquidityFee:       lf,
		BuyBackFee:         bbf,
		TreasuryFee:        tf,
		HighVolumeMakerFee: num.UintZero(),
	}
}

func (e *Engine) calculateAuctionModeFees(trade *types.Trade) *types.Fee {
	fee := e.calculateContinuousModeFees(trade)
	two := num.DecimalFromInt64(2)
	inf, _ := num.UintFromDecimal(fee.InfrastructureFee.ToDecimal().Div(two).Ceil())
	lf, _ := num.UintFromDecimal(fee.LiquidityFee.ToDecimal().Div(two).Ceil())
	bbf, _ := num.UintFromDecimal(fee.BuyBackFee.ToDecimal().Div(two).Ceil())
	tf, _ := num.UintFromDecimal(fee.TreasuryFee.ToDecimal().Div(two).Ceil())
	return &types.Fee{
		MakerFee:          num.UintZero(),
		InfrastructureFee: inf,
		LiquidityFee:      lf,
		TreasuryFee:       tf,
		BuyBackFee:        bbf,
	}
}

func (e *Engine) getAuctionModeFeeReferrerRewardTransfers(reward *num.Uint, p, referrer string) []*types.Transfer {
	return []*types.Transfer{
		{
			Owner: p,
			Amount: &types.FinancialAmount{
				Amount: reward.Clone(),
				Asset:  e.asset,
			},
			Type:      types.TransferTypeFeeReferrerRewardPay,
			MinAmount: reward.Clone(),
		}, {
			Owner: referrer,
			Amount: &types.FinancialAmount{
				Amount: reward.Clone(),
				Asset:  e.asset,
			},
			Type:      types.TransferTypeFeeReferrerRewardDistribute,
			MinAmount: reward.Clone(),
		},
	}
}

func (e *Engine) getAuctionModeFeeTransfers(infraFee, liquiFee, buyBackFee, treasuryFee *num.Uint, p string) []*types.Transfer {
	// we return both transfer for the party in a slice
	// always the infrastructure fee first
	return []*types.Transfer{
		{
			Owner: p,
			Amount: &types.FinancialAmount{
				Asset:  e.asset,
				Amount: infraFee.Clone(),
			},
			Type: types.TransferTypeInfrastructureFeePay,
		},
		{
			Owner: p,
			Amount: &types.FinancialAmount{
				Asset:  e.asset,
				Amount: liquiFee.Clone(),
			},
			Type: types.TransferTypeLiquidityFeePay,
		},
		{
			Owner: p,
			Amount: &types.FinancialAmount{
				Asset:  e.asset,
				Amount: buyBackFee.Clone(),
			},
			Type: types.TransferTypeBuyBackFeePay,
		},
		{
			Owner: p,
			Amount: &types.FinancialAmount{
				Asset:  e.asset,
				Amount: treasuryFee.Clone(),
			},
			Type: types.TransferTypeTreasuryPay,
		},
	}
}

type feesTransfer struct {
	totalFeesAmountsPerParty map[string]*num.Uint
	transfers                []*types.Transfer
}

func (f *feesTransfer) TotalFeesAmountPerParty() map[string]*num.Uint {
	ret := make(map[string]*num.Uint, len(f.totalFeesAmountsPerParty))
	for k, v := range f.totalFeesAmountsPerParty {
		ret[k] = v.Clone()
	}
	return ret
}
func (f *feesTransfer) Transfers() []*types.Transfer { return f.transfers }

func (e *Engine) OnFeeFactorsMakerFeeUpdate(f num.Decimal) {
	e.feeCfg.Factors.MakerFee = f
	e.f.makerFee = f
}

func (e *Engine) OnFeeFactorsBuyBackFeeUpdate(f num.Decimal) {
	e.feeCfg.Factors.BuyBackFee = f
	e.f.buyBackFee = f
}

func (e *Engine) OnFeeFactorsTreasuryFeeUpdate(f num.Decimal) {
	e.feeCfg.Factors.TreasuryFee = f
	e.f.treasuryFee = f
}

func (e *Engine) OnFeeFactorsInfrastructureFeeUpdate(f num.Decimal) {
	e.feeCfg.Factors.InfrastructureFee = f
	e.f.infrastructureFee = f
}

func (e *Engine) GetLiquidityFee() num.Decimal {
	return e.f.liquidityFee
}
