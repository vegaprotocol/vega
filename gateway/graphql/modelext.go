package gql

import (
	"strings"

	"code.vegaprotocol.io/vega/proto"
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
	// ErrUnimplementedMarket ...
	ErrUnimplementedMarket = errors.New("unimplemented market")
	// ErrNilTradableInstrument ...
	ErrNilTradableInstrument = errors.New("nil tradable instrument")
	// ErrUnimplementedTradableInstrument ...
	ErrUnimplementedTradableInstrument = errors.New("unimplemented tradable instrument")
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
	// ErrUnimplementedInstrumentMetadata ...
	ErrUnimplementedInstrumentMetadata = errors.New("unimplemented instrument metadata")
	// ErrNilEthereumEvent ...
	ErrNilEthereumEvent = errors.New("nil ethereum event")
	// ErrUnimplementedEthereumEvent ...
	ErrUnimplementedEthereumEvent = errors.New("unimplemented ethereum event")
	// ErrNilFuture ...
	ErrNilFuture = errors.New("nil future")
	// ErrUnimplementedFuture ...
	ErrUnimplementedFuture = errors.New("unimplemented future")
	// ErrNilInstrument ...
	ErrNilInstrument = errors.New("nil instrument")
	// ErrUnimplementedInstrument ...
	ErrUnimplementedInstrument = errors.New("unimplemented instrument")
	// ErrNilDiscreteTradingDuration ...
	ErrNilDiscreteTradingDuration = errors.New("nil discrete trading duration")
	// ErrNilContinuousTradingTickSize ...
	ErrNilContinuousTradingTickSize = errors.New("nil continuous trading ticksize")
)

// IntoProto ...
func (ct *ContinuousTrading) IntoProto() (*proto.Market_Continuous, error) {
	if ct.TickSize == nil {
		return nil, ErrNilContinuousTradingTickSize
	}
	return &proto.Market_Continuous{Continuous: &proto.ContinuousTrading{TickSize: uint64(*ct.TickSize)}}, nil
}

// IntoProto ...
func (dt *DiscreteTrading) IntoProto() (*proto.Market_Discrete, error) {
	if dt.Duration == nil {
		return nil, ErrNilDiscreteTradingDuration
	}
	return &proto.Market_Discrete{
		Discrete: &proto.DiscreteTrading{
			Duration: int64(*dt.Duration),
		},
	}, nil
}

