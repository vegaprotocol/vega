package marshallers

import (
	"fmt"
	"io"
	"strconv"

	"github.com/99designs/gqlgen/graphql"
)

func MarshalUint32(t uint32) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		io.WriteString(w, fmt.Sprint(t))
	})
}

func UnmarshalUint32(v interface{}) (uint32, error) {
	s, ok := v.(string)
	if !ok {
		return 0, fmt.Errorf("Expected uint32 to be a string")
	}

	value64, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("Couldn't parse value into uint32")
	}

	return uint32(value64), nil
}
