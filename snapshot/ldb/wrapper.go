package ldb

import (
	"errors"
	"fmt"
	"time"

	"code.vegaprotocol.io/shared/paths"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
	db "github.com/tendermint/tm-db"
)

var (
	ErrNext = errors.New("next call on iterator failed")
)

type Wrapper struct {
	*leveldb.DB
}

type WIterator struct {
	iterator.Iterator
	start, end []byte
	nerr       error
}

type WBatch struct {
	db    *leveldb.DB
	tx    *leveldb.Transaction
	batch *leveldb.Batch
}

func OpenDB(vp paths.Paths, o *opt.Options) (*Wrapper, error) {
	dbPath, err := vp.DataPathFor(paths.SnapshotDBStateFile)
	if err != nil {
		return nil, err
	}
	conn, err := leveldb.OpenFile(dbPath, o)
	if err != nil {
		return nil, nil
	}
	return &Wrapper{
		DB: conn,
	}, nil
}

// Delete forwards the delete call, but without options
func (w *Wrapper) Delete(key []byte) error {
	return w.DB.Delete(key, nil)
}

// DeleteSync synchronous deletion of a key
func (w *Wrapper) DeleteSync(key []byte) error {
	return w.DB.Delete(key, &opt.WriteOptions{
		Sync: true,
	})
}

// Get forwards call with default options - used to implement desired interface
func (w *Wrapper) Get(key []byte) ([]byte, error) {
	return w.DB.Get(key, nil)
}

// Has returns bool if a given key exists
func (w *Wrapper) Has(key []byte) (bool, error) {
	return w.DB.Has(key, nil)
}

func (w *Wrapper) Iterator(start, end []byte) (db.Iterator, error) {
	sl := &util.Range{
		Start: start,
		Limit: end,
	}
	it := w.DB.NewIterator(sl, nil)
	return &WIterator{
		Iterator: it,
		start:    start,
		end:      end,
	}, nil
}

func (w *Wrapper) ReverseIterator(start, end []byte) (db.Iterator, error) {
	// swap arg order?
	sl := &util.Range{
		Start: end,
		Limit: start,
	}
	it := w.DB.NewIterator(sl, &opt.ReadOptions{})
	return &WIterator{
		Iterator: it,
		start:    start,
		end:      end,
	}, nil
}

func (w *Wrapper) Set(k, v []byte) error {
	return w.DB.Put(k, v, nil)
}

func (w *Wrapper) SetSync(k, v []byte) error {
	return w.DB.Put(k, v, &opt.WriteOptions{
		Sync: true,
	})
}

func (w *Wrapper) Stats() map[string]string {
	s := &leveldb.DBStats{}
	if err := w.DB.Stats(s); err != nil {
		return nil
	}
	lvlSizes := sizeToInt(s.LevelSizes)
	lvlRead := sizeToInt(s.LevelRead)
	lvlWrite := sizeToInt(s.LevelWrite)
	lvlDur := durationToStr(s.LevelDurations)
	return map[string]string{
		"WriteDelayCount":    fmt.Sprintf("%d", s.WriteDelayCount),
		"WriteDelayDuration": s.WriteDelayDuration.String(),
		"WritePaused":        fmt.Sprintf("%v", s.WritePaused),
		"AliveSnapshots":     fmt.Sprintf("%d", s.AliveSnapshots),
		"AliveIterators":     fmt.Sprintf("%d", s.AliveIterators),
		"IOWrite":            fmt.Sprintf("%d", s.IOWrite),
		"IORead":             fmt.Sprintf("%d", s.IORead),
		"BlockCacheSize":     fmt.Sprintf("%d", s.BlockCacheSize),
		"OpenedTablesCount":  fmt.Sprintf("%d", s.OpenedTablesCount),
		"LevelSizes":         fmt.Sprintf("%v", lvlSizes),
		"LevelTablesCounts":  fmt.Sprintf("%v", s.LevelTablesCounts),
		"LevelRead":          fmt.Sprintf("%v", lvlRead),
		"LevelWrite":         fmt.Sprintf("%v", lvlWrite),
		"LevelDurations":     fmt.Sprintf("%v", lvlDur),
		"MemComp":            fmt.Sprintf("%d", s.MemComp),
		"Level0Comp":         fmt.Sprintf("%d", s.Level0Comp),
		"NonLevel0Comp":      fmt.Sprintf("%d", s.NonLevel0Comp),
		"SeekComp":           fmt.Sprintf("%d", s.SeekComp),
	}
}

func (w *Wrapper) NewBatch() db.Batch {
	return &WBatch{
		db:    w.DB,
		batch: &leveldb.Batch{},
	}
}

func (w *Wrapper) Print() error {
	s := leveldb.DBStats{}
	if err := w.DB.Stats(&s); err != nil {
		return err
	}
	// just print the stats object?
	fmt.Printf("%#v\n", s)
	return nil
}

func (w *WIterator) Close() error {
	w.Iterator.Release()
	return nil
}

func (w WIterator) Domain() ([]byte, []byte) {
	return w.start, w.end
}

func (w *WIterator) Next() {
	if !w.Iterator.Next() {
		w.nerr = ErrNext
	}
}

func (w *WIterator) Error() error {
	if err := w.Error(); err != nil {
		w.nerr = nil
		return err
	}
	if err := w.nerr; err != nil {
		w.nerr = nil
		return err
	}
	return nil
}

func (b *WBatch) Close() error {
	return b.tx.Commit()
}

func (b *WBatch) Delete(key []byte) error {
	b.batch.Delete(key)
	return nil
}

func (b *WBatch) Set(k, v []byte) error {
	b.batch.Put(k, v)
	return nil
}

func (b *WBatch) Write() error {
	tx, err := b.db.OpenTransaction()
	if err != nil {
		return err
	}
	b.tx = tx
	return tx.Write(b.batch, nil)
}

func (b *WBatch) WriteSync() error {
	tx, err := b.db.OpenTransaction()
	if err != nil {
		return err
	}
	b.tx = tx
	return tx.Write(b.batch, &opt.WriteOptions{
		Sync: true,
	})
}

func sizeToInt(sizes leveldb.Sizes) []int64 {
	ret := make([]int64, 0, len(sizes))
	for _, s := range sizes {
		ret = append(ret, s)
	}
	return ret
}

func durationToStr(dur []time.Duration) []string {
	ret := make([]string, 0, len(dur))
	for _, d := range dur {
		ret = append(ret, d.String())
	}
	return ret
}
