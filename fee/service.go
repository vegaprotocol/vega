package fee

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
)

// MarketStore ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/market_store_mock.go -package mocks code.vegaprotocol.io/vega/fee MarketStore
type MarketStore interface {
	GetByID(name string) (*types.Market, error)
}

// MarketDataStore ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/market_data_store_mock.go -package mocks code.vegaprotocol.io/vega/fee MarketDataStore
type MarketDataStore interface {
	GetByID(marketID string) (types.MarketData, error)
}

type Svc struct {
	cfg          Config
	log          *logging.Logger
	mktStore     MarketStore
	mktDataStore MarketDataStore
}

func NewService(log *logging.Logger, cfg Config, mktStore MarketStore, mktDataStore MarketDataStore) *Svc {
	return &Svc{
		cfg:          cfg,
		log:          log,
		mktStore:     mktStore,
		mktDataStore: mktDataStore,
	}
}

// ReloadConf is used in order to reload the internal configuration of
// the of the fee engine
func (s *Svc) ReloadConf(cfg Config) {
	s.log.Info("reloading configuration")
	if s.log.GetLevel() != cfg.Level.Get() {
		s.log.Info("updating log level",
			logging.String("old", s.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		s.log.SetLevel(cfg.Level.Get())
	}

	s.cfg = cfg
}

func (s *Svc) EstimateFee(ctx context.Context, o *types.Order) (*types.Fee, error) {
	mkt, err := s.mktStore.GetByID(o.MarketID)
	if err != nil {
		return nil, err
	}
	price := o.Price
	if o.PeggedOrder != nil {
		mktdata, err := s.mktDataStore.GetByID(o.MarketID)
		if err != nil {
			return nil, err
		}

		switch o.PeggedOrder.Reference {
		case types.PeggedReference_PEGGED_REFERENCE_MID:
			price = mktdata.StaticMidPrice
		case types.PeggedReference_PEGGED_REFERENCE_BEST_BID:
			price = mktdata.BestStaticBidPrice
		case types.PeggedReference_PEGGED_REFERENCE_BEST_ASK:
			price = mktdata.BestStaticOfferPrice
		default:
			return nil, errors.New("can't calculate fees for pegged order without a reference")
		}

		if o.PeggedOrder.Offset >= 0 {
			price += uint64(o.PeggedOrder.Offset)
		} else {
			offset := uint64(-o.PeggedOrder.Offset)
			if price <= offset {
				return nil, fmt.Errorf("can't calculate fees, pegged order price would be negative, price(%v), offset(-%v)", price, offset)
			}
			price -= offset
		}
	}

	base := float64(price * o.Size)
	maker, infra, liqui, err := s.feeFactors(mkt)
	if err != nil {
		return nil, err
	}

	fee := &types.Fee{
		MakerFee:          uint64(math.Ceil(base * maker)),
		InfrastructureFee: uint64(math.Ceil(base * infra)),
		LiquidityFee:      uint64(math.Ceil(base * liqui)),
	}

	// if mkt.State == types.MarketState_MARKET_STATE_OPENNING_AUCTION {
	// 	// half price paid by both partis
	// 	fee.MakerFee = fee.MakerFee / 2
	// 	fee.InfrastructureFee = fee.InfrastructureFee / 2
	// 	fee.LiquidityFee = fee.LiquidityFee / 2
	// }

	return fee, nil
}

func (s *Svc) feeFactors(mkt *types.Market) (maker, infra, liqui float64, err error) {
	maker, err = strconv.ParseFloat(mkt.Fees.Factors.MakerFee, 64)
	if err != nil {
		return
	}
	infra, err = strconv.ParseFloat(mkt.Fees.Factors.InfrastructureFee, 64)
	if err != nil {
		return
	}
	liqui, err = strconv.ParseFloat(mkt.Fees.Factors.LiquidityFee, 64)
	if err != nil {
		return
	}
	return
}
