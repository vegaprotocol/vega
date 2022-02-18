package marshallers

import (
	"fmt"
	"io"
	"strconv"

	"github.com/99designs/gqlgen/graphql"
)

func MarshalTimestamp(t int64) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		io.WriteString(w, strconv.Quote(strconv.FormatInt(t, 10)))
	})
}

func UnmarshalTimestamp(v interface{}) (int64, error) {
	s, ok := v.(string)
	if !ok {
		return 0, fmt.Errorf("Expected timestamp to be a string")
	}

	ts, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("Expected timestamp to be a string")
	}

	return ts, nil
}
