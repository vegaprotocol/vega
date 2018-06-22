package datastore

// NotFoundError indicates that a record could not be located.
// This differentiates between not finding a record and the
// storage layer having an error.
type NotFoundError struct {
	error
}

func (n NotFoundError) isNotFound() {}

// NotFound indicates if the error is that the ID could
// not be found.
func NotFound(e error) bool {
	if _, ok := e.(NotFoundError); ok {
		return true
	}
	return false
}
