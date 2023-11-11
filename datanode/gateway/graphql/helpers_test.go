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

package gql

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSafeStringUint64(t *testing.T) {
	convTests := []struct {
		in          string
		out         uint64
		expectError bool
	}{
		{"-1", 0, true},
		{"-9223372036854775808", 0, true},
		{"x';INSERT INTO users ('email','passwd') VALUES ('ned@fladers.org','hello');--", 0, true},
		{"0", 0, false},
		{"100", 100, false},
		{"9223372036854775807", 9223372036854775807, false},
		{"18446744073709551615", 18446744073709551615, false},
	}

	for _, tt := range convTests {
		c, err := safeStringUint64(tt.in)

		assert.Equal(t, tt.out, c)

		if tt.expectError {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}
	}
}

func TestSecondsTSToDatetime(t *testing.T) {
	aTime := "2020-05-30T00:00:00Z"
	testTime, err := time.Parse(time.RFC3339Nano, aTime)
	assert.NoError(t, err)

	stringified := secondsTSToDatetime(testTime.Unix())
	assert.EqualValues(t, aTime, stringified)

	badValue := secondsTSToDatetime(testTime.UnixNano())
	assert.NotEqual(t, aTime, badValue)
}

func TestNanoTSToDatetime(t *testing.T) {
	aTime := "2020-05-30T00:00:00Z"
	testTime, err := time.Parse(time.RFC3339Nano, aTime)
	assert.NoError(t, err)

	stringified := nanoTSToDatetime(testTime.UnixNano())
	assert.EqualValues(t, aTime, stringified)

	badValue := nanoTSToDatetime(testTime.Unix())
	assert.NotEqual(t, aTime, badValue)
}
