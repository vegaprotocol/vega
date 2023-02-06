// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package gql

import (
	"errors"
	"strconv"

	types "code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
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
	// ErrNilAssetSource returned when an asset source is not specified at creation.
	ErrNilAssetSource = errors.New("nil asset source")
	// ErrUnimplementedAssetSource returned when an asset source specified at creation is not recognised.
	ErrUnimplementedAssetSource = errors.New("unimplemented asset source")
	// ErrMultipleProposalChangesSpecified is raised when multiple proposal changes are set
	// (non-null) for a singe proposal terms.
	ErrMultipleProposalChangesSpecified = errors.New("multiple proposal changes specified")
	// ErrMultipleAssetSourcesSpecified is raised when multiple asset source are specified.
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
		Parameters: params,
	}, nil
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
			ID:    e.Id,
			Type:  et,
			Block: e.Block,
			Event: evt,
		}
		r = append(r, &be)
	}
	return r
}

func eventFromProto(e *eventspb.BusEvent) Event {
	switch e.Type {
	case eventspb.BusEventType_BUS_EVENT_TYPE_TIME_UPDATE:
		return &TimeUpdate{
			Timestamp: e.GetTimeUpdate().Timestamp,
		}
	case eventspb.BusEventType_BUS_EVENT_TYPE_DEPOSIT:
		return e.GetDeposit()
	case eventspb.BusEventType_BUS_EVENT_TYPE_WITHDRAWAL:
		return e.GetWithdrawal()
	case eventspb.BusEventType_BUS_EVENT_TYPE_TRANSACTION_RESULT:
		return e.GetTransactionResult()
	}
	return nil
}

func eventTypeToProto(btypes ...BusEventType) []eventspb.BusEventType {
	r := make([]eventspb.BusEventType, 0, len(btypes))
	for _, t := range btypes {
		switch t {
		case BusEventTypeTimeUpdate:
			r = append(r, eventspb.BusEventType_BUS_EVENT_TYPE_TIME_UPDATE)
			r = append(r, eventspb.BusEventType_BUS_EVENT_TYPE_LIQUIDITY_PROVISION)
		case BusEventTypeDeposit:
			r = append(r, eventspb.BusEventType_BUS_EVENT_TYPE_DEPOSIT)
		case BusEventTypeWithdrawal:
			r = append(r, eventspb.BusEventType_BUS_EVENT_TYPE_WITHDRAWAL)
		case BusEventTypeTransactionResult:
			r = append(r, eventspb.BusEventType_BUS_EVENT_TYPE_TRANSACTION_RESULT)
		}
	}
	return r
}

func eventTypeFromProto(t eventspb.BusEventType) (BusEventType, error) {
	switch t {
	case eventspb.BusEventType_BUS_EVENT_TYPE_TIME_UPDATE:
		return BusEventTypeTimeUpdate, nil
	case eventspb.BusEventType_BUS_EVENT_TYPE_DEPOSIT:
		return BusEventTypeDeposit, nil
	case eventspb.BusEventType_BUS_EVENT_TYPE_WITHDRAWAL:
		return BusEventTypeWithdrawal, nil
	case eventspb.BusEventType_BUS_EVENT_TYPE_TRANSACTION_RESULT:
		return BusEventTypeTransactionResult, nil
	}
	return "", errors.New("unsupported proto event type")
}
