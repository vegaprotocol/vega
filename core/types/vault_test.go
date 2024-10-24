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
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/stretchr/testify/require"
)

func TestVaultIntoProto(t *testing.T) {
	v := &types.Vault{
		ID:    "1",
		Owner: "2",
		Asset: "3",
		MetaData: &vega.VaultMetaData{
			Name:        "4",
			Description: "5",
			Url:         "6",
			ImageUrl:    "7",
		},
		FeePeriod:            time.Hour,
		ManagementFeeFactor:  num.NewDecimalFromFloat(0.1),
		PerformanceFeeFactor: num.NewDecimalFromFloat(0.2),
		CutOffPeriodLength:   10,
		RedemptionDates: []*types.RedemptionDate{
			{
				RedemptionType: types.RedemptionTypeFreeCashOnly,
				RedemptionDate: time.Unix(1729451525, 0),
				MaxFraction:    num.NewDecimalFromFloat(0.5),
			},
			{
				RedemptionType: types.RedemptionTypeNormal,
				RedemptionDate: time.Unix(1729537925, 0),
				MaxFraction:    num.NewDecimalFromFloat(0.7),
			},
		},
	}

	protoV := v.IntoProto()
	require.Equal(t, "1", protoV.VaultId)
	require.Equal(t, "2", protoV.Owner)
	require.Equal(t, "3", protoV.Asset)
	require.Equal(t, "4", protoV.VaultMetadata.Name)
	require.Equal(t, "5", protoV.VaultMetadata.Description)
	require.Equal(t, "6", protoV.VaultMetadata.Url)
	require.Equal(t, "7", protoV.VaultMetadata.ImageUrl)
	require.Equal(t, "1h0m0s", protoV.FeePeriod)
	require.Equal(t, "0.1", protoV.ManagementFeeFactor)
	require.Equal(t, "0.2", protoV.PerformanceFeeFactor)
	require.Equal(t, int64(10), protoV.CutOffPeriodLength)
	require.Equal(t, 2, len(protoV.RedemptionDates))
	require.Equal(t, "0.5", protoV.RedemptionDates[0].MaxFraction)
	require.Equal(t, "0.7", protoV.RedemptionDates[1].MaxFraction)
	require.Equal(t, types.RedemptionTypeFreeCashOnly, protoV.RedemptionDates[0].RedemptionType)
	require.Equal(t, types.RedemptionTypeNormal, protoV.RedemptionDates[1].RedemptionType)
	require.Equal(t, int64(1729451525), protoV.RedemptionDates[0].RedemptionDate)
	require.Equal(t, int64(1729537925), protoV.RedemptionDates[1].RedemptionDate)

	newV := types.VaultFromProto(protoV)
	require.Equal(t, v.ID, newV.ID)
	require.Equal(t, v.Owner, newV.Owner)
	require.Equal(t, v.Asset, newV.Asset)
	require.Equal(t, v.MetaData, newV.MetaData)
	require.Equal(t, v.FeePeriod, newV.FeePeriod)
	require.Equal(t, v.ManagementFeeFactor, newV.ManagementFeeFactor)
	require.Equal(t, v.PerformanceFeeFactor, newV.PerformanceFeeFactor)
	require.Equal(t, v.CutOffPeriodLength, newV.CutOffPeriodLength)
	require.Equal(t, len(v.RedemptionDates), len(newV.RedemptionDates))
	for i := 0; i < len(v.RedemptionDates); i++ {
		require.Equal(t, v.RedemptionDates[i].MaxFraction, newV.RedemptionDates[i].MaxFraction)
		require.Equal(t, v.RedemptionDates[i].RedemptionDate, newV.RedemptionDates[i].RedemptionDate)
		require.Equal(t, v.RedemptionDates[i].RedemptionType, newV.RedemptionDates[i].RedemptionType)
	}
}
