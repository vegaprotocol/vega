// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package marshallers

import (
	"errors"
	"fmt"
	"io"
	"strconv"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/99designs/gqlgen/graphql"
)

var ErrUnimplemented = errors.New("unmarshaller not implemented as this API is query only")

func MarshalIndividualScope(t vega.IndividualScope) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(t.String())))
	})
}

func UnmarshalIndividualScope(v interface{}) (vega.IndividualScope, error) {
	s, ok := v.(string)
	if !ok {
		return vega.IndividualScope_INDIVIDUAL_SCOPE_UNSPECIFIED, fmt.Errorf("expected individual scope to be a string")
	}

	t, ok := vega.IndividualScope_value[s]
	if !ok {
		return vega.IndividualScope_INDIVIDUAL_SCOPE_UNSPECIFIED, fmt.Errorf("failed to convert IndividualScope from GraphQL to Proto: %v", s)
	}

	return vega.IndividualScope(t), nil
}

func MarshalDistributionStrategy(t vega.DistributionStrategy) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(t.String())))
	})
}

func UnmarshalDistributionStrategy(v interface{}) (vega.DistributionStrategy, error) {
	s, ok := v.(string)
	if !ok {
		return vega.DistributionStrategy_DISTRIBUTION_STRATEGY_UNSPECIFIED, fmt.Errorf("expected distribution strategy to be a string")
	}

	t, ok := vega.DistributionStrategy_value[s]
	if !ok {
		return vega.DistributionStrategy_DISTRIBUTION_STRATEGY_UNSPECIFIED, fmt.Errorf("failed to convert DistributionStrategy from GraphQL to Proto: %v", s)
	}

	return vega.DistributionStrategy(t), nil
}

func MarshalEntityScope(t vega.EntityScope) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(t.String())))
	})
}

func UnmarshalEntityScope(v interface{}) (vega.EntityScope, error) {
	s, ok := v.(string)
	if !ok {
		return vega.EntityScope_ENTITY_SCOPE_UNSPECIFIED, fmt.Errorf("expected entity scope to be a string")
	}

	t, ok := vega.EntityScope_value[s]
	if !ok {
		return vega.EntityScope_ENTITY_SCOPE_UNSPECIFIED, fmt.Errorf("failed to convert EntityScope from GraphQL to Proto: %v", s)
	}

	return vega.EntityScope(t), nil
}

func MarshalAccountType(t vega.AccountType) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(t.String())))
	})
}

func UnmarshalAccountType(v interface{}) (vega.AccountType, error) {
	s, ok := v.(string)
	if !ok {
		return vega.AccountType_ACCOUNT_TYPE_UNSPECIFIED, fmt.Errorf("expected account type to be a string")
	}

	t, ok := vega.AccountType_value[s]
	if !ok {
		return vega.AccountType_ACCOUNT_TYPE_UNSPECIFIED, fmt.Errorf("failed to convert AccountType from GraphQL to Proto: %v", s)
	}

	return vega.AccountType(t), nil
}

func MarshalSide(s vega.Side) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalSide(v interface{}) (vega.Side, error) {
	s, ok := v.(string)
	if !ok {
		return vega.Side_SIDE_UNSPECIFIED, fmt.Errorf("expected account type to be a string")
	}

	side, ok := vega.Side_value[s]
	if !ok {
		return vega.Side_SIDE_UNSPECIFIED, fmt.Errorf("failed to convert AccountType from GraphQL to Proto: %v", s)
	}

	return vega.Side(side), nil
}

func MarshalProposalState(s vega.Proposal_State) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalProposalState(v interface{}) (vega.Proposal_State, error) {
	s, ok := v.(string)
	if !ok {
		return vega.Proposal_STATE_UNSPECIFIED, fmt.Errorf("expected proposal state to be a string")
	}

	side, ok := vega.Proposal_State_value[s]
	if !ok {
		return vega.Proposal_STATE_UNSPECIFIED, fmt.Errorf("failed to convert ProposalState from GraphQL to Proto: %v", s)
	}

	return vega.Proposal_State(side), nil
}

func MarshalTransferType(t vega.TransferType) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(t.String())))
	})
}

