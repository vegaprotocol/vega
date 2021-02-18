package gql

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"

	types "code.vegaprotocol.io/vega/proto"
	protoapi "code.vegaprotocol.io/vega/proto/api"
	oraclesv1 "code.vegaprotocol.io/vega/proto/oracles/v1"
)

var (
	// ErrNilTradingMode ...
	ErrNilTradingMode = errors.New("nil trading mode")
	// ErrAmbiguousTradingMode ...
	ErrAmbiguousTradingMode = errors.New("more than one trading mode selected")
	// ErrUnimplementedTradingMode ...
	ErrUnimplementedTradingMode = errors.New("unimplemented trading mode")
	// ErrNilProduct ...
	ErrNilProduct = errors.New("nil product")
	// ErrNilRiskModel ...
	ErrNilRiskModel = errors.New("nil risk model")
	// ErrUnimplementedRiskModel ...
	ErrUnimplementedRiskModel = errors.New("unimplemented risk model")
	// ErrNilInstrumentMetadata ...
	ErrNilInstrumentMetadata = errors.New("nil instrument metadata")
	// ErrTradingDurationNegative ...
	ErrTradingDurationNegative = errors.New("invalid trading duration (negative)")
	// ErrTickSizeNegative ...
	ErrTickSizeNegative = errors.New("invalid tick size (negative)")
	// ErrNilScalingFactors ...
	ErrNilScalingFactors = errors.New("nil scaling factors")
	// ErrNilMarginCalculator ...
	ErrNilMarginCalculator = errors.New("nil margin calculator")
	// ErrInvalidChange ...
	ErrInvalidChange = errors.New("nil update market, new market and update network")
	// ErrInvalidRiskConfiguration ...
	ErrInvalidRiskConfiguration = errors.New("invalid risk configuration")
	// ErrNilAssetSource returned when an asset source is not specified at creation
	ErrNilAssetSource = errors.New("nil asset source")
	// ErrUnimplementedAssetSource returned when an asset source specified at creation is not recognised
	ErrUnimplementedAssetSource = errors.New("unimplemented asset source")
	// ErrMultipleProposalChangesSpecified is raised when multiple proposal changes are set
	// (non-null) for a singe proposal terms
	ErrMultipleProposalChangesSpecified = errors.New("multiple proposal changes specified")
	// ErrMultipleAssetSourcesSpecified is raised when multiple asset source are specified
	ErrMultipleAssetSourcesSpecified = errors.New("multiple asset sources specified")
	// ErrNilFeeFactors is raised when the fee factors are missing from the fees
	ErrNilFeeFactors = errors.New("nil fee factors")
	// ErrNilPriceMonitoringParameters ...
	ErrNilPriceMonitoringParameters = errors.New("nil price monitoring parameters")
)

type MarketLogEvent interface {
	GetMarketID() string
	GetPayload() string
}

// IntoProto ...
func (c *ContinuousTrading) IntoProto() (*types.Market_Continuous, error) {
	if len(c.TickSize) <= 0 {
		return nil, ErrTickSizeNegative
	}
	// parsing just make sure it's a valid float
	_, err := strconv.ParseFloat(c.TickSize, 64)
	if err != nil {
		return nil, err
	}

	return &types.Market_Continuous{
		Continuous: &types.ContinuousTrading{
			TickSize: c.TickSize,
		},
	}, nil
}

// IntoProto ...
func (d *DiscreteTrading) IntoProto() (*types.Market_Discrete, error) {
	if len(d.TickSize) <= 0 {
		return nil, ErrTickSizeNegative
	}
	// parsing just make sure it's a valid float
	_, err := strconv.ParseFloat(d.TickSize, 64)
	if err != nil {
		return nil, err
	}
	if d.Duration < 0 {
		return nil, ErrTradingDurationNegative
	}
	return &types.Market_Discrete{
		Discrete: &types.DiscreteTrading{
			TickSize:   d.TickSize,
			DurationNs: int64(d.Duration),
		},
	}, nil
}

