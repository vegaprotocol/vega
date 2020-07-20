package fee

import (
	"errors"
	"strconv"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
)

type Engine struct {
	log *logging.Logger
	cfg Config

	feeCfg types.Fees
	f      factors
}

type factors struct {
	makerFee          float64
	infrastructureFee float64
	liquidityFee      float64
}

func New(log *logging.Logger, cfg Config, feeCfg types.Fees) (*Engine, error) {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	e := &Engine{
		log:    log,
		feeCfg: feeCfg,
		cfg:    cfg,
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
	return nil, nil
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
