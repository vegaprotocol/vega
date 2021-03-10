package fee

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
)

var (
	ErrEmptyTrades = errors.New("empty trades slice sent to fees")
)

type Engine struct {
	log *logging.Logger
	cfg Config

	asset  string
	feeCfg types.Fees
	f      factors
}

type factors struct {
	makerFee          float64
	infrastructureFee float64
	liquidityFee      float64
}

func New(log *logging.Logger, cfg Config, feeCfg types.Fees, asset string) (*Engine, error) {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	e := &Engine{
		log:    log,
		feeCfg: feeCfg,
		cfg:    cfg,
		asset:  asset,
	}
	return e, e.UpdateFeeFactors(e.feeCfg)
}

// ReloadConf is used in order to reload the internal configuration of
// the of the fee engine
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
	f, err := strconv.ParseFloat(fees.Factors.MakerFee, 64)
	if err != nil {
		e.log.Error("unable to load makerfee", logging.Error(err))
		return err
	}
	e.f.makerFee = f
	f, err = strconv.ParseFloat(fees.Factors.InfrastructureFee, 64)
	if err != nil {
		e.log.Error("unable to load infrastructurefee", logging.Error(err))
		return err
	}
	e.f.infrastructureFee = f

	if err := e.SetLiquidityFee(fees.Factors.LiquidityFee); err != nil {
		e.log.Error("unable to load liquidityfee", logging.Error(err))
		return err
	}

	e.feeCfg = fees
	return nil
}

func (e *Engine) SetLiquidityFee(v string) error {
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return err
	}
	e.f.liquidityFee = f
	return nil
}

// CalculateForContinuousMode calculate the fee for
// trades which were produced from a market running in
// in continuous trading mode.
// A single FeesTransfer is produced here as all fees
// are paid by the agressive order
func (e *Engine) CalculateForContinuousMode(
	trades []*types.Trade,
) (events.FeesTransfer, error) {
	if len(trades) <= 0 {
		return nil, ErrEmptyTrades
	}

	var (
		aggressor                    string
		maker                        string
		totalFeeAmount               uint64
		totalInfrastructureFeeAmount uint64
		totalLiquidityFeeAmount      uint64
		// we allocate the len of the trades + 2
		// len(trade) = number of makerFee + 1 infra fee + 1 liquidity fee
		transfers     = make([]*types.Transfer, 0, (len(trades)*2)+2)
		transfersRecv = make([]*types.Transfer, 0, len(trades)+2)
	)

	for _, v := range trades {
		fee := e.calculateContinuousModeFees(v)
		switch v.Aggressor {
		case types.Side_SIDE_BUY:
			v.BuyerFee = fee
			v.SellerFee = &types.Fee{}
			aggressor = v.Buyer
			maker = v.Seller
		case types.Side_SIDE_SELL:
			v.SellerFee = fee
			v.BuyerFee = &types.Fee{}
			aggressor = v.Seller
			maker = v.Buyer
		}

		totalFeeAmount += fee.InfrastructureFee + fee.LiquidityFee + fee.MakerFee
		totalInfrastructureFeeAmount += fee.InfrastructureFee
		totalLiquidityFeeAmount += fee.LiquidityFee

		// create a transfer for the aggressor
		transfers = append(transfers, &types.Transfer{
			Owner: aggressor,
			Amount: &types.FinancialAmount{
				Asset:  e.asset,
				Amount: fee.MakerFee,
			},
			Type: types.TransferType_TRANSFER_TYPE_MAKER_FEE_PAY,
		})
		// create a transfer for the maker
		transfersRecv = append(transfersRecv, &types.Transfer{
			Owner: maker,
			Amount: &types.FinancialAmount{
				Asset:  e.asset,
				Amount: fee.MakerFee,
			},
			Type: types.TransferType_TRANSFER_TYPE_MAKER_FEE_RECEIVE,
		})
	}

	// now create transfer for the infrastructure
	transfers = append(transfers, &types.Transfer{
		Owner: aggressor,
		Amount: &types.FinancialAmount{
			Asset:  e.asset,
			Amount: totalInfrastructureFeeAmount,
		},
		Type: types.TransferType_TRANSFER_TYPE_INFRASTRUCTURE_FEE_PAY,
	})
	// now create transfer for the liquidity
	transfers = append(transfers, &types.Transfer{
		Owner: aggressor,
		Amount: &types.FinancialAmount{
			Asset:  e.asset,
			Amount: totalLiquidityFeeAmount,
		},
		Type: types.TransferType_TRANSFER_TYPE_LIQUIDITY_FEE_PAY,
	})

	return &feesTransfer{
		totalFeesAmountsPerParty: map[string]uint64{aggressor: totalFeeAmount},
		transfers:                append(transfers, transfersRecv...),
	}, nil
}

