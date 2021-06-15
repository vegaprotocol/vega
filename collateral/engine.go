package collateral

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/crypto"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

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
	// collateral engine have not been initialised properly
	ErrSystemAccountsMissing = errors.New("system accounts missing for collateral engine to work")
	// ErrFeeAccountsMissing signals that a fee account is missing, which may means that the
	// collateral engine have not been initialised properly
	ErrFeeAccountsMissing = errors.New("fee accounts missing for collateral engine to work")
	// ErrTraderAccountsMissing signals that the accounts for this trader do not exists
	ErrTraderAccountsMissing = errors.New("trader accounts missing, cannot collect")
	// ErrAccountDoesNotExist signals that an account par of a transfer do not exists
	ErrAccountDoesNotExist                     = errors.New("account does not exists")
	ErrNoGeneralAccountWhenCreateMarginAccount = errors.New("party general account missing when trying to create a margin account")
	ErrNoGeneralAccountWhenCreateBondAccount   = errors.New("party general account missing when trying to create a bond account")
	ErrMinAmountNotReached                     = errors.New("unable to reach minimum amount transfer")
	ErrPartyHasNoTokenAccount                  = errors.New("no token account for party")
	ErrSettlementBalanceNotZero                = errors.New("settlement balance should be zero") // E991 YOU HAVE TOO MUCH ROPE TO HANG YOURSELF
	// ErrAssetAlreadyEnabled signals the given asset has already been enabled in this engine
	ErrAssetAlreadyEnabled = errors.New("asset already enabled")
	// ErrInvalidAssetID signals that an asset id does not exists
	ErrInvalidAssetID = errors.New("invalid asset ID")
	// ErrInsufficientFundsToPayFees the party do not have enough funds to pay the feeds
	ErrInsufficientFundsToPayFees = errors.New("insufficient funds to pay fees")
	// ErrInvalidTransferTypeForFeeRequest an invalid transfer type was send to build a fee transfer request
	ErrInvalidTransferTypeForFeeRequest = errors.New("an invalid transfer type was send to build a fee transfer request")
	// ErrNotEnoughFundsToWithdraw a party requested to withdraw more than on its general account
	ErrNotEnoughFundsToWithdraw = errors.New("not enough funds to withdraw")
)

// Broker send events
// we no longer need to generate this mock here, we can use the broker/mocks package instead
type Broker interface {
	Send(event events.Event)
	SendBatch(events []events.Event)
}

// Engine is handling the power of the collateral
type Engine struct {
	Config
	log   *logging.Logger
	cfgMu sync.Mutex

	accs map[string]*types.Account
	// map of partyID -> account ID -> account
	// this is used to check if a party have any balances in
	// any assets at all
	partiesAccs  map[string]map[string]*types.Account
	hashableAccs []*types.Account
	broker       Broker
	// could be a unix.Time but storing it like this allow us to now time.UnixNano() all the time
	currentTime int64

	idbuf []byte

	// asset ID to asset
	enabledAssets map[string]types.Asset
}

// New instantiates a new collateral engine
func New(log *logging.Logger, conf Config, broker Broker, now time.Time) (*Engine, error) {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(conf.Level.Get())
	return &Engine{
		log:           log,
		Config:        conf,
		accs:          make(map[string]*types.Account, initialAccountSize),
		partiesAccs:   map[string]map[string]*types.Account{},
		hashableAccs:  []*types.Account{},
		broker:        broker,
		currentTime:   now.UnixNano(),
		idbuf:         make([]byte, 256),
		enabledAssets: map[string]types.Asset{},
	}, nil
}

// OnChainTimeUpdate is used to be specified as a callback in over services
// in order to be called when the chain time is updated (basically EndBlock)
func (e *Engine) OnChainTimeUpdate(_ context.Context, t time.Time) {
	e.currentTime = t.UnixNano()
}

func (e *Engine) HasBalance(party string) bool {
	// FIXME(): we temporary just want to make
	// accs, ok := e.partiesAccs[party]
	// sure that the party ever deposited at least
	// once
	// if !ok {
	// 	return false
	// }

	// for _, acc := range accs {
	// 	if acc.Balance > 0 {
	// 		return true
	// 	}
	// }

	// return false
	_, ok := e.partiesAccs[party]
	return ok
}

func (e *Engine) addPartyAccount(party, accid string, acc *types.Account) {
	accs, ok := e.partiesAccs[party]
	if !ok {
		accs = map[string]*types.Account{}
		e.partiesAccs[party] = accs
	}
	// this is called only when an account is created first time
	// and never twice
	accs[accid] = acc
}

func (e *Engine) rmPartyAccount(party, accid string) {
	// this cannot be called for a party which do not have an account already
	// so no risk here
	accs := e.partiesAccs[party]
	delete(accs, accid)
	// delete if the number of accounts for the party
	// is down to 0
	// FIXME(): for now we do not delete the
	// party, this means that the numbner of
	// party will grow forever if they were to
	// get distressed, or people would be adding
	// funds the withdrawing them forever on load
	// of party, but that is better than having
	// transaction stay in the mempool forever.
	// if len(accs) <= 0 {
	// 	delete(e.partiesAccs, party)
	// }
}

func (e *Engine) removeAccountFromHashableSlice(id string) {
	i := sort.Search(len(e.hashableAccs), func(i int) bool {
		return e.hashableAccs[i].Id >= id
	})

	copy(e.hashableAccs[i:], e.hashableAccs[i+1:])
	e.hashableAccs = e.hashableAccs[:len(e.hashableAccs)-1]
}

func (e *Engine) addAccountToHashableSlice(acc *types.Account) {
	// sell side levels should be ordered in ascending
	i := sort.Search(len(e.hashableAccs), func(i int) bool {
		return e.hashableAccs[i].Id >= acc.Id
	})

	if i < len(e.hashableAccs) && e.hashableAccs[i].Id == acc.Id {
		// for some reason it was already there, return now
		return
	}

	e.hashableAccs = append(e.hashableAccs, nil)
	copy(e.hashableAccs[i+1:], e.hashableAccs[i:])
	e.hashableAccs[i] = acc
}

func (e *Engine) Hash() []byte {
	output := make([]byte, len(e.hashableAccs)*32)
	var i int
	for _, k := range e.hashableAccs {
		bal := k.Balance.Bytes()
		copy(output[i:], bal[:])
		i += 32
	}

	return crypto.Hash(output)
}

// ReloadConf updates the internal configuration of the collateral engine
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

// EnableAsset adds a new asset in the collateral engine
// this enable the asset to be used by new markets or
// parties to deposit funds
func (e *Engine) EnableAsset(ctx context.Context, asset types.Asset) error {
	if e.AssetExists(asset.Id) {
		return ErrAssetAlreadyEnabled
	}
	e.enabledAssets[asset.Id] = asset
	e.broker.Send(events.NewAssetEvent(ctx, asset))
	// then creat a new infrastructure fee account for the asset
	// these are fee related account only
	infraFeeID := e.accountID("", "", asset.Id, types.AccountType_ACCOUNT_TYPE_FEES_INFRASTRUCTURE)
	_, ok := e.accs[infraFeeID]
	if !ok {
		infraFeeAcc := &types.Account{
			Id:       infraFeeID,
			Asset:    asset.Id,
			Owner:    systemOwner,
			Balance:  num.NewUint(0),
			MarketId: noMarket,
			Type:     types.AccountType_ACCOUNT_TYPE_FEES_INFRASTRUCTURE,
		}
		e.accs[infraFeeID] = infraFeeAcc
		e.addAccountToHashableSlice(infraFeeAcc)
		e.broker.Send(events.NewAccountEvent(ctx, *infraFeeAcc))
	}
	externalID := e.accountID(noMarket, systemOwner, asset.Id, types.AccountType_ACCOUNT_TYPE_EXTERNAL)
	if _, ok := e.accs[externalID]; !ok {
		externalAcc := &types.Account{
			Id:       externalID,
			Asset:    asset.Id,
			Owner:    systemOwner,
			Balance:  num.NewUint(0),
			MarketId: noMarket,
			Type:     types.AccountType_ACCOUNT_TYPE_EXTERNAL,
		}
		e.accs[externalID] = externalAcc
		// e.addAccountToHashableSlice(externalAcc)
	}
	e.log.Info("new asset added successfully",
		logging.String("asset-id", asset.Id),
	)
	return nil
}

// AssetExists no errors if the asset exists
func (e *Engine) AssetExists(assetID string) bool {
	_, ok := e.enabledAssets[assetID]
	return ok
}

// this func uses named returns because it makes body of the func look clearer
func (e *Engine) getSystemAccounts(marketID, asset string) (settle, insurance *types.Account, err error) {

	insID := e.accountID(marketID, systemOwner, asset, types.AccountType_ACCOUNT_TYPE_INSURANCE)
	setID := e.accountID(marketID, systemOwner, asset, types.AccountType_ACCOUNT_TYPE_SETTLEMENT)

	if insurance, err = e.GetAccountByID(insID); err != nil {
		if e.log.GetLevel() == logging.DebugLevel {
			e.log.Debug("missing system account",
				logging.String("asset", asset),
				logging.String("id", insID),
				logging.String("market", marketID),
				logging.Error(err),
			)
		}
		err = ErrSystemAccountsMissing
		return
	}

	if settle, err = e.GetAccountByID(setID); err != nil {
		if e.log.GetLevel() == logging.DebugLevel {
			e.log.Debug("missing system account",
				logging.String("asset", asset),
				logging.String("id", setID),
				logging.String("market", marketID),
				logging.Error(err),
			)
		}
		err = ErrSystemAccountsMissing
	}

	return
}

