// Copyright (C) 2023  Gobalsky Labs Limited
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

package stringer_test

import (
	"testing"

	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/stringer"
	"github.com/stretchr/testify/assert"
)

func TestReflectPointerToString(t *testing.T) {
	tcs := []struct {
		name     string
		stringer stringer.Stringer
		expected string
	}{
		{
			name:     "with nil interface",
			stringer: nil,
			expected: "nil",
		}, {
			name:     "with nil struct",
			stringer: stringer.Stringer(nil),
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
			str := stringer.ReflectPointerToString(tc.stringer)

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
			str := stringer.UintPointerToString(tc.num)

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
			str := stringer.Int64PointerToString(tc.num)

			// then
			assert.Equal(tt, tc.expected, str)
		})
	}
}

type dummyStringer struct{}

func (d dummyStringer) String() string {
	return "stringer"
}
