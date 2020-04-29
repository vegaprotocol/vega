package processor

import (
	"context"
	"encoding/hex"
	"fmt"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallet"
	"code.vegaprotocol.io/vega/notary"
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
	ErrCommandKindUnknown                           = errors.New("unknown command kind when validating payload")
	ErrUnknownNodeKey                               = errors.New("node pubkey unknown")
	ErrUnknownProposal                              = errors.New("proposal unknown")
	ErrNotAnAssetProposal                           = errors.New("proposal is not a new asset proposal")
	ErrNoVegaWalletFound                            = errors.New("node wallet not found")
	ErrAssetProposalReferenceDuplicate              = errors.New("duplicate asset proposal for reference")
	ErrRegisterNodePubKeyDoesNotMatch               = errors.New("node register key does not match")
	ErrProposalValidationTimestampInvalid           = errors.New("asset proposal validation timestamp invalid")
	ErrUnknownSignatureKind                         = errors.New("unknown signature kind")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/processor TimeService
type TimeService interface {
	GetTimeNow() (time.Time, error)
	GetTimeLastBatch() (time.Time, error)
	NotifyOnTick(f func(time.Time))
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/execution_engine_mock.go -package mocks code.vegaprotocol.io/vega/processor ExecutionEngine
type ExecutionEngine interface {
	SubmitOrder(order *types.Order) (*types.OrderConfirmation, error)
	CancelOrder(order *types.OrderCancellation) (*types.OrderCancellationConfirmation, error)
	AmendOrder(order *types.OrderAmendment) (*types.OrderConfirmation, error)
	NotifyTraderAccount(notif *types.NotifyTraderAccount) error
	Withdraw(*types.Withdraw) error
	Generate() error
	EnactProposal(*types.Proposal) error
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/governance_engine_mock.go -package mocks code.vegaprotocol.io/vega/processor GovernanceEngine
type GovernanceEngine interface {
	OnChainTimeUpdate(t time.Time) []*types.Proposal
	AddProposal(p types.Proposal) error
	AddVote(v types.Vote) error
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
	Get(assetID string) (assets.Asset, error)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/commander_mock.go -package mocks code.vegaprotocol.io/vega/processor Commander
type Commander interface {
	Command(key nodewallet.Wallet, cmd blockchain.Command, payload proto.Message) error
}

// ProposalBuf...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/proposal_buf_mock.go -package mocks code.vegaprotocol.io/vega/processor ProposalBuf
type ProposalBuf interface {
	Add(types.Proposal)
	Flush()
}

// VoteBuf...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/vote_buf_mock.go -package mocks code.vegaprotocol.io/vega/processor VoteBuf
type VoteBuf interface {
	Add(types.Vote)
	Flush()
}

// NodeSigsBuf...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/node_sigs_buf_mock.go -package mocks code.vegaprotocol.io/vega/processor NodeSigsBuf
type NodeSigsBuf interface {
	Add([]types.NodeSignature)
	Flush()
}

const (
	notValidAssetProposal uint32 = iota
	validAssetProposal
	voteSentAssetProposal
)

type nodeProposal struct {
	*types.Proposal
	votes     map[string]struct{}
	validTime time.Time
	assetID   string
	// use for the node internal validation
	validState uint32
	cancel     func()
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
	assets            Assets
	nodes             map[string]struct{} // all other nodes in the network
	nodeProposals     map[string]*nodeProposal
	pendingValidation []*types.Proposal
	cmd               Commander
	currentTimestamp  time.Time
	previousTimestamp time.Time
	idgen             *IDgenerator

	proposalBuf ProposalBuf
	voteBuf     VoteBuf
	nodeSigBuf  NodeSigsBuf

	notary *notary.Notary
}

// NewProcessor instantiates a new transactions processor
func New(log *logging.Logger, config Config, exec ExecutionEngine, ts TimeService, stat Stats, cmd Commander, wallet Wallet, assets Assets, gov GovernanceEngine, proposalBuf ProposalBuf, voteBuf VoteBuf, notry *notary.Notary, nodeSigsBuf NodeSigsBuf) *Processor {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	p := &Processor{
		log:               log,
		stat:              stat,
		Config:            config,
		exec:              exec,
		gov:               gov,
		time:              ts,
		wallet:            wallet,
		assets:            assets,
		nodes:             map[string]struct{}{},
		nodeProposals:     map[string]*nodeProposal{},
		pendingValidation: []*types.Proposal{},
		cmd:               cmd,
		proposalBuf:       proposalBuf,
		voteBuf:           voteBuf,
		idgen:             NewIDGen(),
		notary:            notry,
		nodeSigBuf:        nodeSigsBuf,
	}
	ts.NotifyOnTick(p.onTick)
	return p
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
	if !p.hasRegistered {
		w, ok := p.wallet.Get(nodewallet.Vega)
		if !ok {
			return ErrNoVegaWalletFound
		}
		payload := &types.NodeRegistration{
			PubKey: w.PubKeyOrAddress(),
		}
		if err := p.cmd.Command(w, blockchain.RegisterNodeCommand, payload); err != nil {
			return err
		}
		p.hasRegistered = true
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

func (p *Processor) getOrder(payload []byte) (*types.Order, error) {
	order := &types.Order{}
	if err := proto.Unmarshal(payload, order); err != nil {
		return nil, err
	}
	return order, nil
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
		Status:      types.Order_Active,
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

func (p *Processor) getNotifyTraderAccount(payload []byte) (*types.NotifyTraderAccount, error) {
	notif := &types.NotifyTraderAccount{}
	err := proto.Unmarshal(payload, notif)
	if err != nil {
		return nil, errors.Wrap(err, "error decoding order to proto")
	}
	return notif, nil
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
	nodeSignature := &types.NodeSignature{}
	err := proto.Unmarshal(payload, nodeSignature)
	if err != nil {
		return nil, err
	}
	return nodeSignature, nil
}

// ValidateSigned - validates a signed transaction. This sits here because it's actual data processing
// related. We need to unmarshal the payload to validate the partyID
func (p *Processor) ValidateSigned(key, data, sig []byte, cmd blockchain.Command) error {
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
		return nil
	}
	return errors.New("unknown command when validating payload")
}

// Process performs validation and then sends the command and data to
// the underlying blockchain service handlers e.g. submit order, etc.
func (p *Processor) Process(key, data, sig []byte, cmd blockchain.Command) error {
	// first is that a signed or unsigned command?
	switch cmd {
	case blockchain.SubmitOrderCommand:
		order, err := p.getOrderSubmission(data)
		if err != nil {
			return err
		}
		err = p.submitOrder(order)
	case blockchain.CancelOrderCommand:
		order, err := p.getOrderCancellation(data)
		if err != nil {
			return err
		}
		return p.cancelOrder(order)
	case blockchain.AmendOrderCommand:
		order, err := p.getOrderAmendment(data)
		if err != nil {
			return err
		}
		return p.amendOrder(order)
	case blockchain.WithdrawCommand:
		withdraw, err := p.getWithdraw(data)
		if err != nil {
			return err
		}
		return p.exec.Withdraw(withdraw)
	case blockchain.ProposeCommand:
		proposal, err := p.getProposalSubmission(data)
		if err != nil {
			return err
		}
		// proposal is a new asset proposal?

		if na := proposal.Terms.GetNewAsset(); na != nil {
			return p.startAssetNodeProposal(proposal)
		}
		return p.SubmitProposal(proposal)
	case blockchain.VoteCommand:
		vote, err := p.getVoteSubmission(data)
		if err != nil {
			return err
		}
		return p.VoteOnProposal(vote)
	case blockchain.RegisterNodeCommand:
		node, err := p.getNodeRegistration(data)
		if err != nil {
			return err
		}
		p.nodes[hex.EncodeToString(node.PubKey)] = struct{}{}
	case blockchain.NodeVoteCommand:
		vote, err := p.getNodeVote(data)
		if err != nil {
			return err
		}
		pubKey := hex.EncodeToString(vote.PubKey)
		if _, ok := p.nodes[pubKey]; !ok {
			return ErrUnknownNodeKey
		}
		prop, ok := p.nodeProposals[vote.Reference]
		if !ok {
			return ErrUnknownProposal
		}
		prop.votes[pubKey] = struct{}{}
	case blockchain.NotifyTraderAccountCommand:
		notify, err := p.getNotifyTraderAccount(data)
		if err != nil {
			return err
		}
		return p.exec.NotifyTraderAccount(notify)
	case blockchain.NodeSignatureCommand:
		ns, err := p.getNodeSignature(data)
		if err != nil {
			return err
		}

		return p.registerNodeSignature(key, ns)
	default:
		p.log.Warn("Unknown command received", logging.String("command", cmd.String()))
		return fmt.Errorf("unknown command received: %s", cmd)
	}
	return nil
}

func (p *Processor) registerNodeSignature(nodePubKey []byte, ns *types.NodeSignature) (err error) {
	sigs, ok, err := p.notary.AddSig(ns.ID, ns.Kind, nodePubKey, ns.Sig)
	if err != nil {
		return err
	}

	// nothing to do
	if !ok {
		return err
	}

	defer func() {
		if err == nil && len(sigs) > 0 {
			p.nodeSigBuf.Add(sigs)
		}
	}()
	// signature are OK, move on to apply changes or propagat
	// down to the exec engine
	switch sigs[0].Kind {
	case types.NodeSignatureKind_ASSET_NEW:
		// actually nothing to here apart from making sure
		// the sig are sent to the buf and made available to the users.
		return nil
	case types.NodeSignatureKind_ASSET_WITHDRAWAL:
		// TODO
		return nil
	default:
		// this should most likely not happen but..
		return ErrUnknownSignatureKind
	}
}

func (p *Processor) startAssetNodeProposal(proposal *types.Proposal) error {
	asset := proposal.Terms.GetNewAsset()
	if asset == nil {
		p.log.Error("not an asset proposal", logging.String("ref", proposal.Reference))
		return ErrNotAnAssetProposal
	}

	_, ok := p.nodeProposals[proposal.Reference]
	if ok {
		return ErrAssetProposalReferenceDuplicate
	}
	if err := p.checkAssetProposal(proposal); err != nil {
		return err
	}

	assetID, err := p.assets.NewAsset(proposal.Reference,
		proposal.Terms.GetNewAsset().GetChanges())
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	// @TODO check valid timestamps
	np := &nodeProposal{
		Proposal:   proposal,
		votes:      map[string]struct{}{},
		validTime:  time.Unix(proposal.Terms.ValidationTimestamp, 0),
		validState: notValidAssetProposal,
		cancel:     cancel,
		assetID:    assetID,
	}
	p.nodeProposals[proposal.Reference] = np
	// start asset validation
	go p.validateAsset(ctx, np, proposal)

	return nil
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

func (p *Processor) submitOrder(o *types.Order) error {
	p.stat.IncTotalCreateOrder()
	if p.log.GetLevel() == logging.DebugLevel {
		p.log.Debug("Processor received a SUBMIT ORDER request", logging.Order(*o))
	}

	o.CreatedAt = p.currentTimestamp.UnixNano()

	// Submit the create order request to the execution engine
	conf, err := p.exec.SubmitOrder(o)
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

func (p *Processor) cancelOrder(order *types.OrderCancellation) error {
	p.stat.IncTotalCancelOrder()
	if p.log.GetLevel() == logging.DebugLevel {
		p.log.Debug("Blockchain service received a CANCEL ORDER request", logging.String("order-id", order.OrderID))
	}

	// Submit the cancel new order request to the Vega trading core
	msg, err := p.exec.CancelOrder(order)
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

func (p *Processor) amendOrder(order *types.OrderAmendment) error {
	p.stat.IncTotalAmendOrder()
	if p.log.GetLevel() == logging.DebugLevel {
		p.log.Debug("Blockchain service received a AMEND ORDER request",
			logging.String("order", order.String()))
	}

	// Submit the Amendment new order request to the Vega trading core
	_, err := p.exec.AmendOrder(order)
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

func (p *Processor) checkAssetProposal(prop *types.Proposal) error {
	asset := prop.Terms.GetNewAsset()
	// only validate timestamps for new asset proposal
	if asset == nil {
		return nil
	}
	if prop.Terms.ClosingTimestamp < prop.Terms.ValidationTimestamp {
		return ErrProposalValidationTimestampInvalid
	}
	minValid, maxValid := p.currentTimestamp.Add(minValidationPeriod*time.Second), p.currentTimestamp.Add(maxValidationPeriod*time.Second)
	if prop.Terms.ValidationTimestamp < minValid.Unix() || prop.Terms.ValidationTimestamp > maxValid.Unix() {
		return ErrProposalValidationTimestampInvalid
	}
	return nil
}

func (p *Processor) validateAsset(ctx context.Context, np *nodeProposal, prop *types.Proposal) {

	// get the asset to validate from the assets pool
	asset, err := p.assets.Get(np.assetID)
	if err != nil {
		p.log.Error("Validating asset, unable to get the asset",
			logging.String("ref", prop.GetTerms().String()),
			logging.Error(err),
		)
		return
	}

	// wait time between call to validation
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		// first try to validate the asset
		p.log.Debug("Validating asset",
			logging.String("asset-source", prop.GetTerms().String()),
		)
		if asset == nil {

		}
		if err != nil {
			p.log.Error("error validating asset", logging.Error(err))

		} else {
			if asset.IsValid() {
				atomic.StoreUint32(&np.validState, validAssetProposal)
				return
			}
		}

		// wait or break if the time's up
		select {
		case <-ctx.Done():
			p.log.Error("asset validation context error", logging.Error(ctx.Err()))
			return
		case _ = <-ticker.C:
		}
	}
}

// check the asset proposals on tick
func (p *Processor) onTick(t time.Time) {
	vegaKey, ok := p.wallet.Get(nodewallet.Vega)
	if !ok {
		p.log.Error("no vega wallet found")
		return
	}
	for k, prop := range p.nodeProposals {
		// this proposal has passed the node-voting period, or all nodes have voted/approved
		// time expired, or all vote agregated, and own vote sent
		state := atomic.LoadUint32(&prop.validState)
		if prop.validTime.Before(t) || (len(prop.votes) == len(p.nodes) && state == voteSentAssetProposal) {
			// if not all nodes have approved, just remove
			if len(prop.votes) < len(p.nodes) {
				p.log.Warn("proposal was not accepted by all nodes",
					logging.String("proposal", prop.Proposal.String()),
					logging.Int("vote-count", len(prop.votes)),
					logging.Int("node-count", len(p.nodes)),
				)
			} else if err := p.SubmitProposal(prop.Proposal); err != nil {
				p.log.Error("Failed to submit node-approved proposal",
					logging.String("proposal", prop.Proposal.String()),
				)
				continue // try again next block
			}
			// either proposal wasn't accepted, or it's been passed on to governance
			delete(p.nodeProposals, k)
			// cancelling this but it should already be exited if th proposal
			// was valid
			prop.cancel()
		}

		// or check if the proposal if valid,
		// if it is, we will send our own message through the network.
		if state == validAssetProposal {
			nv := &types.NodeVote{
				PubKey:    vegaKey.PubKeyOrAddress(),
				Reference: prop.Reference,
			}
			if err := p.cmd.Command(vegaKey, blockchain.NodeVoteCommand, nv); err != nil {
				p.log.Error("unable tosend command", logging.Error(err))
				// @TODO keep in memory, retry later?
				continue
			}
			atomic.StoreUint32(&prop.validState, voteSentAssetProposal)
			// cancelling this but it should already be exited if th proposal
			// was valid
			prop.cancel()
		}
	}

	// then run proposal through governance
	acceptedProposals := p.gov.OnChainTimeUpdate(t)
	for _, proposal := range acceptedProposals {
		// is asset prposal we need to start aggregating
		asset := proposal.Terms.GetNewAsset()
		if asset != nil {
			err := p.assetEnactFirstStep(proposal)
			if err != nil {
				continue
			}
		} else {
			if err := p.exec.EnactProposal(proposal); err != nil {
				proposal.State = types.Proposal_FAILED
				p.log.Error("unable to enact proposal",
					logging.String("proposal-id", proposal.ID),
					logging.Error(err))
			}
		}
		p.proposalBuf.Add(*proposal)
	}

	// governance buffers
	p.proposalBuf.Flush()
	p.voteBuf.Flush()

}

func (p *Processor) assetEnactFirstStep(proposal *types.Proposal) error {
	vegaKey, ok := p.wallet.Get(nodewallet.Vega)
	if !ok {
		p.log.Error("no vega wallet found")
		return errors.New("no vega wallet found")
	}
	// this will be enacted later on, once the next steps of it are done
	p.notary.StartAggregate(proposal.ID, types.NodeSignatureKind_ASSET_NEW)
	// then send the command to the

	asset, err := p.assets.Get(proposal.ID)
	if err != nil {
		p.log.Error("invalid asset", logging.String("id", proposal.ID), logging.Error(err))
		return err
	}

	_, sig, err := asset.SignBridgeWhitelisting()
	if err != nil {
		p.log.Error("unable to sign bridge whitelisting command", logging.String("id", proposal.ID), logging.Error(err))
		return err
	}

	ns := &types.NodeSignature{
		ID:   proposal.ID,
		Sig:  sig,
		Kind: types.NodeSignatureKind_ASSET_NEW,
	}
	if err := p.cmd.Command(vegaKey, blockchain.NodeVoteCommand, ns); err != nil {
		p.log.Error("unable to send command", logging.Error(err))
		// @TODO keep in memory, retry later?
		return err
	}

	// TODO(jeremy): once we receive the event from the bridge,
	// we'll need to enact this proposal
	return nil
}

// SubmitProposal generates and assigns new id for given proposal and sends it to governance engine
func (p *Processor) SubmitProposal(proposal *types.Proposal) error {
	if p.log.GetLevel() == logging.DebugLevel {
		p.log.Debug("Submitting proposal",
			logging.String("proposal-id", proposal.ID),
			logging.String("proposal-reference", proposal.Reference),
			logging.String("proposal-party", proposal.PartyID),
			logging.String("proposal-terms", proposal.Terms.String()))
	}
	p.idgen.SetProposalID(proposal)
	return p.gov.AddProposal(*proposal)
}

// VoteOnProposal sends proposal vote to governance engine
func (p *Processor) VoteOnProposal(vote *types.Vote) error {
	if p.log.GetLevel() == logging.DebugLevel {
		p.log.Debug("Voting on proposal",
			logging.String("proposal-id", vote.ProposalID),
			logging.String("vote-party", vote.PartyID),
			logging.String("vote-value", vote.Value.String()))
	}
	return p.gov.AddVote(*vote)
}
