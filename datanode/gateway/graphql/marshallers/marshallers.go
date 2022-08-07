package marshallers

import (
	"fmt"
	"io"
	"strconv"

	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/99designs/gqlgen/graphql"
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
