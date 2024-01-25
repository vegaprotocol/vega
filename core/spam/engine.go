// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package spam

import (
	"context"
	"encoding/hex"
	"sync"

	"code.vegaprotocol.io/vega/core/blockchain/abci"
	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/txn"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	protoapi "code.vegaprotocol.io/vega/protos/vega/api/v1"
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
	noSpamProtection        bool // flag that disables chesk for the spam policies, that is useful for the nullchain
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

	createReferralSetPolicy := NewSimpleSpamPolicy("createReferralSet", netparams.ReferralProgramMinStakedVegaTokens, netparams.SpamProtectionMaxCreateReferralSet, log, accounting)
	updateReferralSetPolicy := NewSimpleSpamPolicy("updateReferralSet", netparams.ReferralProgramMinStakedVegaTokens, netparams.SpamProtectionMaxUpdateReferralSet, log, accounting)
	applyReferralCodePolicy := NewSimpleSpamPolicy("applyReferralCode", "", netparams.SpamProtectionMaxApplyReferralCode, log, accounting)
	updatePartyProfilePolicy := NewSimpleSpamPolicy("updatePartyProfile", "", netparams.SpamProtectionMaxUpdatePartyProfile, log, accounting)

	// complex policies
	votePolicy := NewVoteSpamPolicy(netparams.SpamProtectionMinTokensForVoting, netparams.SpamProtectionMaxVotes, log, accounting)

	voteKey := (&types.PayloadVoteSpamPolicy{}).Key()
	e.policyNameToPolicy = map[string]Policy{
		proposalPolicy.policyName:           proposalPolicy,
		valJoinPolicy.policyName:            valJoinPolicy,
		delegationPolicy.policyName:         delegationPolicy,
		transferPolicy.policyName:           transferPolicy,
		issuesSignaturesPolicy.policyName:   issuesSignaturesPolicy,
		voteKey:                             votePolicy,
		createReferralSetPolicy.policyName:  createReferralSetPolicy,
		updateReferralSetPolicy.policyName:  updateReferralSetPolicy,
		applyReferralCodePolicy.policyName:  applyReferralCodePolicy,
		updatePartyProfilePolicy.policyName: updatePartyProfilePolicy,
	}
	e.hashKeys = []string{
		proposalPolicy.policyName,
		valJoinPolicy.policyName,
		delegationPolicy.policyName,
		transferPolicy.policyName,
		issuesSignaturesPolicy.policyName,
		createReferralSetPolicy.policyName,
		updateReferralSetPolicy.policyName,
		applyReferralCodePolicy.policyName,
		updatePartyProfilePolicy.policyName,
		voteKey,
	}

	e.transactionTypeToPolicy[txn.ProposeCommand] = proposalPolicy
	e.transactionTypeToPolicy[txn.BatchProposeCommand] = proposalPolicy
	e.transactionTypeToPolicy[txn.AnnounceNodeCommand] = valJoinPolicy
	e.transactionTypeToPolicy[txn.DelegateCommand] = delegationPolicy
	e.transactionTypeToPolicy[txn.UndelegateCommand] = delegationPolicy
	e.transactionTypeToPolicy[txn.TransferFundsCommand] = transferPolicy
	e.transactionTypeToPolicy[txn.CancelTransferFundsCommand] = transferPolicy
	e.transactionTypeToPolicy[txn.IssueSignatures] = issuesSignaturesPolicy
	e.transactionTypeToPolicy[txn.VoteCommand] = votePolicy
	e.transactionTypeToPolicy[txn.CreateReferralSetCommand] = createReferralSetPolicy
	e.transactionTypeToPolicy[txn.UpdateReferralSetCommand] = updateReferralSetPolicy
	e.transactionTypeToPolicy[txn.ApplyReferralCodeCommand] = applyReferralCodePolicy
	e.transactionTypeToPolicy[txn.UpdatePartyProfileCommand] = updatePartyProfilePolicy

	// register for epoch end notifications
	epochEngine.NotifyOnEpoch(e.OnEpochEvent, e.OnEpochRestore)
	e.log.Info("Spam protection started")

	return e
}