func MarshalTransferScope(s v2.ListTransfersRequest_Scope) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalTransferScope(v interface{}) (v2.ListTransfersRequest_Scope, error) {
	s, ok := v.(string)
	if !ok {
		return v2.ListTransfersRequest_SCOPE_UNSPECIFIED, fmt.Errorf("expected transfer scope to be a string")
	}

	t, ok := v2.ListGovernanceDataRequest_Type_value[s]
	if !ok {
		return v2.ListTransfersRequest_SCOPE_UNSPECIFIED, fmt.Errorf("failed to convert transfer scope from GraphQL to Proto: %v", s)
	}

	return v2.ListTransfersRequest_Scope(t), nil
}

func UnmarshalTransferType(v interface{}) (vega.TransferType, error) {
	s, ok := v.(string)
	if !ok {
		return vega.TransferType_TRANSFER_TYPE_UNSPECIFIED, fmt.Errorf("expected transfer type to be a string")
	}

	t, ok := vega.TransferType_value[s]
	if !ok {
		return vega.TransferType_TRANSFER_TYPE_UNSPECIFIED, fmt.Errorf("failed to convert TransferType from GraphQL to Proto: %v", s)
	}

	return vega.TransferType(t), nil
}

func MarshalTransferStatus(s eventspb.Transfer_Status) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalTransferStatus(v interface{}) (eventspb.Transfer_Status, error) {
	s, ok := v.(string)
	if !ok {
		return eventspb.Transfer_STATUS_UNSPECIFIED, fmt.Errorf("expected transfer status to be a string")
	}

	t, ok := eventspb.Transfer_Status_value[s]
	if !ok {
		return eventspb.Transfer_STATUS_UNSPECIFIED, fmt.Errorf("failed to convert TransferStatus from GraphQL to Proto: %v", s)
	}

	return eventspb.Transfer_Status(t), nil
}

func MarshalDispatchMetric(s vega.DispatchMetric) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalDispatchMetric(v interface{}) (vega.DispatchMetric, error) {
	return vega.DispatchMetric_DISPATCH_METRIC_UNSPECIFIED, ErrUnimplemented
}

func MarshalNodeStatus(s vega.NodeStatus) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalNodeStatus(v interface{}) (vega.NodeStatus, error) {
	return vega.NodeStatus_NODE_STATUS_UNSPECIFIED, ErrUnimplemented
}

func MarshalAssetStatus(s vega.Asset_Status) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalAssetStatus(v interface{}) (vega.Asset_Status, error) {
	return vega.Asset_STATUS_UNSPECIFIED, ErrUnimplemented
}

func MarshalNodeSignatureKind(s commandspb.NodeSignatureKind) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalNodeSignatureKind(v interface{}) (commandspb.NodeSignatureKind, error) {
	return commandspb.NodeSignatureKind_NODE_SIGNATURE_KIND_UNSPECIFIED, ErrUnimplemented
}

func MarshalOracleSpecStatus(s vegapb.DataSourceSpec_Status) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalOracleSpecStatus(v interface{}) (vegapb.DataSourceSpec_Status, error) {
	return vegapb.DataSourceSpec_STATUS_UNSPECIFIED, ErrUnimplemented
}

func MarshalPropertyKeyType(s datapb.PropertyKey_Type) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalPropertyKeyType(v interface{}) (datapb.PropertyKey_Type, error) {
	return datapb.PropertyKey_TYPE_UNSPECIFIED, ErrUnimplemented
}

func MarshalConditionOperator(s datapb.Condition_Operator) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalConditionOperator(v interface{}) (datapb.Condition_Operator, error) {
	return datapb.Condition_OPERATOR_UNSPECIFIED, ErrUnimplemented
}

func MarshalVoteValue(s vega.Vote_Value) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalVoteValue(v interface{}) (vega.Vote_Value, error) {
	return vega.Vote_VALUE_UNSPECIFIED, ErrUnimplemented
}

func MarshalAuctionTrigger(s vega.AuctionTrigger) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalAuctionTrigger(v interface{}) (vega.AuctionTrigger, error) {
	return vega.AuctionTrigger_AUCTION_TRIGGER_UNSPECIFIED, ErrUnimplemented
}

func MarshalStakeLinkingStatus(s eventspb.StakeLinking_Status) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalStakeLinkingStatus(v interface{}) (eventspb.StakeLinking_Status, error) {
	return eventspb.StakeLinking_STATUS_UNSPECIFIED, ErrUnimplemented
}

