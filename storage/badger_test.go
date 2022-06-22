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

package storage

import (
	"fmt"
	"testing"

	"github.com/dgraph-io/badger/v2"
	"github.com/stretchr/testify/require"
)

func runBadgerStoreTest(t *testing.T, opts *badger.Options, test func(t *testing.T, bs *badgerStore)) {
	dir, tidy, err := TempDir("badger-test")
	if err != nil {
		t.Fatalf("Failed to create tmp dir: %s", err.Error())
	}
	defer tidy()

	if opts == nil {
		cpy := badger.DefaultOptions("")
		opts = &cpy
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
