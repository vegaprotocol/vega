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

package types_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	proto "code.vegaprotocol.io/vega/protos/vega"

	"github.com/stretchr/testify/require"
)

func TestInvalidThreshold(t *testing.T) {
	lbadTable := &proto.LongBlockAuctionDurationTable{
		ThresholdAndDuration: []*proto.LongBlockAuction{
			{Threshold: "-1", Duration: "2s"},
		},
	}
	_, err := types.LongBlockAuctionDurationTableFromProto(lbadTable)
	require.Error(t, err)

	lbadTable = &proto.LongBlockAuctionDurationTable{
		ThresholdAndDuration: []*proto.LongBlockAuction{
			{Threshold: "hjk", Duration: "2s"},
		},
	}
	_, err = types.LongBlockAuctionDurationTableFromProto(lbadTable)
	require.Error(t, err)

	lbadTable = &proto.LongBlockAuctionDurationTable{
		ThresholdAndDuration: []*proto.LongBlockAuction{
			{Threshold: "1s", Duration: "2s"},
			{Threshold: "banana", Duration: "2s"},
		},
	}
	_, err = types.LongBlockAuctionDurationTableFromProto(lbadTable)
	require.Error(t, err)
}

func TestInvalidDuration(t *testing.T) {
	lbadTable := &proto.LongBlockAuctionDurationTable{
		ThresholdAndDuration: []*proto.LongBlockAuction{
			{Threshold: "1s", Duration: "-2"},
		},
	}
	_, err := types.LongBlockAuctionDurationTableFromProto(lbadTable)
	require.Error(t, err)

	lbadTable = &proto.LongBlockAuctionDurationTable{
		ThresholdAndDuration: []*proto.LongBlockAuction{
			{Threshold: "1s", Duration: "hjk"},
		},
	}
	_, err = types.LongBlockAuctionDurationTableFromProto(lbadTable)
	require.Error(t, err)

	lbadTable = &proto.LongBlockAuctionDurationTable{
		ThresholdAndDuration: []*proto.LongBlockAuction{
			{Threshold: "1s", Duration: "2s"},
			{Threshold: "2s", Duration: "banana"},
		},
	}
	_, err = types.LongBlockAuctionDurationTableFromProto(lbadTable)
	require.Error(t, err)
}

func TestFindLongBlockDuration(t *testing.T) {
	lbadTable := &proto.LongBlockAuctionDurationTable{
		ThresholdAndDuration: []*proto.LongBlockAuction{
			{Threshold: "3s", Duration: "1m"},
			{Threshold: "40s", Duration: "10m"},
			{Threshold: "2m", Duration: "1h"},
		},
	}
	table, err := types.LongBlockAuctionDurationTableFromProto(lbadTable)
	require.NoError(t, err)

	require.Nil(t, table.GetLongBlockAuctionDurationForBlockDuration(1*time.Second))
	require.Nil(t, table.GetLongBlockAuctionDurationForBlockDuration(2*time.Second))
	require.Equal(t, int64(60), int64(table.GetLongBlockAuctionDurationForBlockDuration(3*time.Second).Seconds()))
	require.Equal(t, int64(60), int64(table.GetLongBlockAuctionDurationForBlockDuration(39*time.Second).Seconds()))
	require.Equal(t, int64(600), int64(table.GetLongBlockAuctionDurationForBlockDuration(40*time.Second).Seconds()))
	require.Equal(t, int64(600), int64(table.GetLongBlockAuctionDurationForBlockDuration(119*time.Second).Seconds()))
	require.Equal(t, int64(3600), int64(table.GetLongBlockAuctionDurationForBlockDuration(120*time.Second).Seconds()))
	require.Equal(t, int64(3600), int64(table.GetLongBlockAuctionDurationForBlockDuration(100000*time.Second).Seconds()))
}
