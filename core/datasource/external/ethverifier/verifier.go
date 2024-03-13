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
	"code.vegaprotocol.io/vega/core/datasource/external/ethcall"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/metrics"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/validators"
	"code.vegaprotocol.io/vega/logging"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"

	"github.com/emirpasic/gods/sets/treeset"
)

const keepHashesDuration = 24 * 2 * time.Hour

//go:generate go run github.com/golang/mock/mockgen -destination mocks/witness_mock.go -package mocks code.vegaprotocol.io/vega/core/datasource/external/ethverifier Witness
type Witness interface {
	StartCheckWithDelay(validators.Resource, func(interface{}, bool), time.Time, int64) error
	RestoreResource(validators.Resource, func(interface{}, bool)) error
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/oracle_data_broadcaster_mock.go -package mocks code.vegaprotocol.io/vega/core/datasource/external/ethverifier OracleDataBroadcaster
type OracleDataBroadcaster interface {
	BroadcastData(context.Context, common.Data) error
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/eth_confirmations_mock.go -package mocks code.vegaprotocol.io/vega/core/datasource/external/ethverifier EthereumConfirmations
type EthereumConfirmations interface {
	CheckRequiredConfirmations(block uint64, required uint64) error
	Check(block uint64) error
	GetConfirmations() uint64
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/ethcallengine_mock.go -package mocks code.vegaprotocol.io/vega/core/datasource/external/ethverifier EthCallEngine
type EthCallEngine interface {
	MakeResult(specID string, bytes []byte) (ethcall.Result, error)
	CallSpec(ctx context.Context, id string, atBlock uint64) (ethcall.Result, error)
	GetEthTime(ctx context.Context, atBlock uint64) (uint64, error)
	GetRequiredConfirmations(specId string) (uint64, error)
	GetInitialTriggerTime(id string) (uint64, error)
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
	isValidator      bool

	pendingCallEvents    []*pendingCallEvent
	finalizedCallResults []*ethcall.ContractCallEvent

	// the eth block height of the last seen ethereum TX
	lastBlock *types.EthBlock
	// the eth block height when we did the patch upgrade to fix the missing seen map
	patchBlock *types.EthBlock

	mu        sync.Mutex
	ackedEvts *ackedEvents
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
	isValidator bool,
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
		isValidator:      isValidator,
		ackedEvts: &ackedEvents{
			timeService: ts,
			events:      treeset.NewWith(ackedEvtBucketComparator),
		},
	}
	return s
}

func (s *Verifier) ensureNotTooOld(callEvent ethcall.ContractCallEvent) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.patchBlock != nil && callEvent.BlockHeight < s.patchBlock.Height {
		return false
	}

	tt := time.Unix(int64(callEvent.BlockTime), 0)
	removeBefore := s.timeService.GetTimeNow().Add(-keepHashesDuration)

	return !tt.Before(removeBefore)
}

func (s *Verifier) ensureNotDuplicate(callEvent ethcall.ContractCallEvent) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ackedEvts.Contains(callEvent.Hash()) {
		return false
	}

	s.ackedEvts.AddAt(int64(callEvent.BlockTime), callEvent.Hash())
	return true
}

func (s *Verifier) getConfirmations(callEvent ethcall.ContractCallEvent) (uint64, error) {
	if !callEvent.Heartbeat {
		return s.ethEngine.GetRequiredConfirmations(callEvent.SpecId)
	}

	if !s.isValidator {
		// non-validator doesn't know, and doesn't need to know
		return 0, nil
	}
	return s.ethConfirmations.GetConfirmations(), nil
}

// TODO: non finalized events could cause a memory leak, this needs to be addressed in a way that will prevent processing
// duplicates but not result in a memory leak, agreed to postpone for now  (other verifiers have the same issue)