// CalculateForAuctionMode calculate the fee for
// trades which were produced from a market running in
// in auction trading mode.
// A list FeesTransfer is produced each containing fees transfer from a
// single trader
func (e *Engine) CalculateForAuctionMode(
	trades []*types.Trade,
) (events.FeesTransfer, error) {
	if len(trades) <= 0 {
		return nil, ErrEmptyTrades
	}
	var (
		totalFeesAmounts = map[string]uint64{}
		// we allocate for len of trades *4 as all trades generate
		// 2 fees per party
		transfers = make([]*types.Transfer, 0, len(trades)*4)
	)

	// we iterate over all trades
	// for each trades both party needs to pay half of the fees
	// no maker fees are to be paid here.
	for _, v := range trades {
		fee, newTransfers := e.getAuctionModeFeesAndTransfers(v)
		totalFee := fee.InfrastructureFee + fee.LiquidityFee
		transfers = append(transfers, newTransfers...)

		// increase the total fee for the parties
		if sellerTotalFee, ok := totalFeesAmounts[v.Seller]; !ok {
			totalFeesAmounts[v.Seller] = totalFee
		} else {
			totalFeesAmounts[v.Seller] = sellerTotalFee + totalFee
		}
		if buyerTotalFee, ok := totalFeesAmounts[v.Buyer]; !ok {
			totalFeesAmounts[v.Buyer] = totalFee
		} else {
			totalFeesAmounts[v.Buyer] = buyerTotalFee + totalFee
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
// single trader
func (e *Engine) CalculateForFrequentBatchesAuctionMode(
	trades []*types.Trade,
) (events.FeesTransfer, error) {
	if len(trades) <= 0 {
		return nil, ErrEmptyTrades
	}

	var (
		totalFeesAmounts = map[string]uint64{}
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
			sellerTotalFee, buyerTotalFee uint64
			newTransfers                  []*types.Transfer
		)
		// we are in the same auction, normal auction fees applies
		if v.BuyerAuctionBatch == v.SellerAuctionBatch {
			var fee *types.Fee
			fee, newTransfers = e.getAuctionModeFeesAndTransfers(v)
			v.SellerFee, v.BuyerFee = fee, fee
			totalFee := fee.InfrastructureFee + fee.LiquidityFee
			sellerTotalFee, buyerTotalFee = totalFee, totalFee

		} else {
			// set the aggressor to be the side of the trader
			// entering the later auction
			v.Aggressor = types.Side_SIDE_SELL
			if v.BuyerAuctionBatch > v.SellerAuctionBatch {
				v.Aggressor = types.Side_SIDE_BUY
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
			totalFeesAmounts[v.Seller] = sellerTotalFee
		} else {
			totalFeesAmounts[v.Seller] = prevTotalFee + sellerTotalFee
		}
		if prevTotalFee, ok := totalFeesAmounts[v.Buyer]; !ok {
			totalFeesAmounts[v.Buyer] = buyerTotalFee
		} else {
			totalFeesAmounts[v.Buyer] = prevTotalFee + buyerTotalFee
		}
	}

	return &feesTransfer{
		totalFeesAmountsPerParty: totalFeesAmounts,
		transfers:                transfers,
	}, nil
}

func (e *Engine) CalculateFeeForPositionResolution(
	// the trade from the good traders which 0 out the networl order
	trades []*types.Trade,
	// the positions of the traders being closed out.
	closedMPs []events.MarketPosition,
) (events.FeesTransfer, map[string]*types.Fee, error) {
	var (
		totalFeesAmounts = map[string]uint64{}
		partiesFees      = map[string]*types.Fee{}
		// this is the share of each party to be paid
		partiesShare     = map[string]*feeShare{}
		totalAbsolutePos uint64
		transfers        = []*types.Transfer{}
	)

	// first calculate the share of all distressedTraders
	for _, v := range closedMPs {
		var size = v.Size()
		if size < 0 {
			size = -size
		}
		totalAbsolutePos += uint64(size)
		partiesShare[v.Party()] = &feeShare{pos: uint64(size)}

		// while we are at it, we initial the map of all fees per party
		partiesFees[v.Party()] = &types.Fee{}
	}

	// no we accumulated all the absolute position, we
	// will get the share of each party
	for _, v := range partiesShare {
		v.share = float64(v.pos) / float64(totalAbsolutePos)
	}

	// now we have the share of each distressed parties
	// we can iterate over the trades, and make the transfers
	for _, t := range trades {
		// continuous trading fees apply here
		// the we'll split them in between all parties
		fees := e.calculateContinuousModeFees(t)

		// lets fine which side is the good party
		var goodParty = t.Buyer
		t.SellerFee = fees
		if goodParty == "network" {
			goodParty = t.Seller
			t.SellerFee = &types.Fee{}
			t.BuyerFee = fees
		}

		// now we iterate over all parties,
		// and create a pay for each distressed parties
		for _, v := range closedMPs {
			partyTransfers, fees, feesAmount := e.getPositionResolutionFeesTransfers(
				v.Party(), partiesShare[v.Party()].share, fees)

			if prevTotalFee, ok := totalFeesAmounts[v.Party()]; !ok {
				totalFeesAmounts[v.Party()] = feesAmount
			} else {
				totalFeesAmounts[v.Party()] = prevTotalFee + feesAmount
			}
			transfers = append(transfers, partyTransfers...)

			// increase the party full fees
			pf := partiesFees[v.Party()]
			pf.MakerFee += fees.MakerFee
			pf.InfrastructureFee += fees.InfrastructureFee
			pf.LiquidityFee += fees.LiquidityFee
			partiesFees[v.Party()] = pf

		}

		// then 1 receive transfer for the good party
		transfers = append(transfers, &types.Transfer{
			Owner: goodParty,
			Amount: &types.FinancialAmount{
				Asset:  e.asset,
				Amount: fees.MakerFee,
			},
			Type: types.TransferType_TRANSFER_TYPE_MAKER_FEE_RECEIVE,
		})

	}

	// calculate the
	return &feesTransfer{
		totalFeesAmountsPerParty: totalFeesAmounts,
		transfers:                transfers,
	}, partiesFees, nil
}

// BuildLiquidityFeeDistributionTransfer returns the set of transfers that will
// be used by the collateral engine to distribute the fees.  As shares are
// represented in float64 and fees are uint64, shares are floored and the
// remainder is assigned to the last party on the share map. Note that the map
// is sorted lexicographically to keep determinism.
func (e *Engine) BuildLiquidityFeeDistributionTransfer(shares map[string]float64, acc *types.Account) events.FeesTransfer {
	if len(shares) == 0 {
		return nil
	}

	ft := &feesTransfer{
		totalFeesAmountsPerParty: map[string]uint64{},
		transfers:                make([]*types.Transfer, 0, len(shares)),
	}

	// Get all the map keys
	keys := make([]string, 0, len(shares))

	for key := range shares {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var floored float64
	for _, key := range keys {
		share := shares[key]
		cs := math.Floor(share * float64(acc.Balance))
		floored += cs

		// populate the return value
		ft.totalFeesAmountsPerParty[key] = uint64(cs)
		ft.transfers = append(ft.transfers, &types.Transfer{
			Owner: key,
			Amount: &types.FinancialAmount{
				Amount: uint64(cs),
				Asset:  acc.Asset,
			},
			MinAmount: int64(cs),
			Type:      types.TransferType_TRANSFER_TYPE_LIQUIDITY_FEE_DISTRIBUTE,
		})
	}

	// last is the party who will get the remaining from ceil
	last := keys[len(keys)-1]
	diff := acc.Balance - uint64(floored)
	ft.totalFeesAmountsPerParty[last] += diff
	ft.transfers[len(ft.transfers)-1].Amount.Amount += diff

	return ft
}

// this will calculate the transfer the distressed party needs
// to do
func (e *Engine) getPositionResolutionFeesTransfers(
	party string, share float64, fees *types.Fee,
) ([]*types.Transfer, *types.Fee, uint64) {
	makerFee := uint64(math.Ceil(share * float64(fees.MakerFee)))
	infraFee := uint64(math.Ceil(share * float64(fees.InfrastructureFee)))
	liquiFee := uint64(math.Ceil(share * float64(fees.LiquidityFee)))

	return []*types.Transfer{
			{
				Owner: party,
				Amount: &types.FinancialAmount{
					Asset:  e.asset,
					Amount: makerFee,
				},
				Type: types.TransferType_TRANSFER_TYPE_MAKER_FEE_PAY,
			},
			{
				Owner: party,
				Amount: &types.FinancialAmount{
					Asset:  e.asset,
					Amount: infraFee,
				},
				Type: types.TransferType_TRANSFER_TYPE_INFRASTRUCTURE_FEE_PAY,
			},
			{
				Owner: party,
				Amount: &types.FinancialAmount{
					Asset:  e.asset,
					Amount: liquiFee,
				},
				Type: types.TransferType_TRANSFER_TYPE_LIQUIDITY_FEE_PAY,
			},
		},
		&types.Fee{
			MakerFee:          uint64(makerFee),
			InfrastructureFee: uint64(infraFee),
			LiquidityFee:      uint64(liquiFee),
		}, uint64(makerFee + infraFee + liquiFee)
}

type feeShare struct {
	// the absolute position of the party which had to be recovered
	pos uint64
	// the share out of the total volume
	share float64
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
	tradeValueForFeePurpose := float64(trade.Price * trade.Size)
	return &types.Fee{
		MakerFee:          uint64(math.Ceil(tradeValueForFeePurpose * e.f.makerFee)),
		InfrastructureFee: uint64(math.Ceil(tradeValueForFeePurpose * e.f.infrastructureFee)),
		LiquidityFee:      uint64(math.Ceil(tradeValueForFeePurpose * e.f.liquidityFee)),
	}
}

func (e *Engine) calculateAuctionModeFees(trade *types.Trade) *types.Fee {
	fee := e.calculateContinuousModeFees(trade)
	return &types.Fee{
		MakerFee:          0,
		InfrastructureFee: uint64(math.Ceil(float64(fee.InfrastructureFee) / 2)),
		LiquidityFee:      uint64(math.Ceil(float64(fee.LiquidityFee) / 2)),
	}
}

func (e *Engine) getAuctionModeFeeTransfers(infraFee, liquiFee uint64, p string) []*types.Transfer {
	// we return both transfer for the party in a slice
	// always the infrastructure fee first
	return []*types.Transfer{
		{
			Owner: p,
			Amount: &types.FinancialAmount{
				Asset:  e.asset,
				Amount: infraFee,
			},
			Type: types.TransferType_TRANSFER_TYPE_INFRASTRUCTURE_FEE_PAY,
		},
		{
			Owner: p,
			Amount: &types.FinancialAmount{
				Asset:  e.asset,
				Amount: liquiFee,
			},
			Type: types.TransferType_TRANSFER_TYPE_LIQUIDITY_FEE_PAY,
		},
	}
}

type feesTransfer struct {
	totalFeesAmountsPerParty map[string]uint64
	transfers                []*types.Transfer
}

func (f *feesTransfer) TotalFeesAmountPerParty() map[string]uint64 { return f.totalFeesAmountsPerParty }
func (f *feesTransfer) Transfers() []*types.Transfer               { return f.transfers }

func (e *Engine) OnFeeFactorsMakerFeeUpdate(ctx context.Context, f float64) error {
	e.feeCfg.Factors.MakerFee = fmt.Sprintf("%f", f)
	e.f.makerFee = f
	return nil
}

func (e *Engine) OnFeeFactorsInfrastructureFeeUpdate(ctx context.Context, f float64) error {
	e.feeCfg.Factors.InfrastructureFee = fmt.Sprintf("%f", f)
	e.f.infrastructureFee = f
	return nil
}

func (e *Engine) GetLiquidityFee() float64 {
	return e.f.liquidityFee
}
