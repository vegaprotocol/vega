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

package collateral

import (
	"context"
	"sort"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"

	"code.vegaprotocol.io/vega/libs/proto"
	"github.com/pkg/errors"
)

type accState struct {
	accPL              types.PayloadCollateralAccounts
	assPL              types.PayloadCollateralAssets
	assets             map[string]types.Asset
	assetIDs           []string
	serialisedAccounts []byte
	serialisedAssets   []byte
	hashKeys           []string
	accountsKey        string
	assetsKey          string
}

var (
	ErrInvalidSnapshotNamespace = errors.New("invalid snapshot namespace")
	ErrUnknownSnapshotType      = errors.New("snapshot data type not known")
)

func (e *Engine) Namespace() types.SnapshotNamespace {
	return types.CollateralSnapshot
}

func (e *Engine) Keys() []string {
	return e.state.hashKeys
}

func (e *Engine) Stopped() bool {
	return false
}

func (e *Engine) GetState(k string) ([]byte, []types.StateProvider, error) {
	state, err := e.state.getState(k)
	return state, nil, err
}

func (e *Engine) LoadState(ctx context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if e.Namespace() != p.Data.Namespace() {
		return nil, ErrInvalidSnapshotNamespace
	}
	// see what we're reloading
	switch pl := p.Data.(type) {
	case *types.PayloadCollateralAssets:
		err := e.restoreAssets(pl.CollateralAssets, p)
		return nil, err
	case *types.PayloadCollateralAccounts:
		err := e.restoreAccounts(ctx, pl.CollateralAccounts, p)
		return nil, err
	default:
		return nil, ErrUnknownSnapshotType
	}
}

func (e *Engine) restoreAccounts(ctx context.Context, accs *types.CollateralAccounts, p *types.Payload) error {
	e.log.Debug("restoring accounts snapshot", logging.Int("n_accounts", len(accs.Accounts)))

	evts := []events.Event{}
	pevts := []events.Event{}
	e.accs = make(map[string]*types.Account, len(accs.Accounts))
	e.partiesAccs = map[string]map[string]*types.Account{}
	e.hashableAccs = make([]*types.Account, 0, len(accs.Accounts))
	assets := map[string]struct{}{}
	for _, acc := range accs.Accounts {
		e.accs[acc.ID] = acc
		assets[acc.Asset] = struct{}{}
		if _, ok := e.partiesAccs[acc.Owner]; !ok {
			e.partiesAccs[acc.Owner] = map[string]*types.Account{}
		}
		e.partiesAccs[acc.Owner][acc.ID] = acc
		e.hashableAccs = append(e.hashableAccs, acc)
		e.addAccountToHashableSlice(acc)

		evts = append(evts, events.NewAccountEvent(ctx, *acc))

		if acc.Owner != systemOwner {
			pevts = append(pevts, events.NewPartyEvent(ctx, types.Party{Id: acc.Owner}))
		}
	}
	e.state.updateAccs(e.hashableAccs)
	e.broker.SendBatch(evts)
	e.broker.SendBatch(pevts)
	var err error
	e.state.serialisedAccounts, err = proto.Marshal(p.IntoProto())
	e.getOrCreateNetTreasuryAndGlobalInsForAssets(ctx, assets)
	return err
}

func (e *Engine) getOrCreateNetTreasuryAndGlobalInsForAssets(ctx context.Context, assets map[string]struct{}) {
	// bit of migration - ensure that the network treasury and global insurance account are created for all assets
	assetStr := make([]string, 0, len(assets))
	for k := range assets {
		assetStr = append(assetStr, k)
	}
	sort.Strings(assetStr)
	for _, asset := range assetStr {
		e.GetOrCreateNetworkTreasuryAccount(ctx, asset)
		e.GetOrCreateGlobalInsuranceAccount(ctx, asset)
	}
}

func (e *Engine) restoreAssets(assets *types.CollateralAssets, p *types.Payload) error {
	// @TODO the ID and name might not be the same, perhaps we need
	// to wrap the asset details to preserve that data
	e.log.Debug("restoring assets snapshot", logging.Int("n_assets", len(assets.Assets)))
	e.enabledAssets = make(map[string]types.Asset, len(assets.Assets))
	e.state.assetIDs = make([]string, 0, len(assets.Assets))
	e.state.assets = make(map[string]types.Asset, len(assets.Assets))
	for _, a := range assets.Assets {
		ast := types.Asset{
			ID:      a.ID,
			Details: a.Details,
			Status:  a.Status,
		}
		e.enabledAssets[a.ID] = ast
		e.state.enableAsset(ast)
	}
	var err error
	e.state.serialisedAssets, err = proto.Marshal(p.IntoProto())
	return err
}

func newAccState() *accState {
	state := &accState{
		accPL: types.PayloadCollateralAccounts{
			CollateralAccounts: &types.CollateralAccounts{},
		},
		assPL: types.PayloadCollateralAssets{
			CollateralAssets: &types.CollateralAssets{},
		},
		assets:   map[string]types.Asset{},
		assetIDs: []string{},
	}
	state.accountsKey = state.accPL.Key()
	state.assetsKey = state.assPL.Key()
	state.hashKeys = []string{
		state.assetsKey,
		state.accountsKey,
	}

	return state
}

func (a *accState) enableAsset(asset types.Asset) {
	a.assets[asset.ID] = asset
	a.assetIDs = append(a.assetIDs, asset.ID)
	sort.Strings(a.assetIDs)
}

func (a *accState) updateAsset(asset types.Asset) {
	a.assets[asset.ID] = asset
}

func (a *accState) updateAccs(accs []*types.Account) {
	a.accPL.CollateralAccounts.Accounts = accs[:]
}

func (a *accState) hashAssets() ([]byte, error) {
	assets := make([]*types.Asset, 0, len(a.assetIDs))
	for _, id := range a.assetIDs {
		ast := a.assets[id]
		assets = append(assets, &ast)
	}
	a.assPL.CollateralAssets.Assets = assets
	pl := types.Payload{
		Data: &a.assPL,
	}
	data, err := proto.Marshal(pl.IntoProto())
	if err != nil {
		return nil, err
	}
	a.serialisedAssets = data
	return data, nil
}

func (a *accState) hashAccounts() ([]byte, error) {
	// the account slice is already set, sorted and all
	pl := types.Payload{
		Data: &a.accPL,
	}
	data, err := proto.Marshal(pl.IntoProto())
	if err != nil {
		return nil, err
	}
	a.serialisedAccounts = data
	return data, nil
}

func (a *accState) serialiseK(serialFunc func() ([]byte, error), dataField *[]byte) ([]byte, error) {
	data, err := serialFunc()
	if err != nil {
		return nil, err
	}
	*dataField = data
	return data, nil
}

// get the serialised form and hash of the given key.
func (a *accState) getState(k string) ([]byte, error) {
	switch k {
	case a.accountsKey:
		return a.serialiseK(a.hashAccounts, &a.serialisedAccounts)
	case a.assetsKey:
		return a.serialiseK(a.hashAssets, &a.serialisedAssets)
	default:
		return nil, types.ErrSnapshotKeyDoesNotExist
	}
}
