package sqlstore_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/sqlstore"
	v1 "code.vegaprotocol.io/protos/vega/commands/v1"
	"github.com/jackc/pgx/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotary(t *testing.T) {
	t.Run("Adding a single signature", testAddSignatures)
	t.Run("Adding multiple signatures for multiple resources", testAddMultipleSignatures)
	t.Run("Getting a non-existing resource signatures", testNoResource)
}

func setupNotaryStoreTests(t *testing.T, ctx context.Context) (*sqlstore.Notary, *pgx.Conn) {
	t.Helper()
	err := testStore.DeleteEverything()
	require.NoError(t, err)
	ns := sqlstore.NewNotary(testStore)

	config := NewTestConfig(testDBPort)

	conn, err := pgx.Connect(ctx, connectionString(config))
	require.NoError(t, err)

	return ns, conn
}

func testAddSignatures(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	ws, conn := setupNotaryStoreTests(t, ctx)

	var rowCount int

	err := conn.QueryRow(ctx, `select count(*) from withdrawals`).Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	ns := getTestNodeSignature(t, "deadbeef")
	err = ws.Add(ns)
	require.NoError(t, err)

	err = conn.QueryRow(ctx, `select count(*) from node_signatures`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, rowCount)
}

func testAddMultipleSignatures(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	ws, _ := setupNotaryStoreTests(t, ctx)

	nodeSig1 := getTestNodeSignature(t, "deadbeef")
	nodeSig2 := getTestNodeSignature(t, "deadbeef")         // this will have a different sig
	nodeSig3 := getTestNodeSignature(t, "deadbeef")         // this will be a dupe of ns2
	nodeSig4 := getTestNodeSignature(t, "deadbeefdeadbeef") // this will have a different sig and id

	nodeSig2.Sig = []byte("iamdifferentsig")
	nodeSig4.Sig = []byte("iamdifferentsigagain")

	err := ws.Add(nodeSig1)
	require.NoError(t, err)

	err = ws.Add(nodeSig2)
	require.NoError(t, err)

	err = ws.Add(nodeSig3)
	require.NoError(t, err)

	err = ws.Add(nodeSig4)
	require.NoError(t, err)

	res, err := ws.GetByResourceID(ctx, "deadbeef")
	require.NoError(t, err)
	require.Len(t, res, 2)

	res, err = ws.GetByResourceID(ctx, "deadbeefdeadbeef")
	require.NoError(t, err)
	require.Len(t, res, 1)
}

func getTestNodeSignature(t *testing.T, id string) *entities.NodeSignature {
	t.Helper()
	ns, err := entities.NodeSignatureFromProto(
		&v1.NodeSignature{
			Id:   id,
			Sig:  []byte("iamsig"),
			Kind: v1.NodeSignatureKind_NODE_SIGNATURE_KIND_ASSET_WITHDRAWAL,
		},
	)
	require.NoError(t, err)
	return ns
}

func testNoResource(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	ws, _ := setupNotaryStoreTests(t, ctx)

	res, err := ws.GetByResourceID(ctx, "deadbeefdeadbeef")
	require.NoError(t, err)
	require.Len(t, res, 0)
}
