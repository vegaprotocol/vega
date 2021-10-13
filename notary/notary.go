package notary

import (
	"context"
	"math"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/txn"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

const (
	// by default all validators needs to sign
	defaultValidatorsVoteRequired = 1.0
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
	Len() int
}

// Broker needs no mocks
type Broker interface {
	Send(event events.Event)
	SendBatch(events []events.Event)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/commander_mock.go -package mocks code.vegaprotocol.io/vega/processor Commander
type Commander interface {
	Command(ctx context.Context, cmd txn.Command, payload proto.Message, f func(bool))
}

// Notary will aggregate all signatures of a node for
// a specific Command
// e.g: asset withdrawal, asset allowlisting, etc
type Notary struct {
	cfg Config
	log *logging.Logger

	// resource to be signed -> signatures
	sigs   map[idKind]map[nodeSig]struct{}
	top    ValidatorTopology
	cmd    Commander
	broker Broker

	validatorVotesRequired float64
}

type idKind struct {
	id   string
	kind commandspb.NodeSignatureKind
}

// / nodeSig is a pair of a node and it signature
type nodeSig struct {
	node string
	sig  string
}

func New(log *logging.Logger, cfg Config, top ValidatorTopology, broker Broker, cmd Commander) *Notary {
	log = log.Named(namedLogger)
	return &Notary{
		cfg:                    cfg,
		log:                    log,
		sigs:                   map[idKind]map[nodeSig]struct{}{},
		top:                    top,
		broker:                 broker,
		cmd:                    cmd,
		validatorVotesRequired: defaultValidatorsVoteRequired,
	}
}

func (n *Notary) OnDefaultValidatorsVoteRequiredUpdate(ctx context.Context, f float64) error {
	n.validatorVotesRequired = f
	return nil
}

// ReloadConf updates the internal configuration
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

func (n *Notary) StartAggregate(resID string, kind commandspb.NodeSignatureKind) {
	if _, ok := n.sigs[idKind{resID, kind}]; ok {
		n.log.Panic("aggregate already started for a resource",
			logging.String("resource", resID),
			logging.String("signature-kind", kind.String()),
		)
	}
	n.sigs[idKind{resID, kind}] = map[nodeSig]struct{}{}
}

func (n *Notary) AddSig(
	ctx context.Context,
	pubKey string,
	ns commandspb.NodeSignature,
) ([]commandspb.NodeSignature, bool, error) {
	sigs, ok := n.sigs[idKind{ns.Id, ns.Kind}]
	if !ok {
		return nil, false, ErrUnknownResourceID
	}

	// not a validator signature
	if !n.top.IsValidatorVegaPubKey(pubKey) {
		return nil, false, ErrNotAValidatorSignature
	}

	sigs[nodeSig{pubKey, string(ns.Sig)}] = struct{}{}

	sigsout, ok := n.IsSigned(ctx, ns.Id, ns.Kind)
	if ok {
		// enough signature to reach the threshold have been received, let's send them to the
		// the api
		evts := make([]events.Event, 0, len(sigsout))
		for _, ns := range sigsout {
			evts = append(evts, events.NewNodeSignatureEvent(ctx, ns))
		}
		n.broker.SendBatch(evts)
	}
	return sigsout, ok, nil
}

func (n *Notary) IsSigned(ctx context.Context, resID string, kind commandspb.NodeSignatureKind) ([]commandspb.NodeSignature, bool) {
	// early exit if we don't have enough sig anyway
	if !n.votePassed(len(n.sigs[idKind{resID, kind}]), n.top.Len()) {
		return nil, false
	}

	// aggregate node sig
	sig := map[string]struct{}{}
	out := []commandspb.NodeSignature{}
	for k := range n.sigs[idKind{resID, kind}] {
		// is node sig is part of the registered nodes,
		// add it to the map
		// we may have a node which have been unregistered there, hence
		// us checkung
		if n.top.IsValidatorVegaPubKey(k.node) {
			sig[k.node] = struct{}{}
			out = append(out, commandspb.NodeSignature{
				Id:   resID,
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

func (n *Notary) votePassed(votesCount, topLen int) bool {
	return math.Min((float64(topLen)*n.validatorVotesRequired)+1, float64(topLen)) <= float64(votesCount)
}

func (n *Notary) SendSignature(ctx context.Context, id string, sig []byte, kind commandspb.NodeSignatureKind) error {
	if !n.top.IsValidator() {
		return nil
	}
	nsig := &commandspb.NodeSignature{
		Id:   id,
		Sig:  sig,
		Kind: kind,
	}

	//  may need to figure out retries with this one.
	n.cmd.Command(ctx, txn.NodeSignatureCommand, nsig, nil)

	return nil
}
