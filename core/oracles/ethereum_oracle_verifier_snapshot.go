package oracles

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"

	"code.vegaprotocol.io/vega/libs/proto"
)

var (
	contractCall = (&types.PayloadEthContractCallEvent{}).Key()
	lastEthBlock = (&types.PayloadEthOracleLastBlock{}).Key()
	hashKeys     = []string{
		contractCall, lastEthBlock,
	}
)

type ethereumOracleVerifierSnapshotState struct {
	serialisedPendingCallEvents []byte
	serialisedLastEthBlock      []byte
}

func (s *EthereumOracleVerifier) serialisePendingContractCallEvents() ([]byte, error) {
	s.log.Info("serialising pending call events", logging.Int("n", len(s.pendingCallEvents)))
	pendingCallEvents := make([]*types.EthContractCallEvent, 0, len(s.pendingCallEvents))

	for _, p := range s.pendingCallEvents {
		pendingCallEvents = append(pendingCallEvents, &p.callEvent)
	}

	pl := types.Payload{
		Data: &types.PayloadEthContractCallEvent{
			EthContractCallEvent: pendingCallEvents,
		},
	}
	return proto.Marshal(pl.IntoProto())
}

func (s *EthereumOracleVerifier) serialiseLastEthBlock() ([]byte, error) {
	s.log.Info("serialising last eth block", logging.String("last-eth-block", fmt.Sprintf("%+v", s.lastBlock)))

	var pl types.Payload
	if s.lastBlock != nil {
		pl = types.Payload{
			Data: &types.PayloadEthOracleLastBlock{
				EthOracleLastBlock: &types.EthBlock{
					Height: s.lastBlock.Height,
					Time:   s.lastBlock.Time,
				},
			},
		}
	} else {
		pl = types.Payload{
			Data: &types.PayloadEthOracleLastBlock{},
		}
	}

	return proto.Marshal(pl.IntoProto())
}

func (s *EthereumOracleVerifier) serialiseK(serialFunc func() ([]byte, error), dataField *[]byte) ([]byte, error) {
	data, err := serialFunc()
	if err != nil {
		return nil, err
	}
	*dataField = data
	return data, nil
}

// get the serialised form and hash of the given key.
func (s *EthereumOracleVerifier) serialise(k string) ([]byte, error) {
	switch k {
	case contractCall:
		return s.serialiseK(s.serialisePendingContractCallEvents, &s.snapshotState.serialisedPendingCallEvents)
	case lastEthBlock:
		return s.serialiseK(s.serialiseLastEthBlock, &s.snapshotState.serialisedLastEthBlock)
	default:
		return nil, types.ErrSnapshotKeyDoesNotExist
	}
}

func (s *EthereumOracleVerifier) Namespace() types.SnapshotNamespace {
	return types.EthereumOracleVerifierSnapshot
}

func (s *EthereumOracleVerifier) Keys() []string {
	return hashKeys
}

func (s *EthereumOracleVerifier) Stopped() bool {
	return false
}

func (s *EthereumOracleVerifier) GetState(k string) ([]byte, []types.StateProvider, error) {
	data, err := s.serialise(k)
	return data, nil, err
}

func (s *EthereumOracleVerifier) LoadState(ctx context.Context, payload *types.Payload) ([]types.StateProvider, error) {
	if s.Namespace() != payload.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	switch pl := payload.Data.(type) {
	case *types.PayloadEthContractCallEvent:
		return nil, s.restorePendingCallEvents(ctx, pl.EthContractCallEvent, payload)
	case *types.PayloadEthOracleLastBlock:
		return nil, s.restoreLastEthBlock(pl.EthOracleLastBlock, payload)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (s *EthereumOracleVerifier) OnStateLoaded(ctx context.Context) error {
	// tell the eth call engine what the last block seen was so it does not retrigger calls
	if s.lastBlock != nil && s.lastBlock.Height > 0 {
		s.ethEngine.UpdatePreviousEthBlock(s.lastBlock.Height, s.lastBlock.Time)
	}

	return nil
}

func (s *EthereumOracleVerifier) restoreLastEthBlock(lastBlock *types.EthBlock, p *types.Payload) error {
	s.log.Info("restoring last eth block", logging.String("last-eth-block", fmt.Sprintf("%+v", lastBlock)))
	s.lastBlock = lastBlock

	var err error
	if s.snapshotState.serialisedLastEthBlock, err = proto.Marshal(p.IntoProto()); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	return nil
}

func (s *EthereumOracleVerifier) restorePendingCallEvents(_ context.Context,
	results []*types.EthContractCallEvent, p *types.Payload,
) error {
	s.log.Debug("restoring pending call events snapshot", logging.Int("n_pending", len(results)))
	s.pendingCallEvents = make([]*pendingCallEvent, 0, len(results))

	for _, callEvent := range results {
		// this populates the id/hash structs
		if !s.ensureNotDuplicate(callEvent.Hash()) {
			s.log.Panic("pendingCallEvents's unexpectedly pre-populated when restoring from snapshot")
		}

		pending := &pendingCallEvent{
			callEvent: *callEvent,
			check:     func(ctx context.Context) error { return s.checkCallEventResult(ctx, *callEvent) },
		}

		s.pendingCallEvents = append(s.pendingCallEvents, pending)

		if err := s.witness.RestoreResource(pending, s.onCallEventVerified); err != nil {
			s.log.Panic("unable to restore pending call event resource", logging.String("ID", pending.GetID()), logging.Error(err))
		}
	}

	var err error
	if s.snapshotState.serialisedPendingCallEvents, err = proto.Marshal(p.IntoProto()); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	return nil
}
