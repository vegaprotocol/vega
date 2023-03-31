package v1

import (
	"errors"

	"code.vegaprotocol.io/vega/commands"
)

var (
	ErrInvalidToken              = errors.New("invalid token")
	ErrInvalidClaims             = errors.New("invalid claims")
	ErrInvalidOrMissingToken     = newErrorResponse("invalid or missing token")
	ErrCouldNotReadRequest       = errors.New("couldn't read request")
	ErrCouldNotGetBlockHeight    = errors.New("couldn't get last block height")
	ErrCouldNotGetChainID        = errors.New("couldn't get chain-id")
	ErrShouldBeBase64Encoded     = errors.New("should be base64 encoded")
	ErrRejectedSignRequest       = errors.New("user rejected sign request")
	ErrInterruptedConsentRequest = errors.New("process to request consent has been interrupted")
)

type ErrorsResponse struct {
	Errors commands.Errors `json:"errors"`
}

type ErrorResponse struct { //nolint:errname
	ErrorStr string   `json:"error"`
	Details  []string `json:"details,omitempty"`
}

func (e ErrorResponse) Error() string {
	return e.ErrorStr
}

func newErrorResponse(e string) ErrorResponse {
	return ErrorResponse{
		ErrorStr: e,
	}
}
