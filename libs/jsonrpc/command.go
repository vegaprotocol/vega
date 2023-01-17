package jsonrpc

import "context"

//go:generate go run github.com/golang/mock/mockgen -destination mocks/command_mock.go -package mocks code.vegaprotocol.io/vega/libs/jsonrpc Command
type Command interface {
	Handle(ctx context.Context, params Params) (Result, *ErrorDetails)
}