func (e *Engine) TransferFees(ctx context.Context, marketID string, assetID string, ft events.FeesTransfer) ([]*types.TransferResponse, error) {
	return e.transferFees(ctx, marketID, assetID, ft)
}

func (e *Engine) TransferFeesContinuousTrading(ctx context.Context, marketID string, assetID string, ft events.FeesTransfer) ([]*types.TransferResponse, error) {
	if len(ft.Transfers()) <= 0 {
		return nil, nil
	}
	// check quickly that all traders have enough monies in their accounts
	// this may be done only in case of continuous trading
	for party, amount := range ft.TotalFeesAmountPerParty() {
		generalAcc, err := e.GetAccountByID(e.accountID(noMarket, party, assetID, types.AccountType_ACCOUNT_TYPE_GENERAL))
		if err != nil {
			e.log.Error("unable to get party account",
				logging.String("account-type", "general"),
				logging.String("party-id", party),
				logging.String("asset", assetID))
			return nil, ErrAccountDoesNotExist
		}

		marginAcc, err := e.GetAccountByID(e.accountID(marketID, party, assetID, types.AccountType_ACCOUNT_TYPE_MARGIN))
		if err != nil {
			e.log.Error("unable to get party account",
				logging.String("account-type", "margin"),
				logging.String("party-id", party),
				logging.String("asset", assetID),
				logging.String("market-id", marketID))
			return nil, ErrAccountDoesNotExist
		}

		if num.Sum(marginAcc.Balance, generalAcc.Balance).LT(amount) {
			return nil, ErrInsufficientFundsToPayFees
		}
	}

	return e.transferFees(ctx, marketID, assetID, ft)
}

func (e *Engine) transferFees(ctx context.Context, marketID string, assetID string, ft events.FeesTransfer) ([]*types.TransferResponse, error) {
	makerFee, infraFee, liquiFee, err := e.getFeesAccounts(marketID, assetID)
	if err != nil {
		return nil, err
	}

	transfers := ft.Transfers()
	responses := make([]*types.TransferResponse, 0, len(transfers))

	for _, transfer := range transfers {
		req, err := e.getFeeTransferRequest(
			transfer, makerFee, infraFee, liquiFee, marketID, assetID)
		if err != nil {
			e.log.Error("Failed to build transfer request for event",
				logging.Error(err))
			return nil, err
		}

		res, err := e.getLedgerEntries(ctx, req)
		if err != nil {
			e.log.Error("Failed to transfer funds", logging.Error(err))
			return nil, err
		}
		for _, bal := range res.Balances {
			if err := e.IncrementBalance(ctx, bal.Account.Id, bal.Balance); err != nil {
				e.log.Error("Could not update the target account in transfer",
					logging.String("account-id", bal.Account.Id),
					logging.Error(err))
				return nil, err
			}
		}
		responses = append(responses, res)
	}

	return responses, nil
}

// this func uses named returns because it makes body of the func look clearer
func (e *Engine) getFeesAccounts(marketID, asset string) (maker, infra, liqui *types.Account, err error) {
	makerID := e.accountID(marketID, systemOwner, asset, types.AccountType_ACCOUNT_TYPE_FEES_MAKER)
	infraID := e.accountID(noMarket, systemOwner, asset, types.AccountType_ACCOUNT_TYPE_FEES_INFRASTRUCTURE)
	liquiID := e.accountID(marketID, systemOwner, asset, types.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY)

	if maker, err = e.GetAccountByID(makerID); err != nil {
		if e.log.GetLevel() == logging.DebugLevel {
			e.log.Debug("missing fee account",
				logging.String("asset", asset),
				logging.String("id", makerID),
				logging.String("market", marketID),
				logging.Error(err),
			)
		}
		err = ErrFeeAccountsMissing
		return
	}

	if infra, err = e.GetAccountByID(infraID); err != nil {
		if e.log.GetLevel() == logging.DebugLevel {
			e.log.Debug("missing fee account",
				logging.String("asset", asset),
				logging.String("id", infraID),
				logging.Error(err),
			)
		}
	}

	if liqui, err = e.GetAccountByID(liquiID); err != nil {
		if e.log.GetLevel() == logging.DebugLevel {
			e.log.Debug("missing system account",
				logging.String("asset", asset),
				logging.String("id", liquiID),
				logging.String("market", marketID),
				logging.Error(err),
			)
		}
	}

	if err != nil {
		err = ErrFeeAccountsMissing
	}

	return
}

// FinalSettlement will process the list of transfer instructed by other engines
// This func currently only expects TransferType_{LOSS,WIN} transfers
// other transfer types have dedicated funcs (MartToMarket, MarginUpdate)
func (e *Engine) FinalSettlement(ctx context.Context, marketID string, transfers []*types.Transfer) ([]*types.TransferResponse, error) {
	// stop immediately if there aren't any transfers, channels are closed
	if len(transfers) == 0 {
		return nil, nil
	}
	responses := make([]*types.TransferResponse, 0, len(transfers))
	asset := transfers[0].Amount.Asset
	// This is where we'll implement everything
	settle, insurance, err := e.getSystemAccounts(marketID, asset)
	if err != nil {
		e.log.Error(
			"Failed to get system accounts required for final settlement",
			logging.Error(err),
		)
		return nil, err
	}
	// create this event, we're not using it, but it's required in getTransferRequests
	mevt := &marginUpdate{}
	// get the component that calculates the loss socialisation etc... if needed
	for _, transfer := range transfers {
		req, err := e.getTransferRequest(ctx, transfer, settle, insurance, mevt)
		if err != nil {
			e.log.Error(
				"Failed to build transfer request for event",
				logging.Error(err),
			)
			return nil, err
		}
		// set the amount (this can change the req.Amount value if we entered loss socialisation
		res, err := e.getLedgerEntries(ctx, req)
		if err != nil {
			e.log.Error(
				"Failed to transfer funds",
				logging.Error(err),
			)
			return nil, err
		}
		for _, bal := range res.Balances {
			if err := e.UpdateBalance(ctx, bal.Account.Id, bal.Balance); err != nil {
				e.log.Error(
					"Could not update the target account in transfer",
					logging.String("account-id", bal.Account.Id),
					logging.Error(err),
				)
				return nil, err
			}
		}
		responses = append(responses, res)
	}
	return responses, nil
}

