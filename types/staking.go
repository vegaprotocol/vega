package types

import (
	"errors"
	"fmt"

	vgproto "code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/types/num"
)

type StakeLinkingType = eventspb.StakeLinking_Type

const (
	StakeLinkingTypeUnspecified StakeLinkingType = eventspb.StakeLinking_TYPE_UNSPECIFIED
	StakeLinkingTypeDeposited                    = eventspb.StakeLinking_TYPE_LINK
	StakeLinkingTypeRemoved                      = eventspb.StakeLinking_TYPE_UNLINK
)

type StakeLinkingStatus = eventspb.StakeLinking_Status

const (
	StakeLinkingStatusUnspecified StakeLinkingStatus = eventspb.StakeLinking_STATUS_UNSPECIFIED
	StakeLinkingStatusPending                        = eventspb.StakeLinking_STATUS_PENDING
	StakeLinkingStatusAccepted                       = eventspb.StakeLinking_STATUS_ACCEPTED
	StakeLinkingStatusRejected                       = eventspb.StakeLinking_STATUS_REJECTED
)

type StakeLinking struct {
	ID          string
	Type        StakeLinkingType
	TS          int64
	Party       string
	Amount      *num.Uint
	Status      StakeLinkingStatus
	FinalizedAt int64
	TxHash      string
}

func (s *StakeLinking) IntoProto() *eventspb.StakeLinking {
	return &eventspb.StakeLinking{
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
	amount, overflowed := num.UintFromString(s.Amount, 10)
	if overflowed {
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

func (s *StakeDeposited) IntoStakeLinking() *StakeLinking {
	return &StakeLinking{
		ID:     s.ID,
		Type:   StakeLinkingTypeDeposited,
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
	amount, overflowed := num.UintFromString(s.Amount, 10)
	if overflowed {
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

func (s *StakeRemoved) IntoStakeLinking() *StakeLinking {
	return &StakeLinking{
		ID:     s.ID,
		Type:   StakeLinkingTypeRemoved,
		TS:     s.BlockTime,
		Party:  s.VegaPubKey,
		Amount: s.Amount.Clone(),
		TxHash: s.TxID,
	}
}
