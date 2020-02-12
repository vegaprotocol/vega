package gql

import (
	"strings"

	types "code.vegaprotocol.io/vega/proto"
	"github.com/pkg/errors"
)

var (
	// ErrNilTradingMode ...
	ErrNilTradingMode = errors.New("nil trading mode")
	// ErrUnimplementedTradingMode ...
	ErrUnimplementedTradingMode = errors.New("unimplemented trading mode")
	// ErrNilMarket ...
	ErrNilMarket = errors.New("nil market")
	// ErrNilTradableInstrument ...
	ErrNilTradableInstrument = errors.New("nil tradable instrument")
	// ErrNilOracle ..
	ErrNilOracle = errors.New("nil oracle")
	// ErrUnimplementedOracle ...
	ErrUnimplementedOracle = errors.New("unimplemented oracle")
	// ErrNilProduct ...
	ErrNilProduct = errors.New("nil product")
	// ErrUnimplementedProduct ...
	ErrUnimplementedProduct = errors.New("unimplemented product")
	// ErrNilRiskModel ...
	ErrNilRiskModel = errors.New("nil risk model")
	// ErrUnimplementedRiskModel ...
	ErrUnimplementedRiskModel = errors.New("unimplemented risk model")
	// ErrNilInstrumentMetadata ...
	ErrNilInstrumentMetadata = errors.New("nil instrument metadata")
	// ErrNilEthereumEvent ...
	ErrNilEthereumEvent = errors.New("nil ethereum event")
	// ErrNilFuture ...
	ErrNilFuture = errors.New("nil future")
	// ErrNilInstrument ...
	ErrNilInstrument = errors.New("nil instrument")
	// ErrNilDiscreteTradingDuration ...
	ErrNilDiscreteTradingDuration = errors.New("nil discrete trading duration")
	// ErrNilContinuousTradingTickSize ...
	ErrNilContinuousTradingTickSize = errors.New("nil continuous trading tick-size")
	// ErrnilScalingFactors...
	ErrNilScalingFactors = errors.New("nil scaling factors")
	// ErrNilMarginCalculator
	ErrNilMarginCalculator = errors.New("nil margin calculator")
)

// IntoProto ...
func (ct *ContinuousTrading) IntoProto() (*types.Market_Continuous, error) {
	if ct.TickSize == nil {
		return nil, ErrNilContinuousTradingTickSize
	}
	return &types.Market_Continuous{Continuous: &types.ContinuousTrading{TickSize: uint64(*ct.TickSize)}}, nil
}

// IntoProto ...
func (dt *DiscreteTrading) IntoProto() (*types.Market_Discrete, error) {
	if dt.Duration == nil {
		return nil, ErrNilDiscreteTradingDuration
	}
	return &types.Market_Discrete{
		Discrete: &types.DiscreteTrading{
			Duration: int64(*dt.Duration),
		},
	}, nil
}

func (m *Market) tradingModeIntoProto(mkt *types.Market) (err error) {
	if m.TradingMode == nil {
		return ErrNilTradingMode
	}
	switch tm := m.TradingMode.(type) {
	case *ContinuousTrading:
		mkt.TradingMode, err = tm.IntoProto()
	case *DiscreteTrading:
		mkt.TradingMode, err = tm.IntoProto()
	default:
		err = ErrUnimplementedTradingMode
	}
	return err
}

// IntoProto ...
func (ee *EthereumEvent) IntoProto() (*types.Future_EthereumEvent, error) {
	return &types.Future_EthereumEvent{
		EthereumEvent: &types.EthereumEvent{
			ContractID: ee.ContractID,
			Event:      ee.Event,
		},
	}, nil
}

func (f *Future) oracleIntoProto(pf *types.Future) (err error) {
	if f.Oracle == nil {
		return ErrNilOracle
	}
	switch o := f.Oracle.(type) {
	case *EthereumEvent:
		pf.Oracle, err = o.IntoProto()
	default:
		err = ErrUnimplementedOracle
	}
	return err

}

