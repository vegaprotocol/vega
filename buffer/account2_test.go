package buffer_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/buffer"
	types "code.vegaprotocol.io/vega/proto"
	"github.com/stretchr/testify/assert"
)

type accBufTst struct {
	*buffer.AccountCh
	ctx   context.Context
	cfunc context.CancelFunc
}

func TestAccountCtx(t *testing.T) {
	buf := getAccBuf()
	buf.cfunc()
}

func TestAccBufSubUnsub(t *testing.T) {
	buf := getAccBuf()
	sub1 := buf.Subscribe(1)
	sub2 := buf.Subscribe(1)
	assert.NotNil(t, sub1.Recv())
	assert.NotNil(t, sub2.Recv())
	buf.Unsubscribe(sub1)
	// this should cancel the context of sub1, but have no impact on sub2
	assert.Error(t, sub1.Err())
	assert.NoError(t, sub2.Err())
	_, ok := <-sub1.Recv()
	assert.False(t, ok)
	// create another subscriber, this should reuse the first sub's key (covered code-path to verify)
	sub1 = buf.Subscribe(1)
	buf.cfunc()
	// ctx is cancelled, sub channels should be closed
	<-sub1.Done()
	<-sub2.Done()
	_, ok = <-sub1.Recv()
	assert.False(t, ok)
	_, ok = <-sub2.Recv()
	assert.False(t, ok)
}

func TestAccBufFlush(t *testing.T) {
	buf := getAccBuf()
	sub := buf.Subscribe(1)
	accId := "acc-1"
	account := types.Account{
		Id:       accId,
		Owner:    "trader-1",
		MarketID: "test-market",
	}
	buf.Add(account)
	buf.Flush()
	subAcc := <-sub.Recv()
	assert.Equal(t, 1, len(subAcc))
	// we expect the ID to be set to empty string by the buffer
	account.Id = ""
	assert.Equal(t, account, subAcc[accId])
	assert.NoError(t, sub.Err())
	buf.cfunc() // close down everything
	assert.NotNil(t, <-sub.Done())
	assert.Error(t, sub.Err())
}

func TestAccBufLastOnly(t *testing.T) {
	buf := getAccBuf()
	sub := buf.Subscribe(1)
	accId, accOwner, accMarket := "acc-1", "trader-1", "test-market"
	data := []types.Account{
		{
			Id:       accId,
			Owner:    accOwner,
			MarketID: accMarket,
		},
		{
			Id:       accId,
			Owner:    accOwner,
			MarketID: accMarket,
		},
		{
			Id:       accId,
			Owner:    accOwner,
			MarketID: accMarket,
		},
	}
	balances := []int64{1000, 2000, 1234}
	var lastAcc types.Account
	// keep pushing accounts onto buffer, only the last should remain
	for i := range data {
		lastAcc = data[i]
		lastAcc.Balance = balances[i]
		buf.Add(lastAcc)
		lastAcc.Id = ""
	}
	otherID := "acc-2"
	account := types.Account{
		Id:       otherID,
		Owner:    "trader-2",
		MarketID: accMarket,
	}
	buf.Add(account)
	buf.Flush()
	subAcc := <-sub.Recv()
	assert.Equal(t, 2, len(subAcc))
	// we expect the ID to be set to empty string by the buffer
	account.Id = ""
	assert.Equal(t, account, subAcc[otherID])
	assert.Equal(t, lastAcc, subAcc[accId])
	assert.NoError(t, sub.Err())
	buf.cfunc() // close down everything
	assert.NotNil(t, <-sub.Done())
	assert.Error(t, sub.Err())
}

func getAccBuf() *accBufTst {
	ctx, cfunc := context.WithCancel(context.Background())
	return &accBufTst{
		AccountCh: buffer.NewAccountCh(ctx),
		ctx:       ctx,
		cfunc:     cfunc,
	}
}
