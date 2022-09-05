package network

import "fmt"

type AlreadyExistsError struct {
	Name string
}

func NewAlreadyExistsError(n string) AlreadyExistsError {
	return AlreadyExistsError{
		Name: n,
	}
}

func (e AlreadyExistsError) Error() string {
	return fmt.Sprintf("network \"%s\" already exists", e.Name)
}

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
