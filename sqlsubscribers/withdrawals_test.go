package sqlsubscribers_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/sqlsubscribers"
	"code.vegaprotocol.io/data-node/sqlsubscribers/mocks"
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/golang/mock/gomock"
)

func TestWithdrawal_Push(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mocks.NewMockWithdrawalStore(ctrl)

	store.EXPECT().Upsert(context.Background(), gomock.Any()).Times(1)
	subscriber := sqlsubscribers.NewWithdrawal(store, logging.NewTestLogger())
	subscriber.Push(context.Background(), events.NewTime(context.Background(), time.Now()))
	subscriber.Push(context.Background(), events.NewWithdrawalEvent(context.Background(), types.Withdrawal{
		ID:             "DEADBEEF",
		PartyID:        "DEADBEEF",
		Amount:         num.NewUint(1000),
		Asset:          "DEADBEEF",
		Status:         types.WithdrawalStatusOpen,
		Ref:            "",
		TxHash:         "",
		CreationDate:   0,
		WithdrawalDate: 0,
		ExpirationDate: 0,
		Ext:            &vega.WithdrawExt{},
	}))
}
