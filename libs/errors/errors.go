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
