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

package flags

import (
	"errors"
)

const (
	InteractiveOutput = "interactive"
	JSONOutput        = "json"
)

var (
	ErrUnsupportedOutput = errors.New("unsupported output")

	AvailableOutputs = []string{
		InteractiveOutput,
		JSONOutput,
	}
)

func ValidateOutput(output string) error {
	if len(output) == 0 {
		return MustBeSpecifiedError("output")
	}

	for _, o := range AvailableOutputs {
		if output == o {
			return nil
		}
	}

	// The output flag has special treatment because error reporting depends on
	// it, and we need to differentiate output errors from the rest to select
	// the right way to print the data.
	// As a result, we return a specific error, instead of a generic
	// UnsupportedFlagValueError.
	return ErrUnsupportedOutput
}
