package validators_test

import (
	"context"
	"encoding/hex"
	"testing"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types/num"
	"code.vegaprotocol.io/vega/validators"
	"github.com/stretchr/testify/require"
	abcitypes "github.com/tendermint/tendermint/abci/types"
	types1 "github.com/tendermint/tendermint/proto/tendermint/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

var (
	tmkey1 = "uBr9FP/M/QyVtOa3j18+hjksXra7qxCa7e25/FVW5c0="
	tmkey2 = "7xmxwJpTnPHt6u+18ggFIJzlTWtfKSLKBFkGD6AC99o="
	tmkey3 = "7kRL1jCJH8QUDTHK90/Nz9lIAvl8/s1Z70XL1EXFkaM="
	tmkey4 = "g2QGqXnfkNmmVO4QSw/wu5kpk8rbaY+I4qYNzk7QJTc="
	tmkey5 = "vQVqN1N0+k1GtGZmB8gb1b9BR/cdcYFZtxgiywaTVYM="

	address1, _ = hex.DecodeString("91484AD0B6343D73690F1D36A80EF92B67622C47")
	address2, _ = hex.DecodeString("3619F6EC431527F02457875B7355041ADBB54772")
	address3, _ = hex.DecodeString("13FA0B679D6064772567C7A6050B42CCA1C7C8CD")
	address4, _ = hex.DecodeString("15B7DA235BEED81158737FBFE79C6264D5E2E5FF")
	address5, _ = hex.DecodeString("34DA2E4636D96ABE36AE63D3A01A9AC86802A1CF")
)

func TestValidatorPerformanceNoPerformance(t *testing.T) {
	vp := validators.NewValidatorPerformance(logging.NewTestLogger())
	require.Equal(t, num.DecimalFromFloat(1), vp.ValidatorPerformanceScore("some name"))
}

func TestElectedExpectationWithVotingPower(t *testing.T) {
	vp := validators.NewValidatorPerformance(logging.NewTestLogger())

	vd1 := []*tmtypes.Validator{
		{Address: address1, VotingPower: 3715, ProposerPriority: 5249},
		{Address: address2, VotingPower: 3351, ProposerPriority: 796},
		{Address: address3, VotingPower: 2793, ProposerPriority: -797},
		{Address: address4, VotingPower: 139, ProposerPriority: 1016},
		{Address: address5, VotingPower: 1, ProposerPriority: -6264},
	}
	req1 := abcitypes.RequestBeginBlock{Header: types1.Header{ProposerAddress: address1, Height: int64(1)}}
	vp.BeginBlock(context.Background(), req1, vd1)

	// expect all validators to have the same performance score, all but address1 for not being selected and address1 for being selected and proposing
	require.Equal(t, "1", vp.ValidatorPerformanceScore(tmkey1).String())
	require.Equal(t, "1", vp.ValidatorPerformanceScore(tmkey2).String())
	require.Equal(t, "1", vp.ValidatorPerformanceScore(tmkey3).String())
	require.Equal(t, "1", vp.ValidatorPerformanceScore(tmkey4).String())
	require.Equal(t, "1", vp.ValidatorPerformanceScore(tmkey5).String())

	vd2 := []*tmtypes.Validator{
		{Address: address1, VotingPower: 3715, ProposerPriority: 6433},
		{Address: address2, VotingPower: 3351, ProposerPriority: -1853},
		{Address: address3, VotingPower: 2793, ProposerPriority: 5347},
		{Address: address4, VotingPower: 139, ProposerPriority: -3701},
		{Address: address5, VotingPower: 1, ProposerPriority: -6226},
	}

	// expecting address1 to propose but got address3
	req2 := abcitypes.RequestBeginBlock{Header: types1.Header{ProposerAddress: address3, Height: int64(1)}}
	vp.BeginBlock(context.Background(), req2, vd2)

	vd3 := []*tmtypes.Validator{
		{Address: address1, VotingPower: 3715, ProposerPriority: -6433},
		{Address: address2, VotingPower: 3351, ProposerPriority: -1853},
		{Address: address3, VotingPower: 2793, ProposerPriority: -5347},
		{Address: address4, VotingPower: 139, ProposerPriority: 3701},
		{Address: address5, VotingPower: 1, ProposerPriority: -6226},
	}

	// expecting address4 to propose but got address5
	req3 := abcitypes.RequestBeginBlock{Header: types1.Header{ProposerAddress: address5, Height: int64(1)}}
	vp.BeginBlock(context.Background(), req3, vd3)

	require.Equal(t, "0.5", vp.ValidatorPerformanceScore(tmkey1).String())
	require.Equal(t, "1", vp.ValidatorPerformanceScore(tmkey2).String())
	require.Equal(t, "1", vp.ValidatorPerformanceScore(tmkey3).String())
	require.Equal(t, "0.05", vp.ValidatorPerformanceScore(tmkey4).String())
	require.Equal(t, "1", vp.ValidatorPerformanceScore(tmkey5).String())
}
