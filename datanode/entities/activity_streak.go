package entities

import (
	"encoding/json"
	"fmt"
	"time"

	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type PartyActivityStreak struct {
	PartyID                              PartyID
	ActiveFor                            uint64
	InactiveFor                          uint64
	IsActive                             bool
	RewardDistributionActivityMultiplier string
	RewardVestingActivityMultiplier      string
	Epoch                                uint64
	TradedVolume                         string
	OpenVolume                           string
	VegaTime                             time.Time
	TxHash                               TxHash
}

func (pas *PartyActivityStreak) Fields() []interface{} {
	return []interface{}{
		pas.PartyID, pas.ActiveFor, pas.InactiveFor, pas.IsActive, pas.RewardDistributionActivityMultiplier, pas.RewardVestingActivityMultiplier, pas.Epoch, pas.TradedVolume, pas.OpenVolume, pas.VegaTime, pas.TxHash,
	}
}

func NewPartyActivityStreakFromProto(
	ev *eventspb.PartyActivityStreak,
	txHash TxHash,
	t time.Time,
) *PartyActivityStreak {
	return &PartyActivityStreak{
		PartyID:                              PartyID(ev.Party),
		ActiveFor:                            ev.ActiveFor,
		InactiveFor:                          ev.InactiveFor,
		IsActive:                             ev.IsActive,
		RewardDistributionActivityMultiplier: ev.RewardDistributionActivityMultiplier,
		RewardVestingActivityMultiplier:      ev.RewardVestingActivityMultiplier,
		Epoch:                                ev.Epoch,
		TradedVolume:                         ev.TradedVolume,
		OpenVolume:                           ev.OpenVolume,
		VegaTime:                             t,
		TxHash:                               txHash,
	}
}

func (pas *PartyActivityStreak) ToProto() *eventspb.PartyActivityStreak {
	return &eventspb.PartyActivityStreak{
		Party:                                pas.PartyID.String(),
		ActiveFor:                            pas.ActiveFor,
		InactiveFor:                          pas.InactiveFor,
		IsActive:                             pas.IsActive,
		RewardDistributionActivityMultiplier: pas.RewardDistributionActivityMultiplier,
		RewardVestingActivityMultiplier:      pas.RewardVestingActivityMultiplier,
		Epoch:                                pas.Epoch,
		TradedVolume:                         pas.TradedVolume,
		OpenVolume:                           pas.OpenVolume,
	}
}

func (pas PartyActivityStreak) Cursor() *Cursor {
	return NewCursor(
		PartyActivityStreakCursor{
			Party: pas.PartyID,
			Epoch: pas.Epoch,
		}.String(),
	)
}

type PartyActivityStreakCursor struct {
	Party PartyID `json:"party"`
	Epoch uint64  `json:"epoch"`
}

func (c PartyActivityStreakCursor) String() string {
	bs, err := json.Marshal(c)
	if err != nil {
		panic(fmt.Errorf("could not marshal party activity streak cursor: %w", err))
	}
	return string(bs)
}

func (c *PartyActivityStreakCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), c)
}
