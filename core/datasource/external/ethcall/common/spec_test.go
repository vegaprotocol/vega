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

package common_test

import (
	"errors"
	"fmt"
	"strconv"
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

		assert.Nil(t, s)
	})

	t.Run("non-empty with empty lists", func(t *testing.T) {
		protoSource := &vegapb.EthCallSpec{
			Abi:     "",
			Args:    nil,
			Filters: nil,
		}
		s, err := common.SpecFromProto(protoSource)
		assert.Error(t, errors.New(" error unmarshalling trigger: trigger proto is nil"), err)

		assert.Nil(t, s)
	})

	t.Run("non-empty with error", func(t *testing.T) {
		timeNow := uint64(time.Now().UnixNano())
		every := uint64(2)
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
						Every:   &every,
						Until:   &timeNow,
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
		assert.Nil(t, ds)

		protoSource.Args = nil
		ds, err = common.SpecFromProto(protoSource)
		assert.Nil(t, err)
		assert.IsType(t, &common.Spec{}, ds)

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
		tr := ds.Trigger.IntoTriggerProto().Trigger
		assert.IsType(t, &vegapb.EthCallTrigger_TimeTrigger{}, tr)
		assert.Equal(t, fmt.Sprintf("initial(%s) every(2) until(%s)", strconv.FormatUint(timeNow, 10), strconv.FormatUint(timeNow, 10)), ds.Trigger.String())
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
		assert.IsType(t, &common.Spec{}, ds)

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
