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

package execution

import (
	"context"
	"sort"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/execution/common"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"golang.org/x/exp/maps"
)

type MarketCommunityTags struct {
	// market id -> set of tags
	tags   map[string]map[string]struct{}
	broker common.Broker
}

func NewMarketCommunityTags(broker common.Broker) *MarketCommunityTags {
	return &MarketCommunityTags{
		tags:   map[string]map[string]struct{}{},
		broker: broker,
	}
}

func NewMarketCommunityTagFromProto(
	broker common.Broker,
	state []*eventspb.MarketCommunityTags,
) *MarketCommunityTags {
	m := NewMarketCommunityTags(broker)

	for _, v := range state {
		m.tags[v.MarketId] = map[string]struct{}{}
		for _, t := range v.Tags {
			m.tags[v.MarketId][t] = struct{}{}
		}
	}

	return m
}

func (m *MarketCommunityTags) serialize() []*eventspb.MarketCommunityTags {
	out := make([]*eventspb.MarketCommunityTags, 0, len(m.tags))

	for mkt, tags := range m.tags {
		mct := &eventspb.MarketCommunityTags{
			MarketId: mkt,
			Tags:     make([]string, 0, len(tags)),
		}

		for tag := range tags {
			mct.Tags = append(mct.Tags, tag)
		}

		sort.Strings(mct.Tags)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].MarketId < out[j].MarketId })

	return out
}

// UpdateTags by that point the tags have been validated in length,
// so no need to do that again.
func (m *MarketCommunityTags) UpdateTags(
	ctx context.Context,
	market string,
	addTags []string,
	removeTags []string,
) {
	tags, ok := m.tags[market]
	if !ok {
		tags = map[string]struct{}{}
	}

	for _, t := range addTags {
		tags[t] = struct{}{}
	}

	for _, t := range removeTags {
		delete(tags, t)
	}

	evt := eventspb.MarketCommunityTags{
		MarketId: market,
		Tags:     maps.Keys(tags),
	}

	sort.Strings(evt.Tags)
	m.broker.Send(events.NewMarketCommunityTagsEvent(ctx, evt))
}
