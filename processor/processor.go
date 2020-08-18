package processor

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/governance"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallet"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

var (
	ErrUnknownCommand                               = errors.New("unknown command when validating payload")
	ErrInvalidSignature                             = errors.New("invalid signature")
	ErrOrderSubmissionPartyAndPubKeyDoesNotMatch    = errors.New("order submission party and pubkey does not match")
	ErrOrderCancellationPartyAndPubKeyDoesNotMatch  = errors.New("order cancellation party and pubkey does not match")
	ErrOrderAmendmentPartyAndPubKeyDoesNotMatch     = errors.New("order amendment party and pubkey does not match")
	ErrProposalSubmissionPartyAndPubKeyDoesNotMatch = errors.New("proposal submission party and pubkey does not match")
	ErrVoteSubmissionPartyAndPubKeyDoesNotMatch     = errors.New("vote submission party and pubkey does not match")
	ErrWithdrawPartyAndPublKeyDoesNotMatch          = errors.New("withdraw party and pubkey does not match")
	ErrNodeSignatureKeyDoesNotMatch                 = errors.New("node signature pubkey does not match")
	ErrCommandKindUnknown                           = errors.New("unknown command kind when validating payload")
	ErrUnknownNodeKey                               = errors.New("node pubkey unknown")
	ErrUnknownProposal                              = errors.New("proposal unknown")
	ErrNotAnAssetProposal                           = errors.New("proposal is not a new asset proposal")
	ErrNoVegaWalletFound                            = errors.New("node wallet not found")
	ErrAssetProposalReferenceDuplicate              = errors.New("duplicate asset proposal for reference")
	ErrRegisterNodePubKeyDoesNotMatch               = errors.New("node register key does not match")
	ErrProposalValidationTimestampInvalid           = errors.New("asset proposal validation timestamp invalid")
	ErrVegaWalletRequired                           = errors.New("vega wallet required")
	ErrProposalCorrupted                            = errors.New("proposal internal data corrupted")
	ErrChainEventFromNonValidator                   = errors.New("chain event emitted from a non-validator node")
	ErrUnsupportedChainEvent                        = errors.New("unsupprted chain event")
	ErrNotAnAssetListChainEvent                     = errors.New("not an asset list chain event")
	ErrNodeSignatureFromNonValidator                = errors.New("node signature not sent by validator")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/processor TimeService
type TimeService interface {
	GetTimeNow() (time.Time, error)
	GetTimeLastBatch() (time.Time, error)
	NotifyOnTick(f func(context.Context, time.Time))
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/execution_engine_mock.go -package mocks code.vegaprotocol.io/vega/processor ExecutionEngine
type ExecutionEngine interface {
	SubmitOrder(ctx context.Context, order *types.Order) (*types.OrderConfirmation, error)
	CancelOrder(ctx context.Context, order *types.OrderCancellation) (*types.OrderCancellationConfirmation, error)
	AmendOrder(ctx context.Context, order *types.OrderAmendment) (*types.OrderConfirmation, error)
	Generate() error
	SubmitMarket(ctx context.Context, marketConfig *types.Market) error
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/governance_engine_mock.go -package mocks code.vegaprotocol.io/vega/processor GovernanceEngine
type GovernanceEngine interface {
	SubmitProposal(context.Context, types.Proposal) error
	AddVote(context.Context, types.Vote) error
	OnChainTimeUpdate(context.Context, time.Time) []*governance.ToEnact
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/stats_mock.go -package mocks code.vegaprotocol.io/vega/processor Stats
type Stats interface {
	IncTotalCreateOrder()
	AddCurrentTradesInBatch(i uint64)
	AddTotalTrades(i uint64) uint64
	IncTotalOrders()
	IncCurrentOrdersInBatch()
	IncTotalCancelOrder()
	IncTotalAmendOrder()
	// batch stats
	IncTotalBatches()
	NewBatch()
	TotalOrders() uint64
	TotalBatches() uint64
	SetAverageOrdersPerBatch(i uint64)
	SetBlockDuration(uint64)
	CurrentOrdersInBatch() uint64
	CurrentTradesInBatch() uint64
	SetOrdersPerSecond(i uint64)
	SetTradesPerSecond(i uint64)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/wallet_mock.go -package mocks code.vegaprotocol.io/vega/processor Wallet
type Wallet interface {
	Get(chain nodewallet.Blockchain) (nodewallet.Wallet, bool)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/assets_mock.go -package mocks code.vegaprotocol.io/vega/processor Assets
type Assets interface {
	NewAsset(ref string, assetSrc *types.AssetSource) (string, error)
	Get(assetID string) (*assets.Asset, error)
	IsEnabled(string) bool
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/commander_mock.go -package mocks code.vegaprotocol.io/vega/processor Commander
type Commander interface {
	Command(cmd blockchain.Command, payload proto.Message) error
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/validator_topology_mock.go -package mocks code.vegaprotocol.io/vega/processor ValidatorTopology
type ValidatorTopology interface {
	AddNodeRegistration(nr *types.NodeRegistration) error
	SelfChainPubKey() []byte
	Ready() bool
	Exists(key []byte) bool
	Len() int
	AllPubKeys() [][]byte
	IsValidator() bool
}

// Broker - the event bus
//go:generate go run github.com/golang/mock/mockgen -destination mocks/broker_mock.go -package mocks code.vegaprotocol.io/vega/processor Broker
type Broker interface {
	Send(e events.Event)
}

// Notary ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/notary_mock.go -package mocks code.vegaprotocol.io/vega/processor Notary
type Notary interface {
	StartAggregate(resID string, kind types.NodeSignatureKind) error
	AddSig(ctx context.Context, pubKey []byte, ns types.NodeSignature) ([]types.NodeSignature, bool, error)
	IsSigned(string, types.NodeSignatureKind) ([]types.NodeSignature, bool)
}

// ExtResChecker ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/erc_mock.go -package mocks code.vegaprotocol.io/vega/processor ExtResChecker
type ExtResChecker interface {
	AddNodeCheck(ctx context.Context, nv *types.NodeVote) error
}

// EvtForwarder ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/evtforwarder_mock.go -package mocks code.vegaprotocol.io/vega/processor EvtForwarder
type EvtForwarder interface {
	Ack(*types.ChainEvent) bool
}

// Collateral ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/collateral_mock.go -package mocks code.vegaprotocol.io/vega/processor Collateral
type Collateral interface {
	Deposit(ctx context.Context, partyID, asset string, amount uint64) error
	Withdraw(ctx context.Context, partyID, asset string, amount uint64) error
	EnableAsset(ctx context.Context, asset types.Asset) error
}

// Banking ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/banking_mock.go -package mocks code.vegaprotocol.io/vega/processor Banking
type Banking interface {
	EnableBuiltinAsset(context.Context, string) error
	DepositBuiltinAsset(*types.BuiltinAssetDeposit, uint64) error
	WithdrawalBuiltinAsset(context.Context, string, string, uint64) error
	EnableERC20(context.Context, *types.ERC20AssetList, uint64, uint64) error
	DepositERC20(*types.ERC20Deposit, uint64, uint64) error
}

// Processor handle processing of all transaction sent through the node
type Processor struct {
	log *logging.Logger
	Config
	hasRegistered     bool
	stat              Stats
	exec              ExecutionEngine
	gov               GovernanceEngine
	time              TimeService
	wallet            Wallet
	vegaWallet        nodewallet.Wallet
	assets            Assets
	cmd               Commander
	currentTimestamp  time.Time
	previousTimestamp time.Time
	top               ValidatorTopology
	idgen             *IDgenerator
	broker            Broker
	notary            Notary
	evtfwd            EvtForwarder
	col               Collateral
	erc               ExtResChecker
	banking           Banking
}

// NewProcessor instantiates a new transactions processor
func New(log *logging.Logger, config Config, exec ExecutionEngine, ts TimeService, stat Stats, cmd Commander, wallet Wallet, assets Assets, top ValidatorTopology, gov GovernanceEngine, broker Broker, notary Notary, evtfwd EvtForwarder, col Collateral, erc ExtResChecker, banking Banking) (*Processor, error) {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	vegaWallet, ok := wallet.Get(nodewallet.Vega)
	if !ok {
		return nil, ErrVegaWalletRequired
	}

	p := &Processor{
		log:        log,
		stat:       stat,
		Config:     config,
		exec:       exec,
		time:       ts,
		wallet:     wallet,
		assets:     assets,
		cmd:        cmd,
		top:        top,
		vegaWallet: vegaWallet,
		gov:        gov,
		broker:     broker,
		idgen:      NewIDGen(),
		notary:     notary,
		evtfwd:     evtfwd,
		col:        col,
		erc:        erc,
		banking:    banking,
	}
	ts.NotifyOnTick(p.onTick)
	return p, nil
}

// Begin update timestamps
func (p *Processor) Begin() error {
	if p.log.GetLevel() == logging.DebugLevel {
		p.log.Debug("Processor service BEGIN starting")
	}
	var err error
	// Load the latest consensus block time
	if p.currentTimestamp, err = p.time.GetTimeNow(); err != nil {
		return err
	}

	if p.previousTimestamp, err = p.time.GetTimeLastBatch(); err != nil {
		return err
	}
	if !p.hasRegistered && p.top.IsValidator() && !p.top.Ready() {
		// get our tendermint pubkey
		chainPubKey := p.top.SelfChainPubKey()
		if chainPubKey != nil {
			payload := &types.NodeRegistration{
				ChainPubKey: chainPubKey,
				PubKey:      p.vegaWallet.PubKeyOrAddress(),
			}
			if err := p.cmd.Command(blockchain.RegisterNodeCommand, payload); err != nil {
				return err
			}
			p.hasRegistered = true
		}
	}

	if p.log.GetLevel() == logging.DebugLevel {
		p.log.Debug("ABCI service BEGIN completed",
			logging.Int64("current-timestamp", p.currentTimestamp.UnixNano()),
			logging.Int64("previous-timestamp", p.previousTimestamp.UnixNano()),
			logging.String("current-datetime", vegatime.Format(p.currentTimestamp)),
			logging.String("previous-datetime", vegatime.Format(p.previousTimestamp)),
		)
	}
	return nil
}

func (p *Processor) Commit() error {
	if p.log.GetLevel() == logging.DebugLevel {
		p.log.Debug("Processor COMMIT starting")
	}
	p.stats()
	if err := p.exec.Generate(); err != nil {
		return errors.Wrap(err, "failure generating data in execution engine (commit)")
	}
	if p.log.GetLevel() == logging.DebugLevel {
		p.log.Debug("Processor COMMIT completed")
	}
	return nil
}

func (p *Processor) stats() {
	p.stat.IncTotalBatches()
	avg := p.stat.TotalOrders() / p.stat.TotalBatches()
	p.stat.SetAverageOrdersPerBatch(avg)
	duration := time.Duration(p.currentTimestamp.UnixNano() - p.previousTimestamp.UnixNano()).Seconds()
	var (
		currentOrders, currentTrades uint64
	)
	p.stat.SetBlockDuration(uint64(duration * float64(time.Second.Nanoseconds())))
	if duration > 0 {
		currentOrders, currentTrades = uint64(float64(p.stat.CurrentOrdersInBatch())/duration),
			uint64(float64(p.stat.CurrentTradesInBatch())/duration)
	}
	p.stat.SetOrdersPerSecond(currentOrders)
	p.stat.SetTradesPerSecond(currentTrades)
	// log stats
	p.log.Debug("Processor batch stats",
		logging.Int64("previousTimestamp", p.previousTimestamp.UnixNano()),
		logging.Int64("currentTimestamp", p.currentTimestamp.UnixNano()),
		logging.Float64("duration", duration),
		logging.Uint64("currentOrdersInBatch", p.stat.CurrentOrdersInBatch()),
		logging.Uint64("currentTradesInBatch", p.stat.CurrentTradesInBatch()),
		logging.Uint64("total-batches", p.stat.TotalBatches()),
		logging.Uint64("avg-orders-batch", avg),
		logging.Uint64("orders-per-sec", currentOrders),
		logging.Uint64("trades-per-sec", currentTrades),
	)
	p.stat.NewBatch() // sets previous batch orders/trades to current, zeroes current tally
}

func (p *Processor) SetTime(now time.Time) {
	p.previousTimestamp = p.currentTimestamp
	p.currentTimestamp = now
}

// ReloadConf update the internal configuration of the processor
func (p *Processor) ReloadConf(cfg Config) {
	p.log.Info("reloading configuration")
	if p.log.GetLevel() != cfg.Level.Get() {
		p.log.Info("updating log level",
			logging.String("old", p.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		p.log.SetLevel(cfg.Level.Get())
	}

	p.Config = cfg
}

func (p *Processor) getOrderSubmission(payload []byte) (*types.Order, error) {
	orderSubmission := &types.OrderSubmission{}
	err := proto.Unmarshal(payload, orderSubmission)
	if err != nil {
		return nil, err
	}

	order := types.Order{
		Id:          orderSubmission.Id,
		MarketID:    orderSubmission.MarketID,
		PartyID:     orderSubmission.PartyID,
		Price:       orderSubmission.Price,
		Size:        orderSubmission.Size,
		Side:        orderSubmission.Side,
		TimeInForce: orderSubmission.TimeInForce,
		Type:        orderSubmission.Type,
		ExpiresAt:   orderSubmission.ExpiresAt,
		Reference:   orderSubmission.Reference,
		Status:      types.Order_STATUS_ACTIVE,
		CreatedAt:   0,
		Remaining:   orderSubmission.Size,
	}

	return &order, nil
}

func (p *Processor) getOrderCancellation(payload []byte) (*types.OrderCancellation, error) {
	order := &types.OrderCancellation{}
	err := proto.Unmarshal(payload, order)
	if err != nil {
		return nil, err
	}
	return order, nil
}

func (p *Processor) getOrderAmendment(payload []byte) (*types.OrderAmendment, error) {
	amendment := &types.OrderAmendment{}
	err := proto.Unmarshal(payload, amendment)
	if err != nil {
		return nil, errors.Wrap(err, "error decoding order to proto")
	}
	return amendment, nil
}

func (p *Processor) getWithdraw(payload []byte) (*types.Withdraw, error) {
	w := &types.Withdraw{}
	err := proto.Unmarshal(payload, w)
	if err != nil {
		return nil, errors.Wrap(err, "error decoding order to proto")
	}
	return w, nil
}

func (p *Processor) getProposalSubmission(payload []byte) (*types.Proposal, error) {
	proposalSubmission := &types.Proposal{}
	err := proto.Unmarshal(payload, proposalSubmission)
	if err != nil {
		return nil, err
	}
	return proposalSubmission, nil
}

func (p *Processor) getVoteSubmission(payload []byte) (*types.Vote, error) {
	voteSubmission := &types.Vote{}
	err := proto.Unmarshal(payload, voteSubmission)
	if err != nil {
		return nil, err
	}
	return voteSubmission, nil
}

func (p *Processor) getNodeSignature(payload []byte) (*types.NodeSignature, error) {
	ns := &types.NodeSignature{}
	err := proto.Unmarshal(payload, ns)
	if err != nil {
		return nil, err
	}
	return ns, nil
}

func (p *Processor) getChainEvent(payload []byte) (*types.ChainEvent, error) {
	ce := &types.ChainEvent{}
	err := proto.Unmarshal(payload, ce)
	if err != nil {
		return nil, err
	}
	return ce, nil
}

func (p *Processor) getNodeVote(payload []byte) (*types.NodeVote, error) {
	vote := &types.NodeVote{}
	if err := proto.Unmarshal(payload, vote); err != nil {
		return nil, err
	}
	return vote, nil
}

func (p *Processor) getNodeRegistration(payload []byte) (*types.NodeRegistration, error) {
	cmd := &types.NodeRegistration{}
	err := proto.Unmarshal(payload, cmd)
	if err != nil {
		return nil, err
	}
	return cmd, nil
}

// ValidateSigned - validates a signed transaction. This sits here because it's actual data processing
// related. We need to unmarshal the payload to validate the partyID
func (p *Processor) ValidateSigned(key, data []byte, cmd blockchain.Command) error {
	switch cmd {
	case blockchain.SubmitOrderCommand:
		order, err := p.getOrderSubmission(data)
		if err != nil {
			return err
		}
		// partyID is hex encoded pubkey
		if order.PartyID != hex.EncodeToString(key) {
			return ErrOrderSubmissionPartyAndPubKeyDoesNotMatch
		}
		return nil
	case blockchain.CancelOrderCommand:
		order, err := p.getOrderCancellation(data)
		if err != nil {
			return err
		}
		// partyID is hex encoded pubkey
		if order.PartyID != hex.EncodeToString(key) {
			return ErrOrderCancellationPartyAndPubKeyDoesNotMatch
		}
		return nil
	case blockchain.AmendOrderCommand:
		order, err := p.getOrderAmendment(data)
		if err != nil {
			return err
		}
		// partyID is hex encoded pubkey
		if order.PartyID != hex.EncodeToString(key) {
			return ErrOrderAmendmentPartyAndPubKeyDoesNotMatch
		}
		return nil
	case blockchain.ProposeCommand:
		proposal, err := p.getProposalSubmission(data)
		if err != nil {
			return err
		}
		// partyID is hex encoded pubkey
		if proposal.PartyID != hex.EncodeToString(key) {
			return ErrProposalSubmissionPartyAndPubKeyDoesNotMatch
		}
		return nil
	case blockchain.VoteCommand:
		vote, err := p.getVoteSubmission(data)
		if err != nil {
			return err
		}
		// partyID is hex encoded pubkey
		if vote.PartyID != hex.EncodeToString(key) {
			return ErrVoteSubmissionPartyAndPubKeyDoesNotMatch
		}
		return nil
	case blockchain.NodeVoteCommand:
		_, err := p.getNodeVote(data)
		if err != nil {
			return err
		}
		// partyID is hex encoded pubkey
		// if vote.PartyID != hex.EncodeToString(key) {
		// return ErrNodeVote
		// }
		return nil
	case blockchain.WithdrawCommand:
		withdraw, err := p.getWithdraw(data)
		if err != nil {
			return err
		}
		if withdraw.PartyID != hex.EncodeToString(key) {
			return ErrWithdrawPartyAndPublKeyDoesNotMatch
		}
		return nil
	case blockchain.RegisterNodeCommand:
		reg, err := p.getNodeRegistration(data)
		if err != nil {
			return err
		}
		if hex.EncodeToString(reg.PubKey) != hex.EncodeToString(key) {
			return ErrRegisterNodePubKeyDoesNotMatch
		}
		return nil
	case blockchain.NodeSignatureCommand:
		_, err := p.getNodeSignature(data)
		if err != nil {
			return err
		}
		// reject if the node signature is not coming from a validator
		if !p.top.Exists(key) {
			return ErrNodeSignatureFromNonValidator
		}
		return nil
	case blockchain.ChainEventCommand:
		_, err := p.getChainEvent(data)
		if err != nil {
			return err
		}
		// reject if the chain event is not coming from a validator
		if !p.top.Exists(key) {
			return ErrChainEventFromNonValidator
		}
		return nil
	}
	return errors.New("unknown command when validating payload")
}

// Process performs validation and then sends the command and data to
// the underlying blockchain service handlers e.g. submit order, etc.
func (p *Processor) Process(ctx context.Context, data []byte, pubkey []byte, cmd blockchain.Command) error {
	// first is that a signed or unsigned command?
	switch cmd {
	case blockchain.SubmitOrderCommand:
		order, err := p.getOrderSubmission(data)
		if err != nil {
			return err
		}
		err = p.submitOrder(ctx, order)
	case blockchain.CancelOrderCommand:
		order, err := p.getOrderCancellation(data)
		if err != nil {
			return err
		}
		return p.cancelOrder(ctx, order)
	case blockchain.AmendOrderCommand:
		order, err := p.getOrderAmendment(data)
		if err != nil {
			return err
		}
		_ = order
		return errors.New("not implemented")
		// return p.amendOrder(ctx, order)
	case blockchain.WithdrawCommand:
		withdraw, err := p.getWithdraw(data)
		if err != nil {
			return err
		}
		return p.processWithdraw(ctx, withdraw)
	case blockchain.ProposeCommand:
		proposal, err := p.getProposalSubmission(data)
		if err != nil {
			return err
		}
		return p.SubmitProposal(ctx, proposal)
	case blockchain.VoteCommand:
		vote, err := p.getVoteSubmission(data)
		if err != nil {
			return err
		}
		return p.VoteOnProposal(ctx, vote)
	case blockchain.RegisterNodeCommand:
		node, err := p.getNodeRegistration(data)
		if err != nil {
			return err
		}
		err = p.top.AddNodeRegistration(node)
		if err != nil {
			p.log.Warn("unable to register node",
				logging.Error(err))
		}
	case blockchain.NodeVoteCommand:
		vote, err := p.getNodeVote(data)
		if err != nil {
			return err
		}
		return p.erc.AddNodeCheck(ctx, vote)
	case blockchain.NodeSignatureCommand:
		ns, err := p.getNodeSignature(data)
		if err != nil {
			return err
		}
		_, _, err = p.notary.AddSig(ctx, pubkey, *ns)
		return err
	case blockchain.ChainEventCommand:
		ce, err := p.getChainEvent(data)
		if err != nil {
			return err
		}
		return p.processChainEvent(ctx, ce, pubkey)
	default:
		p.log.Warn("Unknown command received", logging.String("command", cmd.String()))
		return fmt.Errorf("unknown command received: %s", cmd)
	}
	return nil
}

func (p *Processor) submitOrder(ctx context.Context, o *types.Order) error {
	p.stat.IncTotalCreateOrder()
	if p.log.GetLevel() == logging.DebugLevel {
		p.log.Debug("Processor received a SUBMIT ORDER request", logging.Order(*o))
	}

	o.CreatedAt = p.currentTimestamp.UnixNano()

	// Submit the create order request to the execution engine
	conf, err := p.exec.SubmitOrder(ctx, o)
	if conf != nil {

		if p.log.GetLevel() == logging.DebugLevel {
			p.log.Debug("Order confirmed",
				logging.Order(*o),
				logging.OrderWithTag(*conf.Order, "aggressive-order"),
				logging.String("passive-trades", fmt.Sprintf("%+v", conf.Trades)),
				logging.String("passive-orders", fmt.Sprintf("%+v", conf.PassiveOrdersAffected)))
		}
		p.stat.AddCurrentTradesInBatch(uint64(len(conf.Trades)))
		p.stat.AddTotalTrades(uint64(len(conf.Trades)))
		p.stat.IncCurrentOrdersInBatch()
	}

	// increment total orders, even for failures so current ID strategy is valid.
	p.stat.IncTotalOrders()

	if err != nil {
		p.log.Error("error message on creating order",
			logging.Order(*o),
			logging.Error(err))
	}

	return err
}

func (p *Processor) cancelOrder(ctx context.Context, order *types.OrderCancellation) error {
	p.stat.IncTotalCancelOrder()
	if p.log.GetLevel() == logging.DebugLevel {
		p.log.Debug("Blockchain service received a CANCEL ORDER request", logging.String("order-id", order.OrderID))
	}

	// Submit the cancel new order request to the Vega trading core
	msg, err := p.exec.CancelOrder(ctx, order)
	if err != nil {
		p.log.Error("error on cancelling order",
			logging.String("order-id", order.OrderID),
			logging.Error(err),
		)
		return err
	}
	if p.LogOrderCancelDebug {
		p.log.Debug("Order cancelled", logging.Order(*msg.Order))
	}

	return nil
}

func (p *Processor) amendOrder(ctx context.Context, order *types.OrderAmendment) error {
	p.stat.IncTotalAmendOrder()
	if p.log.GetLevel() == logging.DebugLevel {
		p.log.Debug("Blockchain service received a AMEND ORDER request",
			logging.String("order", order.String()))
	}

	// Submit the Amendment new order request to the Vega trading core
	_, err := p.exec.AmendOrder(ctx, order)
	if err != nil {
		p.log.Error("Error amending order",
			logging.String("order", order.String()),
			logging.Error(err),
		)
		return err
	}
	if p.LogOrderAmendDebug {
		p.log.Debug("Order amended", logging.String("order", order.String()))
	}
	return nil
}

// SubmitProposal generates and assigns new id for given proposal and sends it to governance engine
func (p *Processor) SubmitProposal(ctx context.Context, proposal *types.Proposal) error {
	if p.log.GetLevel() == logging.DebugLevel {
		p.log.Debug("Submitting proposal",
			logging.String("proposal-id", proposal.ID),
			logging.String("proposal-reference", proposal.Reference),
			logging.String("proposal-party", proposal.PartyID),
			logging.String("proposal-terms", proposal.Terms.String()))
	}
	// TODO(JEREMY): use hash of the signature here.
	p.idgen.SetProposalID(proposal)
	proposal.Timestamp = p.currentTimestamp.UnixNano()
	return p.gov.SubmitProposal(ctx, *proposal)
}

// VoteOnProposal sends proposal vote to governance engine
func (p *Processor) VoteOnProposal(ctx context.Context, vote *types.Vote) error {
	if p.log.GetLevel() == logging.DebugLevel {
		p.log.Debug("Voting on proposal",
			logging.String("proposal-id", vote.ProposalID),
			logging.String("vote-party", vote.PartyID),
			logging.String("vote-value", vote.Value.String()))
	}
	vote.Timestamp = p.currentTimestamp.UnixNano()
	return p.gov.AddVote(ctx, *vote)
}

func (p *Processor) enactMarket(ctx context.Context, prop *types.Proposal, mkt *types.Market) {
	prop.State = types.Proposal_STATE_ENACTED
	if err := p.exec.SubmitMarket(ctx, mkt); err != nil {
		prop.State = types.Proposal_STATE_FAILED
		p.log.Error("failed to submit new market",
			logging.String("market-id", mkt.Id),
			logging.Error(err))
	}
}

func (p *Processor) enactAsset(ctx context.Context, prop *types.Proposal, _ *types.Asset) {
	prop.State = types.Proposal_STATE_ENACTED
	// first check if this asset is real
	asset, err := p.assets.Get(prop.ID)
	if err != nil {
		// this should not happen
		p.log.Error("invalid asset is getting enacted",
			logging.String("asset-id", prop.ID),
			logging.Error(err))
		prop.State = types.Proposal_STATE_FAILED
		return
	}

	// if this is a builtin asset nothing needs to be done, just start the asset
	// straigh away
	if asset.IsBuiltinAsset() {
		err = p.banking.EnableBuiltinAsset(ctx, asset.ProtoAsset().ID)
		if err != nil {
			// this should not happen
			p.log.Error("unable to get builtin asset enabled",
				logging.String("asset-id", prop.ID),
				logging.Error(err))
			prop.State = types.Proposal_STATE_FAILED
		}
		return
	}

	// then instruct the notary to start getting signature from validators
	if err := p.notary.StartAggregate(prop.ID, types.NodeSignatureKind_NODE_SIGNATURE_KIND_ASSET_NEW); err != nil {
		prop.State = types.Proposal_STATE_FAILED
		p.log.Error("unable to enact proposal",
			logging.String("proposal-id", prop.ID),
			logging.Error(err))
		return
	}

	// if we are not a validator the job is done here
	if !p.top.IsValidator() {
		// nothing to do
		return
	}

	var sig []byte
	switch {
	case asset.IsERC20():
		asset, _ := asset.ERC20()
		_, sig, err = asset.SignBridgeWhitelisting()
	}
	if err != nil {
		p.log.Error("unable to sign whitelisting transaction",
			logging.String("asset-id", prop.ID),
			logging.Error(err))
		prop.State = types.Proposal_STATE_FAILED
		return
	}
	payload := &types.NodeSignature{
		ID:   prop.ID,
		Sig:  sig,
		Kind: types.NodeSignatureKind_NODE_SIGNATURE_KIND_ASSET_NEW,
	}
	if err := p.cmd.Command(blockchain.NodeSignatureCommand, payload); err != nil {
		// do nothing for now, we'll need a retry mechanism for this and all command soon
		p.log.Error("unable to send command for notary",
			logging.Error(err))
	}
}

// check the asset proposals on tick
func (p *Processor) onTick(ctx context.Context, t time.Time) {
	p.idgen.NewBatch()
	acceptedProposals := p.gov.OnChainTimeUpdate(ctx, t)
	for _, toEnact := range acceptedProposals {
		prop := toEnact.Proposal()
		switch {
		case toEnact.IsNewMarket():
			p.enactMarket(ctx, prop, toEnact.NewMarket())
		case toEnact.IsNewAsset():
			p.enactAsset(ctx, prop, toEnact.NewAsset())
		case toEnact.IsUpdateMarket():
			p.log.Error("update market enactment is not implemented")
		case toEnact.IsUpdateNetwork():
			p.log.Error("update network enactment is not implemented")
		default:
			prop.State = types.Proposal_STATE_FAILED
			p.log.Error("unknown proposal cannot be enacted", logging.String("proposal-id", prop.ID))
		}
		p.broker.Send(events.NewProposalEvent(ctx, *prop))
	}
}
