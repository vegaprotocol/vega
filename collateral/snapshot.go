package collateral

import (
	"context"
	"sort"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"

	"code.vegaprotocol.io/vega/libs/proto"
	"github.com/pkg/errors"
)

type accState struct {
	accPL              types.PayloadCollateralAccounts
	assPL              types.PayloadCollateralAssets
	assets             map[string]types.Asset
	assetIDs           []string
	updatesAccounts    bool
	updatesAssets      bool
	serialisedAccounts []byte
	serialisedAssets   []byte
	hashKeys           []string
	accountsKey        string
	assetsKey          string
}

var (
	ErrSnapshotKeyDoesNotExist  = errors.New("unknown key for collateral snapshot")
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

func (e *Engine) HasChanged(k string) bool {
	return e.state.HasChanged(k)
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
		err := e.restoreAssets(ctx, pl.CollateralAssets, p)
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
	for _, acc := range accs.Accounts {
		e.accs[acc.ID] = acc
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
	e.state.updatesAccounts = false
	e.state.serialisedAccounts, err = proto.Marshal(p.IntoProto())

	return err
}

func (e *Engine) restoreAssets(ctx context.Context, assets *types.CollateralAssets, p *types.Payload) error {
	// @TODO the ID and name might not be the same, perhaps we need
	// to wrap the asset details to preserve that data
	e.log.Debug("restoring assets snapshot", logging.Int("n_assets", len(assets.Assets)))
	e.enabledAssets = make(map[string]types.Asset, len(assets.Assets))
	e.state.assetIDs = make([]string, 0, len(assets.Assets))
	e.state.assets = make(map[string]types.Asset, len(assets.Assets))
	evts := []events.Event{}
	for _, a := range assets.Assets {
		ast := types.Asset{
			ID:      a.ID,
			Details: a.Details,
		}
		e.enabledAssets[a.ID] = ast
		e.state.enableAsset(ast)
		evts = append(evts, events.NewAssetEvent(ctx, *a))
	}
	e.broker.SendBatch(evts)
	var err error
	e.state.updatesAssets = false
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
		assets:          map[string]types.Asset{},
		assetIDs:        []string{},
		updatesAccounts: true,
		updatesAssets:   true,
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
	a.updatesAssets = true
}

func (a *accState) updateAccs(accs []*types.Account) {
	a.updatesAccounts = true
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
	a.updatesAssets = false
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
	a.updatesAccounts = false
	return data, nil
}

func (a *accState) HasChanged(k string) bool {
	switch k {
	case a.accountsKey:
		return a.updatesAccounts
	case a.assetsKey:
		return a.updatesAssets
	default:
		return false
	}
}

func (a *accState) serialiseK(k string, serialFunc func() ([]byte, error), dataField *[]byte, changedField *bool) ([]byte, error) {
	if !a.HasChanged(k) {
		if dataField == nil {
			return nil, nil
		}
		return *dataField, nil
	}
	data, err := serialFunc()
	if err != nil {
		return nil, err
	}
	*dataField = data
	*changedField = false
	return data, nil
}

// get the serialised form and hash of the given key.
func (a *accState) getState(k string) ([]byte, error) {
	switch k {
	case a.accountsKey:
		return a.serialiseK(k, a.hashAccounts, &a.serialisedAccounts, &a.updatesAccounts)
	case a.assetsKey:
		return a.serialiseK(k, a.hashAssets, &a.serialisedAssets, &a.updatesAssets)
	default:
		return nil, types.ErrSnapshotKeyDoesNotExist
	}
}
