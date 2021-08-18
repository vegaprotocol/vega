package types

import (
	"errors"
	"fmt"

	vgproto "code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/types/num"
)

type StakingEventType = eventspb.StakingEvent_Type

const (
	StakingEventTypeUnspecified = eventspb.StakingEvent_TYPE_UNSPECIFIED
	StakingEventTypeDeposited   = eventspb.StakingEvent_TYPE_DEPOSIT
	StakingEventTypeRemoved     = eventspb.StakingEvent_TYPE_REMOVE
)

type StakingEventStatus = eventspb.StakingEvent_Status

const (
	StakingEventStatusUnspecified = eventspb.StakingEvent_STATUS_UNSPECIFIED
	StakingEventStatusPending     = eventspb.StakingEvent_STATUS_PENDING
	StakingEventStatusAccepted    = eventspb.StakingEvent_STATUS_ACCEPTED
	StakingEventStatusRejected    = eventspb.StakingEvent_STATUS_REJECTED
)

type StakingEvent struct {
	ID          string
	Type        StakingEventType
	TS          int64
	Party       string
	Amount      *num.Uint
	Status      StakingEventStatus
	FinalizedAt int64
	TxHash      string
}

func (s *StakingEvent) IntoProto() *eventspb.StakingEvent {
	return &eventspb.StakingEvent{
		Id:          s.ID,
		Type:        s.Type,
		Ts:          s.TS,
		Party:       s.Party,
		Amount:      num.UintToString(s.Amount),
		Status:      s.Status,
		FinalizedAt: s.FinalizedAt,
		TxHash:      s.TxHash,
	}
}

type StakeDeposited struct {
	BlockNumber, LogIndex uint64
	TxID                  string // hash

	ID              string
	VegaPubKey      string
	EthereumAddress string
	Amount          *num.Uint
	BlockTime       int64
}

func StakeDepositedFromProto(
	s *vgproto.StakeDeposited,
	blockNumber, logIndex uint64,
	txID, id string,
) (*StakeDeposited, error) {
	amount, ok := num.UintFromString(s.Amount, 10)
	if !ok {
		return nil, errors.New("invalid amount (not a base 10 uint)")
	}

	return &StakeDeposited{
		ID:              id,
		BlockNumber:     blockNumber,
		LogIndex:        logIndex,
		TxID:            txID,
		VegaPubKey:      s.VegaPublicKey,
		EthereumAddress: s.EthereumAddress,
		Amount:          amount,
		BlockTime:       s.BlockTime,
	}, nil
}

func (s *StakeDeposited) IntoStakingEvent() *StakingEvent {
	return &StakingEvent{
		ID:     s.ID,
		Type:   StakingEventTypeDeposited,
		TS:     s.BlockTime,
		Party:  s.VegaPubKey,
		Amount: s.Amount.Clone(),
		TxHash: s.TxID,
	}
}

func (s StakeDeposited) String() string {
	return fmt.Sprintf("%#v", s)
}

type StakeRemoved struct {
	BlockNumber, LogIndex uint64
	TxID                  string // hash

	ID              string
	VegaPubKey      string
	EthereumAddress string
	Amount          *num.Uint
	BlockTime       int64
}

func StakeRemovedFromProto(
	s *vgproto.StakeRemoved,
	blockNumber, logIndex uint64,
	txID, id string,
) (*StakeRemoved, error) {
	amount, ok := num.UintFromString(s.Amount, 10)
	if !ok {
		return nil, errors.New("invalid amount (not a base 10 uint)")
	}

	return &StakeRemoved{
		ID:              id,
		BlockNumber:     blockNumber,
		LogIndex:        logIndex,
		TxID:            txID,
		VegaPubKey:      s.VegaPublicKey,
		EthereumAddress: s.EthereumAddress,
		Amount:          amount,
		BlockTime:       s.BlockTime,
	}, nil
}

func (s StakeRemoved) String() string {
	return fmt.Sprintf("%#v", s)
}

func (s *StakeRemoved) IntoStakingEvent() *StakingEvent {
	return &StakingEvent{
		ID:     s.ID,
		Type:   StakingEventTypeRemoved,
		TS:     s.BlockTime,
		Party:  s.VegaPubKey,
		Amount: s.Amount.Clone(),
		TxHash: s.TxID,
	}
}
