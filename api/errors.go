package api

import (
	protobufproto "github.com/golang/protobuf/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newErrorMap() map[error]int32 {
	em := make(map[error]int32)

	em[ErrChainNotConnected] = 1001

	// TBD

	return em
}

func lookupError(em map[error]int32, err error) int32 {
	errCode, found := em[err]
	if found {
		return errCode
	}
	return 0
}

func errorWithDetails(code codes.Code, message string, details ...protobufproto.Message) error {
	s := status.New(code, message)

	for _, detail := range details {
		// TODO: Handle err returned from WithDetails
		s, _ = s.WithDetails(detail)
	}
	return s.Err()
}
