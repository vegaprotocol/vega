package types

import (
	"reflect"
	"strconv"

	"code.vegaprotocol.io/vega/types/num"
)

type Stringer interface {
	String() string
}

func reflectPointerToString(obj Stringer) string {
	if obj == nil || reflect.ValueOf(obj).Kind() == reflect.Ptr && reflect.ValueOf(obj).IsNil() {
		return "nil"
	}
	return obj.String()
}

func uintPointerToString(obj *num.Uint) string {
	if obj == nil {
		return "nil"
	}
	return obj.String()
}

func int64PointerToString(n *int64) string {
	if n == nil {
		return "nil"
	}
	return strconv.FormatInt(*n, 10)
}
