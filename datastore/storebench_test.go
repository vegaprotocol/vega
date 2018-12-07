package datastore

import (
	"testing"
	"vega/msg"
	"fmt"
	"math/rand"
	"github.com/dgraph-io/badger"
)

//type OrderB struct {
//	Id string
//	Market string
//	Party string
//}
//
//func BenchmarkInsertToBadger2(b *testing.B) {
//
//	opts := badger.DefaultOptions
//	opts.Dir = "./data"
//	opts.ValueDir = "./data"
//	db, _ := badger.Open(opts)
//	defer db.Close()
//
//	var order OrderB
//	for i := 0; i < b.N; i++ {
//		order = OrderB{
//			Id:        fmt.Sprintf("%d", rand.Intn(1000000000000)),
//			Market:    testMarket,
//			Party:     testPartyA,
//		}
//		db.Update(func(txn *badger.Txn) error {
//			marketKey := fmt.Sprintf("M:%s_ID:%s", order.Market, order.Id)
//			txn.Set([]byte(marketKey), []byte(marketKey))
//			return	nil
//		})
//	}
//}
//
//
//func BenchmarkInsertToBadgerAsync2(b *testing.B) {
//
//	opts := badger.DefaultOptions
//	opts.Dir = "./data"
//	opts.ValueDir = "./data"
//	db, _ := badger.Open(opts)
//	defer db.Close()
//
//	var order OrderB
//	var wg sync.WaitGroup
//	for i := 0; i < b.N; i++ {
//		wg.Add(1)
//		go func() {
//			order = OrderB{
//				Id:        fmt.Sprintf("%d", rand.Intn(1000000000000)),
//				Market:    testMarket,
//				Party:     testPartyA,
//			}
//			db.Update(func(txn *badger.Txn) error {
//				marketKey := fmt.Sprintf("M:%s_ID:%s", order.Market, order.Id)
//				txn.Set([]byte(marketKey), []byte(marketKey))
//				return	nil
//			})
//
//			wg.Done()
//		}()
//	}
//	wg.Wait()
//}

//func prepopulateStore(store OrderStore) {
//	for i := 0; i < 10000; i++ {
//		_ = store.Post(
//			Order{
//			msg.Order{
//				Id:     fmt.Sprintf("%d", rand.Intn(1000000000000)),
//				Market: testMarket,
//				Party:  testPartyA,
//				Price: uint64(100),
//				Timestamp: uint64(3),
//				Size: uint64(123),
//			},
//		})
//	}
//}
//
//func BenchmarkInsertToMemStore(b *testing.B) {
//
//	var memStore = NewMemStore([]string{testMarket}, []string{testPartyA})
//	var newOrderStore = NewOrderStore(&memStore)
//
//	var order Order
//	for i := 0; i < b.N; i++ {
//		order = Order{
//			msg.Order{
//				Id:     fmt.Sprintf("%d", rand.Intn(1000000000000)),
//				Market: testMarket,
//				Party:  testPartyA,
//				Price: uint64(100),
//				Timestamp: uint64(3),
//				Size: uint64(123),
//			},
//		}
//		_ = newOrderStore.Post(order)
//	}
//}

//func BenchmarkInsertToBadger(b *testing.B) {
//
//	var memStore = NewMemStore([]string{testMarket}, []string{testPartyA})
//	var newOrderStore = NewOrderStoreP(&memStore, "./Data")
//	defer newOrderStore.Close()
//
//	//stateOrders, _ := newOrderStore.GetByMarket2(testMarket, nil)
//	//fmt.Printf("state %d\n", len(stateOrders))
//
//	var order Order
//	for i := 0; i < b.N; i++ {
//		order = Order{
//			msg.Order{
//				Id:     fmt.Sprintf("%d", rand.Intn(1000000000000)),
//				Market: testMarket,
//				Party:  testPartyA,
//				Price: uint64(100),
//				Timestamp: uint64(3),
//				Size: uint64(123),
//			},
//		}
//		_ = newOrderStore.PostP(order)
//	}
//
//	//orders, _ := newOrderStore.GetByMarket2(testMarket, nil)
//	//fmt.Printf("inserted %d\n", len(orders) - len(stateOrders))
//}

