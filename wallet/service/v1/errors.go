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
