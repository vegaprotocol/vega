package stubs

import (
	"context"

	"code.vegaprotocol.io/vega/txn"
	"github.com/golang/protobuf/proto"
)

type CommanderStub struct{}

func NewCommanderStub() *CommanderStub {
	return &CommanderStub{}
}

func (*CommanderStub) Command(ctx context.Context, cmd txn.Command, payload proto.Message, f func(error)) {
}
