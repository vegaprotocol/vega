package commands

import (
	"encoding/json"
	"fmt"
)

type Errors []error

func (e Errors) Error() string {
	if len(e) <= 0 {
		return ""
	}

	out := e[0].Error()
	for _, err := range e[1:] {
		out = fmt.Sprintf("%v, %v", out, err.Error())
	}
	return out
}

func (e Errors) ErrorOrNil() error {
	if len(e) <= 0 {
		return nil
	}
	return e
}

func (e Errors) MarshalJSON() ([]byte, error) {
	out := []string{}
	for _, v := range e {
		out = append(out, v.Error())
	}
	return json.Marshal(out)
}
