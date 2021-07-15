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
	// ErrPartyAccountsMissing signals that the accounts for this party do not exists
	ErrPartyAccountsMissing = errors.New("party accounts missing, cannot collect")
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
func New(log *logging.Logger, conf Config, broker Broker, now time.Time) *Engine {
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
	}
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
		return e.hashableAccs[i].ID >= id
	})

	copy(e.hashableAccs[i:], e.hashableAccs[i+1:])
	e.hashableAccs = e.hashableAccs[:len(e.hashableAccs)-1]
}

func (e *Engine) addAccountToHashableSlice(acc *types.Account) {
	// sell side levels should be ordered in ascending
	i := sort.Search(len(e.hashableAccs), func(i int) bool {
		return e.hashableAccs[i].ID >= acc.ID
	})

	if i < len(e.hashableAccs) && e.hashableAccs[i].ID == acc.ID {
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
	if e.AssetExists(asset.ID) {
		return ErrAssetAlreadyEnabled
	}
	e.enabledAssets[asset.ID] = asset
	e.broker.Send(events.NewAssetEvent(ctx, asset))
	// then creat a new infrastructure fee account for the asset
	// these are fee related account only
	infraFeeID := e.accountID("", "", asset.ID, types.AccountTypeFeesInfrastructure)
	_, ok := e.accs[infraFeeID]
	if !ok {
		infraFeeAcc := &types.Account{
			ID:       infraFeeID,
			Asset:    asset.ID,
			Owner:    systemOwner,
			Balance:  num.Zero(),
			MarketID: noMarket,
			Type:     types.AccountTypeFeesInfrastructure,
		}
		e.accs[infraFeeID] = infraFeeAcc
		e.addAccountToHashableSlice(infraFeeAcc)
		e.broker.Send(events.NewAccountEvent(ctx, *infraFeeAcc))
	}
	externalID := e.accountID(noMarket, systemOwner, asset.ID, types.AccountTypeExternal)
	if _, ok := e.accs[externalID]; !ok {
		externalAcc := &types.Account{
			ID:       externalID,
			Asset:    asset.ID,
			Owner:    systemOwner,
			Balance:  num.Zero(),
			MarketID: noMarket,
			Type:     types.AccountTypeExternal,
		}
		e.accs[externalID] = externalAcc
		// e.addAccountToHashableSlice(externalAcc)
	}

	// when an asset is enabled a global insurance account is created for it
	globalInsuranceID := e.accountID(noMarket, systemOwner, asset.ID, types.AccountTypeGlobalInsurance)
	if _, ok := e.accs[globalInsuranceID]; !ok {
		insuranceAcc := &types.Account{
			ID:       globalInsuranceID,
			Asset:    asset.ID,
			Owner:    systemOwner,
			Balance:  num.Zero(),
			MarketID: noMarket,
			Type:     types.AccountTypeGlobalInsurance,
		}
		e.accs[globalInsuranceID] = insuranceAcc
		e.addAccountToHashableSlice(insuranceAcc)
		e.broker.Send(events.NewAccountEvent(ctx, *insuranceAcc))
	}

	e.log.Info("new asset added successfully",
		logging.String("asset-id", asset.ID),
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

	insID := e.accountID(marketID, systemOwner, asset, types.AccountTypeInsurance)
	setID := e.accountID(marketID, systemOwner, asset, types.AccountTypeSettlement)

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
	// check quickly that all parties have enough monies in their accounts
	// this may be done only in case of continuous trading
	for party, amount := range ft.TotalFeesAmountPerParty() {
		generalAcc, err := e.GetAccountByID(e.accountID(noMarket, party, assetID, types.AccountTypeGeneral))
		if err != nil {
			e.log.Error("unable to get party account",
				logging.String("account-type", "general"),
				logging.String("party-id", party),
				logging.String("asset", assetID))
			return nil, ErrAccountDoesNotExist
		}

		marginAcc, err := e.GetAccountByID(e.accountID(marketID, party, assetID, types.AccountTypeMargin))
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
			if err := e.IncrementBalance(ctx, bal.Account.ID, bal.Balance); err != nil {
				e.log.Error("Could not update the target account in transfer",
					logging.String("account-id", bal.Account.ID),
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
	makerID := e.accountID(marketID, systemOwner, asset, types.AccountTypeFeesMaker)
	infraID := e.accountID(noMarket, systemOwner, asset, types.AccountTypeFeesInfrastructure)
	liquiID := e.accountID(marketID, systemOwner, asset, types.AccountTypeFeesLiquidity)

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

	var (
		winidx               int
		expectCollected      num.Decimal
		expCollected         = num.Zero()
		totalAmountCollected = num.Zero()
	)

	brokerEvts := make([]events.Event, 0, len(transfers))

	settle, insurance, err := e.getSystemAccounts(marketID, asset)
	if err != nil {
		e.log.Error(
			"Failed to get system accounts required for final settlement",
			logging.Error(err),
		)
		return nil, err
	}

	// process loses first
	for i, transfer := range transfers {
		if transfer == nil {
			continue
		}
		if transfer.Type == types.TransferType_TRANSFER_TYPE_WIN {
			// we processed all losses break then
			winidx = i
			break
		}

		req, err := e.getTransferRequest(ctx, transfer, settle, insurance, &marginUpdate{})
		if err != nil {
			e.log.Error(
				"Failed to build transfer request for event",
				logging.Error(err),
			)
			return nil, err
		}

		// accumulate the expected transfer size
		expCollected.AddSum(req.Amount)
		expectCollected = expectCollected.Add(num.DecimalFromUint(req.Amount))
		// doing a copy of the amount here, as the request is send to getLedgerEntries, which actually
		// modifies it
		requestAmount := req.Amount.Clone()

		// set the amount (this can change the req.Amount value if we entered loss socialisation
		res, err := e.getLedgerEntries(ctx, req)
		if err != nil {
			e.log.Error(
				"Failed to transfer funds",
				logging.Error(err),
			)
			return nil, err
		}
		amountCollected := num.Zero()

		for _, bal := range res.Balances {
			amountCollected.AddSum(bal.Balance)
			if err := e.UpdateBalance(ctx, bal.Account.Id, bal.Balance); err != nil {
				e.log.Error(
					"Could not update the target account in transfer",
					logging.String("account-id", bal.Account.Id),
					logging.Error(err),
				)
				return nil, err
			}
		}
		totalAmountCollected.AddSum(amountCollected)
		responses = append(responses, res)

		// Update to see how much we still need
		requestAmount = requestAmount.Sub(requestAmount, amountCollected)
		if transfer.Owner != types.NetworkParty {
			// no error possible here, we're just reloading the accounts to ensure the correct balance
			general, margin, _ := e.getMTMPartyAccounts(transfer.Owner, marketID, asset)
			if totalInAccount := num.Sum(general.Balance, margin.Balance); totalInAccount.LT(requestAmount) {
				delta := req.Amount.Sub(requestAmount, totalInAccount)
				e.log.Warn("loss socialization missing amount to be collected or used from insurance pool",
					logging.String("party-id", transfer.Owner),
					logging.BigUint("amount", delta),
					logging.String("market-id", settle.MarketId))

				brokerEvts = append(brokerEvts,
					events.NewLossSocializationEvent(ctx, transfer.Owner, marketID, delta, false, e.currentTime))
			}
		}
	}

	if len(brokerEvts) > 0 {
		e.broker.SendBatch(brokerEvts)
	}

	// if winidx is 0, this means we had now win and loss, but may have some event which
	// needs to be propagated forward so we return now.
	if winidx == 0 {
		if !settle.Balance.IsZero() {
			return nil, ErrSettlementBalanceNotZero
		}
		return responses, nil
	}

	// now compare what's in the settlement account what we expect initially to redistribute.
	// if there's not enough we enter loss socialization
	distr := simpleDistributor{
		log:             e.log,
		marketID:        marketID,
		expectCollected: expCollected,
		collected:       totalAmountCollected,
		requests:        make([]request, 0, len(transfers)-winidx),
		ts:              e.currentTime,
	}

	if distr.LossSocializationEnabled() {
		e.log.Warn("Entering loss socialization on final settlement",
			logging.String("market-id", marketID),
			logging.String("asset", asset),
			logging.BigUint("expect-collected", expCollected),
			logging.BigUint("collected", settle.Balance))
		for _, transfer := range transfers[winidx:] {
			if transfer != nil && transfer.Type == types.TransferType_TRANSFER_TYPE_WIN {
				distr.Add(transfer)
			}
		}
		if evts := distr.Run(ctx); len(evts) != 0 {
			e.broker.SendBatch(evts)
		}
	}

	// then we process all the wins
	for _, transfer := range transfers[winidx:] {
		if transfer == nil {
			continue
		}

		req, err := e.getTransferRequest(ctx, transfer, settle, insurance, &marginUpdate{})
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

		// update the to accounts now
		for _, bal := range res.Balances {
			if err := e.UpdateBalance(ctx, bal.Account.ID, bal.Balance); err != nil {
				e.log.Error(
					"Could not update the target account in transfer",
					logging.String("account-id", bal.Account.ID),
					logging.Error(err),
				)
				return nil, err
			}
		}
		responses = append(responses, res)
	}

	if !settle.Balance.IsZero() {
		return nil, ErrSettlementBalanceNotZero
	}

	return responses, nil
}

func (e *Engine) getMTMPartyAccounts(party, marketID, asset string) (gen, margin *types.Account, err error) {
	// no need to look any further
	if party == types.NetworkParty {
		return nil, nil, nil
	}
	gen, err = e.GetAccountByID(e.accountID(noMarket, party, asset, types.AccountTypeGeneral))
	if err != nil {
		return nil, nil, err
	}
	margin, err = e.GetAccountByID(e.accountID(marketID, party, asset, types.AccountTypeMargin))
	return
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
		expCollected    = num.Zero()
	)

	// create batch of events
	brokerEvts := make([]events.Event, 0, len(transfers))
	// iterate over transfer until we get the first win, so we need we accumulated all loss
	for i, evt := range transfers {
		party := evt.Party()
		transfer := evt.Transfer()
		marginEvt := &marginUpdate{
			MarketPosition: evt,
			asset:          asset,
			marketID:       settle.MarketID,
		}

		if party != types.NetworkParty {
			// get the state of the accounts before processing transfers
			// so they can be used in the marginEvt, and to calculate the missing funds
			marginEvt.general, marginEvt.margin, err = e.getMTMPartyAccounts(party, settle.MarketID, asset)
			if err != nil {
				e.log.Error("unable to get party account",
					logging.String("party-id", party),
					logging.String("asset", asset),
					logging.String("market-id", settle.MarketID))
			}
		}

		// no transfer needed if transfer is nil, just build the marginUpdate
		if transfer == nil {
			// no error when getting MTM accounts, and no margin account == network position
			// we are not interested in this event, continue here
			if party != types.NetworkParty {
				marginEvts = append(marginEvts, marginEvt)
			}
			continue
		}

		if transfer.Type == types.TransferTypeMTMWin {
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
		// doing a copy of the amount here, as the request is send to getLedgerEntries, which actually
		// modifies it
		requestAmount := req.Amount.Clone()

		// set the amount (this can change the req.Amount value if we entered loss socialisation
		res, err := e.getLedgerEntries(ctx, req)
		if err != nil {
			e.log.Error(
				"Failed to transfer funds",
				logging.Error(err),
			)
			return nil, nil, err
		}

		amountCollected := num.Zero()
		// // update the to accounts now
		for _, bal := range res.Balances {
			amountCollected.AddSum(bal.Balance)
			if err := e.IncrementBalance(ctx, bal.Account.ID, bal.Balance); err != nil {
				e.log.Error(
					"Could not update the target account in transfer",
					logging.String("account-id", bal.Account.ID),
					logging.Error(err),
				)
				return nil, nil, err
			}
		}
		responses = append(responses, res)

		// Update to see how much we still need
		requestAmount = requestAmount.Sub(requestAmount, amountCollected)

		// here we check if we were able to collect all monies,
		// if not send an event to notify the plugins
		if party != types.NetworkParty {
			// no error possible here, we're just reloading the accounts to ensure the correct balance
			marginEvt.general, marginEvt.margin, _ = e.getMTMPartyAccounts(party, settle.MarketID, asset)
			if totalInAccount := num.Sum(marginEvt.general.Balance, marginEvt.margin.Balance); totalInAccount.LT(requestAmount) {
				delta := req.Amount.Sub(requestAmount, totalInAccount)
				e.log.Warn("loss socialization missing amount to be collected or used from insurance pool",
					logging.String("party-id", party),
					logging.BigUint("amount", delta),
					logging.String("market-id", settle.MarketID))

				brokerEvts = append(brokerEvts,
					events.NewLossSocializationEvent(ctx, party, settle.MarketID, delta, false, e.currentTime))
			}
			marginEvts = append(marginEvts, marginEvt)
		} else {
			// we've used the insurance account as a margin account for the network
			// we have to update it to ensure we're aware of its balance
			settle, insurance, _ = e.getSystemAccounts(marketID, asset)
		}
	}

	if len(brokerEvts) > 0 {
		e.broker.SendBatch(brokerEvts)
	}
	// we couldn't have reached this point without settlement account
	// it's also prudent to reset the insurance account... the network position
	// relies on its balance being accurate
	settle, insurance, _ = e.getSystemAccounts(marketID, asset)
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
		marketID:        settle.MarketID,
		expectCollected: expCollected,
		collected:       settle.Balance,
		requests:        make([]request, 0, len(transfers)-winidx),
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
			if transfer != nil && transfer.Type == types.TransferTypeMTMWin {
				distr.Add(evt.Transfer())
			}
		}
		if evts := distr.Run(ctx); len(evts) != 0 {
			e.broker.SendBatch(evts)
		}
	}

	// then we process all the wins
	for _, evt := range transfers[winidx:] {
		transfer := evt.Transfer()
		party := evt.Party()
		marginEvt := &marginUpdate{
			MarketPosition: evt,
			asset:          asset,
			marketID:       settle.MarketID,
		}
		// no transfer needed if transfer is nil, just build the marginUpdate
		if party != types.NetworkParty {
			marginEvt.general, marginEvt.margin, err = e.getMTMPartyAccounts(party, settle.MarketID, asset)
			if err != nil {
				e.log.Error("unable to get party account",
					logging.String("account-type", "margin"),
					logging.String("party-id", evt.Party()),
					logging.String("asset", asset),
					logging.String("market-id", settle.MarketID))
			}

			if transfer == nil {
				marginEvts = append(marginEvts, marginEvt)
				continue
			}
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
			if err := e.IncrementBalance(ctx, bal.Account.ID, bal.Balance); err != nil {
				e.log.Error(
					"Could not update the target account in transfer",
					logging.String("account-id", bal.Account.ID),
					logging.Error(err),
				)
				return nil, nil, err
			}
		}
		responses = append(responses, res)
		if party == types.NetworkParty {
			continue
		}
		// updating the accounts stored in the marginEvt
		// this can't return an error
		marginEvt.general, marginEvt.margin, _ = e.getMTMPartyAccounts(party, settle.MarketID, asset)

		marginEvts = append(marginEvts, marginEvt)
	}

	if !settle.Balance.IsZero() {
		return nil, nil, ErrSettlementBalanceNotZero
	}
	return marginEvts, responses, nil
}

// GetPartyMargin will return the current margin for a given party
func (e *Engine) GetPartyMargin(pos events.MarketPosition, asset, marketID string) (events.Margin, error) {
	genID := e.accountID("", pos.Party(), asset, types.AccountTypeGeneral)
	marginID := e.accountID(marketID, pos.Party(), asset, types.AccountTypeMargin)
	bondID := e.accountID(marketID, pos.Party(), asset, types.AccountTypeBond)
	genAcc, err := e.GetAccountByID(genID)
	if err != nil {
		e.log.Error(
			"Party doesn't have a general account somehow?",
			logging.String("party-id", pos.Party()))
		return nil, ErrPartyAccountsMissing
	}
	marAcc, err := e.GetAccountByID(marginID)
	if err != nil {
		e.log.Error(
			"Party doesn't have a margin account somehow?",
			logging.String("party-id", pos.Party()),
			logging.String("market-id", marketID))
		return nil, ErrPartyAccountsMissing
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
		marginShortFall: num.Zero(),
	}, nil
}

// MarginUpdate will run the margin updates over a set of risk events (margin updates)
func (e *Engine) MarginUpdate(ctx context.Context, marketID string, updates []events.Risk) ([]*types.TransferResponse, []events.Margin, []events.Margin, error) {
	response := make([]*types.TransferResponse, 0, len(updates))
	var (
		closed     = make([]events.Margin, 0, len(updates)/2) // half the cap, if we have more than that, the slice will double once, and will fit all updates anyway
		toPenalise = []events.Margin{}
		settle     = &types.Account{
			MarketID: marketID,
		}
	)
	// create "fake" settle account for market ID
	for _, update := range updates {
		transfer := update.Transfer()
		// although this is mainly a duplicate event, we need to pass it to getTransferRequest
		mevt := &marginUpdate{
			MarketPosition:  update,
			asset:           update.Asset(),
			marketID:        update.MarketID(),
			marginShortFall: num.Zero(),
		}

		req, err := e.getTransferRequest(ctx, transfer, settle, nil, mevt)
		if err != nil {
			return nil, nil, nil, err
		}

		// calculate the marginShortFall in case of a liquidityProvider
		if mevt.bond != nil && transfer.Amount.Amount.GT(mevt.general.Balance) {
			mevt.marginShortFall.Sub(transfer.Amount.Amount, mevt.general.Balance)
		}

		res, err := e.getLedgerEntries(ctx, req)
		if err != nil {
			return nil, nil, nil, err
		}
		// we didn't manage to top up to even the minimum required system margin, close out party
		// we need to be careful with this, only apply this to transfer for low margin
		// the MinAmount in the transfer is always set to 0 but in 2 case:
		// - first when a new order is created, the MinAmount is the same than Amount, which is
		//   what's required to reach the InitialMargin level
		// - second when a party margin is under the MaintenanceLevel, the MinAmount is supposed
		//   to be at least to get back to the search level, and the amount will be enough to reach
		//   InitialMargin
		// In both case either the order will not be accepted, or the party will be closed
		if transfer.Type == types.TransferTypeMarginLow &&
			res.Balances[0].Account.Balance.LT(num.Sum(update.MarginBalance(), transfer.MinAmount)) {
			closed = append(closed, mevt)
		} else if !mevt.marginShortFall.IsZero() {
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
	margin, err := e.GetAccountByID(e.accountID(marketID, transfer.Owner, assetID, types.AccountTypeMargin))
	if err != nil {
		e.log.Error(
			"Failed to get the margin party account",
			logging.String("owner-id", transfer.Owner),
			logging.String("market-id", marketID),
			logging.Error(err),
		)
		return nil, err
	}
	// we'll need this account for all transfer types anyway (settlements, margin-risk updates)
	general, err := e.GetAccountByID(e.accountID(noMarket, transfer.Owner, assetID, types.AccountTypeGeneral))
	if err != nil {
		e.log.Error(
			"Failed to get the general party account",
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
		MinAmount: num.Zero(),
		Asset:     assetID,
		Reference: transfer.Type.String(),
	}
	// @TODO we should be able to clone the min amount regardless
	if transfer.MinAmount != nil {
		req.MinAmount.Set(transfer.MinAmount)
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
func (e *Engine) BondUpdate(ctx context.Context, market string, transfer *types.Transfer) (*types.TransferResponse, error) {
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
		MarketID: marketID,
	}
	transfer := update.Transfer()
	// although this is mainly a duplicate event, we need to pass it to getTransferRequest
	mevt := marginUpdate{
		MarketPosition:  update,
		asset:           update.Asset(),
		marketID:        update.MarketID(),
		marginShortFall: num.Zero(),
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
		mevt.marginShortFall.Sub(transfer.Amount.Amount, mevt.general.Balance)
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

	if !mevt.marginShortFall.IsZero() {
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

	// the accounts for the party we need

	// we do not load the margin all the time
	// as do not always need it.
	getMargin := func() (*types.Account, error) {
		margin, err = e.GetAccountByID(e.accountID(marketID, t.Owner, assetID, types.AccountTypeMargin))
		if err != nil {
			e.log.Error(
				"Failed to get the margin party account",
				logging.String("owner-id", t.Owner),
				logging.String("market-id", marketID),
				logging.Error(err),
			)
			return nil, err
		}
		return margin, err
	}
	general, err = e.GetAccountByID(e.accountID(noMarket, t.Owner, assetID, types.AccountTypeGeneral))
	if err != nil {
		e.log.Error(
			"Failed to get the general party account",
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
	case types.TransferTypeInfrastructureFeePay:
		margin, err := getMargin()
		if err != nil {
			return nil, err
		}
		treq.FromAccount = []*types.Account{general, margin}
		treq.ToAccount = []*types.Account{infraFee}
		return treq, nil
	case types.TransferTypeInfrastructureFeeDistribute:
		treq.FromAccount = []*types.Account{infraFee}
		treq.ToAccount = []*types.Account{general}
		return treq, nil
	case types.TransferTypeLiquidityFeePay:
		margin, err := getMargin()
		if err != nil {
			return nil, err
		}
		treq.FromAccount = []*types.Account{general, margin}
		treq.ToAccount = []*types.Account{liquiFee}
		return treq, nil
	case types.TransferTypeLiquidityFeeDistribute:
		margin, err := getMargin()
		if err != nil {
			return nil, err
		}
		treq.FromAccount = []*types.Account{liquiFee}
		treq.ToAccount = []*types.Account{margin}
		return treq, nil
	case types.TransferTypeMakerFeePay:
		margin, err := getMargin()
		if err != nil {
			return nil, err
		}
		treq.FromAccount = []*types.Account{general, margin}
		treq.ToAccount = []*types.Account{makerFee}
		return treq, nil
	case types.TransferTypeMakerFeeReceive:
		treq.FromAccount = []*types.Account{makerFee}
		treq.ToAccount = []*types.Account{general}
		return treq, nil
	default:
		return nil, ErrInvalidTransferTypeForFeeRequest
	}
}

func (e *Engine) getBondTransferRequest(t *types.Transfer, market string) (*types.TransferRequest, error) {
	bond, err := e.GetAccountByID(e.accountID(market, t.Owner, t.Amount.Asset, types.AccountTypeBond))
	if err != nil {
		e.log.Error(
			"Failed to get the margin party account",
			logging.String("owner-id", t.Owner),
			logging.String("market-id", market),
			logging.Error(err),
		)
		return nil, err
	}

	// we'll need this account for all transfer types anyway (settlements, margin-risk updates)
	general, err := e.GetAccountByID(e.accountID(noMarket, t.Owner, t.Amount.Asset, types.AccountTypeGeneral))
	if err != nil {
		e.log.Error(
			"Failed to get the general party account",
			logging.String("owner-id", t.Owner),
			logging.String("market-id", market),
			logging.Error(err),
		)
		return nil, err
	}

	// we'll need this account for all transfer types anyway (settlements, margin-risk updates)
	insurancePool, err := e.GetAccountByID(e.accountID(market, systemOwner, t.Amount.Asset, types.AccountTypeInsurance))
	if err != nil {
		e.log.Error(
			"Failed to get the general party account",
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
	case types.TransferTypeBondLow:
		// do we have enough in the general account to make the transfer?
		if !t.Amount.Amount.IsZero() && general.Balance.LT(t.Amount.Amount) {
			return nil, errors.New("not enough collateral in general account")
		}
		treq.FromAccount = []*types.Account{general}
		treq.ToAccount = []*types.Account{bond}
		return treq, nil
	case types.TransferTypeBondHigh:
		treq.FromAccount = []*types.Account{bond}
		treq.ToAccount = []*types.Account{general}
		return treq, nil
	case types.TransferTypeBondSlashing:
		treq.FromAccount = []*types.Account{bond}
		// it's possible the bond account is insufficient, and falling back to margin balance
		// won't cause a close-out
		if marginAcc, err := e.GetAccountByID(e.accountID(market, t.Owner, t.Amount.Asset, types.AccountTypeMargin)); err == nil {
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
	if p.Type == types.TransferTypeMTMLoss ||
		p.Type == types.TransferTypeWin ||
		p.Type == types.TransferTypeMarginLow {
		// we do not care about errors here as the bond account is not mandatory for the transfers
		// a partry would have a bond account only if it was also a market maker
		mEvt.bond, _ = e.GetAccountByID(e.accountID(settle.MarketID, p.Owner, asset, types.AccountTypeBond))
	}

	if settle != nil && mEvt.margin == nil && p.Owner != types.NetworkParty {
		// the accounts for the party we need
		// the accounts for the trader we need
		mEvt.margin, err = e.GetAccountByID(e.accountID(settle.MarketID, p.Owner, asset, types.AccountTypeMargin))
		if err != nil {
			e.log.Error(
				"Failed to get the party margin account",
				logging.String("owner-id", p.Owner),
				logging.String("market-id", settle.MarketID),
				logging.Error(err),
			)
			return nil, err
		}
	}
	// we'll need this account for all transfer types anyway (settlements, margin-risk updates)
	if mEvt.general == nil && p.Owner != types.NetworkParty {
		mEvt.general, err = e.GetAccountByID(e.accountID(noMarket, p.Owner, asset, types.AccountTypeGeneral))
		if err != nil {
			e.log.Error(
				"Failed to get the party general account",
				logging.String("owner-id", p.Owner),
				logging.String("market-id", settle.MarketID),
				logging.Error(err),
			)
			return nil, err
		}
	}
	if p.Type == types.TransferTypeWithdraw || p.Type == types.TransferTypeDeposit {
		// external account:
		eacc, _ = e.GetAccountByID(e.accountID(noMarket, systemOwner, asset, types.AccountTypeExternal))
	}

	switch p.Type {
	// final settle, or MTM settle, makes no difference, it's win/loss still
	case types.TransferTypeLoss, types.TransferTypeMTMLoss:
		req.ToAccount = []*types.Account{
			settle,
		}
		req.Amount = p.Amount.Amount.Clone()
		req.MinAmount = num.Zero() // default value, but keep it here explicitly
		// losses are collected first from the margin account, then the general account, and finally
		// taken out of the insurance pool. Network party will only have insurance pool available
		if mEvt.bond != nil {
			// network party will never have a bond account, so we know what to do
			req.FromAccount = []*types.Account{
				mEvt.margin,
				mEvt.general,
				mEvt.bond,
				insurance,
			}
		} else if p.Owner == types.NetworkParty {
			req.FromAccount = []*types.Account{
				insurance,
			}
		} else {
			// regular party, no bond account:
			req.FromAccount = []*types.Account{
				mEvt.margin,
				mEvt.general,
				insurance,
			}
		}
	case types.TransferTypeWin, types.TransferTypeMTMWin:
		req.Amount = p.Amount.Amount.Clone()
		req.MinAmount = num.Zero() // default value, but keep it here explicitly
		// the insurance pool in the Req.FromAccountAccount is not used ATM (losses should fully cover wins
		// or the insurance pool has already been drained).
		if p.Owner == types.NetworkParty {
			req.FromAccount = []*types.Account{
				settle,
			}
			req.ToAccount = []*types.Account{
				insurance,
			}
		} else {
			req.FromAccount = []*types.Account{
				settle,
				insurance,
			}
			req.ToAccount = []*types.Account{
				mEvt.margin,
			}
		}
	case types.TransferTypeMarginLow:
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
		req.Amount = p.Amount.Amount.Clone()
		req.MinAmount = p.MinAmount.Clone()
	case types.TransferTypeMarginHigh:
		req.FromAccount = []*types.Account{
			mEvt.margin,
		}
		req.ToAccount = []*types.Account{
			mEvt.general,
		}
		req.Amount = p.Amount.Amount.Clone()
		req.MinAmount = p.MinAmount.Clone()
	case types.TransferTypeWithdrawLock:
		req.FromAccount = []*types.Account{
			mEvt.general,
		}
		req.ToAccount = []*types.Account{
			mEvt.lock,
		}
		req.Amount = p.Amount.Amount.Clone()
		req.MinAmount = p.Amount.Amount.Clone()
	case types.TransferTypeDeposit:
		// ensure we have the funds req.ToAccount deposit
		eacc.Balance = eacc.Balance.Add(eacc.Balance, p.Amount.Amount)
		req.FromAccount = []*types.Account{
			eacc,
		}
		req.ToAccount = []*types.Account{
			mEvt.general,
		}
		req.Amount = p.Amount.Amount.Clone()
		req.MinAmount = p.Amount.Amount.Clone()
	case types.TransferTypeWithdraw:
		req.FromAccount = []*types.Account{
			mEvt.lock,
		}
		req.ToAccount = []*types.Account{
			eacc,
		}
		req.Amount = p.Amount.Amount.Clone()
		req.MinAmount = p.Amount.Amount.Clone()
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
			Balance: num.Zero(),
		})
	}
	amount := req.Amount
	for _, acc := range req.FromAccount {
		// give each to account an equal share
		nToAccounts := num.NewUint(uint64(len(req.ToAccount)))
		parts := num.Zero().Div(amount, nToAccounts)
		// add remaining pennies to last ledger movement
		remainder := num.Zero().Mod(amount, nToAccounts)
		var (
			to *types.TransferBalance
			lm *types.LedgerEntry
		)
		// either the account contains enough, or we're having to access insurance pool money
		if acc.Balance.GTE(amount) {
			acc.Balance.Sub(acc.Balance, amount)
			if err := e.UpdateBalance(ctx, acc.ID, acc.Balance); err != nil {
				e.log.Error(
					"Failed to update balance for account",
					logging.String("account-id", acc.ID),
					logging.BigUint("balance", acc.Balance),
					logging.Error(err),
				)
				return nil, err
			}
			for _, to = range ret.Balances {
				lm = &types.LedgerEntry{
					FromAccount: acc.ID,
					ToAccount:   to.Account.ID,
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
			if err := e.UpdateBalance(ctx, acc.ID, acc.Balance); err != nil {
				e.log.Error(
					"Failed to set balance of account to 0",
					logging.String("account-id", acc.ID),
					logging.Error(err),
				)
				return nil, err
			}
			for _, to = range ret.Balances {
				lm = &types.LedgerEntry{
					FromAccount: acc.ID,
					ToAccount:   to.Account.ID,
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
	e.removeAccount(req.FromAccount[0].ID)
	// remove account from balances tracking
	e.rmPartyAccount(party, req.FromAccount[0].ID)

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

		generalAcc, err := e.GetAccountByID(e.accountID("", v, asset, types.AccountTypeGeneral))
		if err != nil {
			e.log.Debug(
				"Failed to get the general account",
				logging.String("party-id", v),
				logging.String("market-id", mktID),
				logging.String("asset", asset),
				logging.Error(err))
			// just try to do other parties
			continue
		}

		// we start first with the margin account if it exists
		marginAcc, err := e.GetAccountByID(e.accountID(mktID, v, asset, types.AccountTypeMargin))
		if err != nil {
			e.log.Debug(
				"Failed to get the margin account",
				logging.String("party-id", v),
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
		bondAcc, err := e.GetAccountByID(e.accountID(mktID, v, asset, types.AccountTypeBond))
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

	// redistribute the remaining funds in the market insurance account between other markets insurance accounts and global insurance account
	marketInsuranceID := e.accountID(mktID, "", asset, types.AccountTypeInsurance)
	marketInsuranceAcc, ok := e.accs[marketInsuranceID]
	if !ok || marketInsuranceAcc.Balance.EQ(num.Zero()) {
		// if there's no market insurance account or it has no balance, nothing to do here
		return resps, nil
	}

	// get all other market insurance accounts for the same asset
	var insuranceAccounts []*types.Account
	for _, acc := range e.accs {
		if acc.ID != marketInsuranceID && acc.Asset == asset && acc.Type == types.AccountTypeInsurance {
			insuranceAccounts = append(insuranceAccounts, acc.Clone())
		}
	}

	// add the global account
	insuranceAccounts = append(insuranceAccounts, e.GetAssetInsurancePoolAccount(asset))

	// redistribute market insurance funds between the global and other markets equally
	req.FromAccount[0] = marketInsuranceAcc
	req.ToAccount = insuranceAccounts
	req.Amount = marketInsuranceAcc.Balance.Clone()
	insuranceledgerEntries, err := e.getLedgerEntries(ctx, req)
	if err != nil {
		e.log.Panic("unable to redistribute market insurance funds", logging.Error(err))
	}
	for _, acc := range insuranceAccounts {
		e.broker.Send(events.NewAccountEvent(ctx, *acc))
	}

	return append(resps, insuranceledgerEntries), nil
}

func (e *Engine) CanCoverBond(market, party, asset string, amount *num.Uint) bool {
	bondID := e.accountID(
		market, party, asset, types.AccountTypeBond,
	)
	genID := e.accountID(
		noMarket, party, asset, types.AccountTypeGeneral,
	)

	availableBalance := num.Zero()

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
// if no general account exist for the party for the given asset
func (e *Engine) CreatePartyBondAccount(ctx context.Context, partyID, marketID, asset string) (string, error) {
	if !e.AssetExists(asset) {
		return "", ErrInvalidAssetID
	}
	bondID := e.accountID(marketID, partyID, asset, types.AccountTypeBond)
	if _, ok := e.accs[bondID]; !ok {
		// OK no bond ID, so let's try to get the general id then
		// first check if general account exists
		generalID := e.accountID(noMarket, partyID, asset, types.AccountTypeGeneral)
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
			ID:       bondID,
			Asset:    asset,
			MarketID: marketID,
			Balance:  num.Zero(),
			Owner:    partyID,
			Type:     types.AccountTypeBond,
		}
		e.accs[bondID] = &acc
		e.addPartyAccount(partyID, bondID, &acc)
		e.addAccountToHashableSlice(&acc)
		e.broker.Send(events.NewAccountEvent(ctx, acc))
	}
	return bondID, nil
}

// CreatePartyMarginAccount creates a margin account if it does not exist, will return an error
// if no general account exist for the party for the given asset
func (e *Engine) CreatePartyMarginAccount(ctx context.Context, partyID, marketID, asset string) (string, error) {
	if !e.AssetExists(asset) {
		return "", ErrInvalidAssetID
	}
	marginID := e.accountID(marketID, partyID, asset, types.AccountTypeMargin)
	if _, ok := e.accs[marginID]; !ok {
		// OK no margin ID, so let's try to get the general id then
		// first check if general account exists
		generalID := e.accountID(noMarket, partyID, asset, types.AccountTypeGeneral)
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
			ID:       marginID,
			Asset:    asset,
			MarketID: marketID,
			Balance:  num.Zero(),
			Owner:    partyID,
			Type:     types.AccountTypeMargin,
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
	margin := e.accountID(market, party, asset, types.AccountTypeMargin)
	return e.GetAccountByID(margin)
}

// GetPartyGeneralAccount returns a general account given the partyID.
func (e *Engine) GetPartyGeneralAccount(partyID, asset string) (*types.Account, error) {
	generalID := e.accountID(noMarket, partyID, asset, types.AccountTypeGeneral)
	return e.GetAccountByID(generalID)
}

// GetPartyBondAccount returns a general account given the partyID.
func (e *Engine) GetPartyBondAccount(market, partyID, asset string) (*types.Account, error) {
	id := e.accountID(
		market, partyID, asset, types.AccountTypeBond)
	return e.GetAccountByID(id)
}

// CreatePartyGeneralAccount create the general account for a party
func (e *Engine) CreatePartyGeneralAccount(ctx context.Context, partyID, asset string) (string, error) {
	if !e.AssetExists(asset) {
		return "", ErrInvalidAssetID
	}

	generalID := e.accountID(noMarket, partyID, asset, types.AccountTypeGeneral)
	if _, ok := e.accs[generalID]; !ok {
		acc := types.Account{
			ID:       generalID,
			Asset:    asset,
			MarketID: noMarket,
			Balance:  num.Zero(),
			Owner:    partyID,
			Type:     types.AccountTypeGeneral,
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

	id := e.accountID(noMarket, partyID, asset, types.AccountTypeLockWithdraw)
	var (
		acc *types.Account
		ok  bool
	)
	if acc, ok = e.accs[id]; !ok {
		acc = &types.Account{
			ID:       id,
			Asset:    asset,
			MarketID: noMarket,
			Balance:  num.Zero(),
			Owner:    partyID,
			Type:     types.AccountTypeLockWithdraw,
		}
		e.accs[id] = acc
		e.addPartyAccount(partyID, id, acc)
		e.addAccountToHashableSlice(acc)
		e.broker.Send(events.NewAccountEvent(ctx, *acc))
	}

	return acc, nil
}

// RemoveDistressed will remove all distressed party in the event positions
// for a given market and asset
func (e *Engine) RemoveDistressed(ctx context.Context, parties []events.MarketPosition, marketID, asset string) (*types.TransferResponse, error) {
	tl := len(parties)
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
	for _, party := range parties {
		bondAcc, err := e.GetAccountByID(e.accountID(marketID, party.Party(), asset, types.AccountTypeBond))
		if err != nil {
			bondAcc = &types.Account{}
		}
		genAcc, err := e.GetAccountByID(e.accountID(noMarket, party.Party(), asset, types.AccountTypeGeneral))
		if err != nil {
			return nil, err
		}
		marginAcc, err := e.GetAccountByID(e.accountID(marketID, party.Party(), asset, types.AccountTypeMargin))
		if err != nil {
			return nil, err
		}
		// If any balance remains on bond account, move it over to margin account
		if bondAcc.Balance != nil && !bondAcc.Balance.IsZero() {
			resp.Transfers = append(resp.Transfers, &types.LedgerEntry{
				FromAccount: bondAcc.ID,
				ToAccount:   marginAcc.ID,
				Amount:      bondAcc.Balance.Clone(),
				Reference:   types.TransferTypeMarginLow.String(),
				Type:        "position-resolution",
				Timestamp:   e.currentTime,
			})
			if err := e.IncrementBalance(ctx, marginAcc.ID, bondAcc.Balance); err != nil {
				return nil, err
			}
			if err := e.UpdateBalance(ctx, bondAcc.ID, bondAcc.Balance.SetUint64(0)); err != nil {
				return nil, err
			}
		}
		// take whatever is left on the general account, and move to margin balance
		// we can take everything from the account, as whatever amount was left here didn't cover the minimum margin requirement
		if genAcc.Balance != nil && !genAcc.Balance.IsZero() {
			resp.Transfers = append(resp.Transfers, &types.LedgerEntry{
				FromAccount: genAcc.ID,
				ToAccount:   marginAcc.ID,
				Amount:      genAcc.Balance.Clone(),
				Reference:   types.TransferTypeMarginLow.String(),
				Type:        "position-resolution",
				Timestamp:   e.currentTime,
			})
			if err := e.IncrementBalance(ctx, marginAcc.ID, genAcc.Balance); err != nil {
				return nil, err
			}
			if err := e.UpdateBalance(ctx, genAcc.ID, genAcc.Balance.SetUint64(0)); err != nil {
				return nil, err
			}
		}
		// move monies from the margin account (balance is general, bond, and margin combined now)
		if !marginAcc.Balance.IsZero() {
			resp.Transfers = append(resp.Transfers, &types.LedgerEntry{
				FromAccount: marginAcc.ID,
				ToAccount:   ins.ID,
				Amount:      marginAcc.Balance.Clone(),
				Reference:   types.TransferTypeMarginConfiscated.String(),
				Type:        "position-resolution",
				Timestamp:   e.currentTime,
			})
			if err := e.IncrementBalance(ctx, ins.ID, marginAcc.Balance); err != nil {
				return nil, err
			}
			if err := e.UpdateBalance(ctx, marginAcc.ID, marginAcc.Balance.SetUint64(0)); err != nil {
				return nil, err
			}
		}

		// we remove the margin account
		e.removeAccount(marginAcc.ID)
		// remove account from balances tracking
		e.rmPartyAccount(party.Party(), marginAcc.ID)

	}
	return &resp, nil
}

func (e *Engine) ClearPartyMarginAccount(ctx context.Context, party, market, asset string) (*types.TransferResponse, error) {
	acc, err := e.GetAccountByID(e.accountID(market, party, asset, types.AccountTypeMargin))
	if err != nil {
		return nil, err
	}
	resp := types.TransferResponse{
		Transfers: []*types.LedgerEntry{},
	}

	if !acc.Balance.IsZero() {
		genAcc, err := e.GetAccountByID(e.accountID(noMarket, party, asset, types.AccountTypeGeneral))
		if err != nil {
			return nil, err
		}

		resp.Transfers = append(resp.Transfers, &types.LedgerEntry{
			FromAccount: acc.ID,
			ToAccount:   genAcc.ID,
			Amount:      acc.Balance.Clone(),
			Reference:   types.TransferTypeMarginHigh.String(),
			Type:        types.TransferTypeMarginHigh.String(),
			Timestamp:   e.currentTime,
		})
		if err := e.IncrementBalance(ctx, genAcc.ID, acc.Balance); err != nil {
			return nil, err
		}
		if err := e.UpdateBalance(ctx, acc.ID, acc.Balance.SetUint64(0)); err != nil {
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
	insuranceID = e.accountID(marketID, "", asset, types.AccountTypeInsurance)
	_, ok := e.accs[insuranceID]
	if !ok {
		insAcc := &types.Account{
			ID:       insuranceID,
			Asset:    asset,
			Owner:    systemOwner,
			Balance:  num.Zero(),
			MarketID: marketID,
			Type:     types.AccountTypeInsurance,
		}
		e.accs[insuranceID] = insAcc
		e.addAccountToHashableSlice(insAcc)
		e.broker.Send(events.NewAccountEvent(ctx, *insAcc))

	}
	settleID = e.accountID(marketID, "", asset, types.AccountTypeSettlement)
	_, ok = e.accs[settleID]
	if !ok {
		setAcc := &types.Account{
			ID:       settleID,
			Asset:    asset,
			Owner:    systemOwner,
			Balance:  num.Zero(),
			MarketID: marketID,
			Type:     types.AccountTypeSettlement,
		}
		e.accs[settleID] = setAcc
		e.addAccountToHashableSlice(setAcc)
		e.broker.Send(events.NewAccountEvent(ctx, *setAcc))
	}

	// these are fee related account only
	liquidityFeeID := e.accountID(marketID, "", asset, types.AccountTypeFeesLiquidity)
	_, ok = e.accs[liquidityFeeID]
	if !ok {
		liquidityFeeAcc := &types.Account{
			ID:       liquidityFeeID,
			Asset:    asset,
			Owner:    systemOwner,
			Balance:  num.Zero(),
			MarketID: marketID,
			Type:     types.AccountTypeFeesLiquidity,
		}
		e.accs[liquidityFeeID] = liquidityFeeAcc
		e.addAccountToHashableSlice(liquidityFeeAcc)
		e.broker.Send(events.NewAccountEvent(ctx, *liquidityFeeAcc))
	}
	makerFeeID := e.accountID(marketID, "", asset, types.AccountTypeFeesMaker)
	_, ok = e.accs[makerFeeID]
	if !ok {
		makerFeeAcc := &types.Account{
			ID:       makerFeeID,
			Asset:    asset,
			Owner:    systemOwner,
			Balance:  num.Zero(),
			MarketID: marketID,
			Type:     types.AccountTypeFeesMaker,
		}
		e.accs[makerFeeID] = makerFeeAcc
		e.addAccountToHashableSlice(makerFeeAcc)
		e.broker.Send(events.NewAccountEvent(ctx, *makerFeeAcc))
	}

	return
}

func (e *Engine) HasGeneralAccount(party, asset string) bool {
	_, err := e.GetAccountByID(
		e.accountID("", party, asset, types.AccountTypeGeneral))
	return err == nil
}

// LockFundsForWithdraw will lock funds in a separate account to be withdrawn later on by the party
func (e *Engine) LockFundsForWithdraw(ctx context.Context, partyID, asset string, amount *num.Uint) (*types.TransferResponse, error) {
	if !e.AssetExists(asset) {
		return nil, ErrInvalidAssetID
	}
	genacc, err := e.GetAccountByID(e.accountID("", partyID, asset, types.AccountTypeGeneral))
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
		Type:      types.TransferTypeWithdrawLock,
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
		if err := e.UpdateBalance(ctx, bal.Account.ID, bal.Account.Balance); err != nil {
			return nil, err
		}
	}
	return res, nil
}

// Withdraw will remove the specified amount from the party
// general account
func (e *Engine) Withdraw(ctx context.Context, partyID, asset string, amount *num.Uint) (*types.TransferResponse, error) {
	if !e.AssetExists(asset) {
		return nil, ErrInvalidAssetID
	}
	acc, err := e.GetAccountByID(e.accountID("", partyID, asset, types.AccountTypeLockWithdraw))
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
		Type:      types.TransferTypeWithdraw,
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
	if err := e.IncrementBalance(ctx, req.ToAccount[0].ID, amount); err != nil {
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
		Type:      types.TransferTypeDeposit,
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
	if err := e.IncrementBalance(ctx, acc.ID, amount); err != nil {
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
	if acc.Type != types.AccountTypeExternal {
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
	if acc.Type != types.AccountTypeExternal {
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
	if acc.Type != types.AccountTypeExternal {
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
	return acc.Clone(), nil
}

// GetAssetTotalSupply - return the total supply of the asset if it's known
// from the collateral engine.
func (e *Engine) GetAssetTotalSupply(asset string) (*num.Uint, error) {
	asst, ok := e.enabledAssets[asset]
	if !ok {
		return nil, fmt.Errorf("invalid asset: %s", asset)
	}

	return asst.GetAssetTotalSupply(), nil
}

func (e *Engine) removeAccount(id string) {
	delete(e.accs, id)
	e.removeAccountFromHashableSlice(id)
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
	liquidityAccID := e.accountID(market, systemOwner, asset, types.AccountTypeFeesLiquidity)
	return e.GetAccountByID(liquidityAccID)
}

func (e *Engine) GetMarketInsurancePoolAccount(market, asset string) (*types.Account, error) {
	insuranceAccID := e.accountID(market, systemOwner, asset, types.AccountTypeInsurance)
	return e.GetAccountByID(insuranceAccID)
}

// GetAssetInsurancePoolAccount returns the global insurance account for the asset
func (e *Engine) GetAssetInsurancePoolAccount(asset string) *types.Account {
	globalInsuranceID := e.accountID(noMarket, systemOwner, asset, types.AccountTypeGlobalInsurance)
	globalInsuranceAcc := e.accs[globalInsuranceID]
	return globalInsuranceAcc
}
