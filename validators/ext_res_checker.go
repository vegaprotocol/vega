package validators

import (
	"context"
	"encoding/hex"
	"errors"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
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
	NotifyOnTick(f func(time.Time))
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/commander_mock.go -package mocks code.vegaprotocol.io/vega/validators Commander
type Commander interface {
	Command(cmd blockchain.Command, payload proto.Message) error
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
	minValidationPeriod = 600       // ten minutes
	maxValidationPeriod = 48 * 3600 // 2 days
	nodeApproval        = 1         // float for percentage
)

type res struct {
	res Resource
	// how long to run the check
	checkUntil time.Time
	// checks vote sent by the nodes
	votes map[string]struct{}
	// the stated of the checking
	state uint32
	// the context used to notify the routine to exit
	cfunc context.CancelFunc
	// the function to call one validation is done
	cb func(interface{}, bool)
}

type ExtResChecker struct {
	log       *logging.Logger
	resources map[string]*res
	now       time.Time
	top       ValidatorTopology
	cmd       Commander
}

func NewExtResChecker(log *logging.Logger, top ValidatorTopology, cmd Commander, tsvc TimeService) (e *ExtResChecker) {
	defer func() {
		tsvc.NotifyOnTick(e.OnTick)
	}()

	now, _ := tsvc.GetTimeNow()
	return &ExtResChecker{
		log:       log,
		now:       now,
		cmd:       cmd,
		top:       top,
		resources: map[string]*res{},
	}
}

func (e ExtResChecker) Stop() {
	// cancelling all context of checks which might be running
	for _, v := range e.resources {
		v.cfunc()
	}
}

// AddNodeVote registers a vote from a validator node for a given resource
func (e *ExtResChecker) AddNodeCheck(nv *types.NodeVote) error {
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

	_, ok = r.votes[string(nv.PubKey)]
	if ok {
		return ErrDuplicateVoteFromNode
	}

	// add the vote
	r.votes[string(nv.PubKey)] = struct{}{}

	return nil
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

	ctx, cfunc := context.WithCancel(context.Background())
	res := &res{
		res:        r,
		checkUntil: checkUntil,
		state:      notValidated,
		cfunc:      cfunc,
		cb:         cb,
		votes:      map[string]struct{}{},
	}

	e.resources[id] = res
	e.start(ctx, res)
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

func (e ExtResChecker) start(ctx context.Context, r *res) {
	// wait time between call to validation
	var (
		err    error
		ticker = time.NewTicker(500 * time.Millisecond)
	)
	defer ticker.Stop()
	for {
		// first try to validate the asset
		e.log.Debug("Checking the resource",
			logging.String("asset-source", r.res.GetID()),
		)

		// call checking
		err = r.res.Check()
		if err != nil {
			// we just log the error, but these are not criticals, as it may be
			// things unrelated to the current node, and would recover later on.
			// it's just informative
			e.log.Warn("error checking resource", logging.Error(err))
		} else {
			atomic.StoreUint32(&r.state, validated)
			return
		}

		// wait or break if the time's up
		select {
		case <-ctx.Done():
			e.log.Error("resource checking context done",
				logging.Error(ctx.Err()))
			return
		case _ = <-ticker.C:
		}
	}
}

func (e *ExtResChecker) OnTick(t time.Time) {
	e.now = t
	topLen := e.top.Len()

	// check if any resources passed checks
	for k, v := range e.resources {
		state := atomic.LoadUint32(&v.state)
		votesLen := len(v.votes)

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
				err := e.cmd.Command(blockchain.NodeVoteCommand, nv)
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
