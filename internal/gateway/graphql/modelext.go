package gql

import (
	"code.vegaprotocol.io/vega/proto"
	"github.com/pkg/errors"
)

var (
	ErrNilTradingMode                  = errors.New("nil trading mode")
	ErrUnimplementedTradingMode        = errors.New("unimplemented trading mode")
	ErrNilMarket                       = errors.New("nil trading mode")
	ErrUnimplementedMarket             = errors.New("unimplemented trading mode")
	ErrNilTradableInstrument           = errors.New("nil tradable instrument")
	ErrUnimplementedTradableInstrument = errors.New("unimplemented tradable instrument")
	ErrNilOracle                       = errors.New("nil oracle")
	ErrUnimplementedOracle             = errors.New("unimplemented oracle")
	ErrNilProduct                      = errors.New("nil product")
	ErrUnimplementedProduct            = errors.New("unimplemented product")
	ErrNilRiskModel                    = errors.New("nil risk model")
	ErrUnimplementedRiskModel          = errors.New("unimplemented risk model")
	ErrNilInstrumentMetadata           = errors.New("nil instrument metadata")
	ErrUnimplementedInstrumentMetadata = errors.New("unimplemented instrument metadata")
	ErrNilEthereumEvent                = errors.New("nil ethereum event")
	ErrUnimplementedEthereumEvent      = errors.New("unimplemented ethereum event")
	ErrNilFuture                       = errors.New("nil future")
	ErrUnimplementedFuture             = errors.New("unimplemented future")
	ErrNilInstrument                   = errors.New("nil instrument")
	ErrUnimplementedInstrument         = errors.New("unimplemented instrument")
	ErrNilDiscreteTradingDuration      = errors.New("nil discrete trading duration")
	ErrNilContinuousTradingTickSize    = errors.New("nil continuous trading ticksize")
)

func (ct *ContinuousTrading) IntoProto() (*proto.Market_Continuous, error) {
	if ct.TickSize == nil {
		return nil, ErrNilContinuousTradingTickSize
	}
	return &proto.Market_Continuous{Continuous: &proto.ContinuousTrading{TickSize: uint64(*ct.TickSize)}}, nil
}

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

func (i *Instrument) IntoProto() (*proto.Instrument, error) {
	var err error
	pinst := &proto.Instrument{
		Id:   i.ID,
		Code: i.Code,
		Name: i.Name,
	}
	pinst.Metadata, err = i.Metadata.IntoProto()
	if err != nil {
		return nil, err
	}
	err = i.productIntoProto(pinst)
	if err != nil {
		return nil, err
	}

	return pinst, err
}

func (f *Forward) IntoProto() (*proto.TradableInstrument_Forward, error) {
	return &proto.TradableInstrument_Forward{
		Forward: &proto.Forward{
			Lambd: f.Lambd,
			Tau:   f.Tau,
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
	case *Forward:
		pti.RiskModel, err = rm.IntoProto()
	default:
		err = ErrUnimplementedRiskModel
	}
	return err
}

func (ti *TradableInstrument) IntoProto() (*proto.TradableInstrument, error) {
	var err error
	pti := &proto.TradableInstrument{}
	pti.Instrument, err = ti.Instrument.IntoProto()
	if err != nil {
		return nil, err
	}
	err = ti.riskModelIntoProto(pti)
	if err != nil {
		return nil, err
	}

	return pti, nil
}

func (m *Market) IntoProto() (*proto.Market, error) {
	var err error
	pmkt := &proto.Market{}
	pmkt.Id = m.ID
	if err := m.tradingModeIntoProto(pmkt); err != nil {
		return nil, err
	}

	pmkt.TradableInstrument, err = m.TradableInstrument.IntoProto()
	if err != nil {
		return nil, err
	}

	return pmkt, nil
}

func ContinuousTradingFromProto(pct *proto.ContinuousTrading) (*ContinuousTrading, error) {
	ts := int(pct.TickSize)
	return &ContinuousTrading{TickSize: &ts}, nil
}
func DiscreteTradingFromProto(pdt *proto.DiscreteTrading) (*DiscreteTrading, error) {
	dur := int(pdt.Duration)
	return &DiscreteTrading{
		Duration: &dur,
	}, nil
}

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

func EthereumEventFromProto(pee *proto.EthereumEvent) (*EthereumEvent, error) {
	if pee == nil {
		return nil, ErrNilEthereumEvent
	}

	return &EthereumEvent{
		ContractID: pee.ContractID,
		Event:      pee.Event,
	}, nil
}

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

func InstrumentFromProto(pi *proto.Instrument) (*Instrument, error) {
	if pi == nil {
		return nil, ErrNilInstrument
	}
	var err error
	i := &Instrument{
		ID:   pi.Id,
		Code: pi.Code,
		Name: pi.Name,
	}
	meta, err := InstrumentMetadataFromProto(pi.Metadata)
	if err != nil {
		return nil, err
	}
	i.Metadata = *meta
	i.Product, err = ProductFromProto(pi.Product)
	if err != nil {
		return nil, err
	}

	return i, nil
}

func ForwardFromProto(f *proto.Forward) (*Forward, error) {
	return &Forward{
		Lambd: f.Lambd,
		Tau:   f.Tau,
		Params: ModelParamsBs{
			Mu:    f.Params.Mu,
			R:     f.Params.R,
			Sigma: f.Params.Sigma,
		},
	}, nil
}

func RiskModelFromProto(rm interface{}) (RiskModel, error) {
	if rm == nil {
		return nil, ErrNilRiskModel
	}

	switch rmimpl := rm.(type) {
	case *proto.TradableInstrument_Forward:
		return ForwardFromProto(rmimpl.Forward)
	default:
		return nil, ErrUnimplementedRiskModel
	}
}

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
	ti.Instrument = *instrument
	ti.RiskModel, err = RiskModelFromProto(pti.RiskModel)
	if err != nil {
		return nil, err
	}
	return ti, nil
}

func MarketFromProto(pmkt *proto.Market) (*Market, error) {
	if pmkt == nil {
		return nil, ErrNilMarket
	}
	var err error
	mkt := &Market{}
	mkt.ID = pmkt.Id
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
	mkt.TradableInstrument = *tradableInstrument

	return mkt, nil
}
