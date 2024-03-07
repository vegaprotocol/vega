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

package limits

import (
	"context"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/proto"
)

var (
	allKey = (&types.PayloadLimitState{}).Key()

	hashKeys = []string{
		allKey,
	}
)

type limitsSnapshotState struct {
	serialised []byte
}

// serialiseLimits returns the engine's limit data as marshalled bytes.
func (e *Engine) serialiseLimits() ([]byte, error) {
	pl := types.Payload{
		Data: &types.PayloadLimitState{
			LimitState: &types.LimitState{
				CanProposeMarket:          e.canProposeMarket,
				CanProposeAsset:           e.canProposeAsset,
				GenesisLoaded:             e.genesisLoaded,
				ProposeMarketEnabled:      e.proposeMarketEnabled,
				ProposeSpotMarketEnabled:  e.proposeSpotMarketEnabled,
				ProposePerpsMarketEnabled: e.proposePerpsMarketEnabled,
				ProposeAssetEnabled:       e.proposeAssetEnabled,
				ProposeMarketEnabledFrom:  e.proposeMarketEnabledFrom,
				ProposeAssetEnabledFrom:   e.proposeAssetEnabledFrom,
				CanUseAMMEnabled:          e.useAMMEnabled,
			},
		},
	}
	return proto.Marshal(pl.IntoProto())
}

// get the serialised form of the given key.
func (e *Engine) serialise(k string) ([]byte, error) {
	if k != allKey {
		return nil, types.ErrSnapshotKeyDoesNotExist
	}

	data, err := e.serialiseLimits()
	if err != nil {
		return nil, err
	}

	e.lss.serialised = data
	return data, nil
}

func (e *Engine) Namespace() types.SnapshotNamespace {
	return types.LimitSnapshot
}

func (e *Engine) Keys() []string {
	return hashKeys
}

func (e *Engine) Stopped() bool {
	return false
}

func (e *Engine) GetState(k string) ([]byte, []types.StateProvider, error) {
	data, err := e.serialise(k)
	return data, nil, err
}

func (e *Engine) LoadState(ctx context.Context, payload *types.Payload) ([]types.StateProvider, error) {
	if e.Namespace() != payload.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	switch pl := payload.Data.(type) {
	case *types.PayloadLimitState:
		return nil, e.restoreLimits(ctx, pl.LimitState, payload)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (e *Engine) restoreLimits(ctx context.Context, l *types.LimitState, p *types.Payload) error {
	e.canProposeAsset = l.CanProposeAsset
	e.canProposeMarket = l.CanProposeMarket
	e.genesisLoaded = l.GenesisLoaded
	e.proposeMarketEnabled = l.ProposeMarketEnabled
	e.proposeAssetEnabled = l.ProposeAssetEnabled
	e.proposeMarketEnabledFrom = l.ProposeMarketEnabledFrom
	e.proposeAssetEnabledFrom = l.ProposeAssetEnabledFrom
	e.proposeSpotMarketEnabled = l.ProposeSpotMarketEnabled
	e.proposePerpsMarketEnabled = l.ProposePerpsMarketEnabled
	e.useAMMEnabled = l.CanUseAMMEnabled

	e.sendEvent(ctx)
	var err error
	e.lss.serialised, err = proto.Marshal(p.IntoProto())
	return err
}
