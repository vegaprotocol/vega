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

package spam

import (
	"context"
	"encoding/hex"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/core/netparams"

	"code.vegaprotocol.io/vega/core/blockchain/abci"
	"code.vegaprotocol.io/vega/core/txn"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

var (
	increaseFactor             = num.NewUint(2)
	banDurationAsEpochFraction = num.DecimalOne().Div(num.DecimalFromInt64(48)) // 1/48 of an epoch will be the default 30 minutes ban
	banFactor                  = num.DecimalFromFloat(0.5)
)

const (
	rejectRatioForIncrease         float64 = 0.3
	numberOfEpochsBan              uint64  = 4
	numberOfBlocksForIncreaseCheck uint64  = 10
	minBanDuration                         = time.Second * 30 // minimum ban duration
	formatBase                             = 10
)

type StakingAccounts interface {
	GetAvailableBalance(party string) (*num.Uint, error)
}

type EpochEngine interface {
	NotifyOnEpoch(f func(context.Context, types.Epoch), r func(context.Context, types.Epoch))
}

type Engine struct {
	log        *logging.Logger
	config     Config
	accounting StakingAccounts
	cfgMu      sync.Mutex

	transactionTypeToPolicy map[txn.Command]Policy
	currentEpoch            *types.Epoch
	policyNameToPolicy      map[string]Policy
	hashKeys                []string
	banDuration             time.Duration
}

type Policy interface {
	Reset(epoch types.Epoch)
	EndOfBlock(blockHeight uint64, now time.Time, banDuration time.Duration)
	PreBlockAccept(tx abci.Tx) (bool, error)
	PostBlockAccept(tx abci.Tx) (bool, error)
	UpdateUintParam(name string, value *num.Uint) error
	UpdateIntParam(name string, value int64) error
	Serialise() ([]byte, error)
	Deserialise(payload *types.Payload) error
	GetStats(partyID string) Statistic
}

// ReloadConf updates the internal configuration of the spam engine.
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
	e.config = cfg
	e.cfgMu.Unlock()
}

// New instantiates a new spam engine.
func New(log *logging.Logger, config Config, epochEngine EpochEngine, accounting StakingAccounts) *Engine {
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())
	e := &Engine{
		config:                  config,
		log:                     log,
		accounting:              accounting,
		transactionTypeToPolicy: map[txn.Command]Policy{},
	}

	proposalPolicy := NewSimpleSpamPolicy("proposal", netparams.SpamProtectionMinTokensForProposal, netparams.SpamProtectionMaxProposals, log, accounting)
	valJoinPolicy := NewSimpleSpamPolicy("validatorJoin", netparams.StakingAndDelegationRewardMinimumValidatorStake, "", log, accounting)
	delegationPolicy := NewSimpleSpamPolicy("delegation", netparams.SpamProtectionMinTokensForDelegation, netparams.SpamProtectionMaxDelegations, log, accounting)
	votePolicy := NewVoteSpamPolicy(netparams.SpamProtectionMinTokensForVoting, netparams.SpamProtectionMaxVotes, log, accounting)
	transferPolicy := NewSimpleSpamPolicy("transfer", "", netparams.TransferMaxCommandsPerEpoch, log, accounting)

	voteKey := (&types.PayloadVoteSpamPolicy{}).Key()
	e.policyNameToPolicy = map[string]Policy{voteKey: votePolicy, proposalPolicy.policyName: proposalPolicy, delegationPolicy.policyName: delegationPolicy}
	e.hashKeys = []string{voteKey, proposalPolicy.policyName, delegationPolicy.policyName}

	e.transactionTypeToPolicy[txn.ProposeCommand] = proposalPolicy
	e.transactionTypeToPolicy[txn.VoteCommand] = votePolicy
	e.transactionTypeToPolicy[txn.DelegateCommand] = delegationPolicy
	e.transactionTypeToPolicy[txn.UndelegateCommand] = delegationPolicy
	e.transactionTypeToPolicy[txn.TransferFundsCommand] = transferPolicy
	e.transactionTypeToPolicy[txn.CancelTransferFundsCommand] = transferPolicy
	e.transactionTypeToPolicy[txn.AnnounceNodeCommand] = valJoinPolicy

	// register for epoch end notifications
	epochEngine.NotifyOnEpoch(e.OnEpochEvent, e.OnEpochRestore)
	e.log.Info("Spam protection started")

	return e
}

