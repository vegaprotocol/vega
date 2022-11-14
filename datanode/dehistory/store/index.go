package store

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"

	bolt "go.etcd.io/bbolt"
)

var ErrIndexEntryNotFound = errors.New("index entry not found")

type BBoltBackedIndex struct {
	db *bolt.DB
}

const indexBucket = "index"

func NewIndex(dataDir string) (*BBoltBackedIndex, error) {
	db, err := bolt.Open(filepath.Join(dataDir, "index.db"), 0o666, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open db file:%w", err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(indexBucket))
		if err != nil {
			return fmt.Errorf("failed to create bucket:%w", err)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to update database:%w", err)
	}

	return &BBoltBackedIndex{
		db: db,
	}, nil
}

func (l *BBoltBackedIndex) Get(height int64) (SegmentIndexEntry, error) {
	var entry SegmentIndexEntry
	err := l.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(indexBucket))
		value := bucket.Get(heightToKey(height))
		if value == nil {
			return ErrIndexEntryNotFound
		}

		err := json.Unmarshal(value, &entry)
		if err != nil {
			return fmt.Errorf("failed to unmarshal value:%w", err)
		}
		return nil
	})

	if errors.Is(err, ErrIndexEntryNotFound) {
		return entry, err
	}

	if err != nil {
		return entry, fmt.Errorf("failed to get database view:%w", err)
	}

	return entry, nil
}

func (l *BBoltBackedIndex) Add(indexEntry SegmentIndexEntry) error {
	err := l.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(indexBucket))

		bytes, err := json.Marshal(indexEntry)
		if err != nil {
			return fmt.Errorf("failed to marshal index entry:%w", err)
		}

		err = bucket.Put(heightToKey(indexEntry.HeightTo), bytes)
		if err != nil {
			return fmt.Errorf("failed to put index entry:%w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to update database:%w", err)
	}

	return nil
}

func (l *BBoltBackedIndex) Remove(indexEntry SegmentIndexEntry) error {
	return l.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(indexBucket))

		err := bucket.Delete(heightToKey(indexEntry.HeightTo))
		if err != nil {
			return fmt.Errorf("failed to delete index entry:%w", err)
		}

		return nil
	})
}

func (l *BBoltBackedIndex) ListAllEntriesOldestFirst() ([]SegmentIndexEntry, error) {
	var segments []SegmentIndexEntry
	err := l.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(indexBucket))

		cur := bucket.Cursor()

		for k, v := cur.First(); k != nil; k, v = cur.Next() {
			var indexEntry SegmentIndexEntry
			err := json.Unmarshal(v, &indexEntry)
			if err != nil {
				return fmt.Errorf("failed to unmarshal index entry:%w", err)
			}

			segments = append(segments, indexEntry)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to iterate over index:%w", err)
	}

	sort.Slice(segments, func(i, j int) bool {
		return segments[i].HeightFrom < segments[j].HeightFrom
	})

	return segments, nil
}

func (l *BBoltBackedIndex) GetHighestBlockHeightEntry() (SegmentIndexEntry, error) {
	entries, err := l.ListAllEntriesOldestFirst()
	if err != nil {
		return SegmentIndexEntry{}, fmt.Errorf("failed to list all entries:%w", err)
	}

	if len(entries) == 0 {
		return SegmentIndexEntry{}, ErrIndexEntryNotFound
	}

	return entries[len(entries)-1], nil
}

func (l *BBoltBackedIndex) Close() error {
	return l.db.Close()
}

func heightToKey(height int64) []byte {
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, uint64(height))
	return bytes
}
