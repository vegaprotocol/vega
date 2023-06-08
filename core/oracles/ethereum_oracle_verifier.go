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

//go:generate go run github.com/golang/mock/mockgen -destination mocks/eth_call_mock.go -package mocks code.vegaprotocol.io/vega/core/oracles EthCall
type EthCall interface {
	GetDataSource(id string) (ethcall.DataSource, bool)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/eth_contract_caller.go -package mocks code.vegaprotocol.io/vega/core/oracles ContractCaller
type ContractCaller interface {
	CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/eth_confirmations_mock.go -package mocks code.vegaprotocol.io/vega/core/oracles EthereumConfirmations
type EthereumConfirmations interface {
	Check(uint64) error
}

type EthereumOracleVerifier struct {
	log *logging.Logger

	witness           Witness
	timeService       TimeService
	oracleEngine      OracleDataBroadcaster
	ethCall           EthCall
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
	ethCallSpecSource EthCall,
	ethContractCaller ethereum.ContractCaller,
	ethConfirmations EthereumConfirmations,
) (sv *EthereumOracleVerifier) {
	log = log.Named("ethereum-oracle-verifier")
	s := &EthereumOracleVerifier{
		log:               log,
		witness:           witness,
		timeService:       ts,
		oracleEngine:      oracleBroadcaster,
		ethCall:           ethCallSpecSource,
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
	dataSource, exists := s.ethCall.GetDataSource(contractCall.SpecId)
	if !exists {
		// It is possible though unlikely that this could happen if a spec is deactivated after the chain event is sent
		// but it would probably indicate incorrect behaviour.
		return fmt.Errorf("datasource for spec id %s does not exist", contractCall.SpecId)
	}

	blockHeight := &big.Int{}
	blockHeight.SetUint64(contractCall.BlockHeight)
	value, err := dataSource.CallContract(context.Background(), s.ethContractCaller, blockHeight)
	if err != nil {
		return fmt.Errorf("failed to execute call event dataSource: %w", err)
	}

	if !bytes.Equal(contractCall.Result, value) {
		return fmt.Errorf("mismatched results for block %d", contractCall.BlockHeight)
	}

	if !dataSource.PassesFilters(contractCall.Result, contractCall.BlockHeight, contractCall.BlockTime) {
		return fmt.Errorf("failed to pass filter check")
	}

	if err = s.ethConfirmations.Check(dataSource.RequiredConfirmations()); err != nil {
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
		dataSource, exists := s.ethCall.GetDataSource(callResult.SpecId)
		if !exists {
			// It is possible this could happen if a spec is deactivated after the chain event is sent
			// but it would probably indicate incorrect behaviour.
			s.log.Errorf("datasource for spec id %s does not exist", callResult.SpecId)
			continue
		}

		normalisedData, err := dataSource.Normalise(callResult.Result)
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