// MarkToMarket will run the mark to market settlement over a given set of positions
// return ledger move stuff here, too (separate return value, because we need to stream those)
func (e *Engine) MarkToMarket(ctx context.Context, marketID string, transfers []events.Transfer, asset string) ([]events.Margin, []*types.TransferResponse, error) {
	// stop immediately if there aren't any transfers, channels are closed
	if len(transfers) == 0 {
		return nil, nil, nil
	}
	marginEvts := make([]events.Margin, 0, len(transfers))
	responses := make([]*types.TransferResponse, 0, len(transfers))

	// This is where we'll implement everything
	settle, insurance, err := e.getSystemAccounts(marketID, asset)
	if err != nil {
		e.log.Error(
			"Failed to get system accounts required for MTM settlement",
			logging.Error(err),
		)
		return nil, nil, err
	}
	// get the component that calculates the loss socialisation etc... if needed

	var (
		winidx          int
		expectCollected num.Decimal
		expCollected    = num.NewUint(0)
	)

	// create batch of events
	brokerEvts := make([]events.Event, 0, len(transfers))
	// iterate over transfer until we get the first win, so we need we accumulated all loss
	for i, evt := range transfers {
		transfer := evt.Transfer()

		// get the state of the accounts before processing transfers
		// so they can be used in the marginEvt, and to calculate the missing funds
		generalAcc, err := e.GetAccountByID(e.accountID(noMarket, evt.Party(), asset, types.AccountType_ACCOUNT_TYPE_GENERAL))
		if err != nil {
			e.log.Error("unable to get party account",
				logging.String("account-type", "general"),
				logging.String("party-id", evt.Party()),
				logging.String("asset", asset))
		}

		marginAcc, err := e.GetAccountByID(e.accountID(settle.MarketId, evt.Party(), asset, types.AccountType_ACCOUNT_TYPE_MARGIN))
		if err != nil {
			e.log.Error("unable to get party account",
				logging.String("account-type", "margin"),
				logging.String("party-id", evt.Party()),
				logging.String("asset", asset),
				logging.String("market-id", settle.MarketId))
		}

		marginEvt := &marginUpdate{
			MarketPosition: evt,
			asset:          asset,
			marketID:       settle.MarketId,
		}
		// no transfer needed if transfer is nil, just build the marginUpdate
		if transfer == nil {
			marginEvt.general = generalAcc
			marginEvt.margin = marginAcc
			marginEvts = append(marginEvts, marginEvt)
			continue
		}

		if transfer.Type == types.TransferType_TRANSFER_TYPE_MTM_WIN {
			// we processed all loss break then
			winidx = i
			break
		}

		req, err := e.getTransferRequest(ctx, transfer, settle, insurance, marginEvt)
		if err != nil {
			e.log.Error(
				"Failed to build transfer request for event",
				logging.Error(err),
			)
			return nil, nil, err
		}
		// accumulate the expected transfer size
		expCollected.AddSum(req.Amount)
		expectCollected = expectCollected.Add(num.DecimalFromUint(req.Amount))

		// set the amount (this can change the req.Amount value if we entered loss socialisation
		res, err := e.getLedgerEntries(ctx, req)
		if err != nil {
			e.log.Error(
				"Failed to transfer funds",
				logging.Error(err),
			)
			return nil, nil, err
		}

		amountCollected := num.NewUint(0)
		// // update the to accounts now
		for _, bal := range res.Balances {
			amountCollected.AddSum(bal.Balance)
			if err := e.IncrementBalance(ctx, bal.Account.Id, bal.Balance); err != nil {
				e.log.Error(
					"Could not update the target account in transfer",
					logging.String("account-id", bal.Account.Id),
					logging.Error(err),
				)
				return nil, nil, err
			}
		}

		// here we check if we were able to collect all monies,
		// if not send an event to notify the plugins
		if totalInAccount := num.Sum(marginAcc.Balance, generalAcc.Balance); totalInAccount.LT(req.Amount) {
			lsevt := &lossSocializationEvt{
				market:     settle.MarketId,
				party:      evt.Party(),
				amountLost: num.NewUint(0).Sub(req.Amount, totalInAccount),
			}

			e.log.Warn("loss socialization missing amount to be collected or used from insurance pool",
				logging.String("party-id", lsevt.party),
				logging.BigUint("amount", lsevt.amountLost),
				logging.String("market-id", lsevt.market))

			delta, neg := req.Amount.Delta(req.Amount, totalInAccount)
			brokerEvts = append(brokerEvts,
				events.NewLossSocializationEvent(ctx, evt.Party(), settle.MarketId, delta, neg, e.currentTime))
		}

		// updating the accounts stored in the marginEvt
		marginEvt.general, err = e.GetAccountByID(marginEvt.general.Id)
		if err != nil {
			e.log.Error("unable to get party account",
				logging.String("account-type", "general"),
				logging.String("party-id", evt.Party()),
				logging.String("asset", asset))
		}
		marginEvt.margin, err = e.GetAccountByID(marginEvt.margin.Id)
		if err != nil {
			e.log.Error("unable to get party account",
				logging.String("account-type", "margin"),
				logging.String("party-id", evt.Party()),
				logging.String("asset", asset),
				logging.String("market-id", settle.MarketId))
		}

		responses = append(responses, res)
		marginEvts = append(marginEvts, marginEvt)
	}

	if len(brokerEvts) > 0 {
		e.broker.SendBatch(brokerEvts)
	}
	// we couldn't have reached this point without settlement account
	settle, _, _ = e.getSystemAccounts(marketID, asset)
	// if winidx is 0, this means we had now wind and loss, but may have some event which
	// needs to be propagated forward so we return now.
	if winidx == 0 {
		if !settle.Balance.IsZero() {
			return nil, nil, ErrSettlementBalanceNotZero
		}
		return marginEvts, responses, nil
	}

	// now compare what's in the settlement account what we expect initially to redistribute.
	// if there's not enough we enter loss socialization
	distr := simpleDistributor{
		log:             e.log,
		marketID:        settle.MarketId,
		expectCollected: expCollected,
		collected:       settle.Balance,
		requests:        []request{},
		ts:              e.currentTime,
	}

	if distr.LossSocializationEnabled() {
		e.log.Warn("Entering loss socialization",
			logging.String("market-id", marketID),
			logging.String("asset", asset),
			logging.BigUint("expect-collected", expCollected),
			logging.BigUint("collected", settle.Balance))
		for _, evt := range transfers[winidx:] {
			transfer := evt.Transfer()
			if transfer != nil && transfer.Type == types.TransferType_TRANSFER_TYPE_MTM_WIN {
				distr.Add(evt.Transfer())
			}
		}
		evts := distr.Run(ctx)
		e.broker.SendBatch(evts)
	}

	// then we process all the wins
	for _, evt := range transfers[winidx:] {
		transfer := evt.Transfer()
		marginEvt := &marginUpdate{
			MarketPosition: evt,
			asset:          asset,
			marketID:       settle.MarketId,
		}
		// no transfer needed if transfer is nil, just build the marginUpdate
		if transfer == nil {
			marginEvt.general, err = e.GetAccountByID(e.accountID(noMarket, evt.Party(), asset, types.AccountType_ACCOUNT_TYPE_GENERAL))
			if err != nil {
				e.log.Error("unable to get party account",
					logging.String("account-type", "general"),
					logging.String("party-id", evt.Party()),
					logging.String("asset", asset))
			}

			marginEvt.margin, err = e.GetAccountByID(e.accountID(settle.MarketId, evt.Party(), asset, types.AccountType_ACCOUNT_TYPE_MARGIN))
			if err != nil {
				e.log.Error("unable to get party account",
					logging.String("account-type", "margin"),
					logging.String("party-id", evt.Party()),
					logging.String("asset", asset),
					logging.String("market-id", settle.MarketId))
			}

			marginEvts = append(marginEvts, marginEvt)
			continue
		}

		req, err := e.getTransferRequest(ctx, transfer, settle, insurance, marginEvt)
		if err != nil {
			e.log.Error(
				"Failed to build transfer request for event",
				logging.Error(err),
			)
			return nil, nil, err
		}

		// set the amount (this can change the req.Amount value if we entered loss socialisation
		res, err := e.getLedgerEntries(ctx, req)
		if err != nil {
			e.log.Error(
				"Failed to transfer funds",
				logging.Error(err),
			)
			return nil, nil, err
		}

		// update the to accounts now
		for _, bal := range res.Balances {
			if err := e.IncrementBalance(ctx, bal.Account.Id, bal.Balance); err != nil {
				e.log.Error(
					"Could not update the target account in transfer",
					logging.String("account-id", bal.Account.Id),
					logging.Error(err),
				)
				return nil, nil, err
			}
		}
		// updating the accounts stored in the marginEvt
		marginEvt.general, err = e.GetAccountByID(marginEvt.general.Id)
		if err != nil {
			e.log.Error("unable to get party account",
				logging.String("account-type", "general"),
				logging.String("party-id", evt.Party()),
				logging.String("asset", asset))
		}

		marginEvt.margin, err = e.GetAccountByID(marginEvt.margin.Id)
		if err != nil {
			e.log.Error("unable to get party account",
				logging.String("account-type", "margin"),
				logging.String("party-id", evt.Party()),
				logging.String("asset", asset),
				logging.String("market-id", settle.MarketId))
		}

		responses = append(responses, res)
		marginEvts = append(marginEvts, marginEvt)
	}

	if !settle.Balance.IsZero() {
		return nil, nil, ErrSettlementBalanceNotZero
	}
	return marginEvts, responses, nil
}

// GetPartyMargin will return the current margin for a given party
func (e *Engine) GetPartyMargin(pos events.MarketPosition, asset, marketID string) (events.Margin, error) {
	genID := e.accountID("", pos.Party(), asset, types.AccountType_ACCOUNT_TYPE_GENERAL)
	marginID := e.accountID(marketID, pos.Party(), asset, types.AccountType_ACCOUNT_TYPE_MARGIN)
	bondID := e.accountID(marketID, pos.Party(), asset, types.AccountType_ACCOUNT_TYPE_BOND)
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
			logging.String("market-id", marketID))
		return nil, ErrTraderAccountsMissing
	}
	// do not check error,
	// not all parties have a bond account
	bondAcc, _ := e.GetAccountByID(bondID)

	return marginUpdate{
		MarketPosition:  pos,
		margin:          marAcc,
		general:         genAcc,
		lock:            nil,
		bond:            bondAcc,
		asset:           asset,
		marketID:        marketID,
		marginShortFall: nil,
	}, nil
}

// MarginUpdate will run the margin updates over a set of risk events (margin updates)
func (e *Engine) MarginUpdate(ctx context.Context, marketID string, updates []events.Risk) ([]*types.TransferResponse, []events.Margin, []events.Margin, error) {
	response := make([]*types.TransferResponse, 0, len(updates))
	var (
		closed     = make([]events.Margin, 0, len(updates)/2) // half the cap, if we have more than that, the slice will double once, and will fit all updates anyway
		toPenalise = []events.Margin{}
		settle     = &types.Account{
			MarketId: marketID,
		}
	)
	// create "fake" settle account for market ID
	for _, update := range updates {
		transfer := update.Transfer()
		// although this is mainly a duplicate event, we need to pass it to getTransferRequest
		mevt := &marginUpdate{
			MarketPosition: update,
			asset:          update.Asset(),
			marketID:       update.MarketID(),
		}

		req, err := e.getTransferRequest(ctx, transfer, settle, nil, mevt)
		if err != nil {
			return nil, nil, nil, err
		}

		// calculate the marginShortFall in case of a liquidityProvider
		if mevt.bond != nil && transfer.Amount.Amount.GT(mevt.general.Balance) {
			mevt.marginShortFall = num.NewUint(0).Sub(transfer.Amount.Amount, mevt.general.Balance)
		}

		res, err := e.getLedgerEntries(ctx, req)
		if err != nil {
			return nil, nil, nil, err
		}
		// we didn't manage to top up to even the minimum required system margin, close out trader
		// we need to be careful with this, only apply this to transfer for low margin
		// the MinAmount in the transfer is always set to 0 but in 2 case:
		// - first when a new order is created, the MinAmount is the same than Amount, which is
		//   what's required to reach the InitialMargin level
		// - second when a trader margin is under the MaintenanceLevel, the MinAmount is supposed
		//   to be at least to get back to the search level, and the amount will be enough to reach
		//   InitialMargin
		// In both case either the order will not be accepted, or the trader will be closed
		if transfer.Type == types.TransferType_TRANSFER_TYPE_MARGIN_LOW &&
			res.Balances[0].Account.Balance.LT(num.Sum(update.MarginBalance(), transfer.MinAmount)) {
			closed = append(closed, mevt)
		} else if mevt.marginShortFall != nil && !mevt.marginShortFall.IsZero() {
			// party not closed out, but could also not fulfill it's margin requirement
			// from it's general account we need to return this information so penalty can be
			// calculated an taken out from him.
			toPenalise = append(toPenalise, mevt)
		}
		response = append(response, res)
		for _, v := range res.Transfers {
			// increment the to account
			if err := e.IncrementBalance(ctx, v.ToAccount, v.Amount); err != nil {
				e.log.Error(
					"Failed to increment balance for account",
					logging.String("account-id", v.ToAccount),
					logging.BigUint("amount", v.Amount),
					logging.Error(err),
				)
			}
		}
	}

	return response, closed, toPenalise, nil
}

