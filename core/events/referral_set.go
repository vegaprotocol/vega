package events

import (
	"context"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
	"golang.org/x/exp/slices"
)

type ReferralSetCreated struct {
	*Base
	e eventspb.ReferralSetCreated
}

func (t ReferralSetCreated) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(t.Base)
	busEvent.Event = &eventspb.BusEvent_ReferralSetCreated{
		ReferralSetCreated: &t.e,
	}

	return busEvent
}

func (t ReferralSetCreated) GetProtoEvent() *eventspb.ReferralSetCreated {
	return &t.e
}

func NewReferralSetCreatedEvent(ctx context.Context, set *types.ReferralSet) *ReferralSetCreated {
	return &ReferralSetCreated{
		Base: newBase(ctx, ReferralSetCreatedEvent),
		e: eventspb.ReferralSetCreated{
			SetId:     string(set.ID),
			Referrer:  string(set.Referrer.PartyID),
			CreatedAt: set.CreatedAt.UnixNano(),
			UpdatedAt: set.CreatedAt.UnixNano(),
		},
	}
}

func ReferralSetCreatedEventFromStream(ctx context.Context, be *eventspb.BusEvent) *ReferralSetCreated {
	return &ReferralSetCreated{
		Base: newBaseFromBusEvent(ctx, ReferralSetCreatedEvent, be),
		e:    *be.GetReferralSetCreated(),
	}
}

type ReferralSetStatsUpdated struct {
	*Base
	e eventspb.ReferralSetStatsUpdated
}

func (t ReferralSetStatsUpdated) Unwrap() *types.ReferralSetStats {
	volume, _ := num.UintFromString(t.e.ReferralSetRunningNotionalTakerVolume, 10)
	stats := map[types.PartyID]*types.RefereeStats{}
	rewardFactor, _ := num.DecimalFromString(t.e.RewardFactor)

	for _, stat := range t.e.RefereesStats {
		discountFactor, _ := num.DecimalFromString(stat.DiscountFactor)
		stats[types.PartyID(stat.PartyId)] = &types.RefereeStats{
			DiscountFactor: discountFactor,
		}
	}

	return &types.ReferralSetStats{
		AtEpoch:                  t.e.AtEpoch,
		SetID:                    types.ReferralSetID(t.e.SetId),
		ReferralSetRunningVolume: volume,
		RefereesStats:            stats,
		RewardFactor:             rewardFactor,
	}
}

func (t ReferralSetStatsUpdated) GetProtoEvent() *eventspb.ReferralSetStatsUpdated {
	return &t.e
}

func (t ReferralSetStatsUpdated) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(t.Base)
	busEvent.Event = &eventspb.BusEvent_ReferralSetStatsUpdated{
		ReferralSetStatsUpdated: &t.e,
	}

	return busEvent
}

func NewReferralSetStatsUpdatedEvent(ctx context.Context, update *types.ReferralSetStats) *ReferralSetStatsUpdated {
	refereesStats := make([]*eventspb.RefereeStats, 0, len(update.RefereesStats))
	for partyID, stat := range update.RefereesStats {
		refereesStats = append(refereesStats, &eventspb.RefereeStats{
			PartyId:                  string(partyID),
			DiscountFactor:           stat.DiscountFactor.String(),
			EpochNotionalTakerVolume: stat.TakerVolume.String(),
		})
	}

	slices.SortStableFunc(refereesStats, func(a, b *eventspb.RefereeStats) bool {
		return a.PartyId < b.PartyId
	})

	return &ReferralSetStatsUpdated{
		Base: newBase(ctx, ReferralSetStatsUpdatedEvent),
		e: eventspb.ReferralSetStatsUpdated{
			SetId:                                 string(update.SetID),
			AtEpoch:                               update.AtEpoch,
			ReferralSetRunningNotionalTakerVolume: update.ReferralSetRunningVolume.String(),
			RefereesStats:                         refereesStats,
			RewardFactor:                          update.RewardFactor.String(),
		},
	}
}

func ReferralSetStatsUpdatedEventFromStream(ctx context.Context, be *eventspb.BusEvent) *ReferralSetStatsUpdated {
	return &ReferralSetStatsUpdated{
		Base: newBaseFromBusEvent(ctx, ReferralSetStatsUpdatedEvent, be),
		e:    *be.GetReferralSetStatsUpdated(),
	}
}

type RefereeJoinedReferralSet struct {
	*Base
	e eventspb.RefereeJoinedReferralSet
}

func (t RefereeJoinedReferralSet) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(t.Base)
	busEvent.Event = &eventspb.BusEvent_RefereeJoinedReferralSet{
		RefereeJoinedReferralSet: &t.e,
	}

	return busEvent
}

func (t RefereeJoinedReferralSet) GetProtoEvent() *eventspb.RefereeJoinedReferralSet {
	return &t.e
}

func NewRefereeJoinedReferralSetEvent(ctx context.Context, setID types.ReferralSetID, membership *types.Membership) *RefereeJoinedReferralSet {
	return &RefereeJoinedReferralSet{
		Base: newBase(ctx, RefereeJoinedReferralSetEvent),
		e: eventspb.RefereeJoinedReferralSet{
			SetId:    string(setID),
			Referee:  string(membership.PartyID),
			JoinedAt: membership.JoinedAt.UnixNano(),
			AtEpoch:  membership.StartedAtEpoch,
		},
	}
}

func RefereeJoinedReferralSetEventFromStream(ctx context.Context, be *eventspb.BusEvent) *RefereeJoinedReferralSet {
	return &RefereeJoinedReferralSet{
		Base: newBaseFromBusEvent(ctx, RefereeJoinedReferralSetEvent, be),
		e:    *be.GetRefereeJoinedReferralSet(),
	}
}
