package oracles

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/core/evtforward/ethcall"

	"github.com/ethereum/go-ethereum"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/validators"
	"code.vegaprotocol.io/vega/logging"
)

var ErrDuplicatedEthereumCallEvent = errors.New("duplicated call event")

//go:generate go run github.com/golang/mock/mockgen -destination mocks/witness_mock.go -package mocks code.vegaprotocol.io/vega/core/oracles Witness
type Witness interface {
	StartCheck(validators.Resource, func(interface{}, bool), time.Time) error
	RestoreResource(validators.Resource, func(interface{}, bool)) error
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/oracle_data_broadcaster_mock.go -package mocks code.vegaprotocol.io/vega/core/oracles OracleDataBroadcaster
type OracleDataBroadcaster interface {
	BroadcastData(context.Context, OracleData) error
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/eth_call_spec_source_mock.go -package mocks code.vegaprotocol.io/vega/core/oracles EthCallSpecSource
type EthCallSpecSource interface {
	GetCall(id string) (EthCall, error)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/eth_contract_caller.go -package mocks code.vegaprotocol.io/vega/core/oracles ContractCaller
type ContractCaller interface {
	CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/eth_confirmations_mock.go -package mocks code.vegaprotocol.io/vega/core/oracles EthereumConfirmations
type EthereumConfirmations interface {
	Check(uint64) error
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/eth_call_mock.go -package mocks code.vegaprotocol.io/vega/core/oracles EthCall
type EthCall interface {
	Normalise(callResult []byte) (map[string]string, error)
	Call(ctx context.Context, caller ethereum.ContractCaller, blockNumber *big.Int) ([]byte, error)
	PassesFilters(result []byte, blockHeight uint64, blockTime uint64) bool
	RequiredConfirmations() uint64
}

type EthereumOracleVerifier struct {
	log *logging.Logger

	witness           Witness
	timeService       TimeService
	oracleEngine      OracleDataBroadcaster
	ethCallSpecSource EthCallSpecSource
	ethContractCaller ethereum.ContractCaller
	ethConfirmations  EthereumConfirmations

	pendingCallEvents    []*pendingCallEvent
	finalizedCallResults []*types.EthContractCallEvent

	mu     sync.Mutex
	hashes map[string]struct{}

	// snapshot data
	snapshotState *ethereumOracleVerifierSnapshotState
}

type pendingCallEvent struct {
	callEvent types.EthContractCallEvent
	check     func() error
}

func (p pendingCallEvent) GetID() string { return p.callEvent.Hash() }

func (p pendingCallEvent) GetType() types.NodeVoteType {
	return types.NodeVoteTypeEthereumContractCallResult
}
func (p *pendingCallEvent) Check() error { return p.check() }

func NewEthereumOracleVerifier(
	log *logging.Logger,
	witness Witness,
	ts TimeService,
	oracleBroadcaster OracleDataBroadcaster,
	ethCallSpecSource EthCallSpecSource,
	ethContractCaller ethereum.ContractCaller,
	ethConfirmations EthereumConfirmations,
) (sv *EthereumOracleVerifier) {
	log = log.Named("ethereum-oracle-verifier")
	s := &EthereumOracleVerifier{
		log:               log,
		witness:           witness,
		timeService:       ts,
		oracleEngine:      oracleBroadcaster,
		ethCallSpecSource: ethCallSpecSource,
		ethContractCaller: ethContractCaller,
		ethConfirmations:  ethConfirmations,
		hashes:            map[string]struct{}{},
		snapshotState:     &ethereumOracleVerifierSnapshotState{},
	}
	return s
}

func (s *EthereumOracleVerifier) ensureNotDuplicate(hash string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.hashes[hash]; ok {
		return false
	}

	s.hashes[hash] = struct{}{}

	return true
}

// TODO: non finalized events could cause a memory leak, this needs to be addressed in a way that will prevent processing
// duplicates but not result in a memory leak, agreed to postpone for now  (other verifiers have the same issue)

func (s *EthereumOracleVerifier) ProcessEthereumContractCallResult(callEvent types.EthContractCallEvent) error {
	// Possible that a single oracle trigger, where trigger is base off of vegatime, could result in chain events with
	// different contents (and therefore different hashes) due to difference in the time when the validators query
	// ethereum, it is assumed that the consumer of the oracle events will handle this.  Alternatively we could restrict
	// triggers to use ethereum time and not allow vega time triggers, then no such issue.
	if ok := s.ensureNotDuplicate(callEvent.Hash()); !ok {
		s.log.Error("ethereum call event already exists",
			logging.String("event", fmt.Sprintf("%+v", callEvent)))
		return ErrDuplicatedEthereumCallEvent
	}

	pending := &pendingCallEvent{
		callEvent: callEvent,
		check:     func() error { return s.checkCallEventResult(callEvent) },
	}

	s.pendingCallEvents = append(s.pendingCallEvents, pending)

	s.log.Info("ethereum call event received, starting validation",
		logging.String("call-event", fmt.Sprintf("%+v", callEvent)))

	// Timeout for the check set to 1 day, to allow for validator outage scenarios
	return s.witness.StartCheck(
		pending, s.onCallEventVerified, s.timeService.GetTimeNow().Add(24*time.Hour))
}

func (s *EthereumOracleVerifier) checkCallEventResult(contractCall types.EthContractCallEvent) error {
	spec, err := s.ethCallSpecSource.GetCall(contractCall.SpecId)
	if err != nil {
		return fmt.Errorf("failed to get call specification for id %s: %w", contractCall.SpecId, err)
	}

	blockHeight := &big.Int{}
	blockHeight.SetUint64(contractCall.BlockHeight)
	value, err := spec.Call(context.Background(), s.ethContractCaller, blockHeight)
	if err != nil {
		return fmt.Errorf("failed to execute call event spec: %w", err)
	}

	if !bytes.Equal(contractCall.Result, value) {
		return fmt.Errorf("mismatched results for block %d", contractCall.BlockHeight)
	}

	if !spec.PassesFilters(contractCall.Result, contractCall.BlockHeight, contractCall.BlockTime) {
		return fmt.Errorf("failed to pass filter check")
	}

	if err = s.ethConfirmations.Check(spec.RequiredConfirmations()); err != nil {
		return fmt.Errorf("failed confirmations check: %w", err)
	}

	return nil
}

func (s *EthereumOracleVerifier) removePendingCallEvent(id string) error {
	for i, v := range s.pendingCallEvents {
		if v.GetID() == id {
			s.pendingCallEvents = s.pendingCallEvents[:i+copy(s.pendingCallEvents[i:], s.pendingCallEvents[i+1:])]
			return nil
		}
	}
	return fmt.Errorf("invalid pending call event hash: %s", id)
}

func (s *EthereumOracleVerifier) onCallEventVerified(event interface{}, ok bool) {
	pv, isPendingCallEvent := event.(*pendingCallEvent)
	if !isPendingCallEvent {
		s.log.Errorf("expected pending call event go: %T", event)
		return
	}

	if err := s.removePendingCallEvent(pv.GetID()); err != nil {
		s.log.Error("could not remove pending stake deposited event", logging.Error(err))
	}

	if ok {
		s.finalizedCallResults = append(s.finalizedCallResults, &pv.callEvent)
	} else {
		s.log.Error("failed to verify call event")
	}
}

func (s *EthereumOracleVerifier) OnTick(ctx context.Context, t time.Time) {
	for _, callResult := range s.finalizedCallResults {
		spec, err := s.ethCallSpecSource.GetCall(callResult.SpecId)
		if err != nil {
			s.log.Error("failed to get spec for call result", logging.Error(err))
			continue
		}

		normalisedData, err := spec.Normalise(callResult.Result)
		if err != nil {
			s.log.Error("failed to normalise oracle data", logging.Error(err))
			continue
		}

		s.oracleEngine.BroadcastData(ctx, OracleData{
			Signers: nil,
			Data:    normalisedData,
		})
	}

	s.finalizedCallResults = nil
}

// TODO review and figure out the refactor to remove need for this.
type EthCallSpecSourceAdapter struct {
	Engine *ethcall.Engine
}

func (e *EthCallSpecSourceAdapter) GetCall(id string) (EthCall, error) {
	source, ok := e.Engine.GetDataSource(id)
	if !ok {
		return nil, fmt.Errorf("failed to get spec for id: %s", id)
	}
	return &EthCallSpecAdapter{spec: &source}, nil
}

type EthCallSpecAdapter struct {
	spec *ethcall.DataSource
}

func (e *EthCallSpecAdapter) Normalise(callResult []byte) (map[string]string, error) {
	return e.spec.Normalise(callResult)
}

func (e *EthCallSpecAdapter) Call(ctx context.Context, caller ethereum.ContractCaller, blockNumber *big.Int) ([]byte, error) {
	result, err := e.spec.Call.Call(ctx, caller, blockNumber)
	if err != nil {
		return nil, err
	}
	return result.Bytes, nil
}

func (e *EthCallSpecAdapter) PassesFilters(result []byte, blockHeight uint64, blockTime uint64) bool {
	return e.spec.Pass(result, blockHeight, blockTime)
}

func (e *EthCallSpecAdapter) RequiredConfirmations() uint64 {
	return e.spec.RequiredConfirmations()
}
