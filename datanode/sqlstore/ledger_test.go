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

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/datanode/sqlstore/helpers"
	"code.vegaprotocol.io/vega/protos/vega"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func addTestLedgerEntry(t *testing.T, ledger *sqlstore.Ledger,
	fromAccount entities.Account,
	toAccount entities.Account,
	block entities.Block,
	quantity int64,
	transferType entities.LedgerMovementType,
	fromAccountBalance, toAccountBalance int64,
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
	}

	err := ledger.Add(ledgerEntry)
	require.NoError(t, err)
	return ledgerEntry
}

func TestLedger(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

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
		accounts[10] => asset2, parties[5], markets[5], vega.AccountType_ACCOUNT_TYPE_INSURANCE
		accounts[11] => asset2, parties[5], markets[6], vega.AccountType_ACCOUNT_TYPE_INSURANCE

		accounts[12] => asset3, parties[6], markets[6], vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY
		accounts[13] => asset3, parties[6], markets[7], vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY
		accounts[14] => asset3, parties[7], markets[7], vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY
		accounts[15] => asset3, parties[7], markets[8], vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY
		accounts[16] => asset3, parties[8], markets[8], vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY
		accounts[17] => asset3, parties[8], markets[9], vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY
		accounts[18] => asset3, parties[9], markets[9], vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY
		accounts[19] => asset3, parties[9], markets[10], vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY
		accounts[20] => asset3, parties[10], markets[10], vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY
		accounts[21] => asset3, parties[10], markets[11], vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY
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
		if i >= 2 && i < 6 {
			fromAccount := helpers.AddTestAccountWithMarketAndType(t, ctx, accountStore, parties[i], asset2, blocks[i], markets[i].ID, vega.AccountType_ACCOUNT_TYPE_INSURANCE)
			toAccount := helpers.AddTestAccountWithMarketAndType(t, ctx, accountStore, parties[i], asset2, blocks[i], markets[mt].ID, vega.AccountType_ACCOUNT_TYPE_INSURANCE)
			accounts = append(accounts, fromAccount)
			accounts = append(accounts, toAccount)
			continue
		}

		// accounts 12, 13, 14, 15, 16, 17, 18, 19, 20, 21
		fromAccount := helpers.AddTestAccountWithMarketAndType(t, ctx, accountStore, parties[i], asset3, blocks[i], markets[i].ID, vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY)
		toAccount := helpers.AddTestAccountWithMarketAndType(t, ctx, accountStore, parties[i], asset3, blocks[i], markets[mt].ID, vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY)
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
		accounts[10]->accounts[11] => asset2, parties[5], markets[5-6], vega.AccountType_ACCOUNT_TYPE_INSURANCE

		accounts[5]->accounts[10] => asset2, parties[2-5], markets[3-5], vega.AccountType_ACCOUNT_TYPE_INSURANCE

		accounts[5]->accounts[11] => asset2, parties[2-5], markets[3-6], vega.AccountType_ACCOUNT_TYPE_INSURANCE

		Asset3:
		accounts[14]->accounts[16] => asset3, parties[7-8], markets[7-8], vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY

		accounts[17]->accounts[15] => asset3, parties[8-7], markets[9-8], vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY

		accounts[21]->accounts[15] => asset3, parties[10-7], markets[9-8], vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY
	*/
	var ledgerEntries []entities.LedgerEntry
	ledgerEntries = append(ledgerEntries, addTestLedgerEntry(t, ledgerStore, accounts[0], accounts[1], blocks[1], int64(15), entities.LedgerMovementTypeBondSlashing, int64(500), int64(115)))
	ledgerEntries = append(ledgerEntries, addTestLedgerEntry(t, ledgerStore, accounts[2], accounts[3], blocks[2], int64(10), entities.LedgerMovementTypeBondSlashing, int64(170), int64(17890)))
	ledgerEntries = append(ledgerEntries, addTestLedgerEntry(t, ledgerStore, accounts[4], accounts[5], blocks[3], int64(25), entities.LedgerMovementTypeBondSlashing, int64(1700), int64(2590)))
	ledgerEntries = append(ledgerEntries, addTestLedgerEntry(t, ledgerStore, accounts[6], accounts[7], blocks[4], int64(80), entities.LedgerMovementTypeBondSlashing, int64(2310), int64(17000)))
	ledgerEntries = append(ledgerEntries, addTestLedgerEntry(t, ledgerStore, accounts[8], accounts[9], blocks[5], int64(1), entities.LedgerMovementTypeDeposit, int64(120), int64(900)))
	ledgerEntries = append(ledgerEntries, addTestLedgerEntry(t, ledgerStore, accounts[10], accounts[11], blocks[6], int64(40), entities.LedgerMovementTypeDeposit, int64(1500), int64(5680)))
	ledgerEntries = append(ledgerEntries, addTestLedgerEntry(t, ledgerStore, accounts[14], accounts[16], blocks[7], int64(12), entities.LedgerMovementTypeDeposit, int64(5000), int64(9100)))
	ledgerEntries = append(ledgerEntries, addTestLedgerEntry(t, ledgerStore, accounts[17], accounts[15], blocks[8], int64(14), entities.LedgerMovementTypeDeposit, int64(180), int64(1410)))
	ledgerEntries = append(ledgerEntries, addTestLedgerEntry(t, ledgerStore, accounts[21], accounts[15], blocks[9], int64(28), entities.LedgerMovementTypeDeposit, int64(2180), int64(1438)))
	ledgerEntries = append(ledgerEntries, addTestLedgerEntry(t, ledgerStore, accounts[5], accounts[11], blocks[10], int64(3), entities.LedgerMovementTypeRewardPayout, int64(2587), int64(5683)))
	ledgerEntries = append(ledgerEntries, addTestLedgerEntry(t, ledgerStore, accounts[5], accounts[10], blocks[11], int64(5), entities.LedgerMovementTypeRewardPayout, int64(2582), int64(1510)))
	ledgerEntries = append(ledgerEntries, addTestLedgerEntry(t, ledgerStore, accounts[6], accounts[7], blocks[12], int64(9), entities.LedgerMovementTypeRewardPayout, int64(2301), int64(17009)))
	_ = append(ledgerEntries, addTestLedgerEntry(t, ledgerStore, accounts[6], accounts[7], blocks[13], int64(41), entities.LedgerMovementTypeRewardPayout, int64(2260), int64(17050)))

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
				assert.Equal(t, 1, len(*entries))
				for _, e := range *entries {
					assert.Equal(t, e.Quantity.Abs().String(), strconv.Itoa(40))
					assert.Equal(t, *e.FromAccountType, vega.AccountType_ACCOUNT_TYPE_INSURANCE)
					assert.Equal(t, *e.ToAccountType, vega.AccountType_ACCOUNT_TYPE_INSURANCE)
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
			assert.Equal(t, 3, len(*entries))
			for _, e := range *entries {
				if e.Quantity.Abs().String() == strconv.Itoa(12) {
					assert.Equal(t, *e.FromAccountPartyID, parties[7].ID)
					assert.Equal(t, *e.ToAccountPartyID, parties[8].ID)
					assert.Equal(t, *e.FromAccountMarketID, markets[7].ID)
					assert.Equal(t, *e.ToAccountMarketID, markets[8].ID)
					assert.Equal(t, strconv.Itoa(5000), e.FromAccountBalance.Abs().String())
					assert.Equal(t, strconv.Itoa(9100), e.ToAccountBalance.Abs().String())
				}

				if e.Quantity.Abs().String() == strconv.Itoa(14) {
					assert.Equal(t, *e.FromAccountPartyID, parties[8].ID)
					assert.Equal(t, *e.ToAccountPartyID, parties[7].ID)
					assert.Equal(t, *e.FromAccountMarketID, markets[9].ID)
					assert.Equal(t, *e.ToAccountMarketID, markets[8].ID)
					assert.Equal(t, strconv.Itoa(180), e.FromAccountBalance.Abs().String())
					assert.Equal(t, strconv.Itoa(1410), e.ToAccountBalance.Abs().String())
				}

				if e.Quantity.Abs().String() == strconv.Itoa(28) {
					assert.Equal(t, *e.FromAccountPartyID, parties[10].ID)
					assert.Equal(t, *e.ToAccountPartyID, parties[7].ID)

					assert.Equal(t, *e.FromAccountMarketID, markets[9].ID)
					assert.Equal(t, *e.ToAccountMarketID, markets[8].ID)
					assert.Equal(t, strconv.Itoa(2180), e.FromAccountBalance.Abs().String())
					assert.Equal(t, strconv.Itoa(1438), e.ToAccountBalance.Abs().String())
				}

				assert.Equal(t, *e.FromAccountType, vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY)
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
			assert.Equal(t, 0, len(*entries))
		})
	})
}
