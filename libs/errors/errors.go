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

import (
	"fmt"
	"strings"
)

var (
	errSelfReference   = fmt.Errorf("<self reference>")
	errParentReference = fmt.Errorf("<parent reference>")
)

type CumulatedErrors struct {
	Errors []error
}

func NewCumulatedErrors() *CumulatedErrors {
	return &CumulatedErrors{}
}

func (e *CumulatedErrors) Add(err error) {
	// prevent adding this instance of cumulatedErrors to itself.
	if err == e {
		err = errSelfReference
	} else if cerr, ok := err.(*CumulatedErrors); ok {
		// nothing to add.
		if !cerr.HasAny() {
			return
		}
		// create a copy of the error we're adding
		cpy := &CumulatedErrors{
			Errors: append([]error{}, cerr.Errors...),
		}
		// remove any references to the parent from the error we're adding.
		err = cpy.checkRef(e, errParentReference)
	}
	e.Errors = append(e.Errors, err)
}

// check recursively if a cumulated errors object contains a certain reference, and if so, replace with a placehold, simple error.
// returns either itself (with the replaced references), or a replacement error.
func (e *CumulatedErrors) checkRef(ref *CumulatedErrors, repl error) error {
	if e == ref {
		return repl
	}
	// recursively remove a given reference.
	for i, subE := range e.Errors {
		if subE == ref {
			e.Errors[i] = repl
		} else if cErr, ok := subE.(*CumulatedErrors); ok {
			e.Errors[i] = cErr.checkRef(ref, repl)
		}
	}
	return e
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
