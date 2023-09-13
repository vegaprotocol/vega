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
	"strconv"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/datanode/sqlstore/helpers"
	"code.vegaprotocol.io/vega/protos/vega"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ledgerEntryEqual(t *testing.T, expected, actual entities.LedgerEntry) {
	t.Helper()

	assert.Equal(t, expected.FromAccountID, actual.FromAccountID)
	assert.Equal(t, expected.ToAccountID, actual.ToAccountID)
	assert.Equal(t, expected.Quantity, actual.Quantity)
	assert.Equal(t, expected.ToAccountBalance, actual.ToAccountBalance)
	assert.Equal(t, expected.FromAccountBalance, actual.FromAccountBalance)
	assert.Equal(t, expected.TxHash, actual.TxHash)
	assert.Equal(t, expected.VegaTime, actual.VegaTime)
	assert.Equal(t, expected.Type, actual.Type)
}

func addTestLedgerEntry(t *testing.T, ledger *sqlstore.Ledger,
	fromAccount entities.Account,
	toAccount entities.Account,
	block entities.Block,
	quantity int64,
	transferType entities.LedgerMovementType,
	fromAccountBalance, toAccountBalance int64,
	txHash entities.TxHash,
) entities.LedgerEntry {
	t.Helper()
	ledgerEntry := entities.LedgerEntry{
		FromAccountID:      fromAccount.ID,
		ToAccountID:        toAccount.ID,
		Quantity:           decimal.NewFromInt(quantity),
		VegaTime:           block.VegaTime,
		TransferTime:       block.VegaTime.Add(-time.Second),
		Type:               transferType,
		FromAccountBalance: decimal.NewFromInt(fromAccountBalance),
		ToAccountBalance:   decimal.NewFromInt(toAccountBalance),
		TxHash:             txHash,
	}

	err := ledger.Add(ledgerEntry)
	require.NoError(t, err)
	return ledgerEntry
}

