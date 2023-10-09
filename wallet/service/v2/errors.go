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

package v2

import "errors"

var (
	ErrAdminEndpointsNotExposed                 = errors.New("administrative endpoints are not exposed, for security reasons")
	ErrAuthorizationHeaderIsRequired            = errors.New("the Authorization header is required")
	ErrAuthorizationHeaderOnlySupportsVWTScheme = errors.New("the Authorization header only support the VWT scheme")
	ErrAuthorizationTokenIsNotValidVWT          = errors.New("the Authorization value is not a valid VWT")
	ErrCouldNotReadRequestBody                  = errors.New("couldn't read the HTTP request body")
	ErrOriginHeaderIsRequired                   = errors.New("the Origin header is required")
	ErrRequestCannotBeBlank                     = errors.New("the request can't be blank")
)
