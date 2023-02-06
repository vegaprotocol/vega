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
	return eventspb.Transfer_STATUS_UNSPECIFIED, ErrUnimplemented
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

func UnmarshalPositionStatus(v interface{}) (vega.ValidatorNodeStatus, error) {
	return vega.PositionStatus_POSITION_STATUS_UNSPECIFIED, ErrUnimplemented
}