func MarshalStakeLinkingType(s eventspb.StakeLinking_Type) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalStakeLinkingType(v interface{}) (eventspb.StakeLinking_Type, error) {
	return eventspb.StakeLinking_TYPE_UNSPECIFIED, ErrUnimplemented
}

func MarshalWithdrawalStatus(s vega.Withdrawal_Status) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalWithdrawalStatus(v interface{}) (vega.Withdrawal_Status, error) {
	return vega.Withdrawal_STATUS_UNSPECIFIED, ErrUnimplemented
}

func MarshalDepositStatus(s vega.Deposit_Status) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalDepositStatus(v interface{}) (vega.Deposit_Status, error) {
	return vega.Deposit_STATUS_UNSPECIFIED, ErrUnimplemented
}

func MarshalOrderStatus(s vega.Order_Status) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalOrderStatus(v interface{}) (vega.Order_Status, error) {
	s, ok := v.(string)
	if !ok {
		return vega.Order_STATUS_UNSPECIFIED, fmt.Errorf("exoected order status to be a string")
	}

	t, ok := vega.Order_Status_value[s]
	if !ok {
		return vega.Order_STATUS_UNSPECIFIED, fmt.Errorf("failed to convert order status from GraphQL to Proto: %v", s)
	}

	return vega.Order_Status(t), nil
}

func MarshalOrderTimeInForce(s vega.Order_TimeInForce) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalOrderTimeInForce(v interface{}) (vega.Order_TimeInForce, error) {
	s, ok := v.(string)
	if !ok {
		return vega.Order_TIME_IN_FORCE_UNSPECIFIED, fmt.Errorf("expected order time in force to be a string")
	}

	t, ok := vega.Order_TimeInForce_value[s]
	if !ok {
		return vega.Order_TIME_IN_FORCE_UNSPECIFIED, fmt.Errorf("failed to convert TimeInForce from GraphQL to Proto: %v", s)
	}

	return vega.Order_TimeInForce(t), nil
}

func MarshalPeggedReference(s vega.PeggedReference) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalPeggedReference(v interface{}) (vega.PeggedReference, error) {
	return vega.PeggedReference_PEGGED_REFERENCE_UNSPECIFIED, ErrUnimplemented
}

func MarshalProposalRejectionReason(s vega.ProposalError) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalProposalRejectionReason(v interface{}) (vega.ProposalError, error) {
	return vega.ProposalError_PROPOSAL_ERROR_UNSPECIFIED, ErrUnimplemented
}

func MarshalOrderRejectionReason(s vega.OrderError) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalOrderRejectionReason(v interface{}) (vega.OrderError, error) {
	return vega.OrderError_ORDER_ERROR_UNSPECIFIED, ErrUnimplemented
}

func MarshalStopOrderRejectionReason(s vega.StopOrder_RejectionReason) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalStopOrderRejectionReason(v interface{}) (vega.StopOrder_RejectionReason, error) {
	return vega.StopOrder_REJECTION_REASON_UNSPECIFIED, ErrUnimplemented
}

func MarshalOrderType(s vega.Order_Type) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalOrderType(v interface{}) (vega.Order_Type, error) {
	s, ok := v.(string)
	if !ok {
		return vega.Order_TYPE_UNSPECIFIED, fmt.Errorf("expected order type to be a string")
	}

	t, ok := vega.Order_Type_value[s]
	if !ok {
		return vega.Order_TYPE_UNSPECIFIED, fmt.Errorf("failed to convert OrderType from GraphQL to Proto: %v", s)
	}

	return vega.Order_Type(t), nil
}

func MarshalMarketState(s vega.Market_State) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalMarketState(v interface{}) (vega.Market_State, error) {
	return vega.Market_STATE_UNSPECIFIED, ErrUnimplemented
}

func MarshalMarketTradingMode(s vega.Market_TradingMode) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalMarketTradingMode(v interface{}) (vega.Market_TradingMode, error) {
	return vega.Market_TRADING_MODE_UNSPECIFIED, ErrUnimplemented
}

