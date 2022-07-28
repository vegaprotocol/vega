// Copyright (c) 2022 Gobalsky Labs Limited
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

package types

import (
	"testing"

	"code.vegaprotocol.io/vega/core/types/num"
	"github.com/stretchr/testify/assert"
)

func TestReflectPointerToString(t *testing.T) {
	tcs := []struct {
		name     string
		stringer Stringer
		expected string
	}{
		{
			name:     "with nil interface",
			stringer: nil,
			expected: "nil",
		}, {
			name:     "with nil struct",
			stringer: Stringer(nil),
			expected: "nil",
		}, {
			name:     "with existing struct",
			stringer: dummyStringer{},
			expected: "stringer",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// when
			str := reflectPointerToString(tc.stringer)

			// then
			assert.Equal(tt, tc.expected, str)
		})
	}
}

func TestUintPointerToString(t *testing.T) {
	tcs := []struct {
		name     string
		num      *num.Uint
		expected string
	}{
		{
			name:     "with nil number",
			num:      nil,
			expected: "nil",
		}, {
			name:     "with existing number",
			num:      num.NewUint(42),
			expected: "42",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// when
			str := uintPointerToString(tc.num)

			// then
			assert.Equal(tt, tc.expected, str)
		})
	}
}

func TestInt64PointerToString(t *testing.T) {
	tcs := []struct {
		name     string
		num      *int64
		expected string
	}{
		{
			name:     "with nil number",
			num:      nil,
			expected: "nil",
		}, {
			name: "with existing number",
			num: func() *int64 {
				n := int64(42)
				return &n
			}(),
			expected: "42",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// when
			str := int64PointerToString(tc.num)

			// then
			assert.Equal(tt, tc.expected, str)
		})
	}
}

type dummyStringer struct{}

func (d dummyStringer) String() string {
	return "stringer"
}
