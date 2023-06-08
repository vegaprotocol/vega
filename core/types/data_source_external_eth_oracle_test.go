package types_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	v1 "code.vegaprotocol.io/vega/protos/vega/data/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestEthCallSpecFromProto(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		s, err := types.EthCallSpecFromProto(nil)
		assert.Error(t, errors.New("ethereum call spec proto is nil"), err)

		assert.NotNil(t, s)
		assert.IsType(t, types.EthCallSpec{}, s)

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
			Abi:     nil,
			Args:    nil,
			Filters: nil,
		}
		s, err := types.EthCallSpecFromProto(protoSource)
		assert.Error(t, errors.New(" error unmarshalling trigger: trigger proto is nil"), err)

		assert.NotNil(t, s)
		assert.IsType(t, types.EthCallSpec{}, s)

		assert.Equal(t, "", s.Address)
		assert.Equal(t, []byte(nil), s.AbiJson)
		assert.Nil(t, nil, s.Method)
		assert.Nil(t, s.ArgsJson)
		assert.Equal(t, 0, len(s.ArgsJson))
		assert.Nil(t, s.Filters)
		assert.Nil(t, s.Trigger)
	})

	t.Run("non-empty", func(t *testing.T) {
		timeNow := uint64(time.Now().UnixNano())
		protoSource := &vegapb.EthCallSpec{
			Address: "test-eth-address",
			Abi: &structpb.ListValue{
				Values: []*structpb.Value{
					{
						Kind: &structpb.Value_NumberValue{
							NumberValue: float64(5),
						},
					},
				},
			},
			Method: "test-method",
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

		ds, err := types.EthCallSpecFromProto(protoSource)
		assert.Error(t, errors.New("error marshalling arg: proto: google.protobuf.Value: none of the oneof fields is set"), err)
		assert.IsType(t, types.EthCallSpec{}, ds)

		assert.Equal(t, "", ds.Address)
		assert.Equal(t, []byte(nil), ds.AbiJson)
		assert.Nil(t, nil, ds.Method)
		assert.Nil(t, ds.ArgsJson)
		assert.Equal(t, 0, len(ds.ArgsJson))
		assert.Nil(t, ds.Filters)
		assert.Nil(t, ds.Trigger)

		protoSource.Args = nil
		ds, err = types.EthCallSpecFromProto(protoSource)
		assert.Nil(t, err)
		assert.IsType(t, types.EthCallSpec{}, ds)

		assert.Equal(t, "test-eth-address", ds.Address)
		assert.Equal(t, []byte{91, 53, 93}, ds.AbiJson)
		assert.Equal(t, "test-method", ds.Method)
		assert.NotNil(t, ds.ArgsJson)
		assert.Equal(t, 0, len(ds.ArgsJson))
		assert.NotNil(t, ds.Filters)
		assert.Equal(t, 1, len(ds.Filters))
		assert.Equal(t, "test-key", ds.Filters[0].Key.Name)
		assert.Equal(t, types.DataSourceSpecPropertyKeyType(1), ds.Filters[0].Key.Type)
		assert.NotNil(t, ds.Trigger)
		assert.IsType(t, &vegapb.EthCallTrigger_TimeTrigger{}, ds.Trigger.IntoEthCallTriggerProto().Trigger)
	})
}

func TestEthCallSpecIntoProto(t *testing.T) {
	// TODO: Not sure we need this, because it will be a copy of
	// TestDataSourceDefinitionIntoProto
}

func TestEthCallSpecToDataSourceDefinitionProto(t *testing.T) {
	// Same as above
}

func TestEthCallSpecString(t *testing.T) {
	timeNow := uint64(time.Now().UnixNano())
	ds := &vegapb.EthCallSpec{
		Address: "test-eth-address",
		Abi: &structpb.ListValue{
			Values: []*structpb.Value{
				{
					Kind: &structpb.Value_NumberValue{
						NumberValue: float64(5),
					},
				},
			},
		},
		Method: "test-method",
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

	str := ds.String()
	assert.Equal(
		t,
		fmt.Sprintf("address:\"test-eth-address\" abi:{values:{number_value:5}} method:\"test-method\" args:{} trigger:{time_trigger:{initial:%d}} filters:{key:{name:\"test-key\" type:TYPE_EMPTY}}", timeNow),
		str,
	)
}