// RollbackMarginUpdateOnOrder moves funds from the margin to the general account.
func (e *Engine) RollbackMarginUpdateOnOrder(ctx context.Context, marketID string, assetID string, transfer *types.Transfer) (*types.TransferResponse, error) {
	margin, err := e.GetAccountByID(e.accountID(marketID, transfer.Owner, assetID, types.AccountType_ACCOUNT_TYPE_MARGIN))
	if err != nil {
		e.log.Error(
			"Failed to get the margin trader account",
			logging.String("owner-id", transfer.Owner),
			logging.String("market-id", marketID),
			logging.Error(err),
		)
		return nil, err
	}
	// we'll need this account for all transfer types anyway (settlements, margin-risk updates)
	general, err := e.GetAccountByID(e.accountID(noMarket, transfer.Owner, assetID, types.AccountType_ACCOUNT_TYPE_GENERAL))
	if err != nil {
		e.log.Error(
			"Failed to get the general trader account",
			logging.String("owner-id", transfer.Owner),
			logging.String("market-id", marketID),
			logging.Error(err),
		)
		return nil, err
	}

	req := &types.TransferRequest{
		FromAccount: []*types.Account{
			margin,
		},
		ToAccount: []*types.Account{
			general,
		},
		Amount:    transfer.Amount.Amount.Clone(),
		MinAmount: transfer.MinAmount,
		Asset:     assetID,
		Reference: transfer.Type.String(),
	}
	if req.MinAmount != nil {
		req.MinAmount = req.MinAmount.Clone()
	}

	res, err := e.getLedgerEntries(ctx, req)
	if err != nil {
		return nil, err
	}
	for _, v := range res.Transfers {
		// increment the to account
		if err := e.IncrementBalance(ctx, v.ToAccount, v.Amount); err != nil {
			e.log.Error(
				"Failed to increment balance for account",
				logging.String("account-id", v.ToAccount),
				logging.BigUint("amount", v.Amount),
				logging.Error(err),
			)
		}
	}

	return res, nil
}

// BondUpdate is to be used for any bond account transfers.
// Update on new orders, updates on commitment changes, or on slashing
func (e *Engine) BondUpdate(ctx context.Context, market, party string, transfer *types.Transfer) (*types.TransferResponse, error) {
	req, err := e.getBondTransferRequest(transfer, market)
	if err != nil {
		return nil, err
	}

	res, err := e.getLedgerEntries(ctx, req)
	if err != nil {
		return nil, err
	}

	for _, v := range res.Transfers {
		// increment the to account
		if err := e.IncrementBalance(ctx, v.ToAccount, v.Amount); err != nil {
			e.log.Error(
				"Failed to increment balance for account",
				logging.String("account-id", v.ToAccount),
				logging.BigUint("amount", v.Amount),
				logging.Error(err),
			)
		}
	}

	return res, nil

}

