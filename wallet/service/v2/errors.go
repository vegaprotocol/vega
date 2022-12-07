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
