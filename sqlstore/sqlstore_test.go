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

package sqlstore_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"strconv"
	"testing"
	"time"

	uuid "github.com/satori/go.uuid"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/sqlstore"
	"github.com/cenkalti/backoff/v4"
	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/jackc/pgx/v4"
)

var (
	embeddedPostgres *embeddedpostgres.EmbeddedPostgres
	connectionSource *sqlstore.ConnectionSource
	sqlTestsEnabled  bool = true
	minPort               = 30000
	maxPort               = 40000
	testDBPort       int

	tableNames = [...]string{
		"ledger", "accounts", "parties", "assets", "blocks", "node_signatures",
		"erc20_multisig_signer_events", "trades", "market_data", "orders_live", "orders_history",
		"margin_levels", "liquidity_provisions", "nodes", "ranking_scores", "reward_scores", "delegations", "rewards",
		"nodes_announced",
	}

	postgresServerTimeout = time.Second * 10
)

func TestMain(m *testing.M) {
	testDBPort = getNextPort()
	sqlConfig := NewTestConfig(testDBPort)

	if sqlTestsEnabled {
		log := logging.NewTestLogger()

		testID := uuid.NewV4().String()
		tempDir, err := ioutil.TempDir("", testID)
		if err != nil {
			panic(err)
		}

		embeddedPostgres, err = sqlstore.StartEmbeddedPostgres(log, sqlConfig, tempDir)
		if err != nil {
			panic(err)
		}

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

		m.Run()
	}
}

func DeleteEverything() {
	ctx, cancelFn := context.WithTimeout(context.Background(), postgresServerTimeout)
	defer cancelFn()
	sqlConfig := NewTestConfig(testDBPort)
	connStr := connectionString(sqlConfig.ConnectionConfig)
	conn, err := pgx.Connect(ctx, connStr)
	defer conn.Close(context.Background())
	if err != nil {
		panic(fmt.Errorf("failed to delete everything:%w", err))
	}

	for _, table := range tableNames {
		if _, err := conn.Exec(context.Background(), "truncate table "+table+" CASCADE"); err != nil {
			panic(fmt.Errorf("error truncating table: %s %w", table, err))
		}
	}
}

// Generate a 256 bit pseudo-random hash ID based on the time
func generateID() string {
	currentTime := time.Now().UnixNano()
	currentTimeString := strconv.FormatInt(currentTime, 10)
	hash := sha256.Sum256([]byte(currentTimeString))
	return hex.EncodeToString(hash[:])
}

func generateEthereumAddress() string {
	currentTime := time.Now().UnixNano()
	currentTimeString := strconv.FormatInt(currentTime, 10)
	hash := sha256.Sum256([]byte(currentTimeString))
	return "0x" + hex.EncodeToString(hash[1:21])
}

func generateTendermintPublicKey() string {
	currentTime := time.Now().UnixNano()
	currentTimeString := strconv.FormatInt(currentTime, 10)
	hash := sha256.Sum256([]byte(currentTimeString))
	return base64.StdEncoding.EncodeToString(hash[:])
}

func NewTestConfig(port int) sqlstore.Config {
	sqlConfig := sqlstore.NewDefaultConfig()
	sqlConfig.Enabled = true
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
