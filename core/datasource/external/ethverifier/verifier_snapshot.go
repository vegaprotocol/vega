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
	"context"
	"fmt"
	"slices"

	"code.vegaprotocol.io/vega/core/datasource/external/ethcall"
	"code.vegaprotocol.io/vega/core/metrics"
	"code.vegaprotocol.io/vega/core/types"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"golang.org/x/exp/maps"
)

var (
	contractCall = (&types.PayloadEthContractCallEvent{}).Key()
	lastEthBlock = (&types.PayloadEthOracleLastBlock{}).Key()
	misc         = (&types.PayloadEthVerifierMisc{}).Key()
	hashKeys     = []string{
		contractCall, lastEthBlock, misc,
	}
)

func (s *Verifier) pendingContractCallEventsPayloadData() *types.PayloadEthContractCallEvent {
	pendingCallEvents := make([]*ethcall.ContractCallEvent, 0, len(s.pendingCallEvents))

	for _, p := range s.pendingCallEvents {
		pendingCallEvents = append(pendingCallEvents, &p.callEvent)
	}

	return &types.PayloadEthContractCallEvent{
		EthContractCallEvent: pendingCallEvents,
	}
}

func (s *Verifier) serialisePendingContractCallEvents() ([]byte, error) {
	s.log.Info("serialising pending call events", logging.Int("n", len(s.pendingCallEvents)))

	pl := types.Payload{
		Data: s.pendingContractCallEventsPayloadData(),
	}

	return proto.Marshal(pl.IntoProto())
}

func (s *Verifier) ethBlockPayloadData(bl *types.EthBlock) *types.PayloadEthOracleLastBlock {
	if bl != nil {
		return &types.PayloadEthOracleLastBlock{
			EthOracleLastBlock: &types.EthBlock{
				Height: bl.Height,
				Time:   bl.Time,
			},
		}
	}

	return &types.PayloadEthOracleLastBlock{}
}

func (s *Verifier) serialiseLastEthBlock() ([]byte, error) {
	s.log.Info("serialising last eth block", logging.String("last-eth-block", fmt.Sprintf("%+v", s.lastBlock)))

	pl := types.Payload{
		Data: s.ethBlockPayloadData(s.lastBlock),
	}

	return proto.Marshal(pl.IntoProto())
}

func (s *Verifier) serialiseMisc() ([]byte, error) {
	s.log.Info("serialising last eth block", logging.String("last-eth-block", fmt.Sprintf("%+v", s.lastBlock)))

	slice := make([]*snapshotpb.EthVerifierBucket, 0, s.ackedEvts.Size())
	iter := s.ackedEvts.events.Iterator()
	for iter.Next() {
		v := (iter.Value().(*ackedEvtBucket))
		hashes := maps.Keys(v.hashes)
		slices.Sort(hashes)
		slice = append(slice, &snapshotpb.EthVerifierBucket{
			Ts:     v.ts,
			Hashes: hashes,
		})
	}

	pl := types.Payload{
		Data: &types.PayloadEthVerifierMisc{
			Misc: &snapshotpb.EthOracleVerifierMisc{
				PatchBlock: s.ethBlockPayloadData(s.patchBlock).IntoProto().EthOracleVerifierLastBlock,
				Buckets:    slice,
			},
		},
	}

	return proto.Marshal(pl.IntoProto())
}

func (s *Verifier) serialiseK(serialFunc func() ([]byte, error)) ([]byte, error) {
	data, err := serialFunc()
	if err != nil {
		return nil, err
	}
	return data, nil
}

// get the serialised form and hash of the given key.
func (s *Verifier) serialise(k string) ([]byte, error) {
	switch k {
	case contractCall:
		return s.serialiseK(s.serialisePendingContractCallEvents)
	case lastEthBlock:
		return s.serialiseK(s.serialiseLastEthBlock)
	case misc:
		return s.serialiseK(s.serialiseMisc)
	default:
		return nil, types.ErrSnapshotKeyDoesNotExist
	}
}

func (s *Verifier) Namespace() types.SnapshotNamespace {
	return types.EthereumOracleVerifierSnapshot
}

func (s *Verifier) Keys() []string {
	return hashKeys
}

func (s *Verifier) Stopped() bool {
	return false
}

func (s *Verifier) GetState(k string) ([]byte, []types.StateProvider, error) {
	data, err := s.serialise(k)
	return data, nil, err
}

