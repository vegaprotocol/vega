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

	"code.vegaprotocol.io/data-node/datanode/sqlsubscribers"
	"code.vegaprotocol.io/data-node/datanode/sqlsubscribers/mocks"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/golang/mock/gomock"
)

func TestDeposit_Push(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mocks.NewMockDepositStore(ctrl)

	store.EXPECT().Upsert(context.Background(), gomock.Any()).Times(1)
	subscriber := sqlsubscribers.NewDeposit(store, logging.NewTestLogger())
	subscriber.Push(context.Background(), events.NewDepositEvent(context.Background(), types.Deposit{
		ID:           "DEADBEEF",
		Status:       types.DepositStatusOpen,
		PartyID:      "DEADBEEF",
		Asset:        "DEADBEEF",
		Amount:       num.NewUint(1000),
		TxHash:       "",
		CreditDate:   0,
		CreationDate: 0,
	}))
}
