package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

var (
	ErrIsRequired           = errors.New("is required")
	ErrMustBePositive       = errors.New("must be positive")
	ErrMustBePositiveOrZero = errors.New("must be positive or zero")
	ErrMustBeNegative       = errors.New("must be negative")
	ErrMustBeNegativeOrZero = errors.New("must be negative or zero")
	ErrIsNotValid           = errors.New("is not a valid value")
	ErrIsNotSupported       = errors.New("is not supported")
	ErrIsUnauthorised       = errors.New("is unauthorised")
)

type Errors map[string]error

func NewErrors() Errors {
	return Errors{}
}

func (e Errors) Error() string {
	if len(e) <= 0 {
		return ""
	}

	messages := []string{}
	for prop, err := range e {
		messages = append(messages, fmt.Sprintf("%v(%v)", prop, err.Error()))
	}
	sort.Strings(messages)
	return strings.Join(messages, ", ")
}

func (e Errors) Empty() bool {
	return len(e) == 0
}

// AddForProperty adds an error for a given property.
func (e Errors) AddForProperty(prop string, err error) {
	e[prop] = err
}

// Add adds a general error that is not related to a specific property.
func (e Errors) Add(err error) {
	e.AddForProperty("*", err)
}

// FinalAdd behaves like Add, but is meant to be called in a "return" statement.
// This helper is usually used for terminal errors.
func (e Errors) FinalAdd(err error) Errors {
	e.Add(err)
	return e
}

func (e Errors) Merge(oth Errors) {
	for prop, err := range oth {
		e.AddForProperty(prop, err)
	}
}

func (e Errors) Get(prop string) error {
	msg, ok := e[prop]
	if !ok {
		return nil
	}
	return msg
}

func (e Errors) ErrorOrNil() error {
	if len(e) <= 0 {
		return nil
	}
	return e
}

func (e Errors) MarshalJSON() ([]byte, error) {
	out := map[string]string{}
	for k, v := range e {
		out[k] = v.Error()
	}
	return json.Marshal(out)
}
