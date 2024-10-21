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
	"code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/stretchr/testify/require"
)

func TestStopOrdersProtoConversion(t *testing.T) {
	position := vega.StopOrder_SIZE_OVERRIDE_SETTING_POSITION
	none := vega.StopOrder_SIZE_OVERRIDE_SETTING_NONE

	submissionProto := &commandspb.StopOrdersSubmission{
		FallsBelow: &commandspb.StopOrderSetup{
			OrderSubmission: &commandspb.OrderSubmission{
				MarketId:    "123",
				Price:       "100",
				Size:        20,
				Side:        vega.Side_SIDE_BUY,
				TimeInForce: vega.Order_TIME_IN_FORCE_FOK,
				ExpiresAt:   12345,
				Type:        vega.Order_TYPE_LIMIT,
				Reference:   "ref_buy",
			},
			SizeOverrideSetting: &position,
			SizeOverrideValue:   &vega.StopOrder_SizeOverrideValue{Percentage: "0.5"},
		},
		RisesAbove: &commandspb.StopOrderSetup{
			OrderSubmission: &commandspb.OrderSubmission{
				MarketId:    "456",
				Price:       "200",
				Size:        10,
				Side:        vega.Side_SIDE_SELL,
				TimeInForce: vega.Order_TIME_IN_FORCE_GFA,
				ExpiresAt:   54321,
				Type:        vega.Order_TYPE_MARKET,
				Reference:   "ref_sell",
			},
			SizeOverrideSetting: &none,
			SizeOverrideValue:   nil,
		},
	}
	submission, err := types.NewStopOrderSubmissionFromProto(submissionProto)
	require.NoError(t, err)
	now := time.Date(2024, 3, 29, 10, 0, 0, 0, time.UTC)
	fallsBelow, risesAbove := submission.IntoStopOrders("party1", "party1", "1", "2", now)

	fallsBelowEvent := fallsBelow.ToProtoEvent()
	risesAboveEvent := risesAbove.ToProtoEvent()

	fallsBelowFromEvent := types.NewStopOrderFromProto(fallsBelowEvent)
	risesAboveFromEvent := types.NewStopOrderFromProto(risesAboveEvent)
	require.Equal(t, fallsBelow.String(), fallsBelowFromEvent.String())
	require.Equal(t, risesAbove.String(), risesAboveFromEvent.String())
}