//func BenchmarkInsertToBadgerAsync(b *testing.B) {
//
//	var memStore = NewMemStore([]string{testMarket}, []string{testPartyA})
//	var newOrderStore = NewOrderStoreP(&memStore, "./DataAsync")
//	defer newOrderStore.Close()
//
//	//stateOrders, _ := newOrderStore.GetByMarket2(testMarket, nil)
//	//fmt.Printf("state %d\n", len(stateOrders))
//
//	var order Order
//	var wg sync.WaitGroup
//	for i := 0; i < b.N; i++ {
//		wg.Add(1)
//		go func() {
//			order = Order{
//				msg.Order{
//					Id:        fmt.Sprintf("%d", rand.Intn(1000000000000)),
//					Market:    testMarket,
//					Party:     testPartyA,
//					Price:     uint64(100),
//					Timestamp: uint64(3),
//					Size:      uint64(123),
//				},
//			}
//			_ = newOrderStore.PostP(order)
//			wg.Done()
//		}()
//	}
//	wg.Wait()
//
//	//orders, _ := newOrderStore.GetByMarket2(testMarket, nil)
//	//fmt.Printf("inserted %d\n", len(orders) - len(stateOrders))
//}


//func BenchmarkGetFromMemStore(b *testing.B) {
//
//	var memStore = NewMemStore([]string{testMarket}, []string{testPartyA})
//	var newOrderStore = NewOrderStore(&memStore)
//	prepopulateStore(newOrderStore)
//
//	// get last 50
//	last := uint64(50)
//	qfp := filters.QueryFilterPaginated{Last: &last}
//	for i := 0; i < b.N; i++ {
//		_, _ = newOrderStore.GetByMarket(testMarket, &filters.OrderQueryFilters{QueryFilterPaginated: qfp})
//	}
//}

//func BenchmarkGetFromBadger(b *testing.B) {
//
//	var memStore = NewMemStore([]string{testMarket}, []string{testPartyA})
//	var newOrderStore = NewOrderStoreP(&memStore, "./DataAsync")
//	defer newOrderStore.Close()
//
//	// get last 50
//	last := uint64(50)
//	qfp := filters.QueryFilterPaginated{Last: &last}
//	for i := 0; i < b.N; i++ {
//		_, _ = newOrderStore.GetByMarket2(testMarket, &filters.OrderQueryFilters{QueryFilterPaginated: qfp})
//	}
//}

func BenchmarkBatchInsertToBadger(b *testing.B) {

	db, err := badger.Open(customBadgerOptions("./Data"))
	if err != nil {
		b.Fatalf("database could not be initialised: %s", err)
	}
	bs := badgerStore{db: db}
	var newOrderStore = &badgerOrderStore{
		badger:         &bs,
		orderBookDepth: NewMarketDepthUpdaterGetter(),
		subscribers:    make(map[uint64]chan<- []msg.Order),
		buffer:         make([]msg.Order, 0),
	}
	
	defer newOrderStore.Close()

	//stateOrders, _ := newOrderStore.GetByMarket2(testMarket, nil)
	//fmt.Printf("state %d\n", len(stateOrders))

	var batchedOrders []msg.Order
	for i := 0; i < b.N; i++ {
		order := msg.Order{
			Id:     fmt.Sprintf("%d", rand.Intn(1000000000000)),
			Market: testMarket,
			Party:  testPartyA,
			Price: uint64(100),
			Timestamp: uint64(3),
			Size: uint64(123),
		}
		batchedOrders = append(batchedOrders, order)
	}

	_ = newOrderStore.writeBatch(batchedOrders)

	//orders, _ := newOrderStore.GetByMarket2(testMarket, nil)
	//fmt.Printf("inserted %d\n", len(orders) - len(stateOrders))
}