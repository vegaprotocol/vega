// Copyright (c) 2022 Gobalsky Labs Limited
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
	TxHash string

	ID                    string
	Address               string
	Nonce                 string
	BlockNumber, LogIndex uint64
	BlockTime             int64

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
	txhash, id string,
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
	}
}

func (s *SignerEvent) String() string {
	return fmt.Sprintf(
		"blockNumber(%v) txHash(%s) ID(%s) address(%s) nonce(%s) blockTime(%v) kind(%s) ",
		s.BlockNumber,
		s.TxHash,
		s.ID,
		s.Address,
		s.Nonce,
		s.BlockTime,
		s.Kind.String(),
	)
}

func SignerEventFromSignerRemovedProto(
	s *vgproto.ERC20SignerRemoved,
	blockNumber, logIndex uint64,
	txhash, id string,
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
	}, nil
}

type SignerThresholdSetEvent struct {
	TxHash string

	ID                    string
	Nonce                 string
	BlockNumber, LogIndex uint64
	BlockTime             int64
	Threshold             uint32
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
	txhash, id string,
) (*SignerThresholdSetEvent, error) {
	return &SignerThresholdSetEvent{
		ID:          id,
		BlockNumber: blockNumber,
		LogIndex:    logIndex,
		TxHash:      txhash,
		Threshold:   s.NewThreshold,
		Nonce:       s.Nonce,
		BlockTime:   s.BlockTime,
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
	}
}

func (s *SignerThresholdSetEvent) String() string {
	return fmt.Sprintf(
		"ID(%s) blockNumber(%v) logIndex(%v) txHash(%s) threshold(%v) nonce(%s) blockTime(%v)",
		s.ID,
		s.BlockNumber,
		s.LogIndex,
		s.TxHash,
		s.Threshold,
		s.Nonce,
		s.BlockTime,
	)
}
