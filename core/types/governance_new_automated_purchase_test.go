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
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	v1 "code.vegaprotocol.io/vega/protos/vega/data/v1"

	"github.com/stretchr/testify/require"
)

func TestNewProtocolAutomatedPurchaseChangesFromProto(t *testing.T) {
	apc := &types.NewProtocolAutomatedPurchaseChanges{
		From:            "abc",
		FromAccountType: types.AccountTypeBuyBackFees,
		ToAccountType:   types.AccountTypeBuyBackFees,
		MarketID:        "def",
		PriceOracle: &vegapb.DataSourceDefinition{
			SourceType: &vegapb.DataSourceDefinition_External{
				External: &vegapb.DataSourceDefinitionExternal{
					SourceType: &vegapb.DataSourceDefinitionExternal_Oracle{
						Oracle: &vegapb.DataSourceSpecConfiguration{
							Signers: []*v1.Signer{
								{
									Signer: &v1.Signer_PubKey{
										PubKey: &v1.PubKey{
											Key: "0xiBADC0FFEE0DDF00D",
										},
									},
								},
							},
							Filters: []*v1.Filter{
								{
									Key: &v1.PropertyKey{
										Name:                "oracle.price",
										Type:                v1.PropertyKey_TYPE_INTEGER,
										NumberDecimalPlaces: ptr[uint64](5),
									},
									Conditions: []*v1.Condition{
										{
											Operator: v1.Condition_OPERATOR_UNSPECIFIED,
										},
									},
								},
							},
						},
					},
				},
			},
		},
		OracleOffsetFactor: num.MustDecimalFromString("0.99"),
		PriceOracleBinding: &vegapb.SpecBindingForCompositePrice{
			PriceSourceProperty: "oracle.price",
		},
		AuctionDuration: time.Hour,
		AuctionSchedule: &vegapb.DataSourceDefinition{
			SourceType: &vegapb.DataSourceDefinition_Internal{
				Internal: &vegapb.DataSourceDefinitionInternal{
					SourceType: &vegapb.DataSourceDefinitionInternal_TimeTrigger{
						TimeTrigger: &vegapb.DataSourceSpecConfigurationTimeTrigger{
							Triggers: []*v1.InternalTimeTrigger{
								{
									Initial: nil,
									Every:   1000,
								},
							},
						},
					},
				},
			},
		},
		AuctionVolumeSnapshotSchedule: &vegapb.DataSourceDefinition{
			SourceType: &vegapb.DataSourceDefinition_Internal{
				Internal: &vegapb.DataSourceDefinitionInternal{
					SourceType: &vegapb.DataSourceDefinitionInternal_TimeTrigger{
						TimeTrigger: &vegapb.DataSourceSpecConfigurationTimeTrigger{
							Triggers: []*v1.InternalTimeTrigger{
								{
									Initial: nil,
									Every:   800,
								},
							},
						},
					},
				},
			},
		},
		AutomatedPurchaseSpecBinding: &vegapb.DataSourceSpecToAutomatedPurchaseBinding{
			AuctionScheduleProperty:               "",
			AuctionVolumeSnapshotScheduleProperty: "",
		},
		MaximumAuctionSize: num.NewUint(1000),
		MinimumAuctionSize: num.NewUint(100),
		ExpiryTimestamp:    time.Now(),
	}

	apcProto := apc.IntoProto()
	apcFromProto := types.NewProtocolAutomatedPurchaseChangesFromProto(apcProto)
	apcProto2 := apcFromProto.IntoProto()
	require.Equal(t, apcProto.String(), apcProto2.String())
}

func TestAutomatedPurchaseChangesClone(t *testing.T) {
	apc := &types.NewProtocolAutomatedPurchaseChanges{
		From:            "abc",
		FromAccountType: types.AccountTypeBuyBackFees,
		ToAccountType:   types.AccountTypeBuyBackFees,
		MarketID:        "def",
		PriceOracle: &vegapb.DataSourceDefinition{
			SourceType: &vegapb.DataSourceDefinition_External{
				External: &vegapb.DataSourceDefinitionExternal{
					SourceType: &vegapb.DataSourceDefinitionExternal_Oracle{
						Oracle: &vegapb.DataSourceSpecConfiguration{
							Signers: []*v1.Signer{
								{
									Signer: &v1.Signer_PubKey{
										PubKey: &v1.PubKey{
											Key: "0xiBADC0FFEE0DDF00D",
										},
									},
								},
							},
							Filters: []*v1.Filter{
								{
									Key: &v1.PropertyKey{
										Name:                "oracle.price",
										Type:                v1.PropertyKey_TYPE_INTEGER,
										NumberDecimalPlaces: ptr[uint64](5),
									},
									Conditions: []*v1.Condition{
										{
											Operator: v1.Condition_OPERATOR_UNSPECIFIED,
										},
									},
								},
							},
						},
					},
				},
			},
		},
		OracleOffsetFactor: num.MustDecimalFromString("0.99"),
		PriceOracleBinding: &vegapb.SpecBindingForCompositePrice{
			PriceSourceProperty: "oracle.price",
		},
		AuctionDuration: time.Hour,
		AuctionSchedule: &vegapb.DataSourceDefinition{
			SourceType: &vegapb.DataSourceDefinition_Internal{
				Internal: &vegapb.DataSourceDefinitionInternal{
					SourceType: &vegapb.DataSourceDefinitionInternal_TimeTrigger{
						TimeTrigger: &vegapb.DataSourceSpecConfigurationTimeTrigger{
							Triggers: []*v1.InternalTimeTrigger{
								{
									Initial: nil,
									Every:   1000,
								},
							},
						},
					},
				},
			},
		},
		AuctionVolumeSnapshotSchedule: &vegapb.DataSourceDefinition{
			SourceType: &vegapb.DataSourceDefinition_Internal{
				Internal: &vegapb.DataSourceDefinitionInternal{
					SourceType: &vegapb.DataSourceDefinitionInternal_TimeTrigger{
						TimeTrigger: &vegapb.DataSourceSpecConfigurationTimeTrigger{
							Triggers: []*v1.InternalTimeTrigger{
								{
									Initial: nil,
									Every:   800,
								},
							},
						},
					},
				},
			},
		},
		AutomatedPurchaseSpecBinding: &vegapb.DataSourceSpecToAutomatedPurchaseBinding{
			AuctionScheduleProperty:               "",
			AuctionVolumeSnapshotScheduleProperty: "",
		},
		MaximumAuctionSize: num.NewUint(1000),
		MinimumAuctionSize: num.NewUint(100),
		ExpiryTimestamp:    time.Now(),
	}

	apcClone := apc.DeepClone()
	require.Equal(t, apc.IntoProto().String(), apcClone.IntoProto().String())
}
