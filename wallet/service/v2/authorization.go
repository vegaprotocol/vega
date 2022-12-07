package v2

import (
	"net/http"
	"strings"

	"code.vegaprotocol.io/vega/wallet/service/v2/connections"
)

// VWTPrefix is the scheme that prefixes the token in the Authorization HTTP header
// It is our non-standard scheme that stands for Vega Wallet Token.
const VWTPrefix = "VWT"

// VWT stands for Vega Wallet Token. It has the following format:
//
//	VWT <TOKEN>
//
// Example:
//
//	VWT QK6QoNLA2XEZdLFLxkFlq2oTX8cp8Xw1GOzxDAM0aSXxQAR33CGkvDh4vh2ZyQSh
type VWT struct {
	token connections.Token
}

func (t VWT) Token() connections.Token {
	return t.token
}

func (t VWT) String() string {
	return VWTPrefix + " " + t.Token().String()
}

func AsVWT(token connections.Token) VWT {
	return VWT{
		token: token,
	}
}

// ParseVWT parses a VWT into a VWT. If malformed, an error is returned.
func ParseVWT(rawVWT string) (VWT, error) {
	if !strings.HasPrefix(rawVWT, VWTPrefix+" ") {
		return VWT{}, ErrAuthorizationHeaderOnlySupportsVWTScheme
	}

	if len(rawVWT) < 5 {
		return VWT{}, ErrAuthorizationTokenIsNotValidVWT
	}

	rawToken := trimBlankCharacters(rawVWT[4:])

	if rawToken == "" {
		return VWT{}, ErrAuthorizationTokenIsNotValidVWT
	}

	token, err := connections.AsToken(rawToken)
	if err != nil {
		return VWT{}, err
	}

	return VWT{
		token: token,
	}, nil
}

// ExtractVWT extracts the Vega Wallet Token from the `Authorization` header.
func ExtractVWT(r *http.Request) (VWT, error) {
	rawToken := r.Header.Get("Authorization")
	if rawToken == "" {
		return VWT{}, ErrAuthorizationHeaderIsRequired
	}

	return ParseVWT(rawToken)
}
