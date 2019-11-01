package collateral

import (
	"fmt"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

const (
	initialAccountSize = 4096
	// use weird character here, maybe non-displayable ones in the future
	// if needed
	systemOwner = "*"
	noMarket    = "!"
)

var (
	// ErrSystemAccountsMissing signals that a system account is missing, which may means that the
	// collateral engine have not been initialized properly
	ErrSystemAccountsMissing = errors.New("system accounts missing for collateral engine to work")
	// ErrTraderAccountsMissing signals that the accounts for this trader do not exists
	ErrTraderAccountsMissing = errors.New("trader accounts missing, cannot collect")
	// ErrBalanceNotSet signals that the collateral failed to update an account balance
	ErrBalanceNotSet = errors.New("failed to update account balance")
	// ErrAccountDoNotExists signals that an account par of a transfer do not exists
	ErrAccountDoNotExists = errors.New("account do not exists")
	// ErrAccountAlreadyExists signals that the collateral has been instructed to created
	// an account which already exists
	ErrAccountAlreadyExists = errors.New("account already exists")
	// ErrInsufficientTraderBalance signals that the collateral have been instructed to transfer
	// funds to a trader margin account but the trader general account do not
	// have sufficient balance
	ErrInsufficientTraderBalance = errors.New("trader has insufficient balance for margin")
	// ErrInvalidTransferTypeForOp signals that the transfer type is not valid for the current operation
	ErrInvalidTransferTypeForOp = errors.New("invalid transfer type for operation")
)

// AccountBuffer ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/account_buffer_mock.go -package mocks code.vegaprotocol.io/vega/collateral AccountBuffer
type AccountBuffer interface {
	Add(types.Account)
}

type collectCB func(p *types.Transfer) error
type setupF func(*types.Transfer) (*types.TransferResponse, error)

// Engine is handling the power of the collateral
type Engine struct {
	Config
	log   *logging.Logger
	cfgMu sync.Mutex

	// map of trader ID's to map of account types + account ID's
	// traderAccounts map[string]map[types.AccountType]map[string]string // by trader, type, and asset
	// marketAccounts map[types.AccountType]map[string]string            // by type and asset

	accs map[string]*types.Account
	buf  AccountBuffer
	// cool be a unix.Time but storing it like this allow us to now time.UnixNano() all the time
	currentTime int64

	idbuf []byte
}

func (e *Engine) accountID(marketID, partyID, asset string, ty types.AccountType) string {
	if len(marketID) <= 0 {
		marketID = noMarket
	}

	// market account
	if len(partyID) <= 0 {
		partyID = systemOwner
	}

	copy(e.idbuf, marketID)
	ln := len(marketID)
	copy(e.idbuf[ln:], partyID)
	ln += len(partyID)
	copy(e.idbuf[ln:], asset)
	ln += len(asset)
	e.idbuf[ln] = byte(ty + 48)
	return string(e.idbuf[:ln+1])
}

// New instanciate a new collateral engine
func New(log *logging.Logger, conf Config, buf AccountBuffer, now time.Time) (*Engine, error) {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(conf.Level.Get())
	return &Engine{
		log:         log,
		Config:      conf,
		accs:        make(map[string]*types.Account, initialAccountSize),
		buf:         buf,
		currentTime: now.UnixNano(),
		idbuf:       make([]byte, 256),
	}, nil
}

// OnChainTimeUpdate is used to be specified as a callback in over services
// in order to be called when the chain time is updated (basically EndBlock)
func (e *Engine) OnChainTimeUpdate(t time.Time) {
	e.currentTime = t.UnixNano()
}

// ReloadConf upadte the internal configuration of the collateral engine
func (e *Engine) ReloadConf(cfg Config) {
	e.log.Info("reloading configuration")
	if e.log.GetLevel() != cfg.Level.Get() {
		e.log.Info("updating log level",
			logging.String("old", e.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		e.log.SetLevel(cfg.Level.Get())
	}

	e.cfgMu.Lock()
	e.Config = cfg
	e.cfgMu.Unlock()
}

func (e *Engine) getSystemAccounts(marketID, asset string) (settle, insurance *types.Account, err error) {

	insID := e.accountID(marketID, "", asset, types.AccountType_INSURANCE)
	setID := e.accountID(marketID, "", asset, types.AccountType_SETTLEMENT)

	var ok bool
	insurance, ok = e.accs[insID]
	if !ok {
		if e.log.GetLevel() == logging.DebugLevel {
			e.log.Debug("missing system account",
				logging.String("asset", asset),
				logging.String("id", insID),
				logging.String("market", marketID),
			)
		}
		err = ErrSystemAccountsMissing
		return
	}

	settle, ok = e.accs[setID]
	if !ok {
		if e.log.GetLevel() == logging.DebugLevel {
			e.log.Debug("missing system account",
				logging.String("asset", asset),
				logging.String("id", setID),
				logging.String("market", marketID),
			)
		}
		err = ErrSystemAccountsMissing
		return
	}

	return
}

// AddTraderToMarket - when a new trader enters a market, ensure general + margin accounts both exist
func (e *Engine) AddTraderToMarket(marketID, traderID, asset string) error {
	// accountID(marketID, traderID, asset string, ty types.AccountType) accountIDT
	genID := e.accountID("", traderID, asset, types.AccountType_GENERAL)
	marginID := e.accountID(marketID, traderID, asset, types.AccountType_MARGIN)
	_, err := e.GetAccountByID(genID)
	if err != nil {
		e.log.Error(
			"Trader doesn't have a general account somehow?",
			logging.String("trader-id", traderID))
		return ErrTraderAccountsMissing
	}
	_, err = e.GetAccountByID(marginID)
	if err != nil {
		e.log.Error(
			"Trader doesn't have a margin account somehow?",
			logging.String("trader-id", traderID),
			logging.String("Market", marketID))
		return ErrTraderAccountsMissing
	}

	return nil
}

// MarkToMarket will run the mark to market settlement over a given set of positions
func (e *Engine) MarkToMarket(marketID string, positions []events.Transfer) ([]*types.TransferResponse, error) {
	// for now, this is the same as collect, but once we finish the closing positions bit in positions/settlement
	// we'll first handle the close settlement, then the updated positions for mark-to-market
	transfers := make([]*types.Transfer, 0, len(positions))
	for _, p := range positions {
		transfers = append(transfers, p.Transfer())
	}
	return e.Transfer(marketID, transfers)
}

// Transfer will process the list of transfer instructed by other engines
func (e *Engine) Transfer(marketID string, transfers []*types.Transfer) ([]*types.TransferResponse, error) {
	if len(transfers) == 0 {
		return nil, nil
	}
	if isSettle(transfers[0]) {
		return e.collect(marketID, transfers)
	}
	// this is a balance top-up or some other thing we haven't implemented yet
	return nil, nil
}

func isSettle(transfer *types.Transfer) bool {
	switch transfer.Type {
	case types.TransferType_WIN, types.TransferType_LOSS, types.TransferType_MTM_WIN, types.TransferType_MTM_LOSS:
		return true
	}
	return false
}

// GetPartyMargin will return the current margin for a given party
func (e *Engine) GetPartyMargin(pos events.MarketPosition, asset, marketID string) (events.Margin, error) {
	genID := e.accountID("", pos.Party(), asset, types.AccountType_GENERAL)
	marginID := e.accountID(marketID, pos.Party(), asset, types.AccountType_MARGIN)
	genAcc, err := e.GetAccountByID(genID)
	if err != nil {
		e.log.Error(
			"Party doesn't have a general account somehow?",
			logging.String("party-id", pos.Party()))
		return nil, ErrTraderAccountsMissing
	}
	marAcc, err := e.GetAccountByID(marginID)
	if err != nil {
		e.log.Error(
			"Party doesn't have a margin account somehow?",
			logging.String("party-id", pos.Party()),
			logging.String("marke-id", marketID))
		return nil, ErrTraderAccountsMissing
	}

	return newOrderMarginUpdate{
		pos,
		marAcc,
		genAcc,
		asset,
		marketID,
	}, nil
}

// MarginUpdate will run the margin updates over a set of risk events (margin updates)
func (e *Engine) MarginUpdate(marketID string, updates []events.Risk,
) ([]*types.TransferResponse, []events.Margin, error) {
	response := make([]*types.TransferResponse, 0, len(updates))
	closed := make([]events.Margin, 0, len(updates)/2) // half the cap, if we have more than that, the slice will double once, and will fit all updates anyway
	// create "fake" settle account for market ID
	settle := &types.Account{
		MarketID: marketID,
	}
	for _, update := range updates {
		transfer := update.Transfer()
		req, err := e.getTransferRequest(transfer, settle, nil)
		if err != nil {
			// log this
			return nil, nil, err
		}
		res, err := e.getLedgerEntries(req)
		if err != nil {
			return nil, nil, err
		}
		// we didn't manage to top up to even the minimum required system margen, close out trader
		// we need to be careful with this, only apply this to transfer for low margin
		// the MinAmount in the transfer is always set to 0 but in 2 case:
		// - first when a new order is created, the MinAmount is the same than Amount, which is
		//   what's required to reach the InitialMargin level
		// - second when a trader margin is under the MaintenanceLevel, the MinAmount is supposed
		//   to be at least to get back to the search level, and the amount will be enough to reach
		//   InitialMargin
		// In both case either the order will not be accepted, or the trader will be closed
		if transfer.Type == types.TransferType_MARGIN_LOW &&
			res.Balances[0].Account.Balance < (int64(update.MarginBalance())+transfer.Amount.MinAmount) {
			closed = append(closed, update) // update interface embeds events.MarketPosition
		} else {
			response = append(response, res)
			for _, v := range res.GetTransfers() {
				// increment the to account
				if err := e.IncrementBalance(v.ToAccount, v.Amount); err != nil {
					e.log.Error(
						"Failed to increment balance for account",
						logging.String("account-id", v.ToAccount),
						logging.Int64("amount", v.Amount),
						logging.Error(err),
					)
					continue
				}
			}

		}
	}

	return response, closed, nil
}

// collect, handles collects for both market close as mark-to-market stuff
func (e *Engine) collect(marketID string, positions []*types.Transfer) ([]*types.TransferResponse, error) {
	if len(positions) == 0 {
		return nil, nil
	}

	// FIXME(): get asset properly
	asset := positions[0].Amount.Asset

	reference := fmt.Sprintf("%s close", marketID) // ledger moves need to indicate that they happened because market was closed
	settle, insurance, err := e.getSystemAccounts(marketID, asset)
	if err != nil {
		return nil, err
	}
	// this way we know if we need to check loss response
	haveLoss := (positions[0].Type == types.TransferType_LOSS || positions[0].Type == types.TransferType_MTM_LOSS)
	// tracks delta, wins & losses and determines how to distribute losses amongst wins if needed
	distr := distributor{}
	lossResp, winResp := getTransferResponses(positions, settle, insurance)
	// get the callbacks used to process positions
	lossCB, winCB := e.getCallbacks(&distr, reference, settle, insurance, lossResp, winResp)
	// begin work, start by processing the loss positions, and get win positions while we're at it
	winPos, err := collectLoss(positions, lossCB)
	if err != nil {
		return nil, err
	}
	// process lossResp before moving on to win...
	if haveLoss {
		for _, bacc := range lossResp.Balances {
			distr.lossDelta += uint64(bacc.Balance)
			if err := e.IncrementBalance(bacc.Account.Id, bacc.Balance); err != nil {
				e.log.Error(
					"Failed to update target account",
					logging.String("target-account", bacc.Account.Id),
					logging.Int64("balance", bacc.Balance),
					logging.Error(err),
				)
				return nil, err
			}
		}
		if distr.lossDelta != distr.expLoss {
			e.log.Debug(
				"collect: Expected to distribute and actual balance mismatch",
				logging.Uint64("expected-balance", distr.expLoss),
				logging.Uint64("actual-balance", distr.lossDelta),
			)
		}
	}
	if len(winPos) == 0 {
		return []*types.TransferResponse{
			lossResp,
		}, nil
	}
	// each position, multiplied by 2 (move from account, to account == 2 moves)
	winResp.Transfers = make([]*types.LedgerEntry, 0, len(winPos)*2)
	if err := collectWin(winPos, winCB); err != nil {
		return nil, err
	}
	// possibly verify balances?
	for _, b := range winResp.Balances {
		b.Balance = b.Account.Balance

		// save the balance now
		if err := e.UpdateBalance(b.Account.Id, b.Balance); err != nil {
			e.log.Error(
				"Failed to update target account",
				logging.String("target-account", b.Account.Id),
				logging.Int64("balance", b.Balance),
				logging.Error(err),
			)
			return nil, err
		}
	}

	if haveLoss {
		return []*types.TransferResponse{
			lossResp,
			winResp,
		}, nil
	}
	return []*types.TransferResponse{
		winResp,
	}, nil
}

func getTransferResponses(positions []*types.Transfer, settle, insurance *types.Account) (loss, win *types.TransferResponse) {
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

func (e *Engine) getCallbacks(distr *distributor, reference string, settle, insurance *types.Account, lossResp, winResp *types.TransferResponse) (collectCB, collectCB) {
	// this callback is internal only
	setupCB := e.getSetupCB(distr, reference, settle, insurance)
	lossCB := e.getLossCB(distr, lossResp, setupCB)
	winCB := e.getWinCB(distr, winResp, setupCB)
	return lossCB, winCB
}

func (e *Engine) getSetupCB(distr *distributor, reference string, settle, insurance *types.Account) setupF {
	// common tasks performed for both win and loss positions
	return func(p *types.Transfer) (*types.TransferResponse, error) {
		req, err := e.getTransferRequest(p, settle, insurance)
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
		return res, nil
	}
}

func (e *Engine) getLossCB(distr *distributor, lossResp *types.TransferResponse, setupCB setupF) collectCB {
	return func(p *types.Transfer) error {
		res, err := setupCB(p)
		if err != nil {
			return err
		}
		expAmount := uint64(-p.Amount.Amount) * p.Size
		distr.expLoss += expAmount
		// could increment distr.balanceDelta, but we're iterating over this later on anyway
		// and we might need to change this to handle multiple balances, best keep it there
		if uint64(res.Balances[0].Balance) != expAmount {
			e.log.Debug(
				"getLossCB: Loss trader accounts for full amount failed",
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

func (e *Engine) getWinCB(distr *distributor, winResp *types.TransferResponse, setupCB setupF) collectCB {
	return func(p *types.Transfer) error {
		res, err := setupCB(p)
		if err != nil {
			return err
		}
		distr.expWin += uint64(res.Balances[0].Balance)
		// there's only 1 balance account here (the ToAccount)
		// if err := e.IncrementBalance(res.Balances[0].Account.Id, res.Balances[0].Balance); err != nil {
		// 	// this account might get accessed concurrently -> use increment
		// 	e.log.Error(
		// 		"Failed to increment balance of general account",
		// 		logging.String("account-id", res.Balances[0].Account.Id),
		// 		logging.Int64("increment", res.Balances[0].Balance),
		// 		logging.Error(err),
		// 	)
		// 	return err
		// }
		winResp.Transfers = append(winResp.Transfers, res.Transfers...)
		return nil
	}
}

func collectLoss(positions []*types.Transfer, cb collectCB) ([]*types.Transfer, error) {
	// collect whatever we have until we reach the DEBIT part of the positions
	for i, p := range positions {
		if p.Type == types.TransferType_WIN || p.Type == types.TransferType_MTM_WIN {
			return positions[i:], nil
		}
		if err := cb(p); err != nil {
			return nil, err
		}
	}
	// only CREDIT positions found OR positions was empty to begin with
	return nil, nil
}

func collectWin(positions []*types.Transfer, cb collectCB) error {
	// this is really simple -> just collect whatever was left
	for _, p := range positions {
		if err := cb(p); err != nil {
			return err
		}
	}
	return nil
}

// getTransferRequest builds the request, and sets the required accounts based on the type of the Transfer argument
func (e *Engine) getTransferRequest(p *types.Transfer, settle, insurance *types.Account) (*types.TransferRequest, error) {
	asset := p.Amount.Asset

	// we'll need this account for all transfer types anyway (settlements, margin-risk updates)
	marginAcc, err := e.GetAccountByID(e.accountID(settle.MarketID, p.Owner, asset, types.AccountType_MARGIN))
	if err != nil {
		e.log.Error(
			"Failed to get the margin account",
			logging.String("owner", p.Owner),
			logging.String("market", settle.MarketID),
			logging.Error(err))
		return nil, err
	}
	// final settle, or MTM settle, makes no difference, it's win/loss still
	if p.Type == types.TransferType_LOSS || p.Type == types.TransferType_MTM_LOSS {
		req := types.TransferRequest{
			FromAccount: []*types.Account{
				marginAcc, insurance}, // we'll need 2 accounts, last one is insurance pool
			ToAccount: []*types.Account{
				settle,
			},
			Amount:    uint64(-p.Amount.Amount) * p.Size,
			MinAmount: 0,     // default value, but keep it here explicitly
			Asset:     asset, // TBC
			Reference: p.Type.String(),
		}
		return &req, nil
	}
	if p.Type == types.TransferType_WIN || p.Type == types.TransferType_MTM_WIN {
		return &types.TransferRequest{
			FromAccount: []*types.Account{
				settle,
				insurance,
			},
			ToAccount: []*types.Account{
				marginAcc,
			},
			Amount:    uint64(p.Amount.Amount) * p.Size,
			MinAmount: 0,     // default value, but keep it here explicitly
			Asset:     asset, // TBC
			Reference: p.Type.String(),
		}, nil
	}

	// now the margin/risk updates, we need to get the general account
	genAcc, err := e.GetAccountByID(
		e.accountID("", p.Owner, asset, types.AccountType_GENERAL),
	)
	if err != nil {
		return nil, err
	}
	// just in case...
	if p.Size == 0 {
		p.Size = 1
	}
	if p.Type == types.TransferType_MARGIN_LOW {
		return &types.TransferRequest{
			FromAccount: []*types.Account{
				genAcc,
			},
			ToAccount: []*types.Account{
				marginAcc,
			},
			Amount:    uint64(p.Amount.Amount) * p.Size,
			MinAmount: uint64(p.Amount.MinAmount),
			Asset:     asset,
			Reference: p.Type.String(),
		}, nil
	}
	return &types.TransferRequest{
		FromAccount: []*types.Account{
			marginAcc,
		},
		ToAccount: []*types.Account{
			genAcc,
		},
		Amount:    uint64(p.Amount.Amount) * p.Size,
		MinAmount: uint64(p.Amount.MinAmount),
		Asset:     asset,
		Reference: p.Type.String(),
	}, nil
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
		if acc.Balance >= amount {
			acc.Balance -= amount
			if err := e.IncrementBalance(acc.Id, -amount); err != nil {
				e.log.Error(
					"Failed to update balance for account",
					logging.String("account-id", acc.Id),
					logging.Int64("balance", acc.Balance),
					logging.Error(err),
				)
				return nil, err
			}
			for _, to = range ret.Balances {
				lm = &types.LedgerEntry{
					FromAccount: acc.Id,
					ToAccount:   to.Account.Id,
					Amount:      parts,
					Reference:   req.Reference,
					Type:        "settlement",
					Timestamp:   e.currentTime,
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
			// partial amount resolves differently
			parts = acc.Balance / int64(len(req.ToAccount))
			if err := e.UpdateBalance(acc.Id, 0); err != nil {
				e.log.Error(
					"Failed to set balance of account to 0",
					logging.String("account-id", acc.Id),
					logging.Error(err),
				)
				return nil, err
			}
			acc.Balance = 0
			for _, to = range ret.Balances {
				lm = &types.LedgerEntry{
					FromAccount: acc.Id,
					ToAccount:   to.Account.Id,
					Amount:      parts,
					Reference:   req.Reference,
					Type:        "settlement",
					Timestamp:   e.currentTime,
				}
				ret.Transfers = append(ret.Transfers, lm)
				to.Account.Balance += parts
				to.Balance += parts
			}
		}
		if amount == 0 {
			break
		}
	}
	return &ret, nil
}

// ClearMarket will remove all monies or accounts for parties allocated for a market (margin accounts)
// when the market reach end of life (maturity)
func (e *Engine) ClearMarket(mktID, asset string, parties []string) ([]*types.TransferResponse, error) {
	// creatre a transfer request that we will reuse all the time in order to make allocations smallers
	req := &types.TransferRequest{
		FromAccount: make([]*types.Account, 1),
		ToAccount:   make([]*types.Account, 1),
		Asset:       asset,
	}

	// assume we have as much transfer response than parties
	resps := make([]*types.TransferResponse, 0, len(parties))

	for _, v := range parties {
		marginAcc, err := e.GetAccountByID(e.accountID(mktID, v, asset, types.AccountType_MARGIN))
		if err != nil {
			e.log.Error(
				"Failed to get the margin account",
				logging.String("trader-id", v),
				logging.String("market-id", mktID),
				logging.String("asset", asset),
				logging.Error(err))
			// just try to do other traders
			continue
		}

		generalAcc, err := e.GetAccountByID(e.accountID("", v, asset, types.AccountType_GENERAL))
		if err != nil {
			e.log.Error(
				"Failed to get the general account",
				logging.String("trader-id", v),
				logging.String("market-id", mktID),
				logging.String("asset", asset),
				logging.Error(err))
			// just try to do other traders
			continue
		}

		req.FromAccount[0] = marginAcc
		req.ToAccount[0] = generalAcc
		req.Amount = uint64(marginAcc.Balance)

		if e.log.GetLevel() == logging.DebugLevel {
			e.log.Debug("Clearing trader margin account",
				logging.String("market-id", mktID),
				logging.String("asset", asset),
				logging.String("trader-id", v),
				logging.Int64("margin-before", marginAcc.Balance),
				logging.Int64("general-before", generalAcc.Balance),
				logging.Int64("general-after", generalAcc.Balance+marginAcc.Balance))
		}

		ledgerEntries, err := e.getLedgerEntries(req)
		if err != nil {
			e.log.Error(
				"Failed to move monies from margin to genral account",
				logging.String("trader-id", v),
				logging.String("market-id", mktID),
				logging.String("asset", asset),
				logging.Error(err))
			// just try to do other traders
			continue
		}

		for _, v := range ledgerEntries.Transfers {
			// increment the to account
			if err := e.IncrementBalance(v.ToAccount, v.Amount); err != nil {
				e.log.Error(
					"Failed to increment balance for account",
					logging.String("account-id", v.ToAccount),
					logging.Int64("amount", v.Amount),
					logging.Error(err),
				)
				return nil, err
			}
		}

		resps = append(resps, ledgerEntries)
	}

	return resps, nil
}

// insert and stuff relate to accounts map from here

// CreateMarketAccounts will create all required accounts for a market once
// a new market is accepted through the network
func (e *Engine) CreateMarketAccounts(marketID, asset string, insurance int64) (insuranceID, settleID string) {
	insuranceID = e.accountID(marketID, "", asset, types.AccountType_INSURANCE)
	_, ok := e.accs[insuranceID]
	if !ok {
		insAcc := &types.Account{
			Id:       insuranceID,
			Asset:    asset,
			Owner:    systemOwner,
			Balance:  insurance,
			MarketID: marketID,
			Type:     types.AccountType_INSURANCE,
		}
		e.accs[insuranceID] = insAcc
		e.buf.Add(*insAcc)

	}
	settleID = e.accountID(marketID, "", asset, types.AccountType_SETTLEMENT)
	_, ok = e.accs[settleID]
	if !ok {
		setAcc := &types.Account{
			Id:       settleID,
			Asset:    asset,
			Owner:    systemOwner,
			Balance:  0,
			MarketID: marketID,
			Type:     types.AccountType_SETTLEMENT,
		}
		e.accs[settleID] = setAcc
		e.buf.Add(*setAcc)
	}

	return
}

// CreateTraderAccount will create trader accounts for a given market
// basically one account per market, per asset for each trader
func (e *Engine) CreateTraderAccount(traderID, marketID, asset string) (marginID, generalID string) {
	// first margin account
	marginID = e.accountID(marketID, traderID, asset, types.AccountType_MARGIN)
	_, ok := e.accs[marginID]
	if !ok {
		acc := &types.Account{
			Id:       marginID,
			Asset:    asset,
			MarketID: marketID,
			Balance:  0,
			Owner:    traderID,
			Type:     types.AccountType_MARGIN,
		}
		e.accs[marginID] = acc
		e.buf.Add(*acc)
	}

	generalID = e.accountID(noMarket, traderID, asset, types.AccountType_GENERAL)
	_, ok = e.accs[generalID]
	if !ok {
		acc := &types.Account{
			Id:       generalID,
			Asset:    asset,
			MarketID: noMarket,
			Balance:  0,
			Owner:    traderID,
			Type:     types.AccountType_GENERAL,
		}
		e.accs[generalID] = acc
		e.buf.Add(*acc)
	}

	return
}

// RemoveDistressed will remove all distressed trader in the event positions
// for a given market and asset
func (e *Engine) RemoveDistressed(traders []events.MarketPosition, marketID, asset string) (*types.TransferResponse, error) {
	tl := len(traders)
	if tl == 0 {
		return nil, nil
	}
	// insurance account is the one we're after
	_, ins, err := e.getSystemAccounts(marketID, asset)
	if err != nil {
		return nil, err
	}
	resp := types.TransferResponse{
		Transfers: make([]*types.LedgerEntry, 0, tl),
	}
	for _, trader := range traders {
		acc, err := e.GetAccountByID(e.accountID(marketID, trader.Party(), asset, types.AccountType_MARGIN))
		if err != nil {
			return nil, err
		}
		// only create a ledger move if the balance is greater than zero
		if acc.Balance > 0 {
			resp.Transfers = append(resp.Transfers, &types.LedgerEntry{
				FromAccount: acc.Id,
				ToAccount:   ins.Id,
				Amount:      acc.Balance,
				Reference:   "close-out distressed",
				Type:        "", // @TODO determine this value
				Timestamp:   e.currentTime,
			})
			if err := e.IncrementBalance(ins.Id, acc.Balance); err != nil {
				return nil, err
			}
			if err := e.UpdateBalance(acc.Id, 0); err != nil {
				return nil, err
			}
		}
		if err := e.removeAccount(acc.Id); err != nil {
			return nil, err
		}
	}
	return &resp, nil
}

// Withdraw will remove the specified amount from the trader
// general account
func (e *Engine) Withdraw(partyID, asset string, amount uint64) error {
	acc, err := e.GetAccountByID(e.accountID("", partyID, asset, types.AccountType_GENERAL))
	if err != nil {
		return ErrAccountDoNotExists
	}

	// check we have more money than required to withdraw
	if uint64(acc.Balance) < amount {
		// if we have less balance than required to withdraw, just set it to 0
		// and return an error
		if err := e.UpdateBalance(acc.Id, 0); err != nil {
			return err
		}
		return fmt.Errorf("withdraw error, required=%v, available=%v", amount, acc.Balance)
	}

	if err := e.IncrementBalance(acc.Id, -int64(amount)); err != nil {
		return err
	}
	return nil
}

// UpdateBalance will update the balance of a given account
func (e *Engine) UpdateBalance(id string, balance int64) error {
	acc, ok := e.accs[id]
	if !ok {
		return ErrAccountDoNotExists
	}
	acc.Balance = balance
	e.buf.Add(*acc)
	return nil
}

// IncrementBalance will increment the balance of a given account
// using the given value
func (e *Engine) IncrementBalance(id string, inc int64) error {
	acc, ok := e.accs[id]
	if !ok {
		return ErrAccountDoNotExists
	}
	acc.Balance += inc
	e.buf.Add(*acc)
	return nil
}

// GetAccountByID will return an account using the given id
func (e *Engine) GetAccountByID(id string) (*types.Account, error) {
	acc, ok := e.accs[id]
	if !ok {
		return nil, ErrAccountDoNotExists
	}
	acccpy := *acc
	return &acccpy, nil
}

func (e *Engine) removeAccount(id string) error {
	delete(e.accs, id)
	return nil
}
