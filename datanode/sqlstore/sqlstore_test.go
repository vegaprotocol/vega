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
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/rand"
	"strconv"
	"testing"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/datanode/utils/databasetest"
)

var (
	connectionSource *sqlstore.ConnectionSource
	testDBPort       int
)

func TestMain(m *testing.M) {
	databasetest.TestMain(m, func(cfg sqlstore.Config, source *sqlstore.ConnectionSource, snapshotPath string,
		postgresLog *bytes.Buffer,
	) {
		testDBPort = cfg.ConnectionConfig.Port
		connectionSource = source
	})
}

func DeleteEverything() {
	databasetest.DeleteEverything()
}

func NewTestConfig(port int) sqlstore.Config {
	return databasetest.NewTestConfig(port)
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

// Generate a 256 bit pseudo-random hash ID.
func generateID() string {
	randomString := strconv.FormatInt(rand.Int63(), 10)
	hash := sha256.Sum256([]byte(randomString))
	return hex.EncodeToString(hash[:])
}

func generateTxHash() entities.TxHash {
	return entities.TxHash(generateID())
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
