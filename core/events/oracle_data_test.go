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

package events_test

import (
	"context"
	"testing"

	dstypes "code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/events"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"

	"github.com/stretchr/testify/assert"
)

func TestOracleDataDeepClone(t *testing.T) {
	ctx := context.Background()
	pubKeys := []*dstypes.Signer{
		dstypes.CreateSignerFromString("PK1", dstypes.SignerTypePubKey),
		dstypes.CreateSignerFromString("PK2", dstypes.SignerTypePubKey),
		dstypes.CreateSignerFromString("PK3", dstypes.SignerTypePubKey),
	}

	od := datapb.ExternalData{
		Data: &datapb.Data{
			Signers: dstypes.SignersIntoProto(pubKeys),
			Data: []*datapb.Property{
				{
					Name:  "Name",
					Value: "Value",
				},
			},
			MatchedSpecIds: []string{
				"MS1", "MS2",
			},
			BroadcastAt: 10000,
		},
	}

	odEvent := events.NewOracleDataEvent(ctx, vegapb.OracleData{ExternalData: &od})

	od2 := odEvent.OracleData()

	// Change the original values
	pk1 := dstypes.CreateSignerFromString("Changed1", dstypes.SignerTypePubKey)
	pk2 := dstypes.CreateSignerFromString("Changed2", dstypes.SignerTypePubKey)
	pk3 := dstypes.CreateSignerFromString("Changed3", dstypes.SignerTypePubKey)

	od.Data.Signers[0] = pk1.IntoProto()
	od.Data.Signers[1] = pk2.IntoProto()
	od.Data.Signers[2] = pk3.IntoProto()
	od.Data.Data[0].Name = "Changed"
	od.Data.Data[0].Value = "Changed"
	od.Data.MatchedSpecIds[0] = "Changed1"
	od.Data.MatchedSpecIds[1] = "Changed2"
	od.Data.BroadcastAt = 999

	// Check things have changed
	assert.NotEqual(t, od.Data.Signers[0], od2.ExternalData.Data.Signers[0])

	assert.NotEqual(t, od.Data.Signers[1], od2.ExternalData.Data.Signers[1])
	assert.NotEqual(t, od.Data.Signers[2], od2.ExternalData.Data.Signers[2])
	assert.NotEqual(t, od.Data.Data[0].Name, od2.ExternalData.Data.Data[0].Name)
	assert.NotEqual(t, od.Data.Data[0].Value, od2.ExternalData.Data.Data[0].Value)
	assert.NotEqual(t, od.Data.MatchedSpecIds[0], od2.ExternalData.Data.MatchedSpecIds[0])
	assert.NotEqual(t, od.Data.MatchedSpecIds[1], od2.ExternalData.Data.MatchedSpecIds[1])
	assert.NotEqual(t, od.Data.BroadcastAt, od2.ExternalData.Data.BroadcastAt)
}