func (e *Engine) DisableSpamProtection() {
	e.log.Infof("Disabling spam protection for the Spam Engine")
	e.noSpamProtection = true
}

// OnCreateReferralSet is called when the net param for max create referral set per epoch has changed.
func (e *Engine) OnMaxCreateReferralSet(ctx context.Context, max int64) error {
	return e.transactionTypeToPolicy[txn.CreateReferralSetCommand].UpdateIntParam(netparams.SpamProtectionMaxCreateReferralSet, max)
}

// OnMaxPartyProfileUpdate is called when the net param for max update party profile per epoch has changed.
func (e *Engine) OnMaxPartyProfile(ctx context.Context, max int64) error {
	return e.transactionTypeToPolicy[txn.UpdatePartyProfileCommand].UpdateIntParam(netparams.SpamProtectionMaxUpdatePartyProfile, max)
}

// OnMaxUpdateReferralSet is called when the net param for max update referral set per epoch has changed.
func (e *Engine) OnMaxUpdateReferralSet(ctx context.Context, max int64) error {
	return e.transactionTypeToPolicy[txn.UpdateReferralSetCommand].UpdateIntParam(netparams.SpamProtectionMaxUpdateReferralSet, max)
}

// OnMaxApplyReferralCode is called when the net param for max update referral set per epoch has changed.
func (e *Engine) OnMaxApplyReferralCode(ctx context.Context, max int64) error {
	return e.transactionTypeToPolicy[txn.ApplyReferralCodeCommand].UpdateIntParam(netparams.SpamProtectionMaxApplyReferralCode, max)
}

// OnMinTokensForReferral is called when the net param for min staked tokens requirement for referral set create/update has changed.
func (e *Engine) OnMinTokensForReferral(ctx context.Context, minTokens *num.Uint) error {
	err := e.transactionTypeToPolicy[txn.CreateReferralSetCommand].UpdateUintParam(netparams.ReferralProgramMinStakedVegaTokens, minTokens)
	if err != nil {
		return err
	}
	return e.transactionTypeToPolicy[txn.UpdateReferralSetCommand].UpdateUintParam(netparams.ReferralProgramMinStakedVegaTokens, minTokens)
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

	if e.noSpamProtection {
		e.log.Info("Spam protection OnEpochEvent disabled", logging.Uint64("epoch", epoch.Seq))
		return
	}

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
	if e.noSpamProtection {
		e.log.Debug("Spam protection PreBlockAccept disabled for policy", logging.String("txHash", hex.EncodeToString(tx.Hash())), logging.String("command", command.String()))
		return nil
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
		if e.noSpamProtection {
			e.log.Debug("Spam protection PreBlockAccept disabled for policy", logging.String("txHash", hex.EncodeToString(tx.Hash())), logging.String("command", command.String()))
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

	if e.noSpamProtection {
		e.log.Debug("Spam protection PreBlockAccept disabled for policy", logging.String("txHash", hex.EncodeToString(tx.Hash())), logging.String("command", command.String()))
		return nil
	}
	return e.transactionTypeToPolicy[command].CheckBlockTx(tx)
}

func (e *Engine) GetSpamStatistics(partyID string) *protoapi.SpamStatistics {
	stats := &protoapi.SpamStatistics{}

	for txType, policy := range e.transactionTypeToPolicy {
		switch txType {
		case txn.ProposeCommand, txn.BatchProposeCommand:
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
		case txn.CreateReferralSetCommand:
			stats.CreateReferralSet = policy.GetSpamStats(partyID)
		case txn.UpdateReferralSetCommand:
			stats.UpdateReferralSet = policy.GetSpamStats(partyID)
		case txn.ApplyReferralCodeCommand:
			stats.ApplyReferralCode = policy.GetSpamStats(partyID)
		default:
			continue
		}
	}

	return stats
}
