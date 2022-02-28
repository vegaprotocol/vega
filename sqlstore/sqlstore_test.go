package sqlstore_test

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/sqlstore"
)

var (
	testStore       *sqlstore.SQLStore
	sqlTestsEnabled bool = true
	minPort              = 30000
	maxPort              = 40000
	testDBPort      int
)

func TestMain(m *testing.M) {
	var err error
	testDBPort = getNextPort()
	sqlConfig := NewTestConfig(testDBPort)

	if sqlTestsEnabled {
		testStore, err = sqlstore.InitialiseTestStorage(
			logging.NewTestLogger(),
			sqlConfig,
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

func getNextPort() int {
	rand.Seed(time.Now().UnixNano())
	for {
		port := rand.Intn(maxPort-minPort+1) + minPort
		timeout := time.Millisecond * 100
		conn, err := net.DialTimeout("tcp", net.JoinHostPort("localhost", fmt.Sprintf("%d", port)), timeout)
		if err != nil {
			return port
		}

		if conn != nil {
			conn.Close()
		}
	}
}
