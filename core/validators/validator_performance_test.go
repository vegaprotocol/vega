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

package validators_test

import (
	"context"
	"encoding/hex"
	"testing"

	"code.vegaprotocol.io/vega/core/validators"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	"github.com/stretchr/testify/require"
)

var (
	tmkey1 = "uBr9FP/M/QyVtOa3j18+hjksXra7qxCa7e25/FVW5c0="
	tmkey4 = "g2QGqXnfkNmmVO4QSw/wu5kpk8rbaY+I4qYNzk7QJTc="

	address1, _ = hex.DecodeString("91484AD0B6343D73690F1D36A80EF92B67622C47")
	address2, _ = hex.DecodeString("3619F6EC431527F02457875B7355041ADBB54772")
	address3, _ = hex.DecodeString("13FA0B679D6064772567C7A6050B42CCA1C7C8CD")
)

func TestValidatorPerformanceNoPerformance(t *testing.T) {
	vp := validators.NewValidatorPerformance(logging.NewTestLogger())
	require.Equal(t, num.DecimalFromFloat(0.05), vp.ValidatorPerformanceScore("some name", 1, 10, num.DecimalOne()))
}

func TestElectedExpectationWithVotingPower(t *testing.T) {
	vp := validators.NewValidatorPerformance(logging.NewTestLogger())
	vp.BeginBlock(context.Background(), hex.EncodeToString(address1))
	vp.BeginBlock(context.Background(), hex.EncodeToString(address2))
	vp.BeginBlock(context.Background(), hex.EncodeToString(address1))
	vp.BeginBlock(context.Background(), hex.EncodeToString(address3))
	vp.BeginBlock(context.Background(), hex.EncodeToString(address2))

	// validator 1 proposed 2 times, out of 5 (i.e. 40%), they only got 20% voting power so they score should be capped at 1
	require.Equal(t, "1", vp.ValidatorPerformanceScore(tmkey1, 10, 50, num.DecimalZero()).String())

	// validator 1 proposed 2 times, out of 5 (i.e. 40%), they only got 40% voting power so they score should be 1
	require.Equal(t, "1", vp.ValidatorPerformanceScore(tmkey1, 20, 50, num.DecimalZero()).String())

	// validator 1 proposed 2 times, out of 5 (i.e. 40%), they got 60% voting power so they score but with a minimum scaling of 2 blocks they should get a score of 1
	require.Equal(t, "1", vp.ValidatorPerformanceScore(tmkey1, 30, 50, num.DecimalZero()).String())

	// validator 4 never proposed but has a 20% of the voting power so with scaling they proposed 2/5 which is greater than their voting power so they get score of 1
	require.Equal(t, "1", vp.ValidatorPerformanceScore(tmkey4, 10, 50, num.DecimalZero()).String())
}

func TestPerformanceScoreWithScaling(t *testing.T) {
	vp := validators.NewValidatorPerformance(logging.NewTestLogger())
	for i := 0; i < 25; i++ {
		vp.BeginBlock(context.Background(), hex.EncodeToString(address1))
	}
	for i := 0; i < 75; i++ {
		vp.BeginBlock(context.Background(), hex.EncodeToString(address2))
	}

	// validator 1 proposed 1/4 of the blocks with 50% of the voting power
	// with the minimum scaling they get a performance score of 27/50
	require.Equal(t, "0.54", vp.ValidatorPerformanceScore(tmkey1, 25, 50, num.DecimalZero()).String())

	// validator 1 proposed 1/4 of the blocks with 50% of the voting power
	// with the scaling of 15% they get a performance score of 25*1.15/50
	require.Equal(t, "0.575", vp.ValidatorPerformanceScore(tmkey1, 25, 50, num.DecimalFromFloat(0.15)).String())
}
