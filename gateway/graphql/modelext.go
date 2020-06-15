package gql

import (
	types "code.vegaprotocol.io/vega/proto"
	protoapi "code.vegaprotocol.io/vega/proto/api"
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
	// ErrNilScalingFactors ...
	ErrNilScalingFactors = errors.New("nil scaling factors")
	// ErrNilMarginCalculator ...
	ErrNilMarginCalculator = errors.New("nil margin calculator")
	// ErrInvalidTickSize ...
	ErrInvalidTickSize = errors.New("invalid tick size")
	// ErrInvalidDecimalPlaces ...
	ErrInvalidDecimalPlaces = errors.New("invalid decimal places value")
	// ErrInvalidChange ...
	ErrInvalidChange = errors.New("nil update market, new market and update network")
	// ErrInvalidProposalState ...
	ErrInvalidProposalState = errors.New("invalid proposal state")
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
	at, _ := convertAccountType(a)
	return at
}

// ProposalTermsFromProto ...
func ProposalTermsFromProto(terms *types.ProposalTerms) (*ProposalTerms, error) {
	result := &ProposalTerms{
		ClosingDatetime:   secondsTSToDatetime(terms.ClosingTimestamp),
		EnactmentDatetime: secondsTSToDatetime(terms.EnactmentTimestamp),
	}
	if terms.GetUpdateMarket() != nil {
		result.Change = nil
	} else if newMarket := terms.GetNewMarket(); newMarket != nil {
		market, err := MarketFromProto(newMarket.Changes)
		if err != nil {
			return nil, err
		}
		result.Change = &NewMarket{Market: market}
	} else if terms.GetUpdateNetwork() != nil {
		result.Change = nil
	}
	return result, nil
}

// IntoProto ...
func (i *InstrumentInput) IntoProto() (*types.Instrument, error) {
	initMarkPrice, err := safeStringUint64(i.InitialMarkPrice)
	if err != nil {
		return nil, err
	}
	return &types.Instrument{
		Id:        i.ID,
		Code:      i.Code,
		Name:      i.Name,
		BaseName:  i.BaseName,
		QuoteName: i.QuoteName,
		Metadata: &types.InstrumentMetadata{
			Tags: removePointers(i.Metadata.Tags),
		},
		InitialMarkPrice: initMarkPrice,
		Product:          nil,
	}, nil
}

// IntoProto ...
func (m *MarginCalculatorInput) IntoProto() (*types.MarginCalculator, error) {
	if m == nil {
		return nil, ErrNilMarginCalculator
	}
	return &types.MarginCalculator{
		ScalingFactors: &types.ScalingFactors{
			SearchLevel:       m.ScalingFactors.SearchLevel,
			InitialMargin:     m.ScalingFactors.InitialMargin,
			CollateralRelease: m.ScalingFactors.CollateralRelease,
		},
	}, nil
}

func (f *FutureInput) oracleIntoProto(pf *types.Future) error {
	if f.EthereumOracle != nil {
		pf.Oracle = &types.Future_EthereumEvent{
			EthereumEvent: &types.EthereumEvent{
				ContractID: f.EthereumOracle.ContractID,
				Event:      f.EthereumOracle.Event,
			},
		}
		return nil
	}
	return ErrNilOracle
}

func (i *InstrumentInput) productInputIntoProto(pinst *types.Instrument) error {
	if future := i.FutureProduct; future != nil {
		f := &types.Future{
			Maturity: future.Maturity,
			Asset:    future.Asset,
		}
		future.oracleIntoProto(f)
		pinst.Product = &types.Instrument_Future{Future: f}
		return nil
	}
	return ErrNilProduct
}

func (t *TradableInstrumentInput) riskModelInputIntoProto(trIn *types.TradableInstrument) error {
	if t.SimpleRiskModel != nil {
		trIn.RiskModel = &types.TradableInstrument_SimpleRiskModel{
			SimpleRiskModel: &types.SimpleRiskModel{
				Params: &types.SimpleModelParams{
					FactorLong:  t.SimpleRiskModel.Params.FactorLong,
					FactorShort: t.SimpleRiskModel.Params.FactorShort,
				},
			},
		}
	} else if t.LogNormalRiskModel != nil {
		trIn.RiskModel = &types.TradableInstrument_LogNormalRiskModel{
			LogNormalRiskModel: &types.LogNormalRiskModel{
				RiskAversionParameter: t.LogNormalRiskModel.RiskAversionParameter,
				Tau:                   t.LogNormalRiskModel.Tau,
				Params: &types.LogNormalModelParams{
					Mu:    t.LogNormalRiskModel.Params.Mu,
					R:     t.LogNormalRiskModel.Params.R,
					Sigma: t.LogNormalRiskModel.Params.Sigma,
				},
			},
		}
	} else {
		return ErrNilRiskModel
	}
	return nil
}

