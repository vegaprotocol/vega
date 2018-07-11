package datastore

// GetParamsLimitDefault should be used if no limit is specified
// when working with the GetParams struct.
const GetParamsLimitDefault = uint64(1844674407370955161)

// GetParams is used for optional parameters that can be passed
// into the datastores when querying for records.
type GetParams struct {
	Limit uint64
}

// NotFoundError indicates that a record could not be located.
// This differentiates between not finding a record and the
// storage layer having an error.
type NotFoundError struct {
	error
}

func (n *NotFoundError) isNotFound() {}

// NotFound indicates if the error is that the record could
// not be found.
func NotFound(e error) bool {
	if _, ok := e.(NotFoundError); ok {
		return true
	}
	return false
}


