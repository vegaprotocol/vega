// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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
