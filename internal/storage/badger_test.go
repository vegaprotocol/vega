package storage_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"code.vegaprotocol.io/vega/internal/storage"

	"github.com/dgraph-io/badger"
	"github.com/stretchr/testify/require"
)

func tempDir(t *testing.T, prefix string) string {
	baseTempDirs := []string{"/dev/shm", os.TempDir()}
	for _, baseTempDir := range baseTempDirs {
		_, err := os.Stat(baseTempDir)
		if err == nil {
			dir, err := ioutil.TempDir(baseTempDir, prefix)
			require.NoError(t, err)
			return dir
			// Remember: defer os.RemoveAll(dir)
		}
	}
	panic("Could not find a temp dir")
}

func runBadgerStoreTest(t *testing.T, opts *badger.Options, test func(t *testing.T, bs *storage.BadgerStore)) {
	dir := tempDir(t, "badger-test")
	defer os.RemoveAll(dir)

	if opts == nil {
		cpy := badger.DefaultOptions
		opts = &cpy
	}
	opts.Dir, opts.ValueDir = dir, dir

	db, err := badger.Open(*opts)
	require.NoError(t, err)
	defer db.Close()

	bs := storage.BadgerStore{DB: db}
	test(t, &bs)
}

func testkey(prefix string, k int) string {
	return fmt.Sprintf("key%s%08d", prefix, k)
}

func testvalue(prefix string, k int) []byte {
	return []byte(fmt.Sprintf("val%s%08d", prefix, k))
}

func TestWriteBatch(t *testing.T) {

	runBadgerStoreTest(t, nil, func(t *testing.T, bs *storage.BadgerStore) {
		n := 100000
		for {
			kv := make(map[string][]byte)
			for i := 0; i < n; i++ {
				kv[testkey("", i)] = testvalue("", i)
			}
			b, err := bs.WriteBatch(kv)
			require.NoError(t, err)
			fmt.Printf("Wrote %d records in %d batches.\n", n, b)
			if b > 1 {
				break
			}
			n *= 2
		}
	})
}
