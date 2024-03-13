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

package validators

import (
	"context"
	"errors"
	"math/rand"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/vega/core/txn"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

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

type TimeService interface {
	GetTimeNow() time.Time
}

type Commander interface {
	Command(ctx context.Context, cmd txn.Command, payload proto.Message, f func(string, error), bo *backoff.ExponentialBackOff)
	CommandSync(ctx context.Context, cmd txn.Command, payload proto.Message, f func(string, error), bo *backoff.ExponentialBackOff)
}

type ValidatorTopology interface {
	IsValidator() bool
	SelfVegaPubKey() string
	AllVegaPubKeys() []string
	IsValidatorVegaPubKey(string) bool
	IsTendermintValidator(string) bool
	GetVotingPower(pubkey string) int64
	GetTotalVotingPower() int64
}

type Resource interface {
	GetID() string
	GetType() commandspb.NodeVote_Type
	Check(ctx context.Context) error
}

const (
	notValidated uint32 = iota
	validated
	voteSent
)

const (
	minValidationPeriod = 1                   // sec minutes
	maxValidationPeriod = 30 * 24 * time.Hour // 30 days
	// by default all validators needs to sign.
)

var defaultValidatorsVoteRequired = num.MustDecimalFromString("1.0")

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
	state atomic.Uint32
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

func (r *res) votePassed(t ValidatorTopology, requiredMajority num.Decimal) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	count := int64(0)
	for k := range r.votes {
		if t.IsTendermintValidator(k) {
			count += t.GetVotingPower(k)
		}
	}

	return num.DecimalFromInt64(count).Div(num.DecimalFromInt64(t.GetTotalVotingPower())).GreaterThanOrEqual(requiredMajority)
}

type Witness struct {
	log *logging.Logger
	cfg Config
	ctx context.Context
	now time.Time
	top ValidatorTopology
	cmd Commander

	resources map[string]*res
	// handle sending transaction errors
	needResendMu  sync.Mutex
	needResendRes map[string]struct{}

	validatorVotesRequired num.Decimal
	wss                    *witnessSnapshotState
	defaultConfirmations   int64
}

func NewWitness(ctx context.Context, log *logging.Logger, cfg Config, top ValidatorTopology, cmd Commander, tsvc TimeService) (w *Witness) {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	return &Witness{
		ctx:                    ctx,
		log:                    log,
		cfg:                    cfg,
		now:                    tsvc.GetTimeNow(),
		cmd:                    cmd,
		top:                    top,
		resources:              map[string]*res{},
		needResendRes:          map[string]struct{}{},
		validatorVotesRequired: defaultValidatorsVoteRequired,
		wss: &witnessSnapshotState{
			serialised: []byte{},
		},
	}
}

func (w *Witness) SetDefaultConfirmations(c uint64) {
	w.defaultConfirmations = int64(c)
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

	return r.addVote(key.Hex())
}

func (w *Witness) StartCheck(
	r Resource,
	cb func(interface{}, bool),
	checkUntil time.Time,
) error {
	return w.startCheck(r, cb, checkUntil, w.defaultConfirmations)
}

func (w *Witness) StartCheckWithDelay(
	r Resource,
	cb func(interface{}, bool),
	checkUntil time.Time,
	initialDelay int64,
) error {
	return w.startCheck(r, cb, checkUntil, initialDelay)
}

func (w *Witness) startCheck(
	r Resource,
	cb func(interface{}, bool),
	checkUntil time.Time,
	initialDelay int64,
) error {
	id := r.GetID()
	if _, ok := w.resources[id]; ok {
		return ErrResourceDuplicate
	}

	if err := w.validateCheckUntil(checkUntil); err != nil {
		return err
	}

	ctx, cfunc := context.WithDeadline(w.ctx, checkUntil)
	rs := &res{
		res:        r,
		checkUntil: checkUntil,
		state:      atomic.Uint32{},
		cfunc:      cfunc,
		cb:         cb,
		votes:      map[string]struct{}{},
	}
	rs.state.Store(notValidated)

	w.resources[id] = rs

	// if we are a validator, we just start the routine.
	// so we can ensure the resources exists
	if w.top.IsValidator() {
		go w.start(ctx, rs, &initialDelay)
	} else {
		// if not a validator, we just jump to the state voteSent
		// and will wait for all validator to approve basically.
		// check succeeded
		rs.state.Store(voteSent)
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

func (w *Witness) start(ctx context.Context, r *res, initialDelay *int64) {
	if initialDelay != nil {
		t := time.NewTimer(time.Duration(*initialDelay) * w.cfg.ApproxEthereumBlockTime.Duration)
		<-t.C
		t.Stop()
	}

	backff := newBackoff(ctx, r.checkUntil.Sub(w.now))
	f := func() error {
		w.log.Debug("Checking the resource", logging.String("asset-source", r.res.GetID()))

		if err := r.res.Check(ctx); err != nil {
			w.log.Error("Checking the resource failed", logging.Error(err))
			return err
		}
		return nil
	}

	if err := backoff.Retry(f, backff); err != nil {
		return
	}

	// check succeeded
	r.state.Store(validated)
}

func (w *Witness) OnTick(ctx context.Context, t time.Time) {
	w.now = t
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

		state := v.state.Load()
		checkPass := v.votePassed(w.top, w.validatorVotesRequired)

		// if the time is expired, or we received enough votes
		if v.checkUntil.Before(t) || checkPass {
			// cancel the context so it stops the routine right now
			v.cfunc()

			if !checkPass {
				votesReceived := []string{}
				votesMissing := []string{}
				votePowers := []string{}
				for _, k := range w.top.AllVegaPubKeys() {
					if !w.top.IsTendermintValidator(k) {
						continue
					}
					if _, ok := v.votes[k]; ok {
						votesReceived = append(votesReceived, k)
						votePowers = append(votePowers, strconv.FormatInt(w.top.GetVotingPower(k), 10))
						continue
					}
					votesMissing = append(votesMissing, k)
				}
				w.log.Warn("resource checking was not validated by all nodes",
					logging.String("resource-id", v.res.GetID()),
					logging.Strings("votes-received", votesReceived),
					logging.Strings("votes-missing", votesMissing),
					logging.Strings("votes-power-received", votePowers),
					logging.Int64("total-voting-power", w.top.GetTotalVotingPower()),
				)
			}

			// callback to the resource holder
			v.cb(v.res, checkPass)
			// we delete the resource from our map.
			delete(w.resources, k)
			continue
		}

		// if we are a validator, and the resource was validated
		// then we try to send our vote.
		if isValidator && state == validated || w.needResend(k) {
			v.lastSentVote = t
			nv := &commandspb.NodeVote{
				Reference: v.res.GetID(),
				Type:      v.res.GetType(),
			}
			w.cmd.Command(ctx, txn.NodeVoteCommand, nv, w.onCommandSent(k), nil)
			// set new state so we do not try to validate again
			v.state.Store(voteSent)
		} else if (isValidator && state == voteSent) && t.After(v.lastSentVote.Add(w.cfg.NodeVoteResendInterval.Duration)) {
			if v.selfVoteReceived(w.top.SelfVegaPubKey()) {
				continue
			}
			w.onCommandSent(v.res.GetID())("", errors.New("no self votes received after 10 seconds"))
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

func (w *Witness) onCommandSent(res string) func(string, error) {
	return func(_ string, err error) {
		if err != nil {
			w.log.Error("could not send command", logging.String("res-id", res), logging.Error(err))
			w.needResendMu.Lock()
			defer w.needResendMu.Unlock()
			w.needResendRes[res] = struct{}{}
		}
	}
}