func TestLedger(t *testing.T) {
	ctx := tempTransaction(t)

	// Prepare environment entities.
	blockStore := sqlstore.NewBlocks(connectionSource)
	assetStore := sqlstore.NewAssets(connectionSource)
	accountStore := sqlstore.NewAccounts(connectionSource)
	partyStore := sqlstore.NewParties(connectionSource)
	marketStore := sqlstore.NewMarkets(connectionSource)
	ledgerStore := sqlstore.NewLedger(connectionSource)

	// Setup 4 assets
	asset1 := addTestAsset(t, ctx, assetStore, addTestBlock(t, ctx, blockStore))
	asset2 := addTestAsset(t, ctx, assetStore, addTestBlock(t, ctx, blockStore))
	asset3 := addTestAsset(t, ctx, assetStore, addTestBlock(t, ctx, blockStore))

	var blocks []entities.Block
	var parties []entities.Party
	var markets []entities.Market
	var accounts []entities.Account

	/*
		--- env ---
		block 0		block 1		block 2		block 3		block 4		block 5		block 6		block 7		block 8		block 9		block 10
		party 0		party 1		party 2		party 3		party 4		party 5		party 6		party 7		party 8		party 9		party 10

		market 0	market 1 	market 2	market 3	market 4	market 5	market 6	market 7	market 8	market 9	market 10

		--- accounts ---
		accounts[0] => asset1, parties[0], markets[0], vega.AccountType_ACCOUNT_TYPE_GENERAL
		accounts[1] => asset1, parties[0], markets[1], vega.AccountType_ACCOUNT_TYPE_GENERAL
		accounts[2] => asset1, parties[1], markets[1], vega.AccountType_ACCOUNT_TYPE_GENERAL
		accounts[3] => asset1, parties[1], markets[2], vega.AccountType_ACCOUNT_TYPE_GENERAL

		accounts[4] => asset2, parties[2], markets[2], vega.AccountType_ACCOUNT_TYPE_INSURANCE
		accounts[5] => asset2, parties[2], markets[3], vega.AccountType_ACCOUNT_TYPE_INSURANCE
		accounts[6] => asset2, parties[3], markets[3], vega.AccountType_ACCOUNT_TYPE_INSURANCE
		accounts[7] => asset2, parties[3], markets[4], vega.AccountType_ACCOUNT_TYPE_INSURANCE
		accounts[8] => asset2, parties[4], markets[4], vega.AccountType_ACCOUNT_TYPE_INSURANCE
		accounts[9] => asset2, parties[4], markets[5], vega.AccountType_ACCOUNT_TYPE_INSURANCE
		accounts[10] => asset2, parties[5], markets[5], vega.AccountType_ACCOUNT_TYPE_GENERAL
		accounts[11] => asset2, parties[5], markets[6], vega.AccountType_ACCOUNT_TYPE_GENERAL

		accounts[12] => asset3, parties[6], markets[6], vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY
		accounts[13] => asset3, parties[6], markets[7], vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY
		accounts[14] => asset3, parties[7], markets[7], vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY
		accounts[15] => asset3, parties[7], markets[8], vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY
		accounts[16] => asset3, parties[8], markets[8], vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY
		accounts[17] => asset3, parties[8], markets[9], vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY
		accounts[18] => asset3, parties[9], markets[9], vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY
		accounts[19] => asset3, parties[9], markets[10], vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY
		accounts[20] => asset3, parties[10], markets[10], vega.AccountType_ACCOUNT_TYPE_FEES_INSURANCE
		accounts[21] => asset3, parties[10], markets[11], vega.AccountType_ACCOUNT_TYPE_FEES_INSURANCE
	*/
	for i := 0; i < 17; i++ {
		blocks = append(blocks, addTestBlockForTime(t, ctx, blockStore, time.Now().Add((-26*time.Hour)-(time.Duration(5-i)*time.Second))))
		parties = append(parties, addTestParty(t, ctx, partyStore, blocks[i]))
		markets = append(markets, helpers.GenerateMarkets(t, ctx, 1, blocks[0], marketStore)[0])
	}

	for i := 0; i < 11; i++ {
		var mt int
		if i < 11-1 {
			mt = i + 1
		} else {
			mt = i - 1
		}

		if i < 2 {
			// accounts 0, 1, 2, 3
			fromAccount := helpers.AddTestAccountWithMarketAndType(t, ctx, accountStore, parties[i], asset1, blocks[i], markets[i].ID, vega.AccountType_ACCOUNT_TYPE_GENERAL)
			toAccount := helpers.AddTestAccountWithMarketAndType(t, ctx, accountStore, parties[i], asset1, blocks[i], markets[mt].ID, vega.AccountType_ACCOUNT_TYPE_GENERAL)
			accounts = append(accounts, fromAccount)
			accounts = append(accounts, toAccount)
			continue
		}

		// accounts 4, 5, 6, 7, 8, 9, 10, 11
		if i >= 2 && i < 5 {
			fromAccount := helpers.AddTestAccountWithMarketAndType(t, ctx, accountStore, parties[i], asset2, blocks[i], markets[i].ID, vega.AccountType_ACCOUNT_TYPE_INSURANCE)
			toAccount := helpers.AddTestAccountWithMarketAndType(t, ctx, accountStore, parties[i], asset2, blocks[i], markets[mt].ID, vega.AccountType_ACCOUNT_TYPE_INSURANCE)
			accounts = append(accounts, fromAccount)
			accounts = append(accounts, toAccount)
			continue
		}

		// accounts 10, 11
		if i >= 5 && i < 6 {
			fromAccount := helpers.AddTestAccountWithMarketAndType(t, ctx, accountStore, parties[i], asset2, blocks[i], markets[i].ID, vega.AccountType_ACCOUNT_TYPE_GENERAL)
			toAccount := helpers.AddTestAccountWithMarketAndType(t, ctx, accountStore, parties[i], asset2, blocks[i], markets[mt].ID, vega.AccountType_ACCOUNT_TYPE_GENERAL)
			accounts = append(accounts, fromAccount)
			accounts = append(accounts, toAccount)
			continue
		}

		// accounts 12, 13, 14, 15, 16, 17, 18, 19
		if i >= 6 && i < 10 {
			fromAccount := helpers.AddTestAccountWithMarketAndType(t, ctx, accountStore, parties[i], asset3, blocks[i], markets[i].ID, vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY)
			toAccount := helpers.AddTestAccountWithMarketAndType(t, ctx, accountStore, parties[i], asset3, blocks[i], markets[mt].ID, vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY)
			accounts = append(accounts, fromAccount)
			accounts = append(accounts, toAccount)
			continue
		}

		// accounts 20, 21
		fromAccount := helpers.AddTestAccountWithMarketAndType(t, ctx, accountStore, parties[i], asset3, blocks[i], markets[i].ID, vega.AccountType_ACCOUNT_TYPE_INSURANCE)
		toAccount := helpers.AddTestAccountWithMarketAndType(t, ctx, accountStore, parties[i], asset3, blocks[i], markets[mt].ID, vega.AccountType_ACCOUNT_TYPE_INSURANCE)
		accounts = append(accounts, fromAccount)
		accounts = append(accounts, toAccount)
	}

	/*
		--- Transfers ---
		Asset1:
		accounts[0]->accounts[1] => asset1, parties[0], markets[0-1], vega.AccountType_ACCOUNT_TYPE_GENERAL
		accounts[2]->accounts[3] => asset1, parties[1], markets[1-2], vega.AccountType_ACCOUNT_TYPE_GENERAL

		Asset2:
		accounts[4]->accounts[5] => asset2, parties[2], markets[2-3], vega.AccountType_ACCOUNT_TYPE_INSURANCE
		accounts[6]->accounts[7] => asset2, parties[3], markets[3-4], vega.AccountType_ACCOUNT_TYPE_INSURANCE
		accounts[6]->accounts[7] => asset2, parties[3], markets[3-4], vega.AccountType_ACCOUNT_TYPE_INSURANCE
		accounts[6]->accounts[7] => asset2, parties[3], markets[3-4], vega.AccountType_ACCOUNT_TYPE_INSURANCE
		accounts[8]->accounts[9] => asset2, parties[4], markets[4-5], vega.AccountType_ACCOUNT_TYPE_INSURANCE
		accounts[10]->accounts[11] => asset2, parties[5], markets[5-6], vega.AccountType_ACCOUNT_TYPE_GENERAL

		accounts[5]->accounts[10] => asset2, parties[2-5], markets[3-5], vega.AccountType_ACCOUNT_TYPE_INSURANCE -> vega.AccountType_ACCOUNT_TYPE_GENERAL
		accounts[5]->accounts[11] => asset2, parties[2-5], markets[3-6], vega.AccountType_ACCOUNT_TYPE_INSURANCE -> vega.AccountType_ACCOUNT_TYPE_GENERAL
		accounts[4]->accounts[11] => asset2, parties[2-5], markets[2-6], vega.AccountType_ACCOUNT_TYPE_INSURANCE -> vega.AccountType_ACCOUNT_TYPE_GENERAL

		Asset3:
		accounts[14]->accounts[16] => asset3, parties[7-8], markets[7-8], vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY

		accounts[17]->accounts[15] => asset3, parties[8-7], markets[9-8], vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY

		accounts[21]->accounts[15] => asset3, parties[10-7], markets[9-8], vega.AccountType_ACCOUNT_TYPE_INSURANCE -> vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY

	*/
	var ledgerEntries []entities.LedgerEntry
	ledgerEntries = append(ledgerEntries, addTestLedgerEntry(t, ledgerStore, accounts[0], accounts[1], blocks[1], int64(15), entities.LedgerMovementTypeBondSlashing, int64(500), int64(115), txHashFromString("ledger_entry_1")))
	ledgerEntries = append(ledgerEntries, addTestLedgerEntry(t, ledgerStore, accounts[2], accounts[3], blocks[2], int64(10), entities.LedgerMovementTypeBondSlashing, int64(170), int64(17890), txHashFromString("ledger_entry_2")))
	ledgerEntries = append(ledgerEntries, addTestLedgerEntry(t, ledgerStore, accounts[4], accounts[5], blocks[3], int64(25), entities.LedgerMovementTypeBondSlashing, int64(1700), int64(2590), txHashFromString("ledger_entry_3")))
	ledgerEntries = append(ledgerEntries, addTestLedgerEntry(t, ledgerStore, accounts[6], accounts[7], blocks[4], int64(80), entities.LedgerMovementTypeBondSlashing, int64(2310), int64(17000), txHashFromString("ledger_entry_4")))
	ledgerEntries = append(ledgerEntries, addTestLedgerEntry(t, ledgerStore, accounts[8], accounts[9], blocks[5], int64(1), entities.LedgerMovementTypeDeposit, int64(120), int64(900), txHashFromString("ledger_entry_5")))
	ledgerEntries = append(ledgerEntries, addTestLedgerEntry(t, ledgerStore, accounts[10], accounts[11], blocks[6], int64(40), entities.LedgerMovementTypeDeposit, int64(1500), int64(5680), txHashFromString("ledger_entry_6")))
	ledgerEntries = append(ledgerEntries, addTestLedgerEntry(t, ledgerStore, accounts[14], accounts[16], blocks[7], int64(12), entities.LedgerMovementTypeDeposit, int64(5000), int64(9100), txHashFromString("ledger_entry_7")))
	ledgerEntries = append(ledgerEntries, addTestLedgerEntry(t, ledgerStore, accounts[17], accounts[15], blocks[8], int64(14), entities.LedgerMovementTypeDeposit, int64(180), int64(1410), txHashFromString("ledger_entry_8")))
	ledgerEntries = append(ledgerEntries, addTestLedgerEntry(t, ledgerStore, accounts[21], accounts[15], blocks[9], int64(28), entities.LedgerMovementTypeDeposit, int64(2180), int64(1438), txHashFromString("ledger_entry_9")))
	ledgerEntries = append(ledgerEntries, addTestLedgerEntry(t, ledgerStore, accounts[5], accounts[11], blocks[10], int64(3), entities.LedgerMovementTypeRewardPayout, int64(2587), int64(5683), txHashFromString("ledger_entry_10")))
	ledgerEntries = append(ledgerEntries, addTestLedgerEntry(t, ledgerStore, accounts[5], accounts[10], blocks[11], int64(5), entities.LedgerMovementTypeRewardPayout, int64(2582), int64(1510), txHashFromString("ledger_entry_11")))
	ledgerEntries = append(ledgerEntries, addTestLedgerEntry(t, ledgerStore, accounts[6], accounts[7], blocks[12], int64(9), entities.LedgerMovementTypeRewardPayout, int64(2301), int64(17009), txHashFromString("ledger_entry_12")))
	ledgerEntries = append(ledgerEntries, addTestLedgerEntry(t, ledgerStore, accounts[6], accounts[7], blocks[13], int64(41), entities.LedgerMovementTypeRewardPayout, int64(2260), int64(17050), txHashFromString("ledger_entry_13")))
	_ = append(ledgerEntries, addTestLedgerEntry(t, ledgerStore, accounts[4], accounts[11], blocks[13], int64(72), entities.LedgerMovementTypeRewardPayout, int64(2188), int64(17122), txHashFromString("ledger_entry_14")))

	tStart, _ := time.Parse("2006 Jan 02 15:04:05", "2012 Dec 07 00:00:00")
	tEnd := time.Now()

	t.Run("get all ledger records", func(t *testing.T) {
		// Account store should be empty to begin with
		ledgerEntries, err := ledgerStore.GetAll(ctx)
		assert.NoError(t, err)
		assert.Empty(t, ledgerEntries)
	})

	_, err := ledgerStore.Flush(ctx)
	assert.NoError(t, err)

	t.Run("get by tx hash", func(t *testing.T) {
		fetchedEntries, err := ledgerStore.GetByTxHash(ctx, ledgerEntries[0].TxHash)
		assert.NoError(t, err)
		ledgerEntryEqual(t, ledgerEntries[0], fetchedEntries[0])

		fetchedEntries2, err := ledgerStore.GetByTxHash(ctx, ledgerEntries[2].TxHash)
		assert.NoError(t, err)
		ledgerEntryEqual(t, ledgerEntries[2], fetchedEntries2[0])
	})

	t.Run("query ledger entries with no filters", func(t *testing.T) {
		// Set filters for AccountFrom and AcountTo IDs
		filter := &entities.LedgerEntryFilter{
			FromAccountFilter: entities.AccountFilter{},
			ToAccountFilter:   entities.AccountFilter{},
		}

		entries, _, err := ledgerStore.Query(ctx,
			filter,
			entities.DateRange{Start: &tStart, End: &tEnd},
			entities.CursorPagination{},
		)

		assert.ErrorIs(t, err, sqlstore.ErrLedgerEntryFilterForParty)
		// Output entries for accounts positions:
		// None
		assert.Nil(t, entries)
	})

	t.Run("query ledger entries with filters", func(t *testing.T) {
		t.Run("by fromAccount filter", func(t *testing.T) {
			// Set filters for FromAccount and AcountTo IDs
			filter := &entities.LedgerEntryFilter{
				FromAccountFilter: entities.AccountFilter{
					AssetID: asset1.ID,
				},
				ToAccountFilter: entities.AccountFilter{},
			}

			entries, _, err := ledgerStore.Query(ctx,
				filter,
				entities.DateRange{Start: &tStart, End: &tEnd},
				entities.CursorPagination{},
			)

			assert.ErrorIs(t, err, sqlstore.ErrLedgerEntryFilterForParty)
			// Output entries for accounts positions:
			// None
			assert.Nil(t, entries)

			filter.FromAccountFilter.PartyIDs = []entities.PartyID{parties[3].ID}
			entries, _, err = ledgerStore.Query(ctx,
				filter,
				entities.DateRange{Start: &tStart, End: &tEnd},
				entities.CursorPagination{},
			)

			assert.NoError(t, err)
			// Output entries for accounts positions:
			// 0
			assert.NotNil(t, entries)
			assert.Equal(t, 0, len(*entries))

			filter.FromAccountFilter.AssetID = asset2.ID
			entries, _, err = ledgerStore.Query(ctx,
				filter,
				entities.DateRange{Start: &tStart, End: &tEnd},
				entities.CursorPagination{},
			)

			assert.NoError(t, err)
			// Output entries for accounts positions:
			// 6->7, 6->7, 6->7
			assert.NotNil(t, entries)
			assert.Equal(t, 3, len(*entries))

			for _, e := range *entries {
				assert.Equal(t, *e.FromAccountType, vega.AccountType_ACCOUNT_TYPE_INSURANCE)
				assert.Equal(t, *e.ToAccountType, vega.AccountType_ACCOUNT_TYPE_INSURANCE)
				if e.Quantity.Abs().String() == strconv.Itoa(80) {
					assert.Equal(t, *e.TransferType, entities.LedgerMovementTypeBondSlashing)
					assert.Equal(t, strconv.Itoa(2310), e.FromAccountBalance.Abs().String())
					assert.Equal(t, strconv.Itoa(17000), e.ToAccountBalance.Abs().String())
				}

				if e.Quantity.Abs().String() == strconv.Itoa(9) {
					assert.Equal(t, *e.TransferType, entities.LedgerMovementTypeRewardPayout)
					assert.Equal(t, strconv.Itoa(2301), e.FromAccountBalance.Abs().String())
					assert.Equal(t, strconv.Itoa(17009), e.ToAccountBalance.Abs().String())
				}

				if e.Quantity.Abs().String() == strconv.Itoa(41) {
					assert.Equal(t, *e.TransferType, entities.LedgerMovementTypeRewardPayout)
					assert.Equal(t, strconv.Itoa(2260), e.FromAccountBalance.Abs().String())
					assert.Equal(t, strconv.Itoa(17050), e.ToAccountBalance.Abs().String())
				}

				assert.Equal(t, *e.FromAccountMarketID, markets[3].ID)
				assert.Equal(t, *e.ToAccountMarketID, markets[4].ID)
			}

			filter.FromAccountFilter.PartyIDs = append(filter.FromAccountFilter.PartyIDs, parties[4].ID)

			entries, _, err = ledgerStore.Query(ctx,
				filter,
				entities.DateRange{Start: &tStart, End: &tEnd},
				entities.CursorPagination{},
			)

			assert.ErrorIs(t, err, sqlstore.ErrLedgerEntryFilterForParty)
			// Output entries for accounts positions:
			// None
			assert.Nil(t, entries)

			filter.FromAccountFilter.PartyIDs = []entities.PartyID{}
			filter.FromAccountFilter.AccountTypes = []vega.AccountType{vega.AccountType_ACCOUNT_TYPE_GENERAL}

			entries, _, err = ledgerStore.Query(ctx,
				filter,
				entities.DateRange{Start: &tStart, End: &tEnd},
				entities.CursorPagination{},
			)

			assert.ErrorIs(t, err, sqlstore.ErrLedgerEntryFilterForParty)
			// Output entries for accounts positions:
			// None
			assert.Nil(t, entries)
		})

		t.Run("by toAccount filter", func(t *testing.T) {
			// Set filters for FromAccount and AcountTo IDs
			filter := &entities.LedgerEntryFilter{
				FromAccountFilter: entities.AccountFilter{},
				ToAccountFilter: entities.AccountFilter{
					AssetID:  asset2.ID,
					PartyIDs: []entities.PartyID{parties[3].ID},
				},
			}

			entries, _, err := ledgerStore.Query(ctx,
				filter,
				entities.DateRange{Start: &tStart, End: &tEnd},
				entities.CursorPagination{},
			)

			assert.NoError(t, err)
			// Output entries for accounts positions:
			// 6->7, 6->7, 6->7
			assert.NotNil(t, entries)
			assert.Equal(t, 3, len(*entries))
			for _, e := range *entries {
				assert.Equal(t, *e.FromAccountType, vega.AccountType_ACCOUNT_TYPE_INSURANCE)
				assert.Equal(t, *e.ToAccountType, vega.AccountType_ACCOUNT_TYPE_INSURANCE)
				if e.Quantity.Abs().String() == strconv.Itoa(80) {
					assert.Equal(t, *e.TransferType, entities.LedgerMovementTypeBondSlashing)
					assert.Equal(t, strconv.Itoa(2310), e.FromAccountBalance.Abs().String())
					assert.Equal(t, strconv.Itoa(17000), e.ToAccountBalance.Abs().String())
				}

				if e.Quantity.Abs().String() == strconv.Itoa(9) {
					assert.Equal(t, *e.TransferType, entities.LedgerMovementTypeRewardPayout)
					assert.Equal(t, strconv.Itoa(2301), e.FromAccountBalance.Abs().String())
					assert.Equal(t, strconv.Itoa(17009), e.ToAccountBalance.Abs().String())
				}

				if e.Quantity.Abs().String() == strconv.Itoa(41) {
					assert.Equal(t, *e.TransferType, entities.LedgerMovementTypeRewardPayout)
					assert.Equal(t, strconv.Itoa(2260), e.FromAccountBalance.Abs().String())
					assert.Equal(t, strconv.Itoa(17050), e.ToAccountBalance.Abs().String())
				}

				assert.Equal(t, *e.FromAccountMarketID, markets[3].ID)
				assert.Equal(t, *e.ToAccountMarketID, markets[4].ID)
			}

			filter.ToAccountFilter.AccountTypes = []vega.AccountType{vega.AccountType_ACCOUNT_TYPE_GENERAL, vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY}

			entries, _, err = ledgerStore.Query(ctx,
				filter,
				entities.DateRange{Start: &tStart, End: &tEnd},
				entities.CursorPagination{},
			)

			assert.NoError(t, err)
			// Output entries for accounts positions:
			// None
			assert.NotNil(t, entries)
			assert.Equal(t, 0, len(*entries))
		})

		t.Run("by fromAccount+toAccount filters", func(t *testing.T) {
			t.Run("open", func(t *testing.T) {
				// Set filters for FromAccount and AcountTo IDs
				filter := &entities.LedgerEntryFilter{
					FromAccountFilter: entities.AccountFilter{
						AssetID: asset1.ID,
					},
					ToAccountFilter: entities.AccountFilter{
						AssetID: asset3.ID,
					},
				}

				entries, _, err := ledgerStore.Query(ctx,
					filter,
					entities.DateRange{Start: &tStart, End: &tEnd},
					entities.CursorPagination{},
				)

				assert.ErrorIs(t, err, sqlstore.ErrLedgerEntryFilterForParty)
				// Output entries for accounts positions:
				// None
				assert.Nil(t, entries)

				filter.ToAccountFilter.PartyIDs = append(filter.ToAccountFilter.PartyIDs, []entities.PartyID{parties[4].ID}...)
				entries, _, err = ledgerStore.Query(ctx,
					filter,
					entities.DateRange{Start: &tStart, End: &tEnd},
					entities.CursorPagination{},
				)

				assert.NoError(t, err)
				// Output entries for accounts positions:
				// 0->1, 2->3
				assert.NotNil(t, entries)
				assert.Equal(t, 2, len(*entries))
				for _, e := range *entries {
					assert.Equal(t, *e.FromAccountType, vega.AccountType_ACCOUNT_TYPE_GENERAL)
					assert.Equal(t, *e.ToAccountType, vega.AccountType_ACCOUNT_TYPE_GENERAL)
					assert.Equal(t, *e.TransferType, entities.LedgerMovementTypeBondSlashing)

					if e.Quantity.Abs().String() == strconv.Itoa(15) {
						assert.Equal(t, *e.FromAccountPartyID, parties[0].ID)
						assert.Equal(t, *e.ToAccountPartyID, parties[0].ID)
						assert.Equal(t, *e.FromAccountMarketID, markets[0].ID)
						assert.Equal(t, *e.ToAccountMarketID, markets[1].ID)
						assert.Equal(t, strconv.Itoa(500), e.FromAccountBalance.Abs().String())
						assert.Equal(t, strconv.Itoa(115), e.ToAccountBalance.Abs().String())
					}

					if e.Quantity.Abs().String() == strconv.Itoa(10) {
						assert.Equal(t, *e.FromAccountPartyID, parties[1].ID)
						assert.Equal(t, *e.ToAccountPartyID, parties[1].ID)
						assert.Equal(t, *e.FromAccountMarketID, markets[1].ID)
						assert.Equal(t, *e.ToAccountMarketID, markets[2].ID)
						assert.Equal(t, strconv.Itoa(170), e.FromAccountBalance.Abs().String())
						assert.Equal(t, strconv.Itoa(17890), e.ToAccountBalance.Abs().String())
					}
				}

				filter.ToAccountFilter.AccountTypes = []vega.AccountType{vega.AccountType_ACCOUNT_TYPE_GENERAL, vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY}
				entries, _, err = ledgerStore.Query(ctx,
					filter,
					entities.DateRange{Start: &tStart, End: &tEnd},
					entities.CursorPagination{},
				)

				assert.NoError(t, err)
				// Output entries for accounts positions:
				// 0->1, 2->3
				assert.NotNil(t, entries)
				assert.Equal(t, 2, len(*entries))
				for _, e := range *entries {
					assert.Equal(t, *e.FromAccountType, vega.AccountType_ACCOUNT_TYPE_GENERAL)
					assert.Equal(t, *e.ToAccountType, vega.AccountType_ACCOUNT_TYPE_GENERAL)
					assert.Equal(t, *e.TransferType, entities.LedgerMovementTypeBondSlashing)

					if e.Quantity.Abs().String() == strconv.Itoa(15) {
						assert.Equal(t, *e.FromAccountPartyID, parties[0].ID)
						assert.Equal(t, *e.ToAccountPartyID, parties[0].ID)
						assert.Equal(t, *e.FromAccountMarketID, markets[0].ID)
						assert.Equal(t, *e.ToAccountMarketID, markets[1].ID)
						assert.Equal(t, strconv.Itoa(500), e.FromAccountBalance.Abs().String())
						assert.Equal(t, strconv.Itoa(115), e.ToAccountBalance.Abs().String())
					}

					if e.Quantity.Abs().String() == strconv.Itoa(10) {
						assert.Equal(t, *e.FromAccountPartyID, parties[1].ID)
						assert.Equal(t, *e.ToAccountPartyID, parties[1].ID)
						assert.Equal(t, *e.FromAccountMarketID, markets[1].ID)
						assert.Equal(t, *e.ToAccountMarketID, markets[2].ID)
						assert.Equal(t, strconv.Itoa(170), e.FromAccountBalance.Abs().String())
						assert.Equal(t, strconv.Itoa(17890), e.ToAccountBalance.Abs().String())
					}
				}
			})

			t.Run("closed", func(t *testing.T) {
				// Set filters for FromAccount and AcountTo IDs
				filter := &entities.LedgerEntryFilter{
					FromAccountFilter: entities.AccountFilter{
						AssetID: asset2.ID,
					},
					ToAccountFilter: entities.AccountFilter{},
				}

				filter.CloseOnAccountFilters = true
				entries, _, err := ledgerStore.Query(ctx,
					filter,
					entities.DateRange{Start: &tStart, End: &tEnd},
					entities.CursorPagination{},
				)

				assert.ErrorIs(t, err, sqlstore.ErrLedgerEntryFilterForParty)
				// Output entries for accounts positions:
				// None
				assert.Nil(t, entries)

				filter.FromAccountFilter.PartyIDs = []entities.PartyID{parties[5].ID}
				entries, _, err = ledgerStore.Query(ctx,
					filter,
					entities.DateRange{Start: &tStart, End: &tEnd},
					entities.CursorPagination{},
				)

				assert.NoError(t, err)
				// Output entries for accounts positions -> should output transfers for asset2 only:
				// 10->11
				assert.NotNil(t, entries)
				assert.Equal(t, 1, len(*entries))
				for _, e := range *entries {
					assert.Equal(t, e.Quantity.Abs().String(), strconv.Itoa(40))
					assert.Equal(t, *e.FromAccountType, vega.AccountType_ACCOUNT_TYPE_GENERAL)
					assert.Equal(t, *e.ToAccountType, vega.AccountType_ACCOUNT_TYPE_GENERAL)
					assert.Equal(t, *e.TransferType, entities.LedgerMovementTypeDeposit)

					assert.Equal(t, *e.FromAccountPartyID, parties[5].ID)
					assert.Equal(t, *e.ToAccountPartyID, parties[5].ID)
					assert.Equal(t, *e.FromAccountMarketID, markets[5].ID)
					assert.Equal(t, *e.ToAccountMarketID, markets[6].ID)
					assert.Equal(t, strconv.Itoa(1500), e.FromAccountBalance.Abs().String())
					assert.Equal(t, strconv.Itoa(5680), e.ToAccountBalance.Abs().String())
				}

				// Add some grouping options
				filter.ToAccountFilter = entities.AccountFilter{AssetID: asset3.ID}
				entries, _, err = ledgerStore.Query(ctx,
					filter,
					entities.DateRange{Start: &tStart, End: &tEnd},
					entities.CursorPagination{},
				)

				assert.NoError(t, err)
				// Output entries for accounts positions:
				// None
				assert.NotNil(t, entries)
				assert.Equal(t, 0, len(*entries))

				filter.FromAccountFilter = entities.AccountFilter{AssetID: asset3.ID}
				filter.FromAccountFilter.PartyIDs = []entities.PartyID{parties[7].ID}
				filter.ToAccountFilter.AccountTypes = []vega.AccountType{vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY}
				entries, _, err = ledgerStore.Query(ctx,
					filter,
					entities.DateRange{Start: &tStart, End: &tEnd},
					entities.CursorPagination{},
				)

				assert.NoError(t, err)
				// Output entries for accounts positions:
				// 14->16
				assert.NotNil(t, entries)
				assert.Equal(t, 1, len(*entries))
				for _, e := range *entries {
					assert.Equal(t, e.Quantity.Abs().String(), strconv.Itoa(12))
					assert.Equal(t, *e.FromAccountType, vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY)
					assert.Equal(t, *e.ToAccountType, vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY)
					assert.Equal(t, *e.TransferType, entities.LedgerMovementTypeDeposit)

					assert.Equal(t, *e.FromAccountPartyID, parties[7].ID)
					assert.Equal(t, *e.ToAccountPartyID, parties[8].ID)
					assert.Equal(t, *e.FromAccountMarketID, markets[7].ID)
					assert.Equal(t, *e.ToAccountMarketID, markets[8].ID)
					assert.Equal(t, strconv.Itoa(5000), e.FromAccountBalance.Abs().String())
					assert.Equal(t, strconv.Itoa(9100), e.ToAccountBalance.Abs().String())
				}
			})
		})

		t.Run("by account filters+transferType", func(t *testing.T) {
			// open on account filters
			// Set filters for FromAccount and AcountTo IDs
			filter := &entities.LedgerEntryFilter{
				FromAccountFilter: entities.AccountFilter{
					AssetID:  asset2.ID,
					PartyIDs: []entities.PartyID{parties[8].ID},
				},
				ToAccountFilter: entities.AccountFilter{
					AssetID: asset3.ID,
				},
				TransferTypes: []entities.LedgerMovementType{
					entities.LedgerMovementTypeDeposit,
				},
			}

			entries, _, err := ledgerStore.Query(ctx,
				filter,
				entities.DateRange{Start: &tStart, End: &tEnd},
				entities.CursorPagination{},
			)

			assert.NoError(t, err)
			// Output entries for accounts positions -> should output transfers for asset3 only:
			// 14->16, 17->15, 21->15
			assert.NotNil(t, entries)
			assert.Equal(t, 3, len(*entries))
			for _, e := range *entries {
				if e.Quantity.Abs().String() == strconv.Itoa(12) {
					assert.Equal(t, *e.FromAccountPartyID, parties[7].ID)
					assert.Equal(t, *e.ToAccountPartyID, parties[8].ID)
					assert.Equal(t, *e.FromAccountMarketID, markets[7].ID)
					assert.Equal(t, *e.ToAccountMarketID, markets[8].ID)
					assert.Equal(t, strconv.Itoa(5000), e.FromAccountBalance.Abs().String())
					assert.Equal(t, strconv.Itoa(9100), e.ToAccountBalance.Abs().String())
					assert.Equal(t, *e.FromAccountType, vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY)
				}

				if e.Quantity.Abs().String() == strconv.Itoa(14) {
					assert.Equal(t, *e.FromAccountPartyID, parties[8].ID)
					assert.Equal(t, *e.ToAccountPartyID, parties[7].ID)
					assert.Equal(t, *e.FromAccountMarketID, markets[9].ID)
					assert.Equal(t, *e.ToAccountMarketID, markets[8].ID)
					assert.Equal(t, strconv.Itoa(180), e.FromAccountBalance.Abs().String())
					assert.Equal(t, strconv.Itoa(1410), e.ToAccountBalance.Abs().String())
					assert.Equal(t, *e.FromAccountType, vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY)
				}

				if e.Quantity.Abs().String() == strconv.Itoa(28) {
					assert.Equal(t, *e.FromAccountPartyID, parties[10].ID)
					assert.Equal(t, *e.ToAccountPartyID, parties[7].ID)

					assert.Equal(t, *e.FromAccountMarketID, markets[9].ID)
					assert.Equal(t, *e.ToAccountMarketID, markets[8].ID)
					assert.Equal(t, strconv.Itoa(2180), e.FromAccountBalance.Abs().String())
					assert.Equal(t, strconv.Itoa(1438), e.ToAccountBalance.Abs().String())
					assert.Equal(t, *e.FromAccountType, vega.AccountType_ACCOUNT_TYPE_INSURANCE)
				}

				assert.Equal(t, *e.ToAccountType, vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY)
				assert.Equal(t, *e.TransferType, entities.LedgerMovementTypeDeposit)
			}

			// closed on account filters
			filter.CloseOnAccountFilters = true
			entries, _, err = ledgerStore.Query(ctx,
				filter,
				entities.DateRange{Start: &tStart, End: &tEnd},
				entities.CursorPagination{},
			)

			assert.NoError(t, err)
			// Output entries for accounts positions:
			// None
			assert.NotNil(t, entries)
			assert.Equal(t, 0, len(*entries))

			filter.ToAccountFilter = entities.AccountFilter{
				AssetID:      asset3.ID,
				AccountTypes: []vega.AccountType{vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY},
			}

			entries, _, err = ledgerStore.Query(ctx,
				filter,
				entities.DateRange{Start: &tStart, End: &tEnd},
				entities.CursorPagination{},
			)

			assert.NoError(t, err)
			// Output entries for accounts positions:
			// 0
			assert.NotNil(t, entries)
			assert.Equal(t, 0, len(*entries))
		})

		t.Run("test open/closing with different account and transfer types", func(t *testing.T) {
			filter := &entities.LedgerEntryFilter{
				FromAccountFilter: entities.AccountFilter{
					PartyIDs: []entities.PartyID{parties[2].ID},
				},
				ToAccountFilter: entities.AccountFilter{
					PartyIDs: []entities.PartyID{parties[5].ID},
				},
				TransferTypes: []entities.LedgerMovementType{
					entities.LedgerMovementTypeRewardPayout,
				},
			}

			entries, _, err := ledgerStore.Query(ctx,
				filter,
				entities.DateRange{Start: &tStart, End: &tEnd},
				entities.CursorPagination{},
			)

			assert.NoError(t, err)
			// 4->5, 5->10, 5->11, 4->11
			assert.NotNil(t, entries)
			assert.Equal(t, 3, len(*entries))
			for _, e := range *entries {
				if e.Quantity.Abs().String() == strconv.Itoa(3) {
					assert.Equal(t, *e.FromAccountPartyID, parties[2].ID)
					assert.Equal(t, *e.ToAccountPartyID, parties[5].ID)
					assert.Equal(t, *e.FromAccountMarketID, markets[3].ID)
					assert.Equal(t, *e.ToAccountMarketID, markets[6].ID)
					assert.Equal(t, *e.FromAccountType, vega.AccountType_ACCOUNT_TYPE_INSURANCE)
					assert.Equal(t, *e.ToAccountType, vega.AccountType_ACCOUNT_TYPE_GENERAL)
					assert.Equal(t, *e.TransferType, entities.LedgerMovementTypeRewardPayout)
				}

				if e.Quantity.Abs().String() == strconv.Itoa(5) {
					assert.Equal(t, *e.FromAccountPartyID, parties[2].ID)
					assert.Equal(t, *e.ToAccountPartyID, parties[5].ID)
					assert.Equal(t, *e.FromAccountMarketID, markets[3].ID)
					assert.Equal(t, *e.ToAccountMarketID, markets[5].ID)
					assert.Equal(t, *e.FromAccountType, vega.AccountType_ACCOUNT_TYPE_INSURANCE)
					assert.Equal(t, *e.ToAccountType, vega.AccountType_ACCOUNT_TYPE_GENERAL)
					assert.Equal(t, *e.TransferType, entities.LedgerMovementTypeRewardPayout)
				}

				if e.Quantity.Abs().String() == strconv.Itoa(72) {
					assert.Equal(t, *e.FromAccountPartyID, parties[2].ID)
					assert.Equal(t, *e.ToAccountPartyID, parties[5].ID)

					assert.Equal(t, *e.FromAccountMarketID, markets[2].ID)
					assert.Equal(t, *e.ToAccountMarketID, markets[6].ID)
					assert.Equal(t, strconv.Itoa(2188), e.FromAccountBalance.Abs().String())
					assert.Equal(t, strconv.Itoa(17122), e.ToAccountBalance.Abs().String())
					assert.Equal(t, *e.FromAccountType, vega.AccountType_ACCOUNT_TYPE_INSURANCE)
					assert.Equal(t, *e.ToAccountType, vega.AccountType_ACCOUNT_TYPE_GENERAL)
					assert.Equal(t, *e.TransferType, entities.LedgerMovementTypeRewardPayout)
				}
			}

			filter.CloseOnAccountFilters = true
			entries, _, err = ledgerStore.Query(ctx,
				filter,
				entities.DateRange{Start: &tStart, End: &tEnd},
				entities.CursorPagination{},
			)

			assert.NoError(t, err)
			// Output entries for accounts positions -> should output transfers for asset3 only:
			// 5->10, 5->11, 4->11
			assert.NotNil(t, entries)
			assert.Equal(t, 3, len(*entries))
			for _, e := range *entries {
				if e.Quantity.Abs().String() == strconv.Itoa(3) {
					assert.Equal(t, *e.FromAccountPartyID, parties[2].ID)
					assert.Equal(t, *e.ToAccountPartyID, parties[5].ID)
					assert.Equal(t, *e.FromAccountMarketID, markets[3].ID)
					assert.Equal(t, *e.ToAccountMarketID, markets[6].ID)
					assert.Equal(t, *e.FromAccountType, vega.AccountType_ACCOUNT_TYPE_INSURANCE)
					assert.Equal(t, *e.ToAccountType, vega.AccountType_ACCOUNT_TYPE_GENERAL)
					assert.Equal(t, *e.TransferType, entities.LedgerMovementTypeRewardPayout)
				}

				if e.Quantity.Abs().String() == strconv.Itoa(5) {
					assert.Equal(t, *e.FromAccountPartyID, parties[2].ID)
					assert.Equal(t, *e.ToAccountPartyID, parties[5].ID)
					assert.Equal(t, *e.FromAccountMarketID, markets[3].ID)
					assert.Equal(t, *e.ToAccountMarketID, markets[5].ID)
					assert.Equal(t, *e.FromAccountType, vega.AccountType_ACCOUNT_TYPE_INSURANCE)
					assert.Equal(t, *e.ToAccountType, vega.AccountType_ACCOUNT_TYPE_GENERAL)
					assert.Equal(t, *e.TransferType, entities.LedgerMovementTypeRewardPayout)
				}

				if e.Quantity.Abs().String() == strconv.Itoa(72) {
					assert.Equal(t, *e.FromAccountPartyID, parties[2].ID)
					assert.Equal(t, *e.ToAccountPartyID, parties[5].ID)

					assert.Equal(t, *e.FromAccountMarketID, markets[2].ID)
					assert.Equal(t, *e.ToAccountMarketID, markets[6].ID)
					assert.Equal(t, strconv.Itoa(2188), e.FromAccountBalance.Abs().String())
					assert.Equal(t, strconv.Itoa(17122), e.ToAccountBalance.Abs().String())
					assert.Equal(t, *e.FromAccountType, vega.AccountType_ACCOUNT_TYPE_INSURANCE)
					assert.Equal(t, *e.ToAccountType, vega.AccountType_ACCOUNT_TYPE_GENERAL)
					assert.Equal(t, *e.TransferType, entities.LedgerMovementTypeRewardPayout)
				}
				assert.Equal(t, *e.TransferType, entities.LedgerMovementTypeRewardPayout)
			}

			filter = &entities.LedgerEntryFilter{
				FromAccountFilter: entities.AccountFilter{
					PartyIDs: []entities.PartyID{parties[2].ID},
				},
				ToAccountFilter: entities.AccountFilter{
					AccountTypes: []types.AccountType{vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY},
				},
				TransferTypes: []entities.LedgerMovementType{
					entities.LedgerMovementTypeRewardPayout,
				},
			}
			filter.FromAccountFilter.AccountTypes = append(filter.FromAccountFilter.AccountTypes, vega.AccountType_ACCOUNT_TYPE_INSURANCE)
			entries, _, err = ledgerStore.Query(ctx,
				filter,
				entities.DateRange{Start: &tStart, End: &tEnd},
				entities.CursorPagination{},
			)

			assert.NoError(t, err)
			// 4->5, 5->10, 5->11, 4->11
			assert.NotNil(t, entries)
			assert.Equal(t, 3, len(*entries))
			for _, e := range *entries {
				if e.Quantity.Abs().String() == strconv.Itoa(3) {
					assert.Equal(t, *e.FromAccountPartyID, parties[2].ID)
					assert.Equal(t, *e.ToAccountPartyID, parties[5].ID)
					assert.Equal(t, *e.FromAccountMarketID, markets[3].ID)
					assert.Equal(t, *e.ToAccountMarketID, markets[6].ID)
					assert.Equal(t, *e.FromAccountType, vega.AccountType_ACCOUNT_TYPE_INSURANCE)
					assert.Equal(t, *e.ToAccountType, vega.AccountType_ACCOUNT_TYPE_GENERAL)
					assert.Equal(t, *e.TransferType, entities.LedgerMovementTypeRewardPayout)
				}

				if e.Quantity.Abs().String() == strconv.Itoa(5) {
					assert.Equal(t, *e.FromAccountPartyID, parties[2].ID)
					assert.Equal(t, *e.ToAccountPartyID, parties[5].ID)
					assert.Equal(t, *e.FromAccountMarketID, markets[3].ID)
					assert.Equal(t, *e.ToAccountMarketID, markets[5].ID)
					assert.Equal(t, *e.FromAccountType, vega.AccountType_ACCOUNT_TYPE_INSURANCE)
					assert.Equal(t, *e.ToAccountType, vega.AccountType_ACCOUNT_TYPE_GENERAL)
					assert.Equal(t, *e.TransferType, entities.LedgerMovementTypeRewardPayout)
				}

				if e.Quantity.Abs().String() == strconv.Itoa(72) {
					assert.Equal(t, *e.FromAccountPartyID, parties[2].ID)
					assert.Equal(t, *e.ToAccountPartyID, parties[5].ID)

					assert.Equal(t, *e.FromAccountMarketID, markets[2].ID)
					assert.Equal(t, *e.ToAccountMarketID, markets[6].ID)
					assert.Equal(t, strconv.Itoa(2188), e.FromAccountBalance.Abs().String())
					assert.Equal(t, strconv.Itoa(17122), e.ToAccountBalance.Abs().String())
					assert.Equal(t, *e.FromAccountType, vega.AccountType_ACCOUNT_TYPE_INSURANCE)
					assert.Equal(t, *e.ToAccountType, vega.AccountType_ACCOUNT_TYPE_GENERAL)
					assert.Equal(t, *e.TransferType, entities.LedgerMovementTypeRewardPayout)
				}
			}

			filter = &entities.LedgerEntryFilter{
				FromAccountFilter: entities.AccountFilter{
					PartyIDs: []entities.PartyID{parties[2].ID},
				},
				ToAccountFilter: entities.AccountFilter{
					AccountTypes: []types.AccountType{vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY},
				},
				CloseOnAccountFilters: true,
				TransferTypes: []entities.LedgerMovementType{
					entities.LedgerMovementTypeDeposit,
				},
			}

			entries, _, err = ledgerStore.Query(ctx,
				filter,
				entities.DateRange{Start: &tStart, End: &tEnd},
				entities.CursorPagination{},
			)

			assert.NoError(t, err)
			assert.NotNil(t, entries)
			assert.Equal(t, 0, len(*entries))

			filter = &entities.LedgerEntryFilter{
				FromAccountFilter: entities.AccountFilter{
					PartyIDs: []entities.PartyID{parties[2].ID},
				},
				ToAccountFilter:       entities.AccountFilter{},
				CloseOnAccountFilters: true,
				TransferTypes: []entities.LedgerMovementType{
					entities.LedgerMovementTypeBondSlashing,
					entities.LedgerMovementTypeDeposit,
					entities.LedgerMovementTypeRewardPayout,
				},
			}

			entries, _, err = ledgerStore.Query(ctx,
				filter,
				entities.DateRange{Start: &tStart, End: &tEnd},
				entities.CursorPagination{},
			)

			assert.NoError(t, err)
			// List transfers:
			// accounts 5->11 - 3 - ACCOUNT_TYPE_INSURANCE ACCOUNT_TYPE_GENERAL
			// accounts 5->10 - 5 - ACCOUNT_TYPE_INSURANCE ACCOUNT_TYPE_GENERAL
			// accounts 4->11 - 72 - ACCOUNT_TYPE_INSURANCE ACCOUNT_TYPE_GENERAL
			// accounts 4->5 - 25 - ACCOUNT_TYPE_INSURANCE ACCOUNT_TYPE_INSURANCE
			assert.NotNil(t, entries)
			assert.Equal(t, 4, len(*entries))
			for _, e := range *entries {
				if e.Quantity.Abs().String() == strconv.Itoa(3) {
					assert.Equal(t, *e.FromAccountPartyID, parties[2].ID)
					assert.Equal(t, *e.ToAccountPartyID, parties[5].ID)
					assert.Equal(t, *e.FromAccountMarketID, markets[3].ID)
					assert.Equal(t, *e.ToAccountMarketID, markets[6].ID)
					assert.Equal(t, strconv.Itoa(2587), e.FromAccountBalance.Abs().String())
					assert.Equal(t, strconv.Itoa(5683), e.ToAccountBalance.Abs().String())
					assert.Equal(t, *e.FromAccountType, vega.AccountType_ACCOUNT_TYPE_INSURANCE)
					assert.Equal(t, *e.ToAccountType, vega.AccountType_ACCOUNT_TYPE_GENERAL)
					assert.Equal(t, *e.TransferType, entities.LedgerMovementTypeRewardPayout)
				}

				if e.Quantity.Abs().String() == strconv.Itoa(5) {
					assert.Equal(t, *e.FromAccountPartyID, parties[2].ID)
					assert.Equal(t, *e.ToAccountPartyID, parties[5].ID)
					assert.Equal(t, *e.FromAccountMarketID, markets[3].ID)
					assert.Equal(t, *e.ToAccountMarketID, markets[5].ID)
					assert.Equal(t, strconv.Itoa(2582), e.FromAccountBalance.Abs().String())
					assert.Equal(t, strconv.Itoa(1510), e.ToAccountBalance.Abs().String())
					assert.Equal(t, *e.FromAccountType, vega.AccountType_ACCOUNT_TYPE_INSURANCE)
					assert.Equal(t, *e.ToAccountType, vega.AccountType_ACCOUNT_TYPE_GENERAL)
					assert.Equal(t, *e.TransferType, entities.LedgerMovementTypeRewardPayout)
				}

				if e.Quantity.Abs().String() == strconv.Itoa(72) {
					assert.Equal(t, *e.FromAccountPartyID, parties[2].ID)
					assert.Equal(t, *e.ToAccountPartyID, parties[5].ID)

					assert.Equal(t, *e.FromAccountMarketID, markets[2].ID)
					assert.Equal(t, *e.ToAccountMarketID, markets[6].ID)
					assert.Equal(t, strconv.Itoa(2188), e.FromAccountBalance.Abs().String())
					assert.Equal(t, strconv.Itoa(17122), e.ToAccountBalance.Abs().String())
					assert.Equal(t, *e.FromAccountType, vega.AccountType_ACCOUNT_TYPE_INSURANCE)
					assert.Equal(t, *e.ToAccountType, vega.AccountType_ACCOUNT_TYPE_GENERAL)
					assert.Equal(t, *e.TransferType, entities.LedgerMovementTypeRewardPayout)
				}

				if e.Quantity.Abs().String() == strconv.Itoa(25) {
					assert.Equal(t, *e.FromAccountPartyID, parties[2].ID)
					assert.Equal(t, *e.ToAccountPartyID, parties[2].ID)

					assert.Equal(t, *e.FromAccountMarketID, markets[2].ID)
					assert.Equal(t, *e.ToAccountMarketID, markets[3].ID)
					assert.Equal(t, strconv.Itoa(1700), e.FromAccountBalance.Abs().String())
					assert.Equal(t, strconv.Itoa(2590), e.ToAccountBalance.Abs().String())
					assert.Equal(t, *e.FromAccountType, vega.AccountType_ACCOUNT_TYPE_INSURANCE)
					assert.Equal(t, *e.ToAccountType, vega.AccountType_ACCOUNT_TYPE_INSURANCE)
					assert.Equal(t, *e.TransferType, entities.LedgerMovementTypeBondSlashing)
				}
			}

			filter = &entities.LedgerEntryFilter{
				FromAccountFilter: entities.AccountFilter{
					PartyIDs: []entities.PartyID{parties[2].ID},
				},
				ToAccountFilter: entities.AccountFilter{
					AccountTypes: []types.AccountType{vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY},
				},
				CloseOnAccountFilters: true,
				TransferTypes: []entities.LedgerMovementType{
					entities.LedgerMovementTypeBondSlashing,
					entities.LedgerMovementTypeDeposit,
					entities.LedgerMovementTypeRewardPayout,
				},
			}

			entries, _, err = ledgerStore.Query(ctx,
				filter,
				entities.DateRange{Start: &tStart, End: &tEnd},
				entities.CursorPagination{},
			)

			assert.NoError(t, err)
			// List transfers:
			// 0
			assert.NotNil(t, entries)
			assert.Equal(t, 0, len(*entries))
		})

		t.Run("test with same account and transfer types given multiple times", func(t *testing.T) {
			filter := &entities.LedgerEntryFilter{
				FromAccountFilter: entities.AccountFilter{
					PartyIDs:     []entities.PartyID{parties[2].ID},
					AccountTypes: []types.AccountType{vega.AccountType_ACCOUNT_TYPE_INSURANCE, vega.AccountType_ACCOUNT_TYPE_INSURANCE, vega.AccountType_ACCOUNT_TYPE_INSURANCE},
				},
				ToAccountFilter: entities.AccountFilter{
					AccountTypes: []types.AccountType{vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY, vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY},
				},
				TransferTypes: []entities.LedgerMovementType{
					entities.LedgerMovementTypeRewardPayout, entities.LedgerMovementTypeRewardPayout, entities.LedgerMovementTypeRewardPayout,
				},
			}
			entries, _, err := ledgerStore.Query(ctx,
				filter,
				entities.DateRange{Start: &tStart, End: &tEnd},
				entities.CursorPagination{},
			)

			assert.NoError(t, err)
			// 4->5, 5->10, 5->11, 4->11
			assert.NotNil(t, entries)
			assert.Equal(t, 3, len(*entries))
			for _, e := range *entries {
				if e.Quantity.Abs().String() == strconv.Itoa(3) {
					assert.Equal(t, *e.FromAccountPartyID, parties[2].ID)
					assert.Equal(t, *e.ToAccountPartyID, parties[5].ID)
					assert.Equal(t, *e.FromAccountMarketID, markets[3].ID)
					assert.Equal(t, *e.ToAccountMarketID, markets[6].ID)
					assert.Equal(t, *e.FromAccountType, vega.AccountType_ACCOUNT_TYPE_INSURANCE)
					assert.Equal(t, *e.ToAccountType, vega.AccountType_ACCOUNT_TYPE_GENERAL)
					assert.Equal(t, *e.TransferType, entities.LedgerMovementTypeRewardPayout)
				}

				if e.Quantity.Abs().String() == strconv.Itoa(5) {
					assert.Equal(t, *e.FromAccountPartyID, parties[2].ID)
					assert.Equal(t, *e.ToAccountPartyID, parties[5].ID)
					assert.Equal(t, *e.FromAccountMarketID, markets[3].ID)
					assert.Equal(t, *e.ToAccountMarketID, markets[5].ID)
					assert.Equal(t, *e.FromAccountType, vega.AccountType_ACCOUNT_TYPE_INSURANCE)
					assert.Equal(t, *e.ToAccountType, vega.AccountType_ACCOUNT_TYPE_GENERAL)
					assert.Equal(t, *e.TransferType, entities.LedgerMovementTypeRewardPayout)
				}

				if e.Quantity.Abs().String() == strconv.Itoa(72) {
					assert.Equal(t, *e.FromAccountPartyID, parties[2].ID)
					assert.Equal(t, *e.ToAccountPartyID, parties[5].ID)

					assert.Equal(t, *e.FromAccountMarketID, markets[2].ID)
					assert.Equal(t, *e.ToAccountMarketID, markets[6].ID)
					assert.Equal(t, strconv.Itoa(2188), e.FromAccountBalance.Abs().String())
					assert.Equal(t, strconv.Itoa(17122), e.ToAccountBalance.Abs().String())
					assert.Equal(t, *e.FromAccountType, vega.AccountType_ACCOUNT_TYPE_INSURANCE)
					assert.Equal(t, *e.ToAccountType, vega.AccountType_ACCOUNT_TYPE_GENERAL)
					assert.Equal(t, *e.TransferType, entities.LedgerMovementTypeRewardPayout)
				}
			}

			filter = &entities.LedgerEntryFilter{
				FromAccountFilter: entities.AccountFilter{
					PartyIDs: []entities.PartyID{parties[2].ID},
					AccountTypes: []types.AccountType{
						vega.AccountType_ACCOUNT_TYPE_INSURANCE, vega.AccountType_ACCOUNT_TYPE_INSURANCE, vega.AccountType_ACCOUNT_TYPE_INSURANCE,
						vega.AccountType_ACCOUNT_TYPE_GENERAL, vega.AccountType_ACCOUNT_TYPE_GENERAL,
					},
				},
				ToAccountFilter: entities.AccountFilter{
					AccountTypes: []types.AccountType{vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY, vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY},
				},
				TransferTypes: []entities.LedgerMovementType{
					entities.LedgerMovementTypeRewardPayout, entities.LedgerMovementTypeRewardPayout, entities.LedgerMovementTypeRewardPayout,
					entities.LedgerMovementTypeBondSlashing, entities.LedgerMovementTypeBondSlashing,
				},
			}
			entries, _, err = ledgerStore.Query(ctx,
				filter,
				entities.DateRange{Start: &tStart, End: &tEnd},
				entities.CursorPagination{},
			)

			assert.NoError(t, err)
			// 4->5, 5->10, 5->11, 4->11
			assert.NotNil(t, entries)
			assert.Equal(t, 4, len(*entries))
			for _, e := range *entries {
				if e.Quantity.Abs().String() == strconv.Itoa(3) {
					assert.Equal(t, *e.FromAccountPartyID, parties[2].ID)
					assert.Equal(t, *e.ToAccountPartyID, parties[5].ID)
					assert.Equal(t, *e.FromAccountMarketID, markets[3].ID)
					assert.Equal(t, *e.ToAccountMarketID, markets[6].ID)
					assert.Equal(t, *e.FromAccountType, vega.AccountType_ACCOUNT_TYPE_INSURANCE)
					assert.Equal(t, *e.ToAccountType, vega.AccountType_ACCOUNT_TYPE_GENERAL)
					assert.Equal(t, *e.TransferType, entities.LedgerMovementTypeRewardPayout)
				}

				if e.Quantity.Abs().String() == strconv.Itoa(5) {
					assert.Equal(t, *e.FromAccountPartyID, parties[2].ID)
					assert.Equal(t, *e.ToAccountPartyID, parties[5].ID)
					assert.Equal(t, *e.FromAccountMarketID, markets[3].ID)
					assert.Equal(t, *e.ToAccountMarketID, markets[5].ID)
					assert.Equal(t, *e.FromAccountType, vega.AccountType_ACCOUNT_TYPE_INSURANCE)
					assert.Equal(t, *e.ToAccountType, vega.AccountType_ACCOUNT_TYPE_GENERAL)
					assert.Equal(t, *e.TransferType, entities.LedgerMovementTypeRewardPayout)
				}

				if e.Quantity.Abs().String() == strconv.Itoa(25) {
					assert.Equal(t, *e.FromAccountPartyID, parties[2].ID)
					assert.Equal(t, *e.ToAccountPartyID, parties[2].ID)

					assert.Equal(t, *e.FromAccountMarketID, markets[2].ID)
					assert.Equal(t, *e.ToAccountMarketID, markets[3].ID)
					assert.Equal(t, strconv.Itoa(1700), e.FromAccountBalance.Abs().String())
					assert.Equal(t, strconv.Itoa(2590), e.ToAccountBalance.Abs().String())
					assert.Equal(t, *e.FromAccountType, vega.AccountType_ACCOUNT_TYPE_INSURANCE)
					assert.Equal(t, *e.ToAccountType, vega.AccountType_ACCOUNT_TYPE_INSURANCE)
					assert.Equal(t, *e.TransferType, entities.LedgerMovementTypeBondSlashing)
				}

				if e.Quantity.Abs().String() == strconv.Itoa(72) {
					assert.Equal(t, *e.FromAccountPartyID, parties[2].ID)
					assert.Equal(t, *e.ToAccountPartyID, parties[5].ID)

					assert.Equal(t, *e.FromAccountMarketID, markets[2].ID)
					assert.Equal(t, *e.ToAccountMarketID, markets[6].ID)
					assert.Equal(t, strconv.Itoa(2188), e.FromAccountBalance.Abs().String())
					assert.Equal(t, strconv.Itoa(17122), e.ToAccountBalance.Abs().String())
					assert.Equal(t, *e.FromAccountType, vega.AccountType_ACCOUNT_TYPE_INSURANCE)
					assert.Equal(t, *e.ToAccountType, vega.AccountType_ACCOUNT_TYPE_GENERAL)
					assert.Equal(t, *e.TransferType, entities.LedgerMovementTypeRewardPayout)
				}
			}
		})
	})
}
