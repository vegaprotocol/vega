package sqlstore_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/sqlstore"
	"github.com/cenkalti/backoff/v4"
	"github.com/jackc/pgx/v4"
)

var (
	testStore       *sqlstore.SQLStore
	sqlTestsEnabled bool = true
	minPort              = 30000
	maxPort              = 40000
	testDBPort      int

	postgresServerTimeout = time.Second * 10
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

		// Make sure the database has started before we run the tests.
		ctx, cancel := context.WithTimeout(context.Background(), postgresServerTimeout)

		op := func() error {
			connStr := connectionString(sqlConfig)
			conn, err := pgx.Connect(ctx, connStr)

			if err != nil {
				return err
			}

			return conn.Ping(ctx)
		}

		if err := backoff.Retry(op, backoff.NewExponentialBackOff()); err != nil {
			cancel()
			panic(err)
		}

		cancel()

		defer testStore.Stop()

		m.Run()
	}
}

// Generate a 256 bit pseudo-random hash ID based on the time
func generateID() string {
	currentTime := time.Now().UnixNano()
	currentTimeString := strconv.FormatInt(currentTime, 10)
	hash := sha256.Sum256([]byte(currentTimeString))
	return hex.EncodeToString(hash[:])
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
