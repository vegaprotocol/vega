// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/datanode/utils/databasetest"

	"github.com/stretchr/testify/require"
)

var (
	connectionSource *sqlstore.ConnectionSource
	testDBPort       int
	testConfig       sqlstore.Config
)

func TestMain(m *testing.M) {
	ctx, cfunc := context.WithCancel(context.Background())
	defer cfunc()
	tempDir, err := os.MkdirTemp("", "datanode")
	if err != nil {
		panic(err)
	}
	postgresRuntimePath := filepath.Join(tempDir, "sqlstore")
	defer os.RemoveAll(tempDir)

	databasetest.TestMain(m, ctx, func(cfg sqlstore.Config, source *sqlstore.ConnectionSource,
		postgresLog *bytes.Buffer,
	) {
		// ensures nested transactions execute the post-commit hooks while the parent transaction still rolls back the overall changes.
		source.ToggleTest()
		testDBPort = cfg.ConnectionConfig.Port
		connectionSource = source
		testConfig = cfg
	}, postgresRuntimePath, sqlstore.EmbedMigrations)
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

func tempTransaction(t *testing.T) context.Context {
	t.Helper()

	ctx, err := connectionSource.WithTransaction(context.Background())
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = connectionSource.Rollback(ctx)
	})

	return ctx
}

func getEnums[T entities.ProtoEnum](t *testing.T, enumsT T) map[int32]string {
	t.Helper()
	return enumsT.GetEnums()
}