// OnEpochDurationChanged updates the ban duration as a fraction of the epoch duration.
func (e *Engine) OnEpochDurationChanged(_ context.Context, duration time.Duration) error {
	epochImpliedDurationNano, _ := num.UintFromDecimal(num.DecimalFromInt64(duration.Nanoseconds()).Mul(banDurationAsEpochFraction))
	epochImpliedDurationDuration := time.Duration(epochImpliedDurationNano.Uint64())
	if epochImpliedDurationDuration < minBanDuration {
		e.banDuration = minBanDuration
	} else {
		e.banDuration = epochImpliedDurationDuration
	}
	return nil
}

// OnMaxDelegationsChanged is called when the net param for max delegations per epoch has changed.
func (e *Engine) OnMaxDelegationsChanged(ctx context.Context, maxDelegations int64) error {
	return e.transactionTypeToPolicy[txn.DelegateCommand].UpdateIntParam(netparams.SpamProtectionMaxDelegations, maxDelegations)
}

// OnMinTokensForDelegationChanged is called when the net param for min tokens requirement for voting has changed.
func (e *Engine) OnMinTokensForDelegationChanged(ctx context.Context, minTokens num.Decimal) error {
	minTokensFoDelegation, _ := num.UintFromDecimal(minTokens)
	return e.transactionTypeToPolicy[txn.DelegateCommand].UpdateUintParam(netparams.SpamProtectionMinTokensForDelegation, minTokensFoDelegation)
}

// OnMaxVotesChanged is called when the net param for max votes per epoch has changed.
func (e *Engine) OnMaxVotesChanged(ctx context.Context, maxVotes int64) error {
	return e.transactionTypeToPolicy[txn.VoteCommand].UpdateIntParam(netparams.SpamProtectionMaxVotes, maxVotes)
}

// OnMinTokensForVotingChanged is called when the net param for min tokens requirement for voting has changed.
func (e *Engine) OnMinTokensForVotingChanged(ctx context.Context, minTokens num.Decimal) error {
	minTokensForVoting, _ := num.UintFromDecimal(minTokens)
	return e.transactionTypeToPolicy[txn.VoteCommand].UpdateUintParam(netparams.SpamProtectionMinTokensForVoting, minTokensForVoting)
}

// OnMaxProposalsChanged is called when the net param for max proposals per epoch has changed.
func (e *Engine) OnMaxProposalsChanged(ctx context.Context, maxProposals int64) error {
	return e.transactionTypeToPolicy[txn.ProposeCommand].UpdateIntParam(netparams.SpamProtectionMaxProposals, maxProposals)
}

// OnMinTokensForProposalChanged is called when the net param for min tokens requirement for submitting a proposal has changed.
func (e *Engine) OnMinTokensForProposalChanged(ctx context.Context, minTokens num.Decimal) error {
	minTokensForProposal, _ := num.UintFromDecimal(minTokens)
	return e.transactionTypeToPolicy[txn.ProposeCommand].UpdateUintParam(netparams.SpamProtectionMinTokensForProposal, minTokensForProposal)
}

// OnMaxTransfersChanged is called when the net param for max transfers per epoch changes.
func (e *Engine) OnMaxTransfersChanged(_ context.Context, maxTransfers int64) error {
	return e.transactionTypeToPolicy[txn.TransferFundsCommand].UpdateIntParam(netparams.TransferMaxCommandsPerEpoch, maxTransfers)
}

