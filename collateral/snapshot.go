package collateral

import (
	"context"
	"sort"

	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

type accState struct {
	accPL      types.PayloadCollateralAccounts
	assPL      types.PayloadCollateralAssets
	assets     map[string]types.Asset
	assetIDs   []string
	hashes     map[string][]byte
	updates    map[string]bool
	serialised map[string][]byte
	hashKeys   []string
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

func (e *Engine) GetHash(k string) ([]byte, error) {
	return e.state.getHash(k)
}

func (e *Engine) GetState(k string) ([]byte, error) {
	return e.state.getState(k)
}

func (e *Engine) Snapshot() (map[string][]byte, error) {
	r := make(map[string][]byte, len(e.state.hashKeys))
	for _, k := range e.state.hashKeys {
		state, err := e.state.getState(k)
		if err != nil {
			return nil, err
		}
		r[k] = state
	}
	return r, nil
}

func (e *Engine) LoadState(ctx context.Context, p *types.Payload) error {
	if e.Namespace() != p.Data.Namespace() {
		return ErrInvalidSnapshotNamespace
	}
	// see what we're reloading
	switch pl := p.Data.(type) {
	case *types.PayloadCollateralAssets:
		return e.restoreAssets(ctx, pl.CollateralAssets)
	case *types.PayloadCollateralAccounts:
		return e.restoreAccounts(ctx, pl.CollateralAccounts)
	default:
		return ErrUnknownSnapshotType
	}
}

func (e *Engine) restoreAccounts(ctx context.Context, accs *types.CollateralAccounts) error {
	e.accs = make(map[string]*types.Account, len(accs.Accounts))
	e.partiesAccs = map[string]map[string]*types.Account{}
	e.hashableAccs = make([]*types.Account, 0, len(accs.Accounts))
	for _, acc := range accs.Accounts {
		e.accs[acc.ID] = acc
		if _, ok := e.partiesAccs[acc.Owner]; !ok {
			e.partiesAccs[acc.Owner] = map[string]*types.Account{}
		}
		e.partiesAccs[acc.Owner][acc.ID] = acc
		if acc.Type != types.AccountTypeExternal {
			e.hashableAccs = append(e.hashableAccs, acc)
			e.addAccountToHashableSlice(acc)
		}
	}
	e.state.updateAccs(e.hashableAccs)
	return nil
}

func (e *Engine) restoreAssets(ctx context.Context, assets *types.CollateralAssets) error {
	// @TODO the ID and name might not be the same, perhaps we need
	// to wrap the asset details to preserve that data
	e.enabledAssets = make(map[string]types.Asset, len(assets.Assets))
	e.state.assetIDs = make([]string, 0, len(assets.Assets))
	e.state.assets = make(map[string]types.Asset, len(assets.Assets))
	for _, a := range assets.Assets {
		ast := types.Asset{
			ID:      a.ID,
			Details: a.Details,
		}
		e.enabledAssets[a.ID] = ast
		e.state.enableAsset(ast)
	}
	return nil
}

func newAccState() *accState {
	state := &accState{
		accPL: types.PayloadCollateralAccounts{
			CollateralAccounts: &types.CollateralAccounts{},
		},
		assPL: types.PayloadCollateralAssets{
			CollateralAssets: &types.CollateralAssets{},
		},
		assets:     map[string]types.Asset{},
		assetIDs:   []string{},
		hashes:     map[string][]byte{},
		updates:    map[string]bool{},
		serialised: map[string][]byte{},
	}
	state.hashKeys = []string{
		state.accPL.Key(),
		state.assPL.Key(),
	}
	for _, k := range state.hashKeys {
		state.hashes[k] = nil
		state.updates[k] = false
		state.serialised[k] = nil
	}
	return state
}

func (a *accState) enableAsset(asset types.Asset) {
	a.assets[asset.ID] = asset
	a.assetIDs = append(a.assetIDs, asset.ID)
	sort.Strings(a.assetIDs)
	a.updates[a.assPL.Key()] = true
}

func (a *accState) updateAccs(accs []*types.Account) {
	a.updates[a.accPL.Key()] = true
	a.accPL.CollateralAccounts.Accounts = accs[:]
}

func (a *accState) hashAssets() error {
	k := a.assPL.Key()
	if !a.updates[k] {
		return nil
	}
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
		return err
	}
	a.updates[k] = false
	a.hashes[k] = crypto.Hash(data)
	a.serialised[k] = data
	return nil
}

func (a *accState) hashAccounts() error {
	k := a.accPL.Key()
	if !a.updates[k] {
		return nil
	}
	// the account slice is already set, sorted and all
	pl := types.Payload{
		Data: &a.accPL,
	}
	data, err := proto.Marshal(pl.IntoProto())
	if err != nil {
		return err
	}
	a.serialised[k] = data
	a.hashes[k] = crypto.Hash(data)
	a.updates[k] = false
	return nil
}

func (a *accState) getState(k string) ([]byte, error) {
	update, exist := a.updates[k]
	if !exist {
		return nil, ErrSnapshotKeyDoesNotExist
	}
	if !update {
		h := a.serialised[k]
		return h, nil
	}
	if k == a.assPL.Key() {
		if err := a.hashAssets(); err != nil {
			return nil, err
		}
	} else if err := a.hashAccounts(); err != nil {
		return nil, err
	}
	h := a.serialised[k]
	return h, nil
}

func (a *accState) getHash(k string) ([]byte, error) {
	update, exist := a.updates[k]
	if !exist {
		return nil, ErrSnapshotKeyDoesNotExist
	}
	// we have a pending update
	if update {
		// hash whichever one we need to update
		if k == a.assPL.Key() {
			if err := a.hashAssets(); err != nil {
				return nil, err
			}
		} else if err := a.hashAccounts(); err != nil {
			return nil, err
		}
	}
	// fetch the new hash and return
	h := a.hashes[k]
	return h, nil
}
