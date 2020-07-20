package fee

import (
	"errors"
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

// CalculateForContinuousTrading calculate the fee for
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
		fee := e.calculateContinuousModeFee(v.Price, v.Size)
		switch v.Aggressor {
		case types.Side_SIDE_BUY:
			v.BuyerFee = fee
			aggressor = v.Buyer
			maker = v.Seller
		case types.Side_SIDE_SELL:
			v.SellerFee = fee
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
		transfersRecv = append(transfers, &types.Transfer{
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
		Type: types.TransferType_TRANSFER_TYPE_INFRASTRUCTURE_FEE_PAY,
	})

	return &feesTransfer{
		partyID:        aggressor,
		totalFeeAmount: totalFeeAmount,
		transfers:      append(transfers, transfersRecv...),
	}, nil
}

func (e *Engine) calculateContinuousModeFee(price, size uint64) *types.Fee {
	tradeValueForFeePurpose := float64(price * size)
	return &types.Fee{
		MakerFee:          uint64(tradeValueForFeePurpose * e.f.makerFee),
		InfrastructureFee: uint64(tradeValueForFeePurpose * e.f.infrastructureFee),
		LiquidityFee:      uint64(tradeValueForFeePurpose * e.f.liquidityFee),
	}
}

// CalculateForContinuousTrading calculate the fee for
// trades which were produced from a market running in
// in auction trading mode.
// A list FeesTransfer is produced each containing fees transfer from a
// single trader
func (e *Engine) CalculateForAuctionMode(
	trades []*types.Trade,
) ([]events.FeesTransfer, error) {
	return nil, errors.New("unimplemented")
}

type feesTransfer struct {
	totalFeeAmount uint64
	partyID        string
	transfers      []*types.Transfer
}

func (f *feesTransfer) PartyID() string              { return f.partyID }
func (f *feesTransfer) TotalFeeAmount() uint64       { return f.totalFeeAmount }
func (f *feesTransfer) Transfers() []*types.Transfer { return f.transfers }
