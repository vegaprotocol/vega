package gql

import (
	"errors"
	"fmt"
	"strconv"

	types "code.vegaprotocol.io/vega/proto"
	protoapi "code.vegaprotocol.io/vega/proto/api"
	eventspb "code.vegaprotocol.io/vega/proto/events/v1"
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
	// ErrInvalidChange ...
	ErrInvalidChange = errors.New("nil update market, new market and update network")
	// ErrNilAssetSource returned when an asset source is not specified at creation
	ErrNilAssetSource = errors.New("nil asset source")
	// ErrUnimplementedAssetSource returned when an asset source specified at creation is not recognised
	ErrUnimplementedAssetSource = errors.New("unimplemented asset source")
	// ErrMultipleProposalChangesSpecified is raised when multiple proposal changes are set
	// (non-null) for a singe proposal terms
	ErrMultipleProposalChangesSpecified = errors.New("multiple proposal changes specified")
	// ErrMultipleAssetSourcesSpecified is raised when multiple asset source are specified
	ErrMultipleAssetSourcesSpecified = errors.New("multiple asset sources specified")
	// ErrNilPriceMonitoringParameters ...
	ErrNilPriceMonitoringParameters = errors.New("nil price monitoring parameters")
)

type MarketLogEvent interface {
	GetMarketID() string
	GetPayload() string
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
	return &types.OracleSpecToFutureBinding{
		SettlementPriceProperty: o.SettlementPriceProperty,
	}, nil
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
	if len(b.MaxFaucetAmountMint) <= 0 {
		return nil, errors.New("BuiltinAssetInput.MaxFaucetAmountMint: cannot be empty")
	}
	return &types.BuiltinAsset{
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

func (n *NewAssetInput) IntoProto() (*types.AssetDetails, error) {

	if len(n.Name) <= 0 {
		return nil, errors.New("NewAssetInput.Name: cannot be empty")
	}
	if len(n.Symbol) <= 0 {
		return nil, errors.New("NewAssetInput.Symbol: cannot be empty")
	}
	if len(n.TotalSupply) <= 0 {
		return nil, errors.New("NewAssetInput.TotalSupply: cannot be empty")
	}
	if n.Decimals <= 0 {
		return nil, errors.New("NewAssetInput.Decimals: cannot be <= 0")
	}
	if len(n.MinLpStake) <= 0 {
		return nil, errors.New("NewAssetInput.Decimals: cannot be <= 0")
	}

	var (
		isSet        bool
		assetDetails = &types.AssetDetails{
			Name:        n.Name,
			Symbol:      n.Symbol,
			TotalSupply: n.TotalSupply,
			Decimals:    uint64(n.Decimals),
			MinLpStake:  n.MinLpStake,
		}
	)

	if n.BuiltinAsset != nil {
		isSet = true
		source, err := n.BuiltinAsset.IntoProto()
		if err != nil {
			return nil, err
		}
		assetDetails.Source = &types.AssetDetails_BuiltinAsset{
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
		assetDetails.Source = &types.AssetDetails_Erc20{
			Erc20: source,
		}
	}

	return assetDetails, nil
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

// ProtoCommitment ...
func (n *NewMarketInput) ProtoCommitment() (*types.NewMarketCommitment, error) {
	// FIXME(JEREMY): commitment are optional for now.
	// this check will need to be changed for an error if it's nil
	// in the future
	if n.Commitment == nil {
		return nil, nil
	}

	c := n.Commitment

	amount, err := strconv.ParseUint(c.CommitmentAmount, 10, 64)
	if err != nil || amount == 0 {
		return nil, errors.New("NewMarket.CommitmentAmount: needs to be > 0")
	}
	fee, err := strconv.ParseFloat(c.CommitmentAmount, 64)
	if err != nil || fee < 0 {
		return nil, errors.New("NewMarket.Fee: needs to be >= 0")
	}

	var reference string
	if c.Reference != nil {
		reference = *c.Reference
	}

	if len(c.Buys) <= 0 {
		return nil, errors.New("NewMarket.Buys: cannot be empty")
	}
	if len(c.Sells) <= 0 {
		return nil, errors.New("NewMarket.Sells: cannot be empty")
	}

	buys, err := LiquidityOrderInputs(c.Buys).IntoProto()
	if err != nil {
		return nil, err
	}
	sells, err := LiquidityOrderInputs(c.Sells).IntoProto()
	if err != nil {
		return nil, err
	}

	return &types.NewMarketCommitment{
		CommitmentAmount: amount,
		Fee:              c.Fee,
		Reference:        reference,
		Sells:            sells,
		Buys:             buys,
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
		commitment, err := p.NewMarket.ProtoCommitment()
		if err != nil {
			return nil, err
		}
		result.Change = &types.ProposalTerms_NewMarket{
			NewMarket: &types.NewMarket{
				Changes:             market,
				LiquidityCommitment: commitment,
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
func ProposalVoteFromProto(v *types.Vote) *ProposalVote {
	return &ProposalVote{
		Vote:       v,
		ProposalID: v.ProposalId,
	}
}

// IntoProto ...
func (a AccountType) IntoProto() types.AccountType {
	at, _ := convertAccountTypeToProto(a)
	return at
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

func busEventFromProto(events ...*eventspb.BusEvent) []*BusEvent {
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

func eventFromProto(e *eventspb.BusEvent) Event {
	switch e.Type {
	case eventspb.BusEventType_BUS_EVENT_TYPE_TIME_UPDATE:
		return &TimeUpdate{
			Timestamp: secondsTSToDatetime(e.GetTimeUpdate().Timestamp),
		}
	case eventspb.BusEventType_BUS_EVENT_TYPE_TRANSFER_RESPONSES:
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
	case eventspb.BusEventType_BUS_EVENT_TYPE_POSITION_RESOLUTION:
		pr := e.GetPositionResolution()
		return &PositionResolution{
			MarketID:   pr.MarketId,
			Distressed: int(pr.Distressed),
			Closed:     int(pr.Closed),
			MarkPrice:  int(pr.MarkPrice),
		}
	case eventspb.BusEventType_BUS_EVENT_TYPE_ORDER:
		return e.GetOrder()
	case eventspb.BusEventType_BUS_EVENT_TYPE_ACCOUNT:
		return e.GetAccount()
	case eventspb.BusEventType_BUS_EVENT_TYPE_PARTY:
		return e.GetParty()
	case eventspb.BusEventType_BUS_EVENT_TYPE_TRADE:
		return e.GetTrade()
	case eventspb.BusEventType_BUS_EVENT_TYPE_MARGIN_LEVELS:
		return e.GetMarginLevels()
	case eventspb.BusEventType_BUS_EVENT_TYPE_PROPOSAL:
		return &types.GovernanceData{
			Proposal: e.GetProposal(),
		}
	case eventspb.BusEventType_BUS_EVENT_TYPE_VOTE:
		return e.GetVote()
	case eventspb.BusEventType_BUS_EVENT_TYPE_MARKET_DATA:
		return e.GetMarketData()
	case eventspb.BusEventType_BUS_EVENT_TYPE_NODE_SIGNATURE:
		return e.GetNodeSignature()
	case eventspb.BusEventType_BUS_EVENT_TYPE_LOSS_SOCIALIZATION:
		ls := e.GetLossSocialization()
		return &LossSocialization{
			MarketID: ls.MarketId,
			PartyID:  ls.PartyId,
			Amount:   int(ls.Amount),
		}
	case eventspb.BusEventType_BUS_EVENT_TYPE_SETTLE_POSITION:
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
	case eventspb.BusEventType_BUS_EVENT_TYPE_SETTLE_DISTRESSED:
		de := e.GetSettleDistressed()
		return &SettleDistressed{
			MarketID: de.MarketId,
			PartyID:  de.PartyId,
			Margin:   int(de.Margin),
			Price:    int(de.Price),
		}
	case eventspb.BusEventType_BUS_EVENT_TYPE_MARKET_CREATED:
		return e.GetMarketCreated()
	case eventspb.BusEventType_BUS_EVENT_TYPE_MARKET_UPDATED:
		return e.GetMarketUpdated()
	case eventspb.BusEventType_BUS_EVENT_TYPE_ASSET:
		return e.GetAsset()
	case eventspb.BusEventType_BUS_EVENT_TYPE_MARKET_TICK:
		mt := e.GetMarketTick()
		return &MarketTick{
			MarketID: mt.Id,
			Time:     secondsTSToDatetime(mt.Time),
		}
	case eventspb.BusEventType_BUS_EVENT_TYPE_MARKET:
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
	case eventspb.BusEventType_BUS_EVENT_TYPE_AUCTION:
		return e.GetAuction()
	case eventspb.BusEventType_BUS_EVENT_TYPE_DEPOSIT:
		return e.GetDeposit()
	case eventspb.BusEventType_BUS_EVENT_TYPE_WITHDRAWAL:
		return e.GetWithdrawal()
	case eventspb.BusEventType_BUS_EVENT_TYPE_ORACLE_SPEC:
		return e.GetOracleSpec()
	case eventspb.BusEventType_BUS_EVENT_TYPE_LIQUIDITY_PROVISION:
		return e.GetLiquidityProvision()
	}
	return nil
}

// func (_ GovernanceData) IsEvent() {}

func eventTypeToProto(btypes ...BusEventType) []eventspb.BusEventType {
	r := make([]eventspb.BusEventType, 0, len(btypes))
	for _, t := range btypes {
		switch t {
		case BusEventTypeTimeUpdate:
			r = append(r, eventspb.BusEventType_BUS_EVENT_TYPE_TIME_UPDATE)
		case BusEventTypeTransferResponses:
			r = append(r, eventspb.BusEventType_BUS_EVENT_TYPE_TRANSFER_RESPONSES)
		case BusEventTypePositionResolution:
			r = append(r, eventspb.BusEventType_BUS_EVENT_TYPE_POSITION_RESOLUTION)
		case BusEventTypeOrder:
			r = append(r, eventspb.BusEventType_BUS_EVENT_TYPE_ORDER)
		case BusEventTypeAccount:
			r = append(r, eventspb.BusEventType_BUS_EVENT_TYPE_ACCOUNT)
		case BusEventTypeParty:
			r = append(r, eventspb.BusEventType_BUS_EVENT_TYPE_PARTY)
		case BusEventTypeTrade:
			r = append(r, eventspb.BusEventType_BUS_EVENT_TYPE_TRADE)
		case BusEventTypeMarginLevels:
			r = append(r, eventspb.BusEventType_BUS_EVENT_TYPE_MARGIN_LEVELS)
		case BusEventTypeProposal:
			r = append(r, eventspb.BusEventType_BUS_EVENT_TYPE_PROPOSAL)
		case BusEventTypeVote:
			r = append(r, eventspb.BusEventType_BUS_EVENT_TYPE_VOTE)
		case BusEventTypeMarketData:
			r = append(r, eventspb.BusEventType_BUS_EVENT_TYPE_MARKET_DATA)
		case BusEventTypeNodeSignature:
			r = append(r, eventspb.BusEventType_BUS_EVENT_TYPE_NODE_SIGNATURE)
		case BusEventTypeLossSocialization:
			r = append(r, eventspb.BusEventType_BUS_EVENT_TYPE_LOSS_SOCIALIZATION)
		case BusEventTypeSettlePosition:
			r = append(r, eventspb.BusEventType_BUS_EVENT_TYPE_SETTLE_POSITION)
		case BusEventTypeSettleDistressed:
			r = append(r, eventspb.BusEventType_BUS_EVENT_TYPE_SETTLE_DISTRESSED)
		case BusEventTypeMarketCreated:
			r = append(r, eventspb.BusEventType_BUS_EVENT_TYPE_MARKET_CREATED)
		case BusEventTypeMarketUpdated:
			r = append(r, eventspb.BusEventType_BUS_EVENT_TYPE_MARKET_UPDATED)
		case BusEventTypeAsset:
			r = append(r, eventspb.BusEventType_BUS_EVENT_TYPE_ASSET)
		case BusEventTypeMarketTick:
			r = append(r, eventspb.BusEventType_BUS_EVENT_TYPE_MARKET_TICK)
		case BusEventTypeMarket:
			r = append(r, eventspb.BusEventType_BUS_EVENT_TYPE_MARKET)
		case BusEventTypeAuction:
			r = append(r, eventspb.BusEventType_BUS_EVENT_TYPE_AUCTION)
		case BusEventTypeRiskFactor:
			r = append(r, eventspb.BusEventType_BUS_EVENT_TYPE_RISK_FACTOR)
		case BusEventTypeLiquidityProvision:
			r = append(r, eventspb.BusEventType_BUS_EVENT_TYPE_LIQUIDITY_PROVISION)
		case BusEventTypeDeposit:
			r = append(r, eventspb.BusEventType_BUS_EVENT_TYPE_DEPOSIT)
		case BusEventTypeWithdrawal:
			r = append(r, eventspb.BusEventType_BUS_EVENT_TYPE_WITHDRAWAL)
		case BusEventTypeOracleSpec:
			r = append(r, eventspb.BusEventType_BUS_EVENT_TYPE_ORACLE_SPEC)
		}
	}
	return r
}

func eventTypeFromProto(t eventspb.BusEventType) (BusEventType, error) {
	switch t {
	case eventspb.BusEventType_BUS_EVENT_TYPE_TIME_UPDATE:
		return BusEventTypeTimeUpdate, nil
	case eventspb.BusEventType_BUS_EVENT_TYPE_TRANSFER_RESPONSES:
		return BusEventTypeTransferResponses, nil
	case eventspb.BusEventType_BUS_EVENT_TYPE_POSITION_RESOLUTION:
		return BusEventTypePositionResolution, nil
	case eventspb.BusEventType_BUS_EVENT_TYPE_ORDER:
		return BusEventTypeOrder, nil
	case eventspb.BusEventType_BUS_EVENT_TYPE_ACCOUNT:
		return BusEventTypeAccount, nil
	case eventspb.BusEventType_BUS_EVENT_TYPE_PARTY:
		return BusEventTypeParty, nil
	case eventspb.BusEventType_BUS_EVENT_TYPE_TRADE:
		return BusEventTypeTrade, nil
	case eventspb.BusEventType_BUS_EVENT_TYPE_MARGIN_LEVELS:
		return BusEventTypeMarginLevels, nil
	case eventspb.BusEventType_BUS_EVENT_TYPE_PROPOSAL:
		return BusEventTypeProposal, nil
	case eventspb.BusEventType_BUS_EVENT_TYPE_VOTE:
		return BusEventTypeVote, nil
	case eventspb.BusEventType_BUS_EVENT_TYPE_MARKET_DATA:
		return BusEventTypeMarketData, nil
	case eventspb.BusEventType_BUS_EVENT_TYPE_NODE_SIGNATURE:
		return BusEventTypeNodeSignature, nil
	case eventspb.BusEventType_BUS_EVENT_TYPE_LOSS_SOCIALIZATION:
		return BusEventTypeLossSocialization, nil
	case eventspb.BusEventType_BUS_EVENT_TYPE_SETTLE_POSITION:
		return BusEventTypeSettlePosition, nil
	case eventspb.BusEventType_BUS_EVENT_TYPE_SETTLE_DISTRESSED:
		return BusEventTypeSettleDistressed, nil
	case eventspb.BusEventType_BUS_EVENT_TYPE_MARKET_CREATED:
		return BusEventTypeMarketCreated, nil
	case eventspb.BusEventType_BUS_EVENT_TYPE_MARKET_UPDATED:
		return BusEventTypeMarketUpdated, nil
	case eventspb.BusEventType_BUS_EVENT_TYPE_ASSET:
		return BusEventTypeAsset, nil
	case eventspb.BusEventType_BUS_EVENT_TYPE_MARKET_TICK:
		return BusEventTypeMarketTick, nil
	case eventspb.BusEventType_BUS_EVENT_TYPE_MARKET:
		return BusEventTypeMarket, nil
	case eventspb.BusEventType_BUS_EVENT_TYPE_AUCTION:
		return BusEventTypeAuction, nil
	case eventspb.BusEventType_BUS_EVENT_TYPE_RISK_FACTOR:
		return BusEventTypeRiskFactor, nil
	case eventspb.BusEventType_BUS_EVENT_TYPE_LIQUIDITY_PROVISION:
		return BusEventTypeLiquidityProvision, nil
	case eventspb.BusEventType_BUS_EVENT_TYPE_DEPOSIT:
		return BusEventTypeDeposit, nil
	case eventspb.BusEventType_BUS_EVENT_TYPE_WITHDRAWAL:
		return BusEventTypeWithdrawal, nil
	case eventspb.BusEventType_BUS_EVENT_TYPE_ORACLE_SPEC:
		return BusEventTypeOracleSpec, nil
	}
	return "", errors.New("unsupported proto event type")
}
