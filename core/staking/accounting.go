// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package staking

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/core/contracts/erc20"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	vgproto "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

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

// EvtForwarder forwarder information to the tendermint chain to be agreed on by validators.
type EvtForwarder interface {
	ForwardFromSelf(evt *commandspb.ChainEvent)
}

type Accounting struct {
	log              *logging.Logger
	ethClient        EthereumClientCaller
	cfg              Config
	timeService      TimeService
	broker           Broker
	accounts         map[string]*Account
	hashableAccounts []*Account
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
}

type pendingStakeTotalSupply struct {
	sts   *types.StakeTotalSupply
	check func() error
}

func (p pendingStakeTotalSupply) GetID() string {
	return hex.EncodeToString(vgcrypto.Hash([]byte(p.sts.String())))
}

func (p pendingStakeTotalSupply) GetType() types.NodeVoteType {
	return types.NodeVoteTypeStakeTotalSupply
}

func (p *pendingStakeTotalSupply) Check() error { return p.check() }

func NewAccounting(
	log *logging.Logger,
	cfg Config,
	ts TimeService,
	broker Broker,
	ethClient EthereumClientCaller,
	evtForward EvtForwarder,
	witness Witness,
	isValidator bool,
) (acc *Accounting) {
	log = log.Named("accounting")

	return &Accounting{
		log:                     log,
		cfg:                     cfg,
		timeService:             ts,
		broker:                  broker,
		ethClient:               ethClient,
		accounts:                map[string]*Account{},
		stakingAssetTotalSupply: num.UintZero(),
		accState:                accountingSnapshotState{},
		evtFwd:                  evtForward,
		witness:                 witness,
		isValidator:             isValidator,
	}
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
	a.log.Debug("stake accounts state hash", logging.String("hash", hex.EncodeToString(h)))
	return h
}

func (a *Accounting) AddEvent(ctx context.Context, evt *types.StakeLinking) {
	acc, ok := a.accounts[evt.Party]
	if !ok {
		acc = NewStakingAccount(evt.Party)
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

	// only add the account if all went well
	if !ok {
		a.broker.Send(events.NewPartyEvent(ctx, types.Party{Id: evt.Party}))
		a.accounts[evt.Party] = acc
		a.hashableAccounts = append(a.hashableAccounts, acc)
	}
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
	if a.stakingAssetTotalSupply.NEQ(num.UintZero()) {
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
		a.timeService.GetTimeNow().Add(timeTilCancel),
	)
}

func (a *Accounting) getLastBlockSeen() uint64 {
	var block uint64
	for _, acc := range a.hashableAccounts {
		if len(acc.Events) == 0 {
			continue
		}
		height := acc.Events[len(acc.Events)-1].BlockHeight
		if block < height {
			block = height
		}
	}
	return block
}

func (a *Accounting) onStakeTotalSupplyVerified(event interface{}, ok bool) {
	if ok {
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
		return num.UintZero(), ErrNoBalanceForParty
	}

	return acc.GetAvailableBalance(), nil
}

func (a *Accounting) GetAvailableBalanceAt(
	party string, at time.Time,
) (*num.Uint, error) {
	acc, ok := a.accounts[party]
	if !ok {
		return num.UintZero(), ErrNoBalanceForParty
	}

	return acc.GetAvailableBalanceAt(at)
}

func (a *Accounting) GetAvailableBalanceInRange(
	party string, from, to time.Time,
) (*num.Uint, error) {
	acc, ok := a.accounts[party]
	if !ok {
		return num.UintZero(), ErrNoBalanceForParty
	}

	return acc.GetAvailableBalanceInRange(from, to)
}

func (a *Accounting) GetStakingAssetTotalSupply() *num.Uint {
	return a.stakingAssetTotalSupply.Clone()
}
