package notary

import (
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

var (
	ErrAggregateSigAlreadyStartedForResource = errors.New("aggregate signature already started for resource")
	ErrUnknownResourceID                     = errors.New("unknown resource ID")
	ErrNotAValidatorSignature                = errors.New("not a validator signature")
)

// ValidatorTopology...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/validator_topology_mock.go -package mocks code.vegaprotocol.io/vega/notary ValidatorTopology
type ValidatorTopology interface {
	Exists([]byte) bool
	Len() int
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/event_broker_mock.go -package mocks code.vegaprotocol.io/vega/notary Broker
type Broker interface {
	Send(event events.Event)
}

// Notary will aggregate all signatures of a node for
// a specific Command
// e.g: asset withdrawal, asset whitelisting, etc
type Notary struct {
	cfg Config
	log *logging.Logger

	// resource to be signed -> signatures
	sigs   map[idKind]map[nodeSig]struct{}
	top    ValidatorTopology
	broker Broker
}

type idKind struct {
	id   string
	kind types.NodeSignatureKind
}

/// nodeSig is a pair of a node and it signature
type nodeSig struct {
	node string
	sig  string
}

func New(log *logging.Logger, cfg Config, top ValidatorTopology) *Notary {
	log = log.Named(namedLogger)
	return &Notary{
		cfg:  cfg,
		log:  log,
		sigs: map[idKind]map[nodeSig]struct{}{},
		top:  top,
	}
}

// ReloadConf updates the internal configuration of the collateral engine
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

func (n *Notary) StartAggregate(resID string, kind types.NodeSignatureKind) error {
	if _, ok := n.sigs[idKind{resID, kind}]; ok {
		return ErrAggregateSigAlreadyStartedForResource
	}
	n.sigs[idKind{resID, kind}] = map[nodeSig]struct{}{}
	return nil
}

func (n *Notary) AddSig(resID string, kind types.NodeSignatureKind, pubKey []byte, sig []byte) ([]types.NodeSignature, bool, error) {
	sigs, ok := n.sigs[idKind{resID, kind}]
	if !ok {
		return nil, false, ErrUnknownResourceID
	}

	// not a validator signature
	if !n.top.Exists(pubKey) {
		return nil, false, ErrNotAValidatorSignature
	}

	sigs[nodeSig{string(pubKey), string(sig)}] = struct{}{}

	sigsout, ok := n.isSigned(resID, kind)
	return sigsout, ok, nil
}

func (n *Notary) isSigned(resID string, kind types.NodeSignatureKind) ([]types.NodeSignature, bool) {
	// early exit if we don't have enough sig anyway
	if float64(len(n.sigs[idKind{resID, kind}]))/float64(n.top.Len()) < n.cfg.SignaturesRequiredPercent {
		return nil, false
	}

	// aggregate node sig
	sig := map[string]struct{}{}
	out := []types.NodeSignature{}
	for k, _ := range n.sigs[idKind{resID, kind}] {
		// is node sig is part of the registered nodes,
		// add it to the map
		// we may have a node which have been unregistered there, hence
		// us checkung
		if n.top.Exists([]byte(k.node)) {
			sig[k.node] = struct{}{}
			out = append(out, types.NodeSignature{
				ID:   resID,
				Kind: kind,
				Sig:  []byte(k.sig),
			})
		}
	}

	// now we check the number of required node sigs
	if float64(len(sig))/float64(n.top.Len()) >= n.cfg.SignaturesRequiredPercent {
		return out, true
	}

	return nil, false
}
