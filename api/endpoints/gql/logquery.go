package gql

import (
	"fmt"

	graphql "github.com/99designs/gqlgen/graphql"
	"go.uber.org/zap"
)

func (f *OrderFilter) String() string {
	if f == nil {
		return "[nil]"
	}
	return "TBD"
}

func (f *TradeFilter) String() string {
	if f == nil {
		return "[nil]"
	}
	return "TBD"
}

func logFieldsGraphQLQueryArgs(k string, v interface{}) []zap.Field {
	fields := make([]zap.Field, 0)
	switch k {
	case "where":
		if vOrderFilterPtr, ok := v.(*OrderFilter); ok {
			if vOrderFilterPtr != nil {
				fields = append(fields, zap.String(k, vOrderFilterPtr.String()))
			}
		} else if vTradeFilterPtr, ok := v.(*TradeFilter); ok {
			if vTradeFilterPtr != nil {
				fields = append(fields, zap.String(k, vTradeFilterPtr.String()))
			}
		} else {
			fields = append(fields, zap.String(k, "[unknown type]"))
		}
	case "expiration":
		fields = append(fields, zap.String(k, "[TBD]"))
	case "side":
		fields = append(fields, zap.String(k, (v.(Side)).String()))
	case "type":
		fields = append(fields, zap.String(k, (v.(OrderType)).String()))
	default:
		if vInt, ok := v.(int); ok {
			fields = append(fields, zap.Int(k, vInt))
		} else if vIntPtr, ok := v.(*int); ok {
			if vIntPtr != nil {
				fields = append(fields, zap.Int(k, *vIntPtr))
			} // else {
			//	fields = append(fields, zap.String(k, "[null pointer / not specified]"))
			// }
		} else if vStr, ok := v.(string); ok {
			fields = append(fields, zap.String(k, vStr))
		} else {
			fields = append(fields, zap.String(k, "[unknown type]"))
		}
	}
	return fields
}

func logFieldsGraphQLQuery(rc *graphql.ResolverContext) []zap.Field {
	fields := make([]zap.Field, 0)
	fields = append(fields, zap.String("op", fmt.Sprintf("%s.%s", rc.Object, rc.Field.Name)))

	for k, v := range rc.Args {
		fields = append(fields, logFieldsGraphQLQueryArgs(k, v)...)
	}

	if rc.Index != nil {
		fields = append(fields, zap.Int("index", *rc.Index))
	}

	return fields
}
