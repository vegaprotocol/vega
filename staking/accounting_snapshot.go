package staking

import (
	"context"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"

	"github.com/golang/protobuf/proto"
)

var accountsKey = (&types.PayloadStakingAccounts{}).Key()

type accountingSnapshotState struct {
	hash        []byte
	serialised  []byte
	changed     bool
	isRestoring bool
}

func (a *Accounting) serialiseStakingAccounts() ([]byte, error) {
	accounts := make([]*types.StakingAccount, 0, len(a.hashableAccounts))
	for _, acc := range a.hashableAccounts {
		accounts = append(accounts,
			&types.StakingAccount{
				Party:   acc.Party,
				Balance: acc.Balance,
				Events:  acc.Events,
			})
	}

	pl := types.Payload{
		Data: &types.PayloadStakingAccounts{
			StakingAccounts: &types.StakingAccounts{
				Accounts:                accounts,
				StakingAssetTotalSupply: a.stakingAssetTotalSupply.Clone(),
			},
		},
	}

	return proto.Marshal(pl.IntoProto())
}

// get the serialised form and hash of the given key.
func (a *Accounting) getSerialisedAndHash(k string) ([]byte, []byte, error) {
	if k != accountsKey {
		return nil, nil, types.ErrSnapshotKeyDoesNotExist
	}

	if !a.accState.changed {
		return a.accState.serialised, a.accState.hash, nil
	}

	data, err := a.serialiseStakingAccounts()
	if err != nil {
		return nil, nil, err
	}

	hash := crypto.Hash(data)
	a.accState.serialised = data
	a.accState.hash = hash
	a.accState.changed = false
	return data, hash, nil
}

func (a *Accounting) OnStateLoaded(_ context.Context) error {
	a.accState.isRestoring = false
	return nil
}

func (a *Accounting) OnStateLoadStarts(_ context.Context) error {
	a.accState.isRestoring = true
	return nil
}

func (a *Accounting) Namespace() types.SnapshotNamespace {
	return types.StakingSnapshot
}

func (a *Accounting) Keys() []string {
	return []string{accountsKey}
}

func (a *Accounting) GetHash(k string) ([]byte, error) {
	_, hash, err := a.getSerialisedAndHash(k)
	return hash, err
}

func (a *Accounting) GetState(k string) ([]byte, []types.StateProvider, error) {
	data, _, err := a.getSerialisedAndHash(k)
	return data, nil, err
}

func (a *Accounting) LoadState(ctx context.Context, payload *types.Payload) ([]types.StateProvider, error) {
	if a.Namespace() != payload.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	switch pl := payload.Data.(type) {
	case *types.PayloadStakingAccounts:
		return nil, a.restoreStakingAccounts(ctx, pl.StakingAccounts)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (a *Accounting) restoreStakingAccounts(ctx context.Context, accounts *types.StakingAccounts) error {
	a.hashableAccounts = make([]*StakingAccount, 0, len(accounts.Accounts))
	evts := []events.Event{}
	pevts := []events.Event{}
	for _, acc := range accounts.Accounts {
		stakingAcc := &StakingAccount{
			Party:   acc.Party,
			Balance: acc.Balance,
			Events:  acc.Events,
		}
		a.hashableAccounts = append(a.hashableAccounts, stakingAcc)
		a.accounts[acc.Party] = stakingAcc
		pevts = append(pevts, events.NewPartyEvent(ctx, types.Party{Id: acc.Party}))
		a.log.Debug("restoring staking account",
			logging.String("party", acc.Party),
			logging.Int("stakelinkings", len(acc.Events)),
		)
		for _, e := range acc.Events {
			evts = append(evts, events.NewStakeLinking(ctx, *e))
		}
	}

	a.stakingAssetTotalSupply = accounts.StakingAssetTotalSupply.Clone()
	a.accState.changed = true
	a.broker.SendBatch(evts)
	a.broker.SendBatch(pevts)
	return nil
}
