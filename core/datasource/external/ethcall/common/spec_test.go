// Copyright (c) 2023 Gobalsky Labs Limited
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

package common_test

import (
	"errors"
	"testing"
	"time"

	dscommon "code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/datasource/external/ethcall/common"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	v1 "code.vegaprotocol.io/vega/protos/vega/data/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestEthCallSpecFromProto(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		s, err := common.SpecFromProto(nil)
		assert.Error(t, errors.New("ethereum call spec proto is nil"), err)

		assert.NotNil(t, s)
		assert.IsType(t, common.Spec{}, s)

		assert.Equal(t, "", s.Address)
		assert.Equal(t, []byte(nil), s.AbiJson)
		assert.Nil(t, nil, s.Method)
		assert.Nil(t, s.ArgsJson)
		assert.Equal(t, 0, len(s.ArgsJson))
		assert.Nil(t, s.Filters)
		assert.Nil(t, s.Trigger)
	})

	t.Run("non-empty with empty lists", func(t *testing.T) {
		protoSource := &vegapb.EthCallSpec{
			Abi:     "",
			Args:    nil,
			Filters: nil,
		}
		s, err := common.SpecFromProto(protoSource)
		assert.Error(t, errors.New(" error unmarshalling trigger: trigger proto is nil"), err)

		assert.NotNil(t, s)
		assert.IsType(t, common.Spec{}, s)

		assert.Equal(t, "", s.Address)
		assert.Equal(t, []byte(nil), s.AbiJson)
		assert.Nil(t, nil, s.Method)
		assert.Nil(t, s.ArgsJson)
		assert.Equal(t, 0, len(s.ArgsJson))
		assert.Nil(t, s.Filters)
		assert.Nil(t, s.Trigger)
	})

	t.Run("non-empty with error", func(t *testing.T) {
		timeNow := uint64(time.Now().UnixNano())
		protoSource := &vegapb.EthCallSpec{
			Address: "test-eth-address",
			Abi:     "5",
			Method:  "test-method",
			Args: []*structpb.Value{
				{},
			},
			Trigger: &vegapb.EthCallTrigger{
				Trigger: &vegapb.EthCallTrigger_TimeTrigger{
					TimeTrigger: &vegapb.EthTimeTrigger{
						Initial: &timeNow,
					},
				},
			},
			Filters: []*v1.Filter{
				{
					Key: &v1.PropertyKey{
						Name: "test-key",
						Type: v1.PropertyKey_Type(1),
					},
				},
			},
		}

		ds, err := common.SpecFromProto(protoSource)
		assert.Error(t, errors.New("error marshalling arg: proto: google.protobuf.Value: none of the oneof fields is set"), err)
		assert.IsType(t, common.Spec{}, ds)

		assert.Equal(t, "", ds.Address)
		assert.Equal(t, []byte(nil), ds.AbiJson)
		assert.Nil(t, nil, ds.Method)
		assert.Nil(t, ds.ArgsJson)
		assert.Equal(t, 0, len(ds.ArgsJson))
		assert.Nil(t, ds.Filters)
		assert.Nil(t, ds.Trigger)

		protoSource.Args = nil
		ds, err = common.SpecFromProto(protoSource)
		assert.Nil(t, err)
		assert.IsType(t, common.Spec{}, ds)

		assert.Equal(t, "test-eth-address", ds.Address)
		assert.Equal(t, []byte("5"), ds.AbiJson)
		assert.Equal(t, "test-method", ds.Method)
		assert.NotNil(t, ds.ArgsJson)
		assert.Equal(t, 0, len(ds.ArgsJson))
		assert.NotNil(t, ds.Filters)
		assert.Equal(t, 1, len(ds.Filters))
		assert.Equal(t, "test-key", ds.Filters[0].Key.Name)
		assert.Equal(t, dscommon.SpecPropertyKeyType(1), ds.Filters[0].Key.Type)
		assert.NotNil(t, ds.Trigger)
		assert.IsType(t, &vegapb.EthCallTrigger_TimeTrigger{}, ds.Trigger.IntoTriggerProto().Trigger)
	})

	t.Run("non-empty", func(t *testing.T) {
		timeNow := uint64(time.Now().UnixNano())
		protoSource := &vegapb.EthCallSpec{
			Address: "test-eth-address",
			Abi:     "5",
			Method:  "test-method",
			Args: []*structpb.Value{
				structpb.NewStringValue("test-arg-value"),
			},
			Trigger: &vegapb.EthCallTrigger{
				Trigger: &vegapb.EthCallTrigger_TimeTrigger{
					TimeTrigger: &vegapb.EthTimeTrigger{
						Initial: &timeNow,
					},
				},
			},
			Filters: []*v1.Filter{
				{
					Key: &v1.PropertyKey{
						Name: "test-key",
						Type: v1.PropertyKey_Type(1),
					},
				},
			},
		}

		ds, err := common.SpecFromProto(protoSource)
		assert.Nil(t, err)
		assert.IsType(t, common.Spec{}, ds)

		assert.Equal(t, "test-eth-address", ds.Address)
		assert.Equal(t, []byte("5"), ds.AbiJson)
		assert.Equal(t, "test-method", ds.Method)
		assert.Equal(t, []string{"\"test-arg-value\""}, ds.ArgsJson)
		filters := ds.Filters
		assert.Equal(t, 1, len(filters))
		assert.Equal(t, 0, len(filters[0].Conditions))
		assert.Equal(t, "test-key", filters[0].Key.Name)
		assert.Equal(t, v1.PropertyKey_Type(1), filters[0].Key.Type)
		assert.NotNil(t, ds.Trigger)
		assert.IsType(t, &vegapb.EthCallTrigger_TimeTrigger{}, ds.Trigger.IntoTriggerProto().Trigger)
	})
}

func TestEthCallSpecIntoProto(t *testing.T) {
	// Implicitly tested with TestDataSourceDefinitionIntoProto.
}

func TestEthCallSpecToDataSourceDefinitionProto(t *testing.T) {
	// Implicitly tested with TestDataSourceDefinitionIntoProto.
}
