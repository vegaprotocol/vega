package entities

import "code.vegaprotocol.io/vega/protos/vega"

type OrderFilter struct {
	Reference        *string
	DateRange        *DateRange
	Statuses         []vega.Order_Status
	Types            []vega.Order_Type
	TimeInForces     []vega.Order_TimeInForce
	PartyIDs         []string
	MarketIDs        []string
	ExcludeLiquidity bool
	LiveOnly         bool
}
