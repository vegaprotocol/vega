package collateral

import (
	"fmt"

	"code.vegaprotocol.io/vega/internal/events"
	"code.vegaprotocol.io/vega/internal/logging"

	types "code.vegaprotocol.io/vega/proto"
)

// generate this mock so we can write tests more easilyh
//go:generate go run github.com/golang/mock/mockgen -destination mocks/mtm_transfer_mock.go -package mocks code.vegaprotocol.io/vega/internal/collateral Transfer
type MTMTransfer events.Transfer

type processF func(t *transferT) (*types.TransferResponse, error)

type collectF func(t *transferT) error

// transferT internal type, keeps account reference etc...
type transferT struct {
	events.Transfer
	t       *types.Transfer
	res     *types.TransferResponse
	margin  *types.Account
	market  *types.Account
	general *types.Account
}

func (t transferT) Asset() string {
	return t.t.Amount.Asset
}

func (t transferT) MarginBalance() uint64 {
	return uint64(t.margin.Balance)
}

func (t transferT) GeneralBalance() uint64 {
	return uint64(t.general.Balance)
}

// TransferType indicates whether this was a win or a loss
func (t transferT) TransferaType() types.TransferType {
	return t.t.Type
}

func (e *Engine) TransferCh(transfers []events.Transfer) (<-chan events.Margin, <-chan error) {
	ech := make(chan error)
	// create channel for events
	ch := make(chan events.Margin, len(transfers))
	go func() {
		// once this is done, close the channel
		defer func() {
			e.log.Debug("Closing channels")
			close(ech)
			close(ch)
		}()
		// stop immediately if there aren't any transfers, channels are closed
		if len(transfers) == 0 {
			return
		}
		// This is where we'll implement everything
		settle, insurance, err := e.getSystemAccounts()
		if err != nil {
			ech <- err
			return
		}
		reference := fmt.Sprintf("%s close", e.market) // ledger moves need to indicate that they happened because market was closed
		// this way we know if we need to check loss response
		haveLoss := (transfers[0].Transfer().Type == types.TransferType_LOSS || transfers[0].Transfer().Type == types.TransferType_MTM_LOSS)
		// tracks delta, wins & losses and determines how to distribute losses amongst wins if needed
		distr := &distributor{}
		// get a generic setup Callback function to get trader accounts etc...
		// this returns either an error or a transfer response...
		process := e.getProcessCB(distr, reference, settle, insurance)
		lossResp, winResp := buildResponses(transfers, settle, insurance)
		loss := e.lossCB(distr, lossResp, process)
		winPos, err := processLoss(ch, transfers, loss)
		if err != nil {
			ech <- err
			return
		}
		if haveLoss {
			for _, bacc := range lossResp.Balances {
				distr.lossDelta += uint64(bacc.Balance)
				// accounts have been updated when we get response (ledger movements)
				if err := e.accountStore.IncrementBalance(bacc.Account.Id, bacc.Balance); err != nil {
					e.log.Error(
						"Failed to update target account",
						logging.String("target-account", bacc.Account.Id),
						logging.Int64("balance", bacc.Balance),
						logging.Error(err),
					)
					ech <- err
				}
			}
			if distr.lossDelta != distr.expLoss {
				e.log.Warn(
					"Expected to distribute and actual balance mismatch",
					logging.Uint64("expected-balance", distr.expLoss),
					logging.Uint64("actual-balance", distr.lossDelta),
				)
			}
		}
		if len(winPos) == 0 {
			// nothing more to do
			return
		}
		winResp.Transfers = make([]*types.LedgerEntry, 0, len(winPos)*2)
		win := e.winCallback(distr, winResp, process)
		if err := processWin(ch, winPos, win); err != nil {
			ech <- err
			return
		}
	}()
	return ch, ech
}

// buildTransferRequest builds the request, and sets the required accounts based on the type of the Transfer argument
func (e *Engine) buildTransferRequest(t *transferT, settle, insurance *types.Account) (*types.TransferRequest, error) {
	// final settle, or MTM settle, makes no difference, it's win/loss still
	// get the actual trasfer value here, for convenience
	p := t.t
	accounts, err := e.accountStore.GetMarketAccountsForOwner(e.market, p.Owner)
	if err != nil {
		e.log.Error(
			"could not get accounts for market",
			logging.String("account-owner", p.Owner),
			logging.String("market", e.market),
			logging.Error(err),
		)
		return nil, err
	}
	// set all accounts onto transferT internal type
	for _, ac := range accounts {
		switch ac.Type {
		case types.AccountType_MARGIN:
			t.margin = ac
		case types.AccountType_GENERAL:
			t.general = ac
		case types.AccountType_MARKET:
			t.market = ac
		}
	}
	if p.Type == types.TransferType_LOSS || p.Type == types.TransferType_MTM_LOSS {
		req := types.TransferRequest{
			FromAccount: []*types.Account{
				t.margin,
				t.general,
				insurance,
			},
			ToAccount: []*types.Account{
				settle,
			},
			Amount:    uint64(-p.Amount.Amount) * p.Size,
			MinAmount: 0,  // default value, but keep it here explicitly
			Asset:     "", // TBC
		}
		if req.FromAccount[0] == nil || req.FromAccount[1] == nil {
			return nil, ErrTraderAccountsMissing
		}
		return &req, nil
	}
	// this should probably go into margin account, no?
	return &types.TransferRequest{
		FromAccount: []*types.Account{
			settle,
			insurance,
		},
		ToAccount: []*types.Account{
			t.general,
		},
		Amount:    uint64(p.Amount.Amount) * p.Size,
		MinAmount: 0,  // default value, but keep it here explicitly
		Asset:     "", // TBC
	}, nil
}

