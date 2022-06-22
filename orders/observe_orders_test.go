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

package orders_test

import (
	"context"
	"sync"
	"testing"

	proto "code.vegaprotocol.io/protos/vega"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestObserveOrders(t *testing.T) {
	t.Run("Observe orders - all markets/parties success", testObserveAllOrdersSuccess)
	t.Run("Observe orders - some markets/parties success", testObservePartialSuccess)
}

func testObserveAllOrdersSuccess(t *testing.T) {
	svc := getTestService(t)
	defer svc.ctrl.Finish()
	ctx, cfunc := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	// channel used to indicate to subscriber routine that test is ready to read values from channel
	ready := make(chan struct{})
	done := make(chan struct{})
	subRef := uint64(1)
	orders := []proto.Order{
		{
			Id:       "order_id1",
			MarketId: "market1",
			PartyId:  "party1",
		},
		{
			Id:       "order_id2",
			MarketId: "market2",
			PartyId:  "party2",
		},
	}

	wg.Add(1)
	subscriber := func(ch chan<- []proto.Order) {
		<-ready
		defer wg.Done()
		ch <- orders
	}
	svc.orderStore.EXPECT().Subscribe(gomock.Any()).Times(1).Return(subRef).Do(func(ch chan<- []proto.Order) {
		go subscriber(ch)
	})
	svc.orderStore.EXPECT().Unsubscribe(subRef).Times(1).Return(nil).Do(func(_ uint64) {
		done <- struct{}{}
	})
	// all orders
	ch, ref := svc.svc.ObserveOrders(ctx, 0, nil, nil)
	close(ready)
	gotOrders := <-ch
	assert.Equal(t, subRef, ref)

	wg.Wait()
	cfunc() // cancel context
	<-done
	assert.Equal(t, len(orders), len(gotOrders))
	for i := range orders {
		assert.Equal(t, orders[i], gotOrders[i])
	}
}

func testObservePartialSuccess(t *testing.T) {
	svc := getTestService(t)
	defer svc.ctrl.Finish()
	ctx, cfunc := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	ready := make(chan struct{})
	done := make(chan struct{})
	subRef := uint64(1)
	market, party := "market1", "party1"
	orders := []proto.Order{
		{
			Id:       "order_id1",
			MarketId: "market1",
			PartyId:  "party1",
		},
		{
			Id:       "order_id2",
			MarketId: "market2",
			PartyId:  "party2",
		},
	}

	wg.Add(1)
	subscriber := func(ch chan<- []proto.Order) {
		<-ready
		defer wg.Done()
		ch <- orders
	}
	svc.orderStore.EXPECT().Subscribe(gomock.Any()).Times(1).Return(subRef).Do(func(ch chan<- []proto.Order) {
		go subscriber(ch)
	})
	svc.orderStore.EXPECT().Unsubscribe(subRef).Times(1).Return(nil).Do(func(_ uint64) {
		done <- struct{}{}
	})
	// all orders
	ch, ref := svc.svc.ObserveOrders(ctx, 0, &market, &party)
	close(ready)
	gotOrders := <-ch
	assert.Equal(t, subRef, ref)

	wg.Wait()
	cfunc() // cancel context
	<-done
	assert.Equal(t, 1, len(gotOrders))
	assert.Equal(t, orders[0], gotOrders[0])
}
