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

package commands_test

import (
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/commands"

	"github.com/stretchr/testify/assert"
)

func TestErrors(t *testing.T) {
	t.Run("Adding errors succeeds", testAddingErrorsSucceeds)
}

func testAddingErrorsSucceeds(t *testing.T) {
	errs := commands.NewErrors()
	prop := "user"
	err1 := errors.New("this is a first error")
	err2 := errors.New("this is a second error")

	errs.AddForProperty(prop, err1)
	errs.AddForProperty(prop, err2)

	assert.Equal(t, []error{err1, err2}, errs.Get(prop))
}
