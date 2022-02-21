package staking

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	vgproto "code.vegaprotocol.io/protos/vega"
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/assets/erc20"
	"code.vegaprotocol.io/vega/events"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcmn "github.com/ethereum/go-ethereum/common"
)

var (
	ErrNoBalanceForParty                = errors.New("no balance for party")
	ErrStakeTotalSupplyAlreadyProcessed = errors.New("stake total supply already processed")
	ErrStakeTotalSupplyBeingProcessed   = errors.New("stake total supply being processed")
)

// Broker - the event bus.
type Broker interface {
	Send(events.Event)
	SendBatch([]events.Event)
}

type EthereumClientCaller interface {
	bind.ContractCaller
}

// EvtForwarder forwarder information to the tendermint chain to be agreed on by validators
//go:generate go run github.com/golang/mock/mockgen -destination mocks/evt_forwarder_mock.go -package mocks code.vegaprotocol.io/vega/staking EvtForwarder
type EvtForwarder interface {
	ForwardFromSelf(evt *commandspb.ChainEvent)
}

type Accounting struct {
	log              *logging.Logger
	ethClient        EthereumClientCaller
	cfg              Config
	broker           Broker
	accounts         map[string]*StakingAccount
	hashableAccounts []*StakingAccount
	isValidator      bool

	stakingAssetTotalSupply *num.Uint
	stakingBridgeAddress    ethcmn.Address

	// snapshot bits
	accState accountingSnapshotState

	// these two are used in order to propagate
	// the staking asset total supply at genesis.
	evtFwd                  EvtForwarder
	witness                 Witness
	pendingStakeTotalSupply *pendingStakeTotalSupply
	currentTime             time.Time
}

type pendingStakeTotalSupply struct {
	sts   *types.StakeTotalSupply
	check func() error
}

func (p pendingStakeTotalSupply) GetID() string {
	return hex.EncodeToString(vgcrypto.Hash([]byte(p.sts.String())))
}
func (p *pendingStakeTotalSupply) Check() error { return p.check() }

func NewAccounting(
	log *logging.Logger,
	cfg Config,
	broker Broker,
	ethClient EthereumClientCaller,
	evtForward EvtForwarder,
	witness Witness,
	tt TimeTicker,
	isValidator bool,
) (acc *Accounting) {
	defer func() {
		tt.NotifyOnTick(acc.onTick)
	}()
	log = log.Named("accounting")

	return &Accounting{
		log:                     log,
		cfg:                     cfg,
		broker:                  broker,
		ethClient:               ethClient,
		accounts:                map[string]*StakingAccount{},
		stakingAssetTotalSupply: num.Zero(),
		accState:                accountingSnapshotState{changed: true},
		evtFwd:                  evtForward,
		witness:                 witness,
		isValidator:             isValidator,
	}
}

func (a *Accounting) onTick(_ context.Context, t time.Time) {
	a.currentTime = t
}

func (a *Accounting) Hash() []byte {
	output := make([]byte, len(a.hashableAccounts)*32)
	var i int
	for _, k := range a.hashableAccounts {
		bal := k.Balance.Bytes()
		copy(output[i:], bal[:])
		i += 32
	}
	h := vgcrypto.Hash(output)
	a.log.Debug("stakeccounts state hash", logging.String("hash", hex.EncodeToString(h)))
	return h
}

func (a *Accounting) AddEvent(ctx context.Context, evt *types.StakeLinking) {
	acc, ok := a.accounts[evt.Party]
	if !ok {
		a.broker.Send(events.NewPartyEvent(ctx, types.Party{Id: evt.Party}))
		acc = NewStakingAccount(evt.Party)
		a.accounts[evt.Party] = acc
		a.hashableAccounts = append(a.hashableAccounts, acc)
		a.accState.changed = true
	}

	// errors here do not really matter I'd say
	// they are either validation issue, that we can just log
	// but should never happen as things should be created properly
	// or errors from event being received in the wrong order
	// but that we cannot really prevent and that the account
	// would recover from by itself later on.
	// Negative balance is possible when processing orders in disorder,
	// not a big deal
	if err := acc.AddEvent(evt); err != nil && err != ErrNegativeBalance {
		a.log.Error("could not add event to staking account",
			logging.Error(err))
		return
	}
	a.accState.changed = true
}

// GetAllAvailableBalances returns the staking balance for all parties.
func (a *Accounting) GetAllAvailableBalances() map[string]*num.Uint {
	balances := map[string]*num.Uint{}
	for party, acc := range a.accounts {
		balances[party] = acc.GetAvailableBalance()
	}
	return balances
}

func (a *Accounting) UpdateStakingBridgeAddress(stakingBridgeAddress ethcmn.Address) error {
	a.stakingBridgeAddress = stakingBridgeAddress

	if !a.accState.isRestoring {
		if err := a.updateStakingAssetTotalSupply(); err != nil {
			return fmt.Errorf("couldn't update the total supply of the staking asset: %w", err)
		}
	}

	return nil
}

