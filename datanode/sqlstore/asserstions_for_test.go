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

package sqlstore_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// RequireAllDifferent requires that none of the objects are equal.
// This is useful to ensure the objects used in tests are actually different
// when expecting one or another.
// It's mainly made to ensure the tests are dealing with meaningful setup.
func RequireAllDifferent(t *testing.T, objs ...any) {
	t.Helper()

	for i := range objs {
		for j := i + 1; j < len(objs); j++ {
			require.NotEqual(t, objs[i], objs[j])
		}
	}
}
