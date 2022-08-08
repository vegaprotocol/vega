package marshallers

import (
	"errors"
	"fmt"
	"io"
	"strconv"

	"code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
	oraclespb "code.vegaprotocol.io/vega/protos/vega/oracles/v1"

	"github.com/99designs/gqlgen/graphql"
)

var (
	ErrUnimplemented = errors.New("unmarshaller not implemented as this API is query only")
)

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

func MarshalOracleSpecStatus(s oraclespb.OracleSpec_Status) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalOracleSpecStatus(v interface{}) (oraclespb.OracleSpec_Status, error) {
	return oraclespb.OracleSpec_STATUS_UNSPECIFIED, ErrUnimplemented
}

func MarshalPropertyKeyType(s oraclespb.PropertyKey_Type) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalPropertyKeyType(v interface{}) (oraclespb.PropertyKey_Type, error) {
	return oraclespb.PropertyKey_TYPE_UNSPECIFIED, ErrUnimplemented
}

func MarshalConditionOperator(s oraclespb.Condition_Operator) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(strconv.Quote(s.String())))
	})
}

func UnmarshalConditionOperator(v interface{}) (oraclespb.Condition_Operator, error) {
	return oraclespb.Condition_OPERATOR_UNSPECIFIED, ErrUnimplemented
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
