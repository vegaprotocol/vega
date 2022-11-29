package entities

import "code.vegaprotocol.io/vega/protos/vega"

type OrderFilter struct {
	Statuses         []vega.Order_Status
	Types            []vega.Order_Type
	TimeInForces     []vega.Order_TimeInForce
	ExcludeLiquidity bool
}
