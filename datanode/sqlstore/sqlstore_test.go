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

	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/datanode/sqlstore/helpers"
	"code.vegaprotocol.io/vega/datanode/utils/databasetest"
)

var (
	connectionSource *sqlstore.ConnectionSource
	testDBPort       int
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
		connectionSource = source
	}, postgresRuntimePath, sqlstore.EmbedMigrations)
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

func tempTransaction(t *testing.T) context.Context {
	t.Helper()

	ctx, err := connectionSource.WithTransaction(context.Background())
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = connectionSource.Rollback(ctx)
	})

	return ctx
}