// MarginUpdateOnOrder will run the margin updates over a set of risk events (margin updates)
func (e *Engine) MarginUpdateOnOrder(ctx context.Context, marketID string, update events.Risk) (*types.TransferResponse, events.Margin, error) {
	// create "fake" settle account for market ID
	settle := &types.Account{
		MarketId: marketID,
	}
	transfer := update.Transfer()
	// although this is mainly a duplicate event, we need to pass it to getTransferRequest
	mevt := marginUpdate{
		MarketPosition: update,
		asset:          update.Asset(),
		marketID:       update.MarketID(),
	}

	req, err := e.getTransferRequest(ctx, transfer, settle, nil, &mevt)
	if err != nil {
		return nil, nil, err
	}

	// we do not have enough money to get to the minimum amount,
	// we return an error.
	if num.Sum(mevt.GeneralBalance(), mevt.MarginBalance()).LT(transfer.MinAmount) {
		return nil, mevt, ErrMinAmountNotReached
	}

	if mevt.bond != nil && transfer.Amount.Amount.GT(mevt.general.Balance) {
		// this is a liquidity provider but it did not have enough funds to
		// pay from the general account, we'll have to penalize later on
		mevt.marginShortFall = num.NewUint(0).Sub(transfer.Amount.Amount, mevt.general.Balance)
	}

	// from here we know there's enough money,
	// let get the ledger entries, return the transfers

	res, err := e.getLedgerEntries(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	for _, v := range res.Transfers {
		// increment the to account
		if err := e.IncrementBalance(ctx, v.ToAccount, v.Amount); err != nil {
			e.log.Error(
				"Failed to increment balance for account",
				logging.String("account-id", v.ToAccount),
				logging.BigUint("amount", v.Amount),
				logging.Error(err),
			)
		}
	}

	if mevt.marginShortFall != nil && !mevt.marginShortFall.IsZero() {
		return res, mevt, nil
	}
	return res, nil, nil
}

func (e *Engine) getFeeTransferRequest(
	t *types.Transfer,
	makerFee, infraFee, liquiFee *types.Account,
	marketID, assetID string,
) (*types.TransferRequest, error) {
	var (
		err             error
		margin, general *types.Account
	)

	// the accounts for the trader we need

	// we do not load the margin all the time
	// as do not always need it.
	getMargin := func() (*types.Account, error) {
		margin, err = e.GetAccountByID(e.accountID(marketID, t.Owner, assetID, types.AccountType_ACCOUNT_TYPE_MARGIN))
		if err != nil {
			e.log.Error(
				"Failed to get the margin trader account",
				logging.String("owner-id", t.Owner),
				logging.String("market-id", marketID),
				logging.Error(err),
			)
			return nil, err
		}
		return margin, err
	}
	general, err = e.GetAccountByID(e.accountID(noMarket, t.Owner, assetID, types.AccountType_ACCOUNT_TYPE_GENERAL))
	if err != nil {
		e.log.Error(
			"Failed to get the general trader account",
			logging.String("owner-id", t.Owner),
			logging.String("market-id", marketID),
			logging.Error(err),
		)
		return nil, err
	}

	treq := &types.TransferRequest{
		Amount:    t.Amount.Amount.Clone(),
		MinAmount: t.Amount.Amount.Clone(),
		Asset:     assetID,
		Reference: t.Type.String(),
	}

	switch t.Type {
	case types.TransferType_TRANSFER_TYPE_INFRASTRUCTURE_FEE_PAY:
		margin, err := getMargin()
		if err != nil {
			return nil, err
		}
		treq.FromAccount = []*types.Account{general, margin}
		treq.ToAccount = []*types.Account{infraFee}
		return treq, nil
	case types.TransferType_TRANSFER_TYPE_INFRASTRUCTURE_FEE_DISTRIBUTE:
		treq.FromAccount = []*types.Account{infraFee}
		treq.ToAccount = []*types.Account{general}
		return treq, nil
	case types.TransferType_TRANSFER_TYPE_LIQUIDITY_FEE_PAY:
		margin, err := getMargin()
		if err != nil {
			return nil, err
		}
		treq.FromAccount = []*types.Account{general, margin}
		treq.ToAccount = []*types.Account{liquiFee}
		return treq, nil
	case types.TransferType_TRANSFER_TYPE_LIQUIDITY_FEE_DISTRIBUTE:
		margin, err := getMargin()
		if err != nil {
			return nil, err
		}
		treq.FromAccount = []*types.Account{liquiFee}
		treq.ToAccount = []*types.Account{margin}
		return treq, nil
	case types.TransferType_TRANSFER_TYPE_MAKER_FEE_PAY:
		margin, err := getMargin()
		if err != nil {
			return nil, err
		}
		treq.FromAccount = []*types.Account{general, margin}
		treq.ToAccount = []*types.Account{makerFee}
		return treq, nil
	case types.TransferType_TRANSFER_TYPE_MAKER_FEE_RECEIVE:
		treq.FromAccount = []*types.Account{makerFee}
		treq.ToAccount = []*types.Account{general}
		return treq, nil
	default:
		return nil, ErrInvalidTransferTypeForFeeRequest
	}
}

func (e *Engine) getBondTransferRequest(t *types.Transfer, market string) (*types.TransferRequest, error) {
	bond, err := e.GetAccountByID(e.accountID(market, t.Owner, t.Amount.Asset, types.AccountType_ACCOUNT_TYPE_BOND))
	if err != nil {
		e.log.Error(
			"Failed to get the margin trader account",
			logging.String("owner-id", t.Owner),
			logging.String("market-id", market),
			logging.Error(err),
		)
		return nil, err
	}

	// we'll need this account for all transfer types anyway (settlements, margin-risk updates)
	general, err := e.GetAccountByID(e.accountID(noMarket, t.Owner, t.Amount.Asset, types.AccountType_ACCOUNT_TYPE_GENERAL))
	if err != nil {
		e.log.Error(
			"Failed to get the general trader account",
			logging.String("owner-id", t.Owner),
			logging.String("market-id", market),
			logging.Error(err),
		)
		return nil, err
	}

	// we'll need this account for all transfer types anyway (settlements, margin-risk updates)
	insurancePool, err := e.GetAccountByID(e.accountID(market, systemOwner, t.Amount.Asset, types.AccountType_ACCOUNT_TYPE_INSURANCE))
	if err != nil {
		e.log.Error(
			"Failed to get the general trader account",
			logging.String("owner-id", t.Owner),
			logging.String("market-id", market),
			logging.Error(err),
		)
		return nil, err
	}

	treq := &types.TransferRequest{
		Amount:    t.Amount.Amount.Clone(),
		MinAmount: t.Amount.Amount.Clone(),
		Asset:     t.Amount.Asset,
		Reference: t.Type.String(),
	}

	switch t.Type {
	case types.TransferType_TRANSFER_TYPE_BOND_LOW:
		// do we have enough in the general account to make the transfer?
		if !t.Amount.Amount.IsZero() && general.Balance.LT(t.Amount.Amount) {
			return nil, errors.New("not enough collateral in general account")
		}
		treq.FromAccount = []*types.Account{general}
		treq.ToAccount = []*types.Account{bond}
		return treq, nil
	case types.TransferType_TRANSFER_TYPE_BOND_HIGH:
		treq.FromAccount = []*types.Account{bond}
		treq.ToAccount = []*types.Account{general}
		return treq, nil
	case types.TransferType_TRANSFER_TYPE_BOND_SLASHING:
		treq.FromAccount = []*types.Account{bond}
		// it's possible the bond account is insufficient, and falling back to margin balance
		// won't cause a close-out
		if marginAcc, err := e.GetAccountByID(e.accountID(market, t.Owner, t.Amount.Asset, types.AccountType_ACCOUNT_TYPE_MARGIN)); err == nil {
			treq.FromAccount = append(treq.FromAccount, marginAcc)
		}
		treq.ToAccount = []*types.Account{insurancePool}
		return treq, nil
	default:
		return nil, errors.New("unsupported transfer type for bond account")
	}
}

// getTransferRequest builds the request, and sets the required accounts based on the type of the Transfer argument
func (e *Engine) getTransferRequest(_ context.Context, p *types.Transfer, settle, insurance *types.Account, mEvt *marginUpdate) (*types.TransferRequest, error) {
	var (
		asset = p.Amount.Asset
		err   error
		eacc  *types.Account

		req = types.TransferRequest{
			Asset:     asset, // TBC
			Reference: p.Type.String(),
		}
	)
	if p.Type == types.TransferType_TRANSFER_TYPE_MTM_LOSS ||
		p.Type == types.TransferType_TRANSFER_TYPE_WIN ||
		p.Type == types.TransferType_TRANSFER_TYPE_MARGIN_LOW {
		// we do not care about errors here as the bon account is not mandatory for the transfers
		// a partry would have a bond account only if it was also a market maker
		mEvt.bond, _ = e.GetAccountByID(e.accountID(settle.MarketId, p.Owner, asset, types.AccountType_ACCOUNT_TYPE_BOND))
	}

	if settle != nil {
		// the accounts for the trader we need
		mEvt.margin, err = e.GetAccountByID(e.accountID(settle.MarketId, p.Owner, asset, types.AccountType_ACCOUNT_TYPE_MARGIN))
		if err != nil {
			e.log.Error(
				"Failed to get the party margin account",
				logging.String("owner-id", p.Owner),
				logging.String("market-id", settle.MarketId),
				logging.Error(err),
			)
			return nil, err
		}
	}
	// we'll need this account for all transfer types anyway (settlements, margin-risk updates)
	mEvt.general, err = e.GetAccountByID(e.accountID(noMarket, p.Owner, asset, types.AccountType_ACCOUNT_TYPE_GENERAL))
	if err != nil {
		e.log.Error(
			"Failed to get the party general account",
			logging.String("owner-id", p.Owner),
			logging.String("market-id", settle.MarketId),
			logging.Error(err),
		)
		return nil, err
	}
	if p.Type == types.TransferType_TRANSFER_TYPE_WITHDRAW || p.Type == types.TransferType_TRANSFER_TYPE_DEPOSIT {
		// external account:
		eacc, _ = e.GetAccountByID(e.accountID(noMarket, systemOwner, asset, types.AccountType_ACCOUNT_TYPE_EXTERNAL))
	}

	switch p.Type {
	// final settle, or MTM settle, makes no difference, it's win/loss still
	case types.TransferType_TRANSFER_TYPE_LOSS, types.TransferType_TRANSFER_TYPE_MTM_LOSS:
		// losses are collected first from the margin account, then the general account, and finally
		// taken out of the insurance pool
		if mEvt.bond != nil {
			req.FromAccount = []*types.Account{
				mEvt.margin,
				mEvt.general,
				mEvt.bond,
				insurance,
			}
		} else {
			req.FromAccount = []*types.Account{
				mEvt.margin,
				mEvt.general,
				insurance,
			}
		}
		req.ToAccount = []*types.Account{
			settle,
		}
		req.Amount = p.Amount.Amount.Clone()
		req.MinAmount = num.NewUint(0) // default value, but keep it here explicitly
	case types.TransferType_TRANSFER_TYPE_WIN, types.TransferType_TRANSFER_TYPE_MTM_WIN:
		// the insurance pool in the Req.FromAccountAccount is not used ATM (losses should fully cover wins
		// or the insurance pool has already been drained).
		req.FromAccount = []*types.Account{
			settle,
			insurance,
		}
		req.ToAccount = []*types.Account{
			mEvt.margin,
		}
		req.Amount = p.Amount.Amount.Clone()
		req.MinAmount = num.NewUint(0) // default value, but keep it here explicitly
	case types.TransferType_TRANSFER_TYPE_MARGIN_LOW:
		if mEvt.bond != nil {
			req.FromAccount = []*types.Account{
				mEvt.general,
				mEvt.bond,
			}
		} else {
			req.FromAccount = []*types.Account{
				mEvt.general,
			}
		}
		req.ToAccount = []*types.Account{
			mEvt.margin,
		}
		req.Amount = p.Amount.Amount
		req.MinAmount = p.MinAmount
	case types.TransferType_TRANSFER_TYPE_MARGIN_HIGH:
		req.FromAccount = []*types.Account{
			mEvt.margin,
		}
		req.ToAccount = []*types.Account{
			mEvt.general,
		}
		req.Amount = p.Amount.Amount
		req.MinAmount = p.MinAmount
	case types.TransferType_TRANSFER_TYPE_WITHDRAW_LOCK:
		req.FromAccount = []*types.Account{
			mEvt.general,
		}
		req.ToAccount = []*types.Account{
			mEvt.lock,
		}
		req.Amount = p.Amount.Amount
		req.MinAmount = p.Amount.Amount
	case types.TransferType_TRANSFER_TYPE_DEPOSIT:
		// ensure we have the funds req.ToAccount deposit
		eacc.Balance = eacc.Balance.Add(eacc.Balance, p.Amount.Amount)
		req.FromAccount = []*types.Account{
			eacc,
		}
		req.ToAccount = []*types.Account{
			mEvt.general,
		}
		req.Amount = p.Amount.Amount
		req.MinAmount = p.Amount.Amount
	case types.TransferType_TRANSFER_TYPE_WITHDRAW:
		req.FromAccount = []*types.Account{
			mEvt.lock,
		}
		req.ToAccount = []*types.Account{
			eacc,
		}
		req.Amount = p.Amount.Amount
		req.MinAmount = p.Amount.Amount
	default:
		return nil, errors.New("unexpected transfer type")
	}

	return &req, nil
}

// this builds a TransferResponse for a specific request, we collect all of them and aggregate
func (e *Engine) getLedgerEntries(ctx context.Context, req *types.TransferRequest) (*types.TransferResponse, error) {
	ret := types.TransferResponse{
		Transfers: []*types.LedgerEntry{},
		Balances:  make([]*types.TransferBalance, 0, len(req.ToAccount)),
	}
	for _, t := range req.ToAccount {
		ret.Balances = append(ret.Balances, &types.TransferBalance{
			Account: t,
			Balance: num.NewUint(0),
		})
	}
	amount := req.Amount
	for _, acc := range req.FromAccount {
		// give each to account an equal share
		nToAccounts := num.NewUint(uint64(len(req.ToAccount)))
		parts := num.NewUint(0).Div(amount, nToAccounts)
		// add remaining pennies to last ledger movement
		remainder := num.NewUint(0).Mod(amount, nToAccounts)
		var (
			to *types.TransferBalance
			lm *types.LedgerEntry
		)
		// either the account contains enough, or we're having to access insurance pool money
		if acc.Balance.GTE(amount) {
			acc.Balance.Sub(acc.Balance, amount)
			if err := e.UpdateBalance(ctx, acc.Id, acc.Balance); err != nil {
				e.log.Error(
					"Failed to update balance for account",
					logging.String("account-id", acc.Id),
					logging.BigUint("balance", acc.Balance),
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
				to.Balance.AddSum(parts)
				to.Account.Balance.AddSum(parts)
			}
			// add remainder
			if !remainder.IsZero() {
				lm.Amount.AddSum(remainder)
				to.Balance.AddSum(remainder)
				to.Account.Balance.AddSum(remainder)
			}
			return &ret, nil
		}
		if !acc.Balance.IsZero() {
			amount.Sub(amount, acc.Balance)
			// partial amount resolves differently
			parts.Div(acc.Balance, nToAccounts)
			acc.Balance.SetUint64(0)
			if err := e.UpdateBalance(ctx, acc.Id, acc.Balance); err != nil {
				e.log.Error(
					"Failed to set balance of account to 0",
					logging.String("account-id", acc.Id),
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
				to.Account.Balance.AddSum(parts)
				to.Balance.AddSum(parts)
			}
		}
		if amount.IsZero() {
			break
		}
	}
	return &ret, nil
}

func (e *Engine) clearAccount(
	ctx context.Context, req *types.TransferRequest,
	party, asset, market string,
) (*types.TransferResponse, error) {
	ledgerEntries, err := e.getLedgerEntries(ctx, req)
	if err != nil {
		e.log.Error(
			"Failed to move monies from margin to general account",
			logging.PartyID(party),
			logging.MarketID(market),
			logging.AssetID(asset),
			logging.Error(err))
		return nil, err
	}

	for _, v := range ledgerEntries.Transfers {
		// increment the to account
		if err := e.IncrementBalance(ctx, v.ToAccount, v.Amount); err != nil {
			e.log.Error(
				"Failed to increment balance for account",
				logging.String("account-id", v.ToAccount),
				logging.BigUint("amount", v.Amount),
				logging.Error(err),
			)
			return nil, err
		}
	}

	// we remove the margin account
	if err := e.removeAccount(req.FromAccount[0].Id); err != nil {
		return nil, err
	}
	// remove account from balances tracking
	e.rmPartyAccount(party, req.FromAccount[0].Id)

	return ledgerEntries, nil
}

// ClearMarket will remove all monies or accounts for parties allocated for a market (margin accounts)
// when the market reach end of life (maturity)
func (e *Engine) ClearMarket(ctx context.Context, mktID, asset string, parties []string) ([]*types.TransferResponse, error) {
	// create a transfer request that we will reuse all the time in order to make allocations smaller
	req := &types.TransferRequest{
		FromAccount: make([]*types.Account, 1),
		ToAccount:   make([]*types.Account, 1),
		Asset:       asset,
		Reference:   "clear-market",
	}

	// assume we have as much transfer response than parties
	resps := make([]*types.TransferResponse, 0, len(parties))

	for _, v := range parties {

		generalAcc, err := e.GetAccountByID(e.accountID("", v, asset, types.AccountType_ACCOUNT_TYPE_GENERAL))
		if err != nil {
			e.log.Debug(
				"Failed to get the general account",
				logging.String("trader-id", v),
				logging.String("market-id", mktID),
				logging.String("asset", asset),
				logging.Error(err))
			// just try to do other traders
			continue
		}

		// we start first with the margin account if it exists
		marginAcc, err := e.GetAccountByID(e.accountID(mktID, v, asset, types.AccountType_ACCOUNT_TYPE_MARGIN))
		if err != nil {
			e.log.Debug(
				"Failed to get the margin account",
				logging.String("trader-id", v),
				logging.String("market-id", mktID),
				logging.String("asset", asset),
				logging.Error(err))
		} else {
			req.FromAccount[0] = marginAcc
			req.ToAccount[0] = generalAcc
			req.Amount = marginAcc.Balance

			if e.log.GetLevel() == logging.DebugLevel {
				e.log.Debug("Clearing party margin account",
					logging.String("market-id", mktID),
					logging.String("asset", asset),
					logging.String("party", v),
					logging.BigUint("margin-before", marginAcc.Balance),
					logging.BigUint("general-before", generalAcc.Balance),
					logging.BigUint("general-after", num.Sum(generalAcc.Balance, marginAcc.Balance)))
			}

			ledgerEntries, err := e.clearAccount(ctx, req, v, asset, mktID)
			if err != nil {
				e.log.Panic("unable to clear party account", logging.Error(err))
			}

			// as the entries to the response
			resps = append(resps, ledgerEntries)
		}

		// Then we do bond account
		bondAcc, err := e.GetAccountByID(e.accountID(mktID, v, asset, types.AccountType_ACCOUNT_TYPE_BOND))
		if err != nil {
			// this not an actual error
			// a party may not have a bond account if
			// its not also a liquidity provider
			continue
		}

		req.FromAccount[0] = bondAcc
		req.ToAccount[0] = generalAcc
		req.Amount = bondAcc.Balance.Clone()

		if e.log.GetLevel() == logging.DebugLevel {
			e.log.Debug("Clearing party bond account",
				logging.String("market-id", mktID),
				logging.String("asset", asset),
				logging.String("party", v),
				logging.BigUint("bond-before", bondAcc.Balance),
				logging.BigUint("general-before", generalAcc.Balance),
				logging.BigUint("general-after", num.Sum(generalAcc.Balance, marginAcc.Balance)))
		}

		ledgerEntries, err := e.clearAccount(ctx, req, v, asset, mktID)
		if err != nil {
			e.log.Panic("unable to clear party account", logging.Error(err))
		}

		// add entries to the response
		resps = append(resps, ledgerEntries)
	}

	return resps, nil
}

func (e *Engine) CanCoverBond(market, party, asset string, amount *num.Uint) bool {
	bondID := e.accountID(
		market, party, asset, types.AccountType_ACCOUNT_TYPE_BOND,
	)
	genID := e.accountID(
		noMarket, party, asset, types.AccountType_ACCOUNT_TYPE_GENERAL,
	)

	availableBalance := num.NewUint(0)

	bondAcc, ok := e.accs[bondID]
	if ok {
		availableBalance.AddSum(bondAcc.Balance)
	}
	genAcc, ok := e.accs[genID]
	if ok {
		availableBalance.AddSum(genAcc.Balance)
	}

	return availableBalance.GTE(amount)
}

// GetOrCreatePartyBondAccount returns a bond account given a set of parameters.
// crates it if not exists
func (e *Engine) GetOrCreatePartyBondAccount(ctx context.Context, partyID, marketID, asset string) (*types.Account, error) {
	if !e.AssetExists(asset) {
		return nil, ErrInvalidAssetID
	}

	bondID, err := e.CreatePartyBondAccount(ctx, partyID, marketID, asset)
	if err != nil {
		return nil, err
	}
	return e.GetAccountByID(bondID)
}

// CreatePartyBondAccount creates a bond account if it does not exist, will return an error
// if no general account exist for the trader for the given asset
func (e *Engine) CreatePartyBondAccount(ctx context.Context, partyID, marketID, asset string) (string, error) {
	if !e.AssetExists(asset) {
		return "", ErrInvalidAssetID
	}
	bondID := e.accountID(marketID, partyID, asset, types.AccountType_ACCOUNT_TYPE_BOND)
	if _, ok := e.accs[bondID]; !ok {
		// OK no bond ID, so let's try to get the general id then
		// first check if general account exists
		generalID := e.accountID(noMarket, partyID, asset, types.AccountType_ACCOUNT_TYPE_GENERAL)
		if _, ok := e.accs[generalID]; !ok {
			e.log.Error("Tried to create a bond account for a party with no general account",
				logging.String("party-id", partyID),
				logging.String("asset", asset),
				logging.String("market-id", marketID),
			)
			return "", ErrNoGeneralAccountWhenCreateBondAccount
		}

		// general account id OK, let's create a margin account
		acc := types.Account{
			Id:       bondID,
			Asset:    asset,
			MarketId: marketID,
			Balance:  num.NewUint(0),
			Owner:    partyID,
			Type:     types.AccountType_ACCOUNT_TYPE_BOND,
		}
		e.accs[bondID] = &acc
		e.addPartyAccount(partyID, bondID, &acc)
		e.addAccountToHashableSlice(&acc)
		e.broker.Send(events.NewAccountEvent(ctx, acc))
	}
	return bondID, nil
}

// CreatePartyMarginAccount creates a margin account if it does not exist, will return an error
// if no general account exist for the trader for the given asset
func (e *Engine) CreatePartyMarginAccount(ctx context.Context, partyID, marketID, asset string) (string, error) {
	if !e.AssetExists(asset) {
		return "", ErrInvalidAssetID
	}
	marginID := e.accountID(marketID, partyID, asset, types.AccountType_ACCOUNT_TYPE_MARGIN)
	if _, ok := e.accs[marginID]; !ok {
		// OK no margin ID, so let's try to get the general id then
		// first check if general account exists
		generalID := e.accountID(noMarket, partyID, asset, types.AccountType_ACCOUNT_TYPE_GENERAL)
		if _, ok := e.accs[generalID]; !ok {
			e.log.Error("Tried to create a margin account for a party with no general account",
				logging.String("party-id", partyID),
				logging.String("asset", asset),
				logging.String("market-id", marketID),
			)
			return "", ErrNoGeneralAccountWhenCreateMarginAccount
		}

		// general account id OK, let's create a margin account
		acc := types.Account{
			Id:       marginID,
			Asset:    asset,
			MarketId: marketID,
			Balance:  num.NewUint(0),
			Owner:    partyID,
			Type:     types.AccountType_ACCOUNT_TYPE_MARGIN,
		}
		e.accs[marginID] = &acc
		e.addPartyAccount(partyID, marginID, &acc)
		e.addAccountToHashableSlice(&acc)
		e.broker.Send(events.NewAccountEvent(ctx, acc))
	}
	return marginID, nil
}

// GetPartyMarginAccount returns a margin account given the partyID and market
func (e *Engine) GetPartyMarginAccount(market, party, asset string) (*types.Account, error) {
	margin := e.accountID(market, party, asset, types.AccountType_ACCOUNT_TYPE_MARGIN)
	return e.GetAccountByID(margin)
}

// GetPartyGeneralAccount returns a general account given the partyID.
func (e *Engine) GetPartyGeneralAccount(partyID, asset string) (*types.Account, error) {
	generalID := e.accountID(noMarket, partyID, asset, types.AccountType_ACCOUNT_TYPE_GENERAL)
	return e.GetAccountByID(generalID)
}

// GetPartyBondAccount returns a general account given the partyID.
func (e *Engine) GetPartyBondAccount(market, partyID, asset string) (*types.Account, error) {
	id := e.accountID(
		market, partyID, asset, types.AccountType_ACCOUNT_TYPE_BOND)
	return e.GetAccountByID(id)
}

// CreatePartyGeneralAccount create the general account for a trader
func (e *Engine) CreatePartyGeneralAccount(ctx context.Context, partyID, asset string) (string, error) {
	if !e.AssetExists(asset) {
		return "", ErrInvalidAssetID
	}

	generalID := e.accountID(noMarket, partyID, asset, types.AccountType_ACCOUNT_TYPE_GENERAL)
	if _, ok := e.accs[generalID]; !ok {
		acc := types.Account{
			Id:       generalID,
			Asset:    asset,
			MarketId: noMarket,
			Balance:  num.NewUint(0),
			Owner:    partyID,
			Type:     types.AccountType_ACCOUNT_TYPE_GENERAL,
		}
		e.accs[generalID] = &acc
		e.addPartyAccount(partyID, generalID, &acc)
		e.addAccountToHashableSlice(&acc)
		e.broker.Send(events.NewPartyEvent(ctx, types.Party{Id: partyID}))
		e.broker.Send(events.NewAccountEvent(ctx, acc))
	}

	return generalID, nil
}

// GetOrCreatePartyLockWithdrawAccount gets or creates an account to lock funds to be withdrawn by a party
func (e *Engine) GetOrCreatePartyLockWithdrawAccount(ctx context.Context, partyID, asset string) (*types.Account, error) {
	if !e.AssetExists(asset) {
		return nil, ErrInvalidAssetID
	}

	id := e.accountID(noMarket, partyID, asset, types.AccountType_ACCOUNT_TYPE_LOCK_WITHDRAW)
	var (
		acc *types.Account
		ok  bool
	)
	if acc, ok = e.accs[id]; !ok {
		acc = &types.Account{
			Id:       id,
			Asset:    asset,
			MarketId: noMarket,
			Balance:  num.NewUint(0),
			Owner:    partyID,
			Type:     types.AccountType_ACCOUNT_TYPE_LOCK_WITHDRAW,
		}
		e.accs[id] = acc
		e.addPartyAccount(partyID, id, acc)
		e.addAccountToHashableSlice(acc)
		e.broker.Send(events.NewAccountEvent(ctx, *acc))
	}

	return acc, nil
}

// RemoveDistressed will remove all distressed trader in the event positions
// for a given market and asset
func (e *Engine) RemoveDistressed(ctx context.Context, traders []events.MarketPosition, marketID, asset string) (*types.TransferResponse, error) {
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
		bondAcc, err := e.GetAccountByID(e.accountID(marketID, trader.Party(), asset, types.AccountType_ACCOUNT_TYPE_BOND))
		if err != nil {
			bondAcc = &types.Account{}
		}
		genAcc, err := e.GetAccountByID(e.accountID(noMarket, trader.Party(), asset, types.AccountType_ACCOUNT_TYPE_GENERAL))
		if err != nil {
			return nil, err
		}
		marginAcc, err := e.GetAccountByID(e.accountID(marketID, trader.Party(), asset, types.AccountType_ACCOUNT_TYPE_MARGIN))
		if err != nil {
			return nil, err
		}
		// If any balance remains on bond account, move it over to margin account
		if bondAcc.Balance != nil && !bondAcc.Balance.IsZero() {
			resp.Transfers = append(resp.Transfers, &types.LedgerEntry{
				FromAccount: bondAcc.Id,
				ToAccount:   marginAcc.Id,
				Amount:      bondAcc.Balance.Clone(),
				Reference:   types.TransferType_TRANSFER_TYPE_MARGIN_LOW.String(),
				Type:        "position-resolution",
				Timestamp:   e.currentTime,
			})
			if err := e.IncrementBalance(ctx, marginAcc.Id, bondAcc.Balance); err != nil {
				return nil, err
			}
			if err := e.UpdateBalance(ctx, bondAcc.Id, bondAcc.Balance.SetUint64(0)); err != nil {
				return nil, err
			}
		}
		// take whatever is left on the general account, and move to margin balance
		// we can take everything from the account, as whatever amount was left here didn't cover the minimum margin requirement
		if genAcc.Balance != nil && !genAcc.Balance.IsZero() {
			resp.Transfers = append(resp.Transfers, &types.LedgerEntry{
				FromAccount: genAcc.Id,
				ToAccount:   marginAcc.Id,
				Amount:      genAcc.Balance.Clone(),
				Reference:   types.TransferType_TRANSFER_TYPE_MARGIN_LOW.String(),
				Type:        "position-resolution",
				Timestamp:   e.currentTime,
			})
			if err := e.IncrementBalance(ctx, marginAcc.Id, genAcc.Balance); err != nil {
				return nil, err
			}
			if err := e.UpdateBalance(ctx, genAcc.Id, genAcc.Balance.SetUint64(0)); err != nil {
				return nil, err
			}
		}
		// move monies from the margin account (balance is general, bond, and margin combined now)
		if !marginAcc.Balance.IsZero() {
			resp.Transfers = append(resp.Transfers, &types.LedgerEntry{
				FromAccount: marginAcc.Id,
				ToAccount:   ins.Id,
				Amount:      marginAcc.Balance.Clone(),
				Reference:   types.TransferType_TRANSFER_TYPE_MARGIN_CONFISCATED.String(),
				Type:        "position-resolution",
				Timestamp:   e.currentTime,
			})
			if err := e.IncrementBalance(ctx, ins.Id, marginAcc.Balance); err != nil {
				return nil, err
			}
			if err := e.UpdateBalance(ctx, marginAcc.Id, marginAcc.Balance.SetUint64(0)); err != nil {
				return nil, err
			}
		}

		// we remove the margin account
		if err := e.removeAccount(marginAcc.Id); err != nil {
			return nil, err
		}
		// remove account from balances tracking
		e.rmPartyAccount(trader.Party(), marginAcc.Id)

	}
	return &resp, nil
}

func (e *Engine) ClearPartyMarginAccount(ctx context.Context, party, market, asset string) (*types.TransferResponse, error) {
	acc, err := e.GetAccountByID(e.accountID(market, party, asset, types.AccountType_ACCOUNT_TYPE_MARGIN))
	if err != nil {
		return nil, err
	}
	resp := types.TransferResponse{
		Transfers: []*types.LedgerEntry{},
	}

	if !acc.Balance.IsZero() {
		genAcc, err := e.GetAccountByID(e.accountID(noMarket, party, asset, types.AccountType_ACCOUNT_TYPE_GENERAL))
		if err != nil {
			return nil, err
		}

		resp.Transfers = append(resp.Transfers, &types.LedgerEntry{
			FromAccount: acc.Id,
			ToAccount:   genAcc.Id,
			Amount:      acc.Balance.Clone(),
			Reference:   types.TransferType_TRANSFER_TYPE_MARGIN_HIGH.String(),
			Type:        types.TransferType_TRANSFER_TYPE_MARGIN_HIGH.String(),
			Timestamp:   e.currentTime,
		})
		if err := e.IncrementBalance(ctx, genAcc.Id, acc.Balance); err != nil {
			return nil, err
		}
		if err := e.UpdateBalance(ctx, acc.Id, acc.Balance.SetUint64(0)); err != nil {
			return nil, err
		}
	}
	return &resp, nil
}

// CreateMarketAccounts will create all required accounts for a market once
// a new market is accepted through the network
func (e *Engine) CreateMarketAccounts(ctx context.Context, marketID, asset string) (insuranceID, settleID string, err error) {
	if !e.AssetExists(asset) {
		return "", "", ErrInvalidAssetID
	}
	insuranceID = e.accountID(marketID, "", asset, types.AccountType_ACCOUNT_TYPE_INSURANCE)
	_, ok := e.accs[insuranceID]
	if !ok {
		insAcc := &types.Account{
			Id:       insuranceID,
			Asset:    asset,
			Owner:    systemOwner,
			Balance:  num.NewUint(0),
			MarketId: marketID,
			Type:     types.AccountType_ACCOUNT_TYPE_INSURANCE,
		}
		e.accs[insuranceID] = insAcc
		e.addAccountToHashableSlice(insAcc)
		e.broker.Send(events.NewAccountEvent(ctx, *insAcc))

	}
	settleID = e.accountID(marketID, "", asset, types.AccountType_ACCOUNT_TYPE_SETTLEMENT)
	_, ok = e.accs[settleID]
	if !ok {
		setAcc := &types.Account{
			Id:       settleID,
			Asset:    asset,
			Owner:    systemOwner,
			Balance:  num.NewUint(0),
			MarketId: marketID,
			Type:     types.AccountType_ACCOUNT_TYPE_SETTLEMENT,
		}
		e.accs[settleID] = setAcc
		e.addAccountToHashableSlice(setAcc)
		e.broker.Send(events.NewAccountEvent(ctx, *setAcc))
	}

	// these are fee related account only
	liquidityFeeID := e.accountID(marketID, "", asset, types.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY)
	_, ok = e.accs[liquidityFeeID]
	if !ok {
		liquidityFeeAcc := &types.Account{
			Id:       liquidityFeeID,
			Asset:    asset,
			Owner:    systemOwner,
			Balance:  num.NewUint(0),
			MarketId: marketID,
			Type:     types.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY,
		}
		e.accs[liquidityFeeID] = liquidityFeeAcc
		e.addAccountToHashableSlice(liquidityFeeAcc)
		e.broker.Send(events.NewAccountEvent(ctx, *liquidityFeeAcc))
	}
	makerFeeID := e.accountID(marketID, "", asset, types.AccountType_ACCOUNT_TYPE_FEES_MAKER)
	_, ok = e.accs[makerFeeID]
	if !ok {
		makerFeeAcc := &types.Account{
			Id:       makerFeeID,
			Asset:    asset,
			Owner:    systemOwner,
			Balance:  num.NewUint(0),
			MarketId: marketID,
			Type:     types.AccountType_ACCOUNT_TYPE_FEES_MAKER,
		}
		e.accs[makerFeeID] = makerFeeAcc
		e.addAccountToHashableSlice(makerFeeAcc)
		e.broker.Send(events.NewAccountEvent(ctx, *makerFeeAcc))
	}

	return
}

func (e *Engine) HasGeneralAccount(party, asset string) bool {
	_, err := e.GetAccountByID(
		e.accountID("", party, asset, types.AccountType_ACCOUNT_TYPE_GENERAL))
	return err == nil
}

// LockFundsForWithdraw will lock funds in a separate account to be withdrawn later on by the party
func (e *Engine) LockFundsForWithdraw(ctx context.Context, partyID, asset string, amount *num.Uint) (*types.TransferResponse, error) {
	if !e.AssetExists(asset) {
		return nil, ErrInvalidAssetID
	}
	genacc, err := e.GetAccountByID(e.accountID("", partyID, asset, types.AccountType_ACCOUNT_TYPE_GENERAL))
	if err != nil {
		return nil, ErrAccountDoesNotExist
	}
	if amount.GT(genacc.Balance) {
		return nil, ErrNotEnoughFundsToWithdraw
	}
	lacc, err := e.GetOrCreatePartyLockWithdrawAccount(ctx, partyID, asset)
	if err != nil {
		return nil, err
	}
	// @TODO ensure this is safe, the balances are pointers here!
	mEvt := marginUpdate{
		general: genacc,
		lock:    lacc,
	}
	transf := types.Transfer{
		Owner: partyID,
		Amount: &types.FinancialAmount{
			Amount: amount,
			Asset:  asset,
		},
		Type:      types.TransferType_TRANSFER_TYPE_WITHDRAW_LOCK,
		MinAmount: amount,
	}
	req, err := e.getTransferRequest(ctx, &transf, nil, nil, &mEvt)
	if err != nil {
		return nil, err
	}
	res, err := e.getLedgerEntries(ctx, req)
	if err != nil {
		return nil, err
	}
	// ensure events are sent
	for _, bal := range res.Balances {
		if err := e.UpdateBalance(ctx, bal.Account.Id, bal.Account.Balance); err != nil {
			return nil, err
		}
	}
	return res, nil
}

// Withdraw will remove the specified amount from the trader
// general account
func (e *Engine) Withdraw(ctx context.Context, partyID, asset string, amount *num.Uint) (*types.TransferResponse, error) {
	if !e.AssetExists(asset) {
		return nil, ErrInvalidAssetID
	}
	acc, err := e.GetAccountByID(e.accountID("", partyID, asset, types.AccountType_ACCOUNT_TYPE_LOCK_WITHDRAW))
	if err != nil {
		return nil, ErrAccountDoesNotExist
	}

	// check we have more money than required to withdraw
	if acc.Balance.LT(amount) {
		return nil, fmt.Errorf("withdraw error, required=%v, available=%v", amount, acc.Balance)
	}

	transf := types.Transfer{
		Owner: partyID,
		Amount: &types.FinancialAmount{
			Amount: amount.Clone(),
			Asset:  asset,
		},
		Type:      types.TransferType_TRANSFER_TYPE_WITHDRAW,
		MinAmount: amount.Clone(),
	}
	// @TODO ensure this is safe!
	mEvt := marginUpdate{
		lock: acc,
	}
	req, err := e.getTransferRequest(ctx, &transf, nil, nil, &mEvt)
	if err != nil {
		return nil, err
	}

	res, err := e.getLedgerEntries(ctx, req)
	if err != nil {
		return nil, err
	}
	// increment the external account
	// this could probably be done more generically using the response
	if err := e.IncrementBalance(ctx, req.ToAccount[0].Id, amount); err != nil {
		return nil, err
	}

	return res, nil
}

// Deposit will deposit the given amount into the party account
func (e *Engine) Deposit(ctx context.Context, partyID, asset string, amount *num.Uint) (*types.TransferResponse, error) {
	if !e.AssetExists(asset) {
		return nil, ErrInvalidAssetID
	}
	// this will get or create the account basically
	accID, err := e.CreatePartyGeneralAccount(ctx, partyID, asset)
	if err != nil {
		return nil, err
	}
	acc, _ := e.GetAccountByID(accID)
	transf := types.Transfer{
		Owner: partyID,
		Amount: &types.FinancialAmount{
			Amount: amount.Clone(),
			Asset:  asset,
		},
		Type:      types.TransferType_TRANSFER_TYPE_DEPOSIT,
		MinAmount: amount.Clone(),
	}
	// @TODO -> again, is this safe?
	mEvt := marginUpdate{
		general: acc,
	}
	req, err := e.getTransferRequest(ctx, &transf, nil, nil, &mEvt)
	if err != nil {
		return nil, err
	}
	res, err := e.getLedgerEntries(ctx, req)
	if err != nil {
		return nil, err
	}
	// we need to call increment balance here, because we're working on a copy, acc.Balance will still be 100 if we just use Update
	if err := e.IncrementBalance(ctx, acc.Id, amount); err != nil {
		return nil, err
	}

	return res, nil
}

// UpdateBalance will update the balance of a given account
func (e *Engine) UpdateBalance(ctx context.Context, id string, balance *num.Uint) error {
	acc, ok := e.accs[id]
	if !ok {
		return ErrAccountDoesNotExist
	}
	acc.Balance.Set(balance)
	if acc.Type != types.AccountType_ACCOUNT_TYPE_EXTERNAL {
		e.broker.Send(events.NewAccountEvent(ctx, *acc))
	}
	return nil
}

// IncrementBalance will increment the balance of a given account
// using the given value
func (e *Engine) IncrementBalance(ctx context.Context, id string, inc *num.Uint) error {
	acc, ok := e.accs[id]
	if !ok {
		return fmt.Errorf("account does not exist: %s", id)
	}
	acc.Balance.AddSum(inc)
	if acc.Type != types.AccountType_ACCOUNT_TYPE_EXTERNAL {
		e.broker.Send(events.NewAccountEvent(ctx, *acc))
	}
	return nil
}

// DecrementBalance will decrement the balance of a given account
// using the given value
func (e *Engine) DecrementBalance(ctx context.Context, id string, dec *num.Uint) error {
	acc, ok := e.accs[id]
	if !ok {
		return fmt.Errorf("account does not exist: %s", id)
	}
	acc.Balance.Sub(acc.Balance, dec)
	if acc.Type != types.AccountType_ACCOUNT_TYPE_EXTERNAL {
		e.broker.Send(events.NewAccountEvent(ctx, *acc))
	}
	return nil
}

// GetAccountByID will return an account using the given id
func (e *Engine) GetAccountByID(id string) (*types.Account, error) {
	acc, ok := e.accs[id]
	if !ok {
		return nil, fmt.Errorf("account does not exist: %s", id)
	}
	acccpy := *acc
	acccpy.Balance = acccpy.Balance.Clone()
	return &acccpy, nil
}

// GetAssetTotalSupply - return the total supply of the asset if it's known
// from the collateral engine.
func (e *Engine) GetAssetTotalSupply(asset string) (*num.Uint, error) {
	asst, ok := e.enabledAssets[asset]
	if !ok {
		return nil, fmt.Errorf("invalid asset: %s", asset)
	}

	return asst.TotalSupply.Clone(), nil
}

func (e *Engine) removeAccount(id string) error {
	delete(e.accs, id)
	e.removeAccountFromHashableSlice(id)
	return nil
}

// @TODO this function uses a single slice for each call. This is fine now, as we're processing
// everything sequentially, and so there's no possible data-races here. Once we start doing things
// like cleaning up expired markets asynchronously, then this func is not safe for concurrent use
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

func (e *Engine) GetMarketLiquidityFeeAccount(market, asset string) (*types.Account, error) {
	liquidityAccID := e.accountID(market, systemOwner, asset, types.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY)
	return e.GetAccountByID(liquidityAccID)
}

func (e *Engine) GetMarketInsurancePoolAccount(market, asset string) (*types.Account, error) {
	insuranceAccID := e.accountID(market, systemOwner, asset, types.AccountType_ACCOUNT_TYPE_INSURANCE)
	return e.GetAccountByID(insuranceAccID)
}

// TopUpInsurancePool - this is used only for test purposed for now
// and ease out removing the insurance pool balance from configuration, this should
// definitely never be used in real code.
func (e *Engine) TopUpInsurancePool(market, asset string, amount *num.Uint) error {
	acc, err := e.GetAccountByID(e.accountID(market, "", asset, types.AccountType_ACCOUNT_TYPE_INSURANCE))
	if err != nil {
		return err
	}

	return e.IncrementBalance(context.Background(), acc.Id, amount)
}
