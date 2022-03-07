package notary

import (
	"context"
	"strings"
	"sync"
	"time"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/txn"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/cenkalti/backoff"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

var (
	// by default all validators needs to sign.
	defaultValidatorsVoteRequired = num.MustDecimalFromString("1.0")
	oneDec                        = num.MustDecimalFromString("1")
)

var (
	ErrAggregateSigAlreadyStartedForResource = errors.New("aggregate signature already started for resource")
	ErrUnknownResourceID                     = errors.New("unknown resource ID")
	ErrNotAValidatorSignature                = errors.New("not a validator signature")
)

// ValidatorTopology...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/validator_topology_mock.go -package mocks code.vegaprotocol.io/vega/notary ValidatorTopology
type ValidatorTopology interface {
	IsValidator() bool
	IsValidatorVegaPubKey(string) bool
	SelfVegaPubKey() string
	Len() int
}

// Broker needs no mocks.
type Broker interface {
	Send(event events.Event)
	SendBatch(events []events.Event)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/commander_mock.go -package mocks code.vegaprotocol.io/vega/notary Commander
type Commander interface {
	Command(ctx context.Context, cmd txn.Command, payload proto.Message, f func(error), bo *backoff.ExponentialBackOff)
	CommandSync(ctx context.Context, cmd txn.Command, payload proto.Message, f func(error), bo *backoff.ExponentialBackOff)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_ticker_mock.go -package mocks code.vegaprotocol.io/vega/notary TimeTicker
type TimeTicker interface {
	NotifyOnTick(func(context.Context, time.Time))
}

// Notary will aggregate all signatures of a node for
// a specific Command
// e.g: asset withdrawal, asset allowlisting, etc.
type Notary struct {
	cfg Config
	log *logging.Logger

	// resource to be signed -> signatures
	sigs    map[idKind]map[nodeSig]struct{}
	retries *txTracker
	top     ValidatorTopology
	cmd     Commander
	broker  Broker

	validatorVotesRequired num.Decimal
}

type idKind struct {
	id   string
	kind commandspb.NodeSignatureKind
}

// nodeSig is a pair of a node and it signature.
type nodeSig struct {
	node string
	sig  string
}

func New(
	log *logging.Logger,
	cfg Config,
	top ValidatorTopology,
	broker Broker,
	cmd Commander,
	tt TimeTicker,
) (n *Notary) {
	defer func() { tt.NotifyOnTick(n.onTick) }()
	log.SetLevel(cfg.Level.Get())
	log = log.Named(namedLogger)
	return &Notary{
		cfg:                    cfg,
		log:                    log,
		sigs:                   map[idKind]map[nodeSig]struct{}{},
		top:                    top,
		broker:                 broker,
		cmd:                    cmd,
		validatorVotesRequired: defaultValidatorsVoteRequired,
		retries: &txTracker{
			txs: map[idKind]*signatureTime{},
		},
	}
}

func (n *Notary) OnDefaultValidatorsVoteRequiredUpdate(ctx context.Context, d num.Decimal) error {
	n.validatorVotesRequired = d
	return nil
}

// ReloadConf updates the internal configuration.
func (n *Notary) ReloadConf(cfg Config) {
	n.log.Info("reloading configuration")
	if n.log.GetLevel() != cfg.Level.Get() {
		n.log.Info("updating log level",
			logging.String("old", n.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		n.log.SetLevel(cfg.Level.Get())
	}

	n.cfg = cfg
}

// StartAggregate will register a new signature to be
// sent for a validator, or just ignore the signature and
// start aggregating signature for now validators,
// nil for the signature is OK for non-validators.
func (n *Notary) StartAggregate(
	resource string,
	kind commandspb.NodeSignatureKind,
	signature []byte,
) {
	// start aggregating for the resource
	idkind := idKind{resource, kind}
	if _, ok := n.sigs[idkind]; ok {
		n.log.Panic("aggregate already started for a resource",
			logging.String("resource", resource),
			logging.String("signature-kind", kind.String()),
		)
	}
	n.sigs[idkind] = map[nodeSig]struct{}{}

	// we are not a validator, then just return, job
	// done from here
	if !n.top.IsValidator() {
		return
	}

	// now let's just add the transaction to the retry list
	// no need to send the signature, it will be done on next onTick
	n.retries.Add(idkind, signature)
}

func (n *Notary) RegisterSignature(
	ctx context.Context,
	pubKey string,
	ns commandspb.NodeSignature,
) error {
	idkind := idKind{ns.Id, ns.Kind}
	sigs, ok := n.sigs[idkind]
	if !ok {
		return ErrUnknownResourceID
	}

	// not a validator signature
	if !n.top.IsValidatorVegaPubKey(pubKey) {
		return ErrNotAValidatorSignature
	}

	// if this is our own signature, remove it from the retries thing
	if strings.EqualFold(pubKey, n.top.SelfVegaPubKey()) {
		n.retries.Ack(idkind)
	}

	sigs[nodeSig{pubKey, string(ns.Sig)}] = struct{}{}

	signatures, ok := n.IsSigned(ctx, ns.Id, ns.Kind)
	if ok {
		// enough signature to reach the threshold have been received, let's send them to the
		// the api
		n.sendSignatureEvents(ctx, signatures)
	}
	return nil
}

func (n *Notary) IsSigned(
	ctx context.Context,
	resource string,
	kind commandspb.NodeSignatureKind,
) ([]commandspb.NodeSignature, bool) {
	idkind := idKind{resource, kind}

	// early exit if we don't have enough sig anyway
	if !n.votePassed(len(n.sigs[idkind]), n.top.Len()) {
		return nil, false
	}

	// aggregate node sig
	sig := map[string]struct{}{}
	out := []commandspb.NodeSignature{}
	for k := range n.sigs[idkind] {
		// is node sig is part of the registered nodes,
		// add it to the map
		// we may have a node which have been unregistered there, hence
		// us checkung
		if n.top.IsValidatorVegaPubKey(k.node) {
			sig[k.node] = struct{}{}
			out = append(out, commandspb.NodeSignature{
				Id:   resource,
				Kind: kind,
				Sig:  []byte(k.sig),
			})
		}
	}

	// now we check the number of required node sigs
	if n.votePassed(len(sig), n.top.Len()) {
		return out, true
	}

	return nil, false
}

// onTick is only use to trigger resending transaction.
func (n *Notary) onTick(_ context.Context, t time.Time) {
	toRetry := n.retries.getRetries(t)
	for k, v := range toRetry {
		n.send(k.id, k.kind, v.signature)
	}
}

func (n *Notary) send(id string, kind commandspb.NodeSignatureKind, signature []byte) {
	nsig := &commandspb.NodeSignature{Id: id, Sig: signature, Kind: kind}
	// we send a background context here because the actual context is ignore by the commander
	// which use a timeout of 5 seconds, this API may need to be addressed another day
	n.cmd.Command(context.Background(), txn.NodeSignatureCommand, nsig, func(err error) {
		// just a log is enough here, the transaction will be retried
		// later
		n.log.Error("could not send the transaction to tendermint", logging.Error(err))
	}, nil)
}

func (n *Notary) votePassed(votesCount, topLen int) bool {
	topLenDec := num.DecimalFromInt64(int64(topLen))
	return num.MinD(
		(topLenDec.Mul(n.validatorVotesRequired)).Add(oneDec), topLenDec,
	).LessThanOrEqual(num.DecimalFromInt64(int64(votesCount)))
}

func (n *Notary) sendSignatureEvents(ctx context.Context, signatures []commandspb.NodeSignature) {
	evts := make([]events.Event, 0, len(signatures))
	for _, ns := range signatures {
		evts = append(evts, events.NewNodeSignatureEvent(ctx, ns))
	}
	n.broker.SendBatch(evts)
}

// txTracker is a simple data structure
// to keep track of what transactions have been
// sent by this notary, and if a retry is necessary.
type txTracker struct {
	mu sync.Mutex
	// idKind -> time the tx was sent
	txs map[idKind]*signatureTime
}

type signatureTime struct {
	signature []byte
	time      time.Time
}

func (t *txTracker) getRetries(tm time.Time) map[idKind]signatureTime {
	t.mu.Lock()
	defer t.mu.Unlock()

	retries := map[idKind]signatureTime{}
	for k, v := range t.txs {
		if tm.After(v.time.Add(10 * time.Second)) {
			// add this signature to the retries list
			retries[k] = *v
			// update the entry with the current time of the retry
			v.time = tm
		}
	}

	return retries
}

func (t *txTracker) Ack(key idKind) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.txs, key)
}

func (t *txTracker) Add(key idKind, signature []byte) {
	t.mu.Lock()
	defer t.mu.Unlock()
	// we use the zero value here for the time, meaning it will need a retry
	// straight away
	t.txs[key] = &signatureTime{signature: signature, time: time.Time{}}
}
