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

package node

import (
	"errors"
	"fmt"
	"strings"

	typespb "code.vegaprotocol.io/vega/protos/vega"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var ErrNoHostSpecified = errors.New("no host specified")

type ErrorCode codes.Code

type StatusError struct {
	Code    codes.Code
	Details []string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("%s - %v", e.Code.String(), strings.Join(e.Details, ", "))
}

// intoStatusError extract useful information from a gRPC status error.
// Returns nil if the underlying error is not a gRPC status error.
func intoStatusError(err error) *StatusError {
	st, ok := status.FromError(err)
	if !ok {
		return nil
	}
	statusErr := &StatusError{
		Code:    st.Code(),
		Details: []string{},
	}
	for _, v := range st.Details() {
		v, ok := v.(*typespb.ErrorDetail)
		if !ok {
			continue
		}
		statusErr.Details = append(statusErr.Details, v.GetMessage())
	}
	return statusErr
}
