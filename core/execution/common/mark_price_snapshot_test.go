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

package common

import (
	"testing"

	dstypes "code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/types"
	vega "code.vegaprotocol.io/vega/protos/vega"
	datav1 "code.vegaprotocol.io/vega/protos/vega/data/v1"

	"github.com/stretchr/testify/require"
)

func TestSerialisation(t *testing.T) {
	pubKey := dstypes.CreateSignerFromString("0xDEADBEEF", dstypes.SignerTypePubKey)
	mpcProto := &vega.CompositePriceConfiguration{
		DecayWeight:              "0.1",
		DecayPower:               2,
		CashAmount:               "100",
		SourceStalenessTolerance: []string{"30s", "30s", "15s", "30s"},
		CompositePriceType:       types.CompositePriceTypeByMedian,
		DataSourcesSpec: []*vega.DataSourceDefinition{
			{
				SourceType: &vega.DataSourceDefinition_External{
					External: &vega.DataSourceDefinitionExternal{
						SourceType: &vega.DataSourceDefinitionExternal_Oracle{
							Oracle: &vega.DataSourceSpecConfiguration{
								Signers: []*datav1.Signer{pubKey.IntoProto()},
								Filters: []*datav1.Filter{
									{
										Key: &datav1.PropertyKey{
											Name: "ethereum.oracle.test.settlement_3DB2D971C6",
											Type: datav1.PropertyKey_TYPE_INTEGER,
										},
										Conditions: []*datav1.Condition{},
									},
								},
							},
						},
					},
				},
			},
		},
		DataSourcesSpecBinding: []*vega.SpecBindingForCompositePrice{
			{PriceSourceProperty: "ethereum.oracle.test.settlement_3DB2D971C6"},
		},
	}

	mpc := types.CompositePriceConfigurationFromProto(mpcProto)
	mpcProto2 := mpc.IntoProto()
	require.Equal(t, mpcProto, mpcProto2)
}
