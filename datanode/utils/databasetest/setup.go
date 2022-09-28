package databasetest

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	uuid "github.com/satori/go.uuid"

	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/logging"
	"github.com/cenkalti/backoff/v4"
	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/jackc/pgx/v4"
)

var (
	embeddedPostgres *embeddedpostgres.EmbeddedPostgres
	connectionSource *sqlstore.ConnectionSource
	sqlTestsEnabled  = true
	minPort          = 30000
	maxPort          = 40000
	testDBPort       int

	tableNames = [...]string{
		"accounts",
		"assets",
		"balances",
		"blocks",
		"chain",
		"checkpoints",
		"current_balances",
		"current_liquidity_provisions",
		"current_margin_levels",
		"delegations",
		"delegations_current",
		"deposits",
		"epochs",
		"erc20_multisig_signer_events",
		"ethereum_key_rotations",
		"key_rotations",
		"last_block",
		"ledger",
		"liquidity_provisions",
		"margin_levels",
		"market_data",
		"markets",
		"network_limits",
		"network_parameters",
		"node_signatures",
		"nodes",
		"nodes_announced",
		"oracle_data",
		"oracle_specs",
		"orders_history",
		"orders_live",
		"parties",
		"positions",
		"proposals",
		"ranking_scores",
		"reward_scores",
		"rewards",
		"risk_factors",
		"stake_linking",
		"trades",
		"transfers",
		"votes",
		"withdrawals",
	}

	postgresServerTimeout = time.Second * 10
)

func TestMain(m *testing.M, onSetupComplete func(sqlstore.Config, *sqlstore.ConnectionSource, string, *bytes.Buffer)) int {
	testDBPort = getNextPort()
	sqlConfig := NewTestConfig(testDBPort)

	if sqlTestsEnabled {
		log := logging.NewTestLogger()

		testID := uuid.NewV4().String()
		tempDir, err := ioutil.TempDir("", testID)
		if err != nil {
			panic(err)
		}

		var postgresRuntimePath string
		var postgresLog *bytes.Buffer
		embeddedPostgres, postgresRuntimePath, postgresLog, err = sqlstore.StartEmbeddedPostgres(log, sqlConfig, tempDir)
		if err != nil {
			panic(err)
		}

		snapshotsDir := filepath.Join(postgresRuntimePath, "snapshots")
		err = os.Mkdir(snapshotsDir, os.ModePerm)
		if err != nil {
			panic(fmt.Errorf("failed to create snapshots directory: %w", err))
		}
		defer os.RemoveAll(snapshotsDir)

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
		connectionSource, err = sqlstore.NewTransactionalConnectionSource(log, sqlConfig.ConnectionConfig)
		if err != nil {
			panic(err)
		}
		defer embeddedPostgres.Stop()

		if err = sqlstore.MigrateToLatestSchema(log, sqlConfig); err != nil {
			panic(err)
		}

		if err = sqlstore.ApplyDataRetentionPolicies(sqlConfig); err != nil {
			panic(err)
		}

		onSetupComplete(sqlConfig, connectionSource, snapshotsDir, postgresLog)

		return m.Run()
	}

	return 0
}

func DeleteEverything() {
	ctx, cancelFn := context.WithTimeout(context.Background(), postgresServerTimeout)
	defer cancelFn()
	sqlConfig := NewTestConfig(testDBPort)
	connStr := connectionString(sqlConfig.ConnectionConfig)
	conn, err := pgx.Connect(ctx, connStr)
	defer func() {
		err = conn.Close(context.Background())
		if err != nil {
			log := logging.NewTestLogger()
			log.Errorf("failed to close connection:%w", err)
		}
	}()
	if err != nil {
		panic(fmt.Errorf("failed to delete everything:%w", err))
	}

	for _, table := range tableNames {
		if _, err := conn.Exec(context.Background(), "truncate table "+table+" CASCADE"); err != nil {
			panic(fmt.Errorf("error truncating table: %s %w", table, err))
		}
	}
}

func connectionString(config sqlstore.ConnectionConfig) string {
	//nolint:nosprintfhostport
	return fmt.Sprintf("postgresql://%s:%s@%s:%d/%s",
		config.Username,
		config.Password,
		config.Host,
		config.Port,
		config.Database)
}

func NewTestConfig(port int) sqlstore.Config {
	sqlConfig := sqlstore.NewDefaultConfig()
	sqlConfig.UseEmbedded = true
	sqlConfig.ConnectionConfig.Port = port

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
