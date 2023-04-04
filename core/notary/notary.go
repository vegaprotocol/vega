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

package notary

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/txn"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"golang.org/x/exp/maps"

	"github.com/cenkalti/backoff"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

// by default all validators needs to sign.
var defaultValidatorsVoteRequired = num.MustDecimalFromString("1.0")

var (
	ErrAggregateSigAlreadyStartedForResource = errors.New("aggregate signature already started for resource")
	ErrUnknownResourceID                     = errors.New("unknown resource ID")
	ErrNotAValidatorSignature                = errors.New("not a validator signature")
)

// ValidatorTopology...
//
//go:generate go run github.com/golang/mock/mockgen -destination mocks/validator_topology_mock.go -package mocks code.vegaprotocol.io/vega/core/notary ValidatorTopology
type ValidatorTopology interface {
	IsValidator() bool
	IsValidatorVegaPubKey(string) bool
	IsTendermintValidator(string) bool
	SelfVegaPubKey() string
	Len() int
}

// Broker needs no mocks.
type Broker interface {
	Send(event events.Event)
	SendBatch(events []events.Event)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/commander_mock.go -package mocks code.vegaprotocol.io/vega/core/notary Commander
type Commander interface {
	Command(ctx context.Context, cmd txn.Command, payload proto.Message, f func(string, error), bo *backoff.ExponentialBackOff)
	CommandSync(ctx context.Context, cmd txn.Command, payload proto.Message, f func(string, error), bo *backoff.ExponentialBackOff)
}

// Notary will aggregate all signatures of a node for
// a specific Command
// e.g: asset withdrawal, asset allowlisting, etc.
type Notary struct {
	cfg Config
	log *logging.Logger

	// resource to be signed -> signatures
	sigs              map[idKind]map[nodeSig]struct{}
	pendingSignatures map[idKind]struct{}
	retries           *txTracker
	top               ValidatorTopology
	cmd               Commander
	broker            Broker

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
) (n *Notary) {
	log.SetLevel(cfg.Level.Get())
	log = log.Named(namedLogger)
	return &Notary{
		cfg:                    cfg,
		log:                    log,
		sigs:                   map[idKind]map[nodeSig]struct{}{},
		pendingSignatures:      map[idKind]struct{}{},
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
	n.pendingSignatures[idkind] = struct{}{}

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
		// remove from the pending
		delete(n.pendingSignatures, idkind)
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
		// is node sig is part of the registered nodes, and is a tendermint validator
		// add it to the map
		// we may have a node are validators but with a lesser status sending in votes
		if n.top.IsTendermintValidator(k.node) {
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
func (n *Notary) OnTick(ctx context.Context, t time.Time) {
	toRetry := n.retries.getRetries(t)
	for k, v := range toRetry {
		n.send(k.id, k.kind, v.signature)
	}

	pendings := maps.Keys(n.pendingSignatures)
	sort.Slice(pendings, func(i, j int) bool {
		return pendings[i].id < pendings[j].id
	})

	for _, v := range pendings {
		if signatures, ok := n.IsSigned(ctx, v.id, v.kind); ok {
			// remove from the pending
			delete(n.pendingSignatures, v)
			// enough signature to reach the threshold have been received, let's send them to the
			// the api
			n.sendSignatureEvents(ctx, signatures)
		}
	}
}

func (n *Notary) send(id string, kind commandspb.NodeSignatureKind, signature []byte) {
	nsig := &commandspb.NodeSignature{Id: id, Sig: signature, Kind: kind}
	// we send a background context here because the actual context is ignore by the commander
	// which use a timeout of 5 seconds, this API may need to be addressed another day
	n.cmd.Command(context.Background(), txn.NodeSignatureCommand, nsig, func(_ string, err error) {
		// just a log is enough here, the transaction will be retried
		// later
		n.log.Error("could not send the transaction to tendermint", logging.Error(err))
	}, nil)
}

func (n *Notary) votePassed(votesCount, topLen int) bool {
	return num.DecimalFromInt64(int64(votesCount)).Div(num.DecimalFromInt64(int64(topLen))).GreaterThanOrEqual(n.validatorVotesRequired)
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
