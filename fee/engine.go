package fee

import (
	"errors"
	"sort"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

var (
	ErrEmptyTrades      = errors.New("empty trades slice sent to fees")
	ErrInvalidFeeFactor = errors.New("fee factors must be positive")
)

type Engine struct {
	log *logging.Logger
	cfg Config

	asset          string
	feeCfg         types.Fees
	f              factors
	positionFactor num.Decimal
}

type factors struct {
	makerFee          num.Decimal
	infrastructureFee num.Decimal
	liquidityFee      num.Decimal
}

func New(log *logging.Logger, cfg Config, feeCfg types.Fees, asset string, positionFactor num.Decimal) (*Engine, error) {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	e := &Engine{
		log:            log,
		feeCfg:         feeCfg,
		cfg:            cfg,
		asset:          asset,
		positionFactor: positionFactor,
	}
	return e, e.UpdateFeeFactors(e.feeCfg)
}

// ReloadConf is used in order to reload the internal configuration of
// the of the fee engine.
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
	if fees.Factors.MakerFee.IsNegative() || fees.Factors.InfrastructureFee.IsNegative() || fees.Factors.LiquidityFee.IsNegative() {
		return ErrInvalidFeeFactor
	}
	e.f.makerFee = fees.Factors.MakerFee
	e.f.infrastructureFee = fees.Factors.InfrastructureFee
	// not sure we need the IsPositive check here, that ought to be validation
	if !fees.Factors.LiquidityFee.IsZero() && fees.Factors.LiquidityFee.IsPositive() {
		e.f.liquidityFee = fees.Factors.LiquidityFee
	}

	e.feeCfg = fees
	return nil
}

func (e *Engine) SetLiquidityFee(v num.Decimal) {
	e.f.liquidityFee = v
}

