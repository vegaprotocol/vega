package core

import "vega/msg"

type RiskEngine interface {
	Assess(*msg.Order)
}

func Assess(order *msg.Order) {
	order.RiskFactor = 20
}
