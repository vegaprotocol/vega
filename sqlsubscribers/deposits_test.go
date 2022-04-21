package sqlsubscribers_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/sqlsubscribers"
	"code.vegaprotocol.io/data-node/sqlsubscribers/mocks"
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
	subscriber.Push(context.Background(), events.NewTime(context.Background(), time.Now()))
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
