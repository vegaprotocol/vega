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

package errors_test

import (
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/libs/errors"

	"github.com/stretchr/testify/require"
)

func TestCircularreferences(t *testing.T) {
	parent := errors.NewCumulatedErrors()
	child := errors.NewCumulatedErrors()
	nested := errors.NewCumulatedErrors()
	errs := []error{
		fmt.Errorf("simple error 1"),
		fmt.Errorf("simple error 2"),
		fmt.Errorf("simple error 3"),
	}
	t.Run("try to add parent to itself", func(t *testing.T) {
		parent.Add(parent)
		expect := "<self reference>"
		require.True(t, parent.HasAny())
		require.Equal(t, expect, parent.Error())
	})
	t.Run("try nesting without circular references", func(t *testing.T) {
		child.Add(errs[0])
		parent.Add(child)
		expect := fmt.Sprintf("<self reference>, also %s", errs[0].Error())
		require.Equal(t, expect, parent.Error())
	})
	t.Run("try adding empty cumulated error", func(t *testing.T) {
		parent.Add(nested)
		// still the same expected value
		expect := fmt.Sprintf("<self reference>, also %s", errs[0].Error())
		require.False(t, nested.HasAny())
		require.Equal(t, expect, parent.Error())
		// adding to the nested error should not affect the parent.
		nested.Add(errs[1])
		require.True(t, nested.HasAny())
		require.Equal(t, expect, parent.Error())
	})
	t.Run("try nesting both parent and child, and adding to both", func(t *testing.T) {
		nested.Add(errs[2])
		nested.Add(child)
		nested.Add(parent)
		child.Add(nested)
		parent.Add(nested)
		// self reference>, also simple error 1, also simple error 2, also simple error 3, also simple error 1, also <self reference>, also simple error 1
		expect := fmt.Sprintf("<self reference>, also %s, also %s, also %s, also %s, also <self reference>, also %s", errs[0].Error(), errs[1].Error(), errs[2].Error(), errs[0].Error(), errs[0].Error())
		require.Equal(t, expect, parent.Error())
	})
}
