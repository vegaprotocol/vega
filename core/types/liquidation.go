// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package types

import (
	"time"

	"code.vegaprotocol.io/vega/libs/num"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

type LiquidationStrategy struct {
	DisposalTimeStep    time.Duration
	DisposalFraction    num.Decimal
	FullDisposalSize    uint64
	MaxFractionConsumed num.Decimal
	DisposalSlippage    num.Decimal // this has to be a pointer for the time being, with the need to default to 0.1
}

type LiquidationNode struct {
	MarketID   string
	NetworkPos int64
	NextStep   time.Time
	Config     *LiquidationStrategy
}

func (l *LiquidationNode) isPayload() {}

func (l *LiquidationNode) plToProto() interface{} {
	return &snapshot.Payload_Liquidation{
		Liquidation: l.IntoProto(),
	}
}

func (l *LiquidationNode) Namespace() SnapshotNamespace {
	return LiquidationSnapshot
}

func (l *LiquidationNode) Key() string {
	return l.MarketID
}

func (l *LiquidationNode) IntoProto() *snapshot.Liquidation {
	var cfg *vegapb.LiquidationStrategy
	if l.Config != nil {
		cfg = l.Config.IntoProto()
	}
	var ns int64
	if !l.NextStep.IsZero() {
		ns = l.NextStep.UnixNano()
	}
	return &snapshot.Liquidation{
		MarketId:   l.MarketID,
		NetworkPos: l.NetworkPos,
		NextStep:   ns,
		Config:     cfg,
	}
}

func PayloadLiquidationNodeFromProto(p *snapshot.Payload_Liquidation) *LiquidationNode {
	node, err := LiquidationSnapshotFromProto(p.Liquidation)
	if err != nil {
		// @TODO figure out what to do with this error
		panic("invalid liquidation snapshot payload: " + err.Error())
	}
	return node
}

func LiquidationSnapshotFromProto(p *snapshot.Liquidation) (*LiquidationNode, error) {
	var s *LiquidationStrategy
	if p.Config != nil {
		st, err := LiquidationStrategyFromProto(p.Config)
		if err != nil {
			return nil, err
		}
		s = st
	}
	var ns time.Time
	if p.NextStep > 0 {
		ns = time.Unix(0, p.NextStep)
	}
	return &LiquidationNode{
		MarketID:   p.MarketId,
		NetworkPos: p.NetworkPos,
		NextStep:   ns,
		Config:     s,
	}, nil
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
	slippage, err := num.DecimalFromString(p.DisposalSlippageRange)
	if err != nil {
		slippage = num.DecimalZero()
		// return nil, err
	}
	return &LiquidationStrategy{
		DisposalTimeStep:    time.Second * time.Duration(p.DisposalTimeStep),
		DisposalFraction:    df,
		FullDisposalSize:    p.FullDisposalSize,
		MaxFractionConsumed: mfc,
		DisposalSlippage:    slippage,
	}, nil
}

func (l *LiquidationStrategy) IntoProto() *vegapb.LiquidationStrategy {
	slip := ""
	if !l.DisposalSlippage.IsZero() {
		slip = l.DisposalSlippage.String()
	}
	return &vegapb.LiquidationStrategy{
		DisposalTimeStep:      int64(l.DisposalTimeStep / time.Second),
		DisposalFraction:      l.DisposalFraction.String(),
		FullDisposalSize:      l.FullDisposalSize,
		MaxFractionConsumed:   l.MaxFractionConsumed.String(),
		DisposalSlippageRange: slip,
	}
}

func (l *LiquidationStrategy) DeepClone() *LiquidationStrategy {
	cpy := *l
	return &cpy
}

func (l *LiquidationStrategy) EQ(l2 *LiquidationStrategy) bool {
	// if the memory address is the same, then they are obviously the same
	if l == l2 {
		return true
	}
	if l2 == nil {
		return false
	}
	// this should be fine, there's no pointer fields to think about
	// but just in case we end up switching the decimal types out
	// return *l == *l2
	return l.DisposalTimeStep == l2.DisposalTimeStep && l.FullDisposalSize == l2.FullDisposalSize &&
		l.DisposalFraction.Equals(l2.DisposalFraction) && l.MaxFractionConsumed.Equals(l2.MaxFractionConsumed)
}
