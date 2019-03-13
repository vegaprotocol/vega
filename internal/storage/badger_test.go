package storage

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/dgraph-io/badger"
	"github.com/stretchr/testify/require"
)

func runBadgerStoreTest(t *testing.T, opts *badger.Options, test func(t *testing.T, bs *badgerStore)) {
	dir, err := ioutil.TempDir("/dev/shm", "badger-test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	if opts == nil {
		opts = new(badger.Options)
		*opts = badger.DefaultOptions
	}
	opts.Dir, opts.ValueDir = dir, dir

	db, err := badger.Open(*opts)
	require.NoError(t, err)
	defer db.Close()

	bs := badgerStore{db: db}
	test(t, &bs)
}

func testkey(prefix string, k int) string {
	return fmt.Sprintf("key%s%08d", prefix, k)
}

func testvalue(prefix string, k int) []byte {
	return []byte(fmt.Sprintf("val%s%08d", prefix, k))
}

func TestWriteBatch(t *testing.T) {

	runBadgerStoreTest(t, nil, func(t *testing.T, bs *badgerStore) {
		n := 100000
		for {
			kv := make(map[string][]byte)
			for i := 0; i < n; i++ {
				kv[testkey("", i)] = testvalue("", i)
			}
			b, err := bs.writeBatch(kv)
			require.NoError(t, err)
			fmt.Printf("Wrote %d records in %d batches.\n", n, b)
			if b > 1 {
				break
			}
			n *= 2
		}
	})
}
