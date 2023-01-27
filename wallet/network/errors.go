package network

import "fmt"

type DoesNotExistError struct {
	Name string
}

func NewDoesNotExistError(n string) DoesNotExistError {
	return DoesNotExistError{
		Name: n,
	}
}

func (e DoesNotExistError) Error() string {
	return fmt.Sprintf("network \"%s\" doesn't exist", e.Name)
}
