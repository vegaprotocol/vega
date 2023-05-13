package connections

import (
	"errors"
)

var (
	ErrExpirationDurationMustBeGreaterThan0          = errors.New("the expiration duration must be greater than 0")
	ErrHostnamesMismatchForThisToken                 = errors.New("the hostname from the request does not match the one that initiated the connection")
	ErrInvalidTokenFormat                            = errors.New("the token has not a valid format")
	ErrNoConnectionAssociatedThisAuthenticationToken = errors.New("there is no connection associated to this authentication token")
	ErrTokenDoesNotExist                             = errors.New("the token does not exist")
	ErrTokenHasExpired                               = errors.New("the token has expired")
	ErrTokenIsRequired                               = errors.New("the token is required")
	ErrWalletNameIsRequired                          = errors.New("the wallet name is required")
	ErrWalletPassphraseIsRequired                    = errors.New("the wallet passphrase is required")
)
