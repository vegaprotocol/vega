package entities

import (
	"fmt"
	"math"
	"time"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"github.com/shopspring/decimal"
)

type StakeLinkingID struct {
	ID
}

func NewStakeLinkingID(id string) StakeLinkingID {
	return StakeLinkingID{
		ID: ID(id),
	}
}

type StakeLinking struct {
	ID                 StakeLinkingID
	StakeLinkingType   StakeLinkingType
	EthereumTimestamp  time.Time
	PartyID            PartyID
	Amount             decimal.Decimal
	StakeLinkingStatus StakeLinkingStatus
	FinalizedAt        time.Time
	TxHash             string
	LogIndex           int64
	EthereumAddress    string
	VegaTime           time.Time
}

func StakeLinkingFromProto(stake *eventspb.StakeLinking, vegaTime time.Time) (*StakeLinking, error) {
	id := NewStakeLinkingID(stake.Id)
	partyID := NewPartyID(stake.Party)
	amount, err := decimal.NewFromString(stake.Amount)
	if err != nil {
		return nil, fmt.Errorf("received invalid staking amount: %s - %w", stake.Amount, err)
	}
	if stake.BlockHeight > math.MaxInt64 {
		return nil, fmt.Errorf("block height is too high: %d", stake.BlockHeight)
	}
	if stake.LogIndex > math.MaxInt64 {
		return nil, fmt.Errorf("log index is too hight: %d", stake.LogIndex)
	}

	logIndex := int64(stake.LogIndex)

	return &StakeLinking{
		ID:                 id,
		StakeLinkingType:   StakeLinkingType(stake.Type),
		EthereumTimestamp:  time.Unix(stake.Ts, 0),
		PartyID:            partyID,
		Amount:             amount,
		StakeLinkingStatus: StakeLinkingStatus(stake.Status),
		FinalizedAt:        time.Unix(0, stake.FinalizedAt),
		TxHash:             stake.TxHash,
		LogIndex:           logIndex,
		EthereumAddress:    stake.EthereumAddress,
		VegaTime:           vegaTime,
	}, nil
}

func (s *StakeLinking) ToProto() *eventspb.StakeLinking {
	return &eventspb.StakeLinking{
		Id:              s.ID.String(),
		Type:            eventspb.StakeLinking_Type(s.StakeLinkingType),
		Ts:              s.EthereumTimestamp.Unix(),
		Party:           s.PartyID.String(),
		Amount:          s.Amount.String(),
		Status:          eventspb.StakeLinking_Status(s.StakeLinkingStatus),
		FinalizedAt:     s.FinalizedAt.UnixNano(),
		TxHash:          s.TxHash,
		LogIndex:        uint64(s.LogIndex),
		EthereumAddress: s.EthereumAddress,
	}
}
