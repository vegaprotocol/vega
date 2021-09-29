package collateral

import (
	"context"
	"fmt"
	"sort"
	"strings"

	checkpoint "code.vegaprotocol.io/protos/vega/checkpoint/v1"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

type accState struct {
	accPL      types.PayloadCollateralAccounts
	assPL      types.PayloadCollateralAssets
	accs       map[string]*types.Account
	assets     map[string]types.Asset
	accIDs     []string
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

func (e *Engine) Name() types.CheckpointName {
	return types.CollateralCheckpoint
}

func (e *Engine) Checkpoint() ([]byte, error) {
	msg := &checkpoint.Collateral{
		Balances: e.getCheckpointBalances(),
	}
	ret, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (e *Engine) Load(ctx context.Context, data []byte) error {
	msg := checkpoint.Collateral{}
	if err := proto.Unmarshal(data, &msg); err != nil {
		return err
	}
	for _, balance := range msg.Balances {
		ub, _ := num.UintFromString(balance.Balance, 10)
		if balance.Party == systemOwner {
			accID := e.accountID(noMarket, systemOwner, balance.Asset, types.AccountTypeGlobalInsurance)
			if _, err := e.GetAccountByID(accID); err != nil {
				// this account is created when the asset is enabled. If we can't get this account,
				// then the asset is not yet enabled and we have a problem...
				return err
			}
			e.UpdateBalance(ctx, accID, ub)
			continue
		}
		accID := e.accountID(noMarket, balance.Party, balance.Asset, types.AccountTypeGeneral)
		if _, err := e.GetAccountByID(accID); err != nil {
			accID, _ = e.CreatePartyGeneralAccount(ctx, balance.Party, balance.Asset)
		}
		e.UpdateBalance(ctx, accID, ub)
	}
	return nil
}

// get all balances for snapshot
func (e *Engine) getCheckpointBalances() []*checkpoint.AssetBalance {
	// party -> asset -> balance
	balances := make(map[string]map[string]*num.Uint, len(e.accs))
	for _, acc := range e.accs {
		if acc.Balance.IsZero() {
			continue
		}
		switch acc.Type {
		case types.AccountTypeMargin, types.AccountTypeGeneral, types.AccountTypeBond,
			types.AccountTypeInsurance, types.AccountTypeGlobalInsurance:
			assets, ok := balances[acc.Owner]
			if !ok {
				assets = map[string]*num.Uint{}
				balances[acc.Owner] = assets
			}
			balance, ok := assets[acc.Asset]
			if !ok {
				balance = num.Zero()
				assets[acc.Asset] = balance
			}
			balance.AddSum(acc.Balance)
		case types.AccountTypeSettlement:
			if !acc.Balance.IsZero() {
				e.log.Panic("Settlement balance is not zero",
					logging.String("market-id", acc.MarketID))
			}
		}
	}

	out := make([]*checkpoint.AssetBalance, 0, len(balances))
	for owner, assets := range balances {
		for asset, balance := range assets {
			out = append(out, &checkpoint.AssetBalance{
				Party:   owner,
				Asset:   asset,
				Balance: balance.String(),
			})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		switch strings.Compare(out[i].Party, out[j].Party) {
		case -1:
			return true
		case 1:
			return false
		}
		return out[i].Asset < out[j].Asset
	})
	return out
}

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
	evts := make([]events.Event, 0, len(accs.Accounts))
	e.state.accIDs = make([]string, 0, len(accs.Accounts))
	e.state.accs = make(map[string]*types.Account, len(accs.Accounts))
	for _, acc := range accs.Accounts {
		e.accs[acc.ID] = acc
		if _, ok := e.partiesAccs[acc.Owner]; !ok {
			e.partiesAccs[acc.Owner] = map[string]*types.Account{}
		}
		e.partiesAccs[acc.Owner][acc.ID] = acc
		if acc.Type != types.AccountTypeExternal {
			e.addAccountToHashableSlice(acc)
		}
		evts = append(evts, events.NewAccountEvent(ctx, *acc))
	}
	e.state.add(accs.Accounts...)
	e.broker.SendBatch(evts)
	return nil
}

func (e *Engine) restoreAssets(ctx context.Context, assets *types.CollateralAssets) error {
	// @TODO the ID and name might not be the same, perhaps we need
	// to wrap the asset details to preserve that data
	evts := make([]events.Event, 0, len(assets.Assets))
	e.enabledAssets = make(map[string]types.Asset, len(assets.Assets))
	e.state.assetIDs = make([]string, 0, len(assets.Assets))
	e.state.assets = make(map[string]types.Asset, len(assets.Assets))
	for _, a := range assets.Assets {
		ast := types.Asset{
			ID:      a.ID,
			Details: a.Details,
		}
		e.enabledAssets[a.ID] = ast
		evts = append(evts, events.NewAssetEvent(ctx, ast))
		e.state.enableAsset(ast)
	}
	e.broker.SendBatch(evts)
	return nil
}

func (e *Engine) PrintSnapstate(name string) {
	fmt.Printf("Engine designation: %s\n", name)
	e.state.printState()
	fmt.Println("")
}

func newAccState() *accState {
	state := &accState{
		accPL: types.PayloadCollateralAccounts{
			CollateralAccounts: &types.CollateralAccounts{},
		},
		assPL: types.PayloadCollateralAssets{
			CollateralAssets: &types.CollateralAssets{},
		},
		accs:       map[string]*types.Account{},
		assets:     map[string]types.Asset{},
		accIDs:     []string{},
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

func (a accState) printState() {
	fmt.Printf("account IDs%#v\n", a.accIDs)
	for _, id := range a.accIDs {
		fmt.Printf("%s: %s\n", id, a.accs[id].Balance.String())
	}
	fmt.Println("")
	fmt.Printf("asset IDs: %#v\n", a.assetIDs)
	for _, id := range a.assetIDs {
		fmt.Printf("%s: %#v\n", id, a.assets[id])
		fmt.Printf("Details: %#v\n", *a.assets[id].Details)
	}
	fmt.Println("")
}

func (a *accState) enableAsset(asset types.Asset) {
	a.assets[asset.ID] = asset
	a.assetIDs = append(a.assetIDs, asset.ID)
	sort.Strings(a.assetIDs)
	a.updates[a.assPL.Key()] = true
}

func (a *accState) add(accs ...*types.Account) {
	if len(accs) == 0 {
		return
	}
	ids := make([]string, 0, len(accs))
	for _, acc := range accs {
		if _, ok := a.accs[acc.ID]; !ok {
			ids = append(ids, acc.ID)
		}
		a.accs[acc.ID] = acc.Clone()
	}
	if len(ids) > 0 {
		a.accIDs = append(a.accIDs, ids...)
		sort.Strings(a.accIDs)
	}
	a.updates[a.accPL.Key()] = true
}

func (a *accState) delAcc(accs ...*types.Account) {
	if len(accs) == 0 {
		return
	}
	updated := false
	for _, acc := range accs {
		if _, ok := a.accs[acc.ID]; ok {
			updated = true
			delete(a.accs, acc.ID)
			// find ID in slice, this should always be present
			i := sort.Search(len(a.accIDs), func(i int) bool {
				return a.accIDs[i] >= acc.ID
			})
			// just make sure we found a match, this should be optional
			if a.accIDs[i] == acc.ID {
				copy(a.accIDs[i:], a.accIDs[i+1:])
			}
		}
	}
	if updated {
		a.updates[a.accPL.Key()] = true
	}
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
	// build slice again
	accs := make([]*types.Account, 0, len(a.accIDs))
	for _, id := range a.accIDs {
		accs = append(accs, a.accs[id])
	}
	// set new account slice
	a.accPL.CollateralAccounts.Accounts = accs
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
