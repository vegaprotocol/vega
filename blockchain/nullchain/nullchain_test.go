package nullchain_test

import (
	"path/filepath"
	"testing"

	vgfs "code.vegaprotocol.io/shared/libs/fs"

	"code.vegaprotocol.io/vega/blockchain/nullchain"
	"code.vegaprotocol.io/vega/blockchain/nullchain/mocks"
	"code.vegaprotocol.io/vega/logging"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func newGenesisFile(t *testing.T) string {
	t.Helper()
	data := "{ \"appstate\": { \"stuff\": \"stuff\" }}"

	filePath := filepath.Join(t.TempDir(), "genesis.json")
	if err := vgfs.WriteFile(filePath, []byte(data)); err != nil {
		t.Fatalf("couldn't write file: %v", err)
	}
	return filePath
}

func TestNullChain(t *testing.T) {
	t.Run("test new nullchain calls initchain", testNewNullChainCallsInitChain)
}

func testNewNullChainCallsInitChain(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	app := mocks.NewMockApplicationService(ctrl)
	app.EXPECT().InitChain(gomock.Any()).Times(1)

	cfg := nullchain.NewDefaultConfig()
	cfg.GenesisFile = newGenesisFile(t)

	n, err := nullchain.NewClient(logging.NewTestLogger(), cfg, app)
	assert.NoError(t, err)
	assert.NotNil(t, n)
}
