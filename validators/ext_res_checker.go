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
	nodeApproval        = 1         // float for percentage
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

type ExtResChecker struct {
	log       *logging.Logger
	cfg       Config
	resources map[string]*res
	now       time.Time
	top       ValidatorTopology
	cmd       Commander
}

func NewExtResChecker(log *logging.Logger, cfg Config, top ValidatorTopology, cmd Commander, tsvc TimeService) (e *ExtResChecker) {
	defer func() {
		tsvc.NotifyOnTick(e.OnTick)
	}()

	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	now, _ := tsvc.GetTimeNow()
	return &ExtResChecker{
		log:       log,
		cfg:       cfg,
		now:       now,
		cmd:       cmd,
		top:       top,
		resources: map[string]*res{},
	}
}

// ReloadConf updates the internal configuration
func (e *ExtResChecker) ReloadConf(cfg Config) {
	e.log.Info("reloading configuration")
	if e.log.GetLevel() != cfg.Level.Get() {
		e.log.Info("updating log level",
			logging.String("old", e.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		e.log.SetLevel(cfg.Level.Get())
	}

	e.cfg = cfg
}

func (e ExtResChecker) Stop() {
	// cancelling all context of checks which might be running
	for _, v := range e.resources {
		v.cfunc()
	}
}

// AddNodeCheck registers a vote from a validator node for a given resource
func (e *ExtResChecker) AddNodeCheck(ctx context.Context, nv *types.NodeVote) error {
	// get the node proposal first
	r, ok := e.resources[nv.Reference]
	if !ok {
		return ErrInvalidResourceIDForNodeVote
	}

	// ensure the node is a validator
	if !e.top.Exists(nv.PubKey) {
		e.log.Error("non-validator node tried to register node vote",
			logging.String("pubkey", hex.EncodeToString(nv.PubKey)))
		return ErrVoteFromNonValidator
	}

	return r.addVote(string(nv.PubKey))
}

func (e *ExtResChecker) StartCheck(
	r Resource,
	cb func(interface{}, bool),
	checkUntil time.Time,
) error {
	id := r.GetID()
	if _, ok := e.resources[id]; ok {
		return ErrResourceDuplicate
	}

	if err := e.validateCheckUntil(checkUntil); err != nil {
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

	e.resources[id] = rs

	// validtor or not, we start the routine to validatate the
	// internall data as th resource may require retrieve data from the
	// foreign chains
	go e.start(ctx, rs)
	return nil
}

func (e *ExtResChecker) validateCheckUntil(checkUntil time.Time) error {
	minValid, maxValid :=
		e.now.Add(minValidationPeriod*time.Second),
		e.now.Add(maxValidationPeriod*time.Second)
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

func (e *ExtResChecker) start(ctx context.Context, r *res) {
	backff := newBackoff(ctx, r.checkUntil.Sub(e.now))
	f := func() error {
		e.log.Debug("Checking the resource",
			logging.String("asset-source", r.res.GetID()),
		)
		err := r.res.Check()
		if err != nil {
			e.log.Warn("error checking resource", logging.Error(err))
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

func (e *ExtResChecker) OnTick(ctx context.Context, t time.Time) {
	e.now = t
	topLen := e.top.Len()

	// check if any resources passed checks
	for k, v := range e.resources {
		state := atomic.LoadUint32(&v.state)
		votesLen := v.voteCount()

		// if the time is expired,
		if v.checkUntil.Before(t) ||
			(votesLen == topLen && state == voteSent) {
			// cancel the context so it stops the routine right now
			v.cfunc()

			// if we have all validators votes, lets proceed
			checkPass := votesLen >= topLen
			if !checkPass {
				e.log.Warn("resource checking was not validated by all nodes",
					logging.String("resource-id", v.res.GetID()),
					logging.Int("vote-count", votesLen),
					logging.Int("node-count", topLen),
				)
			}

			// callback to the resource holder
			v.cb(v.res, checkPass)
			delete(e.resources, k)
		}

		// then send votes if needed
		if state == validated {
			// if not a validator no need to send the vote
			if e.top.IsValidator() {
				nv := &types.NodeVote{
					PubKey:    e.top.SelfVegaPubKey(),
					Reference: v.res.GetID(),
				}
				err := e.cmd.Command(ctx, txn.NodeVoteCommand, nv)
				if err != nil {
					e.log.Error("unable to send command", logging.Error(err))
					continue
				}
			}
			// set new state so we do not try to validate again
			atomic.StoreUint32(&v.state, voteSent)
		}
	}
}
