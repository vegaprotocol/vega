package node

import (
	"fmt"
	"strings"

	typespb "code.vegaprotocol.io/vega/protos/vega"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ErrorCode codes.Code

type StatusError struct {
	Code    codes.Code
	Details []string
}

func (e *StatusError) Error() string {
	if len(e.Details) == 0 {
		return e.Code.String()
	}
	return fmt.Sprintf("%s: %v", e.Code.String(), strings.Join(e.Details, ", "))
}

// intoStatusError extracts useful information from a gRPC status error.
// It returns nil if the underlying error is not a gRPC status error.
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
