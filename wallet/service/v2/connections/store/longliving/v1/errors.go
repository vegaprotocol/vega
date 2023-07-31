package v1

import "errors"

var (
	ErrTokenDoesNotExist = errors.New("the token does not exist")
	ErrWrongPassphrase   = errors.New("wrong passphrase")
)
