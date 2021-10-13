package spam

import (
	"context"

	"code.vegaprotocol.io/vega/netparams"

	"code.vegaprotocol.io/vega/blockchain/abci"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/txn"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

var increaseFactor = num.NewUint(2)

const (
	rejectRatioForIncrease         float64 = 0.3
	numberOfEpochsBan              uint64  = 4
	numberOfBlocksForIncreaseCheck uint64  = 10
	banFactor                              = 0.5
)

type Accounting interface {
	GetAllAvailableBalances() map[string]*num.Uint
}

type EpochEngine interface {
	NotifyOnEpoch(f func(context.Context, types.Epoch))
}

type Engine struct {
	log        *logging.Logger
	config     Config
	accounting Accounting

	transactionTypeToPolicy map[txn.Command]SpamPolicy
	currentEpoch            *types.Epoch
	policyNameToPolicy      map[string]SpamPolicy
	hashKeys                []string
}

type SpamPolicy interface {
	Reset(epoch types.Epoch, tokenBalances map[string]*num.Uint)
	EndOfBlock(blockHeight uint64)
	PreBlockAccept(tx abci.Tx) (bool, error)
	PostBlockAccept(tx abci.Tx) (bool, error)
	UpdateUintParam(name string, value *num.Uint) error
	UpdateIntParam(name string, value int64) error
	Serialise() ([]byte, error)
	Deserialise(payload *types.Payload) error
}

// New instantiates a new spam engine
func New(log *logging.Logger, config Config, epochEngine EpochEngine, accounting Accounting) *Engine {
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())
	e := &Engine{
		config:                  config,
		log:                     log,
		accounting:              accounting,
		transactionTypeToPolicy: map[txn.Command]SpamPolicy{},
	}

	proposalPolicy := NewSimpleSpamPolicy("proposal", netparams.SpamProtectionMinTokensForProposal, netparams.SpamProtectionMaxProposals, log)
	delegationPolicy := NewSimpleSpamPolicy("delegation", netparams.SpamProtectionMinTokensForDelegation, netparams.SpamProtectionMaxDelegations, log)
	votePolicy := NewVoteSpamPolicy(netparams.SpamProtectionMinTokensForVoting, netparams.SpamProtectionMaxVotes, log)

	voteKey := (&types.PayloadDelegationActive{}).Key()
	e.policyNameToPolicy = map[string]SpamPolicy{voteKey: votePolicy, proposalPolicy.policyName: proposalPolicy, delegationPolicy.policyName: delegationPolicy}
	e.hashKeys = []string{voteKey, proposalPolicy.policyName, delegationPolicy.policyName}

	e.transactionTypeToPolicy[txn.ProposeCommand] = proposalPolicy
	e.transactionTypeToPolicy[txn.VoteCommand] = votePolicy
	e.transactionTypeToPolicy[txn.DelegateCommand] = delegationPolicy
	e.transactionTypeToPolicy[txn.UndelegateCommand] = delegationPolicy

	// register for epoch end notifications
	epochEngine.NotifyOnEpoch(e.OnEpochEvent)
	e.log.Info("Spam protection started")

	return e
}

// OnMaxDelegationsChanged is called when the net param for max delegations per epoch has changed
func (e *Engine) OnMaxDelegationsChanged(ctx context.Context, maxDelegations int64) error {
	return e.transactionTypeToPolicy[txn.DelegateCommand].UpdateIntParam(netparams.SpamProtectionMaxDelegations, maxDelegations)
}

// OnMinTokensForDelegationChanged is called when the net param for min tokens requirement for voting has changed
func (e *Engine) OnMinTokensForDelegationChanged(ctx context.Context, minTokens num.Decimal) error {
	minTokensFoDelegation, _ := num.UintFromDecimal(minTokens)
	return e.transactionTypeToPolicy[txn.DelegateCommand].UpdateUintParam(netparams.SpamProtectionMinTokensForDelegation, minTokensFoDelegation)
}

