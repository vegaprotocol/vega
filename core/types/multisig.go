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

package types

import (
	"encoding/hex"
	"fmt"
	"strconv"

	"code.vegaprotocol.io/vega/libs/crypto"
	vgproto "code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type SignerEventKind = eventspb.ERC20MultiSigSignerEvent_Type

const (
	SignerEventKindAdded   SignerEventKind = eventspb.ERC20MultiSigSignerEvent_TYPE_ADDED
	SignerEventKindRemoved                 = eventspb.ERC20MultiSigSignerEvent_TYPE_REMOVED
)

type SignerEvent struct {
	BlockNumber, LogIndex uint64
	TxHash                string

	ID        string
	Address   string
	Nonce     string
	BlockTime int64
	ChainID   string

	Kind SignerEventKind
}

func (s SignerEvent) Hash() string {
	var kind string
	switch s.Kind {
	case SignerEventKindAdded:
		kind = "signer_added"
	case SignerEventKindRemoved:
		kind = "signer_removed"
	}
	bn, li := strconv.FormatUint(s.BlockNumber, 10), strconv.FormatUint(s.LogIndex, 10)
	return hex.EncodeToString(
		crypto.Hash(
			[]byte(bn + li + s.TxHash + s.Address + s.Nonce + kind),
		),
	)
}

func SignerEventFromSignerAddedProto(
	s *vgproto.ERC20SignerAdded,
	blockNumber, logIndex uint64,
	txhash, id, chainID string,
) (*SignerEvent, error) {
	return &SignerEvent{
		ID:          id,
		BlockNumber: blockNumber,
		LogIndex:    logIndex,
		TxHash:      txhash,
		Address:     crypto.EthereumChecksumAddress(s.NewSigner),
		Nonce:       s.Nonce,
		Kind:        SignerEventKindAdded,
		BlockTime:   s.BlockTime,
		ChainID:     chainID,
	}, nil
}

func SignerEventFromEventProto(
	event *eventspb.ERC20MultiSigSignerEvent,
) *SignerEvent {
	return &SignerEvent{
		ID:          event.Id,
		BlockNumber: event.BlockNumber,
		LogIndex:    event.LogIndex,
		TxHash:      event.TxHash,
		Address:     crypto.EthereumChecksumAddress(event.Signer),
		Nonce:       event.Nonce,
		Kind:        event.Type,
		BlockTime:   event.BlockTime,
		ChainID:     event.ChainId,
	}
}

func (s *SignerEvent) IntoProto() *eventspb.ERC20MultiSigSignerEvent {
	return &eventspb.ERC20MultiSigSignerEvent{
		Id:          s.ID,
		Type:        s.Kind,
		Nonce:       s.Nonce,
		Signer:      s.Address,
		BlockTime:   s.BlockTime,
		TxHash:      s.TxHash,
		BlockNumber: s.BlockNumber,
		LogIndex:    s.LogIndex,
		ChainId:     s.ChainID,
	}
}

func (s *SignerEvent) String() string {
	return fmt.Sprintf(
		"blockNumber(%v) txHash(%s) ID(%s) address(%s) nonce(%s) blockTime(%v) kind(%s) chainID(%s)",
		s.BlockNumber,
		s.TxHash,
		s.ID,
		s.Address,
		s.Nonce,
		s.BlockTime,
		s.Kind.String(),
		s.ChainID,
	)
}

func SignerEventFromSignerRemovedProto(
	s *vgproto.ERC20SignerRemoved,
	blockNumber, logIndex uint64,
	txhash, id, chainID string,
) (*SignerEvent, error) {
	return &SignerEvent{
		ID:          id,
		BlockNumber: blockNumber,
		LogIndex:    logIndex,
		TxHash:      txhash,
		Address:     crypto.EthereumChecksumAddress(s.OldSigner),
		Nonce:       s.Nonce,
		Kind:        SignerEventKindRemoved,
		BlockTime:   s.BlockTime,
		ChainID:     chainID,
	}, nil
}

type SignerThresholdSetEvent struct {
	BlockNumber, LogIndex uint64
	TxHash                string

	ID        string
	Threshold uint32
	Nonce     string
	BlockTime int64
	ChainID   string
}

func (s SignerThresholdSetEvent) Hash() string {
	bn, li := strconv.FormatUint(s.BlockNumber, 10), strconv.FormatUint(s.LogIndex, 10)
	return hex.EncodeToString(
		crypto.Hash(
			[]byte(bn + li + s.TxHash + s.Nonce),
		),
	)
}

func SignerThresholdSetEventFromProto(
	s *vgproto.ERC20ThresholdSet,
	blockNumber, logIndex uint64,
	txhash, id, chainID string,
) (*SignerThresholdSetEvent, error) {
	return &SignerThresholdSetEvent{
		ID:          id,
		BlockNumber: blockNumber,
		LogIndex:    logIndex,
		TxHash:      txhash,
		Threshold:   s.NewThreshold,
		Nonce:       s.Nonce,
		BlockTime:   s.BlockTime,
		ChainID:     chainID,
	}, nil
}

func SignerThresholdSetEventFromEventProto(
	event *eventspb.ERC20MultiSigThresholdSetEvent,
) *SignerThresholdSetEvent {
	return &SignerThresholdSetEvent{
		ID:          event.Id,
		BlockNumber: event.BlockNumber,
		LogIndex:    event.LogIndex,
		TxHash:      event.TxHash,
		Threshold:   event.NewThreshold,
		Nonce:       event.Nonce,
		BlockTime:   event.BlockTime,
		ChainID:     event.ChainId,
	}
}

func (s *SignerThresholdSetEvent) IntoProto() *eventspb.ERC20MultiSigThresholdSetEvent {
	return &eventspb.ERC20MultiSigThresholdSetEvent{
		Id:           s.ID,
		NewThreshold: s.Threshold,
		Nonce:        s.Nonce,
		BlockTime:    s.BlockTime,
		TxHash:       s.TxHash,
		BlockNumber:  s.BlockNumber,
		LogIndex:     s.LogIndex,
		ChainId:      s.ChainID,
	}
}

func (s *SignerThresholdSetEvent) String() string {
	return fmt.Sprintf(
		"ID(%s) blockNumber(%v) logIndex(%v) txHash(%s) threshold(%v) nonce(%s) blockTime(%v) chainID(%s)",
		s.ID,
		s.BlockNumber,
		s.LogIndex,
		s.TxHash,
		s.Threshold,
		s.Nonce,
		s.BlockTime,
		s.ChainID,
	)
}
