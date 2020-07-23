package fee

import (
	"errors"
	"math"
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
	f, err = strconv.ParseFloat(fees.Factors.LiquidityFee, 64)
	if err != nil {
		e.log.Error("unable to load liquidityfee", logging.Error(err))
		return err
	}
	e.f.liquidityFee = f
	e.feeCfg = fees
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
		transfers     []*types.Transfer = make([]*types.Transfer, 0, (len(trades)*2)+2)
		transfersRecv []*types.Transfer = make([]*types.Transfer, 0, len(trades)+2)
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

		totalFeeAmount += (fee.InfrastructureFee + fee.LiquidityFee + fee.MakerFee)
		totalInfrastructureFeeAmount += fee.InfrastructureFee
		totalLiquidityFeeAmount += fee.LiquidityFee

		// create a transfer for the aggressor
		transfers = append(transfers, &types.Transfer{
			Owner: aggressor,
			Amount: &types.FinancialAmount{
				Asset:  e.asset,
				Amount: int64(fee.MakerFee),
			},
			Type: types.TransferType_TRANSFER_TYPE_MAKER_FEE_PAY,
		})
		// create a transfer for the maker
		transfersRecv = append(transfersRecv, &types.Transfer{
			Owner: maker,
			Amount: &types.FinancialAmount{
				Asset:  e.asset,
				Amount: int64(fee.MakerFee),
			},
			Type: types.TransferType_TRANSFER_TYPE_MAKER_FEE_RECEIVE,
		})
	}

	// now create transfer for the infrastructure
	transfers = append(transfers, &types.Transfer{
		Owner: aggressor,
		Amount: &types.FinancialAmount{
			Asset:  e.asset,
			Amount: int64(totalInfrastructureFeeAmount),
		},
		Type: types.TransferType_TRANSFER_TYPE_INFRASTRUCTURE_FEE_PAY,
	})
	// now create transfer for the liquidity
	transfers = append(transfers, &types.Transfer{
		Owner: aggressor,
		Amount: &types.FinancialAmount{
			Asset:  e.asset,
			Amount: int64(totalInfrastructureFeeAmount),
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
		&types.Transfer{
			Owner: p,
			Amount: &types.FinancialAmount{
				Asset:  e.asset,
				Amount: int64(infraFee),
			},
			Type: types.TransferType_TRANSFER_TYPE_INFRASTRUCTURE_FEE_PAY,
		},
		&types.Transfer{
			Owner: p,
			Amount: &types.FinancialAmount{
				Asset:  e.asset,
				Amount: int64(liquiFee),
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