// OnMinValidatorTokensChanged is called when the net param for min tokens for joining validator changes.
func (e *Engine) OnMinValidatorTokensChanged(_ context.Context, minTokens num.Decimal) error {
	minTokensForJoiningValidator, _ := num.UintFromDecimal(minTokens)
	return e.transactionTypeToPolicy[txn.AnnounceNodeCommand].UpdateUintParam(netparams.StakingAndDelegationRewardMinimumValidatorStake, minTokensForJoiningValidator)
}

// OnEpochEvent is a callback for epoch events.
func (e *Engine) OnEpochEvent(ctx context.Context, epoch types.Epoch) {
	e.log.Info("Spam protection OnEpochEvent called", logging.Uint64("epoch", epoch.Seq))

	if e.currentEpoch == nil || e.currentEpoch.Seq != epoch.Seq {
		if e.log.GetLevel() <= logging.DebugLevel {
			e.log.Debug("Spam protection new epoch started", logging.Uint64("epochSeq", epoch.Seq))
		}
		e.currentEpoch = &epoch

		for _, policy := range e.transactionTypeToPolicy {
			policy.Reset(epoch)
		}
	}
}

// EndOfBlock is called when the block is finished.
func (e *Engine) EndOfBlock(blockHeight uint64, now time.Time) {
	if e.log.GetLevel() <= logging.DebugLevel {
		e.log.Debug("Spam protection EndOfBlock called", logging.Uint64("blockHeight", blockHeight))
	}
	for _, policy := range e.transactionTypeToPolicy {
		policy.EndOfBlock(blockHeight, now, e.banDuration)
	}
}

// PreBlockAccept is called from onCheckTx before a tx is added to mempool
// returns false is rejected by spam engine with a corresponding error.
func (e *Engine) PreBlockAccept(tx abci.Tx) (bool, error) {
	command := tx.Command()
	if _, ok := e.transactionTypeToPolicy[command]; !ok {
		return true, nil
	}
	if e.log.GetLevel() <= logging.DebugLevel {
		e.log.Debug("Spam protection PreBlockAccept called for policy", logging.String("txHash", hex.EncodeToString(tx.Hash())), logging.String("command", command.String()))
	}
	return e.transactionTypeToPolicy[command].PreBlockAccept(tx)
}

// PostBlockAccept is called from onDeliverTx before the block is processed
// returns false is rejected by spam engine with a corresponding error.
func (e *Engine) PostBlockAccept(tx abci.Tx) (bool, error) {
	command := tx.Command()
	if _, ok := e.transactionTypeToPolicy[command]; !ok {
		return true, nil
	}
	if e.log.GetLevel() <= logging.DebugLevel {
		e.log.Debug("Spam protection PostBlockAccept called for policy", logging.String("txHash", hex.EncodeToString(tx.Hash())), logging.String("command", command.String()))
	}
	return e.transactionTypeToPolicy[command].PostBlockAccept(tx)
}

func (e *Engine) GetSpamStatistics(partyID string) Statistics {
	stats := Statistics{}

	for txType, policy := range e.transactionTypeToPolicy {
		statistic := policy.GetStats(partyID)
		switch txType {
		case txn.ProposeCommand:
			stats.Proposals = statistic
		case txn.DelegateCommand:
			stats.Delegations = statistic
		case txn.TransferFundsCommand:
			stats.Transfers = statistic
		case txn.AnnounceNodeCommand:
			stats.NodeAnnouncements = statistic
		case txn.VoteCommand:

			total, err := num.DecimalFromString(statistic.Total)
			if err != nil || total.Equal(num.DecimalZero()) {
				continue
			}

			rejected, err := num.DecimalFromString(statistic.BlockCount)
			if err != nil {
				continue
			}

			ratio := rejected.Div(total)

			statistics := VoteStatistic{
				Total:         statistic.Total,
				Rejected:      statistic.BlockCount,
				RejectedRatio: ratio.String(),
				Limit:         statistic.Limit,
				BlockedUntil:  statistic.BlockedUntil,
			}

			stats.Votes = statistics
		default:
			continue
		}
	}

	return stats
}