func (m *Market) tradingModeIntoProto(mkt *proto.Market) (err error) {
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
func (ee *EthereumEvent) IntoProto() (*proto.Future_EthereumEvent, error) {
	return &proto.Future_EthereumEvent{
		EthereumEvent: &proto.EthereumEvent{
			ContractID: ee.ContractID,
			Event:      ee.Event,
		},
	}, nil
}

func (f *Future) oracleIntoProto(pf *proto.Future) (err error) {
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
func (f *Future) IntoProto() (*proto.Instrument_Future, error) {
	var err error
	pf := &proto.Future{
		Maturity: f.Maturity,
		Asset:    f.Asset,
	}
	err = f.oracleIntoProto(pf)
	if err != nil {
		return nil, err
	}

	return &proto.Instrument_Future{Future: pf}, err
}

// IntoProto ...
func (im *InstrumentMetadata) IntoProto() (*proto.InstrumentMetadata, error) {
	pim := &proto.InstrumentMetadata{
		Tags: []string{},
	}
	for _, v := range im.Tags {
		pim.Tags = append(pim.Tags, *v)
	}
	return pim, nil
}

func (i *Instrument) productIntoProto(pinst *proto.Instrument) (err error) {
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
func (i *Instrument) IntoProto() (*proto.Instrument, error) {
	var err error
	pinst := &proto.Instrument{
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
func (f *ForwardRiskModel) IntoProto() (*proto.TradableInstrument_ForwardRiskModel, error) {
	return &proto.TradableInstrument_ForwardRiskModel{
		ForwardRiskModel: &proto.ForwardRiskModel{
			RiskAversionParameter: f.RiskAversionParameter,
			Tau:                   f.Tau,
			Params: &proto.ModelParamsBS{
				Mu:    f.Params.Mu,
				R:     f.Params.R,
				Sigma: f.Params.Sigma,
			},
		},
	}, nil
}

func (ti *TradableInstrument) riskModelIntoProto(
	pti *proto.TradableInstrument) (err error) {
	if ti.RiskModel == nil {
		return ErrNilRiskModel
	}
	switch rm := ti.RiskModel.(type) {
	case *ForwardRiskModel:
		pti.RiskModel, err = rm.IntoProto()
	default:
		err = ErrUnimplementedRiskModel
	}
	return err
}

// IntoProto ...
func (ti *TradableInstrument) IntoProto() (*proto.TradableInstrument, error) {
	var err error
	pti := &proto.TradableInstrument{}
	if ti.Instrument != nil {
		pti.Instrument, err = ti.Instrument.IntoProto()
		if err != nil {
			return nil, err
		}
	}
	err = ti.riskModelIntoProto(pti)
	if err != nil {
		return nil, err
	}

	return pti, nil
}

// IntoProto ...
func (m *Market) IntoProto() (*proto.Market, error) {
	var err error
	pmkt := &proto.Market{}
	pmkt.Id = m.ID
	if err := m.tradingModeIntoProto(pmkt); err != nil {
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
func ContinuousTradingFromProto(pct *proto.ContinuousTrading) (*ContinuousTrading, error) {
	ts := int(pct.TickSize)
	return &ContinuousTrading{TickSize: &ts}, nil
}

// DiscreteTradingFromProto ...
func DiscreteTradingFromProto(pdt *proto.DiscreteTrading) (*DiscreteTrading, error) {
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
	case *proto.Market_Continuous:
		return ContinuousTradingFromProto(ptmimpl.Continuous)
	case *proto.Market_Discrete:
		return DiscreteTradingFromProto(ptmimpl.Discrete)
	default:
		return nil, ErrUnimplementedTradingMode
	}
}

// InstrumentMetadataFromProto ...
func InstrumentMetadataFromProto(pim *proto.InstrumentMetadata) (*InstrumentMetadata, error) {
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
func EthereumEventFromProto(pee *proto.EthereumEvent) (*EthereumEvent, error) {
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
	case *proto.Future_EthereumEvent:
		return EthereumEventFromProto(oimpl.EthereumEvent)
	default:
		return nil, ErrUnimplementedOracle
	}
}

// FutureFromProto ...
func FutureFromProto(pf *proto.Future) (*Future, error) {
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
	case *proto.Instrument_Future:
		return FutureFromProto(pimpl.Future)
	default:
		return nil, ErrUnimplementedProduct
	}
}

// InstrumentFromProto ...
func InstrumentFromProto(pi *proto.Instrument) (*Instrument, error) {
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
func ForwardFromProto(f *proto.ForwardRiskModel) (*ForwardRiskModel, error) {
	return &ForwardRiskModel{
		RiskAversionParameter: f.RiskAversionParameter,
		Tau:                   f.Tau,
		Params: &ModelParamsBs{
			Mu:    f.Params.Mu,
			R:     f.Params.R,
			Sigma: f.Params.Sigma,
		},
	}, nil
}

// SimpleRiskModelFromProto ...
func SimpleRiskModelFromProto(f *proto.SimpleRiskModel) (*SimpleRiskModel, error) {
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
	case *proto.TradableInstrument_ForwardRiskModel:
		return ForwardFromProto(rmimpl.ForwardRiskModel)
	case *proto.TradableInstrument_SimpleRiskModel:
		return SimpleRiskModelFromProto(rmimpl.SimpleRiskModel)
	default:
		return nil, ErrUnimplementedRiskModel
	}
}

// TradableInstrumentFromProto ...
func TradableInstrumentFromProto(pti *proto.TradableInstrument) (*TradableInstrument, error) {
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
	return ti, nil
}

// MarketFromProto ...
func MarketFromProto(pmkt *proto.Market) (*Market, error) {
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
		return types.AccountType_NO_ACC
	}
	return types.AccountType(types.AccountType_value[strings.ToUpper(string(a))])
}
