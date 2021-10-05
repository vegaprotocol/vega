package banking

import (
	"context"
	"errors"
	"math/big"
	"sort"
	"strings"

	"code.vegaprotocol.io/vega/assets/common"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"
	"github.com/golang/protobuf/proto"
)

var (
	withdrawalsKey = (&types.PayloadBankingWithdrawals{}).Key()
	depositsKey    = (&types.PayloadBankingDeposits{}).Key()
	seenKey        = (&types.PayloadBankingSeen{}).Key()

	hashKeys = []string{
		withdrawalsKey,
		depositsKey,
		seenKey,
	}

	ErrSnapshotKeyDoesNotExist = errors.New("unknown key for banking snapshot")
)

type bankingSnapshotState struct {
	changed    map[string]bool
	hash       map[string][]byte
	serialised map[string][]byte
}

func (e *Engine) Namespace() types.SnapshotNamespace {
	return types.BankingSnapshot
}

func (e *Engine) Keys() []string {
	return hashKeys
}

func (e *Engine) serialiseWithdrawals() ([]byte, error) {
	withdrawals := make([]*types.RWithdrawal, 0, len(e.withdrawals))
	for _, v := range e.withdrawals {
		withdrawals = append(withdrawals, &types.RWithdrawal{Ref: v.ref.String(), Withdrawal: v.w})
	}

	sort.SliceStable(withdrawals, func(i, j int) bool { return withdrawals[i].Ref < withdrawals[j].Ref })
	for _, w := range withdrawals {
		println(w.Ref)
	}

	payload := types.Payload{
		Data: &types.PayloadBankingWithdrawals{
			BankingWithdrawals: &types.BankingWithdrawals{
				Withdrawals: withdrawals,
			},
		},
	}
	return proto.Marshal(payload.IntoProto())
}

func (e *Engine) serialiseSeen() ([]byte, error) {
	seen := make([]*types.TxRef, len(e.seen))
	for v := range e.seen {
		seen = append(seen, &types.TxRef{Asset: string(v.asset), BlockNr: v.blockNumber, Hash: v.hash, LogIndex: v.logIndex})
	}

	sort.SliceStable(seen, func(i, j int) bool {
		switch strings.Compare(seen[i].Asset, seen[j].Asset) {
		case -1:
			return true
		case 1:
			return false
		}

		switch strings.Compare(seen[i].Hash, seen[j].Hash) {
		case -1:
			return true
		case 1:
			return false
		}

		if seen[i].LogIndex == seen[j].LogIndex {
			return seen[i].BlockNr < seen[j].BlockNr
		}

		return seen[i].LogIndex < seen[j].LogIndex
	})

	payload := types.Payload{
		Data: &types.PayloadBankingSeen{
			BankingSeen: &types.BankingSeen{
				Refs: seen,
			},
		},
	}

	return proto.Marshal(payload.IntoProto())
}

func (e *Engine) serialiseDeposits() ([]byte, error) {
	deposits := make([]*types.BDeposit, 0, len(e.deposits))
	for _, v := range e.deposits {
		deposits = append(deposits, &types.BDeposit{ID: v.ID, Deposit: v})
	}

	sort.SliceStable(deposits, func(i, j int) bool { return deposits[i].ID < deposits[j].ID })
	payload := types.Payload{
		Data: &types.PayloadBankingDeposits{
			BankingDeposits: &types.BankingDeposits{
				Deposit: deposits,
			},
		},
	}

	return proto.Marshal(payload.IntoProto())
}

// get the serialised form and hash of the given key
func (e *Engine) getSerialisedAndHash(k string) ([]byte, []byte, error) {
	if _, ok := e.keyToSerialiser[k]; !ok {
		return nil, nil, ErrSnapshotKeyDoesNotExist
	}

	if !e.bss.changed[k] {
		return e.bss.serialised[k], e.bss.hash[k], nil
	}

	data, err := e.keyToSerialiser[k]()
	if err != nil {
		return nil, nil, err
	}

	hash := crypto.Hash(data)
	e.bss.serialised[k] = data
	e.bss.hash[k] = hash
	e.bss.changed[k] = false
	return data, hash, nil
}

func (e *Engine) GetHash(k string) ([]byte, error) {
	_, hash, err := e.getSerialisedAndHash(k)
	return hash, err
}

func (e *Engine) GetState(k string) ([]byte, error) {
	state, _, err := e.getSerialisedAndHash(k)
	return state, err
}

func (e *Engine) Snapshot() (map[string][]byte, error) {
	r := make(map[string][]byte, len(hashKeys))
	for _, k := range hashKeys {
		state, err := e.GetState(k)
		if err != nil {
			return nil, err
		}
		r[k] = state
	}
	return r, nil
}

func (e *Engine) LoadState(ctx context.Context, p *types.Payload) error {
	if e.Namespace() != p.Data.Namespace() {
		return types.ErrInvalidSnapshotNamespace
	}
	// see what we're reloading
	switch pl := p.Data.(type) {
	case *types.PayloadBankingDeposits:
		return e.restoreDeposits(ctx, pl.BankingDeposits)
	case *types.PayloadBankingWithdrawals:
		return e.restoreWithdrawals(ctx, pl.BankingWithdrawals)
	case *types.PayloadBankingSeen:
		return e.restoreSeen(ctx, pl.BankingSeen)
	default:
		return types.ErrUnknownSnapshotType
	}
}

func (e *Engine) restoreDeposits(ctx context.Context, deposits *types.BankingDeposits) error {
	for _, d := range deposits.Deposit {
		e.deposits[d.ID] = d.Deposit
	}

	e.bss.changed[depositsKey] = true
	return nil
}

func (e *Engine) restoreWithdrawals(ctx context.Context, withdrawals *types.BankingWithdrawals) error {
	for _, w := range withdrawals.Withdrawals {
		ref := new(big.Int)
		ref.SetString(w.Ref, 10)
		e.withdrawalCnt.Add(e.withdrawalCnt, big.NewInt(1))
		e.withdrawals[w.Withdrawal.ID] = withdrawalRef{
			w:   w.Withdrawal,
			ref: ref,
		}
	}

	e.bss.changed[withdrawalsKey] = true
	return nil
}

func (e *Engine) restoreSeen(ctx context.Context, seen *types.BankingSeen) error {
	for _, s := range seen.Refs {
		e.seen[txRef{
			asset:       common.AssetClass(s.Asset),
			blockNumber: s.BlockNr,
			hash:        s.Hash,
			logIndex:    s.LogIndex,
		}] = struct{}{}
	}
	e.bss.changed[seenKey] = true
	return nil
}
