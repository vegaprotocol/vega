package datastore
//
//import (
//	"testing"
//	"github.com/dgraph-io/badger"
//	"fmt"
//	"math/rand"
//	"sync"
//)
//
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
//			Market:    "testMarket",
//			Party:     "testPartyA",
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
//				Market:    "testMarket",
//				Party:     "testPartyA",
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
