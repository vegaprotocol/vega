package referral_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	vgtest "code.vegaprotocol.io/vega/libs/test"
	"github.com/stretchr/testify/require"
)

func TestEngine(t *testing.T) {
	t.Run("Updating the referral program succeeds", testUpdatingReferralProgramSucceeds)
}

func testUpdatingReferralProgramSucceeds(t *testing.T) {
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
	lastEpochEndTime := program1.EndOfProgramTimestamp.Add(-2 * time.Hour)
	endEpoch(t, ctx, te, lastEpochEndTime)

	require.False(t, te.engine.HasProgramEnded(), "The program should have started.")

	// Simulating end of epoch.
	// The program should have reached its end.
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
	lastEpochEndTime = program3.EndOfProgramTimestamp.Add(-1 * time.Second)
	endEpoch(t, ctx, te, lastEpochEndTime)

	require.False(t, te.engine.HasProgramEnded(), "The program should have started.")

	// Simulating end of epoch.
	// The third program should have ended.
	lastEpochEndTime = program3.EndOfProgramTimestamp.Add(1 * time.Second)
	endEpoch(t, ctx, te, lastEpochEndTime)

	program4 := &types.ReferralProgram{
		EndOfProgramTimestamp: lastEpochEndTime.Add(10 * time.Hour),
		WindowLength:          10,
		BenefitTiers:          []*types.BenefitTier{},
	}

	// Update with new program.
	te.engine.Update(program4)

	require.True(t, te.engine.HasProgramEnded(), "The program should start only on the next epoch")

	// Simulating end of epoch.
	// The fourth program should have ended before it even started.
	lastEpochEndTime = program4.EndOfProgramTimestamp.Add(1 * time.Second)
	endEpoch(t, ctx, te, lastEpochEndTime)

	require.True(t, te.engine.HasProgramEnded(), "The program should have ended before it even started")
}
