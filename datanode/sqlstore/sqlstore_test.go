// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package sqlstore_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/datanode/sqlstore/helpers"
	"code.vegaprotocol.io/vega/datanode/utils/databasetest"
)

var (
	connectionSource *sqlstore.ConnectionSource
	testDBPort       int
	testDBSocketDir  string
)

func TestMain(m *testing.M) {
	tempDir, err := os.MkdirTemp("", "datanode")
	if err != nil {
		panic(err)
	}
	postgresRuntimePath := filepath.Join(tempDir, "sqlstore")
	defer os.RemoveAll(tempDir)

	databasetest.TestMain(m, func(cfg sqlstore.Config, source *sqlstore.ConnectionSource,
		postgresLog *bytes.Buffer,
	) {
		testDBPort = cfg.ConnectionConfig.Port
		testDBSocketDir = cfg.ConnectionConfig.SocketDir
		connectionSource = source
	}, postgresRuntimePath, sqlstore.EmbedMigrations)
}

func NewTestConfig() sqlstore.Config {
	return databasetest.NewTestConfig(testDBPort, "", testDBSocketDir)
}

func generateTxHash() entities.TxHash {
	return entities.TxHash(helpers.GenerateID())
}

func generateEthereumAddress() string {
	randomString := strconv.FormatInt(rand.Int63(), 10)
	hash := sha256.Sum256([]byte(randomString))
	return "0x" + hex.EncodeToString(hash[1:21])
}

func generateTendermintPublicKey() string {
	randomString := strconv.FormatInt(rand.Int63(), 10)
	hash := sha256.Sum256([]byte(randomString))
	return base64.StdEncoding.EncodeToString(hash[:])
}

func tempTransaction(t *testing.T) (context.Context, func()) {
	t.Helper()
	ctx := context.Background()
	ctx, err := connectionSource.WithTransaction(ctx)
	assert.NoError(t, err)
	cleanup := func() {
		connectionSource.Rollback(ctx)
	}
	return ctx, cleanup
}
