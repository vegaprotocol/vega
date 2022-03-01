package types

import (
	"encoding/hex"
	"strconv"

	vgproto "code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/libs/crypto"
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
		Address:     s.NewSigner,
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
		Address:     event.Signer,
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
	return s.IntoProto().String()
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
		Address:     s.OldSigner,
		Nonce:       s.Nonce,
		Kind:        SignerEventKindRemoved,
		BlockTime:   s.BlockTime,
	}, nil
}

type SignerThresholdSetEvent struct {
	BlockNumber, LogIndex uint64
	TxHash                string

	ID        string
	Threshold uint32
	Nonce     string
	BlockTime int64
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
	return s.IntoProto().String()
}
