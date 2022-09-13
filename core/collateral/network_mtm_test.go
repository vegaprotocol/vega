// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package collateral_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMTMWithNetwork(t *testing.T) {
	t.Run("test MTM with the network on the loss side (insurance pool has enough balance)", testMTMWithNetworkNoLossSoc)
	t.Run("test MTM with network on the loss side (loss socialization)", testMTMWithNetworkLossSoc)
}

type PartyEvt interface {
	Party() string
}

func testMTMWithNetworkNoLossSoc(t *testing.T) {
	party := "test-party"
	moneyParty := "money-party"
	price := num.NewUint(1000)

	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.ID, num.Sum(price, price))
	assert.Nil(t, err)

	// create party accounts
	eng.broker.EXPECT().Send(gomock.Any()).Times(8)
	gID, _ := eng.Engine.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	mID, err := eng.Engine.CreatePartyMarginAccount(context.Background(), party, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	assert.NotEmpty(t, mID)
	assert.NotEmpty(t, gID)

	// create + add balance
	_, _ = eng.Engine.CreatePartyGeneralAccount(context.Background(), moneyParty, testMarketAsset)
	marginMoneyParty, err := eng.Engine.CreatePartyMarginAccount(context.Background(), moneyParty, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	err = eng.Engine.UpdateBalance(context.Background(), marginMoneyParty, num.UintZero().Mul(num.NewUint(5), price))
	assert.Nil(t, err)

	pos := []*types.TransferInstruction{
		{
			Owner: types.NetworkParty,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  testMarketAsset,
			},
			Type: types.TransferInstructionTypeMTMLoss,
		},
		{
			Owner: moneyParty,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  testMarketAsset,
			},
			Type: types.TransferInstructionTypeMTMLoss,
		},
		{
			Owner: party,
			Amount: &types.FinancialAmount{
				Amount: num.Sum(price, price), // one winning party
				Asset:  testMarketAsset,
			},
			Type: types.TransferInstructionTypeMTMWin,
		},
	}

	eng.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes().Do(func(evts []events.Event) {
		for _, e := range evts {
			if lse, ok := e.(PartyEvt); ok {
				require.NotEqual(t, types.NetworkParty, lse.Party())
			}
		}
	})
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes().Do(func(evt events.Event) {
		ae, ok := evt.(accEvt)
		assert.True(t, ok)
		acc := ae.Account()
		// we should never receive an event where an account is owned by the network
		require.False(t, acc.Owner == types.NetworkParty)
		if acc.Owner == party && acc.Type == types.AccountTypeGeneral {
			assert.Equal(t, acc.Balance, int64(833))
		}
		if acc.Owner == moneyParty && acc.Type == types.AccountTypeGeneral {
			assert.Equal(t, acc.Balance, int64(1666))
		}
	})
	transfers := eng.getTestMTMTransfer(pos)
	evts, raw, err := eng.MarkToMarket(context.Background(), testMarketID, transfers, "BTC")
	assert.NoError(t, err)
	assert.Equal(t, 3, len(raw))
	assert.NotEmpty(t, evts)
	for _, e := range evts {
		require.NotEqual(t, types.NetworkParty, e.Party()) // there should be no margin events for the network
	}
	found := false // we should see a transfer from insurance to settlement
	for _, r := range raw {
		for _, tr := range r.TransferInstructions {
			if tr.FromAccount == insurancePool.ID {
				to, _ := eng.GetAccountByID(tr.ToAccount)
				require.Equal(t, types.AccountTypeSettlement, to.Type)
				require.True(t, tr.Amount.EQ(price))
				found = true
				break
			}
		}
	}
	require.True(t, found)
}

func testMTMWithNetworkLossSoc(t *testing.T) {
	party := "test-party"
	moneyParty := "money-party"
	price := num.NewUint(1000)

	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.ID, num.UintZero().Div(price, num.NewUint(2)))
	assert.Nil(t, err)

	// create party accounts
	eng.broker.EXPECT().Send(gomock.Any()).Times(8)
	gID, _ := eng.Engine.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	mID, err := eng.Engine.CreatePartyMarginAccount(context.Background(), party, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	assert.NotEmpty(t, mID)
	assert.NotEmpty(t, gID)

	// create + add balance
	_, _ = eng.Engine.CreatePartyGeneralAccount(context.Background(), moneyParty, testMarketAsset)
	marginMoneyParty, err := eng.Engine.CreatePartyMarginAccount(context.Background(), moneyParty, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	err = eng.Engine.UpdateBalance(context.Background(), marginMoneyParty, num.UintZero().Mul(num.NewUint(5), price))
	assert.Nil(t, err)

	pos := []*types.TransferInstruction{
		{
			Owner: types.NetworkParty,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  testMarketAsset,
			},
			Type: types.TransferInstructionTypeMTMLoss,
		},
		{
			Owner: moneyParty,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  testMarketAsset,
			},
			Type: types.TransferInstructionTypeMTMLoss,
		},
		{
			Owner: party,
			Amount: &types.FinancialAmount{
				Amount: num.Sum(price, price), // one winning party
				Asset:  testMarketAsset,
			},
			Type: types.TransferInstructionTypeMTMWin,
		},
	}

	eng.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes().Do(func(evts []events.Event) {
		for _, e := range evts {
			if lse, ok := e.(PartyEvt); ok {
				require.NotEqual(t, types.NetworkParty, lse.Party())
			}
		}
	})
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes().Do(func(evt events.Event) {
		ae, ok := evt.(accEvt)
		assert.True(t, ok)
		acc := ae.Account()
		// we should never receive an event where an account is owned by the network
		require.False(t, acc.Owner == types.NetworkParty)
		if acc.Owner == party && acc.Type == types.AccountTypeGeneral {
			assert.Equal(t, acc.Balance, int64(833))
		}
		if acc.Owner == moneyParty && acc.Type == types.AccountTypeGeneral {
			assert.Equal(t, acc.Balance, int64(1666))
		}
	})
	transfers := eng.getTestMTMTransfer(pos)
	evts, raw, err := eng.MarkToMarket(context.Background(), testMarketID, transfers, "BTC")
	assert.NoError(t, err)
	assert.Equal(t, 3, len(raw))
	assert.NotEmpty(t, evts)
	for _, e := range evts {
		require.NotEqual(t, types.NetworkParty, e.Party()) // there should be no margin events for the network
	}
	found := false // we should see a transfer from insurance to settlement
	for _, r := range raw {
		for _, tr := range r.TransferInstructions {
			if tr.FromAccount == insurancePool.ID {
				to, _ := eng.GetAccountByID(tr.ToAccount)
				require.Equal(t, types.AccountTypeSettlement, to.Type)
				found = true
				require.False(t, tr.Amount.EQ(price)) // there wasn't enough balance to pay the full MTM share
				require.True(t, tr.Amount.EQ(num.UintZero().Div(price, num.NewUint(2))))
				break
			}
		}
	}
	require.True(t, found)
}