func MarshalInterval(s vega.Interval) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalInterval(v interface{}) (vega.Interval, error) {
	s, ok := v.(string)
	if !ok {
		return vega.Interval_INTERVAL_UNSPECIFIED, fmt.Errorf("expected interval in force to be a string")
	}

	t, ok := vega.Interval_value[s]
	if !ok {
		return vega.Interval_INTERVAL_UNSPECIFIED, fmt.Errorf("failed to convert Interval from GraphQL to Proto: %v", s)
	}

	return vega.Interval(t), nil
}

func MarshalProposalType(s v2.ListGovernanceDataRequest_Type) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalProposalType(v interface{}) (v2.ListGovernanceDataRequest_Type, error) {
	s, ok := v.(string)
	if !ok {
		return v2.ListGovernanceDataRequest_TYPE_UNSPECIFIED, fmt.Errorf("expected proposal type in force to be a string")
	}

	t, ok := v2.ListGovernanceDataRequest_Type_value[s]
	if !ok {
		return v2.ListGovernanceDataRequest_TYPE_UNSPECIFIED, fmt.Errorf("failed to convert proposal type from GraphQL to Proto: %v", s)
	}

	return v2.ListGovernanceDataRequest_Type(t), nil
}

func MarshalLiquidityProvisionStatus(s vega.LiquidityProvision_Status) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalLiquidityProvisionStatus(v interface{}) (vega.LiquidityProvision_Status, error) {
	return vega.LiquidityProvision_STATUS_UNSPECIFIED, ErrUnimplemented
}

func MarshalTradeType(s vega.Trade_Type) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalTradeType(v interface{}) (vega.Trade_Type, error) {
	return vega.Trade_TYPE_UNSPECIFIED, ErrUnimplemented
}

func MarshalValidatorStatus(s vega.ValidatorNodeStatus) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalValidatorStatus(v interface{}) (vega.ValidatorNodeStatus, error) {
	return vega.ValidatorNodeStatus_VALIDATOR_NODE_STATUS_UNSPECIFIED, ErrUnimplemented
}

func MarshalProtocolUpgradeProposalStatus(s eventspb.ProtocolUpgradeProposalStatus) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalProtocolUpgradeProposalStatus(v interface{}) (eventspb.ProtocolUpgradeProposalStatus, error) {
	s, ok := v.(string)
	if !ok {
		return eventspb.ProtocolUpgradeProposalStatus_PROTOCOL_UPGRADE_PROPOSAL_STATUS_UNSPECIFIED, fmt.Errorf("expected proposal type in force to be a string")
	}

	t, ok := eventspb.ProtocolUpgradeProposalStatus_value[s] // v2.ListGovernanceDataRequest_Type_value[s]
	if !ok {
		return eventspb.ProtocolUpgradeProposalStatus_PROTOCOL_UPGRADE_PROPOSAL_STATUS_UNSPECIFIED, fmt.Errorf("failed to convert proposal type from GraphQL to Proto: %v", s)
	}

	return eventspb.ProtocolUpgradeProposalStatus(t), nil
}

func MarshalPositionStatus(s vega.PositionStatus) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalPositionStatus(v interface{}) (vega.PositionStatus, error) {
	s, ok := v.(string)
	if !ok {
		return vega.PositionStatus_POSITION_STATUS_UNSPECIFIED, fmt.Errorf("expected position status to be a string")
	}
	t, ok := vega.PositionStatus_value[s]
	if !ok {
		return vega.PositionStatus_POSITION_STATUS_UNSPECIFIED, fmt.Errorf("failed to convert position status to Proto: %v", s)
	}
	return vega.PositionStatus(t), nil
}

func MarshalStopOrderStatus(s vega.StopOrder_Status) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalStopOrderStatus(v interface{}) (vega.StopOrder_Status, error) {
	s, ok := v.(string)
	if !ok {
		return vega.StopOrder_STATUS_UNSPECIFIED, fmt.Errorf("expected stop order status to be a string")
	}
	t, ok := vega.StopOrder_Status_value[s]
	if !ok {
		return vega.StopOrder_STATUS_UNSPECIFIED, fmt.Errorf("failed to convert stop order status to Proto: %v", s)
	}
	return vega.StopOrder_Status(t), nil
}

