// Copyright (c) 2023 Gobalsky Labs Limited
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

package ethverifier

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/datasource/errors"
	"code.vegaprotocol.io/vega/core/events"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"

	"code.vegaprotocol.io/vega/core/datasource/external/ethcall"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/validators"
	"code.vegaprotocol.io/vega/logging"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/witness_mock.go -package mocks code.vegaprotocol.io/vega/core/datasource/external/ethverifier Witness
type Witness interface {
	StartCheck(validators.Resource, func(interface{}, bool), time.Time) error
	RestoreResource(validators.Resource, func(interface{}, bool)) error
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/oracle_data_broadcaster_mock.go -package mocks code.vegaprotocol.io/vega/core/datasource/external/ethverifier OracleDataBroadcaster
type OracleDataBroadcaster interface {
	BroadcastData(context.Context, common.Data) error
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/eth_confirmations_mock.go -package mocks code.vegaprotocol.io/vega/core/datasource/external/ethverifier EthereumConfirmations
type EthereumConfirmations interface {
	CheckRequiredConfirmations(block uint64, required uint64) error
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/ethcallengine_mock.go -package mocks code.vegaprotocol.io/vega/core/datasource/external/ethverifier EthCallEngine
type EthCallEngine interface {
	MakeResult(specID string, bytes []byte) (ethcall.Result, error)
	CallSpec(ctx context.Context, id string, atBlock uint64) (ethcall.Result, error)
	GetRequiredConfirmations(specId string) (uint64, error)
	StartAtHeight(height uint64, timestamp uint64)
	Start()
}

type CallEngine interface {
	MakeResult(specID string, bytes []byte) (ethcall.Result, error)
	CallSpec(ctx context.Context, id string, atBlock uint64) (ethcall.Result, error)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/ethcall_result.go -package mocks code.vegaprotocol.io/vega/core/datasource/external/ethverifier Result
type Result interface {
	Bytes() []byte
	Values() ([]any, error)
	Normalised() (map[string]string, error)
	PassesFilters() (bool, error)
	HasRequiredConfirmations() bool
}

// Broker interface. Do not need to mock (use package broker/mock).
type Broker interface {
	Send(event events.Event)
	SendBatch(events []events.Event)
}

// TimeService interface.
//
//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/core/datasource/external/ethverifier TimeService
type TimeService interface {
	GetTimeNow() time.Time
}

type Verifier struct {
	log *logging.Logger

	witness          Witness
	timeService      TimeService
	broker           Broker
	oracleEngine     OracleDataBroadcaster
	ethEngine        EthCallEngine
	ethConfirmations EthereumConfirmations

	pendingCallEvents    []*pendingCallEvent
	finalizedCallResults []*ethcall.ContractCallEvent

	lastBlock *types.EthBlock

	mu     sync.Mutex
	hashes map[string]struct{}

	// snapshot data
	snapshotState *verifierSnapshotState
}

type pendingCallEvent struct {
	callEvent ethcall.ContractCallEvent
	check     func(ctx context.Context) error
}

func (p pendingCallEvent) GetID() string { return p.callEvent.Hash() }

func (p pendingCallEvent) GetType() types.NodeVoteType {
	return types.NodeVoteTypeEthereumContractCallResult
}
func (p *pendingCallEvent) Check(ctx context.Context) error { return p.check(ctx) }

func New(
	log *logging.Logger,
	witness Witness,
	ts TimeService,
	broker Broker,
	oracleBroadcaster OracleDataBroadcaster,
	ethCallEngine EthCallEngine,
	ethConfirmations EthereumConfirmations,
) (sv *Verifier) {
	log = log.Named("ethereum-oracle-verifier")
	s := &Verifier{
		log:              log,
		witness:          witness,
		timeService:      ts,
		broker:           broker,
		oracleEngine:     oracleBroadcaster,
		ethEngine:        ethCallEngine,
		ethConfirmations: ethConfirmations,
		hashes:           map[string]struct{}{},
		snapshotState:    &verifierSnapshotState{},
	}
	return s
}

func (s *Verifier) ensureNotDuplicate(hash string) bool {
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

func (s *Verifier) ProcessEthereumContractCallResult(callEvent ethcall.ContractCallEvent) error {
	if ok := s.ensureNotDuplicate(callEvent.Hash()); !ok {
		s.log.Error("ethereum call event already exists",
			logging.String("event", fmt.Sprintf("%+v", callEvent)))
		return errors.ErrDuplicatedEthereumCallEvent
	}

	s.lastBlock = &types.EthBlock{
		Height: callEvent.BlockHeight,
		Time:   callEvent.BlockTime,
	}

	pending := &pendingCallEvent{
		callEvent: callEvent,
		check:     func(ctx context.Context) error { return s.checkCallEventResult(ctx, callEvent) },
	}

	s.pendingCallEvents = append(s.pendingCallEvents, pending)

	s.log.Info("ethereum call event received, starting validation",
		logging.String("call-event", fmt.Sprintf("%+v", callEvent)))

	// Timeout for the check set to 1 day, to allow for validator outage scenarios
	err := s.witness.StartCheck(
		pending, s.onCallEventVerified, s.timeService.GetTimeNow().Add(24*time.Hour))
	if err != nil {
		s.log.Error("could not start witness routine", logging.String("id", pending.GetID()))
		s.removePendingCallEvent(pending.GetID())
	}

	return err
}

func (s *Verifier) checkCallEventResult(ctx context.Context, callEvent ethcall.ContractCallEvent) error {
	checkResult, err := s.ethEngine.CallSpec(ctx, callEvent.SpecId, callEvent.BlockHeight)
	if callEvent.Error != nil {
		if err != nil {
			if err.Error() == *callEvent.Error {
				return nil
			}
			return fmt.Errorf("error mismatch, expected %s, got %s", *callEvent.Error, err.Error())
		}

		return fmt.Errorf("call event has error %s, but no error returned from call spec", *callEvent.Error)
	} else if err != nil {
		return fmt.Errorf("failed to execute call event spec: %w", err)
	}

	if !bytes.Equal(callEvent.Result, checkResult.Bytes) {
		return fmt.Errorf("mismatched results for block %d", callEvent.BlockHeight)
	}

	requiredConfirmations, err := s.ethEngine.GetRequiredConfirmations(callEvent.SpecId)
	if err != nil {
		return fmt.Errorf("failed to get required confirmations: %w", err)
	}

	if err = s.ethConfirmations.CheckRequiredConfirmations(callEvent.BlockHeight, requiredConfirmations); err != nil {
		return fmt.Errorf("failed confirmations check: %w", err)
	}

	if !checkResult.PassesFilters {
		return fmt.Errorf("failed filter check")
	}

	return nil
}

func (s *Verifier) removePendingCallEvent(id string) error {
	for i, v := range s.pendingCallEvents {
		if v.GetID() == id {
			s.pendingCallEvents = s.pendingCallEvents[:i+copy(s.pendingCallEvents[i:], s.pendingCallEvents[i+1:])]
			return nil
		}
	}
	return fmt.Errorf("invalid pending call event hash: %s", id)
}

func (s *Verifier) onCallEventVerified(event interface{}, ok bool) {
	pv, isPendingCallEvent := event.(*pendingCallEvent)
	if !isPendingCallEvent {
		s.log.Errorf("expected pending call event go: %T", event)
		return
	}

	if err := s.removePendingCallEvent(pv.GetID()); err != nil {
		s.log.Error("could not remove pending call event", logging.Error(err))
	}

	if ok {
		s.finalizedCallResults = append(s.finalizedCallResults, &pv.callEvent)
	} else {
		s.log.Error("failed to verify call event")
	}
}

func (s *Verifier) OnTick(ctx context.Context, t time.Time) {
	for _, callResult := range s.finalizedCallResults {
		if callResult.Error == nil {
			result, err := s.ethEngine.MakeResult(callResult.SpecId, callResult.Result)
			if err != nil {
				s.log.Error("failed to create ethcall result", logging.Error(err))
			}

			s.oracleEngine.BroadcastData(ctx, common.Data{
				Signers: nil,
				Data:    result.Normalised,
				MetaData: map[string]string{
					"eth-block-height": strconv.FormatUint(callResult.BlockHeight, 10),
					"eth-block-time":   strconv.FormatUint(callResult.BlockTime, 10),
				},
			})
		} else {
			dataProto := vegapb.OracleData{
				ExternalData: &datapb.ExternalData{
					Data: &datapb.Data{
						MatchedSpecIds: []string{callResult.SpecId},
						BroadcastAt:    t.UnixNano(),
						Error:          callResult.Error,
					},
				},
			}

			s.broker.Send(events.NewOracleDataEvent(ctx, vegapb.OracleData{ExternalData: dataProto.ExternalData}))
		}
	}

	s.finalizedCallResults = nil
}
