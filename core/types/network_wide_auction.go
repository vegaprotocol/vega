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
	"sort"
	"time"

	proto "code.vegaprotocol.io/vega/protos/vega"
)

type LongBlockAuctionDuration struct {
	threshold time.Duration
	duration  time.Duration
}

type LongBlockAuctionDurationTable struct {
	thresholdAndDuration []*LongBlockAuctionDuration
}

func LongBlockAuctionDurationTableFromProto(lbadTable *proto.LongBlockAuctionDurationTable) (*LongBlockAuctionDurationTable, error) {
	thresholdAndDuration := []*LongBlockAuctionDuration{}
	for _, lbad := range lbadTable.ThresholdAndDuration {
		threshold, err := time.ParseDuration(lbad.Threshold)
		if err != nil {
			return nil, err
		}
		duration, err := time.ParseDuration(lbad.Duration)
		if err != nil {
			return nil, err
		}
		thresholdAndDuration = append(thresholdAndDuration, &LongBlockAuctionDuration{threshold: threshold, duration: duration})
	}
	sort.Slice(thresholdAndDuration, func(i, j int) bool {
		return thresholdAndDuration[i].threshold.Nanoseconds() < thresholdAndDuration[j].threshold.Nanoseconds()
	})
	return &LongBlockAuctionDurationTable{thresholdAndDuration: thresholdAndDuration}, nil
}

func (b *LongBlockAuctionDurationTable) GetLongBlockAuctionDurationForBlockDuration(d time.Duration) *time.Duration {
	if len(b.thresholdAndDuration) == 0 {
		return nil
	}
	if d.Nanoseconds() < b.thresholdAndDuration[0].threshold.Nanoseconds() {
		return nil
	}
	auctionDuration := 0 * time.Second
	for _, td := range b.thresholdAndDuration {
		if d.Nanoseconds() < td.threshold.Nanoseconds() {
			break
		}
		auctionDuration = td.duration
	}
	return &auctionDuration
}
