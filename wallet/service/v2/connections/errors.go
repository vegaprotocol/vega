// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
