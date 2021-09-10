package spam

import (
	"context"
	"errors"

	"code.vegaprotocol.io/vega/blockchain/abci"
	"code.vegaprotocol.io/vega/types/num"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/txn"
	"code.vegaprotocol.io/vega/types"
)

var (
	//ErrPartyIsBannedFromVoting is returned when the party is banned from voting
	ErrPartyIsBannedFromVoting = errors.New("party is banned from submitting votes in the current epoch")
	//ErrPartyIsBannedFromProposal is returned when the party is banned from proposals
	ErrPartyIsBannedFromProposal = errors.New("party is banned from submitting proposals in the current epoch")
	//ErrInsufficientTokensForVoting is returned when the party has insufficient tokens for voting
	ErrInsufficientTokensForVoting = errors.New("party has insufficient tokens to submit votes in this epoch")
	//ErrInsufficientTokensForProposal is returned when the party has insufficient tokens for proposal
	ErrInsufficientTokensForProposal = errors.New("party has insufficient tokens to submit proposals in this epoch")
	//ErrTooManyVotes is returned when the party has voted already the maximum allowed votes per proposal per epoch
	ErrTooManyVotes = errors.New("party has already voted the maximum number of times per proposal per epoch")
	//ErrTooManyProposals is returned when the party has proposed the maximum allowed proposals per epoch
	ErrTooManyProposals = errors.New("party has already proposed the maximum number of proposals per epoch")
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
}

type SpamPolicy interface {
	Reset(epoch types.Epoch, tokenBalances map[string]*num.Uint)
	EndOfBlock(blockHeight uint64)
	PreBlockAccept(tx abci.Tx) (bool, error)
	PostBlockAccept(tx abci.Tx) (bool, error)
}

//New instantiates a new spam engine
func New(log *logging.Logger, config Config, epochEngine EpochEngine, accounting Accounting) *Engine {
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())
	e := &Engine{
		config:                  config,
		log:                     log.Named(namedLogger),
		accounting:              accounting,
		transactionTypeToPolicy: map[txn.Command]SpamPolicy{},
	}

	proposalPolicy := NewProposalSpamPolicy()
	votePolicy := NewVoteSpamPolicy()
	e.transactionTypeToPolicy[txn.ProposeCommand] = proposalPolicy
	e.transactionTypeToPolicy[txn.VoteCommand] = votePolicy

	// register for epoch end notifications
	epochEngine.NotifyOnEpoch(e.OnEpochEvent)
	return e
}

//OnEpochEvent is a callback for epoch events
func (e *Engine) OnEpochEvent(ctx context.Context, epoch types.Epoch) {
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

//EndOfBlock is called when the block is finished
func (e *Engine) EndOfBlock(blockHeight uint64) {
	if e.log.GetLevel() <= logging.DebugLevel {
		e.log.Debug("Spam protection EndOfBlock called", logging.Uint64("blockHeight", blockHeight))
	}
	for _, policy := range e.transactionTypeToPolicy {
		policy.EndOfBlock(blockHeight)
	}
}

//PreBlockAccept is called from onCheckTx before a tx is added to mempool
//returns false is rejected by spam engine with a corresponding error
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

//PostBlockAccept is called from onDeliverTx before the block is processed
//returns false is rejected by spam engine with a corresponding error
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