func MarshalStopOrderExpiryStrategy(s vega.StopOrder_ExpiryStrategy) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalStopOrderExpiryStrategy(v interface{}) (vega.StopOrder_ExpiryStrategy, error) {
	s, ok := v.(string)
	if !ok {
		return vega.StopOrder_EXPIRY_STRATEGY_UNSPECIFIED, fmt.Errorf("expected stop order expiry strategy to be a string")
	}
	t, ok := vega.StopOrder_ExpiryStrategy_value[s]
	if !ok {
		return vega.StopOrder_EXPIRY_STRATEGY_UNSPECIFIED, fmt.Errorf("failed to convert stop order expiry strategy to Proto: %v", s)
	}
	return vega.StopOrder_ExpiryStrategy(t), nil
}

func MarshalStopOrderTriggerDirection(s vega.StopOrder_TriggerDirection) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalStopOrderTriggerDirection(v interface{}) (vega.StopOrder_TriggerDirection, error) {
	s, ok := v.(string)
	if !ok {
		return vega.StopOrder_TRIGGER_DIRECTION_UNSPECIFIED, fmt.Errorf("expected stop order trigger direction to be a string")
	}
	t, ok := vega.StopOrder_TriggerDirection_value[s]
	if !ok {
		return vega.StopOrder_TRIGGER_DIRECTION_UNSPECIFIED, fmt.Errorf("failed to convert stop order trigger direction to Proto: %v", s)
	}
	return vega.StopOrder_TriggerDirection(t), nil
}

func MarshalStopOrderSizeOverrideSetting(s vega.StopOrder_SizeOverrideSetting) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalStopOrderSizeOverrideSetting(v interface{}) (vega.StopOrder_SizeOverrideSetting, error) {
	s, ok := v.(string)
	if !ok {
		return vega.StopOrder_SIZE_OVERRIDE_SETTING_UNSPECIFIED, fmt.Errorf("expected stop order size override setting to be a string")
	}
	t, ok := vega.StopOrder_SizeOverrideSetting_value[s]
	if !ok {
		return vega.StopOrder_SIZE_OVERRIDE_SETTING_UNSPECIFIED, fmt.Errorf("failed to convert stop order size override setting to Proto: %v", s)
	}
	return vega.StopOrder_SizeOverrideSetting(t), nil
}

func MarshalFundingPeriodDataPointSource(s eventspb.FundingPeriodDataPoint_Source) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalFundingPeriodDataPointSource(v interface{}) (eventspb.FundingPeriodDataPoint_Source, error) {
	s, ok := v.(string)
	if !ok {
		return eventspb.FundingPeriodDataPoint_SOURCE_UNSPECIFIED, fmt.Errorf("expected funding period source to be a string")
	}
	t, ok := eventspb.FundingPeriodDataPoint_Source_value[s]
	if !ok {
		return eventspb.FundingPeriodDataPoint_SOURCE_UNSPECIFIED, fmt.Errorf("failed to convert funding period source to Proto: %v", s)
	}
	return eventspb.FundingPeriodDataPoint_Source(t), nil
}

func MarshalLiquidityFeeMethod(s vega.LiquidityFeeSettings_Method) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalLiquidityFeeMethod(v interface{}) (vega.LiquidityFeeSettings_Method, error) {
	s, ok := v.(string)
	if !ok {
		return vega.LiquidityFeeSettings_METHOD_UNSPECIFIED, fmt.Errorf("expected method state to be a string")
	}

	side, ok := vega.Proposal_State_value[s]
	if !ok {
		return vega.LiquidityFeeSettings_METHOD_UNSPECIFIED, fmt.Errorf("failed to convert method from GraphQL to Proto: %v", s)
	}

	return vega.LiquidityFeeSettings_Method(side), nil
}

func MarshalMarginMode(s vega.MarginMode) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalMarginMode(v interface{}) (vega.MarginMode, error) {
	s, ok := v.(string)
	if !ok {
		return vega.MarginMode_MARGIN_MODE_UNSPECIFIED, fmt.Errorf("expected margin mode to be a string")
	}

	side, ok := vega.MarginMode_value[s]
	if !ok {
		return vega.MarginMode_MARGIN_MODE_UNSPECIFIED, fmt.Errorf("failed to convert margin mode from GraphQL to Proto: %v", s)
	}

	return vega.MarginMode(side), nil
}
