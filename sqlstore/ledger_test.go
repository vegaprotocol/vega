package sqlstore_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/sqlstore"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func addTestLedgerEntry(t *testing.T, ledger *sqlstore.Ledger,
	accountFrom entities.Account,
	accountTo entities.Account,
	block entities.Block,
) entities.LedgerEntry {
	ledgerEntry := entities.LedgerEntry{
		AccountFromID: accountFrom.ID,
		AccountToID:   accountTo.ID,
		Quantity:      decimal.NewFromInt(100),
		VegaTime:      block.VegaTime,
		TransferTime:  block.VegaTime.Add(-time.Second),
		Reference:     "some reference",
		Type:          "some string",
	}

	err := ledger.Add(&ledgerEntry)
	require.NoError(t, err)
	return ledgerEntry
}

func TestLedger(t *testing.T) {
	defer testStore.DeleteEverything()
	ctx := context.Background()

	blockStore := sqlstore.NewBlocks(testStore)
	assetStore := sqlstore.NewAssets(testStore)
	accountStore := sqlstore.NewAccounts(testStore)
	partyStore := sqlstore.NewParties(testStore)
	ledgerStore := sqlstore.NewLedger(testStore)

	// Account store should be empty to begin with
	ledgerEntries, err := ledgerStore.GetAll()
	assert.NoError(t, err)
	assert.Empty(t, ledgerEntries)

	block := addTestBlock(t, blockStore)
	asset := addTestAsset(t, assetStore, block)
	party := addTestParty(t, partyStore, block)
	accountFrom := addTestAccount(t, accountStore, party, asset, block)
	accountTo := addTestAccount(t, accountStore, party, asset, block)
	ledgerEntry := addTestLedgerEntry(t, ledgerStore, accountFrom, accountTo, block)

	err = ledgerStore.Flush(ctx)
	assert.NoError(t, err)

	// Add it again; we're allowed multiple ledger entries with the same parameters
	err = ledgerStore.Add(&ledgerEntry)
	assert.NoError(t, err)

	err = ledgerStore.Flush(ctx)
	assert.NoError(t, err)

	// Query and check we've got back an asset the same as the one we put in, once we give it an ID
	ledgerEntry.ID = 1
	fetchedLedgerEntry, err := ledgerStore.GetByID(1)
	assert.NoError(t, err)
	assert.Equal(t, ledgerEntry, fetchedLedgerEntry)

	// We should have added two entries in total
	ledgerEntriesAfter, err := ledgerStore.GetAll()
	assert.NoError(t, err)
	assert.Len(t, ledgerEntriesAfter, 2)
}
