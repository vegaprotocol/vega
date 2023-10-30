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

/*
func TestTimeTrigger(t *testing.T) {
	tt := spec.TimeTrigger{
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
	tt := spec.TimeTrigger{
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
	tt := spec.TimeTrigger{
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
	tt := spec.TimeTrigger{
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
	tt := spec.TimeTrigger{
		Initial: 10,
		Every:   5,
		Until:   20,
	}

	proto := tt.ToProto()
	tt2, err := spec.TriggerFromProto(proto)
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
*/
