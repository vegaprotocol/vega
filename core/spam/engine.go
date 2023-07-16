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

	protoapi "code.vegaprotocol.io/vega/protos/vega/api/v1"

	"code.vegaprotocol.io/vega/core/netparams"

	"code.vegaprotocol.io/vega/core/blockchain/abci"
	"code.vegaprotocol.io/vega/core/txn"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
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
}

type Policy interface {
	Reset(epoch types.Epoch)
	UpdateTx(tx abci.Tx)
	RollbackProposal()
	CheckBlockTx(abci.Tx) error
	PreBlockAccept(tx abci.Tx) error
	UpdateUintParam(name string, value *num.Uint) error
	UpdateIntParam(name string, value int64) error
	Serialise() ([]byte, error)
	Deserialise(payload *types.Payload) error
	GetSpamStats(partyID string) *protoapi.SpamStatistic
	GetVoteSpamStats(partyID string) *protoapi.VoteSpamStatistics
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

	// simple policies
	proposalPolicy := NewSimpleSpamPolicy("proposal", netparams.SpamProtectionMinTokensForProposal, netparams.SpamProtectionMaxProposals, log, accounting)
	valJoinPolicy := NewSimpleSpamPolicy("validatorJoin", netparams.StakingAndDelegationRewardMinimumValidatorStake, "", log, accounting)
	delegationPolicy := NewSimpleSpamPolicy("delegation", netparams.SpamProtectionMinTokensForDelegation, netparams.SpamProtectionMaxDelegations, log, accounting)
	transferPolicy := NewSimpleSpamPolicy("transfer", "", netparams.TransferMaxCommandsPerEpoch, log, accounting)
	issuesSignaturesPolicy := NewSimpleSpamPolicy("issueSignature", netparams.SpamProtectionMinMultisigUpdates, "", log, accounting)

	// complex policies
	votePolicy := NewVoteSpamPolicy(netparams.SpamProtectionMinTokensForVoting, netparams.SpamProtectionMaxVotes, log, accounting)

	voteKey := (&types.PayloadVoteSpamPolicy{}).Key()
	e.policyNameToPolicy = map[string]Policy{
		proposalPolicy.policyName:         proposalPolicy,
		valJoinPolicy.policyName:          valJoinPolicy,
		delegationPolicy.policyName:       delegationPolicy,
		transferPolicy.policyName:         transferPolicy,
		issuesSignaturesPolicy.policyName: issuesSignaturesPolicy,
		voteKey:                           votePolicy,
	}
	e.hashKeys = []string{
		proposalPolicy.policyName,
		valJoinPolicy.policyName,
		delegationPolicy.policyName,
		transferPolicy.policyName,
		issuesSignaturesPolicy.policyName,
		voteKey,
	}

	e.transactionTypeToPolicy[txn.ProposeCommand] = proposalPolicy
	e.transactionTypeToPolicy[txn.AnnounceNodeCommand] = valJoinPolicy
	e.transactionTypeToPolicy[txn.DelegateCommand] = delegationPolicy
	e.transactionTypeToPolicy[txn.UndelegateCommand] = delegationPolicy
	e.transactionTypeToPolicy[txn.TransferFundsCommand] = transferPolicy
	e.transactionTypeToPolicy[txn.CancelTransferFundsCommand] = transferPolicy
	e.transactionTypeToPolicy[txn.IssueSignatures] = issuesSignaturesPolicy
	e.transactionTypeToPolicy[txn.VoteCommand] = votePolicy

	// register for epoch end notifications
	epochEngine.NotifyOnEpoch(e.OnEpochEvent, e.OnEpochRestore)
	e.log.Info("Spam protection started")

	return e
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

// OnMinTokensForProposalChanged is called when the net param for min tokens requirement for submitting a proposal has changed.
func (e *Engine) OnMinTokensForMultisigUpdatesChanged(ctx context.Context, minTokens num.Decimal) error {
	minTokensForMultisigUpdates, _ := num.UintFromDecimal(minTokens)
	return e.transactionTypeToPolicy[txn.IssueSignatures].UpdateUintParam(netparams.SpamProtectionMinMultisigUpdates, minTokensForMultisigUpdates)
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

func (e *Engine) BeginBlock(txs []abci.Tx) {
	for _, tx := range txs {
		if _, ok := e.transactionTypeToPolicy[tx.Command()]; !ok {
			continue
		}
		e.transactionTypeToPolicy[tx.Command()].UpdateTx(tx)
	}
}

func (e *Engine) EndPrepareProposal() {
	for _, policy := range e.transactionTypeToPolicy {
		policy.RollbackProposal()
	}
}

// PreBlockAccept is called from onCheckTx before a tx is added to mempool
// returns false is rejected by spam engine with a corresponding error.
func (e *Engine) PreBlockAccept(tx abci.Tx) error {
	command := tx.Command()
	if _, ok := e.transactionTypeToPolicy[command]; !ok {
		return nil
	}
	if e.log.GetLevel() <= logging.DebugLevel {
		e.log.Debug("Spam protection PreBlockAccept called for policy", logging.String("txHash", hex.EncodeToString(tx.Hash())), logging.String("command", command.String()))
	}
	return e.transactionTypeToPolicy[command].PreBlockAccept(tx)
}

func (e *Engine) ProcessProposal(txs []abci.Tx) bool {
	success := true
	for _, tx := range txs {
		command := tx.Command()
		if _, ok := e.transactionTypeToPolicy[command]; !ok {
			continue
		}
		if err := e.transactionTypeToPolicy[command].CheckBlockTx(tx); err != nil {
			success = false
		}
	}
	for _, p := range e.transactionTypeToPolicy {
		p.RollbackProposal()
	}
	return success
}

// PostBlockAccept is called from onDeliverTx before the block is processed
// returns false is rejected by spam engine with a corresponding error.
func (e *Engine) CheckBlockTx(tx abci.Tx) error {
	command := tx.Command()
	if _, ok := e.transactionTypeToPolicy[command]; !ok {
		return nil
	}
	if e.log.GetLevel() <= logging.DebugLevel {
		e.log.Debug("Spam protection PostBlockAccept called for policy", logging.String("txHash", hex.EncodeToString(tx.Hash())), logging.String("command", command.String()))
	}
	return e.transactionTypeToPolicy[command].CheckBlockTx(tx)
}

func (e *Engine) GetSpamStatistics(partyID string) *protoapi.SpamStatistics {
	stats := &protoapi.SpamStatistics{}

	for txType, policy := range e.transactionTypeToPolicy {
		switch txType {
		case txn.ProposeCommand:
			stats.Proposals = policy.GetSpamStats(partyID)
		case txn.DelegateCommand:
			stats.Delegations = policy.GetSpamStats(partyID)
		case txn.TransferFundsCommand:
			stats.Transfers = policy.GetSpamStats(partyID)
		case txn.AnnounceNodeCommand:
			stats.NodeAnnouncements = policy.GetSpamStats(partyID)
		case txn.IssueSignatures:
			stats.IssueSignatures = policy.GetSpamStats(partyID)
		case txn.VoteCommand:
			stats.Votes = policy.GetVoteSpamStats(partyID)
		default:
			continue
		}
	}

	return stats
}
