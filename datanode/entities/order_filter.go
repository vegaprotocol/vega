package entities

import (
	"code.vegaprotocol.io/vega/protos/vega"
)

type OrderFilter struct {
	Statuses         []vega.Order_Status
	Types            []vega.Order_Type
	TimeInForces     []vega.Order_TimeInForce
	Reference        *string
	DateRange        *DateRange
	ExcludeLiquidity bool
	LiveOnly         bool
	PartyIDs         []string
	MarketIDs        []string
}

type StopOrderFilter struct {
	Statuses       []StopOrderStatus
	ExpiryStrategy []StopOrderExpiryStrategy
	DateRange      *DateRange
	PartyIDs       []string
	MarketIDs      []string
}