// IntoProto ...
func (f *Future) IntoProto() (*types.Instrument_Future, error) {
	var err error
	pf := &types.Future{
		Maturity: f.Maturity,
		Asset:    f.Asset,
	}
	err = f.oracleIntoProto(pf)
	if err != nil {
		return nil, err
	}

	return &types.Instrument_Future{Future: pf}, err
}

// IntoProto ...
func (im *InstrumentMetadata) IntoProto() (*types.InstrumentMetadata, error) {
	pim := &types.InstrumentMetadata{
		Tags: []string{},
	}
	for _, v := range im.Tags {
		pim.Tags = append(pim.Tags, *v)
	}
	return pim, nil
}

func (i *Instrument) productIntoProto(pinst *types.Instrument) (err error) {
	if i.Product == nil {
		return ErrNilProduct
	}
	switch p := i.Product.(type) {
	case *Future:
		pinst.Product, err = p.IntoProto()
	default:
		err = ErrUnimplementedProduct
	}
	return err
}

// IntoProto ...
func (i *Instrument) IntoProto() (*types.Instrument, error) {
	var err error
	pinst := &types.Instrument{
		Id:        i.ID,
		Code:      i.Code,
		Name:      i.Name,
		BaseName:  i.BaseName,
		QuoteName: i.QuoteName,
	}

	if i.Metadata != nil {
		pinst.Metadata, err = i.Metadata.IntoProto()
		if err != nil {
			return nil, err
		}
	}
	err = i.productIntoProto(pinst)
	if err != nil {
		return nil, err
	}

	return pinst, err
}

// IntoProto ...
func (f *LogNormalRiskModel) IntoProto() (*types.TradableInstrument_LogNormalRiskModel, error) {
	return &types.TradableInstrument_LogNormalRiskModel{
		LogNormalRiskModel: &types.LogNormalRiskModel{
			RiskAversionParameter: f.RiskAversionParameter,
			Tau:                   f.Tau,
			Params: &types.LogNormalModelParams{
				Mu:    f.Params.Mu,
				R:     f.Params.R,
				Sigma: f.Params.Sigma,
			},
		},
	}, nil
}

func (ti *TradableInstrument) riskModelIntoProto(
	pti *types.TradableInstrument) (err error) {
	if ti.RiskModel == nil {
		return ErrNilRiskModel
	}
	switch rm := ti.RiskModel.(type) {
	case *LogNormalRiskModel:
		pti.RiskModel, err = rm.IntoProto()
	default:
		err = ErrUnimplementedRiskModel
	}
	return err
}

// IntoProto ...
func (ti *TradableInstrument) IntoProto() (*types.TradableInstrument, error) {
	var err error
	pti := &types.TradableInstrument{}
	if ti.Instrument != nil {
		pti.Instrument, err = ti.Instrument.IntoProto()
		if err != nil {
			return nil, err
		}
	}
	if ti.MarginCalculator != nil {
		pti.MarginCalculator, _ = ti.MarginCalculator.IntoProto()
	}
	err = ti.riskModelIntoProto(pti)
	if err != nil {
		return nil, err
	}

	return pti, nil
}

func (m *MarginCalculator) IntoProto() (*types.MarginCalculator, error) {
	pm := &types.MarginCalculator{}
	if m.ScalingFactors != nil {
		pm.ScalingFactors, _ = m.ScalingFactors.IntoProto()
	}
	return pm, nil
}

func (s *ScalingFactors) IntoProto() (*types.ScalingFactors, error) {
	return &types.ScalingFactors{
		SearchLevel:       s.SearchLevel,
		InitialMargin:     s.InitialMargin,
		CollateralRelease: s.CollateralRelease,
	}, nil
}

