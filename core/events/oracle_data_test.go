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

package events_test

import (
	"context"
	"testing"

	oraclespb "code.vegaprotocol.io/protos/vega/oracles/v1"
	"code.vegaprotocol.io/vega/core/events"
	"github.com/stretchr/testify/assert"
)

func TestOracleDataDeepClone(t *testing.T) {
	ctx := context.Background()

	od := &oraclespb.OracleData{
		PubKeys: []string{"PK1", "PK2", "PK3"},
		Data: []*oraclespb.Property{
			{
				Name:  "Name",
				Value: "Value",
			},
		},
		MatchedSpecIds: []string{
			"MS1", "MS2",
		},
		BroadcastAt: 10000,
	}

	odEvent := events.NewOracleDataEvent(ctx, *od)
	od2 := odEvent.OracleData()

	// Change the original values
	od.PubKeys[0] = "Changed1"
	od.PubKeys[1] = "Changed2"
	od.PubKeys[2] = "Changed3"
	od.Data[0].Name = "Changed"
	od.Data[0].Value = "Changed"
	od.MatchedSpecIds[0] = "Changed1"
	od.MatchedSpecIds[1] = "Changed2"
	od.BroadcastAt = 999

	// Check things have changed
	assert.NotEqual(t, od.PubKeys[0], od2.PubKeys[0])
	assert.NotEqual(t, od.PubKeys[1], od2.PubKeys[1])
	assert.NotEqual(t, od.PubKeys[2], od2.PubKeys[2])
	assert.NotEqual(t, od.Data[0].Name, od2.Data[0].Name)
	assert.NotEqual(t, od.Data[0].Value, od2.Data[0].Value)
	assert.NotEqual(t, od.MatchedSpecIds[0], od2.MatchedSpecIds[0])
	assert.NotEqual(t, od.MatchedSpecIds[1], od2.MatchedSpecIds[1])
	assert.NotEqual(t, od.BroadcastAt, od2.BroadcastAt)
}