// OnMaxVotesChanged is called when the net param for max votes per epoch has changed
func (e *Engine) OnMaxVotesChanged(ctx context.Context, maxVotes int64) error {
	return e.transactionTypeToPolicy[txn.VoteCommand].UpdateIntParam(netparams.SpamProtectionMaxVotes, maxVotes)
}

// OnMinTokensForVotingChanged is called when the net param for min tokens requirement for voting has changed
func (e *Engine) OnMinTokensForVotingChanged(ctx context.Context, minTokens num.Decimal) error {
	minTokensForVoting, _ := num.UintFromDecimal(minTokens)
	return e.transactionTypeToPolicy[txn.VoteCommand].UpdateUintParam(netparams.SpamProtectionMinTokensForVoting, minTokensForVoting)
}

// OnMaxProposalsChanged is called when the net param for max proposals per epoch has changed
func (e *Engine) OnMaxProposalsChanged(ctx context.Context, maxProposals int64) error {
	return e.transactionTypeToPolicy[txn.ProposeCommand].UpdateIntParam(netparams.SpamProtectionMaxProposals, maxProposals)
}

// OnMinTokensForProposalChanged is called when the net param for min tokens requirement for submitting a proposal has changed
func (e *Engine) OnMinTokensForProposalChanged(ctx context.Context, minTokens num.Decimal) error {
	minTokensForProposal, _ := num.UintFromDecimal(minTokens)
	return e.transactionTypeToPolicy[txn.ProposeCommand].UpdateUintParam(netparams.SpamProtectionMinTokensForProposal, minTokensForProposal)
}

// OnEpochEvent is a callback for epoch events
func (e *Engine) OnEpochEvent(ctx context.Context, epoch types.Epoch) {
	e.log.Info("Spam protection OnEpochEvent called", logging.Uint64("epoch", epoch.Seq))
	if e.currentEpoch == nil || e.currentEpoch.Seq != epoch.Seq {
		if e.log.GetLevel() <= logging.DebugLevel {
			e.log.Debug("Spam protection new epoch started", logging.Uint64("epochSeq", epoch.Seq))
		}
		e.currentEpoch = &epoch
		balances := e.accounting.GetAllAvailableBalances()
		for _, policy := range e.transactionTypeToPolicy {
			policy.Reset(epoch, balances)
		}
	}
}

// EndOfBlock is called when the block is finished
func (e *Engine) EndOfBlock(blockHeight uint64) {
	if e.log.GetLevel() <= logging.DebugLevel {
		e.log.Debug("Spam protection EndOfBlock called", logging.Uint64("blockHeight", blockHeight))
	}
	for _, policy := range e.transactionTypeToPolicy {
		policy.EndOfBlock(blockHeight)
	}
}

// PreBlockAccept is called from onCheckTx before a tx is added to mempool
// returns false is rejected by spam engine with a corresponding error
func (e *Engine) PreBlockAccept(tx abci.Tx) (bool, error) {
	command := tx.Command()
	if _, ok := e.transactionTypeToPolicy[command]; !ok {
		return true, nil
	}
	if e.log.GetLevel() <= logging.DebugLevel {
		e.log.Debug("Spam protection PreBlockAccept called for policy", logging.String("command", string(command)))
	}
	return e.transactionTypeToPolicy[command].PreBlockAccept(tx)
}

// PostBlockAccept is called from onDeliverTx before the block is processed
// returns false is rejected by spam engine with a corresponding error
func (e *Engine) PostBlockAccept(tx abci.Tx) (bool, error) {
	command := tx.Command()
	if _, ok := e.transactionTypeToPolicy[command]; !ok {
		return true, nil
	}
	if e.log.GetLevel() <= logging.DebugLevel {
		e.log.Debug("Spam protection PostBlockAccept called for policy", logging.String("command", string(command)))
	}
	return e.transactionTypeToPolicy[command].PostBlockAccept(tx)
}
