// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package validators

import (
	"context"
	"errors"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/crypto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/txn"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/cenkalti/backoff"
	"github.com/golang/protobuf/proto"
)

var (
	ErrResourceDuplicate            = errors.New("resource duplicate")
	ErrCheckUntilInvalid            = errors.New("invalid time to check until")
	ErrInvalidResourceIDForNodeVote = errors.New("invalid resource ID")
	ErrVoteFromNonValidator         = errors.New("vote from non validator")
	ErrDuplicateVoteFromNode        = errors.New("duplicate vote from node")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/validators TimeService
type TimeService interface {
	GetTimeNow() time.Time
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/commander_mock.go -package mocks code.vegaprotocol.io/vega/validators Commander
type Commander interface {
	Command(ctx context.Context, cmd txn.Command, payload proto.Message, f func(error), bo *backoff.ExponentialBackOff)
	CommandSync(ctx context.Context, cmd txn.Command, payload proto.Message, f func(error), bo *backoff.ExponentialBackOff)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/validator_topology_mock.go -package mocks code.vegaprotocol.io/vega/validators ValidatorTopology
type ValidatorTopology interface {
	Len() int
	IsValidator() bool
	SelfVegaPubKey() string
	AllVegaPubKeys() []string
	IsValidatorVegaPubKey(string) bool
	IsTendermintValidator(string) bool
}

type Resource interface {
	GetID() string
	Check() error
}

const (
	notValidated uint32 = iota
	validated
	voteSent
)

const (
	minValidationPeriod = 1                   // sec minutes
	maxValidationPeriod = 30 * 24 * time.Hour // 2 days
	// by default all validators needs to sign.
)

var (
	defaultValidatorsVoteRequired = num.MustDecimalFromString("1.0")
	oneDec                        = num.MustDecimalFromString("1")
)

func init() {
	// we seed the random generator just in case
	// as the backoff library use random internally
	// TODO this probably needs to change to something that can be agreed across all nodes.
	rand.Seed(time.Now().UnixNano())
}

type res struct {
	res Resource
	// how long to run the check
	checkUntil time.Time
	mu         sync.Mutex
	votes      map[string]struct{} // checks vote sent by the nodes
	// the stated of the checking
	state uint32
	// the context used to notify the routine to exit
	cfunc context.CancelFunc
	// the function to call one validation is done
	cb           func(interface{}, bool)
	lastSentVote time.Time
}

func (r *res) addVote(key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.votes[key]; ok {
		return ErrDuplicateVoteFromNode
	}

	// add the vote
	r.votes[key] = struct{}{}
	return nil
}

func (r *res) selfVoteReceived(self string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.votes[self]
	return ok
}

func (r *res) voteCount(t ValidatorTopology) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	count := 0
	for k := range r.votes {
		if t.IsTendermintValidator(k) {
			count += 1
		}
	}
	return count
}

type Witness struct {
	log *logging.Logger
	cfg Config
	now time.Time
	top ValidatorTopology
	cmd Commander

	resources map[string]*res
	// handle sending transaction errors
	needResendMu  sync.Mutex
	needResendRes map[string]struct{}

	validatorVotesRequired num.Decimal
	wss                    *witnessSnapshotState
}

func NewWitness(log *logging.Logger, cfg Config, top ValidatorTopology, cmd Commander, tsvc TimeService) (w *Witness) {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	return &Witness{
		log:                    log,
		cfg:                    cfg,
		now:                    tsvc.GetTimeNow(),
		cmd:                    cmd,
		top:                    top,
		resources:              map[string]*res{},
		needResendRes:          map[string]struct{}{},
		validatorVotesRequired: defaultValidatorsVoteRequired,
		wss: &witnessSnapshotState{
			changed:    true,
			serialised: []byte{},
		},
	}
}

func (w *Witness) OnDefaultValidatorsVoteRequiredUpdate(ctx context.Context, d num.Decimal) error {
	w.validatorVotesRequired = d
	return nil
}

// ReloadConf updates the internal configuration.
func (w *Witness) ReloadConf(cfg Config) {
	w.log.Info("reloading configuration")
	if w.log.GetLevel() != cfg.Level.Get() {
		w.log.Info("updating log level",
			logging.String("old", w.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		w.log.SetLevel(cfg.Level.Get())
	}

	w.cfg = cfg
}

func (w *Witness) Stop() {
	// cancelling all context of checks which might be running
	for _, v := range w.resources {
		v.cfunc()
	}
}

// AddNodeCheck registers a vote from a validator node for a given resource.
func (w *Witness) AddNodeCheck(_ context.Context, nv *commandspb.NodeVote, key crypto.PublicKey) error {
	// get the node proposal first
	r, ok := w.resources[nv.Reference]
	if !ok {
		w.log.Error("invalid resource ID received for vote",
			logging.String("resource-ref", nv.Reference),
			logging.String("node-id", key.Hex()),
		)
		return ErrInvalidResourceIDForNodeVote
	}

	// ensure the node is a validator
	if !w.top.IsValidatorVegaPubKey(key.Hex()) {
		w.log.Error("non-validator node tried to register node vote",
			logging.String("node-id", key.Hex()))
		return ErrVoteFromNonValidator
	}

	w.setChangedLocked(true)
	return r.addVote(key.Hex())
}

func (w *Witness) setChangedLocked(changed bool) {
	w.wss.mu.Lock()
	w.wss.changed = changed
	w.wss.mu.Unlock()
}

func (w *Witness) StartCheck(
	r Resource,
	cb func(interface{}, bool),
	checkUntil time.Time,
) error {
	id := r.GetID()
	if _, ok := w.resources[id]; ok {
		return ErrResourceDuplicate
	}

	if err := w.validateCheckUntil(checkUntil); err != nil {
		return err
	}

	ctx, cfunc := context.WithDeadline(context.Background(), checkUntil)
	rs := &res{
		res:        r,
		checkUntil: checkUntil,
		state:      notValidated,
		cfunc:      cfunc,
		cb:         cb,
		votes:      map[string]struct{}{},
	}

	w.resources[id] = rs
	w.setChangedLocked(true)

	// if we are a validator, we just start the routine.
	// so we can ensure the resources exists
	if w.top.IsValidator() {
		go w.start(ctx, rs)
	} else {
		// if not a validator, we just jump to the state voteSent
		// and will wait for all validator to approve basically.
		// check succeeded
		atomic.StoreUint32(&rs.state, voteSent)
	}
	return nil
}

func (w *Witness) validateCheckUntil(checkUntil time.Time) error {
	minValid, maxValid := w.now.Add(minValidationPeriod),
		w.now.Add(maxValidationPeriod)
	if checkUntil.Unix() < minValid.Unix() || checkUntil.Unix() > maxValid.Unix() {
		if w.log.GetLevel() <= logging.DebugLevel {
			w.log.Debug("invalid duration for witness",
				logging.Time("check-until", checkUntil),
				logging.Time("min-valid", minValid),
				logging.Time("max-valid", maxValid),
			)
		}
		return ErrCheckUntilInvalid
	}
	return nil
}

func newBackoff(ctx context.Context, maxElapsedTime time.Duration) backoff.BackOff {
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = maxElapsedTime
	bo.InitialInterval = 1 * time.Second
	return backoff.WithContext(bo, ctx)
}

func (w *Witness) start(ctx context.Context, r *res) {
	backff := newBackoff(ctx, r.checkUntil.Sub(w.now))
	f := func() error {
		w.log.Debug("Checking the resource",
			logging.String("asset-source", r.res.GetID()),
		)
		err := r.res.Check()
		if err != nil {
			w.log.Debug("error checking resource", logging.Error(err))
			// dump error
			return err
		}
		return nil
	}

	err := backoff.Retry(f, backff)
	if err != nil {
		return
	}

	// check succeeded
	atomic.StoreUint32(&r.state, validated)
}

func (w *Witness) votePassed(votesCount, topLen int) bool {
	topLenDec := num.DecimalFromInt64(int64(topLen))
	return num.MinD(
		(topLenDec.Mul(w.validatorVotesRequired)).Add(oneDec), topLenDec,
	).LessThanOrEqual(num.DecimalFromInt64(int64(votesCount)))
}

func (w *Witness) OnTick(ctx context.Context, t time.Time) {
	w.now = t
	topLen := w.top.Len() // this is the number of validators with Tendermint status
	isValidator := w.top.IsValidator()

	// sort resources first
	resourceIDs := make([]string, 0, len(w.resources))
	for k := range w.resources {
		resourceIDs = append(resourceIDs, k)
	}
	sort.Strings(resourceIDs)

	// check if any resources passed checks
	for _, k := range resourceIDs {
		v := w.resources[k]

		state := atomic.LoadUint32(&v.state)
		votesLen := v.voteCount(w.top)
		checkPass := w.votePassed(votesLen, topLen)

		// if the time is expired, or we received enough votes
		if v.checkUntil.Before(t) || checkPass {
			// cancel the context so it stops the routine right now
			v.cfunc()

			if !checkPass {
				votesReceived := []string{}
				votesMissing := []string{}
				for _, k := range w.top.AllVegaPubKeys() {
					if !w.top.IsTendermintValidator(k) {
						continue
					}
					if _, ok := v.votes[k]; ok {
						votesReceived = append(votesReceived, k)
						continue
					}
					votesMissing = append(votesMissing, k)
				}
				w.log.Warn("resource checking was not validated by all nodes",
					logging.String("resource-id", v.res.GetID()),
					logging.Int("vote-count", votesLen),
					logging.Int("node-count", topLen),
					logging.Strings("votes-received", votesReceived),
					logging.Strings("votes-missing", votesMissing),
				)
			}

			// callback to the resource holder
			v.cb(v.res, checkPass)
			// we delete the resource from our map.
			delete(w.resources, k)
			w.setChangedLocked(true)
			continue
		}

		// if we are a validator, and the resource was validated
		// then we try to send our vote.
		if isValidator && state == validated || w.needResend(k) {
			v.lastSentVote = t
			nv := &commandspb.NodeVote{
				Reference: v.res.GetID(),
			}
			w.cmd.Command(ctx, txn.NodeVoteCommand, nv, w.onCommandSent(k), nil)
			// set new state so we do not try to validate again
			atomic.StoreUint32(&v.state, voteSent)
		} else if (isValidator && state == voteSent) && t.After(v.lastSentVote.Add(10*time.Second)) {
			if v.selfVoteReceived(w.top.SelfVegaPubKey()) {
				continue
			}
			w.onCommandSent(v.res.GetID())(errors.New("no self votes received after 10 seconds"))
		}
	}
}

func (w *Witness) needResend(res string) bool {
	w.needResendMu.Lock()
	defer w.needResendMu.Unlock()
	if _, ok := w.needResendRes[res]; ok {
		delete(w.needResendRes, res)
		return true
	}
	return false
}

func (w *Witness) onCommandSent(res string) func(error) {
	return func(err error) {
		if err != nil {
			w.log.Error("could not send command", logging.String("res-id", res), logging.Error(err))
			w.needResendMu.Lock()
			defer w.needResendMu.Unlock()
			w.needResendRes[res] = struct{}{}
		}
	}
}
