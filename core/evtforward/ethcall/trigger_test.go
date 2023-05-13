package ethcall_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/evtforward/ethcall"
	"github.com/stretchr/testify/assert"
)

func TestTimeTrigger(t *testing.T) {
	tt := ethcall.TimeTrigger{
		Initial: 10,
		Every:   5,
		Until:   20,
	}

	assert.False(t, tt.Trigger(testBlock{1}, testBlock{8}))
	assert.True(t, tt.Trigger(testBlock{8}, testBlock{10}))
	assert.False(t, tt.Trigger(testBlock{10}, testBlock{11}))
	assert.False(t, tt.Trigger(testBlock{11}, testBlock{14}))
	assert.True(t, tt.Trigger(testBlock{14}, testBlock{15}))
	assert.False(t, tt.Trigger(testBlock{15}, testBlock{30}))
}

func TestTimeTriggerNoInitial(t *testing.T) {
	tt := ethcall.TimeTrigger{
		Initial: 0,
		Every:   5,
		Until:   20,
	}

	assert.True(t, tt.Trigger(testBlock{1}, testBlock{8}))
	assert.True(t, tt.Trigger(testBlock{8}, testBlock{10}))
	assert.False(t, tt.Trigger(testBlock{10}, testBlock{11}))
	assert.False(t, tt.Trigger(testBlock{11}, testBlock{14}))
	assert.True(t, tt.Trigger(testBlock{14}, testBlock{15}))
	assert.False(t, tt.Trigger(testBlock{15}, testBlock{30}))
}

func TestTimeTriggerNoEvery(t *testing.T) {
	tt := ethcall.TimeTrigger{
		Initial: 10,
		Every:   0,
		Until:   20,
	}

	assert.False(t, tt.Trigger(testBlock{1}, testBlock{8}))
	assert.True(t, tt.Trigger(testBlock{8}, testBlock{10}))
	assert.False(t, tt.Trigger(testBlock{10}, testBlock{11}))
	assert.False(t, tt.Trigger(testBlock{11}, testBlock{14}))
	assert.False(t, tt.Trigger(testBlock{14}, testBlock{15}))
	assert.False(t, tt.Trigger(testBlock{15}, testBlock{30}))
}

func TestTimeTriggerNoUntil(t *testing.T) {
	tt := ethcall.TimeTrigger{
		Initial: 10,
		Every:   5,
		Until:   0,
	}

	assert.False(t, tt.Trigger(testBlock{1}, testBlock{8}))
	assert.True(t, tt.Trigger(testBlock{8}, testBlock{10}))
	assert.False(t, tt.Trigger(testBlock{10}, testBlock{11}))
	assert.False(t, tt.Trigger(testBlock{11}, testBlock{14}))
	assert.True(t, tt.Trigger(testBlock{14}, testBlock{15}))
	assert.True(t, tt.Trigger(testBlock{15}, testBlock{30}))
}

func TestTriggerToFromProto(t *testing.T) {
	tt := ethcall.TimeTrigger{
		Initial: 10,
		Every:   5,
		Until:   20,
	}

	proto := tt.ToProto()
	tt2, err := ethcall.TriggerFromProto(proto)
	assert.NoError(t, err)
	assert.Equal(t, tt, tt2)
}

type testBlock struct {
	n uint64
}

func (t testBlock) NumberU64() uint64 {
	return t.n
}

func (t testBlock) Time() uint64 {
	return t.n
}
