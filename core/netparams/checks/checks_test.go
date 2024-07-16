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

package checks_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/netparams/checks"
	types "code.vegaprotocol.io/vega/protos/vega"

	"github.com/stretchr/testify/require"
)

func TestLongBlockChecks(t *testing.T) {
	// wrong type - error
	type Junk struct{}
	require.Equal(t, "invalid long block auction duration table", checks.LongBlockAuctionDurationTable()(&Junk{}, &Junk{}).Error())

	// empty - is fine
	table := &types.LongBlockAuctionDurationTable{}
	require.NoError(t, checks.LongBlockAuctionDurationTable()(table, nil))

	// invalid threshold at index 0 - error
	table.ThresholdAndDuration = []*types.LongBlockAuction{{Threshold: "banana", Duration: "1s"}}
	require.Equal(t, "invalid long block auction duration table - threshold at index 0 is not a valid duration", checks.LongBlockAuctionDurationTable()(table, nil).Error())

	table.ThresholdAndDuration = []*types.LongBlockAuction{{Threshold: "-1", Duration: "1s"}}
	require.Equal(t, "invalid long block auction duration table - threshold at index 0 is not a valid duration", checks.LongBlockAuctionDurationTable()(table, nil).Error())

	table.ThresholdAndDuration = []*types.LongBlockAuction{{Threshold: "-1s", Duration: "1s"}}
	require.Equal(t, "invalid long block auction duration table - threshold at index 0 is not a positive duration", checks.LongBlockAuctionDurationTable()(table, nil).Error())

	table.ThresholdAndDuration = []*types.LongBlockAuction{{Threshold: "0s", Duration: "1s"}}
	require.Equal(t, "invalid long block auction duration table - threshold at index 0 is not a positive duration", checks.LongBlockAuctionDurationTable()(table, nil).Error())

	// invalid duration a index 0 - error
	table.ThresholdAndDuration = []*types.LongBlockAuction{{Threshold: "1s", Duration: "banana"}}
	require.Equal(t, "invalid long block auction duration table - duration at index 0 is not a valid duration", checks.LongBlockAuctionDurationTable()(table, nil).Error())

	table.ThresholdAndDuration = []*types.LongBlockAuction{{Threshold: "1s", Duration: "-1"}}
	require.Equal(t, "invalid long block auction duration table - duration at index 0 is not a valid duration", checks.LongBlockAuctionDurationTable()(table, nil).Error())

	table.ThresholdAndDuration = []*types.LongBlockAuction{{Threshold: "1s", Duration: "-1s"}}
	require.Equal(t, "invalid long block auction duration table - duration at index 0 is not a positive duration", checks.LongBlockAuctionDurationTable()(table, nil).Error())

	table.ThresholdAndDuration = []*types.LongBlockAuction{{Threshold: "1s", Duration: "0s"}}
	require.Equal(t, "invalid long block auction duration table - duration at index 0 is not a positive duration", checks.LongBlockAuctionDurationTable()(table, nil).Error())

	table.ThresholdAndDuration = []*types.LongBlockAuction{{Threshold: "1s", Duration: "0.5s"}}
	require.Equal(t, "invalid long block auction duration table - duration at index 0 is less than one second", checks.LongBlockAuctionDurationTable()(table, nil).Error())

	// invalid threshold at index 1 - error
	table.ThresholdAndDuration = []*types.LongBlockAuction{{Threshold: "1s", Duration: "1s"}, {Threshold: "banana", Duration: "1s"}}
	require.Equal(t, "invalid long block auction duration table - threshold at index 1 is not a valid duration", checks.LongBlockAuctionDurationTable()(table, nil).Error())

	table.ThresholdAndDuration = []*types.LongBlockAuction{{Threshold: "1s", Duration: "1s"}, {Threshold: "-1", Duration: "1s"}}
	require.Equal(t, "invalid long block auction duration table - threshold at index 1 is not a valid duration", checks.LongBlockAuctionDurationTable()(table, nil).Error())

	table.ThresholdAndDuration = []*types.LongBlockAuction{{Threshold: "1s", Duration: "1s"}, {Threshold: "-1s", Duration: "1s"}}
	require.Equal(t, "invalid long block auction duration table - threshold at index 1 is not a positive duration", checks.LongBlockAuctionDurationTable()(table, nil).Error())

	table.ThresholdAndDuration = []*types.LongBlockAuction{{Threshold: "1s", Duration: "1s"}, {Threshold: "0s", Duration: "1s"}}
	require.Equal(t, "invalid long block auction duration table - threshold at index 1 is not a positive duration", checks.LongBlockAuctionDurationTable()(table, nil).Error())

	// invalid duration a index 1 - error
	table.ThresholdAndDuration = []*types.LongBlockAuction{{Threshold: "1s", Duration: "1s"}, {Threshold: "1s", Duration: "banana"}}
	require.Equal(t, "invalid long block auction duration table - duration at index 1 is not a valid duration", checks.LongBlockAuctionDurationTable()(table, nil).Error())

	table.ThresholdAndDuration = []*types.LongBlockAuction{{Threshold: "1s", Duration: "1s"}, {Threshold: "1s", Duration: "-1"}}
	require.Equal(t, "invalid long block auction duration table - duration at index 1 is not a valid duration", checks.LongBlockAuctionDurationTable()(table, nil).Error())

	table.ThresholdAndDuration = []*types.LongBlockAuction{{Threshold: "1s", Duration: "1s"}, {Threshold: "1s", Duration: "-1s"}}
	require.Equal(t, "invalid long block auction duration table - duration at index 1 is not a positive duration", checks.LongBlockAuctionDurationTable()(table, nil).Error())

	table.ThresholdAndDuration = []*types.LongBlockAuction{{Threshold: "1s", Duration: "1s"}, {Threshold: "1s", Duration: "0s"}}
	require.Equal(t, "invalid long block auction duration table - duration at index 1 is not a positive duration", checks.LongBlockAuctionDurationTable()(table, nil).Error())

	// duplicate threshold - error
	table.ThresholdAndDuration = []*types.LongBlockAuction{{Threshold: "1s", Duration: "1s"}, {Threshold: "1s", Duration: "100s"}}
	require.Equal(t, "invalid long block auction duration table - duplicate threshold", checks.LongBlockAuctionDurationTable()(table, nil).Error())
}
