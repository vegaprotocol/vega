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

package sqlsubscribers_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/datanode/sqlsubscribers"
	"code.vegaprotocol.io/vega/datanode/sqlsubscribers/mocks"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/golang/mock/gomock"
)

func TestDeposit_Push(t *testing.T) {
	ctrl := gomock.NewController(t)

	store := mocks.NewMockDepositStore(ctrl)

	store.EXPECT().Upsert(context.Background(), gomock.Any()).Times(1)
	subscriber := sqlsubscribers.NewDeposit(store)
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
