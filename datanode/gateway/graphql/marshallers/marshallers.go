package marshallers

import (
	"errors"
	"fmt"
	"io"
	"strconv"

	"code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/99designs/gqlgen/graphql"
)

var (
	ErrUnimplemented = errors.New("Unmarshaller not implemented as this API is query only")
)

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