// IntoProto ...
func (im *InstrumentMetadata) IntoProto() (*types.InstrumentMetadata, error) {
	pim := &types.InstrumentMetadata{
		Tags: []string{},
	}
	pim.Tags = append(pim.Tags, im.Tags...)
	return pim, nil
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

func (f *FeeFactors) IntoProto() (*types.FeeFactors, error) {
	return &types.FeeFactors{
		LiquidityFee:      f.LiquidityFee,
		MakerFee:          f.MakerFee,
		InfrastructureFee: f.InfrastructureFee,
	}, nil
}

func (f *Fees) IntoProto() (*types.Fees, error) {
	pf := &types.Fees{}
	if f.Factors != nil {
		pf.Factors, _ = f.Factors.IntoProto()
	}
	return pf, nil
}

func (o *LiquidityOrderInput) IntoProto() (*types.LiquidityOrder, error) {
	if o.Proportion < 0 {
		return nil, errors.New("proportion can't be negative")
	}

	ref, err := convertPeggedReferenceToProto(o.Reference)
	if err != nil {
		return nil, err
	}

	return &types.LiquidityOrder{
		Offset:     int64(o.Offset),
		Proportion: uint32(o.Proportion),
		Reference:  ref,
	}, nil
}

type LiquidityOrderInputs []*LiquidityOrderInput

func (inputs LiquidityOrderInputs) IntoProto() ([]*types.LiquidityOrder, error) {
	orders := make([]*types.LiquidityOrder, len(inputs))
	for i, input := range inputs {
		v, err := input.IntoProto()
		if err != nil {
			return nil, err
		}
		orders[i] = v
	}

	return orders, nil

}

// ContinuousTradingFromProto ...
func ContinuousTradingFromProto(pct *types.ContinuousTrading) (*ContinuousTrading, error) {
	return &ContinuousTrading{
		TickSize: pct.TickSize,
	}, nil
}

// DiscreteTradingFromProto ...
func DiscreteTradingFromProto(pdt *types.DiscreteTrading) (*DiscreteTrading, error) {
	return &DiscreteTrading{
		Duration: int(pdt.DurationNs),
		TickSize: pdt.TickSize,
	}, nil
}

// TradingModeConfigFromProto ...
func TradingModeConfigFromProto(ptm interface{}) (TradingMode, error) {
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

// NewMarketTradingModeFromProto ...
func NewMarketTradingModeFromProto(ptm interface{}) (TradingMode, error) {
	if ptm == nil {
		ptm = defaultTradingMode()
	}
	switch ptmimpl := ptm.(type) {
	case *types.NewMarketConfiguration_Continuous:
		return ContinuousTradingFromProto(ptmimpl.Continuous)
	case *types.NewMarketConfiguration_Discrete:
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
		Tags: []string{},
	}

	for _, v := range pim.Tags {
		v := v
		im.Tags = append(im.Tags, v)
	}

	return im, nil
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

func PriceMonitoringTriggerFromProto(ppmt *types.PriceMonitoringTrigger) *PriceMonitoringTrigger {
	return &PriceMonitoringTrigger{
		HorizonSecs:          int(ppmt.Horizon),
		Probability:          ppmt.Probability,
		AuctionExtensionSecs: int(ppmt.AuctionExtension),
	}
}

func PriceMonitoringParametersFromProto(ppmp *types.PriceMonitoringParameters) (*PriceMonitoringParameters, error) {
	if ppmp == nil {
		return nil, ErrNilPriceMonitoringParameters
	}

	triggers := make([]*PriceMonitoringTrigger, 0, len(ppmp.Triggers))
	for _, v := range ppmp.Triggers {
		triggers = append(triggers, PriceMonitoringTriggerFromProto(v))
	}

	return &PriceMonitoringParameters{
		Triggers: triggers,
	}, nil
}

func PriceMonitoringSettingsFromProto(ppmst *types.PriceMonitoringSettings) (*PriceMonitoringSettings, error) {
	if ppmst == nil {
		// these are not mandatoryu anyway for now, so if nil we return an empty one
		return &PriceMonitoringSettings{}, nil
	}

	params, err := PriceMonitoringParametersFromProto(ppmst.Parameters)
	if err != nil {
		return nil, err
	}
	return &PriceMonitoringSettings{
		Parameters:          params,
		UpdateFrequencySecs: int(ppmst.UpdateFrequency),
	}, nil
}

// RiskConfigurationFromProto ...
func RiskConfigurationFromProto(newMarket *types.NewMarketConfiguration) (RiskModel, error) {
	if newMarket.RiskParameters == nil {
		newMarket.RiskParameters = defaultRiskParameters()
	}
	switch params := newMarket.RiskParameters.(type) {
	case *types.NewMarketConfiguration_Simple:
		return &SimpleRiskModel{
			Params: &SimpleRiskModelParams{
				FactorLong:  params.Simple.FactorLong,
				FactorShort: params.Simple.FactorShort,
			},
		}, nil
	case *types.NewMarketConfiguration_LogNormal:
		return &LogNormalRiskModel{
			RiskAversionParameter: params.LogNormal.RiskAversionParameter,
			Tau:                   params.LogNormal.Tau,
			Params: &LogNormalModelParams{
				Mu:    params.LogNormal.Params.Mu,
				R:     params.LogNormal.Params.R,
				Sigma: params.LogNormal.Params.Sigma,
			},
		}, nil
	default:
		return nil, ErrInvalidRiskConfiguration
	}
}

// IntoProto ...
func (i *InstrumentConfigurationInput) IntoProto() (*types.InstrumentConfiguration, error) {
	if len(i.Name) <= 0 {
		return nil, errors.New("Instrument.Name: string cannot be empty")
	}
	if len(i.Code) <= 0 {
		return nil, errors.New("Instrument.Code: string cannot be empty")
	}

	result := &types.InstrumentConfiguration{
		Name: i.Name,
		Code: i.Code,
	}

	if i.FutureProduct != nil {
		if len(i.FutureProduct.QuoteName) <= 0 {
			return nil, errors.New("FutureProduct.QuoteName: string cannot be empty")
		}
		if len(i.FutureProduct.SettlementAsset) <= 0 {
			return nil, errors.New("FutureProduct.Asset: string cannot be empty")
		}
		if len(i.FutureProduct.Maturity) <= 0 {
			return nil, errors.New("FutureProduct.Maturity: string cannot be empty")
		}

		spec, err := i.FutureProduct.OracleSpec.IntoProto()
		if err != nil {
			return nil, err
		}

		binding, err := i.FutureProduct.OracleSpecBinding.IntoProto()
		if err != nil {
			return nil, err
		}

		result.Product = &types.InstrumentConfiguration_Future{
			Future: &types.FutureProduct{
				SettlementAsset:   i.FutureProduct.SettlementAsset,
				Maturity:          i.FutureProduct.Maturity,
				QuoteName:         i.FutureProduct.QuoteName,
				OracleSpec:        spec,
				OracleSpecBinding: binding,
			},
		}
	} else {
		return nil, ErrNilProduct
	}
	return result, nil
}

// IntoProto ...
func (o *OracleSpecConfigurationInput) IntoProto() (*oraclesv1.OracleSpecConfiguration, error) {
	filters := []*oraclesv1.Filter{}
	for _, f := range o.Filters {
		typ, err := f.Key.Type.IntoProto()
		if err != nil {
			return nil, err
		}

		conditions := []*oraclesv1.Condition{}
		for _, c := range f.Conditions {
			op, err := c.Operator.IntoProto()
			if err != nil {
				return nil, err
			}

			conditions = append(conditions, &oraclesv1.Condition{
				Operator: op,
				Value:    c.Value,
			})
		}

		filters = append(filters, &oraclesv1.Filter{
			Key: &oraclesv1.PropertyKey{
				Name: f.Key.Name,
				Type: typ,
			},
			Conditions: conditions,
		})
	}

	return &oraclesv1.OracleSpecConfiguration{
		PubKeys: o.PubKeys,
		Filters: filters,
	}, nil
}

// IntoProto ...
func (t PropertyKeyType) IntoProto() (oraclesv1.PropertyKey_Type, error) {
	switch t {
	case PropertyKeyTypeTypeEmpty:
		return oraclesv1.PropertyKey_TYPE_EMPTY, nil
	case PropertyKeyTypeTypeInteger:
		return oraclesv1.PropertyKey_TYPE_INTEGER, nil
	case PropertyKeyTypeTypeDecimal:
		return oraclesv1.PropertyKey_TYPE_DECIMAL, nil
	case PropertyKeyTypeTypeBoolean:
		return oraclesv1.PropertyKey_TYPE_BOOLEAN, nil
	case PropertyKeyTypeTypeTimestamp:
		return oraclesv1.PropertyKey_TYPE_TIMESTAMP, nil
	case PropertyKeyTypeTypeString:
		return oraclesv1.PropertyKey_TYPE_STRING, nil
	default:
		err := fmt.Errorf("failed to convert PropertyKeyType from GraphQL to Proto: %v", t)
		return oraclesv1.PropertyKey_TYPE_EMPTY, err
	}
}

// IntoProto ...
func (o ConditionOperator) IntoProto() (oraclesv1.Condition_Operator, error) {
	switch o {
	case ConditionOperatorOperatorEquals:
		return oraclesv1.Condition_OPERATOR_EQUALS, nil
	case ConditionOperatorOperatorGreaterThan:
		return oraclesv1.Condition_OPERATOR_GREATER_THAN, nil
	case ConditionOperatorOperatorGreaterThanOrEqual:
		return oraclesv1.Condition_OPERATOR_GREATER_THAN_OR_EQUAL, nil
	case ConditionOperatorOperatorLessThan:
		return oraclesv1.Condition_OPERATOR_LESS_THAN, nil
	case ConditionOperatorOperatorLessThanOrEqual:
		return oraclesv1.Condition_OPERATOR_LESS_THAN_OR_EQUAL, nil
	default:
		err := fmt.Errorf("failed to convert ConditionOperator from Proto to GraphQL: %v", o)
		return oraclesv1.Condition_OPERATOR_EQUALS, err
	}
}

// IntoProto ...
func (o *OracleSpecToFutureBindingInput) IntoProto() (*types.OracleSpecToFutureBinding, error) {
	return nil, nil
}

// IntoProto ...
func (l *LogNormalModelParamsInput) IntoProto() (*types.LogNormalModelParams, error) {
	if l.Sigma < 0. {
		return nil, errors.New("LogNormalRiskModelParams.Sigma: needs to be any strictly non-negative float")
	}
	return &types.LogNormalModelParams{
		Mu:    l.Mu,
		R:     l.R,
		Sigma: l.Sigma,
	}, nil
}

// IntoProto ...
func (l *LogNormalRiskModelInput) IntoProto() (*types.NewMarketConfiguration_LogNormal, error) {
	if l.RiskAversionParameter <= 0. || l.RiskAversionParameter >= 1. {
		return nil, errors.New("LogNormalRiskModel.RiskAversionParameter: needs to be strictly greater than 0 and strictly smaller than 1")
	}
	if l.Tau < 0. {
		return nil, errors.New("LogNormalRiskModel.Tau: needs to be any strictly non-negative float")
	}

	params, err := l.Params.IntoProto()
	if err != nil {
		return nil, err
	}

	return &types.NewMarketConfiguration_LogNormal{
		LogNormal: &types.LogNormalRiskModel{
			RiskAversionParameter: l.RiskAversionParameter,
			Tau:                   l.Tau,
			Params:                params,
		},
	}, nil
}

// IntoProto ...
func (s *SimpleRiskModelParamsInput) IntoProto() *types.NewMarketConfiguration_Simple {
	return &types.NewMarketConfiguration_Simple{
		Simple: &types.SimpleModelParams{
			FactorLong:  s.FactorLong,
			FactorShort: s.FactorShort,
		},
	}
}

// IntoProto ...
func (r *RiskParametersInput) IntoProto(target *types.NewMarketConfiguration) error {
	if r.Simple != nil {
		target.RiskParameters = r.Simple.IntoProto()
		return nil
	} else if r.LogNormal != nil {
		var err error
		target.RiskParameters, err = r.LogNormal.IntoProto()
		return err
	}
	return ErrNilRiskModel
}

// TradingModeIntoProto ...
func (n *NewMarketInput) TradingModeIntoProto(target *types.NewMarketConfiguration) error {
	if n.ContinuousTrading != nil && n.DiscreteTrading != nil {
		return ErrAmbiguousTradingMode
	} else if n.ContinuousTrading == nil && n.DiscreteTrading == nil {
		return ErrNilTradingMode
	}

	// FIXME(): here both tickSize are being ignore as deprecated for now
	// they will be created internally by the core.
	if n.ContinuousTrading != nil {
		target.TradingMode = &types.NewMarketConfiguration_Continuous{
			Continuous: &types.ContinuousTrading{
				TickSize: "",
			},
		}
	} else if n.DiscreteTrading != nil {
		if n.DiscreteTrading.Duration <= 0 {
			return errors.New("DiscreteTrading.Duration: cannot be < 0")
		}
		target.TradingMode = &types.NewMarketConfiguration_Discrete{
			Discrete: &types.DiscreteTrading{
				DurationNs: int64(n.DiscreteTrading.Duration),
				TickSize:   "",
			},
		}
	}
	return nil
}

func (b *BuiltinAssetInput) IntoProto() (*types.BuiltinAsset, error) {
	if len(b.Name) <= 0 {
		return nil, errors.New("BuiltinAssetInput.Name: cannot be empty")
	}
	if len(b.Symbol) <= 0 {
		return nil, errors.New("BuiltinAssetInput.Symbol: cannot be empty")
	}
	if len(b.TotalSupply) <= 0 {
		return nil, errors.New("BuiltinAssetInput.TotalSupply: cannot be empty")
	}
	if len(b.MaxFaucetAmountMint) <= 0 {
		return nil, errors.New("BuiltinAssetInput.MaxFaucetAmountMint: cannot be empty")
	}
	if b.Decimals <= 0 {
		return nil, errors.New("BuiltinAssetInput.Decimals: cannot be <= 0")
	}

	return &types.BuiltinAsset{
		Name:                b.Name,
		Symbol:              b.Symbol,
		TotalSupply:         b.TotalSupply,
		Decimals:            uint64(b.Decimals),
		MaxFaucetAmountMint: b.MaxFaucetAmountMint,
	}, nil
}

func (e *ERC20Input) IntoProto() (*types.ERC20, error) {
	if len(e.ContractAddress) <= 0 {
		return nil, errors.New("ERC20.ContractAddress: cannot be empty")
	}

	return &types.ERC20{
		ContractAddress: e.ContractAddress,
	}, nil
}

func (n *NewAssetInput) IntoProto() (*types.AssetSource, error) {
	var (
		isSet       bool
		assetSource = &types.AssetSource{}
	)

	if n.BuiltinAsset != nil {
		isSet = true
		source, err := n.BuiltinAsset.IntoProto()
		if err != nil {
			return nil, err
		}
		assetSource.Source = &types.AssetSource_BuiltinAsset{
			BuiltinAsset: source,
		}
	}

	if n.Erc20 != nil {
		if isSet {
			return nil, ErrMultipleAssetSourcesSpecified
		}
		isSet = true
		source, err := n.Erc20.IntoProto()
		if err != nil {
			return nil, err
		}
		assetSource.Source = &types.AssetSource_Erc20{
			Erc20: source,
		}
	}

	return assetSource, nil
}

func (p *PriceMonitoringTriggerInput) IntoProto() *types.PriceMonitoringTrigger {
	return &types.PriceMonitoringTrigger{
		Horizon:          int64(p.HorizonSecs),
		Probability:      p.Probability,
		AuctionExtension: int64(p.AuctionExtensionSecs),
	}
}

func (p *PriceMonitoringParametersInput) IntoProto() (*types.PriceMonitoringParameters, error) {
	triggers := make([]*types.PriceMonitoringTrigger, 0, len(p.Triggers))

	for _, v := range p.Triggers {
		triggers = append(triggers, v.IntoProto())
	}

	return &types.PriceMonitoringParameters{
		Triggers: triggers,
	}, nil
}

func (p *PriceMonitoringSettingsInput) IntoProto() (*types.PriceMonitoringSettings, error) {
	var freq int
	if p.UpdateFrequencySecs != nil {
		freq = *p.UpdateFrequencySecs
	}

	params, err := p.Parameters.IntoProto()
	if err != nil {
		return nil, err
	}

	return &types.PriceMonitoringSettings{
		Parameters:      params,
		UpdateFrequency: int64(freq),
	}, nil
}

// IntoProto ...
func (n *NewMarketInput) IntoProto() (*types.NewMarketConfiguration, error) {
	if n.DecimalPlaces < 0 {
		return nil, errors.New("NewMarket.DecimalPlaces: needs to be > 0")
	}
	instrument, err := n.Instrument.IntoProto()
	if err != nil {
		return nil, err
	}

	result := &types.NewMarketConfiguration{
		Instrument:    instrument,
		DecimalPlaces: uint64(n.DecimalPlaces),
	}

	if err := n.RiskParameters.IntoProto(result); err != nil {
		return nil, err
	}
	if err := n.TradingModeIntoProto(result); err != nil {
		return nil, err
	}
	result.Metadata = append(result.Metadata, n.Metadata...)
	if n.PriceMonitoringParameters != nil {
		params, err := n.PriceMonitoringParameters.IntoProto()
		if err != nil {
			return nil, err
		}

		result.PriceMonitoringParameters = params
	} else {
		result.PriceMonitoringParameters = &types.PriceMonitoringParameters{}
	}

	return result, nil
}

// IntoProto ...
func (p ProposalTermsInput) IntoProto() (*types.ProposalTerms, error) {
	closing, err := datetimeToSecondsTS(p.ClosingDatetime)
	if err != nil {
		err = fmt.Errorf("ProposalTerms.ClosingDatetime: %s", err.Error())
		return nil, err
	}
	enactment, err := datetimeToSecondsTS(p.EnactmentDatetime)
	if err != nil {
		err = fmt.Errorf("ProposalTerms.EnactementDatetime: %s", err.Error())
		return nil, err
	}

	result := &types.ProposalTerms{
		ClosingTimestamp:   closing,
		EnactmentTimestamp: enactment,
	}

	// used to check if the user did not specify multiple ProposalChanges
	// which is an error
	var isSet bool

	if p.UpdateMarket != nil {
		isSet = true
		result.Change = &types.ProposalTerms_UpdateMarket{}
	}

	if p.NewMarket != nil {
		if isSet {
			return nil, ErrMultipleProposalChangesSpecified
		}
		isSet = true
		market, err := p.NewMarket.IntoProto()
		if err != nil {
			return nil, err
		}
		result.Change = &types.ProposalTerms_NewMarket{
			NewMarket: &types.NewMarket{
				Changes: market,
			},
		}
	}

	if p.NewAsset != nil {
		if isSet {
			return nil, ErrMultipleProposalChangesSpecified
		}
		isSet = true
		assetSource, err := p.NewAsset.IntoProto()
		if err != nil {
			return nil, err
		}
		result.Change = &types.ProposalTerms_NewAsset{
			NewAsset: &types.NewAsset{
				Changes: assetSource,
			},
		}
	}

	if p.UpdateNetworkParameter != nil {
		if isSet {
			return nil, ErrMultipleProposalChangesSpecified
		}
		isSet = true
		result.Change = &types.ProposalTerms_UpdateNetworkParameter{
			UpdateNetworkParameter: &types.UpdateNetworkParameter{
				Changes: p.UpdateNetworkParameter.NetworkParameter.IntoProto(),
			},
		}
	}
	if !isSet {
		return nil, ErrInvalidChange
	}

	return result, nil
}

func (n *NetworkParameterInput) IntoProto() *types.NetworkParameter {
	return &types.NetworkParameter{
		Key:   n.Key,
		Value: n.Value,
	}
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
	return convertProposalStateToProto(s)
}

// ProposalVoteFromProto ...
func ProposalVoteFromProto(v *types.Vote, caster *types.Party) *ProposalVote {
	value, _ := convertVoteValueFromProto(v.Value)
	return &ProposalVote{
		Vote: &Vote{
			Party:      caster,
			Value:      value,
			Datetime:   nanoTSToDatetime(v.Timestamp),
			ProposalID: v.ProposalId,
		},
		ProposalID: v.ProposalId,
	}
}

// IntoProto ...
func (a AccountType) IntoProto() types.AccountType {
	at, _ := convertAccountTypeToProto(a)
	return at
}

func BuiltinAssetFromProto(ba *types.BuiltinAsset) *BuiltinAsset {
	return &BuiltinAsset{
		Name:                ba.Name,
		Symbol:              ba.Symbol,
		TotalSupply:         ba.TotalSupply,
		Decimals:            int(ba.Decimals),
		MaxFaucetAmountMint: ba.MaxFaucetAmountMint,
	}
}

func ERC20FromProto(ea *types.ERC20) *Erc20 {
	return &Erc20{
		ContractAddress: ea.ContractAddress,
	}
}

func AssetSourceFromProto(psource *types.AssetSource) (AssetSource, error) {
	if psource == nil {
		return nil, ErrNilAssetSource
	}
	switch asimpl := psource.Source.(type) {
	case *types.AssetSource_BuiltinAsset:
		return BuiltinAssetFromProto(asimpl.BuiltinAsset), nil
	case *types.AssetSource_Erc20:
		return ERC20FromProto(asimpl.Erc20), nil
	default:
		return nil, ErrUnimplementedAssetSource
	}
}

func AssetFromProto(passet *types.Asset) (*Asset, error) {
	source, err := AssetSourceFromProto(passet.Source)
	if err != nil {
		return nil, err
	}

	return &Asset{
		ID:          passet.Id,
		Name:        passet.Name,
		Symbol:      passet.Symbol,
		Decimals:    int(passet.Decimals),
		TotalSupply: passet.TotalSupply,
		Source:      source,
	}, nil
}

func defaultRiskParameters() *types.NewMarketConfiguration_LogNormal {
	return &types.NewMarketConfiguration_LogNormal{
		LogNormal: &types.LogNormalRiskModel{
			RiskAversionParameter: 0,
			Tau:                   0,
			Params: &types.LogNormalModelParams{
				Mu:    0,
				R:     0,
				Sigma: 0,
			},
		},
	}
}

func (e *Erc20WithdrawalDetailsInput) IntoProtoExt() *types.WithdrawExt {
	return &types.WithdrawExt{
		Ext: &types.WithdrawExt_Erc20{
			Erc20: &types.Erc20WithdrawExt{
				ReceiverAddress: e.ReceiverAddress,
			},
		},
	}
}

func defaultTradingMode() *types.NewMarketConfiguration_Continuous {
	return &types.NewMarketConfiguration_Continuous{
		Continuous: &types.ContinuousTrading{
			TickSize: "0",
		},
	}
}

func busEventFromProto(events ...*types.BusEvent) []*BusEvent {
	r := make([]*BusEvent, 0, len(events))
	for _, e := range events {
		evt := eventFromProto(e)
		if evt == nil {
			// @TODO for now just skip unmapped event types, probably better to handle some kind of error
			// in the future though
			continue
		}
		et, err := eventTypeFromProto(e.Type)
		if err != nil {
			// @TODO for now just skip unmapped event types, probably better to handle some kind of error
			// in the future though
			continue
		}
		be := BusEvent{
			EventID: e.Id,
			Type:    et,
			Block:   e.Block,
			Event:   evt,
		}
		r = append(r, &be)
	}
	return r
}

func balancesFromProto(balances []*types.TransferBalance) []*TransferBalance {
	gql := make([]*TransferBalance, 0, len(balances))
	for _, b := range balances {
		gql = append(gql, &TransferBalance{
			Account: b.Account,
			Balance: int(b.Balance),
		})
	}
	return gql
}

func transfersFromProto(transfers []*types.LedgerEntry) []*LedgerEntry {
	gql := make([]*LedgerEntry, 0, len(transfers))
	for _, t := range transfers {
		gql = append(gql, &LedgerEntry{
			FromAccount: t.FromAccount,
			ToAccount:   t.ToAccount,
			Amount:      int(t.Amount),
			Reference:   t.Reference,
			Type:        t.Type,
			Timestamp:   nanoTSToDatetime(t.Timestamp),
		})
	}
	return gql
}

func auctionEventFromProto(ae *types.AuctionEvent) *AuctionEvent {
	t, _ := convertAuctionTriggerFromProto(ae.Trigger)
	r := &AuctionEvent{
		MarketID:       ae.MarketId,
		Leave:          ae.Leave,
		OpeningAuction: ae.OpeningAuction,
		AuctionStart:   nanoTSToDatetime(ae.Start),
		Trigger:        t,
	}
	if ae.End != 0 {
		r.AuctionEnd = nanoTSToDatetime(ae.End)
	}
	return r
}

func eventFromProto(e *types.BusEvent) Event {
	switch e.Type {
	case types.BusEventType_BUS_EVENT_TYPE_TIME_UPDATE:
		return &TimeUpdate{
			Timestamp: secondsTSToDatetime(e.GetTimeUpdate().Timestamp),
		}
	case types.BusEventType_BUS_EVENT_TYPE_TRANSFER_RESPONSES:
		tr := e.GetTransferResponses()
		responses := make([]*TransferResponse, 0, len(tr.Responses))
		for _, r := range tr.Responses {
			responses = append(responses, &TransferResponse{
				Transfers: transfersFromProto(r.Transfers),
				Balances:  balancesFromProto(r.Balances),
			})
		}
		return &TransferResponses{
			Responses: responses,
		}
	case types.BusEventType_BUS_EVENT_TYPE_POSITION_RESOLUTION:
		pr := e.GetPositionResolution()
		return &PositionResolution{
			MarketID:   pr.MarketId,
			Distressed: int(pr.Distressed),
			Closed:     int(pr.Closed),
			MarkPrice:  int(pr.MarkPrice),
		}
	case types.BusEventType_BUS_EVENT_TYPE_ORDER:
		return e.GetOrder()
	case types.BusEventType_BUS_EVENT_TYPE_ACCOUNT:
		return e.GetAccount()
	case types.BusEventType_BUS_EVENT_TYPE_PARTY:
		return e.GetParty()
	case types.BusEventType_BUS_EVENT_TYPE_TRADE:
		return e.GetTrade()
	case types.BusEventType_BUS_EVENT_TYPE_MARGIN_LEVELS:
		return e.GetMarginLevels()
	case types.BusEventType_BUS_EVENT_TYPE_PROPOSAL:
		return &types.GovernanceData{
			Proposal: e.GetProposal(),
		}
	case types.BusEventType_BUS_EVENT_TYPE_VOTE:
		v := e.GetVote()
		val, _ := convertVoteValueFromProto(v.Value)
		return &Vote{
			Value: val,
			Party: &types.Party{
				Id: v.PartyId,
			},
			Datetime:   nanoTSToDatetime(v.Timestamp),
			ProposalID: v.ProposalId,
		}
	case types.BusEventType_BUS_EVENT_TYPE_MARKET_DATA:
		return e.GetMarketData()
	case types.BusEventType_BUS_EVENT_TYPE_NODE_SIGNATURE:
		return e.GetNodeSignature()
	case types.BusEventType_BUS_EVENT_TYPE_LOSS_SOCIALIZATION:
		ls := e.GetLossSocialization()
		return &LossSocialization{
			MarketID: ls.MarketId,
			PartyID:  ls.PartyId,
			Amount:   int(ls.Amount),
		}
	case types.BusEventType_BUS_EVENT_TYPE_SETTLE_POSITION:
		dp := e.GetSettlePosition()
		settlements := make([]*TradeSettlement, 0, len(dp.TradeSettlements))
		for _, ts := range dp.TradeSettlements {
			settlements = append(settlements, &TradeSettlement{
				Size:  int(ts.Size),
				Price: int(ts.Price),
			})
		}
		return &SettlePosition{
			MarketID:         dp.MarketId,
			PartyID:          dp.PartyId,
			Price:            int(dp.Price),
			TradeSettlements: settlements,
		}
	case types.BusEventType_BUS_EVENT_TYPE_SETTLE_DISTRESSED:
		de := e.GetSettleDistressed()
		return &SettleDistressed{
			MarketID: de.MarketId,
			PartyID:  de.PartyId,
			Margin:   int(de.Margin),
			Price:    int(de.Price),
		}
	case types.BusEventType_BUS_EVENT_TYPE_MARKET_CREATED:
		return e.GetMarketCreated()
	case types.BusEventType_BUS_EVENT_TYPE_MARKET_UPDATED:
		return e.GetMarketUpdated()
	case types.BusEventType_BUS_EVENT_TYPE_ASSET:
		a, _ := AssetFromProto(e.GetAsset())
		return a
	case types.BusEventType_BUS_EVENT_TYPE_MARKET_TICK:
		mt := e.GetMarketTick()
		return &MarketTick{
			MarketID: mt.Id,
			Time:     secondsTSToDatetime(mt.Time),
		}
	case types.BusEventType_BUS_EVENT_TYPE_MARKET:
		pe := e.GetEvent()
		if pe == nil {
			return nil
		}
		me, ok := pe.(MarketLogEvent)
		if !ok {
			return nil
		}
		return &MarketEvent{
			MarketID: me.GetMarketID(),
			Payload:  me.GetPayload(),
		}
	case types.BusEventType_BUS_EVENT_TYPE_AUCTION:
		return auctionEventFromProto(e.GetAuction())
	case types.BusEventType_BUS_EVENT_TYPE_DEPOSIT:
		return e.GetDeposit()
	case types.BusEventType_BUS_EVENT_TYPE_WITHDRAWAL:
		return e.GetWithdrawal()
	case types.BusEventType_BUS_EVENT_TYPE_ORACLE_SPEC:
		return e.GetOracleSpec()
	}
	return nil
}

// func (_ GovernanceData) IsEvent() {}

func eventTypeToProto(btypes ...BusEventType) []types.BusEventType {
	r := make([]types.BusEventType, 0, len(btypes))
	for _, t := range btypes {
		switch t {
		case BusEventTypeTimeUpdate:
			r = append(r, types.BusEventType_BUS_EVENT_TYPE_TIME_UPDATE)
		case BusEventTypeTransferResponses:
			r = append(r, types.BusEventType_BUS_EVENT_TYPE_TRANSFER_RESPONSES)
		case BusEventTypePositionResolution:
			r = append(r, types.BusEventType_BUS_EVENT_TYPE_POSITION_RESOLUTION)
		case BusEventTypeOrder:
			r = append(r, types.BusEventType_BUS_EVENT_TYPE_ORDER)
		case BusEventTypeAccount:
			r = append(r, types.BusEventType_BUS_EVENT_TYPE_ACCOUNT)
		case BusEventTypeParty:
			r = append(r, types.BusEventType_BUS_EVENT_TYPE_PARTY)
		case BusEventTypeTrade:
			r = append(r, types.BusEventType_BUS_EVENT_TYPE_TRADE)
		case BusEventTypeMarginLevels:
			r = append(r, types.BusEventType_BUS_EVENT_TYPE_MARGIN_LEVELS)
		case BusEventTypeProposal:
			r = append(r, types.BusEventType_BUS_EVENT_TYPE_PROPOSAL)
		case BusEventTypeVote:
			r = append(r, types.BusEventType_BUS_EVENT_TYPE_VOTE)
		case BusEventTypeMarketData:
			r = append(r, types.BusEventType_BUS_EVENT_TYPE_MARKET_DATA)
		case BusEventTypeNodeSignature:
			r = append(r, types.BusEventType_BUS_EVENT_TYPE_NODE_SIGNATURE)
		case BusEventTypeLossSocialization:
			r = append(r, types.BusEventType_BUS_EVENT_TYPE_LOSS_SOCIALIZATION)
		case BusEventTypeSettlePosition:
			r = append(r, types.BusEventType_BUS_EVENT_TYPE_SETTLE_POSITION)
		case BusEventTypeSettleDistressed:
			r = append(r, types.BusEventType_BUS_EVENT_TYPE_SETTLE_DISTRESSED)
		case BusEventTypeMarketCreated:
			r = append(r, types.BusEventType_BUS_EVENT_TYPE_MARKET_CREATED)
		case BusEventTypeMarketUpdated:
			r = append(r, types.BusEventType_BUS_EVENT_TYPE_MARKET_UPDATED)
		case BusEventTypeAsset:
			r = append(r, types.BusEventType_BUS_EVENT_TYPE_ASSET)
		case BusEventTypeMarketTick:
			r = append(r, types.BusEventType_BUS_EVENT_TYPE_MARKET_TICK)
		case BusEventTypeMarket:
			r = append(r, types.BusEventType_BUS_EVENT_TYPE_MARKET)
		case BusEventTypeAuction:
			r = append(r, types.BusEventType_BUS_EVENT_TYPE_AUCTION)
		case BusEventTypeRiskFactor:
			r = append(r, types.BusEventType_BUS_EVENT_TYPE_RISK_FACTOR)
		case BusEventTypeLiquidityProvision:
			r = append(r, types.BusEventType_BUS_EVENT_TYPE_LIQUIDITY_PROVISION)
		case BusEventTypeDeposit:
			r = append(r, types.BusEventType_BUS_EVENT_TYPE_DEPOSIT)
		case BusEventTypeWithdrawal:
			r = append(r, types.BusEventType_BUS_EVENT_TYPE_WITHDRAWAL)
		case BusEventTypeOracleSpec:
			r = append(r, types.BusEventType_BUS_EVENT_TYPE_ORACLE_SPEC)
		}
	}
	return r
}

func eventTypeFromProto(t types.BusEventType) (BusEventType, error) {
	switch t {
	case types.BusEventType_BUS_EVENT_TYPE_TIME_UPDATE:
		return BusEventTypeTimeUpdate, nil
	case types.BusEventType_BUS_EVENT_TYPE_TRANSFER_RESPONSES:
		return BusEventTypeTransferResponses, nil
	case types.BusEventType_BUS_EVENT_TYPE_POSITION_RESOLUTION:
		return BusEventTypePositionResolution, nil
	case types.BusEventType_BUS_EVENT_TYPE_ORDER:
		return BusEventTypeOrder, nil
	case types.BusEventType_BUS_EVENT_TYPE_ACCOUNT:
		return BusEventTypeAccount, nil
	case types.BusEventType_BUS_EVENT_TYPE_PARTY:
		return BusEventTypeParty, nil
	case types.BusEventType_BUS_EVENT_TYPE_TRADE:
		return BusEventTypeTrade, nil
	case types.BusEventType_BUS_EVENT_TYPE_MARGIN_LEVELS:
		return BusEventTypeMarginLevels, nil
	case types.BusEventType_BUS_EVENT_TYPE_PROPOSAL:
		return BusEventTypeProposal, nil
	case types.BusEventType_BUS_EVENT_TYPE_VOTE:
		return BusEventTypeVote, nil
	case types.BusEventType_BUS_EVENT_TYPE_MARKET_DATA:
		return BusEventTypeMarketData, nil
	case types.BusEventType_BUS_EVENT_TYPE_NODE_SIGNATURE:
		return BusEventTypeNodeSignature, nil
	case types.BusEventType_BUS_EVENT_TYPE_LOSS_SOCIALIZATION:
		return BusEventTypeLossSocialization, nil
	case types.BusEventType_BUS_EVENT_TYPE_SETTLE_POSITION:
		return BusEventTypeSettlePosition, nil
	case types.BusEventType_BUS_EVENT_TYPE_SETTLE_DISTRESSED:
		return BusEventTypeSettleDistressed, nil
	case types.BusEventType_BUS_EVENT_TYPE_MARKET_CREATED:
		return BusEventTypeMarketCreated, nil
	case types.BusEventType_BUS_EVENT_TYPE_MARKET_UPDATED:
		return BusEventTypeMarketUpdated, nil
	case types.BusEventType_BUS_EVENT_TYPE_ASSET:
		return BusEventTypeAsset, nil
	case types.BusEventType_BUS_EVENT_TYPE_MARKET_TICK:
		return BusEventTypeMarketTick, nil
	case types.BusEventType_BUS_EVENT_TYPE_MARKET:
		return BusEventTypeMarket, nil
	case types.BusEventType_BUS_EVENT_TYPE_AUCTION:
		return BusEventTypeAuction, nil
	case types.BusEventType_BUS_EVENT_TYPE_RISK_FACTOR:
		return BusEventTypeRiskFactor, nil
	case types.BusEventType_BUS_EVENT_TYPE_LIQUIDITY_PROVISION:
		return BusEventTypeLiquidityProvision, nil
	case types.BusEventType_BUS_EVENT_TYPE_DEPOSIT:
		return BusEventTypeDeposit, nil
	case types.BusEventType_BUS_EVENT_TYPE_WITHDRAWAL:
		return BusEventTypeWithdrawal, nil
	case types.BusEventType_BUS_EVENT_TYPE_ORACLE_SPEC:
		return BusEventTypeOracleSpec, nil
	}
	return "", errors.New("unsupported proto event type")
}
