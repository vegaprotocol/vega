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

package errors

import "strings"

type CumulatedErrors struct {
	Errors []error
}

func NewCumulatedErrors() *CumulatedErrors {
	return &CumulatedErrors{}
}

func (e *CumulatedErrors) Add(err error) {
	e.Errors = append(e.Errors, err)
}

func (e *CumulatedErrors) HasAny() bool {
	return len(e.Errors) > 0
}

func (e *CumulatedErrors) Error() string {
	fmtErrors := make([]string, 0, len(e.Errors))
	for _, err := range e.Errors {
		fmtErrors = append(fmtErrors, err.Error())
	}

	return strings.Join(fmtErrors, ", also ")
}
