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

package steps

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/cucumber/godog"
)

func TheLongBlockDurationTableIsUploaded(ctx context.Context, exec Execution, data *godog.Table) error {
	rows := parseLongBlockAuctionTable(data)
	tbl := &vega.LongBlockAuctionDurationTable{
		ThresholdAndDuration: make([]*vega.LongBlockAuction, 0, len(rows)),
	}
	for _, row := range rows {
		d := lbDuration{
			r: row,
		}
		d.validate()
		tbl.ThresholdAndDuration = append(tbl.ThresholdAndDuration, d.ToRow())
	}
	return exec.OnNetworkWideAuctionDurationUpdated(ctx, tbl)
}

func ThePreviousBlockDurationWas(ctx context.Context, exec Execution, duration string) error {
	prevDuration, err := time.ParseDuration(duration)
	if err != nil {
		return err
	}
	exec.BeginBlock(ctx, prevDuration)
	return nil
}

func parseLongBlockAuctionTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"threshold",
		"duration",
	}, []string{})
}

type lbDuration struct {
	r         RowWrapper
	Threshold time.Duration
	Duration  time.Duration
}

func (l *lbDuration) validate() {
	l.Threshold = l.r.MustDurationStr("threshold")
	l.Duration = l.r.MustDurationStr("duration")
}

func (l lbDuration) ToRow() *vega.LongBlockAuction {
	return &vega.LongBlockAuction{
		Threshold: l.Threshold.String(),
		Duration:  l.Duration.String(),
	}
}
