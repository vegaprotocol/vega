package validators

import (
	"context"
	"encoding/hex"
	"errors"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/txn"
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
	GetTimeNow() (time.Time, error)
	NotifyOnTick(f func(context.Context, time.Time))
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/commander_mock.go -package mocks code.vegaprotocol.io/vega/validators Commander
type Commander interface {
	Command(ctx context.Context, cmd txn.Command, payload proto.Message) error
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/validator_topology_mock.go -package mocks code.vegaprotocol.io/vega/validators ValidatorTopology
type ValidatorTopology interface {
	Exists([]byte) bool
	Len() int
	IsValidator() bool
	SelfVegaPubKey() []byte
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
	minValidationPeriod = 1         // sec minutes
	maxValidationPeriod = 48 * 3600 // 2 days

)

func init() {
	// we seed the random generator just in case
	// as the backoff library use random internally
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
	cb func(interface{}, bool)
}

func (r *res) addVote(key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.votes[key]
	if ok {
		return ErrDuplicateVoteFromNode
	}

	// add the vote
	r.votes[key] = struct{}{}
	return nil
}

func (r *res) voteCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.votes)
}

type Witness struct {
	log       *logging.Logger
	cfg       Config
	resources map[string]*res
	now       time.Time
	top       ValidatorTopology
	cmd       Commander
}

func NewWitness(log *logging.Logger, cfg Config, top ValidatorTopology, cmd Commander, tsvc TimeService) (w *Witness) {
	defer func() {
		tsvc.NotifyOnTick(w.OnTick)
	}()

	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	now, _ := tsvc.GetTimeNow()
	return &Witness{
		log:       log,
		cfg:       cfg,
		now:       now,
		cmd:       cmd,
		top:       top,
		resources: map[string]*res{},
	}
}

// ReloadConf updates the internal configuration
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

func (w Witness) Stop() {
	// cancelling all context of checks which might be running
	for _, v := range w.resources {
		v.cfunc()
	}
}

// AddNodeCheck registers a vote from a validator node for a given resource
func (w *Witness) AddNodeCheck(ctx context.Context, nv *types.NodeVote) error {
	// get the node proposal first
	r, ok := w.resources[nv.Reference]
	if !ok {
		return ErrInvalidResourceIDForNodeVote
	}

	// ensure the node is a validator
	if !w.top.Exists(nv.PubKey) {
		w.log.Error("non-validator node tried to register node vote",
			logging.String("pubkey", hex.EncodeToString(nv.PubKey)))
		return ErrVoteFromNonValidator
	}

	return r.addVote(string(nv.PubKey))
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

	// if we are a validator, we just start the routinw.
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
	minValid, maxValid :=
		w.now.Add(minValidationPeriod*time.Second),
		w.now.Add(maxValidationPeriod*time.Second)
	if checkUntil.Unix() < minValid.Unix() || checkUntil.Unix() > maxValid.Unix() {
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

func (w Witness) start(ctx context.Context, r *res) {
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

func (w *Witness) OnTick(ctx context.Context, t time.Time) {
	w.now = t
	topLen := w.top.Len()
	isValidator := w.top.IsValidator()

	// check if any resources passed checks
	for k, v := range w.resources {
		state := atomic.LoadUint32(&v.state)
		votesLen := v.voteCount()

		// if the time is expired,
		if v.checkUntil.Before(t) ||
			// if we are a validator, and we want our vote to
			// be sent + all vote to be arrived
			(isValidator && votesLen == topLen && state == voteSent) ||
			// if we are not a validator, and do not care about our
			// own vote, just to have the validator voting OK
			(!isValidator && votesLen == topLen) {

			// cancel the context so it stops the routine right now
			v.cfunc()

			// if we have all validators votes, lets proceed
			checkPass := votesLen >= topLen
			if !checkPass {
				w.log.Warn("resource checking was not validated by all nodes",
					logging.String("resource-id", v.res.GetID()),
					logging.Int("vote-count", votesLen),
					logging.Int("node-count", topLen),
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
		if isValidator && state == validated {
			nv := &types.NodeVote{
				PubKey:    w.top.SelfVegaPubKey(),
				Reference: v.res.GetID(),
			}
			err := w.cmd.Command(ctx, txn.NodeVoteCommand, nv)
			if err != nil {
				w.log.Error("unable to send command", logging.Error(err))
				continue
			}
			// set new state so we do not try to validate again
			atomic.StoreUint32(&v.state, voteSent)
		}
	}
}
