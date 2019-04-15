package collateral

import (
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/internal/storage"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

var (
	ErrSystemAccountsMissing = errors.New("system accounts missing for collateral engine to work")
)

type Engine struct {
	*Config
	market       string
	accountStore Accounts
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/account_store_mock.go -package mocks code.vegaprotocol.io/vega/internal/engines/collateral Accounts
type Accounts interface {
	CreateMarketAccounts(market string) error
	CreateTraderMarketAccounts(owner, market string) error
	UpdateBalance(id string, balance int64) error
	IncrementBalance(id string, inc int64) error
	GetMarketAccountsForOwner(market, owner string) ([]types.Account, error)
	GetAccountsForOwnerByType(owner string, accType types.AccountType) (*types.Account, error)
}

func New(conf *Config, market string, accounts Accounts) (*Engine, error) {
	// ensure market accounts are all good to go
	if err := accounts.CreateMarketAccounts(market); err != nil && err != storage.ErrMarketAccountsExist {
		return nil, err
	}
	return &Engine{
		Config:       conf,
		market:       market,
		accountStore: accounts,
	}, nil
}

func (e *Engine) CollectSells(positions []*types.SettlePosition) (*types.TransferResponse, error) {
	reference := fmt.Sprintf("%s close", e.market)
	sysAccounts, err := e.accountStore.GetMarketAccountsForOwner(e.market, storage.SystemOwner)
	if err != nil {
		e.log.Debugf("Failed to collect buys, system accounts missing: %+v", err)
		return nil, err
	}
	var settle, insurance *types.Account
	for _, sa := range sysAccounts {
		switch sa.Type {
		case types.AccountType_INSURANCE:
			insurance = &sa
		case types.AccountType_SETTLEMENT:
			settle = &sa
		}
	}
	if settle == nil || insurance == nil {
		return nil, ErrSystemAccountsMissing
	}
	resp := types.TransferResponse{
		Transfers: make([]*types.LedgerEntry, 0, len(positions)),
		Balances: []*types.TransferBalance{
			{
				Account: settle,
			},
			{
				Account: insurance,
			},
		},
	}
	for _, p := range positions {
		if e.CreateTraderAccounts {
			_ = e.accountStore.CreateTraderMarketAccounts(p.Owner, e.market)
		}
		// general account:
		gen, err := e.accountStore.GetAccountsForOwnerByType(p.Owner, types.AccountType_GENERAL)
		if err != nil {
			e.log.Debugf("Failed to get general account for %s: %+v", p.Owner, err)
			return nil, err
		}
		req := types.TransferRequest{
			FromAccount: []*types.Account{
				settle,
				insurance,
			},
			ToAccount: []*types.Account{
				gen,
			},
			Amount:    uint64(p.Price) * p.Size,
			MinAmount: 0,  // default value, but keep it here explicitly
			Asset:     "", // TBC
			Reference: reference,
		}
		res, err := e.getLedgerEntries(&req)
		if err != nil {
			e.log.Debugf("Failed to get ledger entries for sell positions of %s: %+v", p.Owner, err)
			return nil, err
		}
		// there's only 1 balance account here (the ToAccount)
		if err := e.accountStore.IncrementBalance(gen.Id, res.Balances[0].Balance); err != nil {
			// this account might get accessed concurrently -> use increment
			e.log.Debugf("Failed to increment the balance of account %s: %+v", gen.Id, res.Balances[0].Balance)
			return nil, err
		}
		resp.Transfers = append(resp.Transfers, res.Transfers...)
	}
	for _, b := range resp.Balances {
		b.Balance = b.Account.Balance
	}
	return &resp, nil
}

// CollectBuys - first half of settle stuff
func (e *Engine) CollectBuys(positions []*types.SettlePosition) (*types.TransferResponse, error) {
	reference := fmt.Sprintf("%s close", e.market)
	sysAccounts, err := e.accountStore.GetMarketAccountsForOwner(e.market, storage.SystemOwner)
	if err != nil {
		e.log.Debugf("Failed to collect buys, system accounts missing: %+v", err)
		return nil, err
	}
	var (
		settle, insurance *types.Account
	)
	for _, sa := range sysAccounts {
		switch sa.Type {
		case types.AccountType_INSURANCE:
			insurance = &sa
		case types.AccountType_SETTLEMENT:
			settle = &sa
		}
	}
	resp := types.TransferResponse{
		Transfers: make([]*types.LedgerEntry, 0, len(positions)), // each position will have at least 1 ledger entry
		Balances: []*types.TransferBalance{
			{
				Account: settle, // settle to this account
				Balance: 0,      // current balance delta -> 0
			},
		},
	}
	if settle == nil || insurance == nil {
		return nil, ErrSystemAccountsMissing
	}
	for _, p := range positions {
		if e.CreateTraderAccounts {
			_ = e.accountStore.CreateTraderMarketAccounts(p.Owner, e.market)
		}
		accounts, err := e.accountStore.GetMarketAccountsForOwner(e.market, p.Owner)
		if err != nil {
			e.log.Debugf("could not get accounts for %s on market %s: %+v", p.Owner, e.market, err)
			return nil, err
		}
		req := types.TransferRequest{
			FromAccount: make([]*types.Account, 3), // create indexes already
			ToAccount: []*types.Account{
				settle,
			},
			Amount:    uint64(p.Price) * p.Size,
			MinAmount: 0,  // default value, but keep it here explicitly
			Asset:     "", // TBC
			Reference: reference,
		}
		for _, ca := range accounts {
			switch ca.Type {
			case types.AccountType_MARGIN:
				req.FromAccount[0] = &ca
			case types.AccountType_MARKET:
				req.FromAccount[1] = &ca
			}
		}
		req.FromAccount[2] = insurance
		res, err := e.getLedgerEntries(&req)
		if err != nil {
			return nil, err
		}
		// append ledger moves
		resp.Transfers = append(resp.Transfers, res.Transfers...)
		// account balance is updated automatically
		// increment balance
		resp.Balances[0].Balance += res.Balances[0].Balance
	}
	if err := e.accountStore.UpdateBalance(resp.Balances[0].Account.Id, resp.Balances[0].Account.Balance); err != nil {
		e.log.Debugf("Failed to update target account %s to balance %v: %+v", settle.Id, settle.Balance, err)
		return nil, err
	}
	return &resp, nil
}

// this builds a TransferResponse for a specific request, we collect all of them and aggregate
func (e *Engine) getLedgerEntries(req *types.TransferRequest) (*types.TransferResponse, error) {
	ret := types.TransferResponse{
		Transfers: []*types.LedgerEntry{},
		Balances:  make([]*types.TransferBalance, 0, len(req.ToAccount)),
	}
	for _, t := range req.ToAccount {
		ret.Balances = append(ret.Balances, &types.TransferBalance{
			Account: t,
		})
	}
	amount := int64(req.Amount)
	for _, acc := range req.FromAccount {
		// give each to account an equal share
		parts := amount / int64(len(req.ToAccount))
		// add remaining pennies to last ledger movement
		remainder := amount % int64(len(req.ToAccount))
		var (
			to *types.TransferBalance
			lm *types.LedgerEntry
		)
		// either the account contains enough, or we're having to access insurance pool money
		if acc.Balance > amount || acc.Type == types.AccountType_INSURANCE {
			acc.Balance -= amount
			if err := e.accountStore.UpdateBalance(acc.Id, acc.Balance); err != nil {
				e.log.Debugf("Failed to update balance of account %s to %d: %+v", acc.Id, acc.Balance, err)
				return nil, err
			}
			for _, to = range ret.Balances {
				lm = &types.LedgerEntry{
					FromAccount: acc.Id,
					ToAccount:   to.Account.Id,
					Amount:      parts,
					Reference:   req.Reference,
					Type:        "settlement",
					Timestamp:   time.Now().Unix(),
				}
				ret.Transfers = append(ret.Transfers, lm)
				to.Balance += parts
				to.Account.Balance += parts
			}
			// add remainder
			if remainder > 0 {
				lm.Amount += remainder
				to.Balance += remainder
				to.Account.Balance += remainder
			}
			return &ret, nil
		}
		if acc.Balance > 0 {
			amount -= acc.Balance
			if err := e.accountStore.UpdateBalance(acc.Id, 0); err != nil {
				e.log.Debugf("Failed to update balance of account %s to 0: %+v", acc.Id, err)
				return nil, err
			}
			for _, to = range ret.Balances {
				lm = &types.LedgerEntry{
					FromAccount: acc.Id,
					ToAccount:   to.Account.Id,
					Amount:      parts,
					Reference:   req.Reference,
					Type:        "settlement",
					Timestamp:   time.Now().Unix(),
				}
				ret.Transfers = append(ret.Transfers, lm)
				to.Account.Balance += parts
				to.Balance += parts
			}
			if remainder > 0 {
				lm.Amount += remainder
				to.Account.Balance += remainder
				to.Balance += remainder
			}
			acc.Balance = 0
		}
	}
	return &ret, nil
}