// IntoProto ...
func (t *TradableInstrumentInput) IntoProto() (*types.TradableInstrument, error) {
	instrument, err := t.Instrument.IntoProto()
	if err != nil {
		return nil, err
	}
	calc, err := t.MarginCalculator.IntoProto()
	if err != nil {
		return nil, err
	}
	result := &types.TradableInstrument{
		Instrument:       instrument,
		MarginCalculator: calc,
		RiskModel:        nil,
	}
	if err := t.Instrument.productInputIntoProto(result.Instrument); err != nil {
		return nil, err
	}
	if err := t.riskModelInputIntoProto(result); err != nil {
		return nil, err
	}

	return result, nil
}

func (m *MarketInput) tradingModeInputIntoProto(market *types.Market) error {
	if m.ContinuousTradingMode != nil {
		if m.ContinuousTradingMode.TickSize < 0 {
			return ErrInvalidTickSize
		}
		market.TradingMode = &types.Market_Continuous{
			Continuous: &types.ContinuousTrading{
				TickSize: uint64(m.ContinuousTradingMode.TickSize),
			},
		}
	} else if m.DiscreteTradingMode != nil {
		market.TradingMode = &types.Market_Discrete{
			Discrete: &types.DiscreteTrading{
				Duration: int64(m.DiscreteTradingMode.Duration),
			},
		}
	} else {
		return ErrNilTradingMode
	}
	return nil
}

// IntoProto ...
func (m *MarketInput) IntoProto() (*types.Market, error) {
	ti, err := m.TradableInstrument.IntoProto()
	if err != nil {
		return nil, err
	}
	if m.DecimalPlaces < 0 {
		return nil, ErrInvalidDecimalPlaces
	}
	result := &types.Market{
		Name:               m.Name,
		TradableInstrument: ti,
		DecimalPlaces:      uint64(m.DecimalPlaces),
		TradingMode:        nil,
	}
	if err := m.tradingModeInputIntoProto(result); err != nil {
		return nil, err
	}
	return result, nil
}

// IntoProto ...
func (p ProposalTermsInput) IntoProto() (*types.ProposalTerms, error) {
	closing, err := datetimeToSecondsTS(p.ClosingDatetime)
	if err != nil {
		return nil, err
	}
	enactment, err := datetimeToSecondsTS(p.EnactmentDatetime)
	if err != nil {
		return nil, err
	}

	result := &types.ProposalTerms{
		ClosingTimestamp:   closing,
		EnactmentTimestamp: enactment,
	}
	if p.UpdateMarket != nil {
		result.Change = &types.ProposalTerms_UpdateMarket{}
	} else if p.NewMarket != nil {
		market, err := p.NewMarket.Market.IntoProto()
		if err != nil {
			return nil, err
		}
		result.Change = &types.ProposalTerms_NewMarket{
			NewMarket: &types.NewMarket{
				Changes: market,
			},
		}
	} else if p.UpdateNetwork != nil {
		result.Change = &types.ProposalTerms_UpdateMarket{}
	} else {
		return nil, ErrInvalidChange
	}

	return result, nil
}

// ToOptionalProposalState ...
func (s *ProposalState) ToOptionalProposalState() (*protoapi.OptionalProposalState, error) {
	if s != nil {
		value, err := s.IntoProtoValue()
		if err != nil {
			return nil, err
		}
		return &protoapi.OptionalProposalState{
			Value: value,
		}, nil
	}
	return nil, nil
}

// IntoProtoValue ...
func (s ProposalState) IntoProtoValue() (types.Proposal_State, error) {
	return convertProposalState(s)
}

// ProposalVoteFromProto ...
func ProposalVoteFromProto(v *types.Vote, caster *types.Party) *ProposalVote {
	value, _ := unconvertVoteValue(v.Value)
	return &ProposalVote{
		Vote: &Vote{
			Party:    caster,
			Value:    value,
			Datetime: nanoTSToDatetime(v.Timestamp),
		},
		ProposalID: v.ProposalID,
	}
}
