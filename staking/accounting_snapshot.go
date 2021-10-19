package staking

import (
	"context"

	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"

	"github.com/golang/protobuf/proto"
)

var accountsKey = (&types.PayloadStakingAccounts{}).Key()

type accountingSnapshotState struct {
	hash       []byte
	serialised []byte
	changed    bool
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
			StakingAccounts: &types.StakingAccounts{Accounts: accounts},
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

func (a *Accounting) GetState(k string) ([]byte, error) {
	data, _, err := a.getSerialisedAndHash(k)
	return data, err
}

func (a *Accounting) LoadState(_ context.Context, payload *types.Payload) error {
	if a.Namespace() != payload.Data.Namespace() {
		return types.ErrInvalidSnapshotNamespace
	}

	switch pl := payload.Data.(type) {
	case *types.PayloadStakingAccounts:
		return a.restoreStakingAccounts(pl.StakingAccounts)
	default:
		return types.ErrUnknownSnapshotType
	}
}

func (a *Accounting) restoreStakingAccounts(accounts *types.StakingAccounts) error {
	a.hashableAccounts = make([]*StakingAccount, 0, len(accounts.Accounts))
	for _, acc := range accounts.Accounts {
		stakingAcc := &StakingAccount{
			Party:   acc.Party,
			Balance: acc.Balance,
			Events:  acc.Events,
		}
		a.hashableAccounts = append(a.hashableAccounts, stakingAcc)
		a.accounts[acc.Party] = stakingAcc
	}

	a.accState.changed = true
	return nil
}