// IntoProto ...
func (m *Market) IntoProto() (*types.Market, error) {
	var err error
	pmkt := &types.Market{}
	pmkt.Id = m.ID
	if err = m.tradingModeIntoProto(pmkt); err != nil {
		return nil, err
	}

	if m.TradableInstrument != nil {
		pmkt.TradableInstrument, err = m.TradableInstrument.IntoProto()
		if err != nil {
			return nil, err
		}
	}

	return pmkt, nil
}

// ContinuousTradingFromProto ...
func ContinuousTradingFromProto(pct *types.ContinuousTrading) (*ContinuousTrading, error) {
	ts := int(pct.TickSize)
	return &ContinuousTrading{TickSize: &ts}, nil
}

// DiscreteTradingFromProto ...
func DiscreteTradingFromProto(pdt *types.DiscreteTrading) (*DiscreteTrading, error) {
	dur := int(pdt.Duration)
	return &DiscreteTrading{
		Duration: &dur,
	}, nil
}

// TradingModeFromProto ...
func TradingModeFromProto(ptm interface{}) (TradingMode, error) {
	if ptm == nil {
		return nil, ErrNilTradingMode
	}

	switch ptmimpl := ptm.(type) {
	case *types.Market_Continuous:
		return ContinuousTradingFromProto(ptmimpl.Continuous)
	case *types.Market_Discrete:
		return DiscreteTradingFromProto(ptmimpl.Discrete)
	default:
		return nil, ErrUnimplementedTradingMode
	}
}

// InstrumentMetadataFromProto ...
func InstrumentMetadataFromProto(pim *types.InstrumentMetadata) (*InstrumentMetadata, error) {
	if pim == nil {
		return nil, ErrNilInstrumentMetadata
	}
	im := &InstrumentMetadata{
		Tags: []*string{},
	}

	for _, v := range pim.Tags {
		v := v
		im.Tags = append(im.Tags, &v)
	}

	return im, nil
}

// EthereumEventFromProto ...
func EthereumEventFromProto(pee *types.EthereumEvent) (*EthereumEvent, error) {
	if pee == nil {
		return nil, ErrNilEthereumEvent
	}

	return &EthereumEvent{
		ContractID: pee.ContractID,
		Event:      pee.Event,
	}, nil
}

// OracleFromProto ...
func OracleFromProto(o interface{}) (Oracle, error) {
	if o == nil {
		return nil, ErrNilOracle
	}

	switch oimpl := o.(type) {
	case *types.Future_EthereumEvent:
		return EthereumEventFromProto(oimpl.EthereumEvent)
	default:
		return nil, ErrUnimplementedOracle
	}
}

// FutureFromProto ...
func FutureFromProto(pf *types.Future) (*Future, error) {
	if pf == nil {
		return nil, ErrNilFuture
	}

	var err error
	f := &Future{}
	f.Maturity = pf.Maturity
	f.Asset = pf.Asset
	f.Oracle, err = OracleFromProto(pf.Oracle)
	if err != nil {
		return nil, err
	}

	return f, nil
}

// ProductFromProto ...
func ProductFromProto(pp interface{}) (Product, error) {
	if pp == nil {
		return nil, ErrNilProduct
	}

	switch pimpl := pp.(type) {
	case *types.Instrument_Future:
		return FutureFromProto(pimpl.Future)
	default:
		return nil, ErrUnimplementedProduct
	}
}

// InstrumentFromProto ...
func InstrumentFromProto(pi *types.Instrument) (*Instrument, error) {
	if pi == nil {
		return nil, ErrNilInstrument
	}
	var err error
	i := &Instrument{
		ID:        pi.Id,
		Code:      pi.Code,
		Name:      pi.Name,
		BaseName:  pi.BaseName,
		QuoteName: pi.QuoteName,
	}
	meta, err := InstrumentMetadataFromProto(pi.Metadata)
	if err != nil {
		return nil, err
	}
	i.Metadata = meta
	i.Product, err = ProductFromProto(pi.Product)
	if err != nil {
		return nil, err
	}

	return i, nil
}