func (e *Engine) getProcessCB(distr *distributor, reference string, settle, insurance *types.Account) processF {
	e.cfgMu.Lock()
	createTraderAccounts := e.CreateTraderAccounts
	e.cfgMu.Unlock()
	// common tasks performed for both win and loss positions
	return func(t *transferT) (*types.TransferResponse, error) {
		p := t.t
		if createTraderAccounts {
			// ignore errors, the only error ATM is the one telling us this call was redundant
			_ = e.accountStore.CreateTraderMarketAccounts(p.Owner, e.market)
		}
		req, err := e.buildTransferRequest(t, settle, insurance)
		if err != nil {
			e.log.Error(
				"Failed to create the transfer request",
				logging.String("settlement-type", p.Type.String()),
				logging.String("trader-id", p.Owner),
				logging.Error(err),
			)
			return nil, err
		}
		distr.amountCB(req)
		req.Reference = reference
		res, err := e.getLedgerEntries(req)
		if err != nil {
			return nil, err
		}
		// we've gotten everything for the value to be put on the channel
		t.res = res
		return res, nil
	}
}

func (e *Engine) winCallback(distr *distributor, winResp *types.TransferResponse, process processF) collectF {
	return func(t *transferT) error {
		res, err := process(t)
		if err != nil {
			return err
		}
		distr.expWin += uint64(res.Balances[0].Balance)
		// there's only 1 balance account here (the ToAccount)
		if err := e.accountStore.IncrementBalance(res.Balances[0].Account.Id, res.Balances[0].Balance); err != nil {
			// this account might get accessed concurrently -> use increment
			e.log.Error(
				"Failed to increment balance of general account",
				logging.String("account-id", res.Balances[0].Account.Id),
				logging.Int64("increment", res.Balances[0].Balance),
				logging.Error(err),
			)
			return err
		}
		winResp.Transfers = append(winResp.Transfers, res.Transfers...)
		return nil
	}
}

func (e *Engine) lossCB(distr *distributor, lossResp *types.TransferResponse, process processF) collectF {
	return func(t *transferT) error {
		res, err := process(t)
		if err != nil {
			return err
		}
		p := t.t
		expAmount := uint64(-p.Amount.Amount) * p.Size
		distr.expLoss += expAmount
		// could increment distr.balanceDelta, but we're iterating over this later on anyway
		// and we might need to change this to handle multiple balances, best keep it there
		if uint64(res.Balances[0].Balance) != expAmount {
			e.log.Warn(
				"Loss trader accounts for full amount failed",
				logging.String("trader-id", p.Owner),
				logging.Uint64("expected-amount", expAmount),
				logging.Int64("actual-amount", res.Balances[0].Balance),
			)
		}
		lossResp.Transfers = append(lossResp.Transfers, res.Transfers...)
		// account balance is updated automatically
		// increment balance
		lossResp.Balances[0].Balance += res.Balances[0].Balance
		return nil
	}
}

func processLoss(ch chan<- events.Margin, positions []events.Transfer, cb collectF) ([]events.Transfer, error) {
	// collect whatever we have until we reach the DEBIT part of the positions
	for i, p := range positions {
		if p.Transfer().Type == types.TransferType_WIN || p.Transfer().Type == types.TransferType_MTM_WIN {
			return positions[i:], nil
		}
		t := &transferT{
			Transfer: p,
			t:        p.Transfer(),
		}
		if err := cb(t); err != nil {
			return nil, err
		}
		// add Margin on channel
		ch <- t
	}
	// only CREDIT positions found OR positions was empty to begin with
	return nil, nil
}

func processWin(ch chan<- events.Margin, positions []events.Transfer, cb collectF) error {
	// this is really simple -> just collect whatever was left
	for _, p := range positions {
		t := &transferT{
			Transfer: p,
			t:        p.Transfer(),
		}
		if err := cb(t); err != nil {
			return err
		}
		ch <- t
	}
	return nil
}

func buildResponses(positions []events.Transfer, settle, insurance *types.Account) (loss, win *types.TransferResponse) {
	loss = &types.TransferResponse{
		Transfers: make([]*types.LedgerEntry, 0, len(positions)), // roughly half should be loss, but create 2 ledger entries, so that's a reasonable cap to use
		Balances: []*types.TransferBalance{
			{
				Account: settle, // settle to this account
				Balance: 0,      // current balance delta -> 0
			},
		},
	}
	win = &types.TransferResponse{
		// we will alloc this slice once we've processed all loss
		// Transfers: make([]*types.LedgerEntry, 0, len(positions)),
		Balances: []*types.TransferBalance{
			{
				Account: settle,
			},
			{
				Account: insurance,
			},
		},
	}
	return
}
