package types

import "code.vegaprotocol.io/vega/types/num"

type ScoreData struct {
	RawValScores      map[string]num.Decimal
	PerformanceScores map[string]num.Decimal
	MultisigScores    map[string]num.Decimal
	ValScores         map[string]num.Decimal
	NormalisedScores  map[string]num.Decimal
	NodeIDSlice       []string
}
