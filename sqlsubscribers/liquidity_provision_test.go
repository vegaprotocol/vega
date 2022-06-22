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

package sqlsubscribers_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/sqlsubscribers"
	"code.vegaprotocol.io/data-node/sqlsubscribers/mocks"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"
	"github.com/golang/mock/gomock"
)

func TestLiquidityProvision_Push(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mocks.NewMockLiquidityProvisionStore(ctrl)

	store.EXPECT().Upsert(context.Background(), gomock.Any()).Times(1)
	store.EXPECT().Flush(gomock.Any()).Times(2)

	subscriber := sqlsubscribers.NewLiquidityProvision(store, logging.NewTestLogger())
	subscriber.Flush(context.Background())
	subscriber.Push(context.Background(), events.NewLiquidityProvisionEvent(context.Background(), &types.LiquidityProvision{}))
	subscriber.Flush(context.Background())
}

func TestLiquidityProvisionDuplicate_Push(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mocks.NewMockLiquidityProvisionStore(ctrl)

	store.EXPECT().Upsert(context.Background(), gomock.Any()).Times(2)
	store.EXPECT().Flush(gomock.Any()).Times(4)

	subscriber := sqlsubscribers.NewLiquidityProvision(store, logging.NewTestLogger())
	subscriber.Flush(context.Background())
	subscriber.Push(context.Background(), events.NewLiquidityProvisionEvent(context.Background(), &types.LiquidityProvision{}))
	subscriber.Flush(context.Background())
	subscriber.Push(context.Background(), events.NewLiquidityProvisionEvent(context.Background(), &types.LiquidityProvision{}))
	subscriber.Flush(context.Background())

	// Now push a non duplicate

	subscriber.Push(context.Background(), events.NewLiquidityProvisionEvent(context.Background(), &types.LiquidityProvision{Version: 1}))
	subscriber.Flush(context.Background())

}
