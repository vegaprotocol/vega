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
	sqlTestsEnabled bool = true
	testDBPort           = 38233
)

func TestMain(m *testing.M) {
	var err error

	sqlConfig := NewTestConfig(testDBPort)
	if sqlTestsEnabled {
		testStore, err = sqlstore.InitialiseTestStorage(
			logging.NewTestLogger(),
			sqlConfig,
			&paths.DefaultPaths{},
		)

		if err != nil {
			panic(err)
		}
		defer testStore.Stop()

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

func NewTestConfig(port int) sqlstore.Config {
	sqlConfig := sqlstore.NewDefaultConfig()
	sqlConfig.Enabled = true
	sqlConfig.UseEmbedded = true
	sqlConfig.Port = port

	return sqlConfig
}