// CalculateForContinuousMode calculate the fee for
// trades which were produced from a market running in
// in continuous trading mode.
// A single FeesTransfer is produced here as all fees
// are paid by the aggressive order.
func (e *Engine) CalculateForContinuousMode(
	trades []*types.Trade,
) (events.FeesTransfer, error) {
	if len(trades) <= 0 {
		return nil, ErrEmptyTrades
	}

	var (
		aggressor, maker             string
		totalFeeAmount               = num.Zero()
		totalInfrastructureFeeAmount = num.Zero()
		totalLiquidityFeeAmount      = num.Zero()
		// we allocate the len of the trades + 2
		// len(trade) = number of makerFee + 1 infra fee + 1 liquidity fee
		transfers     = make([]*types.Transfer, 0, (len(trades)*2)+2)
		transfersRecv = make([]*types.Transfer, 0, len(trades)+2)
	)

	for _, v := range trades {
		fee := e.calculateContinuousModeFees(v)
		switch v.Aggressor {
		case types.SideBuy:
			v.BuyerFee = fee
			v.SellerFee = types.NewFee()
			aggressor = v.Buyer
			maker = v.Seller
		case types.SideSell:
			v.SellerFee = fee
			v.BuyerFee = types.NewFee()
			aggressor = v.Seller
			maker = v.Buyer
		}

		totalFeeAmount.AddSum(fee.InfrastructureFee, fee.LiquidityFee, fee.MakerFee)
		totalInfrastructureFeeAmount.AddSum(fee.InfrastructureFee)
		totalLiquidityFeeAmount.AddSum(fee.LiquidityFee)

		// create a transfer for the aggressor
		transfers = append(transfers, &types.Transfer{
			Owner: aggressor,
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
	}

	// now create transfer for the infrastructure
	transfers = append(transfers, &types.Transfer{
		Owner: aggressor,
		Amount: &types.FinancialAmount{
			Asset:  e.asset,
			Amount: totalInfrastructureFeeAmount,
		},
		Type: types.TransferTypeInfrastructureFeePay,
	})
	// now create transfer for the liquidity
	transfers = append(transfers, &types.Transfer{
		Owner: aggressor,
		Amount: &types.FinancialAmount{
			Asset:  e.asset,
			Amount: totalLiquidityFeeAmount,
		},
		Type: types.TransferTypeLiquidityFeePay,
	})

	return &feesTransfer{
		totalFeesAmountsPerParty: map[string]*num.Uint{aggressor: totalFeeAmount, maker: num.Zero()},
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
		fee, newTransfers := e.getAuctionModeFeesAndTransfers(v)
		totalFee := num.Sum(fee.InfrastructureFee, fee.LiquidityFee)
		transfers = append(transfers, newTransfers...)

		// increase the total fee for the parties
		if sellerTotalFee, ok := totalFeesAmounts[v.Seller]; !ok {
			totalFeesAmounts[v.Seller] = totalFee.Clone()
		} else {
			sellerTotalFee.AddSum(totalFee)
		}
		if buyerTotalFee, ok := totalFeesAmounts[v.Buyer]; !ok {
			totalFeesAmounts[v.Buyer] = totalFee.Clone()
		} else {
			buyerTotalFee.AddSum(totalFee)
		}

		v.BuyerFee = fee
		v.SellerFee = fee
	}

	return &feesTransfer{
		totalFeesAmountsPerParty: totalFeesAmounts,
		transfers:                transfers,
	}, nil
}

// CalculateForFrequentBatchesAuctionMode calculate the fee for
// trades which were produced from a market running in
// in auction trading mode.
// A list FeesTransfer is produced each containing fees transfer from a
// single party.
func (e *Engine) CalculateForFrequentBatchesAuctionMode(
	trades []*types.Trade,
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
			var fee *types.Fee
			fee, newTransfers = e.getAuctionModeFeesAndTransfers(v)
			// clone the fees, obviously
			v.SellerFee, v.BuyerFee = fee, fee.Clone()
			totalFee := num.Sum(fee.InfrastructureFee, fee.LiquidityFee)
			sellerTotalFee, buyerTotalFee = totalFee, totalFee.Clone()
		} else {
			// set the aggressor to be the side of the party
			// entering the later auction
			v.Aggressor = types.SideSell
			if v.BuyerAuctionBatch > v.SellerAuctionBatch {
				v.Aggressor = types.SideBuy
			}
			// fees are being assign to the trade directly
			// no need to do add them there as well
			ftrnsfr, _ := e.CalculateForContinuousMode([]*types.Trade{v})
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

func (e *Engine) CalculateFeeForPositionResolution(
	// the trade from the good parties which 0 out the network order
	trades []*types.Trade,
	// the positions of the parties being closed out.
	closedMPs []events.MarketPosition,
) (events.FeesTransfer, map[string]*types.Fee) {
	var (
		totalFeesAmounts = map[string]*num.Uint{}
		partiesFees      = map[string]*types.Fee{}
		// this is the share of each party to be paid
		partiesShare     = map[string]*feeShare{}
		totalAbsolutePos uint64
		transfers        = []*types.Transfer{}
	)

	// first calculate the share of all distressedParties
	for _, v := range closedMPs {
		size := v.Size()
		if size < 0 {
			size = -size
		}
		totalAbsolutePos += uint64(size)
		partiesShare[v.Party()] = &feeShare{pos: uint64(size)}

		// while we are at it, we initial the map of all fees per party
		partiesFees[v.Party()] = types.NewFee()
	}

	// no we accumulated all the absolute position, we
	// will get the share of each party
	for _, v := range partiesShare {
		v.share = num.DecimalFromInt64(int64(v.pos)).Div(num.DecimalFromInt64(int64(totalAbsolutePos)))
	}

	// now we have the share of each distressed parties
	// we can iterate over the trades, and make the transfers
	for _, t := range trades {
		// continuous trading fees apply here
		// the we'll split them in between all parties
		fees := e.calculateContinuousModeFees(t)

		// lets fine which side is the good party
		goodParty := t.Buyer
		t.SellerFee = fees
		if goodParty == "network" {
			goodParty = t.Seller
			t.SellerFee = types.NewFee()
			t.BuyerFee = fees.Clone()
		}

		// now we iterate over all parties,
		// and create a pay for each distressed parties
		for _, v := range closedMPs {
			partyTransfers, fees, feesAmount := e.getPositionResolutionFeesTransfers(
				v.Party(), partiesShare[v.Party()].share, fees)

			if prevTotalFee, ok := totalFeesAmounts[v.Party()]; !ok {
				totalFeesAmounts[v.Party()] = feesAmount.Clone()
			} else {
				prevTotalFee.AddSum(feesAmount)
			}
			transfers = append(transfers, partyTransfers...)

			// increase the party full fees
			pf := partiesFees[v.Party()]
			pf.MakerFee.AddSum(fees.MakerFee)
			pf.InfrastructureFee.AddSum(fees.InfrastructureFee)
			pf.LiquidityFee.AddSum(fees.LiquidityFee)
			partiesFees[v.Party()] = pf
		}

		// then 1 receive transfer for the good party
		transfers = append(transfers, &types.Transfer{
			Owner: goodParty,
			Amount: &types.FinancialAmount{
				Asset:  e.asset,
				Amount: fees.MakerFee,
			},
			Type: types.TransferTypeMakerFeeReceive,
		})
	}

	// calculate the
	return &feesTransfer{
		totalFeesAmountsPerParty: totalFeesAmounts,
		transfers:                transfers,
	}, partiesFees
}

// BuildLiquidityFeeDistributionTransfer returns the set of transfers that will
// be used by the collateral engine to distribute the fees.  As shares are
// represented in float64 and fees are uint64, shares are floored and the
// remainder is assigned to the last party on the share map. Note that the map
// is sorted lexicographically to keep determinism.
func (e *Engine) BuildLiquidityFeeDistributionTransfer(shares map[string]num.Decimal, acc *types.Account) events.FeesTransfer {
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
		ft.totalFeesAmountsPerParty[key] = num.Zero()
	}
	sort.Strings(keys)

	var floored num.Decimal
	for _, key := range keys {
		share := shares[key]
		cs := acc.Balance.ToDecimal().Mul(share).Floor()
		floored = floored.Add(cs)

		amount, _ := num.UintFromDecimal(cs)
		// populate the return value
		ft.totalFeesAmountsPerParty[key].AddSum(amount)
		ft.transfers = append(ft.transfers, &types.Transfer{
			Owner: key,
			Amount: &types.FinancialAmount{
				Amount: amount.Clone(),
				Asset:  acc.Asset,
			},
			MinAmount: amount.Clone(),
			Type:      types.TransferTypeLiquidityFeeDistribute,
		})
	}

	// last is the party who will get the remaining from ceil
	last := keys[len(keys)-1]
	diff, _ := num.UintFromDecimal(acc.Balance.ToDecimal().Sub(floored))
	ft.totalFeesAmountsPerParty[last].AddSum(diff)
	ft.transfers[len(ft.transfers)-1].Amount.Amount.AddSum(diff)

	return ft
}

// this will calculate the transfer the distressed party needs
// to do.
func (e *Engine) getPositionResolutionFeesTransfers(
	party string, share num.Decimal, fees *types.Fee,
) ([]*types.Transfer, *types.Fee, *num.Uint) {
	makerFee, _ := num.UintFromDecimal(fees.MakerFee.ToDecimal().Mul(share).Ceil())
	infraFee, _ := num.UintFromDecimal(fees.InfrastructureFee.ToDecimal().Mul(share).Ceil())
	liquiFee, _ := num.UintFromDecimal(fees.LiquidityFee.ToDecimal().Mul(share).Ceil())

	return []*types.Transfer{
			{
				Owner: party,
				Amount: &types.FinancialAmount{
					Asset:  e.asset,
					Amount: makerFee.Clone(),
				},
				Type: types.TransferTypeMakerFeePay,
			},
			{
				Owner: party,
				Amount: &types.FinancialAmount{
					Asset:  e.asset,
					Amount: infraFee.Clone(),
				},
				Type: types.TransferTypeInfrastructureFeePay,
			},
			{
				Owner: party,
				Amount: &types.FinancialAmount{
					Asset:  e.asset,
					Amount: liquiFee.Clone(),
				},
				Type: types.TransferTypeLiquidityFeePay,
			},
		},
		&types.Fee{
			MakerFee:          makerFee,
			InfrastructureFee: infraFee,
			LiquidityFee:      liquiFee,
		}, num.Sum(makerFee, infraFee, liquiFee)
}

type feeShare struct {
	// the absolute position of the party which had to be recovered
	pos uint64
	// the share out of the total volume
	share num.Decimal
}

func (e *Engine) getAuctionModeFeesAndTransfers(t *types.Trade) (*types.Fee, []*types.Transfer) {
	fee := e.calculateAuctionModeFees(t)
	transfers := make([]*types.Transfer, 0, 4)
	transfers = append(transfers,
		e.getAuctionModeFeeTransfers(
			fee.InfrastructureFee, fee.LiquidityFee, t.Seller)...)
	transfers = append(transfers,
		e.getAuctionModeFeeTransfers(
			fee.InfrastructureFee, fee.LiquidityFee, t.Buyer)...)
	return fee, transfers
}

func (e *Engine) calculateContinuousModeFees(trade *types.Trade) *types.Fee {
	size := num.NewUint(trade.Size)
	// multiply by size
	total := size.Mul(trade.Price, size).ToDecimal().Div(e.positionFactor)
	mf, _ := num.UintFromDecimal(total.Mul(e.f.makerFee).Ceil())
	inf, _ := num.UintFromDecimal(total.Mul(e.f.infrastructureFee).Ceil())
	lf, _ := num.UintFromDecimal(total.Mul(e.f.liquidityFee).Ceil())
	return &types.Fee{
		MakerFee:          mf,
		InfrastructureFee: inf,
		LiquidityFee:      lf,
	}
}

func (e *Engine) calculateAuctionModeFees(trade *types.Trade) *types.Fee {
	fee := e.calculateContinuousModeFees(trade)
	two := num.DecimalFromInt64(2)
	inf, _ := num.UintFromDecimal(fee.InfrastructureFee.ToDecimal().Div(two).Ceil())
	lf, _ := num.UintFromDecimal(fee.LiquidityFee.ToDecimal().Div(two).Ceil())
	return &types.Fee{
		MakerFee:          num.Zero(),
		InfrastructureFee: inf,
		LiquidityFee:      lf,
	}
}

func (e *Engine) getAuctionModeFeeTransfers(infraFee, liquiFee *num.Uint, p string) []*types.Transfer {
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

func (e *Engine) OnFeeFactorsInfrastructureFeeUpdate(f num.Decimal) {
	e.feeCfg.Factors.InfrastructureFee = f
	e.f.infrastructureFee = f
}

func (e *Engine) GetLiquidityFee() num.Decimal {
	return e.f.liquidityFee
}