func (s *Verifier) ProcessEthereumContractCallResult(callEvent ethcall.ContractCallEvent) error {
	if !s.ensureNotTooOld(callEvent) {
		s.log.Error("historic ethereum event received",
			logging.String("event", fmt.Sprintf("%+v", callEvent)))
		return errors.ErrEthereumCallEventTooOld
	}

	if ok := s.ensureNotDuplicate(callEvent); !ok {
		s.log.Error("ethereum call event already exists",
			logging.String("event", fmt.Sprintf("%+v", callEvent)))
		return errors.ErrDuplicatedEthereumCallEvent
	}

	pending := &pendingCallEvent{
		callEvent: callEvent,
		check:     func(ctx context.Context) error { return s.checkCallEventResult(ctx, callEvent) },
	}

	confirmations, err := s.getConfirmations(callEvent)
	if err != nil {
		return err
	}

	s.pendingCallEvents = append(s.pendingCallEvents, pending)

	s.log.Info("ethereum call event received, starting validation",
		logging.String("call-event", fmt.Sprintf("%+v", callEvent)))

	// Timeout for the check set to 1 day, to allow for validator outage scenarios
	err = s.witness.StartCheckWithDelay(
		pending, s.onCallEventVerified, s.timeService.GetTimeNow().Add(30*time.Minute), int64(confirmations))
	if err != nil {
		s.log.Error("could not start witness routine", logging.String("id", pending.GetID()))
		s.removePendingCallEvent(pending.GetID())
	}

	metrics.DataSourceEthVerifierCallGaugeAdd(1, callEvent.SpecId)

	return err
}

func (s *Verifier) checkCallEventResult(ctx context.Context, callEvent ethcall.ContractCallEvent) error {
	metrics.DataSourceEthVerifierCallCounterInc(callEvent.SpecId)

	// Ensure that the ethtime on the call event matches the block number on the eth chain
	// (submitting call events with malicious times could subvert, e.g. TWAPs on perp markets)
	checkedTime, err := s.ethEngine.GetEthTime(ctx, callEvent.BlockHeight)
	if err != nil {
		return fmt.Errorf("unable to verify eth time at %d", callEvent.BlockHeight)
	}

	if checkedTime != callEvent.BlockTime {
		return fmt.Errorf("call event for block time block %d alleges eth time %d - but found %d",
			callEvent.BlockHeight, callEvent.BlockTime, checkedTime)
	}

	if callEvent.Heartbeat {
		return s.ethConfirmations.Check(callEvent.BlockHeight)
	}

	metrics.DataSourceEthVerifierCallCounterInc(callEvent.SpecId)
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

	initialTriggerTime, err := s.ethEngine.GetInitialTriggerTime(callEvent.SpecId)
	if err != nil {
		return fmt.Errorf("failed to get initial trigger time: %w", err)
	}

	if callEvent.BlockTime < initialTriggerTime {
		return fmt.Errorf("call event block time %d is before the specification's initial time %d",
			callEvent.BlockTime, initialTriggerTime)
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
	} else {
		metrics.DataSourceEthVerifierCallGaugeAdd(-1, pv.callEvent.SpecId)
	}

	if ok {
		s.finalizedCallResults = append(s.finalizedCallResults, &pv.callEvent)
	} else {
		s.log.Error("failed to verify call event")
	}
}

func (s *Verifier) OnTick(ctx context.Context, t time.Time) {
	for _, callResult := range s.finalizedCallResults {
		if s.lastBlock == nil || callResult.BlockHeight > s.lastBlock.Height {
			s.lastBlock = &types.EthBlock{
				Height: callResult.BlockHeight,
				Time:   callResult.BlockTime,
			}
		}

		if callResult.Error == nil {
			result, err := s.ethEngine.MakeResult(callResult.SpecId, callResult.Result)
			if err != nil {
				s.log.Error("failed to create ethcall result", logging.Error(err))
			}

			s.oracleEngine.BroadcastData(ctx, common.Data{
				EthKey:  callResult.SpecId,
				Signers: nil,
				Data:    result.Normalised,
				MetaData: map[string]string{
					"eth-block-height": strconv.FormatUint(callResult.BlockHeight, 10),
					"eth-block-time":   strconv.FormatUint(callResult.BlockTime, 10),
					"vega-time":        strconv.FormatInt(t.Unix(), 10),
				},
			})
		} else {
			dataProto := vegapb.OracleData{
				ExternalData: &datapb.ExternalData{
					Data: &datapb.Data{
						MatchedSpecIds: []string{callResult.SpecId},
						BroadcastAt:    t.UnixNano(),
						Error:          callResult.Error,
						MetaData: []*datapb.Property{
							{
								Name:  "vega-time",
								Value: strconv.FormatInt(t.Unix(), 10),
							},
						},
					},
				},
			}

			s.broker.Send(events.NewOracleDataEvent(ctx, vegapb.OracleData{ExternalData: dataProto.ExternalData}))
		}
	}
	s.finalizedCallResults = nil

	// keep hashes for 2 days
	removeBefore := t.Add(-keepHashesDuration)
	s.ackedEvts.RemoveBefore(removeBefore.Unix())
}