func (s *Verifier) LoadState(ctx context.Context, payload *types.Payload) ([]types.StateProvider, error) {
	if s.Namespace() != payload.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	switch pl := payload.Data.(type) {
	case *types.PayloadEthContractCallEvent:
		s.restorePendingCallEvents(ctx, pl.EthContractCallEvent)
		return nil, nil
	case *types.PayloadEthOracleLastBlock:
		s.restoreLastEthBlock(ctx, pl.EthOracleLastBlock)
		return nil, nil
	case *types.PayloadEthVerifierMisc:
		s.restoreMisc(ctx, pl.Misc)
		return nil, nil
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (s *Verifier) OnStateLoaded(ctx context.Context) error {
	if vgcontext.InProgressUpgradeFrom(ctx, "v0.74.7") {
		s.patchBlock = s.lastBlock
	}

	// tell the eth call engine what the last block seen was, so it does not re-trigger calls
	if s.lastBlock != nil && s.lastBlock.Height > 0 {
		s.ethEngine.StartAtHeight(s.lastBlock.Height, s.lastBlock.Time)
	} else {
		s.ethEngine.Start()
	}

	return nil
}

func (s *Verifier) restoreSeen(ctx context.Context, buckets []*snapshotpb.EthVerifierBucket) {
	// if we are executing a protocol upgrade,
	// let's force bucketing things. This will reduce
	// increase performance at startup, and everyone is starting
	// from the same snapshot, so that will keep state consistent
	if vgcontext.InProgressUpgrade(ctx) {
		for _, v := range buckets {
			s.ackedEvts.AddAt(v.Ts, v.Hashes...)
		}
		return
	}

	for _, v := range buckets {
		s.ackedEvts.RestoreExactAt(v.Ts, v.Hashes...)
	}
}

func (s *Verifier) restoreLastEthBlock(_ context.Context, lastBlock *types.EthBlock) {
	s.log.Info("restoring last eth block", logging.String("last-eth-block", fmt.Sprintf("%+v", lastBlock)))
	s.lastBlock = lastBlock
}

func (s *Verifier) restorePatchBlock(_ context.Context, patchBlock *types.EthBlock) {
	s.log.Info("restoring patch eth block", logging.String("patch-block", fmt.Sprintf("%+v", patchBlock)))

	// we have no history of what eth events we've seen from before this patch, so we will reject
	// any that come in that are older
	s.patchBlock = patchBlock
}

func (s *Verifier) restoreMisc(ctx context.Context, pl *snapshotpb.EthOracleVerifierMisc) {
	if pl.PatchBlock != nil {
		s.patchBlock = &types.EthBlock{
			Height: pl.PatchBlock.BlockHeight,
			Time:   pl.PatchBlock.BlockTime,
		}
	}
	s.restoreSeen(ctx, pl.Buckets)
}

func (s *Verifier) restorePendingCallEvents(ctx context.Context,
	results []*ethcall.ContractCallEvent,
) {
	s.log.Debug("restoring pending call events snapshot", logging.Int("n_pending", len(results)))
	s.pendingCallEvents = make([]*pendingCallEvent, 0, len(results))

	// clear up all the metrics
	seenSpecId := map[string]struct{}{}

	for _, callEvent := range results {
		if _, ok := seenSpecId[callEvent.SpecId]; !ok {
			metrics.DataSourceEthVerifierCallGaugeReset(callEvent.SpecId)
			seenSpecId[callEvent.SpecId] = struct{}{}
		}

		// if we've upgraded from the patch we need to add the pending events in, but after the upgrade
		// we don't need to because they will already be there
		if vgcontext.InProgressUpgradeFrom(ctx, "v0.74.7") {
			if !s.ensureNotDuplicate(*callEvent) {
				s.log.Panic("pendingCallEvents's unexpectedly pre-populated when restoring from snapshot")
			}
		}

		pending := &pendingCallEvent{
			callEvent: *callEvent,
			check:     func(ctx context.Context) error { return s.checkCallEventResult(ctx, *callEvent) },
		}

		s.pendingCallEvents = append(s.pendingCallEvents, pending)

		if err := s.witness.RestoreResource(pending, s.onCallEventVerified); err != nil {
			s.log.Panic("unable to restore pending call event resource", logging.String("ID", pending.GetID()), logging.Error(err))
		}

		metrics.DataSourceEthVerifierCallGaugeAdd(1, callEvent.SpecId)
	}
}
