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

package stubs

import (
	"context"

	"code.vegaprotocol.io/vega/core/txn"

	"github.com/cenkalti/backoff"
	"github.com/golang/protobuf/proto"
)

type CommanderStub struct{}

func NewCommanderStub() *CommanderStub {
	return &CommanderStub{}
}

func (*CommanderStub) Command(ctx context.Context, cmd txn.Command, payload proto.Message, f func(string, error), bo *backoff.ExponentialBackOff) {
}

func (*CommanderStub) CommandSync(ctx context.Context, cmd txn.Command, payload proto.Message, f func(string, error), bo *backoff.ExponentialBackOff) {
}
