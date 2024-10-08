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

package liquidity_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/idgeneration"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngineSnapshotV2(t *testing.T) {
	originalEngine := newTestEngine(t)

	ctx := context.Background()
	idgen := idgeneration.New(crypto.RandomHash())

	party1 := "party-1"
	commitment1 := 1000000
	party1Orders := []*types.Order{
		{Side: types.SideBuy, Price: num.NewUint(98), Size: 5103},
		{Side: types.SideBuy, Price: num.NewUint(93), Size: 5377},
		{Side: types.SideSell, Price: num.NewUint(102), Size: 4902},
		{Side: types.SideSell, Price: num.NewUint(107), Size: 4673},
	}

	party2 := "party-2"
	commitment2 := 3000000
	party2Orders := []*types.Order{
		{Side: types.SideBuy, Price: num.NewUint(98), Size: 15307},
		{Side: types.SideBuy, Price: num.NewUint(93), Size: 16130},
		{Side: types.SideSell, Price: num.NewUint(102), Size: 14706},
		{Side: types.SideSell, Price: num.NewUint(107), Size: 14019},
	}

	party3 := "party-3"
	commitment3 := 2000000
	party3Provision := &types.LiquidityProvisionSubmission{
		MarketID:         originalEngine.marketID,
		CommitmentAmount: num.NewUint(uint64(commitment3)),
		Fee:              num.DecimalFromFloat(0.5),
	}

	// change the SLA parameters values that were not set in initialisation
	slaParams := &types.LiquiditySLAParams{
		PriceRange:                  num.DecimalFromFloat(0.9), // priceRange
		CommitmentMinTimeFraction:   num.DecimalFromFloat(0.9), // commitmentMinTimeFraction
		SlaCompetitionFactor:        num.DecimalFromFloat(0.9), // slaCompetitionFactor,
		PerformanceHysteresisEpochs: 7,                         // performanceHysteresisEpochs
	}
	originalEngine.engine.UpdateSLAParameters(slaParams)
	originalEngine.engine.OnNonPerformanceBondPenaltyMaxUpdate(num.DecimalFromFloat(0.9))
	originalEngine.engine.OnNonPerformanceBondPenaltySlopeUpdate(num.DecimalFromFloat(0.8))
	originalEngine.engine.OnStakeToCcyVolumeUpdate(num.DecimalFromFloat(0.7))

	// Adding some state.
	originalEngine.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	originalEngine.auctionState.EXPECT().IsOpeningAuction().Return(false).AnyTimes()
	originalEngine.auctionState.EXPECT().InAuction().Return(false).AnyTimes()

	// Adding provisions.
	// This helper method flush the initially pending provisions as "on-going"
	// provisions.
	originalEngine.submitLiquidityProvisionAndCreateOrders(t, ctx, party1, commitment1, idgen, party1Orders)
	originalEngine.submitLiquidityProvisionAndCreateOrders(t, ctx, party2, commitment2, idgen, party2Orders)
	// Adding pending provisions.
	// When not calling `ApplyPendingProvisions()`, the submitted provision is
	// automatically pending if market is not in auction.
	provisioned, err := originalEngine.engine.SubmitLiquidityProvision(ctx, party3Provision, party3, idgen)
	require.NoError(t, err)
	require.False(t, provisioned, "this will help testing the pending provisions, so it should not directly be added as provision")

	originalEngine.engine.RegisterAllocatedFeesPerParty(map[string]*num.Uint{
		"party-1": num.UintFromUint64(1),
		"party-2": num.UintFromUint64(2),
		"party-3": num.UintFromUint64(2),
		"party-4": num.UintFromUint64(3),
	})

	preStats := originalEngine.engine.LiquidityProviderSLAStats(time.Now())
	require.Len(t, preStats, 2)

	// Verifying we can salvage the state for each key, and they are a valid
	// Payload.
	engine1Keys := originalEngine.engine.V2StateProvider().Keys()
	stateResults1 := map[string]stateResult{}
	for _, key := range engine1Keys {
		// Salvage the state.
		state, additionalProviders, err := originalEngine.engine.V2StateProvider().GetState(key)
		require.NoError(t, err)
		assert.Nil(t, additionalProviders, "No additional provider should be generated by this engine")
		require.NotNil(t, state)
		require.NotEmpty(t, state)

		// Deserialize the state to Payload.
		var p snapshotpb.Payload
		require.NoError(t, proto.Unmarshal(state, &p))

		stateResults1[key] = stateResult{
			state:   state,
			payload: types.PayloadFromProto(&p),
		}
	}

	// Another engine to test determinism and consistency.
	otherEngine := newTestEngine(t)
	otherEngine.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	otherEngine.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	otherEngine.auctionState.EXPECT().IsOpeningAuction().Return(false).AnyTimes()
	otherEngine.auctionState.EXPECT().InAuction().Return(false).AnyTimes()

	// Just to verify the keys are deterministic.
	require.Equal(t, engine1Keys, otherEngine.engine.V2StateProvider().Keys())

	// Reloading previous payload in another engine.
	for _, key := range engine1Keys {
		additionalProviders, err := otherEngine.engine.V2StateProvider().LoadState(ctx, stateResults1[key].payload)
		require.NoError(t, err)
		require.Nil(t, additionalProviders, "No additional provider should be generated by this engine")
	}

	for _, key1 := range engine1Keys {
		// Salvage the state from other engine to test determinism.
		state2, additionalProviders, err := otherEngine.engine.V2StateProvider().GetState(key1)
		require.NoError(t, err)
		require.Nil(t, additionalProviders, "No additional provider should be generated by this engine")
		require.NotNil(t, state2)
		require.NotEmpty(t, state2)

		var p snapshotpb.Payload
		require.NoError(t, proto.Unmarshal(state2, &p))
		require.Equalf(t, stateResults1[key1].state, state2, "State for key %q between two engines must match", key1)
	}

	// Check that the restored state is complete, and lead to the same results.
	now := time.Now()

	// Check for penalties
	penalties1 := originalEngine.engine.CalculateSLAPenalties(now)
	penalties2 := otherEngine.engine.CalculateSLAPenalties(now)
	assert.Equal(t, penalties1.AllPartiesHaveFullFeePenalty, penalties2.AllPartiesHaveFullFeePenalty)
	assert.Equal(t, len(penalties1.PenaltiesPerParty), len(penalties2.PenaltiesPerParty))

	for k, p1 := range penalties1.PenaltiesPerParty {
		p2, ok := penalties2.PenaltiesPerParty[k]
		assert.True(t, ok)
		assert.Equal(t, p1.Bond.String(), p2.Bond.String())
		assert.Equal(t, p1.Fee.String(), p2.Fee.String())
	}

	assert.Equal(t,
		originalEngine.engine.CalculateSuppliedStake(),
		otherEngine.engine.CalculateSuppliedStake(),
	)

	// Check for fees stats
	feesStats1 := originalEngine.engine.PaidLiquidityFeesStats()
	feesStats2 := otherEngine.engine.PaidLiquidityFeesStats()
	assert.Equal(t,
		feesStats1.TotalFeesPaid.String(),
		feesStats2.TotalFeesPaid.String(),
	)

	for k, s1 := range feesStats1.FeesPaidPerParty {
		s2, ok := feesStats2.FeesPaidPerParty[k]
		assert.True(t, ok)
		assert.Equal(t, s1.String(), s2.String())
	}

	postStats := otherEngine.engine.LiquidityProviderSLAStats(time.Now())
	require.Len(t, postStats, 2)
	for i := range preStats {
		assert.Equal(t, preStats[i].NotionalVolumeBuys, postStats[i].NotionalVolumeBuys)
		assert.Equal(t, preStats[i].NotionalVolumeSells, postStats[i].NotionalVolumeSells)
		assert.Equal(t, preStats[i].RequiredLiquidity, postStats[i].RequiredLiquidity)
	}
}

func TestStopSnapshotTaking(t *testing.T) {
	te := newTestEngine(t)
	keys := te.engine.V2StateProvider().Keys()

	// signal to kill the engine's snapshots
	te.engine.StopSnapshots()

	s, _, err := te.engine.V2StateProvider().GetState(keys[0])
	assert.NoError(t, err)
	assert.Nil(t, s)
	assert.True(t, te.engine.V2StateProvider().Stopped())
}

type stateResult struct {
	state   []byte
	payload *types.Payload
}