func (a *Accounting) ProcessStakeTotalSupply(_ context.Context, evt *types.StakeTotalSupply) error {
	if a.stakingAssetTotalSupply.NEQ(num.Zero()) {
		return ErrStakeTotalSupplyAlreadyProcessed
	}

	if a.pendingStakeTotalSupply != nil {
		return ErrStakeTotalSupplyBeingProcessed
	}

	expectedSupply := evt.TotalSupply.Clone()

	a.pendingStakeTotalSupply = &pendingStakeTotalSupply{
		sts: evt,
		check: func() error {
			totalSupply, err := a.getStakeAssetTotalSupply(a.stakingBridgeAddress)
			if err != nil {
				return err
			}

			if totalSupply.NEQ(expectedSupply) {
				return fmt.Errorf(
					"invalid stake asset total supply, expected %s got %s",
					expectedSupply.String(), totalSupply.String(),
				)
			}

			return nil
		},
	}

	a.log.Info("stake total supply event received, starting validation",
		logging.String("event", evt.String()))

	return a.witness.StartCheck(
		a.pendingStakeTotalSupply,
		a.onStakeTotalSupplyVerified,
		a.currentTime.Add(timeTilCancel),
	)
}

func (a *Accounting) onStakeTotalSupplyVerified(event interface{}, ok bool) {
	if ok {
		a.accState.changed = true
		a.stakingAssetTotalSupply = a.pendingStakeTotalSupply.sts.TotalSupply
		a.log.Info("stake total supply finalized",
			logging.BigUint("total-supply", a.stakingAssetTotalSupply))
	}
	a.pendingStakeTotalSupply = nil
}

func (a *Accounting) updateStakingAssetTotalSupply() error {
	if !a.isValidator {
		// nothing to do here if we are not a validator
		return nil
	}

	totalSupply, err := a.getStakeAssetTotalSupply(a.stakingBridgeAddress)
	if err != nil {
		return err
	}

	// send the event to be forwarded
	a.evtFwd.ForwardFromSelf(&commandspb.ChainEvent{
		TxId: "internal",
		Event: &commandspb.ChainEvent_StakingEvent{
			StakingEvent: &vgproto.StakingEvent{
				Action: &vgproto.StakingEvent_TotalSupply{
					TotalSupply: &vgproto.StakeTotalSupply{
						TokenAddress: a.stakingBridgeAddress.Hex(),
						TotalSupply:  totalSupply.String(),
					},
				},
			},
		},
	})

	return nil
}

func (a *Accounting) getStakeAssetTotalSupply(address ethcmn.Address) (*num.Uint, error) {
	sc, err := NewStakingCaller(address, a.ethClient)
	if err != nil {
		return nil, err
	}

	st, err := sc.StakingToken(&bind.CallOpts{})
	if err != nil {
		return nil, err
	}

	tc, err := erc20.NewErc20Caller(st, a.ethClient)
	if err != nil {
		return nil, err
	}

	ts, err := tc.TotalSupply(&bind.CallOpts{})
	if err != nil {
		return nil, err
	}

	totalSupply, overflowed := num.UintFromBig(ts)
	if overflowed {
		return nil, fmt.Errorf("failed to convert big.Int to num.Uint: %s", ts.String())
	}

	symbol, _ := tc.Symbol(&bind.CallOpts{})
	decimals, _ := tc.Decimals(&bind.CallOpts{})

	a.log.Debug("staking asset loaded",
		logging.String("symbol", symbol),
		logging.Uint8("decimals", decimals),
		logging.String("total-supply", ts.String()),
	)

	return totalSupply, nil
}

func (a *Accounting) GetAvailableBalance(party string) (*num.Uint, error) {
	acc, ok := a.accounts[party]
	if !ok {
		return num.Zero(), ErrNoBalanceForParty
	}

	return acc.GetAvailableBalance(), nil
}

func (a *Accounting) GetAvailableBalanceAt(
	party string, at time.Time) (*num.Uint, error) {
	acc, ok := a.accounts[party]
	if !ok {
		return num.Zero(), ErrNoBalanceForParty
	}

	return acc.GetAvailableBalanceAt(at)
}

func (a *Accounting) GetAvailableBalanceInRange(
	party string, from, to time.Time) (*num.Uint, error) {
	acc, ok := a.accounts[party]
	if !ok {
		return num.Zero(), ErrNoBalanceForParty
	}

	return acc.GetAvailableBalanceInRange(from, to)
}

func (a *Accounting) GetStakingAssetTotalSupply() *num.Uint {
	return a.stakingAssetTotalSupply.Clone()
}

func (a *Accounting) ValidatorKeyChanged(ctx context.Context, oldKey, newKey string) {
	account, ok := a.accounts[oldKey]
	if !ok {
		return
	}

	// find the index of the old pub key in the hashable accounts slice.
	oldInd := -1
	for i, acc := range a.hashableAccounts {
		if acc.Party == oldKey {
			oldInd = i
			break
		}
	}
	if oldInd == -1 {
		a.log.Panic("Accounts and hashable accounts are out of sync", logging.String("public-key", oldKey))
	}

	// instantiate new account and add to it all of the events with a modified party id
	acc := NewStakingAccount(newKey)
	a.broker.Send(events.NewPartyEvent(ctx, types.Party{Id: newKey}))
	a.accounts[newKey] = acc
	for _, event := range account.Events {
		event.Party = newKey
		acc.AddEvent(event)
		a.broker.Send(events.NewStakeLinking(ctx, *event))
	}
	delete(a.accounts, oldKey)

	// update the account with the new stake linking events
	a.hashableAccounts[oldInd] = acc
	a.accState.changed = true
}
