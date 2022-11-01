package store

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

var ErrIndexEntryNotFound = errors.New("index entry not found")

type LevelDbBackedIndex struct {
	db *leveldb.DB
}

func NewIndex(dataDir string) (*LevelDbBackedIndex, error) {
	db, err := leveldb.OpenFile(filepath.Join(dataDir, "index.db"), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open level db file:%w", err)
	}

	return &LevelDbBackedIndex{
		db: db,
	}, nil
}

func (l LevelDbBackedIndex) Get(height int64) (SegmentIndexEntry, error) {
	value, err := l.db.Get(heightToKey(height), &opt.ReadOptions{})
	if errors.Is(err, leveldb.ErrNotFound) {
		return SegmentIndexEntry{}, ErrIndexEntryNotFound
	}

	if err != nil {
		return SegmentIndexEntry{}, fmt.Errorf("failed to get index entry:%w", err)
	}

	var indexEntry SegmentIndexEntry
	err = json.Unmarshal(value, &indexEntry)

	if err != nil {
		return SegmentIndexEntry{}, fmt.Errorf("failed to unmarshal value:%w", err)
	}

	return indexEntry, nil
}

func heightToKey(height int64) []byte {
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, uint64(height))
	return bytes
}

func (l LevelDbBackedIndex) Add(indexEntry SegmentIndexEntry) error {
	bytes, err := json.Marshal(indexEntry)
	if err != nil {
		return fmt.Errorf("failed to marshal index entry:%w", err)
	}

	err = l.db.Put(heightToKey(indexEntry.HeightTo), bytes, &opt.WriteOptions{})
	if err != nil {
		return fmt.Errorf("failed to put index entry:%w", err)
	}

	return nil
}

func (l LevelDbBackedIndex) Remove(indexEntry SegmentIndexEntry) error {
	if err := l.db.Delete(heightToKey(indexEntry.HeightTo), &opt.WriteOptions{}); err != nil {
		return fmt.Errorf("failed to delete key:%w", err)
	}

	return nil
}

func (l LevelDbBackedIndex) ListAllEntriesOldestFirst() ([]SegmentIndexEntry, error) {
	iter := l.db.NewIterator(&util.Range{
		Start: nil,
		Limit: nil,
	}, &opt.ReadOptions{})
	defer iter.Release()

	var segments []SegmentIndexEntry
	if !iter.Last() {
		return segments, nil
	}

	for ok := iter.Last(); ok; ok = iter.Prev() {
		bytes := iter.Value()
		var indexEntry SegmentIndexEntry
		err := json.Unmarshal(bytes, &indexEntry)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal index entry:%w", err)
		}

		segments = append(segments, indexEntry)
	}

	sort.Slice(segments, func(i, j int) bool {
		return segments[i].HeightFrom < segments[j].HeightFrom
	})

	return segments, nil
}

func (l LevelDbBackedIndex) GetHighestBlockHeightEntry() (SegmentIndexEntry, error) {
	entries, err := l.ListAllEntriesOldestFirst()
	if err != nil {
		return SegmentIndexEntry{}, fmt.Errorf("failed to list all entries:%w", err)
	}

	if len(entries) == 0 {
		return SegmentIndexEntry{}, ErrIndexEntryNotFound
	}

	return entries[len(entries)-1], nil
}

func (l LevelDbBackedIndex) Close() error {
	return l.db.Close()
}
