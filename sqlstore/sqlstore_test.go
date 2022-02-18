package sqlstore_test

import (
	"crypto/sha256"
	"strconv"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/sqlstore"
	"code.vegaprotocol.io/shared/paths"
)

var (
	testStore       *sqlstore.SQLStore
	sqlTestsEnabled bool = false
)

func TestMain(m *testing.M) {
	var err error
	// TODO: Launch a test database instance; tests disabled for now
	if sqlTestsEnabled {
		testStore, err = sqlstore.InitialiseStorage(
			logging.NewTestLogger(),
			sqlstore.NewDefaultConfig(),
			&paths.DefaultPaths{},
		)
		if err != nil {
			panic(err)
		}

		m.Run()
	}
}

// Generate a 256 bit pseudo-random hash ID based on the time
func generateID() []byte {
	currentTime := time.Now().UnixNano()
	currentTimeString := strconv.FormatInt(currentTime, 10)
	hash := sha256.Sum256([]byte(currentTimeString))
	return hash[:]
}
