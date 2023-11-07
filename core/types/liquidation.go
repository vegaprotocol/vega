package types

import (
	"time"

	"code.vegaprotocol.io/vega/libs/num"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type LiquidationStrategy struct {
	DisposalTimeStep    time.Duration
	DisposalFraction    num.Decimal
	FullDisposalSize    uint64
	MaxFractionConsumed num.Decimal
}

func LiquidationStrategyFromProto(p *vegapb.LiquidationStrategy) (*LiquidationStrategy, error) {
	df, err := num.DecimalFromString(p.DisposalFraction)
	if err != nil {
		return nil, err
	}
	mfc, err := num.DecimalFromString(p.MaxFractionConsumed)
	if err != nil {
		return nil, err
	}
	return &LiquidationStrategy{
		DisposalTimeStep:    time.Second * time.Duration(p.DisposalTimeStep),
		DisposalFraction:    df,
		FullDisposalSize:    p.FullDisposalSize,
		MaxFractionConsumed: mfc,
	}, nil
}

func (l LiquidationStrategy) IntoProto() *vegapb.LiquidationStrategy {
	return &vegapb.LiquidationStrategy{
		DisposalTimeStep:    int64(l.DisposalTimeStep / time.Second),
		DisposalFraction:    l.DisposalFraction.String(),
		FullDisposalSize:    l.FullDisposalSize,
		MaxFractionConsumed: l.MaxFractionConsumed.String(),
	}
}
