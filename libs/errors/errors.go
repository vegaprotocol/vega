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

// This is copy pasted from the go 1.20 errors package and can be deleted once the CI is updated.
func Join(errs ...error) error {
	n := 0
	for _, err := range errs {
		if err != nil {
			n++
		}
	}
	if n == 0 {
		return nil
	}
	e := &joinError{
		errs: make([]error, 0, n),
	}
	for _, err := range errs {
		if err != nil {
			e.errs = append(e.errs, err)
		}
	}
	return e
}

type joinError struct {
	errs []error
}

func (e *joinError) Error() string {
	var b []byte
	for i, err := range e.errs {
		if i > 0 {
			b = append(b, '\n')
		}
		b = append(b, err.Error()...)
	}
	return string(b)
}

func (e *joinError) Unwrap() []error {
	return e.errs
}