// ForwardFromProto ...
func ForwardFromProto(f *types.LogNormalRiskModel) (*LogNormalRiskModel, error) {
	return &LogNormalRiskModel{
		RiskAversionParameter: f.RiskAversionParameter,
		Tau:                   f.Tau,
		Params: &LogNormalModelParams{
			Mu:    f.Params.Mu,
			R:     f.Params.R,
			Sigma: f.Params.Sigma,
		},
	}, nil
}

// SimpleRiskModelFromProto ...
func SimpleRiskModelFromProto(f *types.SimpleRiskModel) (*SimpleRiskModel, error) {
	return &SimpleRiskModel{
		Params: &SimpleRiskModelParams{
			FactorLong:  f.Params.FactorLong,
			FactorShort: f.Params.FactorShort,
		},
	}, nil
}

// RiskModelFromProto ...
func RiskModelFromProto(rm interface{}) (RiskModel, error) {
	if rm == nil {
		return nil, ErrNilRiskModel
	}
	switch rmimpl := rm.(type) {
	case *types.TradableInstrument_LogNormalRiskModel:
		return ForwardFromProto(rmimpl.LogNormalRiskModel)
	case *types.TradableInstrument_SimpleRiskModel:
		return SimpleRiskModelFromProto(rmimpl.SimpleRiskModel)
	default:
		return nil, ErrUnimplementedRiskModel
	}
}

// TradableInstrumentFromProto ...
func TradableInstrumentFromProto(pti *types.TradableInstrument) (*TradableInstrument, error) {
	if pti == nil {
		return nil, ErrNilTradableInstrument
	}
	var err error
	ti := &TradableInstrument{}
	instrument, err := InstrumentFromProto(pti.Instrument)
	if err != nil {
		return nil, err
	}
	ti.Instrument = instrument
	ti.RiskModel, err = RiskModelFromProto(pti.RiskModel)
	if err != nil {
		return nil, err
	}
	mc, err := MarginCalculatorFromProto(pti.MarginCalculator)
	if err != nil {
		return nil, err
	}
	ti.MarginCalculator = mc
	return ti, nil
}

func MarginCalculatorFromProto(mc *types.MarginCalculator) (*MarginCalculator, error) {
	if mc == nil {
		return nil, ErrNilMarginCalculator
	}
	m := &MarginCalculator{}
	sf, err := ScalingFactorsFromProto(mc.ScalingFactors)
	if err != nil {
		return nil, err
	}
	m.ScalingFactors = sf
	return m, nil
}

func ScalingFactorsFromProto(psf *types.ScalingFactors) (*ScalingFactors, error) {
	if psf == nil {
		return nil, ErrNilScalingFactors
	}
	return &ScalingFactors{
		SearchLevel:       psf.SearchLevel,
		InitialMargin:     psf.InitialMargin,
		CollateralRelease: psf.CollateralRelease,
	}, nil
}

// MarketFromProto ...
func MarketFromProto(pmkt *types.Market) (*Market, error) {
	if pmkt == nil {
		return nil, ErrNilMarket
	}
	var err error
	mkt := &Market{}
	mkt.ID = pmkt.Id
	mkt.Name = pmkt.Name
	mkt.DecimalPlaces = int(pmkt.DecimalPlaces)
	mkt.TradingMode, err = TradingModeFromProto(pmkt.TradingMode)
	if err != nil {
		return nil, err
	}
	tradableInstrument, err :=
		TradableInstrumentFromProto(pmkt.TradableInstrument)
	if err != nil {
		return nil, err
	}
	mkt.TradableInstrument = tradableInstrument

	return mkt, nil
}

// IntoProto ...
func (a AccountType) IntoProto() types.AccountType {
	if !a.IsValid() {
		return types.AccountType_ALL
	}
	return types.AccountType(types.AccountType_value[strings.ToUpper(string(a))])
}
