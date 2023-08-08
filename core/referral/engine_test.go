// Copyright (c) 2023 Gobalsky Labs Limited
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

package referral_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	vgtest "code.vegaprotocol.io/vega/libs/test"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestUpdatingReferralProgramSucceeds(t *testing.T) {
	ctx := vgtest.VegaContext(vgrand.RandomStr(5), vgtest.RandomI64())

	te := newEngine(t)

	require.True(t, te.engine.HasProgramEnded(), "There is no program yet, so the engine should behave as a program ended.")

	program1 := &types.ReferralProgram{
		EndOfProgramTimestamp: time.Now().Add(24 * time.Hour),
		WindowLength:          10,
		BenefitTiers:          []*types.BenefitTier{},
	}

	// Set the first program.
	te.engine.Update(program1)

	require.True(t, te.engine.HasProgramEnded(), "The program should start only on the next epoch")

	// Simulating end of epoch.
	// The program should be applied.
	expectReferralProgramStartedEvent(t, te)
	lastEpochEndTime := program1.EndOfProgramTimestamp.Add(-2 * time.Hour)
	endEpoch(t, ctx, te, lastEpochEndTime)

	require.False(t, te.engine.HasProgramEnded(), "The program should have started.")

	// Simulating end of epoch.
	// The program should have reached its end.
	expectReferralProgramEndedEvent(t, te)
	lastEpochEndTime = program1.EndOfProgramTimestamp.Add(1 * time.Second)
	endEpoch(t, ctx, te, lastEpochEndTime)

	require.True(t, te.engine.HasProgramEnded(), "The program should have reached its ending.")

	program2 := &types.ReferralProgram{
		EndOfProgramTimestamp: lastEpochEndTime.Add(10 * time.Hour),
		WindowLength:          10,
		BenefitTiers:          []*types.BenefitTier{},
	}

	// Set second the program.
	te.engine.Update(program2)

	require.True(t, te.engine.HasProgramEnded(), "The program should start only on the next epoch")

	program3 := &types.ReferralProgram{
		// Ending the program before the second one to show the engine replace the
		// the previous program by this one
		EndOfProgramTimestamp: program2.EndOfProgramTimestamp.Add(-5 * time.Hour),
		WindowLength:          10,
		BenefitTiers:          []*types.BenefitTier{},
	}

	// Override the second program by a third.
	te.engine.Update(program3)

	// Simulating end of epoch.
	// The third program should have started.
	expectReferralProgramStartedEvent(t, te)
	lastEpochEndTime = program3.EndOfProgramTimestamp.Add(-1 * time.Second)
	endEpoch(t, ctx, te, lastEpochEndTime)

	require.False(t, te.engine.HasProgramEnded(), "The program should have started.")

	program4 := &types.ReferralProgram{
		EndOfProgramTimestamp: lastEpochEndTime.Add(10 * time.Hour),
		WindowLength:          10,
		BenefitTiers:          []*types.BenefitTier{},
	}

	// Update to replace the third program by the fourth one.
	te.engine.Update(program4)

	// Simulating end of epoch.
	// The current program should have been updated by the fourth one.
	expectReferralProgramUpdatedEvent(t, te)
	lastEpochEndTime = program4.EndOfProgramTimestamp.Add(-1 * time.Second)
	endEpoch(t, ctx, te, lastEpochEndTime)

	program5 := &types.ReferralProgram{
		EndOfProgramTimestamp: lastEpochEndTime.Add(10 * time.Hour),
		WindowLength:          10,
		BenefitTiers:          []*types.BenefitTier{},
	}

	// Update with new program.
	te.engine.Update(program5)

	require.False(t, te.engine.HasProgramEnded(), "The fourth program should still be up")

	// Simulating end of epoch.
	// The fifth program should have ended before it even started.
	gomock.InOrder(
		expectReferralProgramUpdatedEvent(t, te),
		expectReferralProgramEndedEvent(t, te),
	)
	lastEpochEndTime = program5.EndOfProgramTimestamp.Add(1 * time.Second)
	endEpoch(t, ctx, te, lastEpochEndTime)

	require.True(t, te.engine.HasProgramEnded(), "The program should have ended before it even started")
}
