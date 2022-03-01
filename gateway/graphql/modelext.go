package gql

import (
	"errors"
	"fmt"
	"strconv"

	protoapi "code.vegaprotocol.io/protos/data-node/api/v1"
	types "code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	oraclesv1 "code.vegaprotocol.io/protos/vega/oracles/v1"
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

func PriceMonitoringTriggerFromProto(ppmt *types.PriceMonitoringTrigger) (*PriceMonitoringTrigger, error) {
	probability, err := strconv.ParseFloat(ppmt.Probability, 64)

	if err != nil {
		return nil, err
	}

	return &PriceMonitoringTrigger{
		HorizonSecs:          int(ppmt.Horizon),
		Probability:          probability,
		AuctionExtensionSecs: int(ppmt.AuctionExtension),
	}, nil
}

func PriceMonitoringParametersFromProto(ppmp *types.PriceMonitoringParameters) (*PriceMonitoringParameters, error) {
	if ppmp == nil {
		return nil, ErrNilPriceMonitoringParameters
	}

	triggers := make([]*PriceMonitoringTrigger, 0, len(ppmp.Triggers))
	for _, v := range ppmp.Triggers {
		trigger, err := PriceMonitoringTriggerFromProto(v)
		if err != nil {
			return nil, err
		}
		triggers = append(triggers, trigger)
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
			Balance: b.Balance,
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
			Amount:      t.Amount,
			Reference:   t.Reference,
			Type:        t.Type,
			Timestamp:   nanoTSToDatetime(t.Timestamp),
		})
	}
	return gql
}

func eventFromProto(e *eventspb.BusEvent) Event {
	switch e.Type {
	case eventspb.BusEventType_BUS_EVENT_TYPE_RISK_FACTOR:
		rf := e.GetRiskFactor()
		return rf
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
			MarkPrice:  pr.MarkPrice,
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
			Amount:   ls.Amount,
		}
	case eventspb.BusEventType_BUS_EVENT_TYPE_SETTLE_POSITION:
		dp := e.GetSettlePosition()
		settlements := make([]*TradeSettlement, 0, len(dp.TradeSettlements))
		for _, ts := range dp.TradeSettlements {
			settlements = append(settlements, &TradeSettlement{
				Size:  int(ts.Size),
				Price: ts.Price,
			})
		}
		return &SettlePosition{
			MarketID:         dp.MarketId,
			PartyID:          dp.PartyId,
			Price:            dp.Price,
			TradeSettlements: settlements,
		}
	case eventspb.BusEventType_BUS_EVENT_TYPE_SETTLE_DISTRESSED:
		de := e.GetSettleDistressed()
		return &SettleDistressed{
			MarketID: de.MarketId,
			PartyID:  de.PartyId,
			Margin:   de.Margin,
			Price:    de.Price,
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
