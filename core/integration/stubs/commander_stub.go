// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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

func (*CommanderStub) Command(ctx context.Context, cmd txn.Command, payload proto.Message, f func(error), bo *backoff.ExponentialBackOff) {
}

func (*CommanderStub) CommandSync(ctx context.Context, cmd txn.Command, payload proto.Message, f func(error), bo *backoff.ExponentialBackOff) {
}
