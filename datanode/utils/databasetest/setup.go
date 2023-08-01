package databasetest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/jackc/pgx/v4"

	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/logging"
)

var (
	sqlTestsEnabled       = true
	minPort               = 30000
	maxPort               = 40000
	postgresServerTimeout = time.Second * 10
)

func TestMain(m *testing.M, onSetupComplete func(sqlstore.Config, *sqlstore.ConnectionSource, *bytes.Buffer),
	postgresRuntimePath string, sqlFs fs.FS,
) int {
	testDBSocketDir := filepath.Join(postgresRuntimePath)
	testDBPort := 5432
	testDBHost := ""
	sqlConfig := NewTestConfig(testDBPort, testDBHost, testDBSocketDir)

	if sqlTestsEnabled {
		log := logging.NewTestLogger()

		err := os.Mkdir(postgresRuntimePath, fs.ModePerm)
		if err != nil {
			panic(err)
		}
		defer os.RemoveAll(postgresRuntimePath)

		postgresLog := &bytes.Buffer{}
		embeddedPostgres, err := sqlstore.StartEmbeddedPostgres(log, sqlConfig, postgresRuntimePath, postgresLog)
		if err != nil {
			log.Errorf("failed to start postgres: %s", postgresLog.String())
			panic(err)
		}

		log.Infof("Test DB Socket Directory: %s", testDBSocketDir)
		log.Infof("Test DB Port: %d", testDBPort)

		// Make sure the database has started before we run the tests.
		ctx, cancel := context.WithTimeout(context.Background(), postgresServerTimeout)

		op := func() error {
			connStr := sqlConfig.ConnectionConfig.GetConnectionString()
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
		connectionSource, err := sqlstore.NewTransactionalConnectionSource(log, sqlConfig.ConnectionConfig)
		if err != nil {
			panic(err)
		}
		defer embeddedPostgres.Stop()

		if err = sqlstore.WipeDatabaseAndMigrateSchemaToLatestVersion(log, sqlConfig.ConnectionConfig, sqlFs, false); err != nil {
			log.Errorf("failed to wipe database and migrate schema, dumping postgres log:\n %s", postgresLog.String())
			panic(err)
		}

		if err = sqlstore.ApplyDataRetentionPolicies(sqlConfig, log); err != nil {
			panic(err)
		}

		onSetupComplete(sqlConfig, connectionSource, postgresLog)

		return m.Run()
	}

	return 0
}

func NewTestConfig(port int, host, socketDir string) sqlstore.Config {
	sqlConfig := sqlstore.NewDefaultConfig()
	sqlConfig.UseEmbedded = true
	sqlConfig.ConnectionConfig.Port = port
	sqlConfig.ConnectionConfig.Host = host
	sqlConfig.ConnectionConfig.SocketDir = socketDir

	return sqlConfig
}

func GetNextFreePort() int {
	rand.Seed(time.Now().UnixNano())
	for {
		port := rand.Intn(maxPort-minPort+1) + minPort
		timeout := time.Millisecond * 100
		conn, err := net.DialTimeout("tcp", net.JoinHostPort("localhost", fmt.Sprintf("%d", port)), timeout)
		if err != nil && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
			return port
		}

		if conn != nil {
			conn.Close()
		}
	}
}
