package graphql

import (
	"vega/core"
)

type Resolver struct {
	matching   core.MatchingEngine
	risk       core.RiskEngine
	settlement core.SettlementEngine
}
//
//func NewResolver(vega core.Vega) *Resolver {
//	return &Resolver{
//		matching:   vega,
//		risk:       vega,
//		settlement: vega,
//	}
//}
