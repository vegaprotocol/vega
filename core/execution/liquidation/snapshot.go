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

package liquidation

import (
	"context"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/proto"
)

func (e *Engine) Namespace() types.SnapshotNamespace {
	return types.LiquidationSnapshot
}

func (e *Engine) Keys() []string {
	return []string{e.mID}
}

// GetState must be thread-safe as it may be called from multiple goroutines concurrently!
func (e *Engine) GetState(key string) ([]byte, []types.StateProvider, error) {
	if key != e.mID {
		return nil, nil, types.ErrSnapshotKeyDoesNotExist
	}
	if e.stopped {
		return nil, nil, nil
	}
	payload := e.buildPayload()

	s, err := proto.Marshal(payload.IntoProto())
	return s, nil, err
}

func (e *Engine) LoadState(ctx context.Context, pl *types.Payload) ([]types.StateProvider, error) {
	if e.Namespace() != pl.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	switch d := pl.Data.(type) {
	case *types.LiquidationNode:
		e.mID = d.MarketID
		e.pos.open = d.NetworkPos
		e.nextStep = d.NextStep
		if d.Config != nil {
			e.cfg = d.Config.DeepClone()
		} else {
			// this can probably be removed now
			e.cfg = GetLegacyStrat()
		}
		// @NOTE this should have a protocol upgrade guard around it
		if e.cfg.DisposalFraction.IsZero() {
			e.cfg.DisposalFraction = defaultStrat.DisposalSlippage
		}
	default:
		return nil, types.ErrUnknownSnapshotType
	}
	return nil, nil
}

func (e *Engine) Stopped() bool {
	return e.stopped
}

func (e *Engine) StopSnapshots() {
	e.stopped = true
}

func (e *Engine) buildPayload() *types.Payload {
	// this should not be needed
	var cfg *types.LiquidationStrategy
	if e.cfg != nil {
		cfg = e.cfg.DeepClone()
	}
	return &types.Payload{
		Data: &types.LiquidationNode{
			MarketID:   e.mID,
			NetworkPos: e.pos.open,
			NextStep:   e.nextStep,
			Config:     cfg,
		},
	}
}
