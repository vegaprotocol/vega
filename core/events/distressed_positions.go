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

package events

import (
	"context"

	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

// DistressedPositions contains the market and parties that needed to have their orders closed in order
// to maintain their open positions on the market.
type DistressedPositions struct {
	*Base
	dpIDs map[string]struct{}
	spIDs map[string]struct{}

	pb eventspb.DistressedPositions
}

func NewDistressedPositionsEvent(ctx context.Context, marketID string, dParties, sParties []string) *DistressedPositions {
	dk := make(map[string]struct{}, len(dParties))
	sk := make(map[string]struct{}, len(sParties))
	ret := &DistressedPositions{
		Base:  newBase(ctx, DistressedPositionsEvent),
		dpIDs: dk,
		spIDs: sk,
		pb: eventspb.DistressedPositions{
			MarketId:          marketID,
			DistressedParties: make([]string, 0, len(dParties)),
			SafeParties:       make([]string, 0, len(sParties)),
		},
	}
	ret.AddDistressedParties(dParties...)
	ret.AddSafeParties(sParties...)
	return ret
}

func (d *DistressedPositions) AddDistressedParties(parties ...string) {
	// ensure parties cannot be marked as both safe and distressed
	d.rmSafe(parties)
	for _, p := range parties {
		// party already registered as distressed
		if _, ok := d.dpIDs[p]; ok {
			continue
		}
		// mark party as distressed, ensure we don't mark a party as both distressed and no-longer-distressed
		d.dpIDs[p] = struct{}{}
		d.pb.DistressedParties = append(d.pb.DistressedParties, p)
	}
}

func (d *DistressedPositions) AddSafeParties(parties ...string) {
	// ensure these parties aren't marked as distressed
	d.rmDistressed(parties)
	for _, p := range parties {
		if _, ok := d.spIDs[p]; ok {
			continue
		}
		d.spIDs[p] = struct{}{}
		d.pb.SafeParties = append(d.pb.SafeParties, p)
	}
}

func (d DistressedPositions) MarketID() string {
	return d.pb.MarketId
}

func (d *DistressedPositions) rmSafe(parties []string) {
	for _, p := range parties {
		if _, ok := d.spIDs[p]; ok {
			delete(d.spIDs, p)
			for i := 0; i < len(d.pb.SafeParties); i++ {
				if d.pb.SafeParties[i] == p {
					d.pb.SafeParties = append(d.pb.SafeParties[:i], d.pb.SafeParties[i+1:]...)
					break
				}
			}
		}
	}
}

func (d *DistressedPositions) rmDistressed(parties []string) {
	for _, p := range parties {
		if _, ok := d.dpIDs[p]; ok {
			delete(d.dpIDs, p)
			for i := 0; i < len(d.pb.DistressedParties); i++ {
				if d.pb.DistressedParties[i] == p {
					d.pb.DistressedParties = append(d.pb.DistressedParties[:i], d.pb.DistressedParties[i+1:]...)
					break
				}
			}
		}
	}
}

func (d DistressedPositions) DistressedParties() []string {
	return d.pb.DistressedParties
}

func (d DistressedPositions) SafeParties() []string {
	return d.pb.SafeParties
}

func (d DistressedPositions) IsMarket(marketID string) bool {
	return d.pb.MarketId == marketID
}

func (d DistressedPositions) IsParty(partyID string) bool {
	if _, ok := d.dpIDs[partyID]; ok {
		return true
	}
	_, ok := d.spIDs[partyID]
	return ok
}

func (d DistressedPositions) IsDistressedParty(p string) bool {
	_, ok := d.dpIDs[p]
	return ok
}

func (d DistressedPositions) IsSafeParty(p string) bool {
	_, ok := d.spIDs[p]
	return ok
}

func (d DistressedPositions) Proto() eventspb.DistressedPositions {
	return d.pb
}

func (d DistressedPositions) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(d.Base)
	cpy := d.pb
	busEvent.Event = &eventspb.BusEvent_DistressedPositions{
		DistressedPositions: &cpy,
	}

	return busEvent
}

func (d DistressedPositions) StreamMarketMessage() *eventspb.BusEvent {
	return d.StreamMessage()
}

func DistressedPositionsEventFromStream(ctx context.Context, be *eventspb.BusEvent) *DistressedPositions {
	m := be.GetDistressedPositions()
	dk := make(map[string]struct{}, len(m.DistressedParties))
	for _, p := range m.DistressedParties {
		dk[p] = struct{}{}
	}
	sk := make(map[string]struct{}, len(m.SafeParties))
	for _, p := range m.SafeParties {
		sk[p] = struct{}{}
	}
	return &DistressedPositions{
		Base:  newBaseFromBusEvent(ctx, DistressedPositionsEvent, be),
		dpIDs: dk,
		spIDs: sk,
		pb:    *m,
	}
}
