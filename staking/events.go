package staking

import "code.vegaprotocol.io/vega/types/num"

type StakingEventKind uint8

const (
	StakingEventKindDeposited StakingEventKind = iota
	StakingEventKindRemoved
)

type StakingEvent struct {
	ID     string
	Kind   StakingEventKind
	TS     int64
	Party  string
	Amount *num.Uint
}
