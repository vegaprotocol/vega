// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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
				CanProposeMarket:         e.canProposeMarket,
				CanProposeAsset:          e.canProposeAsset,
				GenesisLoaded:            e.genesisLoaded,
				ProposeMarketEnabled:     e.proposeMarketEnabled,
				ProposeSpotMarketEnabled: e.proposeSpotMarketEnabled,
				ProposeAssetEnabled:      e.proposeAssetEnabled,
				ProposeMarketEnabledFrom: e.proposeMarketEnabledFrom,
				ProposeAssetEnabledFrom:  e.proposeAssetEnabledFrom,
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

	e.sendEvent(ctx)
	var err error
	e.lss.serialised, err = proto.Marshal(p.IntoProto())
	return err
}
